package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InstallationReportTechnician represents the pivot table for many-to-many relationship
// between customer installations and technicians with role assignment
type InstallationReportTechnician struct {
	ID                     string    `gorm:"column:id;type:varchar(36);primaryKey" json:"id"`
	CustomerInstallationID string    `gorm:"column:customer_installation_id;type:varchar(255);not null;index:idx_irt_customer_installation_id" json:"customer_installation_id"`
	TechnicianID           string    `gorm:"column:technician_id;type:varchar(255);not null;index:idx_irt_technician_id" json:"technician_id"`
	Role                   string    `gorm:"column:role;type:enum('senior','junior','helper');not null;default:'junior';index:idx_irt_role" json:"role"`
	IsPrimary              bool      `gorm:"column:is_primary;type:boolean;default:false;index:idx_irt_is_primary" json:"is_primary"`
	Notes                  string    `gorm:"column:notes;type:text" json:"notes,omitempty"`
	CreatedAt              time.Time `gorm:"column:createdAt;default:CURRENT_TIMESTAMP(3)" json:"createdAt"`
	UpdatedAt              time.Time `gorm:"column:updatedAt;default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)" json:"updatedAt"`

	// Relations
	CustomerInstallation *CustomerInstallation `gorm:"foreignKey:CustomerInstallationID;references:ID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"customer_installation,omitempty"`
	Technician           *User                 `gorm:"foreignKey:TechnicianID;references:ID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"technician,omitempty"`
}

func (irt *InstallationReportTechnician) TableName() string {
	return "installation_report_technicians"
}

func (irt *InstallationReportTechnician) BeforeCreate(tx *gorm.DB) error {
	if irt.ID == "" {
		irt.ID = uuid.New().String()
	}
	return nil
}

// TechnicianRole constants for validation
const (
	TechnicianRoleSenior = "senior"
	TechnicianRoleJunior = "junior"
	TechnicianRoleHelper = "helper"
)

// IsValidRole checks if the role is valid
func IsValidTechnicianRole(role string) bool {
	return role == TechnicianRoleSenior || role == TechnicianRoleJunior || role == TechnicianRoleHelper
}

