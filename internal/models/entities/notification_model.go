package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Notification struct {
	ID               string     `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	Recipient        string     `gorm:"column:recipient;type:varchar(32);not null" json:"recipient"`
	TemplateKey      string     `gorm:"column:template_key;type:varchar(100);not null" json:"template_key"`
	Message          string     `gorm:"column:message;type:text;not null" json:"message"`
	Status           string     `gorm:"column:status;type:varchar(20);not null;index:idx_notifications_status_next_attempt" json:"status"`
	Attempts         int        `gorm:"column:attempts;not null;default:0" json:"attempts"`
	MaxAttempts      int        `gorm:"column:max_attempts;not null;default:5" json:"max_attempts"`
	NextAttemptAt    time.Time  `gorm:"column:next_attempt_at;not null;index:idx_notifications_status_next_attempt" json:"next_attempt_at"`
	DeduplicationKey *string    `gorm:"column:deduplication_key;type:varchar(191);uniqueIndex:idx_notifications_deduplication_key" json:"deduplication_key,omitempty"`
	InvoiceID        *string    `gorm:"column:invoice_id;type:varchar(191);index" json:"invoice_id,omitempty"`
	BroadcastID      *string    `gorm:"column:broadcast_id;type:varchar(191);index" json:"broadcast_id,omitempty"`
	GatewayMessageID *string    `gorm:"column:gateway_message_id;type:varchar(191);index" json:"gateway_message_id,omitempty"`
	LastError        *string    `gorm:"column:last_error;type:text" json:"last_error,omitempty"`
	SentAt           *time.Time `gorm:"column:sent_at" json:"sent_at,omitempty"`
	CreatedAt        time.Time  `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;default:CURRENT_TIMESTAMP(3)" json:"updated_at"`
}

func (Notification) TableName() string { return "notifications" }

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	return nil
}

type NotificationLog struct {
	ID             string    `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	NotificationID string    `gorm:"column:notification_id;type:varchar(191);not null;index" json:"notification_id"`
	Status         string    `gorm:"column:status;type:varchar(20);not null" json:"status"`
	Attempt        int       `gorm:"column:attempt;not null;default:0" json:"attempt"`
	Details        string    `gorm:"column:details;type:text" json:"details,omitempty"`
	CreatedAt      time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
}

func (NotificationLog) TableName() string { return "notification_logs" }

func (l *NotificationLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.NewString()
	}
	return nil
}

type NotificationTemplate struct {
	Key       string    `gorm:"column:key;type:varchar(100);primaryKey" json:"key"`
	Name      string    `gorm:"column:name;type:varchar(191);not null" json:"name"`
	Category  string    `gorm:"column:category;type:varchar(50);not null" json:"category"`
	Body      string    `gorm:"column:body;type:text;not null" json:"body"`
	Enabled   bool      `gorm:"column:enabled;not null;default:true" json:"enabled"`
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP(3)" json:"updated_at"`
}

func (NotificationTemplate) TableName() string { return "notification_templates" }

type NotificationPreference struct {
	CustomerID              string    `gorm:"column:customer_id;type:varchar(191);primaryKey" json:"customer_id"`
	InvoiceRemindersEnabled bool      `gorm:"column:invoice_reminders_enabled;not null;default:true" json:"invoice_reminders_enabled"`
	UpdatedAt               time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP(3)" json:"updated_at"`
}

func (NotificationPreference) TableName() string { return "notification_preferences" }
