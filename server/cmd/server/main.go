package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/auth"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/config"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/database"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/handlers"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/middleware"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/websocket"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := config.New()

	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connected successfully")

	yjsService := services.NewYjsService(db)
	log.Println("Yjs service initialized")

	snapshotService := services.NewSnapshotService(db, yjsService)
	log.Println("Snapshot service initialized")

	compactorURL := os.Getenv("COMPACTOR_URL")
	if compactorURL == "" {
		compactorURL = "http://localhost:3001"
	}
	compactorService := services.NewCompactorService(yjsService, snapshotService, services.CompactorConfig{
		WorkerURL: compactorURL,
	})
	log.Printf("Compactor service initialized (worker=%s)", compactorURL)

	hub := websocket.NewHub(yjsService, snapshotService, compactorService)
	go hub.Run()
	log.Println("WebSocket hub started")

	jwtService := auth.NewJWT(cfg)

	router := setupRouter(db, cfg, jwtService, hub, yjsService, snapshotService, compactorService)

	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func setupRouter(db *gorm.DB, cfg *config.Config, jwtService *auth.JWT, hub *websocket.Hub, yjsService *services.YjsService, snapshotService *services.SnapshotService, compactorService *services.CompactorService) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	authHandler := handlers.NewAuthHandler(db, jwtService)
	documentHandler := handlers.NewDocumentHandler(db, snapshotService)
	collaboratorHandler := handlers.NewCollaboratorHandler(db, hub)
	notificationHandler := handlers.NewNotificationHandler(db)
	historyHandler := handlers.NewHistoryHandler(db)
	versionHandler := handlers.NewVersionHandler(db, snapshotService, yjsService, hub)
	wsHandler := handlers.NewWebSocketHandler(hub, db, jwtService)

	permissionService := services.NewPermissionService(db)
	yjsHandler := handlers.NewYjsHandler(yjsService, permissionService, snapshotService, compactorService)

	shareLinkLimiter := middleware.NewRateLimiter(rate.Every(6*time.Second), 5)
	loginLimiter := middleware.NewRateLimiter(rate.Every(12*time.Second), 5)  // 5 req/min per IP
	signupLimiter := middleware.NewRateLimiter(rate.Every(20*time.Second), 3) // 3 req/min per IP

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":   "healthy",
			"database": "connected",
		})
	})

	ws := router.Group("/ws")
	{
		ws.GET("/documents/:id", wsHandler.HandleWebSocket)
	}

	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/signup", signupLimiter.Middleware(), authHandler.Signup)
			auth.POST("/login", loginLimiter.Middleware(), authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.GET("/me", middleware.AuthRequired(jwtService), authHandler.Me)
			auth.PATCH("/me", middleware.AuthRequired(jwtService), authHandler.UpdateMe)
			auth.POST("/logout", middleware.AuthRequired(jwtService), authHandler.Logout)
		}

		documents := api.Group("/documents")
		{
			documents.GET("/shared/:token", shareLinkLimiter.Middleware(), collaboratorHandler.GetByShareLink)

			protected := documents.Group("")
			protected.Use(middleware.AuthRequired(jwtService))
			{
				protected.POST("", documentHandler.Create)
				protected.GET("", documentHandler.List)
				protected.GET("/trash", documentHandler.Trash)
				protected.GET("/:id", documentHandler.GetByID)
				protected.PUT("/:id", documentHandler.Update)
				protected.DELETE("/:id", documentHandler.Delete)
				protected.POST("/:id/restore", documentHandler.Restore)

				protected.GET("/:id/active-users", wsHandler.GetActiveUsers)

				protected.GET("/:id/yjs-updates", yjsHandler.GetUpdates)
				protected.GET("/:id/yjs-state-vector", yjsHandler.GetStateVector)
				protected.GET("/:id/yjs-diff", yjsHandler.GetDiff)

				protected.POST("/:id/collaborators", collaboratorHandler.AddCollaborator)
				protected.GET("/:id/collaborators", collaboratorHandler.ListCollaborators)
				protected.PUT("/:id/collaborators/:user_id", collaboratorHandler.UpdateCollaborator)
				protected.DELETE("/:id/collaborators/:user_id", collaboratorHandler.RemoveCollaborator)

				protected.POST("/:id/share-link", collaboratorHandler.CreateShareLink)
				protected.DELETE("/:id/share-link", collaboratorHandler.DeleteShareLink)

				protected.GET("/:id/history", historyHandler.GetDocumentHistory)

				protected.GET("/:id/versions", versionHandler.GetVersions)
				protected.GET("/:id/versions/:version_number", versionHandler.GetVersion)
				protected.POST("/:id/versions/:version_number/restore", versionHandler.RestoreVersion)
				protected.GET("/:id/versions/compare", versionHandler.CompareVersions)
			}
		}

		notifications := api.Group("/notifications")
		notifications.Use(middleware.AuthRequired(jwtService))
		{
			notifications.GET("", notificationHandler.List)
			notifications.PUT("/:id/read", notificationHandler.MarkAsRead)
			notifications.PUT("/read-all", notificationHandler.MarkAllAsRead)
			notifications.DELETE("/:id", notificationHandler.Delete)
		}
	}

	return router
}
