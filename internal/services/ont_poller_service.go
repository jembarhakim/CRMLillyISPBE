package services

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"skripsi-be/internal/genieacs"
	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OntPollerService handles background polling of ONT devices from GenieACS NBI
type OntPollerService struct {
	db       *gorm.DB
	client   *genieacs.Client
	config   *genieacs.ONTModelsConfig
	interval time.Duration
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

var (
	sharedOntPoller *OntPollerService
	ontPollerOnce   sync.Once
)

// InitSharedOntPoller initializes the singleton ONT poller
func InitSharedOntPoller(db *gorm.DB) *OntPollerService {
	ontPollerOnce.Do(func() {
		nbiURL := os.Getenv("GENIEACS_NBI_URL")
		if nbiURL == "" {
			nbiURL = "http://127.0.0.1:7557"
		}

		pollInterval := 60 * time.Second
		if customInterval := os.Getenv("ONT_POLL_INTERVAL"); customInterval != "" {
			if parsed, err := time.ParseDuration(customInterval); err == nil && parsed > 5*time.Second {
				pollInterval = parsed
			}
		}

		cfg, err := genieacs.LoadConfig("")
		if err != nil {
			log.Printf("[ONT Poller] Warning loading config: %v (using safe defaults)", err)
			cfg = genieacs.DefaultConfig()
		}

		client := genieacs.NewClient(nbiURL, 15*time.Second)

		sharedOntPoller = &OntPollerService{
			db:       db,
			client:   client,
			config:   cfg,
			interval: pollInterval,
		}
	})

	return sharedOntPoller
}

// GetSharedOntPoller returns the initialized ONT poller
func GetSharedOntPoller() *OntPollerService {
	return sharedOntPoller
}

// Start launches the poller goroutine
func (s *OntPollerService) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running = true
	s.mu.Unlock()

	log.Printf("[ONT Poller] Service started with polling interval %v against NBI %s", s.interval, s.client.BaseURL)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		// Run immediate initial poll after 3 seconds startup delay
		time.Sleep(3 * time.Second)
		if err := s.PollOnce(ctx); err != nil {
			log.Printf("[ONT Poller] Initial poll completed with notice: %v", err)
		}

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("[ONT Poller] Stopped cleanly on shutdown signal")
				return
			case <-ticker.C:
				if err := s.PollOnce(ctx); err != nil {
					log.Printf("[ONT Poller] Poll tick error: %v", err)
				}
			}
		}
	}()
}

// Stop stops the poller goroutine gracefully
func (s *OntPollerService) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()

	s.wg.Wait()
}

// PollOnce executes a single bulk query cycle from GenieACS NBI and persists latest status & time-series
func (s *OntPollerService) PollOnce(ctx context.Context) error {
	if s.client == nil || s.db == nil {
		return nil
	}

	// 1. Fetch all raw device documents in a SINGLE bulk query
	devices, err := s.client.GetDevices(ctx, "", "")
	if err != nil {
		return err
	}

	if len(devices) == 0 {
		return nil
	}

	// 2. Parse metrics for all returned devices into map by serial number
	metricsBySerial := make(map[string]*genieacs.OntMetrics)
	for _, doc := range devices {
		metrics := genieacs.ExtractMetrics(doc, s.config)
		if metrics != nil && metrics.SerialNumber != "" {
			metricsBySerial[metrics.SerialNumber] = metrics
			// Also index uppercase & trimmed versions for resilient matching
			metricsBySerial[strings.ToUpper(strings.TrimSpace(metrics.SerialNumber))] = metrics
		}
	}

	// 3. Collect every customer that has an ONT serial registered anywhere:
	//    customer.ont_serial_number  OR  network_devices.ont_serial_number  OR  asset_items.serial_number
	//    The three sources are unified in a single query so a customer is picked up
	//    regardless of which form the technician filled in.
	type customerWithSerial struct {
		ID              string
		OntSerialNumber *string
	}

	var customers []customerWithSerial
	serialUnionQuery := `
		SELECT c.id AS id, c.ont_serial_number AS ont_serial_number
		FROM customer c
		WHERE c.deleted_at IS NULL
		  AND c.ont_serial_number IS NOT NULL AND c.ont_serial_number <> ''
		UNION
		SELECT nd.customer_id AS id, nd.ont_serial_number AS ont_serial_number
		FROM network_devices nd
		JOIN customer c2 ON c2.id = nd.customer_id AND c2.deleted_at IS NULL
		WHERE nd.ont_serial_number IS NOT NULL AND nd.ont_serial_number <> ''
		UNION
		SELECT nd.customer_id AS id, ai.serial_number AS ont_serial_number
		FROM network_devices nd
		JOIN asset_items ai ON ai.id = nd.asset_item_id
		JOIN customer c3 ON c3.id = nd.customer_id AND c3.deleted_at IS NULL
		WHERE ai.serial_number IS NOT NULL AND ai.serial_number <> ''
	`

	if err := s.db.WithContext(ctx).Raw(serialUnionQuery).Scan(&customers).Error; err != nil {
		return err
	}

	// Deduplicate by customer ID, keeping the first non-empty serial found
	deduped := make([]customerWithSerial, 0, len(customers))
	seenCustomer := make(map[string]bool)
	for _, c := range customers {
		if c.ID == "" || seenCustomer[c.ID] {
			continue
		}
		seenCustomer[c.ID] = true
		deduped = append(deduped, c)
	}
	customers = deduped

	now := time.Now()

	// 4. Update OntStatus and record History for each matching customer
	for _, cust := range customers {
		if cust.OntSerialNumber == nil || *cust.OntSerialNumber == "" {
			continue
		}
		sn := strings.TrimSpace(*cust.OntSerialNumber)
		snUpper := strings.ToUpper(sn)

		metrics, found := metricsBySerial[sn]
		if !found {
			metrics, found = metricsBySerial[snUpper]
		}

		if found && metrics != nil {
			// Perangkat ditemukan di GenieACS
			status := entities.OntStatus{
				CustomerID:    cust.ID,
				SerialNumber:  sn,
				Manufacturer:  metrics.Manufacturer,
				ModelName:     metrics.ModelName,
				IsOnline:      metrics.IsOnline,
				RxPower:       metrics.RxPower,
				TxPower:       metrics.TxPower,
				SoftwareVersion: metrics.SoftwareVersion,
				IPAddress:     metrics.IPAddress,
				MacAddress:    metrics.MacAddress,
				OpticalStatus: string(metrics.OpticalStatus),
				SSID:          metrics.SSID,
				ClientCount:   metrics.ClientCount,
				LastInform:    metrics.LastInform,
				IsStale:       metrics.IsStale,
				UpdatedAt:     now,
			}

			// Upsert to ont_statuses (unique customer_id)
			if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "customer_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"serial_number", "manufacturer", "model_name", "is_online",
					"rx_power", "tx_power", "software_version", "ip_address", "mac_address", "optical_status", "ssid", "client_count",
					"last_inform", "is_stale", "updated_at",
				}),
			}).Create(&status).Error; err != nil {
				log.Printf("[ONT Poller] Failed to upsert status for customer %s: %v", cust.ID, err)
			}

			// Record history entry for time-series chart (only if rx_power is present)
			if metrics.RxPower != nil {
				history := entities.OntStatusHistory{
					CustomerID:    cust.ID,
					SerialNumber:  sn,
					IsOnline:      metrics.IsOnline,
					RxPower:       metrics.RxPower,
					TxPower:       metrics.TxPower,
					OpticalStatus: string(metrics.OpticalStatus),
					RecordedAt:    now,
				}
				_ = s.db.WithContext(ctx).Create(&history).Error
			}

			// Backfill IP / MAC reported by the ONT into network_devices when still empty.
			// Only fills blank columns so manual input from the installation form wins.
			s.syncDeviceNetworkInfo(ctx, cust.ID, metrics)
		} else {
			// Perangkat tidak ditemukan di GenieACS (offline / belum terkoneksi)
			var existing entities.OntStatus
			err := s.db.WithContext(ctx).Where("customer_id = ?", cust.ID).First(&existing).Error
			if err == nil {
				// Mark as offline and stale
				_ = s.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
					"is_online": false,
					"is_stale":  true,
					"updated_at": now,
				}).Error
			}
		}
	}

	return nil
}

// Client returns the underlying GenieACS client
func (s *OntPollerService) Client() *genieacs.Client {
	return s.client
}

// syncDeviceNetworkInfo fills blank ip_static / mac_address on the customer's
// network_devices row using values reported by the ONT through TR-069.
// Existing values (typed by technicians) are never overwritten.
func (s *OntPollerService) syncDeviceNetworkInfo(ctx context.Context, customerID string, metrics *genieacs.OntMetrics) {
	if metrics == nil || (metrics.IPAddress == "" && metrics.MacAddress == "") {
		return
	}

	var device entities.NetworkDevice
	err := s.db.WithContext(ctx).
		Where("customer_id = ?", customerID).
		Order("created_at ASC").
		First(&device).Error
	if err != nil {
		return
	}

	updates := map[string]interface{}{}

	if metrics.IPAddress != "" && (device.IPStatic == nil || *device.IPStatic == "") {
		updates["ip_static"] = metrics.IPAddress
	}
	if metrics.MacAddress != "" && (device.MacAddress == nil || *device.MacAddress == "") {
		updates["mac_address"] = metrics.MacAddress
	}

	if len(updates) == 0 {
		return
	}

	if err := s.db.WithContext(ctx).
		Model(&entities.NetworkDevice{}).
		Where("id = ?", device.ID).
		Updates(updates).Error; err != nil {
		log.Printf("[ONT Poller] Failed to sync network info for customer %s: %v", customerID, err)
	}
}

// Config returns the active model & threshold configuration
func (s *OntPollerService) Config() *genieacs.ONTModelsConfig {
	return s.config
}

