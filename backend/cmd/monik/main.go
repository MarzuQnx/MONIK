package main

import (
	"log"

	"monik-enterprise/internal/api"
	"monik-enterprise/internal/config"
	"monik-enterprise/internal/database"
	"monik-enterprise/internal/router"
	"monik-enterprise/internal/service"
	"monik-enterprise/internal/websocket"
	"monik-enterprise/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize logger
	logger.Init()

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db := database.InitDB(cfg.Database.Path)
	defer database.CloseDB()

	// Run database migrations
	database.RunMigrations(db)

	// Initialize services
	routerService := service.NewMikroTikService(cfg.Router)

	// Initialize WAN detection service
	wanService := service.NewWANDetectionService(cfg.WAN)
	wanService.SetRouterClient(routerService.GetClient())

	// Initialize worker pool
	workerPool := service.NewWorkerPool(cfg.Worker, routerService)
	workerPool.Start()

	// Initialize WebSocket manager
	wsManager := websocket.NewWebSocketManager()
	// Note: wsManager is already started in NewWebSocketManager()

	// Initialize monitoring service
	monitoringService := service.NewMonitoringService(db, routerService, wanService, wsManager)

	// Start monitoring service
	go monitoringService.Start()

	// Initialize API handlers
	handlers := api.NewHandlers(db, monitoringService, wanService, workerPool, wsManager)

	// Setup routes
	r := router.SetupRoutes(handlers)

	// Start server
	log.Printf("Starting server on %s:%d", cfg.Server.Host, cfg.Server.Port)

	// Handle graceful shutdown
	go func() {
		if err := r.Run(cfg.Server.Address()); err != nil {
			log.Fatal("Failed to start server:", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	log.Println("Server started. Press Ctrl+C to shutdown.")
	select {}
}
