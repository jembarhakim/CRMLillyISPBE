package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =============================================================================
// B2B field operations — expenses, work orders, BAST
// =============================================================================

// B2BExpenseCategory is the cost bucket used for the margin breakdown.
type B2BExpenseCategory string

const (
	B2BExpenseMaterial  B2BExpenseCategory = "material"
	B2BExpenseTransport B2BExpenseCategory = "transport"
	B2BExpenseWage      B2BExpenseCategory = "upah"
	B2BExpenseOther     B2BExpenseCategory = "lainnya"
)

// B2BProjectExpense is one field cost. When recorded it also writes a matching
// row into the company cash-out ledger (`transactions`) so finance has a single
// source of truth.
type B2BProjectExpense struct {
	ID        string `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	ProjectID string `gorm:"column:project_id;type:varchar(191);not null;index:idx_b2b_expense_project" json:"project_id"`

	Category    B2BExpenseCategory `gorm:"column:category;type:varchar(20);not null;default:'material'" json:"category"`
	Description string             `gorm:"column:description;type:varchar(191);not null" json:"description"`
	Amount      int64              `gorm:"column:amount;type:bigint;not null;default:0" json:"amount"`
	ExpenseDate *time.Time         `gorm:"column:expense_date;type:date" json:"expense_date"`

	// AccountID is the cash/bank account charged; TransactionID links to the
	// created cash-out ledger row.
	AccountID     string `gorm:"column:account_id;type:varchar(191)" json:"account_id"`
	TransactionID string `gorm:"column:transaction_id;type:varchar(191)" json:"transaction_id"`

	ReceiptURL string    `gorm:"column:receipt_url;type:varchar(255)" json:"receipt_url"`
	CreatedBy  string    `gorm:"column:created_by;type:varchar(191)" json:"created_by"`
	CreatedAt  time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP(3)" json:"updated_at"`
}

func (B2BProjectExpense) TableName() string { return "b2b_project_expenses" }

func (e *B2BProjectExpense) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	return nil
}

// B2BWorkOrderStatus is the field execution state.
type B2BWorkOrderStatus string

const (
	B2BWOOpen       B2BWorkOrderStatus = "open"
	B2BWOInProgress B2BWorkOrderStatus = "in_progress"
	B2BWOCompleted  B2BWorkOrderStatus = "completed"
	B2BWOCancelled  B2BWorkOrderStatus = "cancelled"
)

// B2BProjectWorkOrder is the field job ticket bound to a project.
type B2BProjectWorkOrder struct {
	ID        string `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	ProjectID string `gorm:"column:project_id;type:varchar(191);not null;index:idx_b2b_wo_project" json:"project_id"`

	Number      string             `gorm:"column:number;type:varchar(60);uniqueIndex" json:"number"`
	Title       string             `gorm:"column:title;type:varchar(191);not null" json:"title"`
	Description string             `gorm:"column:description;type:text" json:"description"`
	Status      B2BWorkOrderStatus `gorm:"column:status;type:varchar(20);not null;default:'open'" json:"status"`

	ScheduledDate *time.Time `gorm:"column:scheduled_date;type:date" json:"scheduled_date"`
	CompletedDate *time.Time `gorm:"column:completed_date;type:date" json:"completed_date"`

	AssignedTo   string `gorm:"column:assigned_to;type:varchar(191)" json:"assigned_to"`
	AssigneeName string `gorm:"column:assignee_name;type:varchar(191)" json:"assignee_name"`

	CreatedBy string    `gorm:"column:created_by;type:varchar(191)" json:"created_by"`
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP(3)" json:"updated_at"`
}

func (B2BProjectWorkOrder) TableName() string { return "b2b_project_work_orders" }

func (w *B2BProjectWorkOrder) BeforeCreate(tx *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	if w.Status == "" {
		w.Status = B2BWOOpen
	}
	return nil
}

// B2BProjectBast is the hand-over certificate (Berita Acara Serah Terima).
// Signing it drives the parent project to serah_terima.
type B2BProjectBast struct {
	ID        string `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	ProjectID string `gorm:"column:project_id;type:varchar(191);not null;index:idx_b2b_bast_project" json:"project_id"`
	WorkOrderID string `gorm:"column:work_order_id;type:varchar(191)" json:"work_order_id"`

	Number string `gorm:"column:number;type:varchar(60);uniqueIndex" json:"number"`

	// Snapshot of installed items at signing time, stored as JSON text so the
	// printed document never changes when the asset list is edited later.
	ItemsSnapshot string `gorm:"column:items_snapshot;type:longtext" json:"items_snapshot"`

	SignedByClient string     `gorm:"column:signed_by_client;type:varchar(191)" json:"signed_by_client"`
	SignedByVendor string     `gorm:"column:signed_by_vendor;type:varchar(191)" json:"signed_by_vendor"`
	SignedDate     *time.Time `gorm:"column:signed_date;type:date" json:"signed_date"`
	Notes          string     `gorm:"column:notes;type:text" json:"notes"`

	CreatedBy string    `gorm:"column:created_by;type:varchar(191)" json:"created_by"`
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP(3)" json:"updated_at"`
}

func (B2BProjectBast) TableName() string { return "b2b_project_basts" }

func (b *B2BProjectBast) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}
