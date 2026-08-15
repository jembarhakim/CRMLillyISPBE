package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CustomerInstallation model untuk instalasi customer
type CustomerInstallation struct {
	ID                      string     `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	CustomerID              *string    `gorm:"column:customer_id;type:varchar(191);index:idx_customer_installations_customer_id" json:"customer_id,omitempty"`
	TechnicianID            *string    `gorm:"column:technician_id;type:varchar(191);index:idx_customer_installations_technician_id" json:"technician_id,omitempty"` // Legacy: for backward compatibility
	Status                  string     `gorm:"column:status;type:varchar(50);default:'pending'" json:"status,omitempty"`
	Notes                   string     `gorm:"column:notes;type:text" json:"notes,omitempty"`
	IPAddress               *string    `gorm:"column:ip_address;type:varchar(15)" json:"ip_address,omitempty"`
	ProvisioningStatus      *string    `gorm:"column:provisioning_status;type:enum('pending','queued','provisioned','failed','manual');default:'pending';index:idx_ci_provisioning_status" json:"provisioning_status,omitempty"`
	ProvisioningCompletedAt *time.Time `gorm:"column:provisioning_completed_at;type:timestamp" json:"provisioning_completed_at,omitempty"`
	CodeName                *string    `gorm:"column:code_name;type:varchar(255);index:idx_ci_code_name" json:"code_name,omitempty"`
	DocumentType            *string    `gorm:"column:document_type;type:enum('KTP','SIM','Paspor')" json:"document_type,omitempty"`
	DocumentPhoto           *string    `gorm:"column:document_photo;type:varchar(255)" json:"document_photo,omitempty"`
	InstallationType        string     `gorm:"column:installation_type;type:enum('new_installation','maintenance','upgrade','downgrade');default:'new_installation'" json:"installation_type,omitempty"`

	InstallationCompletedAt *time.Time `gorm:"column:installation_completed_at;type:datetime(3)" json:"installation_completed_at,omitempty"`
	TrialEndDate            *time.Time `gorm:"column:trial_end_date;type:date" json:"trial_end_date,omitempty"`
	ServiceReadyDate        *time.Time `gorm:"column:service_ready_date;type:date" json:"service_ready_date,omitempty"`
	OnAirDate               *time.Time `gorm:"column:on_air_date;type:date" json:"on_air_date,omitempty"`

	// Installation location coordinates
	Latitude  *float64 `gorm:"column:latitude;type:double" json:"latitude,omitempty"`
	Longitude *float64 `gorm:"column:longitude;type:double" json:"longitude,omitempty"`

	// Terminal installation fields
	IsTerminal                     *string `gorm:"column:is_terminal;type:enum('yes','no');default:'no'" json:"is_terminal,omitempty"`
	TerminalCustomerInstallationID *string `gorm:"column:terminal_customer_installation_id;type:varchar(191);index:idx_customer_installations_terminal_installation_id" json:"terminal_customer_installation_id,omitempty"`

	// Technician Photo Documentation is now handled via Images relationship
	// Use installation.Images to access technician photos with archive_installation_id

	CreatedAt time.Time `gorm:"column:createdAt;default:CURRENT_TIMESTAMP(3)" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updatedAt;default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)" json:"updatedAt"`

	// Relations
	Customer                     *Customer                      `gorm:"foreignKey:CustomerID;references:id;constraint:OnDelete:SET NULL,OnUpdate:CASCADE" json:"customer,omitempty"`
	Technician                   *User                          `gorm:"foreignKey:TechnicianID;references:id;constraint:OnDelete:SET NULL,OnUpdate:CASCADE" json:"technician,omitempty"` // Legacy: for backward compatibility
	TerminalCustomerInstallation *CustomerInstallation          `gorm:"foreignKey:TerminalCustomerInstallationID;references:ID;constraint:OnDelete:SET NULL,OnUpdate:CASCADE" json:"terminal_customer_installation,omitempty"`
	Images                       []Image                        `gorm:"foreignKey:ArchiveInstallationId;references:ID" json:"images"`
	AssetTransactions            []AssetTransaction             `gorm:"foreignKey:CustomerInstallationID;references:ID" json:"asset_transactions,omitempty"`
	NetworkDevices               []NetworkDevice                `gorm:"foreignKey:CustomerInstallationID;references:ID" json:"network_devices,omitempty"`
	CustomerServices             []CustomerService              `gorm:"foreignKey:CustomerInstallationID;references:ID" json:"customer_services,omitempty"`
	InstallationTechnicians      []InstallationReportTechnician `gorm:"foreignKey:CustomerInstallationID;references:ID" json:"installation_technicians,omitempty"`
	InstallationProvisioningLogs []InstallationProvisioningLog  `gorm:"foreignKey:CustomerInstallationID;references:ID" json:"provisioning_logs,omitempty"`

	// Soft delete marker
	DeletedAt *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
}

func (c *CustomerInstallation) TableName() string {
	return "customer_installations"
}

// Validation Hook: Run this before saving
func (ci *CustomerInstallation) BeforeSave(tx *gorm.DB) (err error) {
	if ci.TerminalCustomerInstallationID != nil {
		var parent CustomerInstallation
		// Check if parent exists AND is actually a terminal
		if err := tx.First(&parent, "id = ?", *ci.TerminalCustomerInstallationID).Error; err != nil {
			return fmt.Errorf("terminal parent not found")
		}

		if parent.IsTerminal == nil || *parent.IsTerminal != "yes" {
			return fmt.Errorf("selected parent is not marked as a Terminal")
		}

		// PREVENT LOOPS: Make sure I am not my own father
		if ci.ID == *ci.TerminalCustomerInstallationID {
			return fmt.Errorf("an installation cannot be its own terminal")
		}
	}
	return nil
}

func (c *CustomerInstallation) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}
