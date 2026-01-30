package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/auth"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/config"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/database"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/handlers"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/middleware"
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

	jwtService := auth.NewJWT(cfg)

	router := setupRouter(db, cfg, jwtService)

	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func setupRouter(db *gorm.DB, cfg *config.Config, jwtService *auth.JWT) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	authHandler := handlers.NewAuthHandler(db, jwtService)
	documentHandler := handlers.NewDocumentHandler(db)
	collaboratorHandler := handlers.NewCollaboratorHandler(db)
	notificationHandler := handlers.NewNotificationHandler(db)
	historyHandler := handlers.NewHistoryHandler(db)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":   "healthy",
			"database": "connected",
		})
	})

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

				protected.POST("/:id/collaborators", collaboratorHandler.AddCollaborator)
				protected.GET("/:id/collaborators", collaboratorHandler.ListCollaborators)
				protected.PUT("/:id/collaborators/:user_id", collaboratorHandler.UpdateCollaborator)
				protected.DELETE("/:id/collaborators/:user_id", collaboratorHandler.RemoveCollaborator)

				protected.POST("/:id/share-link", collaboratorHandler.CreateShareLink)
				protected.DELETE("/:id/share-link", collaboratorHandler.DeleteShareLink)

				protected.GET("/:id/history", historyHandler.GetDocumentHistory)
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
