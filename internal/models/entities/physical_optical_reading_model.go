package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PhysicalOpticalReading struct {
	ID             string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	CustomerID     string    `gorm:"type:varchar(191);not null;index:idx_physical_optical_customer" json:"customer_id"`
	SourceEvent    string    `gorm:"type:varchar(32);not null;index:idx_physical_optical_source" json:"source_event"`
	TechnicianID   string    `gorm:"type:varchar(191);not null" json:"technician_id"`
	TechnicianName string    `gorm:"type:varchar(191)" json:"technician_name,omitempty"`
	RxPower        float64   `gorm:"type:decimal(6,2);not null" json:"rx_power"`
	OpticalStatus  string    `gorm:"type:varchar(20);not null;index:idx_physical_optical_status" json:"optical_status"`
	Notes          string    `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	Customer *Customer `gorm:"foreignKey:CustomerID;references:ID" json:"-"`
}

func (PhysicalOpticalReading) TableName() string {
	return "physical_optical_readings"
}

func (r *PhysicalOpticalReading) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	r.OpticalStatus = ClassifyOpticalPower(r.RxPower)
	return nil
}

func ClassifyOpticalPower(rxPower float64) string {
	if rxPower > -20.00 {
		return "normal"
	}
	if rxPower >= -24.00 {
		return "warning"
	}
	return "critical"
}

type LinkFlappingStatus struct {
	CustomerID      string     `gorm:"primaryKey;type:varchar(191)" json:"customer_id"`
	Status          string     `gorm:"type:varchar(32);not null;default:'stable';index" json:"status"`
	DisconnectCount int        `gorm:"type:int;not null;default:0" json:"disconnect_count"`
	WindowStart     time.Time  `gorm:"column:window_start" json:"window_start,omitempty"`
	LastDetectedAt  *time.Time `gorm:"column:last_detected_at" json:"last_detected_at,omitempty"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	Customer *Customer `gorm:"foreignKey:CustomerID;references:ID" json:"-"`
}

func (LinkFlappingStatus) TableName() string {
	return "link_flapping_status"
}
