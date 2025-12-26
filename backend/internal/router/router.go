package router

import (
	"monik-enterprise/internal/api"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all API routes
func SetupRoutes(handlers *api.Handlers) *gin.Engine {
	r := gin.Default()

	// CORS middleware
	r.Use(cors.Default())

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Interface routes
		v1.GET("/interfaces", handlers.GetInterfaces)
		v1.GET("/interfaces/:name", handlers.GetInterface)

		// System info routes
		v1.GET("/system", handlers.GetSystemInfo)

		// Traffic history routes
		v1.GET("/traffic/:interface", handlers.GetTrafficHistory)

		// Test data routes
		v1.POST("/populate-test-data", handlers.PopulateTestData)

		// WebSocket route for real-time updates
		v1.GET("/ws", handlers.WebSocketHandler)

		// WAN detection routes
		v1.GET("/wan-interface", handlers.GetWANInterface)
		v1.GET("/wan-stats", handlers.GetWANDetectionStats)

		// Worker pool routes
		v1.GET("/worker-status", handlers.GetWorkerPoolStatus)
		v1.POST("/submit-job", handlers.SubmitMonitoringJob)

		// WebSocket stats
		v1.GET("/websocket-stats", handlers.GetWebSocketStats)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	return r
}
