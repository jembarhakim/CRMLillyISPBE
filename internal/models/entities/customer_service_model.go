package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CustomerService model untuk layanan customer
type CustomerService struct {
	ID                     string                `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	CustomerID             string                `gorm:"column:customer_id;type:varchar(191);index:idx_customer_services_customer_id" json:"customer_id"`
	CustomerInstallationID *string               `gorm:"column:customer_installation_id;type:varchar(191);index:idx_customer_services_customer_installation_id" json:"customer_installation_id,omitempty"`
	DeviceID               *string               `gorm:"column:device_id;type:varchar(191);index:idx_customer_services_device_id" json:"device_id,omitempty"`
	CableType              *string               `gorm:"column:cable_type;type:varchar(100);collate:utf8mb4_unicode_ci" json:"cable_type,omitempty"`
	CableLength            *float64              `gorm:"column:cable_length;type:decimal(10,2)" json:"cable_length,omitempty"`
	EndPortType            *string               `gorm:"column:end_port_type;type:varchar(50)" json:"end_port_type,omitempty"`
	UserLogin              *string               `gorm:"column:user_login;type:varchar(191)" json:"user_login,omitempty"`
	Password               *string               `gorm:"column:password;type:varchar(191)" json:"password,omitempty"`
	UserStatus             string                `gorm:"column:user_status;type:enum('Active','Inactive','Suspended','Pending');default:'Active'" json:"user_status,omitempty"`
	InstallationNotes      *string               `gorm:"column:installation_notes;type:text" json:"installation_notes,omitempty"`
	CreatedAt              *time.Time            `gorm:"column:created_at;type:timestamp;default:current_timestamp" json:"created_at,omitempty"`
	UpdatedAt              *time.Time            `gorm:"column:updated_at;type:timestamp;default:current_timestamp on update current_timestamp" json:"updated_at,omitempty"`
	Customer               *Customer             `gorm:"foreignKey:CustomerID;references:ID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"customer,omitempty"`
	CustomerInstallation   *CustomerInstallation `gorm:"foreignKey:CustomerInstallationID;references:ID;constraint:OnDelete:SET NULL,OnUpdate:CASCADE" json:"customer_installation,omitempty"`
	NetworkDevice          *NetworkDevice        `gorm:"foreignKey:DeviceID;references:ID;constraint:OnDelete:SET NULL,OnUpdate:CASCADE" json:"network_device,omitempty"`
}

func (c *CustomerService) TableName() string {
	return "customer_services"
}

func (c *CustomerService) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	
	return nil
}
