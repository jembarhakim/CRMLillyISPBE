package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NetworkRouterCredential struct {
	ID                string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name              string    `json:"name" gorm:"type:varchar(191);not null"`
	Host              string    `json:"host" gorm:"type:varchar(191);not null"`
	Port              int       `json:"port" gorm:"type:int;not null"`
	Username          string    `json:"username" gorm:"type:varchar(191);not null"`
	EncryptedPassword string    `json:"-" gorm:"type:text;not null"`
	UseTLS            bool      `json:"use_tls" gorm:"not null;default:false"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (c *NetworkRouterCredential) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

func (NetworkRouterCredential) TableName() string {
	return "network_router_credentials"
}
