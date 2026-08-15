package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RouterJobStatus string

const (
	RouterJobStatusPending RouterJobStatus = "pending"
	RouterJobStatusSuccess RouterJobStatus = "success"
	RouterJobStatusError   RouterJobStatus = "error"
)

// RouterJob represents an asynchronous MikroTik operation tied to an invoice
type RouterJob struct {
	ID         string          `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	InvoiceID  string          `gorm:"column:invoice_id;type:varchar(191);index:idx_router_jobs_invoice" json:"invoice_id"`
	Action     string          `gorm:"column:action;type:varchar(64);not null" json:"action"`
	UniqueKey  string          `gorm:"column:unique_key;type:varchar(128);uniqueIndex" json:"unique_key"`
	Status     RouterJobStatus `gorm:"column:status;type:enum('pending','success','error');default:'pending'" json:"status"`
	RetryCount int             `gorm:"column:retry_count;type:int;default:0" json:"retry_count"`
	NextRunAt  time.Time       `gorm:"column:next_run_at;type:datetime;index:idx_router_jobs_nextrun" json:"next_run_at"`
	LastError  *string         `gorm:"column:last_error;type:text" json:"last_error,omitempty"`
	CreatedAt  time.Time       `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt  time.Time       `gorm:"column:updated_at;not null" json:"updated_at"`
}

func (RouterJob) TableName() string { return "router_jobs" }

func (rj *RouterJob) BeforeCreate(tx *gorm.DB) error {
	if rj.ID == "" {
		rj.ID = uuid.New().String()
	}
	if rj.Status == "" {
		rj.Status = RouterJobStatusPending
	}
	if rj.NextRunAt.IsZero() {
		rj.NextRunAt = time.Now()
	}
	if rj.UniqueKey == "" {
		rj.UniqueKey = rj.InvoiceID + ":" + rj.Action
	}
	return nil
}
