package service

import (
	"time"
)

// WebSocketManager interface for broadcasting real-time data
type WebSocketManager interface {
	BroadcastData(data *RealTimeData)
	BroadcastEvent(eventType, message string, data interface{})
	GetClientCount() int
	GetSubscriptionCount() int
}

// RealTimeData represents real-time monitoring data
type RealTimeData struct {
	InterfaceName string    `json:"interface_name"`
	RxRate        float64   `json:"rx_rate"`
	TxRate        float64   `json:"tx_rate"`
	RxBytes       uint64    `json:"rx_bytes"`
	TxBytes       uint64    `json:"tx_bytes"`
	Status        string    `json:"status"`
	Comment       string    `json:"comment"`
	Timestamp     time.Time `json:"timestamp"`
	EventType     string    `json:"event_type"`
}

// Event types
const (
	EventTypeTraffic = "traffic"
	EventTypeReset   = "counter_reset"
	EventTypeReboot  = "reboot"
)
