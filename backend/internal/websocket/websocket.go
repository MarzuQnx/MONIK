package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketManager manages WebSocket connections and subscriptions
type WebSocketManager struct {
	clients       map[string]*Client
	subscriptions map[string]map[*Client]bool // interface -> clients
	mu            sync.RWMutex
	broadcast     chan interface{}
	eventBus      *EventBus
	metrics       *WebSocketMetrics
}

// Client represents a WebSocket client connection
type Client struct {
	ID        string
	Conn      *websocket.Conn
	Send      chan []byte
	Closed    chan bool
	Sub       string // Interface ID that is subscribed
	Connected time.Time
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
	EventTypeTraffic       = "traffic"
	EventTypeReset         = "counter_reset"
	EventTypeReboot        = "reboot"
	EventTypeWANDetected   = "wan_detected"
	EventTypeInterfaceUp   = "interface_up"
	EventTypeInterfaceDown = "interface_down"
)

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		clients:       make(map[string]*Client),
		subscriptions: make(map[string]map[*Client]bool),
		broadcast:     make(chan interface{}, 10000), // Increased buffer for high throughput
		eventBus:      NewEventBus(),
		metrics:       NewWebSocketMetrics(),
	}
}

// Start starts the WebSocket manager
func (wm *WebSocketManager) Start() {
	go wm.run()
	go wm.eventBus.Start()
}

// run runs the WebSocket manager main loop
func (wm *WebSocketManager) run() {
	// S1000 Fix: Menggunakan range untuk performa lebih baik dan kode lebih clean
	for data := range wm.broadcast {
		wm.handleBroadcast(data)
	}
}

// handleBroadcast handles broadcasting data to subscribed clients
func (wm *WebSocketManager) handleBroadcast(data interface{}) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	switch dataRealTime := data.(type) {
	case RealTimeData:
		if clients, exists := wm.subscriptions[dataRealTime.InterfaceName]; exists {
			for client := range clients {
				select {
				case client.Send <- wm.serializeData(dataRealTime):
					wm.metrics.RecordMessageSent()
				case <-client.Closed:
					// Client disconnected, will be cleaned up
				default:
					// Channel full, skip this message
					wm.metrics.RecordMessageDropped()
					log.Printf("Client %s channel full, skipping message", client.ID)
				}
			}
		}
	case EventData:
		// Broadcast events to all clients
		jsonData := wm.serializeEvent(dataRealTime)
		for _, client := range wm.clients {
			select {
			case client.Send <- jsonData:
				wm.metrics.RecordMessageSent()
			default:
				wm.metrics.RecordMessageDropped()
			}
		}
	}
}

// HandleConnection handles a new WebSocket connection
func (wm *WebSocketManager) HandleConnection(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for now
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		ID:        fmt.Sprintf("client_%d", time.Now().UnixNano()),
		Conn:      conn,
		Send:      make(chan []byte, 1024), // Increased buffer for better performance
		Closed:    make(chan bool),
		Connected: time.Now(),
	}

	wm.mu.Lock()
	wm.clients[client.ID] = client
	wm.mu.Unlock()

	// Start client handlers
	go wm.readPump(client)
	go wm.writePump(client)

	// Send connection welcome message
	wm.sendWelcome(client)
}

// readPump handles incoming messages from client
func (wm *WebSocketManager) readPump(client *Client) {
	defer func() {
		wm.unregisterClient(client)
		client.Conn.Close()
	}()

	client.Conn.SetReadLimit(1024) // Increased limit for complex messages
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		wm.handleMessage(client, message)
	}
}

// writePump handles outgoing messages to client
func (wm *WebSocketManager) writePump(client *Client) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel closed
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			w.Write(message)

			// Flush
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// Send ping
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles incoming client messages
func (wm *WebSocketManager) handleMessage(client *Client, message []byte) {
	var req struct {
		Action     string   `json:"action"`
		Interface  string   `json:"interface"`
		Interfaces []string `json:"interfaces"`
	}

	if err := json.Unmarshal(message, &req); err != nil {
		wm.sendError(client, "Invalid message format")
		return
	}

	switch req.Action {
	case "subscribe":
		if req.Interface != "" {
			wm.subscribeClient(client, []string{req.Interface})
		} else if len(req.Interfaces) > 0 {
			wm.subscribeClient(client, req.Interfaces)
		}
	case "unsubscribe":
		if req.Interface != "" {
			wm.unsubscribeClient(client, []string{req.Interface})
		} else if len(req.Interfaces) > 0 {
			wm.unsubscribeClient(client, req.Interfaces)
		}
	case "ping":
		wm.sendPong(client)
	case "get_status":
		wm.sendStatus(client)
	default:
		wm.sendError(client, "Unknown action")
	}
}

// subscribeClient subscribes a client to interface updates
func (wm *WebSocketManager) subscribeClient(client *Client, interfaces []string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	for _, iface := range interfaces {
		if _, exists := wm.subscriptions[iface]; !exists {
			wm.subscriptions[iface] = make(map[*Client]bool)
		}
		wm.subscriptions[iface][client] = true
	}

	wm.sendSuccess(client, fmt.Sprintf("Subscribed to interfaces: %v", interfaces))

	// Notify event bus of subscription
	wm.eventBus.Publish(EventData{
		Type:      "subscription",
		Message:   fmt.Sprintf("Client %s subscribed to %v", client.ID, interfaces),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"client_id":    client.ID,
			"interfaces":   interfaces,
			"connected_at": client.Connected,
		},
	})
}

// unsubscribeClient unsubscribes a client from interface updates
func (wm *WebSocketManager) unsubscribeClient(client *Client, interfaces []string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	for _, iface := range interfaces {
		if clients, exists := wm.subscriptions[iface]; exists {
			delete(clients, client)
			if len(clients) == 0 {
				delete(wm.subscriptions, iface)
			}
		}
	}

	wm.sendSuccess(client, fmt.Sprintf("Unsubscribed from interfaces: %v", interfaces))
}

// unregisterClient removes a client from all subscriptions
func (wm *WebSocketManager) unregisterClient(client *Client) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	delete(wm.clients, client.ID)

	for iface, clients := range wm.subscriptions {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			if len(clients) == 0 {
				delete(wm.subscriptions, iface)
			}
		}
	}
}

// sendError sends an error message to client
func (wm *WebSocketManager) sendError(client *Client, message string) {
	resp := map[string]interface{}{
		"type":    "error",
		"message": message,
		"time":    time.Now(),
	}
	data, _ := json.Marshal(resp)
	client.Send <- data
}

// sendSuccess sends a success message to client
func (wm *WebSocketManager) sendSuccess(client *Client, message string) {
	resp := map[string]interface{}{
		"type":    "success",
		"message": message,
		"time":    time.Now(),
	}
	data, _ := json.Marshal(resp)
	client.Send <- data
}

// sendPong sends a pong response to client
func (wm *WebSocketManager) sendPong(client *Client) {
	resp := map[string]interface{}{
		"type": "pong",
		"time": time.Now(),
	}
	data, _ := json.Marshal(resp)
	client.Send <- data
}

// sendWelcome sends a welcome message to newly connected client
func (wm *WebSocketManager) sendWelcome(client *Client) {
	resp := map[string]interface{}{
		"type":    "welcome",
		"message": "Connected to Monik Monitoring WebSocket",
		"time":    time.Now(),
		"metrics": wm.metrics.GetStats(),
	}
	data, _ := json.Marshal(resp)
	client.Send <- data
}

// sendStatus sends current status to client
func (wm *WebSocketManager) sendStatus(client *Client) {
	resp := map[string]interface{}{
		"type":          "status",
		"message":       "Current WebSocket status",
		"time":          time.Now(),
		"metrics":       wm.metrics.GetStats(),
		"subscriptions": wm.GetSubscriptions(),
	}
	data, _ := json.Marshal(resp)
	client.Send <- data
}

// serializeData serializes real-time data to JSON
func (wm *WebSocketManager) serializeData(data RealTimeData) []byte {
	resp := map[string]interface{}{
		"type":       "data",
		"interface":  data.InterfaceName,
		"rx_rate":    data.RxRate,
		"tx_rate":    data.TxRate,
		"rx_bytes":   data.RxBytes,
		"tx_bytes":   data.TxBytes,
		"status":     data.Status,
		"comment":    data.Comment,
		"timestamp":  data.Timestamp,
		"event_type": data.EventType,
	}
	jsonData, _ := json.Marshal(resp)
	return jsonData
}

// serializeEvent serializes event data to JSON
func (wm *WebSocketManager) serializeEvent(data EventData) []byte {
	resp := map[string]interface{}{
		"type":      "event",
		"event":     data.Type,
		"message":   data.Message,
		"timestamp": data.Timestamp,
		"data":      data.Data,
	}
	jsonData, _ := json.Marshal(resp)
	return jsonData
}

// BroadcastData broadcasts real-time data to subscribed clients
func (wm *WebSocketManager) BroadcastData(data RealTimeData) {
	select {
	case wm.broadcast <- data:
		wm.metrics.RecordBroadcast()
	default:
		// Channel full, drop message
		wm.metrics.RecordBroadcastDropped()
		log.Printf("Broadcast channel full, dropping message for %s", data.InterfaceName)
	}
}

// BroadcastEvent broadcasts an event to all clients
func (wm *WebSocketManager) BroadcastEvent(eventType, message string, data interface{}) {
	eventData := EventData{
		Type:      eventType,
		Message:   message,
		Timestamp: time.Now(),
		Data:      data.(map[string]interface{}),
	}

	select {
	case wm.broadcast <- eventData:
		wm.metrics.RecordEventBroadcast()
	default:
		wm.metrics.RecordEventBroadcastDropped()
		log.Printf("Event broadcast channel full, dropping event: %s", eventType)
	}
}

// GetClientCount returns the number of connected clients
func (wm *WebSocketManager) GetClientCount() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return len(wm.clients)
}

// GetSubscriptions returns current subscriptions
func (wm *WebSocketManager) GetSubscriptions() map[string]int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	result := make(map[string]int)
	for iface, clients := range wm.subscriptions {
		result[iface] = len(clients)
	}
	return result
}

// GetMetrics returns WebSocket metrics
func (wm *WebSocketManager) GetMetrics() *WebSocketMetrics {
	return wm.metrics
}

// EventBus handles internal event notifications
type EventBus struct {
	subscribers map[string][]chan EventData
	mu          sync.RWMutex
}

// EventData represents an event in the system
type EventData struct {
	Type      string                 `json:"type"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]chan EventData),
	}
}

// Start starts the event bus
func (eb *EventBus) Start() {
	// Event bus is now integrated into WebSocketManager
}

// Publish publishes an event to all subscribers
func (eb *EventBus) Publish(event EventData) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	for _, subscribers := range eb.subscribers {
		for _, ch := range subscribers {
			select {
			case ch <- event:
			default:
				// Channel full, skip
			}
		}
	}
}

// Subscribe subscribes to events
func (eb *EventBus) Subscribe(eventType string) chan EventData {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan EventData, 100)
	eb.subscribers[eventType] = append(eb.subscribers[eventType], ch)
	return ch
}

// WebSocketMetrics tracks WebSocket performance
type WebSocketMetrics struct {
	messagesSent        int64
	messagesDropped     int64
	broadcastsSent      int64
	broadcastsDropped   int64
	eventsSent          int64
	eventsDropped       int64
	connectionsTotal    int64
	disconnectionsTotal int64
	mu                  sync.RWMutex
}

// NewWebSocketMetrics creates new WebSocket metrics
func NewWebSocketMetrics() *WebSocketMetrics {
	return &WebSocketMetrics{}
}

// RecordMessageSent records a sent message
func (wm *WebSocketMetrics) RecordMessageSent() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.messagesSent++
}

// RecordMessageDropped records a dropped message
func (wm *WebSocketMetrics) RecordMessageDropped() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.messagesDropped++
}

// RecordBroadcast records a broadcast
func (wm *WebSocketMetrics) RecordBroadcast() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.broadcastsSent++
}

// RecordBroadcastDropped records a dropped broadcast
func (wm *WebSocketMetrics) RecordBroadcastDropped() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.broadcastsDropped++
}

// RecordEventBroadcast records an event broadcast
func (wm *WebSocketMetrics) RecordEventBroadcast() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.eventsSent++
}

// RecordEventBroadcastDropped records a dropped event broadcast
func (wm *WebSocketMetrics) RecordEventBroadcastDropped() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.eventsDropped++
}

// RecordConnection records a new connection
func (wm *WebSocketMetrics) RecordConnection() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.connectionsTotal++
}

// RecordDisconnection records a disconnection
func (wm *WebSocketMetrics) RecordDisconnection() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.disconnectionsTotal++
}

// GetStats returns current metrics stats
func (wm *WebSocketMetrics) GetStats() map[string]interface{} {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	return map[string]interface{}{
		"messages_sent":        wm.messagesSent,
		"messages_dropped":     wm.messagesDropped,
		"broadcasts_sent":      wm.broadcastsSent,
		"broadcasts_dropped":   wm.broadcastsDropped,
		"events_sent":          wm.eventsSent,
		"events_dropped":       wm.eventsDropped,
		"connections_total":    wm.connectionsTotal,
		"disconnections_total": wm.disconnectionsTotal,
		"drop_rate":            float64(wm.messagesDropped) / float64(wm.messagesSent+wm.messagesDropped+1),
	}
}
