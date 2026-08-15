package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Cable model untuk kabel
type Cable struct {
	ID                     string                `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	Name                   string                `gorm:"column:name;type:varchar(255)" json:"name"`
	Type                   *string               `gorm:"column:type;type:varchar(100)" json:"type,omitempty"`
	Length                 *float64              `gorm:"column:length;type:decimal(10,2)" json:"length,omitempty"`
	Status                 string                `gorm:"column:status;type:enum('available','in_use','damaged','retired');default:'available'" json:"status,omitempty"`
	CustomerInstallationID *string               `gorm:"column:customer_installation_id;type:varchar(191);index:idx_cable_customer_installation_id" json:"customer_installation_id,omitempty"`
	CreatedAt              *time.Time            `gorm:"column:created_at;type:timestamp;default:current_timestamp" json:"created_at,omitempty"`
	UpdatedAt              *time.Time            `gorm:"column:updated_at;type:timestamp;default:current_timestamp on update current_timestamp" json:"updated_at,omitempty"`
	CustomerInstallation   *CustomerInstallation `gorm:"foreignKey:CustomerInstallationID;references:ID;constraint:OnDelete:SET NULL,OnUpdate:CASCADE" json:"customer_installation,omitempty"`
}

func (c *Cable) TableName() string {
	return "cable"
}

func (c *Cable) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}
