package entities

import "time"

// TicketClassification represents the classification lookup table
type TicketClassification struct {
	ID            string    `gorm:"primaryKey;type:varchar(20)" json:"id"`
	Name          string    `gorm:"type:varchar(50);not null" json:"name"`
	Description   *string   `gorm:"type:text" json:"description,omitempty"`
	ShowNOCAction bool      `gorm:"type:tinyint(1);default:1" json:"show_noc_action"`
	CreatedAt     time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt     time.Time `gorm:"default:CURRENT_TIMESTAMP(3)" json:"updated_at"`
}

func (TicketClassification) TableName() string {
	return "ticket_classification"
}

// Available classification IDs
const (
	ClassificationGangguan  = "gangguan"
	ClassificationPSB       = "psb"
	ClassificationLainnya   = "lainnya"
	ClassificationDismantle = "dismantle"
)
