package service

import (
	"sync"
	"time"

	"monik-enterprise/internal/websocket"
)

// MetricsService collects and aggregates system metrics
type MetricsService struct {
	mu             sync.RWMutex
	systemMetrics  *SystemMetrics
	websocketMgr   *websocket.WebSocketManager
	lastUpdate     time.Time
	updateInterval time.Duration
}

// SystemMetrics represents overall system performance metrics
type SystemMetrics struct {
	WebSocketMetrics    *websocket.WebSocketMetrics
	WANDetectionMetrics *WANDetectionMetrics
	WorkerPoolMetrics   *WorkerMetrics
	LastUpdated         time.Time
	SystemHealth        SystemHealth
}

// SystemHealth represents overall system health status
type SystemHealth struct {
	Status        string        `json:"status"` // healthy, degraded, critical
	LastCheck     time.Time     `json:"last_check"`
	Uptime        time.Duration `json:"uptime"`
	ErrorRate     float64       `json:"error_rate"`
	ResponseTime  time.Duration `json:"response_time"`
	ActiveWorkers int           `json:"active_workers"`
	QueueSize     int           `json:"queue_size"`
}

// NewMetricsService creates a new metrics service
func NewMetricsService(websocketMgr *websocket.WebSocketManager) *MetricsService {
	return &MetricsService{
		websocketMgr:   websocketMgr,
		updateInterval: 30 * time.Second,
		systemMetrics: &SystemMetrics{
			WebSocketMetrics:    websocket.NewWebSocketMetrics(),
			WANDetectionMetrics: NewWANDetectionMetrics(),
			WorkerPoolMetrics: &WorkerMetrics{
				WorkerStats: make(map[int]*WorkerStats),
			},
			SystemHealth: SystemHealth{
				Status:    "unknown",
				LastCheck: time.Now(),
			},
		},
	}
}

// Start starts the metrics collection service
func (ms *MetricsService) Start() {
	go ms.collectMetrics()
}

// collectMetrics collects metrics from all components
func (ms *MetricsService) collectMetrics() {
	ticker := time.NewTicker(ms.updateInterval)
	defer ticker.Stop()

	// S1000 Fix: Menggunakan range untuk menyederhanakan loop ticker
	for range ticker.C {
		ms.updateSystemMetrics()
	}
}

// updateSystemMetrics updates the overall system metrics
func (ms *MetricsService) updateSystemMetrics() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.systemMetrics.LastUpdated = time.Now()

	// Update system health
	ms.updateSystemHealth()

	// Broadcast metrics update via WebSocket
	if ms.websocketMgr != nil {
		ms.websocketMgr.BroadcastEvent("metrics_update",
			"System metrics updated",
			map[string]interface{}{
				"timestamp":             ms.systemMetrics.LastUpdated,
				"system_health":         ms.systemMetrics.SystemHealth,
				"websocket_metrics":     ms.systemMetrics.WebSocketMetrics.GetStats(),
				"wan_detection_metrics": ms.systemMetrics.WANDetectionMetrics.GetStats(),
			})
	}
}

// updateSystemHealth calculates overall system health
func (ms *MetricsService) updateSystemHealth() {
	health := &ms.systemMetrics.SystemHealth
	health.LastCheck = time.Now()

	// Calculate error rate from all components
	var totalErrors, totalRequests float64

	// WebSocket error rate
	wsStats := ms.systemMetrics.WebSocketMetrics.GetStats()
	if wsStats != nil {
		if messagesDropped, ok := wsStats["messages_dropped"].(int64); ok {
			totalErrors += float64(messagesDropped)
		}
		if messagesSent, ok := wsStats["messages_sent"].(int64); ok {
			totalRequests += float64(messagesSent)
		}
	}

	// WAN detection error rate
	wanStats := ms.systemMetrics.WANDetectionMetrics.GetStats()
	if wanStats != nil {
		if failures, ok := wanStats["detection_failure"].(int64); ok {
			totalErrors += float64(failures)
		}
		if total, ok := wanStats["cache_hits"].(int64); ok {
			totalRequests += float64(total)
		}
	}

	// Worker pool error rate
	workerStats := ms.systemMetrics.WorkerPoolMetrics
	if workerStats != nil {
		totalErrors += float64(workerStats.FailedJobs)
		totalRequests += float64(workerStats.TotalJobs)
	}

	// Calculate error rate
	if totalRequests > 0 {
		health.ErrorRate = (totalErrors / totalRequests) * 100
	}

	// Determine health status
	if health.ErrorRate < 1.0 {
		health.Status = "healthy"
	} else if health.ErrorRate < 5.0 {
		health.Status = "degraded"
	} else {
		health.Status = "critical"
	}

	// Calculate uptime
	health.Uptime = time.Since(ms.lastUpdate)

	// Set response time (simplified calculation)
	health.ResponseTime = 100 * time.Millisecond // This would be calculated from actual response times

	// Set worker and queue stats
	health.ActiveWorkers = int(workerStats.ActiveJobs)
	health.QueueSize = 0 // This would be calculated from actual queue size
}

// GetSystemMetrics returns current system metrics
func (ms *MetricsService) GetSystemMetrics() *SystemMetrics {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := &SystemMetrics{
		LastUpdated:  ms.systemMetrics.LastUpdated,
		SystemHealth: ms.systemMetrics.SystemHealth,
	}

	// Copy component metrics
	if ms.systemMetrics.WebSocketMetrics != nil {
		metrics.WebSocketMetrics = ms.systemMetrics.WebSocketMetrics
	}
	if ms.systemMetrics.WANDetectionMetrics != nil {
		metrics.WANDetectionMetrics = ms.systemMetrics.WANDetectionMetrics
	}
	if ms.systemMetrics.WorkerPoolMetrics != nil {
		metrics.WorkerPoolMetrics = ms.systemMetrics.WorkerPoolMetrics
	}

	return metrics
}

// GetSystemHealth returns current system health
func (ms *MetricsService) GetSystemHealth() SystemHealth {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.systemMetrics.SystemHealth
}

// SetWANDetectionMetrics sets the WAN detection metrics
func (ms *MetricsService) SetWANDetectionMetrics(metrics *WANDetectionMetrics) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.systemMetrics.WANDetectionMetrics = metrics
}

// SetWorkerPoolMetrics sets the worker pool metrics
func (ms *MetricsService) SetWorkerPoolMetrics(metrics *WorkerMetrics) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.systemMetrics.WorkerPoolMetrics = metrics
}

// RecordError records an error in the system metrics
func (ms *MetricsService) RecordError(component string, errorType string, message string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// This would integrate with a logging system
	// For now, we just update the health status
	ms.updateSystemHealth()
}

// RecordSuccess records a successful operation
func (ms *MetricsService) RecordSuccess(component string, operation string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Update metrics based on successful operation
	ms.updateSystemHealth()
}

// WANDetectionMetrics tracks WAN detection performance metrics
type WANDetectionMetrics struct {
	mu              sync.RWMutex
	CacheHits       int64            `json:"cache_hits"`
	TotalDetections int64            `json:"total_detections"`
	Failures        int64            `json:"failures"`
	MethodCounts    map[string]int64 `json:"method_counts"`
}

func NewWANDetectionMetrics() *WANDetectionMetrics {
	return &WANDetectionMetrics{
		MethodCounts: make(map[string]int64),
	}
}

func (m *WANDetectionMetrics) RecordCacheHit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CacheHits++
}

func (m *WANDetectionMetrics) RecordDetection(method string, confidence float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TotalDetections++
	m.MethodCounts[method]++
}

func (m *WANDetectionMetrics) RecordDetectionFailure() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Failures++
}

// GetStats mengembalikan statistik deteksi WAN dalam format map untuk dikirim ke API
func (m *WANDetectionMetrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"cache_hits":        m.CacheHits,
		"total_detections":  m.TotalDetections,
		"detection_failure": m.Failures,
		"methods":           m.MethodCounts,
	}
}
