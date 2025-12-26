package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"monik-enterprise/internal/models"
	"monik-enterprise/internal/service"
	"monik-enterprise/internal/websocket"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handlers contains all API handlers
type Handlers struct {
	db               *gorm.DB
	service          *service.MonitoringService
	wanService       *service.WANDetectionService
	workerPool       *service.WorkerPool
	websocketManager *websocket.WebSocketManager
}

// NewHandlers creates new API handlers
func NewHandlers(db *gorm.DB, svc *service.MonitoringService, wanSvc *service.WANDetectionService, workerPool *service.WorkerPool, wsManager *websocket.WebSocketManager) *Handlers {
	return &Handlers{
		db:               db,
		service:          svc,
		wanService:       wanSvc,
		workerPool:       workerPool,
		websocketManager: wsManager,
	}
}

// GetInterfaces returns all interfaces
func (h *Handlers) GetInterfaces(c *gin.Context) {
	interfaces, err := h.service.GetLatestInterfaces()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve interfaces",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"interfaces": interfaces,
	})
}

// GetInterface returns a specific interface
func (h *Handlers) GetInterface(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Interface name is required",
		})
		return
	}

	iface, err := h.service.GetInterfaceByName(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Interface not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve interface",
			})
		}
		return
	}

	c.JSON(http.StatusOK, iface)
}

// GetSystemInfo returns system information
func (h *Handlers) GetSystemInfo(c *gin.Context) {
	info, err := h.service.GetSystemInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve system info",
		})
		return
	}

	c.JSON(http.StatusOK, info)
}

// GetTrafficHistory returns traffic history for an interface
func (h *Handlers) GetTrafficHistory(c *gin.Context) {
	interfaceName := c.Param("interface")
	if interfaceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Interface name is required",
		})
		return
	}

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 100
	}

	// Get traffic history from database
	var snapshots []models.TrafficSnapshot
	if err := h.db.Where("interface_name = ?", interfaceName).
		Order("timestamp DESC").
		Limit(limit).
		Find(&snapshots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve traffic history",
		})
		return
	}

	// Convert to response format
	history := make([]gin.H, len(snapshots))
	for i, snapshot := range snapshots {
		history[i] = gin.H{
			"timestamp": snapshot.Timestamp.Format(time.RFC3339),
			"rx_rate":   snapshot.RxRate,
			"tx_rate":   snapshot.TxRate,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"interface": interfaceName,
		"history":   history,
		"limit":     limit,
	})
}

// PopulateTestData populates the database with test counter reset logs
func (h *Handlers) PopulateTestData(c *gin.Context) {
	if err := h.service.PopulateTestCounterResetLogs(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to populate test data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Test data populated successfully",
	})
}

// WebSocketHandler handles WebSocket connections for real-time updates
func (h *Handlers) WebSocketHandler(c *gin.Context) {
	if h.websocketManager != nil {
		h.websocketManager.HandleConnection(c.Writer, c.Request)
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "WebSocket service not available",
		})
	}
}

// GetWANInterface returns the detected WAN interface
func (h *Handlers) GetWANInterface(c *gin.Context) {
	if h.wanService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "WAN detection service not available",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	wan, err := h.wanService.DetectWANInterface(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Jika WAN interface adalah "none", berikan respons yang lebih informatif
	if wan.Name == "none" {
		c.JSON(http.StatusOK, gin.H{
			"wan_interface": wan,
			"message":       "No active WAN interface detected. Please check your router configuration.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"wan_interface": wan,
	})
}

// GetWANDetectionStats returns WAN detection statistics
func (h *Handlers) GetWANDetectionStats(c *gin.Context) {
	if h.wanService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "WAN detection service not available",
		})
		return
	}

	stats := h.wanService.GetDetectionStats()
	c.JSON(http.StatusOK, stats)
}

// GetWorkerPoolStatus returns worker pool status and metrics
func (h *Handlers) GetWorkerPoolStatus(c *gin.Context) {
	if h.workerPool == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Worker pool service not available",
		})
		return
	}

	metrics := h.workerPool.GetMetrics()
	status := gin.H{
		"worker_count":     h.workerPool.GetWorkerCount(),
		"queue_size":       h.workerPool.GetQueueSize(),
		"queue_capacity":   h.workerPool.GetQueueCapacity(),
		"load_percentage":  h.workerPool.GetLoad(),
		"should_rebalance": h.workerPool.ShouldRebalance(),
		"metrics":          metrics,
	}

	c.JSON(http.StatusOK, status)
}

// GetWebSocketStats returns WebSocket connection statistics
func (h *Handlers) GetWebSocketStats(c *gin.Context) {
	if h.websocketManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "WebSocket service not available",
		})
		return
	}

	// Perbaikan: Gunakan GetSubscriptions() dan hitung totalnya
	subscriptions := h.websocketManager.GetSubscriptions()

	stats := gin.H{
		"client_count":       h.websocketManager.GetClientCount(),
		"subscription_count": len(subscriptions), // Jumlah interface yang sedang dipantau
		"active_channels":    subscriptions,      // Detail interface -> jumlah penyadap
	}

	c.JSON(http.StatusOK, stats)
}

// SubmitMonitoringJob submits a monitoring job to the worker pool
func (h *Handlers) SubmitMonitoringJob(c *gin.Context) {
	if h.workerPool == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Worker pool service not available",
		})
		return
	}

	var req struct {
		InterfaceName string `json:"interface_name" binding:"required"`
		Type          string `json:"type" binding:"required"`
		Timeout       int    `json:"timeout"`
		MaxRetries    int    `json:"max_retries"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	job := service.Job{
		InterfaceName: req.InterfaceName,
		Type:          req.Type,
		Timeout:       time.Duration(req.Timeout) * time.Second,
		MaxRetries:    req.MaxRetries,
	}

	if err := h.workerPool.SubmitJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Job submitted successfully",
		"job": gin.H{
			"interface_name": req.InterfaceName,
			"type":           req.Type,
			"timeout":        req.Timeout,
			"max_retries":    req.MaxRetries,
		},
	})
}
