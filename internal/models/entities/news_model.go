package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type News struct {
	ID          string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Title       string     `json:"title" gorm:"type:varchar(255);not null"`
	Summary     string     `json:"summary" gorm:"type:text;not null"`
	Content     string     `json:"content" gorm:"type:longtext;not null"`
	ImageURL    string     `json:"image_url" gorm:"type:text"`
	IsPublished bool       `json:"is_published" gorm:"not null;default:false;index"`
	PublishedAt *time.Time `json:"published_at" gorm:"index"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (News) TableName() string {
	return "news"
}

func (n *News) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	return nil
}
