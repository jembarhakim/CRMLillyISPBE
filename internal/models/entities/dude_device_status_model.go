package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DudeDeviceStatus struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name      string    `json:"device_name" gorm:"type:varchar(255);not null"`
	IPAddress string    `json:"ip_address" gorm:"type:varchar(45);not null;uniqueIndex"`
	Status    string    `json:"status" gorm:"type:varchar(100);not null"`
	UserName  string    `json:"user_name" gorm:"type:varchar(255)"`
	UserPhone string    `json:"user_phone" gorm:"type:varchar(100)"`
	Address   string    `json:"address" gorm:"type:text"`
	Notes     string    `json:"notes" gorm:"type:text"`
	Latitude  *float64  `json:"latitude" gorm:"type:double"`
	Longitude *float64  `json:"longitude" gorm:"type:double"`
	LastSeen  time.Time `json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (d *DudeDeviceStatus) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

func (DudeDeviceStatus) TableName() string {
	return "dude_device_statuses"
}
