package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeviceStatusLog stores historical status transition events from Uptime Kuma, The Dude, or MikroTik
type DeviceStatusLog struct {
	ID         string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Source     string    `json:"source" gorm:"type:varchar(50);not null;index:idx_dev_status_source"` // "kuma", "mikrotik", "dude"
	MonitorID  string    `json:"monitor_id" gorm:"type:varchar(100);index:idx_dev_status_mon_id"`
	DeviceName string    `json:"device_name" gorm:"type:varchar(255);not null"`
	Status     string    `json:"status" gorm:"type:varchar(50);not null"` // "up", "down", "degraded", "pending"
	PingMs     float64   `json:"ping_ms" gorm:"type:double"`
	Message    string    `json:"message" gorm:"type:text"`
	RawPayload string    `json:"raw_payload,omitempty" gorm:"type:longtext"`
	CreatedAt  time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP(3);index:idx_dev_status_created"`
}

func (d *DeviceStatusLog) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

func (DeviceStatusLog) TableName() string {
	return "device_status_logs"
}
