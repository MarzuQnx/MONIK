package service

import (
	"context"
	"fmt"
	"monik-enterprise/internal/models"
	"monik-enterprise/internal/websocket"
	"regexp"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- MONITORING SERVICE SECTION ---

var dbMutex sync.Mutex

type MonitoringService struct {
	db               *gorm.DB
	routerSvc        *MikroTikService
	wanService       *WANDetectionService
	websocketManager *websocket.WebSocketManager
	isRunning        bool
	stopChan         chan struct{}
	wg               sync.WaitGroup
}

func NewMonitoringService(db *gorm.DB, routerSvc *MikroTikService, wanService *WANDetectionService, wsManager *websocket.WebSocketManager) *MonitoringService {
	return &MonitoringService{
		db:               db,
		routerSvc:        routerSvc,
		wanService:       wanService,
		websocketManager: wsManager,
		stopChan:         make(chan struct{}),
	}
}

func (s *MonitoringService) Start() {
	if s.isRunning {
		return
	}
	s.isRunning = true
	s.wg.Add(1)
	go s.monitoringLoop()
}

func (s *MonitoringService) monitoringLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.collectData()
		}
	}
}

func (s *MonitoringService) collectData() {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	interfaces, err := s.routerSvc.GetInterfaces(ctx)
	if err != nil {
		fmt.Printf("[ERROR] Router unreachable: %v\n", err)
		return
	}

	trafficMap := make(map[string]*InterfaceData)
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)

	for _, iface := range interfaces {
		iface := iface
		g.Go(func() error {
			traffic, err := s.routerSvc.GetTrafficStats(gctx, iface.Name)
			if err == nil {
				mu.Lock()
				trafficMap[iface.Name] = traffic
				mu.Unlock()
			}
			return nil
		})
	}
	g.Wait()

	for _, iface := range interfaces {
		if t, ok := trafficMap[iface.Name]; ok {
			iface.RxRate = t.RxRate
			iface.TxRate = t.TxRate
		}
		s.saveInterfaceData(iface)
	}
}

func (s *MonitoringService) saveInterfaceData(iface InterfaceData) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	var existing models.Interface
	res := s.db.Where("interface_name = ?", iface.Name).First(&existing)

	isReset := false
	if res.Error == nil && (iface.RxBytes < existing.RxBytes || iface.TxBytes < existing.TxBytes) {
		isReset = true
		fmt.Printf("[WARN] Reset detected on %s at %s\n", iface.Name, time.Now().Format("15:04:05"))
	}

	s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "interface_name"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"rx_bytes": iface.RxBytes, "tx_bytes": iface.TxBytes,
			"rx_rate": iface.RxRate, "tx_rate": iface.TxRate,
			"last_seen": time.Now(),
		}),
	}).Create(&models.Interface{
		InterfaceName: iface.Name,
		RxBytes:       iface.RxBytes, TxBytes: iface.TxBytes,
		RxRate: iface.RxRate, TxRate: iface.TxRate,
		LastSeen: time.Now(),
	})

	if isReset {
		s.db.Create(&models.CounterResetLog{
			InterfaceName:   iface.Name,
			ResetTime:       time.Now(),
			PreviousBytes:   existing.RxBytes + existing.TxBytes,
			NewBytes:        iface.RxBytes + iface.TxBytes,
			DetectionMethod: "sudden_drop",
		})
	}

	if iface.Name == "xether2" {
		s.handleSnapshot(iface, isReset)
	}
}

func (s *MonitoringService) handleSnapshot(iface InterfaceData, isReset bool) {
	var last models.TrafficSnapshot
	curr := iface.RxBytes + iface.TxBytes
	err := s.db.Where("interface_name = ?", iface.Name).Order("timestamp DESC").First(&last).Error

	if err == gorm.ErrRecordNotFound || isReset || (curr-last.TotalBytes) > (10*1024*1024*1024) {
		s.db.Create(&models.TrafficSnapshot{
			InterfaceName: iface.Name,
			Timestamp:     time.Now(),
			RxBytes:       iface.RxBytes,
			TxBytes:       iface.TxBytes,
			TotalBytes:    curr,
			CounterReset:  isReset,
		})
		fmt.Printf("[INFO] Snapshot saved for xether2 | Total: %d bytes\n", curr)
	}
}

// Helpers
func filterComment(c string) string {
	return regexp.MustCompile("[^a-zA-Z0-9 ]+").ReplaceAllString(c, "")
}

// --- GETTER METHODS FOR API HANDLERS ---

// GetLatestInterfaces mengambil semua data interface terbaru dari database
func (s *MonitoringService) GetLatestInterfaces() ([]models.Interface, error) {
	var interfaces []models.Interface
	// Mengurutkan berdasarkan nama agar konsisten di UI
	err := s.db.Order("interface_name ASC").Find(&interfaces).Error
	return interfaces, err
}

// GetInterfaceByName mengambil satu data interface berdasarkan nama
func (s *MonitoringService) GetInterfaceByName(name string) (*models.Interface, error) {
	var iface models.Interface
	err := s.db.Where("interface_name = ?", name).First(&iface).Error
	if err != nil {
		return nil, err
	}
	return &iface, nil
}

// GetSystemInfo mengambil informasi sistem MikroTik terbaru dari DB
func (s *MonitoringService) GetSystemInfo() (*models.SystemInfo, error) {
	var info models.SystemInfo
	// Mengambil data terakhir yang diupdate
	err := s.db.Order("last_updated DESC").First(&info).Error
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// PopulateTestCounterResetLogs adalah fungsi helper untuk testing (jika Anda masih membutuhkannya di API)
func (s *MonitoringService) PopulateTestCounterResetLogs() error {
	testLogs := []models.CounterResetLog{
		{
			InterfaceName:   "xether2",
			ResetTime:       time.Now().Add(-1 * time.Hour),
			PreviousBytes:   5000000,
			NewBytes:        100,
			DetectionMethod: "manual_test",
		},
	}
	return s.db.Create(&testLogs).Error
}
