package customerinstallation

// TechnicianAssignment represents a single technician assignment with role
type TechnicianAssignment struct {
	TechnicianID string `json:"technician_id" validate:"required"`
	Role         string `json:"role" validate:"required,oneof=senior junior helper"`
	IsPrimary    bool   `json:"is_primary"`
	Notes        string `json:"notes"`
}

type CreateAdminCustomerInstallationRequest struct {
	CustomerId    string                 `json:"customer_id" validate:"required"`
	TechnicianId  string                 `json:"technician_id"` // Legacy: for backward compatibility
	Technicians   []TechnicianAssignment `json:"technicians"`   // New: multiple technicians with roles
	Status        string                 `json:"status"`
	Notes         string                 `json:"notes"`
	MACAddress    string                 `json:"mac_address"`
	MaxLimit      string                 `json:"max_limit"` // e.g., "10M/10M"
	DocumentType  string                 `json:"document_type" validate:"omitempty,oneof=KTP SIM Paspor"`
	DocumentPhoto string                 `json:"document_photo"`
	ImageIds      []string               `json:"image_ids" validate:"required"`
	OnAirDate     string                 `json:"on_air_date"`
	AutoProvision bool                   `json:"auto_provision"` // Whether to auto-provision on MikroTik
	DryRun        bool                   `json:"dry_run"`        // Test provisioning without executing
}

type IdAdminCustomerInstallationRequest struct {
	Id string `json:"id"`
}

type UpdateAdminCustomerInstallationRequest struct {
	IdAdminCustomerInstallationRequest
	CreateAdminCustomerInstallationRequest
}

// CreateReportInstallationRequest for simpler single-form submission
type CreateReportInstallationRequest struct {
	CustomerID                    string                 `form:"customer_id" json:"customer_id" validate:"required"`
	TechnicianID                  string                 `form:"technician_id" json:"technician_id"` // Legacy
	Technicians                   []TechnicianAssignment `json:"technicians"`                        // Multiple technicians
	AssetsID                      string                 `form:"assets_id" json:"assets_id"`
	Status                        string                 `form:"status" json:"status"`
	Notes                         string                 `form:"notes" json:"notes"`
	MaxLimit                      string                 `form:"max_limit" json:"max_limit"`
	AutoProvision                 bool                   `form:"auto_provision" json:"auto_provision"`
	DryRun                        bool                   `form:"dry_run" json:"dry_run"`
	DocumentType                  string                 `form:"document_type" json:"document_type"`
	DocumentPhoto                 string                 `form:"document_photo" json:"document_photo"` // File path after upload
	InstallationType              string                 `form:"installation_type" json:"installation_type"`
	OnAirDate                     string                 `form:"on_air_date" json:"on_air_date"`
	TrialEndDate                  string                 `form:"trial_end_date" json:"trial_end_date"`
	ServiceReadyDate              string                 `form:"service_ready_date" json:"service_ready_date"`
	InstallationCompletedAt       string                 `form:"installation_completed_at" json:"installation_completed_at"`
	IsTerminal                    string                 `form:"is_terminal" json:"is_terminal"` // 'yes' or 'no'
	TerminalCustomerInstallationId string                `form:"terminal_customer_installation_id" json:"terminal_customer_installation_id"` // Installation ID of the terminal installation
	Latitude                      *float64               `form:"latitude" json:"latitude,omitempty"`
	Longitude                     *float64               `form:"longitude" json:"longitude,omitempty"`
	SwitchID                      string                 `form:"switch_id" json:"switch_id"`
	PortNumber                    string                 `form:"port_number" json:"port_number"`
	RemotePort                    string                 `form:"remote_port" json:"remote_port"`
	EthPort                       string                 `form:"eth_port" json:"eth_port"`
	MacAddress                    string                 `form:"mac_address" json:"mac_address"`     // Device MAC
	AssetItemID                   string                 `form:"asset_item_id" json:"asset_item_id"` // Specific asset item ID
	IPStatic                      string                 `form:"ip_static" json:"ip_static"`
	KepemilikanPerangkat          string                 `form:"kepemilikan_perangkat" json:"kepemilikan_perangkat"`
	ProductID                     string                 `form:"product_id" json:"product_id"`
	CableType                     string                 `form:"cable_type" json:"cable_type"`
	CableLength                   float64                `form:"cable_length" json:"cable_length"` // Changed to float64 to match entity
	EndPortType                   string                 `form:"end_port_type" json:"end_port_type"`
	UserLogin                     string                 `form:"user_login" json:"user_login"`
	Password                      string                 `form:"password" json:"password"`
	UserStatus                    string                 `form:"user_status" json:"user_status"`
	InstallationNotes             string                 `form:"installation_notes" json:"installation_notes"`
	CustomerCompanyID             string                 `form:"customer_company_id" json:"customer_company_id"`
	CustomerSalesRepresentativeID string                 `form:"customer_sales_representative_id" json:"customer_sales_representative_id"`
	ImageIds                      []string               `json:"image_ids"`

	// Technician Photo Documentation
	TechnicianPhotos      []string `form:"technician_photos" json:"technician_photos"`
	TechnicianPhotosNotes string   `form:"technician_photos_notes" json:"technician_photos_notes"`

	// Auto-populated fields
	DeviceID              string `form:"device_id" json:"device_id"`
	ServiceActivationDate string `form:"service_activation_date" json:"service_activation_date"`
	InstallationTeamPhone string `form:"installation_team_phone" json:"installation_team_phone"`
	IsTrial               bool   `form:"is_trial" json:"is_trial"`
}
