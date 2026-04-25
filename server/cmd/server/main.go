package main

import (
	"log"
	"os"

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

	router := setupRouter(db, cfg, jwtService, hub, yjsService, snapshotService)

	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func setupRouter(db *gorm.DB, cfg *config.Config, jwtService *auth.JWT, hub *websocket.Hub, yjsService *services.YjsService, snapshotService *services.SnapshotService) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	authHandler := handlers.NewAuthHandler(db, jwtService)
	documentHandler := handlers.NewDocumentHandler(db)
	collaboratorHandler := handlers.NewCollaboratorHandler(db)
	notificationHandler := handlers.NewNotificationHandler(db)
	historyHandler := handlers.NewHistoryHandler(db)
	versionHandler := handlers.NewVersionHandler(db, snapshotService, hub)
	wsHandler := handlers.NewWebSocketHandler(hub, db, jwtService)

	permissionService := services.NewPermissionService(db)
	yjsHandler := handlers.NewYjsHandler(yjsService, permissionService)

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
			auth.POST("/signup", authHandler.Signup)
			auth.POST("/login", authHandler.Login)
			auth.GET("/me", middleware.AuthRequired(jwtService), authHandler.Me)
		}

		documents := api.Group("/documents")
		{
			documents.GET("/shared/:token", collaboratorHandler.GetByShareLink)

			protected := documents.Group("")
			protected.Use(middleware.AuthRequired(jwtService))
			{
				protected.POST("", documentHandler.Create)
				protected.GET("", documentHandler.List)
				protected.GET("/:id", documentHandler.GetByID)
				protected.PUT("/:id", documentHandler.Update)
				protected.DELETE("/:id", documentHandler.Delete)

				protected.GET("/:id/active-users", wsHandler.GetActiveUsers)

				protected.GET("/:id/yjs-updates", yjsHandler.GetUpdates)

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
