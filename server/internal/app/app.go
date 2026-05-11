package app

import (
	"fmt"
	"log"
	"os"

	"github.com/guijoazeiro/text-editor/tree/main/server/internal/auth"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/config"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/database"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/router"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/websocket"
)

func Run() error {
	cfg := config.New()

	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	log.Println("Database connected successfully")

	yjsService := services.NewYjsService(db)
	log.Println("Yjs service initialised")

	snapshotService := services.NewSnapshotService(db, yjsService)
	log.Println("Snapshot service initialised")

	compactorURL := os.Getenv("COMPACTOR_URL")
	if compactorURL == "" {
		compactorURL = "http://localhost:3001"
	}
	compactorService := services.NewCompactorService(yjsService, snapshotService, services.CompactorConfig{
		WorkerURL: compactorURL,
	})
	log.Printf("Compactor service initialised (worker=%s)", compactorURL)

	permissionService := services.NewPermissionService(db)

	hub := websocket.NewHub(yjsService, snapshotService, compactorService)
	go hub.Run()
	log.Println("WebSocket hub started")

	jwtService := auth.NewJWT(cfg)

	deps := router.Dependencies{
		Config:            cfg,
		DB:                db,
		JWTService:        jwtService,
		Hub:               hub,
		YjsService:        yjsService,
		SnapshotService:   snapshotService,
		CompactorService:  compactorService,
		PermissionService: permissionService,
	}

	r := router.New(deps)

	port := cfg.Port
	log.Printf("Server starting on port %s", port)
	return r.Run(":" + port)
}
