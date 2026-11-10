package b2bproject

import "skripsi-be/internal/models/entities"

// ---------------------------------------------------------------------------
// Project
// ---------------------------------------------------------------------------

type CreateProjectRequest struct {
	Code            string `json:"code" validate:"required,max=50"`
	Title           string `json:"title" validate:"required,max=191"`
	ClientCompanyID string `json:"client_company_id"`
	CustomerID      string `json:"customer_id"`
	ClientName      string `json:"client_name" validate:"required,max=191"`
	PICName         string `json:"pic_name" validate:"max=191"`
	PICPhone        string `json:"pic_phone" validate:"max=50"`
	PICEmail        string `json:"pic_email" validate:"omitempty,email"`
	ContractValue   int64  `json:"contract_value" validate:"gte=0"`
	StartDate       string `json:"start_date"`
	TargetEndDate   string `json:"target_end_date"`
	Location        string `json:"location"`
	Notes           string `json:"notes"`
}

type UpdateProjectRequest struct {
	CreateProjectRequest
}

// ChangeProjectStatusRequest drives the lifecycle timeline.
type ChangeProjectStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=penawaran deal pengerjaan serah_terima pelunasan batal"`
	Note   string `json:"note" validate:"max=500"`
}

// ---------------------------------------------------------------------------
// Quotation (RAB)
// ---------------------------------------------------------------------------

type QuotationItemInput struct {
	Category  string  `json:"category" validate:"required,oneof=perangkat kabel jasa lainnya"`
	Name      string  `json:"name" validate:"required,max=191"`
	Unit      string  `json:"unit" validate:"max=30"`
	Qty       float64 `json:"qty" validate:"required,gt=0"`
	UnitPrice int64   `json:"unit_price" validate:"gte=0"`
	Note      string  `json:"note" validate:"max=191"`
}

type CreateQuotationRequest struct {
	Subject    string               `json:"subject" validate:"max=191"`
	Notes      string               `json:"notes"`
	Terms      string               `json:"terms"`
	TaxPercent float64              `json:"tax_percent" validate:"gte=0,lte=100"`
	ValidUntil string               `json:"valid_until"`
	Items      []QuotationItemInput `json:"items" validate:"required,min=1,dive"`
}

type SetQuotationStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=draft sent approved rejected superseded"`
}

// ---------------------------------------------------------------------------
// Milestone (termin)
// ---------------------------------------------------------------------------

type MilestoneInput struct {
	Name       string  `json:"name" validate:"required,max=100"`
	Percentage float64 `json:"percentage" validate:"required,gt=0,lte=100"`
	DueDate    string  `json:"due_date"`
	SortOrder  int     `json:"sort_order"`
}

// GenerateMilestonesRequest accepts a preset (dp_50_100 | dp_30_50_20 | dp_50_50)
// or an explicit list of terms.
type GenerateMilestonesRequest struct {
	Preset string           `json:"preset" validate:"omitempty,oneof=dp_50_100 dp_30_50_20 dp_50_50"`
	Base   string           `json:"base" validate:"omitempty,oneof=contract quotation"`
	Items  []MilestoneInput `json:"items" validate:"omitempty,min=1,dive"`
}

type UpdateMilestoneRequest struct {
	Name       string  `json:"name" validate:"max=100"`
	Percentage float64 `json:"percentage" validate:"gte=0,lte=100"`
	Amount     int64   `json:"amount" validate:"gte=0"`
	DueDate    string  `json:"due_date"`
}

// ---------------------------------------------------------------------------
// Expense / margin
// ---------------------------------------------------------------------------

type CreateExpenseRequest struct {
	Category    string `json:"category" validate:"required,oneof=material transport upah lainnya"`
	Description string `json:"description" validate:"required,max=191"`
	Amount      int64  `json:"amount" validate:"required,gt=0"`
	ExpenseDate string `json:"expense_date"`
	AccountID   string `json:"account_id"`
	ReceiptURL  string `json:"receipt_url" validate:"max=255"`
}

// ---------------------------------------------------------------------------
// Work order & BAST
// ---------------------------------------------------------------------------

type CreateWorkOrderRequest struct {
	Title         string `json:"title" validate:"required,max=191"`
	Description   string `json:"description"`
	ScheduledDate string `json:"scheduled_date"`
	AssignedTo    string `json:"assigned_to"`
	AssigneeName  string `json:"assignee_name" validate:"max=191"`
}

type UpdateWorkOrderRequest struct {
	Status        string `json:"status" validate:"omitempty,oneof=open in_progress completed cancelled"`
	Title         string `json:"title" validate:"max=191"`
	Description   string `json:"description"`
	ScheduledDate string `json:"scheduled_date"`
	AssignedTo    string `json:"assigned_to"`
	AssigneeName  string `json:"assignee_name" validate:"max=191"`
}

// BastItem is one row of the hand-over checklist (installed equipment).
type BastItem struct {
	Name string  `json:"name" validate:"required,max=191"`
	Qty  float64 `json:"qty" validate:"required,gt=0"`
	Unit string  `json:"unit" validate:"max=30"`
	Note string  `json:"note" validate:"max=191"`
}

type GenerateBastRequest struct {
	WorkOrderID    string     `json:"work_order_id"`
	Number         string     `json:"number" validate:"max=60"`
	SignedByClient string     `json:"signed_by_client" validate:"max=191"`
	SignedByVendor string     `json:"signed_by_vendor" validate:"max=191"`
	SignedDate     string     `json:"signed_date"`
	Notes          string     `json:"notes"`
	Items          []BastItem `json:"items" validate:"required,min=1,dive"`
}

// ---------------------------------------------------------------------------
// Read models (response payloads)
// ---------------------------------------------------------------------------

// MarginSummary powers the budget-vs-actual widget.
type MarginSummary struct {
	Budget        int64            `json:"budget"`        // approved quotation or contract value
	BudgetSource  string           `json:"budget_source"` // "quotation" | "contract"
	Actual        int64            `json:"actual"`        // sum of recorded expenses
	Remaining     int64            `json:"remaining"`
	UsagePercent  float64          `json:"usage_percent"`
	IsOverBudget  bool             `json:"is_over_budget"`
	ByCategory    []CategoryMargin `json:"by_category"`
	ContractValue int64            `json:"contract_value"`
	MarginAmount  int64            `json:"margin_amount"` // contract - actual
	MarginPercent float64          `json:"margin_percent"`
}

type CategoryMargin struct {
	Category string  `json:"category"`
	Label    string  `json:"label"`
	Amount   int64   `json:"amount"`
	Percent  float64 `json:"percent"`
}

// ProjectDetail bundles everything the detail page needs in one round trip.
type ProjectDetail struct {
	Project    entities.B2BProject            `json:"project"`
	StatusLogs []entities.B2BProjectStatusLog `json:"status_logs"`
	Milestones []entities.B2BProjectMilestone `json:"milestones"`
	Quotations []entities.B2BQuotation        `json:"quotations"`
	Expenses   []entities.B2BProjectExpense   `json:"expenses"`
	WorkOrders []entities.B2BProjectWorkOrder `json:"work_orders"`
	Basts      []entities.B2BProjectBast      `json:"basts"`
	Margin     MarginSummary                  `json:"margin"`
}
