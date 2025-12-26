package models

import (
	"time"

	"gorm.io/gorm"
)

// Interface represents a network interface
type Interface struct {
	ID                uint           `json:"id" gorm:"primaryKey"`
	InterfaceName     string         `json:"interface_name" gorm:"uniqueIndex;not null"`
	RxBytes           uint64         `json:"rx_bytes"`
	TxBytes           uint64         `json:"tx_bytes"`
	RxRate            float64        `json:"rx_rate"` // Mbps
	TxRate            float64        `json:"tx_rate"` // Mbps
	LastSeen          time.Time      `json:"last_seen"`
	CounterResetCount int            `json:"counter_reset_count"`
	Status            string         `json:"status"` // up, down, unknown
	Comment           string         `json:"comment"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

// TrafficSnapshot stores historical traffic data
type TrafficSnapshot struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	InterfaceName string         `json:"interface_name" gorm:"index"`
	Timestamp     time.Time      `json:"timestamp" gorm:"index"`
	RxBytes       uint64         `json:"rx_bytes"`
	TxBytes       uint64         `json:"tx_bytes"`
	RxRate        float64        `json:"rx_rate"` // Mbps
	TxRate        float64        `json:"tx_rate"` // Mbps
	TotalBytes    uint64         `json:"total_bytes"`
	CounterReset  bool           `json:"counter_reset"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// CounterResetLog tracks counter reset events
type CounterResetLog struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	InterfaceName   string         `json:"interface_name" gorm:"index"`
	ResetTime       time.Time      `json:"reset_time" gorm:"index"`
	PreviousBytes   uint64         `json:"previous_bytes"`
	NewBytes        uint64         `json:"new_bytes"`
	DetectionMethod string         `json:"detection_method"` // sudden_drop, manual_reset, etc.
	Notes           string         `json:"notes"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// MonthlyQuota tracks monthly data usage quotas
type MonthlyQuota struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	InterfaceName string         `json:"interface_name" gorm:"index"`
	Month         int            `json:"month" gorm:"index"`
	Year          int            `json:"year" gorm:"index"`
	Day           int            `json:"day" gorm:"index"`
	RxBytes       uint64         `json:"rx_bytes"`
	TxBytes       uint64         `json:"tx_bytes"`
	TotalBytes    uint64         `json:"total_bytes"`
	QuotaLimit    uint64         `json:"quota_limit"` // 0 means unlimited
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// SystemInfo stores system information from the router
type SystemInfo struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	RouterName  string         `json:"router_name"`
	BoardName   string         `json:"board_name"`
	Version     string         `json:"version"`
	Uptime      string         `json:"uptime"`
	CPU         string         `json:"cpu"`
	Memory      string         `json:"memory"`
	Disk        string         `json:"disk"`
	Timezone    string         `json:"timezone"`
	Identity    string         `json:"identity"`
	LastUpdated time.Time      `json:"last_updated"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName overrides the table name for SystemInfo
func (SystemInfo) TableName() string {
	return "system_info"
}

// WANInterfaceLog tracks WAN interface detection events
type WANInterfaceLog struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	InterfaceName   string         `json:"interface_name" gorm:"index"`
	DetectionMethod string         `json:"detection_method"`
	Confidence      float64        `json:"confidence"`
	Traffic         uint64         `json:"traffic"`
	DetectedAt      time.Time      `json:"detected_at" gorm:"index"`
	Notes           string         `json:"notes"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// WorkerMetricsLog tracks worker pool performance
type WorkerMetricsLog struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	ActiveJobs     int64          `json:"active_jobs"`
	TotalJobs      int64          `json:"total_jobs"`
	SuccessJobs    int64          `json:"success_jobs"`
	FailedJobs     int64          `json:"failed_jobs"`
	AvgResponse    time.Duration  `json:"avg_response"`
	WorkerCount    int            `json:"worker_count"`
	QueueSize      int            `json:"queue_size"`
	LoadPercentage float64        `json:"load_percentage"`
	LoggedAt       time.Time      `json:"logged_at" gorm:"index"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

// WebSocketConnectionLog tracks WebSocket connections
type WebSocketConnectionLog struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	ClientID         string         `json:"client_id" gorm:"index"`
	InterfaceName    string         `json:"interface_name" gorm:"index"`
	ConnectedAt      time.Time      `json:"connected_at"`
	DisconnectedAt   *time.Time     `json:"disconnected_at"`
	MessageCount     int64          `json:"message_count"`
	BytesTransferred int64          `json:"bytes_transferred"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}
