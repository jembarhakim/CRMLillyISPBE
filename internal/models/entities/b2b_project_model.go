package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =============================================================================
// B2B Incidental Project (borongan) — Sprint 6
//
// Retail customers live in the `customer` table and flow through recurring
// invoices. B2B projects are a separate world: one-off contracts with a
// company, a negotiation cycle, staged payments and field costs.
// =============================================================================

// B2BProjectStatus is the commercial lifecycle of a project.
type B2BProjectStatus string

const (
	B2BStatusPenawaran   B2BProjectStatus = "penawaran"
	B2BStatusDeal        B2BProjectStatus = "deal"
	B2BStatusPengerjaan  B2BProjectStatus = "pengerjaan"
	B2BStatusSerahTerima B2BProjectStatus = "serah_terima"
	B2BStatusPelunasan   B2BProjectStatus = "pelunasan"
	B2BStatusBatal       B2BProjectStatus = "batal"
)

// B2BProjectStatusOrder is the allowed forward progression. A project never
// jumps backwards without an explicit admin action.
var B2BProjectStatusOrder = []B2BProjectStatus{
	B2BStatusPenawaran,
	B2BStatusDeal,
	B2BStatusPengerjaan,
	B2BStatusSerahTerima,
	B2BStatusPelunasan,
}

// B2BProject is the master record for one incidental B2B job.
type B2BProject struct {
	ID     string           `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	Code   string           `gorm:"column:code;type:varchar(50);uniqueIndex" json:"code"`
	Title  string           `gorm:"column:title;type:varchar(191);not null" json:"title"`
	Status B2BProjectStatus `gorm:"column:status;type:varchar(30);not null;default:'penawaran'" json:"status"`

	// Client company (B2B buyer). The optional customer link lets the project
	// reuse a retail customer record for invoicing when one exists.
	ClientCompanyID string `gorm:"column:client_company_id;type:varchar(191);index:idx_b2b_client_company" json:"client_company_id"`
	CustomerID      string `gorm:"column:customer_id;type:varchar(191);index:idx_b2b_customer" json:"customer_id"`
	ClientName      string `gorm:"column:client_name;type:varchar(191)" json:"client_name"`

	// PIC on the client side.
	PICName  string `gorm:"column:pic_name;type:varchar(191)" json:"pic_name"`
	PICPhone string `gorm:"column:pic_phone;type:varchar(50)" json:"pic_phone"`
	PICEmail string `gorm:"column:pic_email;type:varchar(191)" json:"pic_email"`

	// Contract value agreed with the client: budget ceiling for margin tracking.
	ContractValue int64 `gorm:"column:contract_value;type:bigint;not null;default:0" json:"contract_value"`

	StartDate     *time.Time `gorm:"column:start_date;type:date" json:"start_date"`
	TargetEndDate *time.Time `gorm:"column:target_end_date;type:date" json:"target_end_date"`
	Location      string     `gorm:"column:location;type:varchar(191)" json:"location"`
	Notes         string     `gorm:"column:notes;type:text" json:"notes"`

	CreatedBy string    `gorm:"column:created_by;type:varchar(191)" json:"created_by"`
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP(3)" json:"updated_at"`

	Company *Company `gorm:"foreignKey:ClientCompanyID;references:ID" json:"company,omitempty"`
}

func (B2BProject) TableName() string { return "b2b_projects" }

func (p *B2BProject) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	if p.Status == "" {
		p.Status = B2BStatusPenawaran
	}
	return nil
}

// B2BProjectStatusLog keeps an audit trail of every status transition so the
// detail page can render a timeline.
type B2BProjectStatusLog struct {
	ID         string           `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	ProjectID  string           `gorm:"column:project_id;type:varchar(191);not null;index:idx_b2b_log_project" json:"project_id"`
	FromStatus B2BProjectStatus `gorm:"column:from_status;type:varchar(30)" json:"from_status"`
	ToStatus   B2BProjectStatus `gorm:"column:to_status;type:varchar(30);not null" json:"to_status"`
	Note       string           `gorm:"column:note;type:text" json:"note"`
	ChangedBy  string           `gorm:"column:changed_by;type:varchar(191)" json:"changed_by"`
	CreatedAt  time.Time        `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
}

func (B2BProjectStatusLog) TableName() string { return "b2b_project_status_logs" }

func (l *B2BProjectStatusLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	return nil
}

// B2BMilestoneStatus is the payment state of one termin.
type B2BMilestoneStatus string

const (
	B2BMilestonePending  B2BMilestoneStatus = "pending"
	B2BMilestoneInvoiced B2BMilestoneStatus = "invoiced"
	B2BMilestonePaid     B2BMilestoneStatus = "paid"
)

// B2BProjectMilestone is one scheduled payment term (DP / progress / pelunasan).
type B2BProjectMilestone struct {
	ID        string `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	ProjectID string `gorm:"column:project_id;type:varchar(191);not null;index:idx_b2b_milestone_project" json:"project_id"`

	Name       string  `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Percentage float64 `gorm:"column:percentage;type:decimal(5,2);not null" json:"percentage"`
	Amount     int64   `gorm:"column:amount;type:bigint;not null;default:0" json:"amount"`

	DueDate *time.Time         `gorm:"column:due_date;type:date" json:"due_date"`
	Status  B2BMilestoneStatus `gorm:"column:status;type:varchar(20);not null;default:'pending'" json:"status"`

	// InvoiceNumber is generated when the term is issued. CRMInvoiceID is only
	// filled when the project has a linked retail customer (see service layer).
	InvoiceNumber string     `gorm:"column:invoice_number;type:varchar(60)" json:"invoice_number"`
	CRMInvoiceID  string     `gorm:"column:crm_invoice_id;type:varchar(191)" json:"crm_invoice_id"`
	InvoicedAt    *time.Time `gorm:"column:invoiced_at;type:datetime" json:"invoiced_at"`
	PaidAt        *time.Time `gorm:"column:paid_at;type:datetime" json:"paid_at"`

	SortOrder int       `gorm:"column:sort_order;type:int;default:0" json:"sort_order"`
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP(3)" json:"updated_at"`
}

func (B2BProjectMilestone) TableName() string { return "b2b_project_milestones" }

func (m *B2BProjectMilestone) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	if m.Status == "" {
		m.Status = B2BMilestonePending
	}
	return nil
}
