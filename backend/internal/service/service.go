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
		fmt.Printf("[MONITORING] Service already running\n")
		return
	}
	fmt.Printf("[MONITORING] Starting monitoring service...\n")
	s.isRunning = true
	s.wg.Add(1)
	go s.monitoringLoop()
	fmt.Printf("[MONITORING] Monitoring service started successfully\n")
}

func (s *MonitoringService) monitoringLoop() {
	defer s.wg.Done()
	fmt.Printf("[MONITORING] Monitoring loop started - collecting data every 10 seconds\n")
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopChan:
			fmt.Printf("[MONITORING] Stop signal received, exiting loop\n")
			return
		case <-ticker.C:
			fmt.Printf("[MONITORING] ===== TICK RECEIVED at %s =====\n", time.Now().Format("15:04:05"))
			s.collectData()
			fmt.Printf("[MONITORING] ===== DATA COLLECTION COMPLETE =====\n")
		}
	}
}

func (s *MonitoringService) collectData() {
	fmt.Printf("[DEBUG] === COLLECT DATA STARTED at %s ===\n", time.Now().Format("15:04:05"))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Priority 2 Fix: Implement Retry Logic & Anti-Early-Return
	var interfaces []InterfaceData
	var err error

	fmt.Printf("[DEBUG] Attempting to get interfaces from router...\n")

	// Retry up to 3 times when router is unreachable
	for attempt := 1; attempt <= 3; attempt++ {
		fmt.Printf("[DEBUG] Attempt %d: Getting interfaces from router\n", attempt)
		interfaces, err = s.routerSvc.GetInterfaces(ctx)
		if err == nil {
			fmt.Printf("[INFO] Router GMG-SITE connected successfully (attempt %d) - got %d interfaces\n", attempt, len(interfaces))
			break
		}
		fmt.Printf("[RETRY %d] Router unreachable, waiting 2s... Error: %v\n", attempt, err)
		time.Sleep(2 * time.Second)
	}

	// Critical Fix: JANGAN RETURN! Continue flow even when router is offline
	if err != nil {
		fmt.Printf("[CRITICAL] Router GMG-SITE is OFFLINE after retries: %v\n", err)
		fmt.Printf("[INFO] Recording offline status in database...\n")

		// Update all known interfaces as offline in database
		s.RecordOfflineStatus()
		fmt.Printf("[DEBUG] === COLLECT DATA COMPLETE (OFFLINE PATH) ===\n")
		return
	}

	fmt.Printf("[DEBUG] Router connected successfully, processing %d interfaces\n", len(interfaces))

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
				fmt.Printf("[DEBUG] Got traffic stats for %s: RxRate=%.2f, TxRate=%.2f\n", iface.Name, traffic.RxRate, traffic.TxRate)
			} else {
				fmt.Printf("[WARN] Failed to get traffic stats for %s: %v\n", iface.Name, err)
			}
			return nil
		})
	}
	g.Wait()

	fmt.Printf("[DEBUG] Saving interface data for %d interfaces\n", len(interfaces))
	for _, iface := range interfaces {
		if t, ok := trafficMap[iface.Name]; ok {
			iface.RxRate = t.RxRate
			iface.TxRate = t.TxRate
			fmt.Printf("[DEBUG] Updated rates for %s: RxRate=%.2f, TxRate=%.2f\n", iface.Name, iface.RxRate, iface.TxRate)
		} else {
			// Set rates to 0 if traffic stats failed
			iface.RxRate = 0
			iface.TxRate = 0
			fmt.Printf("[DEBUG] No traffic stats for %s, setting rates to 0\n", iface.Name)
		}
		s.saveInterfaceData(iface)
	}
	fmt.Printf("[DEBUG] === COLLECT DATA COMPLETE (ONLINE PATH) ===\n")
}

func (s *MonitoringService) saveInterfaceData(iface InterfaceData) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	fmt.Printf("[DEBUG] saveInterfaceData called for %s: Rx=%d, Tx=%d\n", iface.Name, iface.RxBytes, iface.TxBytes)

	var existing models.Interface
	res := s.db.Where("interface_name = ?", iface.Name).First(&existing)

	isReset := false
	if res.Error == nil && (iface.RxBytes < existing.RxBytes || iface.TxBytes < existing.TxBytes) {
		isReset = true
		fmt.Printf("[WARN] Reset detected on %s at %s\n", iface.Name, time.Now().Format("15:04:05"))
		fmt.Printf("[DEBUG] Reset Details: New Rx=%d < Old Rx=%d OR New Tx=%d < Old Tx=%d\n", iface.RxBytes, existing.RxBytes, iface.TxBytes, existing.TxBytes)
	} else {
		fmt.Printf("[DEBUG] No reset detected for %s. isReset=%v\n", iface.Name, isReset)
		if res.Error == nil {
			fmt.Printf("[DEBUG] Comparison: New Rx=%d vs Old Rx=%d, New Tx=%d vs Old Tx=%d\n", iface.RxBytes, existing.RxBytes, iface.TxBytes, existing.TxBytes)
		} else {
			fmt.Printf("[DEBUG] No existing record found for %s\n", iface.Name)
		}
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

	// Update MonthlyQuota untuk semua interface
	now := time.Now()
	if err := s.updateMonthlyQuota(iface, isReset, now); err != nil {
		fmt.Printf("[ERROR] Gagal update MonthlyQuota untuk %s: %v\n", iface.Name, err)
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

// RecordOfflineStatus updates all interfaces with offline status when router is unreachable
func (s *MonitoringService) RecordOfflineStatus() {
	fmt.Printf("[SELF-HEALING] Starting recordOfflineStatus() method\n")

	// Get all known interfaces from database
	var knownInterfaces []models.Interface
	err := s.db.Find(&knownInterfaces).Error
	if err != nil {
		fmt.Printf("[ERROR] Failed to fetch known interfaces: %v\n", err)
		return
	}

	fmt.Printf("[SELF-HEALING] Found %d known interfaces to record offline status\n", len(knownInterfaces))

	now := time.Now()
	for _, iface := range knownInterfaces {
		// Update interface with offline status
		iface.LastSeen = now
		// Set rates to 0 when offline
		iface.RxRate = 0
		iface.TxRate = 0

		fmt.Printf("[SELF-HEALING] Recording offline status for %s (Rx: %d, Tx: %d)\n",
			iface.InterfaceName, iface.RxBytes, iface.TxBytes)

		// Save to database
		updateErr := s.db.Model(&iface).Updates(map[string]interface{}{
			"last_seen": now,
			"rx_rate":   0,
			"tx_rate":   0,
		}).Error
		if updateErr != nil {
			fmt.Printf("[ERROR] Failed to update interface %s: %v\n", iface.InterfaceName, updateErr)
		}

		// Still call updateMonthlyQuota even when offline to maintain data consistency
		interfaceData := InterfaceData{
			Name:    iface.InterfaceName,
			RxBytes: iface.RxBytes,
			TxBytes: iface.TxBytes,
			RxRate:  0,
			TxRate:  0,
		}

		fmt.Printf("[SELF-HEALING] Calling updateMonthlyQuota for %s\n", iface.InterfaceName)
		if err := s.updateMonthlyQuota(interfaceData, false, now); err != nil {
			fmt.Printf("[ERROR] updateMonthlyQuota failed for %s: %v\n", iface.InterfaceName, err)
		}
	}
}

// updateMonthlyQuota mengupdate atau membuat record MonthlyQuota berdasarkan data interface
func (s *MonitoringService) updateMonthlyQuota(iface InterfaceData, isReset bool, now time.Time) error {
	fmt.Printf("[DEBUG-QUOTA] Processing %s | Rx: %d | Reset: %v\n", iface.Name, iface.RxBytes, isReset)

	dbMutex.Lock()
	defer dbMutex.Unlock()

	fmt.Printf("[DEBUG] updateMonthlyQuota called for %s: isReset=%v, Rx=%d, Tx=%d\n", iface.Name, isReset, iface.RxBytes, iface.TxBytes)

	// Ekstrak informasi tanggal dari waktu saat ini
	day := now.Day()
	month := int(now.Month())
	year := now.Year()

	fmt.Printf("[DEBUG] Current date context: Day=%d, Month=%d, Year=%d\n", day, month, year)

	// Cari record MonthlyQuota berdasarkan interface_name, day, month, year
	var quota models.MonthlyQuota
	err := s.db.Where("interface_name = ? AND day = ? AND month = ? AND year = ?",
		iface.Name, day, month, year).First(&quota).Error

	if err == gorm.ErrRecordNotFound {
		fmt.Printf("[DEBUG] No existing quota record found for %s on %d/%d/%d\n", iface.Name, day, month, year)
		// Inisialisasi record hari baru
		newQuota := models.MonthlyQuota{
			InterfaceName: iface.Name,
			Day:           day,
			Month:         month,
			Year:          year,
			RxBytes:       0,
			TxBytes:       0,
			TotalBytes:    0,
			TotalRx:       0,
			TotalTx:       0,
			LastRxBytes:   iface.RxBytes,
			LastTxBytes:   iface.TxBytes,
		}
		fmt.Printf("[DEBUG] Creating new quota record with LastRxBytes=%d, LastTxBytes=%d\n", iface.RxBytes, iface.TxBytes)
		err := s.db.Create(&newQuota).Error
		if err != nil {
			fmt.Printf("[ERROR] Failed to create new quota record: %v\n", err)
			return err
		}
		fmt.Printf("[SUCCESS] Successfully created new quota record for %s\n", iface.Name)
		return nil
	} else if err != nil {
		fmt.Printf("[ERROR] Database error when querying quota: %v\n", err)
		return err
	}

	fmt.Printf("[DEBUG] Found existing quota record for %s\n", iface.Name)

	var deltaRx, deltaTx uint64

	// Hitung Delta berdasarkan nilai counter terakhir yang tercatat di tabel Quota
	if isReset || iface.RxBytes < quota.LastRxBytes {
		// Skenario Reset: Ambil nilai baru seutuhnya sebagai delta
		deltaRx = iface.RxBytes
		deltaTx = iface.TxBytes
		fmt.Printf("[DEBUG] RESET SCENARIO: Taking full values as delta - deltaRx=%d, deltaTx=%d\n", deltaRx, deltaTx)
	} else {
		// Skenario Normal: Selisih antara counter sekarang dengan counter terakhir yang dicatat
		// Additional validation to prevent false reset detection
		if iface.RxBytes >= quota.LastRxBytes && iface.TxBytes >= quota.LastTxBytes {
			deltaRx = iface.RxBytes - quota.LastRxBytes
			deltaTx = iface.TxBytes - quota.LastTxBytes
			fmt.Printf("[DEBUG] NORMAL SCENARIO: Calculating difference - deltaRx=%d (%d - %d), deltaTx=%d (%d - %d)\n",
				deltaRx, iface.RxBytes, quota.LastRxBytes, deltaTx, iface.TxBytes, quota.LastTxBytes)
		} else {
			// Additional protection: if values are unexpectedly lower, treat as reset
			fmt.Printf("[WARN] Unexpected lower values detected: Rx=%d < LastRx=%d OR Tx=%d < LastTx=%d\n",
				iface.RxBytes, quota.LastRxBytes, iface.TxBytes, quota.LastTxBytes)
			deltaRx = iface.RxBytes
			deltaTx = iface.TxBytes
			fmt.Printf("[DEBUG] PROTECTION SCENARIO: Using full values as delta - deltaRx=%d, deltaTx=%d\n", deltaRx, deltaTx)
		}
	}

	// Update akumulasi harian dan perbarui tracker counter terakhir
	fmt.Printf("[DEBUG] Updating quota: Current RxBytes=%d, TxBytes=%d, adding deltaRx=%d, deltaTx=%d\n",
		quota.RxBytes, quota.TxBytes, deltaRx, deltaTx)
	err = s.db.Model(&quota).Updates(map[string]interface{}{
		"rx_bytes":      quota.RxBytes + deltaRx,
		"tx_bytes":      quota.TxBytes + deltaTx,
		"total_bytes":   (quota.RxBytes + deltaRx) + (quota.TxBytes + deltaTx),
		"total_rx":      quota.TotalRx + deltaRx,
		"total_tx":      quota.TotalTx + deltaTx,
		"last_rx_bytes": iface.RxBytes,
		"last_tx_bytes": iface.TxBytes,
	}).Error
	if err != nil {
		fmt.Printf("[ERROR] Failed to update quota record: %v\n", err)
		return err
	}
	fmt.Printf("[SUCCESS] Successfully updated quota record for %s\n", iface.Name)
	return nil
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

// GetMonthlyQuota mengambil data MonthlyQuota berdasarkan interface, bulan, dan tahun
func (s *MonitoringService) GetMonthlyQuota(interfaceName string, month, year int) ([]models.MonthlyQuota, error) {
	var quotas []models.MonthlyQuota
	err := s.db.Where("interface_name = ? AND month = ? AND year = ?",
		interfaceName, month, year).
		Order("day ASC").
		Find("quotas").Error
	return quotas, err
}

// GetMonthlyQuotaByDay mengambil data MonthlyQuota untuk hari tertentu
func (s *MonitoringService) GetMonthlyQuotaByDay(interfaceName string, day, month, year int) (*models.MonthlyQuota, error) {
	var quota models.MonthlyQuota
	err := s.db.Where("interface_name = ? AND day = ? AND month = ? AND year = ?",
		interfaceName, day, month, year).
		First("quota").Error
	if err != nil {
		return nil, err
	}
	return &quota, nil
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
