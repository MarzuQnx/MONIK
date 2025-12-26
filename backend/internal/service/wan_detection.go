package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"monik-enterprise/internal/config"
	"monik-enterprise/internal/websocket"

	"github.com/go-routeros/routeros/v3"
)

// WANDetectionService handles WAN/ISP interface detection logic
type WANDetectionService struct {
	client       *routeros.Client
	config       config.WANDetectionConfig
	cache        *WANDetectionCache
	mu           sync.RWMutex
	lastUpdate   time.Time
	websocketMgr *websocket.WebSocketManager
	metrics      *WANDetectionMetrics
}

type WANDetectionCache struct {
	Interface   *WANInterface
	LastUpdated time.Time
}

type WANInterface struct {
	Name        string    `json:"name"`
	Method      string    `json:"method"`     // route, traffic, pattern, manual
	Confidence  float64   `json:"confidence"` // 0.0 to 1.0
	LastUpdated time.Time `json:"last_updated"`
	Traffic     uint64    `json:"traffic"`  // bytes
	ISPName     string    `json:"isp_name"` // Detected ISP name
}

const (
	DetectionMethodRoute   = "default_route"
	DetectionMethodTraffic = "traffic_analysis"
	DetectionMethodPattern = "name_pattern"
	DetectionMethodManual  = "manual"
)

// Regex patterns for ISP identification
var ispPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)wan`),
	regexp.MustCompile(`(?i)isp`),
	regexp.MustCompile(`(?i)pppoe`),
	regexp.MustCompile(`(?i)sumber`), // Tambahan keyword: SUMBER
	regexp.MustCompile(`(?i)ether.*wan`),
	regexp.MustCompile(`(?i)bridge.*wan`),
}

var ispNamePatterns = map[string]*regexp.Regexp{
	"telkom":   regexp.MustCompile(`(?i)(telkom|indihome|indihomo)`),
	"indosat":  regexp.MustCompile(`(?i)(indosat|im3|mentari)`),
	"xl":       regexp.MustCompile(`(?i)(xl|axis)`),
	"starlink": regexp.MustCompile(`(?i)(starlink|strlnk)`),
	"biznet":   regexp.MustCompile(`(?i)biznet`),
}

// NewWANDetectionService creates a new detection service
func NewWANDetectionService(cfg config.WANDetectionConfig) *WANDetectionService {
	return &WANDetectionService{
		config: cfg,
		cache: &WANDetectionCache{
			Interface:   nil,
			LastUpdated: time.Time{},
		},
		metrics: NewWANDetectionMetrics(),
	}
}

func (s *WANDetectionService) SetRouterClient(client *routeros.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.client = client
}

func (s *WANDetectionService) SetWebSocketManager(wsMgr *websocket.WebSocketManager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.websocketMgr = wsMgr
}

// ensureConnected melakukan lazy connection dan pengecekan nil
func (s *WANDetectionService) ensureConnected(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Jika client nil atau sudah tertutup, coba hubungkan kembali
	if s.client == nil {
		// Menggunakan RouterConfig dari service (diasumsikan diset sebelumnya)
		// Jika tidak ada, kembalikan error karena tidak bisa membuat koneksi
		return fmt.Errorf("routeros client is nil and no router config available")
	}

	// Gunakan ctx untuk melakukan ping koneksi jika perlu
	// Ini memastikan parameter ctx digunakan dan menghilangkan warning unusedparams
	if s.client != nil {
		// Coba lakukan ping ringan untuk memastikan koneksi masih aktif
		_, err := s.client.RunContext(ctx, "/system/resource/print")
		if err != nil {
			// Jika ping gagal, set client ke nil agar reconnect pada attempt berikutnya
			s.client = nil
			return fmt.Errorf("connection lost: %w", err)
		}
	}
	return nil
}

// DetectWANInterface is the main entry point for WAN detection
func (s *WANDetectionService) DetectWANInterface(ctx context.Context) (*WANInterface, error) {
	// 1. CEK KONEKSI SEBELUM MULAI (Mencegah Panic)
	if err := s.ensureConnected(ctx); err != nil {
		fmt.Printf("[WAN-ERROR] Connection failed: %v\n", err)
		return &WANInterface{
			Name:        "none",
			Method:      "error",
			Confidence:  0.0,
			LastUpdated: time.Now(),
			ISPName:     "error",
		}, nil
	}

	s.mu.RLock()
	if s.cache.Interface != nil && time.Since(s.cache.LastUpdated) < s.config.CacheDuration {
		s.mu.RUnlock()
		s.metrics.RecordCacheHit()
		return s.cache.Interface, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	var bestWAN *WANInterface
	var detectionMethod string
	var confidence float64

	// Hybrid Detection Logic
	switch s.config.DetectionMethod {
	case "auto", "hybrid":
		bestWAN, detectionMethod, confidence = s.detectByHybrid(ctx)
	default:
		if wan := s.detectByRoute(ctx); wan != nil {
			bestWAN = wan
			detectionMethod = DetectionMethodRoute
			confidence = 0.95
		}
	}

	if bestWAN != nil {
		bestWAN.Method = detectionMethod
		bestWAN.Confidence = confidence
		bestWAN.LastUpdated = time.Now()
		bestWAN.ISPName = s.detectISPName(bestWAN.Name)

		s.cache.Interface = bestWAN
		s.cache.LastUpdated = time.Now()
		s.metrics.RecordDetection(detectionMethod, confidence)
		s.notifyWANDetected(bestWAN)
		return bestWAN, nil
	}

	s.metrics.RecordDetectionFailure()
	return &WANInterface{
		Name:        "none",
		Method:      "not_found",
		Confidence:  0.0,
		LastUpdated: time.Now(),
		ISPName:     "unknown",
	}, nil
}

// detectByRoute finds WAN based on the active default gateway (0.0.0.0/0)
func (s *WANDetectionService) detectByRoute(ctx context.Context) *WANInterface {
	if s.client == nil {
		return nil
	}

	// Filter only active routes to avoid picking a down ISP
	reply, err := s.client.RunContext(ctx, "/ip/route/print", "?dst-address=0.0.0.0/0", "?active=true")
	if err != nil {
		// Jika error karena broken pipe, set client ke nil agar reconnect pada attempt berikutnya
		s.client = nil
		return nil
	}
	if len(reply.Re) == 0 {
		return nil
	}

	for _, re := range reply.Re {
		route := re.Map
		ifaceName := ""

		// Handle MikroTik format: "gateway%interface" (e.g., 192.168.1.1%ether1)
		immGw := route["immediate-gw"]
		if immGw != "" {
			if strings.Contains(immGw, "%") {
				parts := strings.Split(immGw, "%")
				ifaceName = parts[len(parts)-1]
			} else {
				ifaceName = immGw
			}
		}

		if ifaceName == "" {
			ifaceName = route["interface"]
		}

		if ifaceName != "" {
			// Cross-check if the interface is actually RUNNING
			iface, err := s.getInternalInterfaceDetails(ctx, ifaceName)
			if err == nil && iface.Status == "true" {
				return &WANInterface{
					Name:        ifaceName,
					Method:      DetectionMethodRoute,
					Confidence:  0.95,
					LastUpdated: time.Now(),
					Traffic:     iface.RxBytes + iface.TxBytes,
				}
			}
		}
	}
	return nil
}

func (s *WANDetectionService) detectByHybrid(ctx context.Context) (*WANInterface, string, float64) {
	routeWAN := s.detectByRoute(ctx)
	trafficWAN := s.detectByTraffic(ctx)
	patternWAN := s.detectByPattern(ctx)

	scores := make(map[string]float64)
	interfaces := make(map[string]*WANInterface)

	if routeWAN != nil {
		scores[routeWAN.Name] += 0.95
		interfaces[routeWAN.Name] = routeWAN
	}

	if trafficWAN != nil {
		scores[trafficWAN.Name] += 0.70
		if _, exists := interfaces[trafficWAN.Name]; !exists {
			interfaces[trafficWAN.Name] = trafficWAN
		}
	}

	if patternWAN != nil {
		iface, _ := s.getInternalInterfaceDetails(ctx, patternWAN.Name)
		if iface != nil && iface.Status == "true" {
			scores[patternWAN.Name] += 0.50
		}
		if _, exists := interfaces[patternWAN.Name]; !exists {
			interfaces[patternWAN.Name] = patternWAN
		}
	}

	var bestWAN *WANInterface
	var maxScore float64
	for name, score := range scores {
		if score > maxScore {
			maxScore = score
			bestWAN = interfaces[name]
		}
	}

	method := DetectionMethodPattern
	if maxScore >= 0.90 {
		method = DetectionMethodRoute
	} else if maxScore >= 0.70 {
		method = DetectionMethodTraffic
	}

	return bestWAN, method, maxScore
}

func (s *WANDetectionService) detectByTraffic(ctx context.Context) *WANInterface {
	interfaces, err := s.getAllInternalInterfaces(ctx)
	if err != nil {
		return nil
	}

	var bestWAN *WANInterface
	maxTraffic := uint64(0)

	for _, iface := range interfaces {
		// Ignore bridges and down interfaces
		if iface.Status != "true" || strings.HasPrefix(iface.Name, "bridge") {
			continue
		}
		total := iface.RxBytes + iface.TxBytes
		if total > maxTraffic {
			maxTraffic = total
			bestWAN = &WANInterface{
				Name:        iface.Name,
				Method:      DetectionMethodTraffic,
				Confidence:  0.7,
				LastUpdated: time.Now(),
				Traffic:     total,
			}
		}
	}
	return bestWAN
}

func (s *WANDetectionService) detectByPattern(ctx context.Context) *WANInterface {
	interfaces, err := s.getAllInternalInterfaces(ctx)
	if err != nil {
		return nil
	}

	for _, iface := range interfaces {
		// Hanya cek interface yang aktif
		if iface.Status != "true" {
			continue
		}

		// Cek setiap pola pada Nama Interface DAN Comment
		for _, pattern := range ispPatterns {
			if pattern.MatchString(iface.Name) || pattern.MatchString(iface.Comment) {
				return &WANInterface{
					Name:        iface.Name,
					Method:      DetectionMethodPattern,
					Confidence:  0.6, // Confidence naik karena ada kecocokan eksplisit
					LastUpdated: time.Now(),
				}
			}
		}
	}
	return nil
}

func (s *WANDetectionService) detectISPName(name string) string {
	for isp, pattern := range ispNamePatterns {
		if pattern.MatchString(name) {
			return isp
		}
	}
	return "unknown"
}

func (s *WANDetectionService) notifyWANDetected(wan *WANInterface) {
	if s.websocketMgr != nil {
		s.websocketMgr.BroadcastEvent(websocket.EventTypeWANDetected,
			fmt.Sprintf("WAN interface detected: %s", wan.Name),
			map[string]interface{}{"name": wan.Name, "isp": wan.ISPName})
	}
}

// --- INTERNAL HELPERS ---
// Menggunakan InterfaceData yang sudah didefinisikan di mikrotik.go

func (s *WANDetectionService) getInternalInterfaceDetails(ctx context.Context, name string) (*InterfaceData, error) {
	if s.client == nil {
		return nil, fmt.Errorf("routeros client is nil")
	}

	reply, err := s.client.RunContext(ctx, "/interface/print", "?name="+name)
	if err != nil {
		// Jika error karena broken pipe, set client ke nil agar reconnect pada attempt berikutnya
		s.client = nil
		return nil, err
	}
	if len(reply.Re) == 0 {
		return nil, fmt.Errorf("interface not found")
	}
	re := reply.Re[0].Map
	return &InterfaceData{
		Name:    name,
		RxBytes: parseUint64(re["rx-byte"]),
		TxBytes: parseUint64(re["tx-byte"]),
		Status:  re["running"],
	}, nil
}

func (s *WANDetectionService) getAllInternalInterfaces(ctx context.Context) ([]*InterfaceData, error) {
	if s.client == nil {
		return nil, fmt.Errorf("routeros client is nil")
	}

	// Request data interface termasuk comment
	reply, err := s.client.RunContext(ctx, "/interface/print")
	if err != nil {
		s.client = nil
		return nil, err
	}

	var res []*InterfaceData
	for _, re := range reply.Re {
		res = append(res, &InterfaceData{
			Name:    re.Map["name"],
			RxBytes: parseUint64(re.Map["rx-byte"]),
			TxBytes: parseUint64(re.Map["tx-byte"]),
			Status:  re.Map["running"],
			Comment: re.Map["comment"], // Pastikan kolom comment diambil
		})
	}
	return res, nil
}

func (s *WANDetectionService) GetCachedWANInterface() *WANInterface {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cache.Interface
}

// GetDetectionStats mengembalikan objek metrics untuk dikirim ke API
func (s *WANDetectionService) GetDetectionStats() *WANDetectionMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metrics
}
