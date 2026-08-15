package entities

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StringArray is a custom type for JSON array of strings
type StringArray []string

// Value implements the driver.Valuer interface
func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "[]", nil
	}
	return json.Marshal(a)
}

// Scan implements the sql.Scanner interface
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = []string{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, a)
}

// BroadcastHistory model
type BroadcastHistory struct {
	ID              string      `json:"id" gorm:"primaryKey"`
	Message         string      `json:"message" gorm:"type:text"`
	TargetGroup     string      `json:"target_group" gorm:"type:varchar(50)"`
	RecipientCount  int         `json:"recipient_count" gorm:"default:0"`
	RecipientPhones StringArray `json:"recipient_phones" gorm:"type:json"`
	Status          string      `json:"status" gorm:"type:enum('sent','failed','pending');default:'pending'"`
	TemplateKey     string      `json:"template_key" gorm:"type:varchar(50)"`
	SentAt          time.Time   `json:"sent_at" gorm:"type:timestamp;default:current_timestamp"`
	CreatedBy       string      `json:"created_by" gorm:"type:varchar(191);index"`
	CreatedAt       time.Time   `json:"created_at" gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt       *time.Time  `json:"updated_at" gorm:"column:updated_at"`

	// Relations
	User User `json:"user" gorm:"foreignKey:CreatedBy;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

func (b *BroadcastHistory) TableName() string {
	return "broadcast_history"
}

// func (b *BroadcastHistory) BeforeSave(tx *gorm.DB) error {
// 	if b.RecipientPhones == nil {
// 		b.RecipientPhones = []string{}
// 	}
// 	return nil
// }

func (b *BroadcastHistory) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}
