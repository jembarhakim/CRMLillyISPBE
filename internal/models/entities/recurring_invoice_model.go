package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RecurringInvoiceFrequency string

const (
	RecurringInvoiceFrequencyMonthly   RecurringInvoiceFrequency = "monthly"
	RecurringInvoiceFrequencyQuarterly RecurringInvoiceFrequency = "quarterly"
	RecurringInvoiceFrequencyYearly    RecurringInvoiceFrequency = "yearly"
)

type RecurringInvoiceStatus string

const (
	RecurringInvoiceStatusActive    RecurringInvoiceStatus = "active"
	RecurringInvoiceStatusStopped   RecurringInvoiceStatus = "stopped"
	RecurringInvoiceStatusCompleted RecurringInvoiceStatus = "completed"
)

type RecurringInvoiceItem struct {
	Name  string `json:"name"`
	Price int64  `json:"price"`
	Qty   int64  `json:"qty"`
	Total int64  `json:"total"`
}

type RecurringInvoice struct {
	ID                   string                    `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	CustomerID           string                    `gorm:"column:customer_id;type:varchar(191);not null" json:"customer_id"`
	Customer             Customer                  `gorm:"foreignKey:CustomerID;references:id;constraint:OnUpdate:RESTRICT" json:"customer"`
	Amount               int64                     `gorm:"column:amount;type:int;not null" json:"amount"`
	InvoiceDate          time.Time                 `gorm:"column:invoice_date;type:date;not null" json:"invoice_date"`
	DueDate              time.Time                 `gorm:"column:due_date;type:date;not null" json:"due_date"`
	NextInvoiceDate      time.Time                 `gorm:"column:next_invoice_date;type:date;not null" json:"next_invoice_date"`
	OriginalDay          int                       `gorm:"column:original_day;type:int;not null" json:"original_day"` // Preserve the original template day (1-31)
	CustomerInstallation *string                   `gorm:"column:customer_installation;type:varchar(191)" json:"customer_installation,omitempty"`
	Frequency            RecurringInvoiceFrequency `gorm:"column:frequency;type:enum('monthly','quarterly','yearly');default:'monthly'" json:"frequency"`
	Status               RecurringInvoiceStatus    `gorm:"column:status;type:enum('active','stopped','completed');default:'active'" json:"status"`
	Description          *string                   `gorm:"column:description;type:text" json:"description"`
	InvoiceItems         string                    `gorm:"column:invoice_items;type:json;not null" json:"-"` // Store as JSON string
	InvoiceItemsData     []RecurringInvoiceItem    `gorm:"-" json:"invoice_items"`                           // Parsed items for API
	CreatedAt            time.Time                 `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt            time.Time                 `gorm:"column:updated_at;not null" json:"updated_at"`
	CreatedBy            *string                   `gorm:"column:created_by;type:varchar(191)" json:"created_by"`
	CreatedByUser        *User                     `gorm:"foreignKey:CreatedBy;references:id;constraint:OnUpdate:RESTRICT" json:"created_by_user,omitempty"`

	// Related data
	History []RecurringInvoiceHistory `gorm:"foreignKey:RecurringInvoiceID;constraint:OnUpdate:RESTRICT" json:"history,omitempty"`
}

func (RecurringInvoice) TableName() string {
	return "recurring_invoices"
}

func (r *RecurringInvoice) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	if r.Status == "" {
		r.Status = RecurringInvoiceStatusActive
	}
	return nil
}

func (r *RecurringInvoice) AfterFind(tx *gorm.DB) error {
	// Parse JSON invoice items
	if r.InvoiceItems != "" {
		// This would need JSON unmarshaling in the repository layer
		// For now, we'll handle it in the repository
	}
	return nil
}

type RecurringInvoiceHistory struct {
	ID                 string           `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	RecurringInvoiceID string           `gorm:"column:recurring_invoice_id;type:varchar(191);not null" json:"recurring_invoice_id"`
	RecurringInvoice   RecurringInvoice `gorm:"foreignKey:RecurringInvoiceID;references:id;constraint:OnUpdate:RESTRICT" json:"recurring_invoice,omitempty"`
	GeneratedInvoiceID string           `gorm:"column:generated_invoice_id;type:varchar(191);not null" json:"generated_invoice_id"`
	GeneratedInvoice   Invoice          `gorm:"foreignKey:GeneratedInvoiceID;references:id;constraint:OnUpdate:RESTRICT" json:"generated_invoice,omitempty"`
	GeneratedAt        time.Time        `gorm:"column:generated_at;default:CURRENT_TIMESTAMP(3)" json:"generated_at"`
	InvoiceDate        time.Time        `gorm:"column:invoice_date;type:date;not null" json:"invoice_date"`
	DueDate            time.Time        `gorm:"column:due_date;type:date;not null" json:"due_date"`
}

func (RecurringInvoiceHistory) TableName() string {
	return "recurring_invoice_history"
}

func (r *RecurringInvoiceHistory) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}
