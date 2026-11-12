package customerinstallation

import (
	"skripsi-be/internal/models/entities"
	"time"
)

// UpdateCompleteInstallationReportRequest - Request untuk mengupdate laporan instalasi lengkap
type UpdateCompleteInstallationReportRequest struct {
	// Basic Installation Data
	CustomerId              string `json:"customer_id" validate:"required"`
	TechnicianId            string `json:"technician_id" validate:"required"`
	Status                  string `json:"status"`
	Notes                   string `json:"notes"`
	DocumentType            string `json:"document_type" validate:"omitempty,oneof=KTP SIM Paspor"`
	DocumentPhoto           string `json:"document_photo"`
	InstallationType        string `json:"installation_type" validate:"omitempty,oneof=new_installation maintenance upgrade downgrade"`
	OnAirDate               string `json:"on_air_date"`
	TrialEndDate            string `json:"trial_end_date"`
	ServiceReadyDate        string `json:"service_ready_date"`
	InstallationCompletedAt string `json:"installation_completed_at"`

	// Terminal installation fields
	IsTerminal                     string `json:"is_terminal"`                       // 'yes' or 'no'
	TerminalCustomerInstallationId string `json:"terminal_customer_installation_id"` // Installation ID of the terminal installation

	// Installation location
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`

	// Network Devices (multiple devices support)
	NetworkDevices []NetworkDeviceRequest `json:"network_devices"`

	// Customer Services (multiple services support)
	CustomerServices []CustomerServiceRequest `json:"customer_services"`

	// Images
	ImageIds []string `json:"image_ids"`

	// Technician Photo Documentation
	TechnicianPhotos      []string `json:"technician_photos"`
	TechnicianPhotosNotes string   `json:"technician_photos_notes"`
}

// CreateCompleteInstallationReportRequest - Request untuk membuat laporan instalasi lengkap
type CreateCompleteInstallationReportRequest struct {
	// Basic Installation Data
	CustomerId              string `json:"customer_id" validate:"required"`
	TechnicianId            string `json:"technician_id" validate:"required"`
	Status                  string `json:"status"`
	Notes                   string `json:"notes"`
	DocumentType            string `json:"document_type" validate:"omitempty,oneof=KTP SIM Paspor"`
	DocumentPhoto           string `json:"document_photo"`
	InstallationType        string `json:"installation_type" validate:"omitempty,oneof=new_installation maintenance upgrade downgrade"`
	OnAirDate               string `json:"on_air_date"`
	TrialEndDate            string `json:"trial_end_date"`
	ServiceReadyDate        string `json:"service_ready_date"`
	InstallationCompletedAt string `json:"installation_completed_at"`

	// Terminal installation fields
	IsTerminal                     string `json:"is_terminal"`                       // 'yes' or 'no'
	TerminalCustomerInstallationId string `json:"terminal_customer_installation_id"` // Installation ID of the terminal installation

	// Installation location
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`

	// Asset Tracking
	AssetTransactions []AssetTransactionRequest `json:"asset_transactions"`

	// Network Devices
	NetworkDevices []NetworkDeviceRequest `json:"network_devices"`

	// Customer Services
	CustomerServices []CustomerServiceRequest `json:"customer_services"`

	// Cables
	Cables []CableRequest `json:"cables"`

	// Images
	ImageIds []string `json:"image_ids" validate:"required"`
}

// AssetTransactionRequest - Request untuk transaksi aset
type AssetTransactionRequest struct {
	AssetId         string `json:"asset_id" validate:"required"`
	TransactionType string `json:"transaction_type" validate:"required,oneof=out in"`
	Quantity        int    `json:"quantity" validate:"required,min=1"`
	Notes           string `json:"notes"`
	TransactionDate string `json:"transaction_date"`
}

// NetworkDeviceRequest - Request untuk network device
type NetworkDeviceRequest struct {
	AssetsID             string `json:"assets_id" validate:"required"`
	AssetItemID          string `json:"asset_item_id"` // Specific asset item ID for MAC address tracking
	SwitchID             string `json:"switch_id"`
	PortNumber           string `json:"port_number"`
	RemotePort           string `json:"remote_port"`
	EthPort              string `json:"eth_port"`
	MacAddress           string `json:"mac_address"`
	IPStatic             string `json:"ip_static"`
	KepemilikanPerangkat string `json:"kepemilikan_perangkat" validate:"omitempty,oneof=owned leased customer"`
	ProductID            string `json:"product_id"`
	RouterBrand          string `json:"router_brand"`
	RouterType           string `json:"router_type"`
}

// CustomerServiceRequest - Request untuk customer service
type CustomerServiceRequest struct {
	DeviceID              string  `json:"device_id"`
	CableType             string  `json:"cable_type"`
	CableLength           float64 `json:"cable_length"`
	EndPortType           string  `json:"end_port_type"`
	UserLogin             string  `json:"user_login"`
	Password              string  `json:"password"`
	UserStatus            string  `json:"user_status" validate:"omitempty,oneof=Active Inactive Suspended Pending"`
	InstallationNotes     string  `json:"installation_notes"`
	ServiceActivationDate string  `json:"service_activation_date"`
}

// CableRequest - Request untuk kabel
type CableRequest struct {
	Name   string  `json:"name" validate:"required"`
	Type   string  `json:"type"`
	Length float64 `json:"length"`
	Status string  `json:"status" validate:"omitempty,oneof=available in_use damaged retired"`
}

// InstallationSummaryResponse - Response untuk summary instalasi per customer
type InstallationSummaryResponse struct {
	CustomerId              string     `json:"customer_id"`
	CustomerName            string     `json:"customer_name"`
	CustomerAddress         string     `json:"customer_address"`
	CustomerPhone           string     `json:"customer_phone"`
	TglPermintaanPsb        *time.Time `json:"tgl_permintaan_psb"`
	TotalInstallations      int64      `json:"total_installations"`
	CompletedInstallations  int64      `json:"completed_installations"`
	PendingInstallations    int64      `json:"pending_installations"`
	InProgressInstallations int64      `json:"in_progress_installations"`
	LatestOnAirDate         *time.Time `json:"latest_on_air_date"`
	LatestCompletionDate    *time.Time `json:"latest_completion_date"`
	AvgDurasiPsb            *float64   `json:"avg_durasi_psb"`
	TepatWaktuCount         int64      `json:"tepat_waktu_count"`
	TerlambatCount          int64      `json:"terlambat_count"`
}

// InstallationAssetReportResponse - Response untuk laporan aset instalasi
type InstallationAssetReportResponse struct {
	InstallationId          string     `json:"installation_id"`
	CustomerName            string     `json:"customer_name"`
	InstallationType        string     `json:"installation_type"`
	InstallationStatus      string     `json:"installation_status"`
	OnAirDate               *time.Time `json:"on_air_date"`
	InstallationCompletedAt *time.Time `json:"installation_completed_at"`
	TotalQuantityOut        int64      `json:"total_quantity_out"`
	TotalQuantityIn         int64      `json:"total_quantity_in"`
	AssetsOutDetails        string     `json:"assets_out_details"`
	AssetsInDetails         string     `json:"assets_in_details"`
}

// InstallationTechnicianReportResponse - Response untuk laporan teknisi
type InstallationTechnicianReportResponse struct {
	TechnicianId            string     `json:"technician_id"`
	TechnicianName          string     `json:"technician_name"`
	TechnicianPhone         string     `json:"technician_phone"`
	TotalInstallations      int64      `json:"total_installations"`
	CompletedInstallations  int64      `json:"completed_installations"`
	PendingInstallations    int64      `json:"pending_installations"`
	InProgressInstallations int64      `json:"in_progress_installations"`
	AvgCompletionDays       float64    `json:"avg_completion_days"`
	LatestCompletionDate    *time.Time `json:"latest_completion_date"`
}

// InstallationReportCompleteResponse - Response untuk laporan instalasi lengkap dari database view
type InstallationReportCompleteResponse struct {
	InstallationId          string     `json:"installation_id"`
	CustomerId              string     `json:"customer_id"`
	CustomerName            string     `json:"customer_name"`
	CustomerAddress         string     `json:"customer_address"`
	CustomerPhone           string     `json:"customer_phone"`
	TglPermintaanPsb        *time.Time `json:"tgl_permintaan_psb"`
	TechnicianId            string     `json:"technician_id"`
	TechnicianName          string     `json:"technician_name"`
	TechnicianPhone         string     `json:"technician_phone"`
	InstallationStatus      string     `json:"installation_status"`
	InstallationType        string     `json:"installation_type"`
	InstallationNotes       string     `json:"installation_notes"`
	OnAirDate               *time.Time `json:"on_air_date"`
	TrialEndDate            *time.Time `json:"trial_end_date"`
	ServiceReadyDate        *time.Time `json:"service_ready_date"`
	InstallationCompletedAt *time.Time `json:"installation_completed_at"`
	DurasiPsb               *int       `json:"durasi_psb"`
	StatusPsb               string     `json:"status_psb"`
	DocumentType            string     `json:"document_type"`
	DocumentPhoto           string     `json:"document_photo"`
	NetworkDeviceId         string     `json:"network_device_id"`
	SwitchId                string     `json:"switch_id"`
	PortNumber              string     `json:"port_number"`
	RemotePort              string     `json:"remote_port"`
	EthPort                 string     `json:"eth_port"`
	MacAddress              string     `json:"mac_address"`
	IPStatic                string     `json:"ip_static"`
	KepemilikanPerangkat    string     `json:"kepemilikan_perangkat"`
	RouterBrand             string     `json:"router_brand"`
	RouterType              string     `json:"router_type"`
	RouterModel             string     `json:"router_model"`
	RouterSerial            string     `json:"router_serial"`
	CustomerServiceId       string     `json:"customer_service_id"`
	UserLogin               string     `json:"user_login"`
	Password                string     `json:"password"`
	UserStatus              string     `json:"user_status"`
	ServiceNotes            string     `json:"service_notes"`
	CableType               string     `json:"cable_type"`
	CableLength             float64    `json:"cable_length"`
	EndPortType             string     `json:"end_port_type"`
	InstallationCreatedAt   time.Time  `json:"installation_created_at"`
	InstallationUpdatedAt   time.Time  `json:"installation_updated_at"`

	// Technician Photo Documentation (now handled via Images relationship)
	// Use the Images relationship to access technician photos with archive_installation_id

	// Product information
	ProductId                string  `json:"product_id"`
	ProductName              string  `json:"product_name"`
	ProductDescription       string  `json:"product_description"`
	ProductPrice             float64 `json:"product_price"`
	ProductDownloadSpeedMbps *int    `json:"download_speed_mbps"`
	ProductUploadSpeedMbps   *int    `json:"upload_speed_mbps"`
}

// Installation Technician Team Response
type InstallationTechnicianTeamResponse struct {
	ID                     string    `json:"id"`
	CustomerInstallationID string    `json:"customer_installation_id"`
	TechnicianID           string    `json:"technician_id"`
	TechnicianName         string    `json:"technician_name"`
	TechnicianPhone        string    `json:"technician_phone"`
	TechnicianEmail        string    `json:"technician_email"`
	Role                   string    `json:"role"`
	IsPrimary              bool      `json:"is_primary"`
	Notes                  string    `json:"notes"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// CompleteInstallationReportWithTechnicianPhotosResponse - Response with computed technician photos
type CompleteInstallationReportWithTechnicianPhotosResponse struct {
	entities.CustomerInstallation

	// Computed technician photo fields (now handled via Images relationship)
	// These fields are computed from the Images relationship in the repository

	// Computed fields for frontend compatibility (from relationships)
	InstallationId           string    `json:"installation_id"`
	CustomerName             string    `json:"customer_name"`
	CustomerAddress          string    `json:"customer_address"`
	CustomerPhone            string    `json:"customer_phone"`
	TglPermintaanPsb         string    `json:"tgl_permintaan_psb"`
	TechnicianName           string    `json:"technician_name"`
	TechnicianPhone          string    `json:"technician_phone"`
	InstallationStatus       string    `json:"installation_status"`
	InstallationNotes        string    `json:"installation_notes"`
	DurasiPsb                *int      `json:"durasi_psb"`
	StatusPsb                string    `json:"status_psb"`
	NetworkDeviceId          string    `json:"network_device_id"`
	SwitchId                 string    `json:"switch_id"`
	PortNumber               string    `json:"port_number"`
	RemotePort               string    `json:"remote_port"`
	EthPort                  string    `json:"eth_port"`
	MacAddress               string    `json:"mac_address"`
	IPStatic                 string    `json:"ip_static"`
	KepemilikanPerangkat     string    `json:"kepemilikan_perangkat"`
	RouterBrand              string    `json:"router_brand"`
	RouterType               string    `json:"router_type"`
	RouterModel              string    `json:"router_model"`
	RouterSerial             string    `json:"router_serial"`
	CustomerServiceId        string    `json:"customer_service_id"`
	UserLogin                string    `json:"user_login"`
	Password                 string    `json:"password"`
	UserStatus               string    `json:"user_status"`
	ServiceNotes             string    `json:"service_notes"`
	InstallationTeamName     string    `json:"installation_team_name"`
	InstallationTeamPhone    string    `json:"installation_team_phone"`
	CableType                string    `json:"cable_type"`
	CableLength              float64   `json:"cable_length"`
	EndPortType              string    `json:"end_port_type"`
	InstallationCreatedAt    time.Time `json:"installation_created_at"`
	InstallationUpdatedAt    time.Time `json:"installation_updated_at"`
	ProductId                string    `json:"product_id"`
	ProductName              string    `json:"product_name"`
	ProductDescription       string    `json:"product_description"`
	ProductPrice             float64   `json:"product_price"`
	ProductDownloadSpeedMbps *int      `json:"download_speed_mbps"`
	ProductUploadSpeedMbps   *int      `json:"upload_speed_mbps"`
}
