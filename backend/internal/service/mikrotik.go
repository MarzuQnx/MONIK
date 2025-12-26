package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"monik-enterprise/internal/config"

	"github.com/go-routeros/routeros/v3"
)

// MikroTikService handles communication with MikroTik router
type MikroTikService struct {
	client *routeros.Client
	config config.RouterConfig
	mu     sync.Mutex
}

// InterfaceData represents interface monitoring data
type InterfaceData struct {
	Name        string    `json:"name"`
	RxBytes     uint64    `json:"rx_bytes"`
	TxBytes     uint64    `json:"tx_bytes"`
	RxRate      float64   `json:"rx_rate"` // Mbps
	TxRate      float64   `json:"tx_rate"` // Mbps
	Status      string    `json:"status"`
	Comment     string    `json:"comment"`
	LastUpdated time.Time `json:"last_updated"`
}

// SystemInfo represents system information
type SystemInfo struct {
	Identity  string `json:"identity"`
	BoardName string `json:"board_name"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	CPU       string `json:"cpu"`
	Memory    string `json:"memory"`
	Disk      string `json:"disk"`
	Timezone  string `json:"timezone"`
}

// NewMikroTikService creates a new MikroTik service
func NewMikroTikService(cfg config.RouterConfig) *MikroTikService {
	return &MikroTikService{
		config: cfg,
	}
}

// connect establishes connection to the router
func (s *MikroTikService) connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		return nil
	}

	address := fmt.Sprintf("%s:%d", s.config.IP, s.config.Port)
	client, err := routeros.DialContext(ctx, address, s.config.Username, s.config.Password)
	if err != nil {
		return fmt.Errorf("failed to connect to router: %w", err)
	}

	s.client = client
	return nil
}

// GetClient returns the router client (for internal use)
func (s *MikroTikService) GetClient() *routeros.Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client
}

// disconnect closes the connection
func (s *MikroTikService) disconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		s.client.Close()
		s.client = nil
	}
}

// GetInterfaces retrieves interface information from the router
func (s *MikroTikService) GetInterfaces(ctx context.Context) ([]InterfaceData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.connect(ctx); err != nil {
		return nil, err
	}

	reply, err := s.client.RunContext(ctx, "/interface/print")
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}

	var interfaces []InterfaceData
	for _, re := range reply.Re {
		iface := InterfaceData{
			Name:        re.Map["name"],
			Status:      re.Map["running"],
			Comment:     re.Map["comment"],
			LastUpdated: time.Now(),
		}

		// Parse RX/TX bytes safely
		iface.RxBytes = parseUint64(re.Map["rx-byte"])
		iface.TxBytes = parseUint64(re.Map["tx-byte"])

		interfaces = append(interfaces, iface)
	}

	return interfaces, nil
}

// GetSystemInfo retrieves system information from the router
func (s *MikroTikService) GetSystemInfo(ctx context.Context) (*SystemInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.connect(ctx); err != nil {
		return nil, err
	}

	info := &SystemInfo{}

	// Get identity
	reply, err := s.client.RunContext(ctx, "/system/identity/print")
	if err == nil && len(reply.Re) > 0 {
		info.Identity = reply.Re[0].Map["name"]
	}

	// Get resource info
	reply, err = s.client.RunContext(ctx, "/system/resource/print")
	if err == nil && len(reply.Re) > 0 {
		re := reply.Re[0].Map
		info.BoardName = re["board-name"]
		info.Version = re["version"]
		info.Uptime = re["uptime"]
		info.CPU = re["cpu-load"] + "%"
		info.Memory = re["free-memory"] + "/" + re["total-memory"]
	}

	// Get disk info
	reply, err = s.client.RunContext(ctx, "/system/resource/print")
	if err == nil && len(reply.Re) > 0 {
		re := reply.Re[0].Map
		if free, total := re["free-hdd-space"], re["total-hdd-space"]; free != "" && total != "" {
			info.Disk = free + "/" + total
		}
	}

	// Get timezone
	reply, err = s.client.RunContext(ctx, "/system/clock/print")
	if err == nil && len(reply.Re) > 0 {
		info.Timezone = reply.Re[0].Map["time-zone-name"]
	}

	return info, nil
}

// GetTrafficStats gets real-time traffic statistics
func (s *MikroTikService) GetTrafficStats(ctx context.Context, interfaceName string) (*InterfaceData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.connect(ctx); err != nil {
		return nil, err
	}

	reply, err := s.client.RunContext(ctx, "/interface/monitor-traffic",
		fmt.Sprintf("=interface=%s", interfaceName),
		"=once=")
	if err != nil {
		return nil, fmt.Errorf("failed to get traffic stats: %w", err)
	}

	if len(reply.Re) == 0 {
		return nil, fmt.Errorf("no data returned for interface %s", interfaceName)
	}

	re := reply.Re[0].Map
	data := &InterfaceData{
		Name:        interfaceName,
		Status:      "up", // Assume up if we can monitor
		LastUpdated: time.Now(),
	}

	// Parse rates (bits per second)
	if rxRate, err := parseRate(re["rx-bits-per-second"]); err == nil {
		data.RxRate = rxRate
	}
	if txRate, err := parseRate(re["tx-bits-per-second"]); err == nil {
		data.TxRate = txRate
	}

	return data, nil
}

// GetLastRebootLog retrieves the timestamp of the last reboot from router logs
func (s *MikroTikService) GetLastRebootLog(ctx context.Context) (time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.connect(ctx); err != nil {
		return time.Time{}, err
	}

	// Query logs for reboot events
	reply, err := s.client.RunContext(ctx, "/log/print",
		"where=topics~\"system\"",
		"?message~\"reboot\"|?message~\"started\"|?message~\"RouterOS\"")
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get logs: %w", err)
	}

	if len(reply.Re) == 0 {
		return time.Time{}, fmt.Errorf("no reboot logs found")
	}

	// Find the most recent reboot log
	var latestTime time.Time
	for _, re := range reply.Re {
		timeStr := re.Map["time"]
		if timeStr == "" {
			continue
		}

		// Parse time (format: "jan/01 12:34:56")
		parsedTime, err := parseMikroTikTime(timeStr)
		if err != nil {
			continue
		}

		if parsedTime.After(latestTime) {
			latestTime = parsedTime
		}
	}

	if latestTime.IsZero() {
		return time.Time{}, fmt.Errorf("could not parse any reboot time")
	}

	return latestTime, nil
}

// parseMikroTikTime parses MikroTik log time format (e.g., "jan/01 12:34:56")
func parseMikroTikTime(timeStr string) (time.Time, error) {
	// MikroTik log time is in format: "mmm/dd hh:mm:ss"
	// We need to add current year since it's not included
	now := time.Now()
	year := now.Year()

	// Parse with year
	fullTimeStr := fmt.Sprintf("%s %d", timeStr, year)
	t, err := time.Parse("jan/02 15:04:05 2006", fullTimeStr)
	if err != nil {
		return time.Time{}, err
	}

	// If parsed time is in the future (year wrap), use previous year
	if t.After(now) {
		t = t.AddDate(-1, 0, 0)
	}

	return t, nil
}

// parseRate converts rate string to Mbps float
func parseRate(rateStr string) (float64, error) {
	if rateStr == "" {
		return 0, nil
	}

	// Remove any suffix and parse as float
	rateStr = strings.TrimSuffix(rateStr, "bps")
	rate, err := strconv.ParseFloat(rateStr, 64)
	if err != nil {
		return 0, err
	}

	// Convert to Mbps
	return rate / 1000000, nil
}

// Close closes the service and cleans up resources
func (s *MikroTikService) Close() {
	s.disconnect()
}
