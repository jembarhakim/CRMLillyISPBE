package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InvoicePayment struct {
	ID            string    `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	InvoiceID     string    `gorm:"column:invoice_id;type:varchar(191);not null;index" json:"invoice_id"`
	Amount        int64     `gorm:"column:amount;not null" json:"amount"`
	PaymentType   string    `gorm:"column:payment_type;type:varchar(50);not null;index" json:"payment_type"`
	Method        string    `gorm:"column:method;type:varchar(50);not null" json:"method"`
	ReceiptPath   string    `gorm:"column:receipt_path;type:varchar(500);not null" json:"receipt_path"`
	Reason        string    `gorm:"column:reason;type:text" json:"reason"`
	CollectedBy   string    `gorm:"column:collected_by;type:varchar(191);not null;index" json:"collected_by"`
	ClosingID     *string   `gorm:"column:closing_id;type:varchar(191);index" json:"closing_id,omitempty"`
	TransactionID *string   `gorm:"column:transaction_id;type:varchar(191)" json:"transaction_id,omitempty"`
	CreatedAt     time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	Invoice       *Invoice  `gorm:"foreignKey:InvoiceID;references:ID" json:"invoice,omitempty"`
	Collector     *User     `gorm:"foreignKey:CollectedBy;references:ID" json:"collector,omitempty"`
}

func (InvoicePayment) TableName() string { return "invoice_payments" }

func (p *InvoicePayment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

type FieldPaymentClosing struct {
	ID          string    `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	CollectorID string    `gorm:"column:collector_id;type:varchar(191);not null;index" json:"collector_id"`
	AccountID   string    `gorm:"column:account_id;type:varchar(191);not null" json:"account_id"`
	VerifiedBy  string    `gorm:"column:verified_by;type:varchar(191);not null" json:"verified_by"`
	Total       int64     `gorm:"column:total;not null" json:"total"`
	CreatedAt   time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
}

func (FieldPaymentClosing) TableName() string { return "field_payment_closings" }

func (c *FieldPaymentClosing) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

type CustomerIsolationLog struct {
	ID           string    `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	CustomerID   string    `gorm:"column:customer_id;type:varchar(191);not null;index" json:"customer_id"`
	Action       string    `gorm:"column:action;type:varchar(20);not null" json:"action"`
	Reason       string    `gorm:"column:reason;type:text;not null" json:"reason"`
	MacAddresses string    `gorm:"column:mac_addresses;type:text;not null" json:"mac_addresses"`
	Result       string    `gorm:"column:result;type:text;not null" json:"result"`
	PerformedBy  string    `gorm:"column:performed_by;type:varchar(191);not null;index" json:"performed_by"`
	CreatedAt    time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	User         *User     `gorm:"foreignKey:PerformedBy;references:ID" json:"user,omitempty"`
}

func (CustomerIsolationLog) TableName() string { return "customer_isolation_logs" }

func (l *CustomerIsolationLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	return nil
}
