package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/handlers"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/middleware"
	"golang.org/x/time/rate"
)

func registerRoutes(r *gin.Engine, deps Dependencies) {
	r.GET("/health", healthHandler)

	registerWebSocket(r, deps)
	registerAPI(r, deps)
}

func healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":   "healthy",
		"database": "connected",
	})
}

func registerWebSocket(r *gin.Engine, deps Dependencies) {
	wsHandler := handlers.NewWebSocketHandler(deps.Hub, deps.DB, deps.JWTService)

	ws := r.Group("/ws")
	ws.GET("/documents/:id", wsHandler.HandleWebSocket)
}

func registerAPI(r *gin.Engine, deps Dependencies) {
	api := r.Group("/api")

	registerAuth(api, deps)
	registerDocuments(api, deps)
	registerNotifications(api, deps)
}

func registerAuth(api *gin.RouterGroup, deps Dependencies) {
	authHandler := handlers.NewAuthHandler(deps.DB, deps.JWTService)

	loginLimiter := middleware.NewRateLimiter(rate.Every(12*time.Second), 5)
	signupLimiter := middleware.NewRateLimiter(rate.Every(20*time.Second), 3)

	g := api.Group("/auth")
	{
		g.POST("/signup", signupLimiter.Middleware(), authHandler.Signup)
		g.POST("/login", loginLimiter.Middleware(), authHandler.Login)
		g.POST("/refresh", authHandler.Refresh)
		g.GET("/me", middleware.AuthRequired(deps.JWTService), authHandler.Me)
		g.PATCH("/me", middleware.AuthRequired(deps.JWTService), authHandler.UpdateMe)
		g.POST("/logout", middleware.AuthRequired(deps.JWTService), authHandler.Logout)
	}
}

func registerDocuments(api *gin.RouterGroup, deps Dependencies) {
	documentHandler := handlers.NewDocumentHandler(deps.DB, deps.SnapshotService)
	collaboratorHandler := handlers.NewCollaboratorHandler(deps.DB, deps.Hub)
	historyHandler := handlers.NewHistoryHandler(deps.DB)
	versionHandler := handlers.NewVersionHandler(deps.DB, deps.SnapshotService, deps.YjsService, deps.Hub)
	wsHandler := handlers.NewWebSocketHandler(deps.Hub, deps.DB, deps.JWTService)
	yjsHandler := handlers.NewYjsHandler(deps.YjsService, deps.PermissionService, deps.SnapshotService, deps.CompactorService)

	shareLinkLimiter := middleware.NewRateLimiter(rate.Every(6*time.Second), 5)

	g := api.Group("/documents")
	{
		g.GET("/shared/:token", shareLinkLimiter.Middleware(), collaboratorHandler.GetByShareLink)

		protected := g.Group("")
		protected.Use(middleware.AuthRequired(deps.JWTService))
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
			protected.POST("/:id/yjs/reset", yjsHandler.ResetState)

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
}

func registerNotifications(api *gin.RouterGroup, deps Dependencies) {
	notificationHandler := handlers.NewNotificationHandler(deps.DB)

	g := api.Group("/notifications")
	g.Use(middleware.AuthRequired(deps.JWTService))
	{
		g.GET("", notificationHandler.List)
		g.PUT("/:id/read", notificationHandler.MarkAsRead)
		g.PUT("/read-all", notificationHandler.MarkAllAsRead)
		g.DELETE("/:id", notificationHandler.Delete)
	}
}
