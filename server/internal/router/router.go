package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/auth"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/config"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/websocket"
	"gorm.io/gorm"
)

type Dependencies struct {
	Config            *config.Config
	DB                *gorm.DB
	JWTService        *auth.JWT
	Hub               *websocket.Hub
	YjsService        *services.YjsService
	SnapshotService   *services.SnapshotService
	CompactorService  *services.CompactorService
	PermissionService *services.PermissionService
}

func New(deps Dependencies) *gin.Engine {
	if deps.Config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(cors.New(buildCORSConfig(deps.Config)))

	registerRoutes(r, deps)

	return r
}

func buildCORSConfig(cfg *config.Config) cors.Config {
	return cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}
}
