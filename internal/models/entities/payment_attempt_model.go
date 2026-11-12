package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentAttempt struct {
	ID         string    `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	InvoiceID  string    `gorm:"column:invoice_id;type:varchar(191);not null;index:idx_payment_attempt_invoice_status,priority:1" json:"invoice_id"`
	OrderID    string    `gorm:"column:order_id;type:varchar(50);not null;uniqueIndex" json:"order_id"`
	Bank       string    `gorm:"column:bank;type:varchar(10);not null" json:"bank"`
	VANumber   string    `gorm:"column:va_number;type:varchar(50);not null" json:"va_number"`
	Amount     int64     `gorm:"column:amount;not null" json:"amount"`
	ExpiryTime time.Time `gorm:"column:expiry_time;not null" json:"expiry_time"`
	Status     string    `gorm:"column:status;type:varchar(32);not null;index:idx_payment_attempt_invoice_status,priority:2" json:"status"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (PaymentAttempt) TableName() string { return "payment_attempts" }

func (p *PaymentAttempt) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}
