package ont

import (
	"context"
	"time"

	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
)

// MapOntStatusSummary represents the lightweight summary for map markers
type MapOntStatusSummary struct {
	CustomerID    string     `json:"customer_id"`
	CustomerName  string     `json:"customer_name"`
	Latitude      float64    `json:"latitude"`
	Longitude     float64    `json:"longitude"`
	HasLocation   bool       `json:"has_location"`
	SerialNumber  string     `json:"serial_number"`
	Manufacturer  string     `json:"manufacturer"`
	ModelName     string     `json:"model_name"`
	RouterSerial  string     `json:"router_serial"`
	MacAddress    string     `json:"mac_address"`
	IPStatic      string     `json:"ip_static"`
	SwitchID      string     `json:"switch_id,omitempty"`
	PortNumber    string     `json:"port_number,omitempty"`
	SoftwareVersion string   `json:"software_version,omitempty"`
	IsOnline      bool       `json:"is_online"`
	RxPower       *float64   `json:"rx_power"`
	TxPower       *float64   `json:"tx_power"`
	OpticalStatus string     `json:"optical_status"` // normal, warning, critical, unknown, no_ont
	SSID          string     `json:"ssid"`
	ClientCount   int        `json:"client_count"`
	LastInform    *time.Time `json:"last_inform"`
	IsStale       bool       `json:"is_stale"`
}

type Repository interface {
	GetMapStatuses(ctx context.Context) ([]MapOntStatusSummary, error)
	GetCustomerOntStatus(ctx context.Context, customerID string) (*entities.OntStatus, error)
	GetCustomerOntHistory(ctx context.Context, customerID string, limit int) ([]entities.OntStatusHistory, error)
	GetCustomerByID(ctx context.Context, customerID string) (*entities.Customer, error)
	SaveOntStatus(ctx context.Context, status *entities.OntStatus) error
	SaveOntHistory(ctx context.Context, history *entities.OntStatusHistory) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// GetMapStatuses fetches summary of all customers with their latest ONT status for map visualization
func (r *repository) GetMapStatuses(ctx context.Context) ([]MapOntStatusSummary, error) {
	var results []MapOntStatusSummary

	// Serial number resolution order:
	//   ont_statuses.serial_number (hasil poll GenieACS)
	//     -> asset_items.serial_number (unit fisik)
	//     -> network_devices.ont_serial_number
	//     -> customer.ont_serial_number
	query := `
		SELECT
			c.id AS customer_id,
			c.name AS customer_name,
			COALESCE(c.latitude, 0) AS latitude,
			COALESCE(c.longitude, 0) AS longitude,
			(CASE WHEN c.latitude IS NOT NULL AND c.longitude IS NOT NULL
			       AND c.latitude <> 0 AND c.longitude <> 0
			      THEN 1 ELSE 0 END) AS has_location,
			COALESCE(os.serial_number, ai.serial_number, nd.ont_serial_number, c.ont_serial_number, '') AS serial_number,
			COALESCE(os.manufacturer, '') AS manufacturer,
			COALESCE(os.model_name, '') AS model_name,
			COALESCE(ai.serial_number, nd.ont_serial_number, '') AS router_serial,
			COALESCE(NULLIF(nd.mac_address, ''), NULLIF(os.mac_address, ''), '') AS mac_address,
			COALESCE(NULLIF(nd.ip_static, ''), NULLIF(os.ip_address, ''), '') AS ip_static,
			COALESCE(nd.switch_id, '') AS switch_id,
			COALESCE(nd.port_number, '') AS port_number,
			COALESCE(os.software_version, '') AS software_version,
			COALESCE(os.is_online, FALSE) AS is_online,
			os.rx_power,
			os.tx_power,
			CASE
				WHEN COALESCE(os.serial_number, ai.serial_number, nd.ont_serial_number, c.ont_serial_number, '') = '' THEN 'no_ont'
				WHEN os.optical_status IS NOT NULL THEN os.optical_status
				ELSE 'unknown'
			END AS optical_status,
			COALESCE(os.ssid, '') AS ssid,
			COALESCE(os.client_count, 0) AS client_count,
			os.last_inform,
			COALESCE(os.is_stale, FALSE) AS is_stale
		FROM customer c
		LEFT JOIN ont_statuses os ON os.customer_id = c.id
		LEFT JOIN (
			SELECT nd1.customer_id, nd1.mac_address, nd1.ip_static, nd1.switch_id,
			       nd1.port_number, nd1.ont_serial_number, nd1.asset_item_id
			FROM network_devices nd1
			INNER JOIN (
				SELECT customer_id, MIN(id) AS first_id
				FROM network_devices
				GROUP BY customer_id
			) pick ON pick.first_id = nd1.id
		) nd ON nd.customer_id = c.id
		LEFT JOIN asset_items ai ON ai.id = nd.asset_item_id
		WHERE c.deleted_at IS NULL
	`

	if err := r.db.WithContext(ctx).Raw(query).Scan(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}

func (r *repository) GetCustomerOntStatus(ctx context.Context, customerID string) (*entities.OntStatus, error) {
	var status entities.OntStatus
	if err := r.db.WithContext(ctx).
		Where("customer_id = ?", customerID).
		First(&status).Error; err != nil {
		return nil, err
	}
	return &status, nil
}

func (r *repository) GetCustomerOntHistory(ctx context.Context, customerID string, limit int) ([]entities.OntStatusHistory, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var history []entities.OntStatusHistory
	if err := r.db.WithContext(ctx).
		Where("customer_id = ?", customerID).
		Order("recorded_at DESC").
		Limit(limit).
		Find(&history).Error; err != nil {
		return nil, err
	}

	// Reverse to return chronological order (oldest to newest) for charting
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	return history, nil
}

func (r *repository) GetCustomerByID(ctx context.Context, customerID string) (*entities.Customer, error) {
	var customer entities.Customer
	if err := r.db.WithContext(ctx).
		Where("id = ?", customerID).
		First(&customer).Error; err != nil {
		return nil, err
	}
	return &customer, nil
}

func (r *repository) SaveOntStatus(ctx context.Context, status *entities.OntStatus) error {
	return r.db.WithContext(ctx).Save(status).Error
}

func (r *repository) SaveOntHistory(ctx context.Context, history *entities.OntStatusHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

