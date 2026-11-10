// custID dereferences a nullable customer id ("" when nil).

package b2bproject

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"skripsi-be/internal/models/entities"
)

func custID(id *string) string {
	if id == nil {
		return ""
	}
	return *id
}

type ServiceInterface interface {
	// Project
	ListProjects(status, keyword string) ([]entities.B2BProject, error)
	CreateProject(req CreateProjectRequest, actor string) (*entities.B2BProject, error)
	GetProjectDetail(id string) (*ProjectDetail, error)
	UpdateProject(id string, req UpdateProjectRequest) (*entities.B2BProject, error)
	DeleteProject(id string) error
	ChangeStatus(id string, req ChangeProjectStatusRequest, actor string) (*entities.B2BProject, error)

	// Quotation
	ListQuotations(projectID string) ([]entities.B2BQuotation, error)
	GetQuotation(id string) (*entities.B2BQuotation, error)
	CreateQuotation(projectID string, req CreateQuotationRequest, actor string) (*entities.B2BQuotation, error)
	SetQuotationStatus(quotationID string, req SetQuotationStatusRequest) (*entities.B2BQuotation, error)
	DeleteQuotation(quotationID string) error

	// Milestone
	ListMilestones(projectID string) ([]entities.B2BProjectMilestone, error)
	GenerateMilestones(projectID string, req GenerateMilestonesRequest) ([]entities.B2BProjectMilestone, error)
	UpdateMilestone(milestoneID string, req UpdateMilestoneRequest) (*entities.B2BProjectMilestone, error)
	IssueMilestoneInvoice(milestoneID string) (*entities.B2BProjectMilestone, error)
	MarkMilestonePaid(milestoneID string) (*entities.B2BProjectMilestone, error)

	// Expense & margin
	ListExpenses(projectID string) ([]entities.B2BProjectExpense, error)
	CreateExpense(projectID string, req CreateExpenseRequest, actor string) (*entities.B2BProjectExpense, error)
	DeleteExpense(expenseID string) error
	Margin(projectID string) (MarginSummary, error)

	// Work order & BAST
	ListWorkOrders(projectID string) ([]entities.B2BProjectWorkOrder, error)
	CreateWorkOrder(projectID string, req CreateWorkOrderRequest, actor string) (*entities.B2BProjectWorkOrder, error)
	UpdateWorkOrder(woID string, req UpdateWorkOrderRequest) (*entities.B2BProjectWorkOrder, error)
	ListBasts(projectID string) ([]entities.B2BProjectBast, error)
	GenerateBast(projectID string, req GenerateBastRequest, actor string) (*entities.B2BProjectBast, error)
}

type Service struct {
	repo RepositoryInterface
}

func NewService(repo RepositoryInterface) *Service {
	return &Service{repo: repo}
}

// ---------------------------------------------------------------------------
// Small helpers
// ---------------------------------------------------------------------------

var categoryLabels = map[entities.B2BExpenseCategory]string{
	entities.B2BExpenseMaterial:  "Material",
	entities.B2BExpenseTransport: "Transport",
	entities.B2BExpenseWage:      "Upah Harian",
	entities.B2BExpenseOther:     "Lainnya",
}

// parseDate accepts "YYYY-MM-DD" or RFC3339; empty means nil.
func parseDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	layouts := []string{"2006-01-02", time.RFC3339, "2006-01-02 15:04:05"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("format tanggal tidak valid: %s (gunakan YYYY-MM-DD)", value)
}

// round2 keeps percentages readable without float noise.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// percentOf returns pct% of base, rounded to whole rupiah.
func percentOf(base int64, pct float64) int64 {
	return int64(math.Round(float64(base) * pct / 100))
}

// buildProjectCode generates PRJ-YYYYMM-#### when the caller leaves it empty.
func (s *Service) buildProjectCode() (string, error) {
	n, err := s.repo.CountProjects()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("PRJ-%s-%04d", time.Now().Format("200601"), n+1), nil
}

func (s *Service) buildWorkOrderNumber() (string, error) {
	n, err := s.repo.CountWorkOrders()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("WO-%s-%04d", time.Now().Format("200601"), n+1), nil
}

func (s *Service) buildBastNumber() (string, error) {
	n, err := s.repo.CountBasts()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("BAST-%s-%04d", time.Now().Format("200601"), n+1), nil
}

// isForwardTransition tells whether newStatus is at or after current status.
func isForwardTransition(current, next entities.B2BProjectStatus) bool {
	idx := func(s entities.B2BProjectStatus) int {
		for i, v := range entities.B2BProjectStatusOrder {
			if v == s {
				return i
			}
		}
		return -1
	}
	c, n := idx(current), idx(next)
	if n < 0 {
		return false
	}
	if c < 0 {
		return true
	}
	return n >= c
}

// ---------------------------------------------------------------------------
// Project
// ---------------------------------------------------------------------------

func (s *Service) ListProjects(status, keyword string) ([]entities.B2BProject, error) {
	return s.repo.ListProjects(status, keyword)
}

func (s *Service) CreateProject(req CreateProjectRequest, actor string) (*entities.B2BProject, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		generated, err := s.buildProjectCode()
		if err != nil {
			return nil, err
		}
		code = generated
	}

	existing, err := s.repo.GetProjectByCode(code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("kode proyek %s sudah dipakai", code)
	}

	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return nil, err
	}
	targetEnd, err := parseDate(req.TargetEndDate)
	if err != nil {
		return nil, err
	}
	if startDate != nil && targetEnd != nil && targetEnd.Before(*startDate) {
		return nil, fmt.Errorf("target selesai tidak boleh sebelum tanggal mulai")
	}

	project := &entities.B2BProject{
		Code:            code,
		Title:           strings.TrimSpace(req.Title),
		Status:          entities.B2BStatusPenawaran,
		ClientCompanyID: req.ClientCompanyID,
		CustomerID:      req.CustomerID,
		ClientName:      strings.TrimSpace(req.ClientName),
		PICName:         req.PICName,
		PICPhone:        req.PICPhone,
		PICEmail:        req.PICEmail,
		ContractValue:   req.ContractValue,
		StartDate:       startDate,
		TargetEndDate:   targetEnd,
		Location:        req.Location,
		Notes:           req.Notes,
		CreatedBy:       actor,
	}

	if err := s.repo.CreateProject(project); err != nil {
		return nil, err
	}

	// Seed the timeline so the detail page always has at least one entry.
	_ = s.repo.AddStatusLog(&entities.B2BProjectStatusLog{
		ProjectID:  project.ID,
		FromStatus: "",
		ToStatus:   entities.B2BStatusPenawaran,
		Note:       "Proyek dibuat",
		ChangedBy:  actor,
	})

	return project, nil
}

func (s *Service) GetProjectDetail(id string) (*ProjectDetail, error) {
	project, err := s.repo.GetProject(id)
	if err != nil {
		return nil, fmt.Errorf("proyek tidak ditemukan")
	}

	detail := &ProjectDetail{Project: *project}

	if detail.StatusLogs, err = s.repo.ListStatusLogs(id); err != nil {
		return nil, err
	}
	if detail.Milestones, err = s.repo.ListMilestones(id); err != nil {
		return nil, err
	}
	if detail.Quotations, err = s.repo.ListQuotations(id); err != nil {
		return nil, err
	}
	if detail.Expenses, err = s.repo.ListExpenses(id); err != nil {
		return nil, err
	}
	if detail.WorkOrders, err = s.repo.ListWorkOrders(id); err != nil {
		return nil, err
	}
	if detail.Basts, err = s.repo.ListBasts(id); err != nil {
		return nil, err
	}

	// Margin is computed from the same rows already loaded, so the dashboard
	// and the list never disagree.
	detail.Margin = s.computeMargin(*project, detail.Expenses, detail.Quotations)

	return detail, nil
}

func (s *Service) UpdateProject(id string, req UpdateProjectRequest) (*entities.B2BProject, error) {
	project, err := s.repo.GetProject(id)
	if err != nil {
		return nil, fmt.Errorf("proyek tidak ditemukan")
	}

	if code := strings.TrimSpace(req.Code); code != "" && code != project.Code {
		dupe, err := s.repo.GetProjectByCode(code)
		if err != nil {
			return nil, err
		}
		if dupe != nil && dupe.ID != project.ID {
			return nil, fmt.Errorf("kode proyek %s sudah dipakai", code)
		}
		project.Code = code
	}

	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return nil, err
	}
	targetEnd, err := parseDate(req.TargetEndDate)
	if err != nil {
		return nil, err
	}
	if startDate != nil && targetEnd != nil && targetEnd.Before(*startDate) {
		return nil, fmt.Errorf("target selesai tidak boleh sebelum tanggal mulai")
	}

	project.Title = strings.TrimSpace(req.Title)
	project.ClientCompanyID = req.ClientCompanyID
	project.CustomerID = req.CustomerID
	project.ClientName = strings.TrimSpace(req.ClientName)
	project.PICName = req.PICName
	project.PICPhone = req.PICPhone
	project.PICEmail = req.PICEmail
	project.ContractValue = req.ContractValue
	project.StartDate = startDate
	project.TargetEndDate = targetEnd
	project.Location = req.Location
	project.Notes = req.Notes

	if err := s.repo.SaveProject(project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *Service) DeleteProject(id string) error {
	if _, err := s.repo.GetProject(id); err != nil {
		return fmt.Errorf("proyek tidak ditemukan")
	}
	return s.repo.DeleteProject(id)
}

func (s *Service) ChangeStatus(id string, req ChangeProjectStatusRequest, actor string) (*entities.B2BProject, error) {
	project, err := s.repo.GetProject(id)
	if err != nil {
		return nil, fmt.Errorf("proyek tidak ditemukan")
	}

	next := entities.B2BProjectStatus(req.Status)
	if next == project.Status {
		return nil, fmt.Errorf("status proyek sudah %s", req.Status)
	}

	// Cancelling is always allowed; other moves must not go backwards.
	if next != entities.B2BStatusBatal && !isForwardTransition(project.Status, next) {
		return nil, fmt.Errorf("status tidak boleh mundur dari %s ke %s", project.Status, req.Status)
	}

	previous := project.Status
	project.Status = next

	if err := s.repo.SaveProject(project); err != nil {
		return nil, err
	}

	_ = s.repo.AddStatusLog(&entities.B2BProjectStatusLog{
		ProjectID:  project.ID,
		FromStatus: previous,
		ToStatus:   next,
		Note:       req.Note,
		ChangedBy:  actor,
	})

	return project, nil
}

// ---------------------------------------------------------------------------
// Quotation (RAB)
// ---------------------------------------------------------------------------

func (s *Service) ListQuotations(projectID string) ([]entities.B2BQuotation, error) {
	return s.repo.ListQuotations(projectID)
}

func (s *Service) GetQuotation(id string) (*entities.B2BQuotation, error) {
	q, err := s.repo.GetQuotation(id)
	if err != nil {
		return nil, fmt.Errorf("penawaran tidak ditemukan")
	}
	return q, nil
}

// CreateQuotation stores a new revision (version auto-increments) and computes
// the money server-side so the client can never post a forged total.
func (s *Service) CreateQuotation(projectID string, req CreateQuotationRequest, actor string) (*entities.B2BQuotation, error) {
	project, err := s.repo.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("proyek tidak ditemukan")
	}

	if len(req.Items) == 0 {
		return nil, fmt.Errorf("penawaran harus memiliki minimal satu item")
	}

	version, err := s.repo.NextQuotationVersion(projectID)
	if err != nil {
		return nil, err
	}

	validUntil, err := parseDate(req.ValidUntil)
	if err != nil {
		return nil, err
	}

	taxPercent := req.TaxPercent
	if taxPercent <= 0 {
		taxPercent = 11 // default PPN
	}

	items := make([]entities.B2BQuotationItem, 0, len(req.Items))
	var subtotal int64

	for idx, in := range req.Items {
		lineTotal := int64(math.Round(in.Qty * float64(in.UnitPrice)))
		subtotal += lineTotal

		items = append(items, entities.B2BQuotationItem{
			Category:  entities.B2BQuotationItemCategory(in.Category),
			Name:      strings.TrimSpace(in.Name),
			Unit:      in.Unit,
			Qty:       in.Qty,
			UnitPrice: in.UnitPrice,
			Subtotal:  lineTotal,
			Note:      in.Note,
			SortOrder: idx,
		})
	}

	taxAmount := percentOf(subtotal, taxPercent)

	quotation := &entities.B2BQuotation{
		ProjectID:  project.ID,
		Version:    version,
		Status:     entities.B2BQuotationDraft,
		Subject:    req.Subject,
		Notes:      req.Notes,
		Terms:      req.Terms,
		Subtotal:   subtotal,
		TaxPercent: taxPercent,
		TaxAmount:  taxAmount,
		GrandTotal: subtotal + taxAmount,
		ValidUntil: validUntil,
		CreatedBy:  actor,
	}

	if err := s.repo.CreateQuotation(quotation, items); err != nil {
		return nil, err
	}

	quotation.Items = items
	return quotation, nil
}

// SetQuotationStatus approves/rejects a revision. Approval supersedes older
// revisions and syncs the project contract value to the approved grand total.
func (s *Service) SetQuotationStatus(quotationID string, req SetQuotationStatusRequest) (*entities.B2BQuotation, error) {
	quotation, err := s.repo.GetQuotation(quotationID)
	if err != nil {
		return nil, fmt.Errorf("penawaran tidak ditemukan")
	}

	status := entities.B2BQuotationStatus(req.Status)
	quotation.Status = status

	if err := s.repo.SaveQuotation(quotation); err != nil {
		return nil, err
	}

	if status == entities.B2BQuotationApproved {
		// Only one approved revision may exist per project.
		if err := s.repo.SupersedeOtherQuotations(quotation.ProjectID, quotation.ID); err != nil {
			return nil, err
		}

		// The approved RAB becomes the contract value / budget baseline.
		if project, err := s.repo.GetProject(quotation.ProjectID); err == nil {
			project.ContractValue = quotation.GrandTotal
			if project.Status == entities.B2BStatusPenawaran {
				project.Status = entities.B2BStatusDeal
			}
			_ = s.repo.SaveProject(project)
			_ = s.repo.AddStatusLog(&entities.B2BProjectStatusLog{
				ProjectID:  project.ID,
				FromStatus: entities.B2BStatusPenawaran,
				ToStatus:   entities.B2BStatusDeal,
				Note:       fmt.Sprintf("Penawaran Rev %d disetujui klien", quotation.Version),
			})
		}
	}

	return quotation, nil
}

func (s *Service) DeleteQuotation(quotationID string) error {
	quotation, err := s.repo.GetQuotation(quotationID)
	if err != nil {
		return fmt.Errorf("penawaran tidak ditemukan")
	}
	if quotation.Status == entities.B2BQuotationApproved {
		return fmt.Errorf("penawaran yang sudah disetujui tidak dapat dihapus")
	}
	return s.repo.DeleteQuotation(quotationID)
}

// ---------------------------------------------------------------------------
// Milestone (termin pembayaran)
// ---------------------------------------------------------------------------

func (s *Service) ListMilestones(projectID string) ([]entities.B2BProjectMilestone, error) {
	return s.repo.ListMilestones(projectID)
}

// milestonePresets maps the termin schemes the sales team uses most.
var milestonePresets = map[string][]MilestoneInput{
	"dp_50_100":   {{Name: "DP 50%", Percentage: 50}, {Name: "Pelunasan 100%", Percentage: 50}},
	"dp_30_50_20": {{Name: "DP 30%", Percentage: 30}, {Name: "Progress 50%", Percentage: 50}, {Name: "Pelunasan 20%", Percentage: 20}},
	"dp_50_50":    {{Name: "DP 50%", Percentage: 50}, {Name: "Pelunasan 50%", Percentage: 50}},
}

// GenerateMilestones creates the payment schedule using either the approved
// quotation total or the project contract value as the base.
func (s *Service) GenerateMilestones(projectID string, req GenerateMilestonesRequest) ([]entities.B2BProjectMilestone, error) {
	project, err := s.repo.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("proyek tidak ditemukan")
	}

	// Guard: never silently regenerate a schedule that already produced invoices.
	existing, err := s.repo.ListMilestones(projectID)
	if err != nil {
		return nil, err
	}
	for _, m := range existing {
		if m.Status != entities.B2BMilestonePending {
			return nil, fmt.Errorf("termin sudah diterbitkan, jadwal tidak dapat dibuat ulang")
		}
	}

	inputs := req.Items
	if len(inputs) == 0 {
		preset := req.Preset
		if preset == "" {
			preset = "dp_50_100"
		}
		inputs = milestonePresets[preset]
	}
	if len(inputs) == 0 {
		return nil, fmt.Errorf("tidak ada termin yang bisa dibuat")
	}

	// Percentages must add up to 100 so the schedule covers the contract.
	var totalPct float64
	for _, in := range inputs {
		totalPct += in.Percentage
	}
	if math.Abs(totalPct-100) > 0.01 {
		return nil, fmt.Errorf("total persentase termin harus 100%% (saat ini %.2f%%)", totalPct)
	}

	base := project.ContractValue
	if req.Base == "quotation" || base <= 0 {
		if qTotal, err := s.repo.ApprovedQuotationTotal(projectID); err == nil && qTotal > 0 {
			base = qTotal
		}
	}
	if base <= 0 {
		return nil, fmt.Errorf("nilai kontrak/penawaran belum ada, isi nilai kontrak terlebih dahulu")
	}

	items := make([]entities.B2BProjectMilestone, 0, len(inputs))
	for idx, in := range inputs {
		dueDate, err := parseDate(in.DueDate)
		if err != nil {
			return nil, err
		}
		items = append(items, entities.B2BProjectMilestone{
			Name:       in.Name,
			Percentage: in.Percentage,
			Amount:     percentOf(base, in.Percentage),
			DueDate:    dueDate,
			Status:     entities.B2BMilestonePending,
			SortOrder:  idx,
		})
	}

	if err := s.repo.ReplaceMilestones(projectID, items); err != nil {
		return nil, err
	}

	return s.repo.ListMilestones(projectID)
}

func (s *Service) UpdateMilestone(milestoneID string, req UpdateMilestoneRequest) (*entities.B2BProjectMilestone, error) {
	m, err := s.repo.GetMilestone(milestoneID)
	if err != nil {
		return nil, fmt.Errorf("termin tidak ditemukan")
	}
	if m.Status == entities.B2BMilestonePaid {
		return nil, fmt.Errorf("termin yang sudah lunas tidak dapat diubah")
	}

	if name := strings.TrimSpace(req.Name); name != "" {
		m.Name = name
	}
	if req.Percentage > 0 {
		m.Percentage = req.Percentage
	}
	if req.Amount > 0 {
		m.Amount = req.Amount
	}
	dueDate, err := parseDate(req.DueDate)
	if err != nil {
		return nil, err
	}
	if dueDate != nil {
		m.DueDate = dueDate
	}

	if err := s.repo.SaveMilestone(m); err != nil {
		return nil, err
	}
	return m, nil
}

// IssueMilestoneInvoice generates the official invoice number for a term.
//
// A CRM `invoices` row is only created when the project is linked to a retail
// customer, because that table enforces customer_id NOT NULL + FK. For pure B2B
// projects the term itself carries the invoice identity.
func (s *Service) IssueMilestoneInvoice(milestoneID string) (*entities.B2BProjectMilestone, error) {
	m, err := s.repo.GetMilestone(milestoneID)
	if err != nil {
		return nil, fmt.Errorf("termin tidak ditemukan")
	}
	if m.Status != entities.B2BMilestonePending {
		return nil, fmt.Errorf("termin ini sudah diterbitkan")
	}

	project, err := s.repo.GetProject(m.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("proyek tidak ditemukan")
	}

	if m.InvoiceNumber == "" {
		m.InvoiceNumber = fmt.Sprintf("INV/%s/%d", project.Code, time.Now().Unix()%100000)
	}

	m.Status = entities.B2BMilestoneInvoiced
	m.InvoicedAt = nowPtr()

	if err := s.repo.SaveMilestone(m); err != nil {
		return nil, err
	}
	return m, nil
}

// MarkMilestonePaid settles a term. Once every term is paid the project moves
// to pelunasan automatically.
func (s *Service) MarkMilestonePaid(milestoneID string) (*entities.B2BProjectMilestone, error) {
	m, err := s.repo.GetMilestone(milestoneID)
	if err != nil {
		return nil, fmt.Errorf("termin tidak ditemukan")
	}
	if m.Status == entities.B2BMilestonePaid {
		return nil, fmt.Errorf("termin sudah lunas")
	}

	m.Status = entities.B2BMilestonePaid
	m.PaidAt = nowPtr()
	if m.InvoiceNumber == "" {
		m.InvoiceNumber = fmt.Sprintf("PAID/%d", time.Now().Unix()%100000)
	}

	if err := s.repo.SaveMilestone(m); err != nil {
		return nil, err
	}

	// Auto-advance the project when the last term is settled.
	if paidAll, err := s.repo.AllMilestonesPaid(m.ProjectID); err == nil && paidAll {
		if project, err := s.repo.GetProject(m.ProjectID); err == nil {
			if project.Status != entities.B2BStatusPelunasan {
				previous := project.Status
				project.Status = entities.B2BStatusPelunasan
				_ = s.repo.SaveProject(project)
				_ = s.repo.AddStatusLog(&entities.B2BProjectStatusLog{
					ProjectID:  project.ID,
					FromStatus: previous,
					ToStatus:   entities.B2BStatusPelunasan,
					Note:       "Seluruh termin telah lunas",
				})
			}
		}
	}

	return m, nil
}

// ---------------------------------------------------------------------------
// Expense & margin tracker
// ---------------------------------------------------------------------------

func (s *Service) ListExpenses(projectID string) ([]entities.B2BProjectExpense, error) {
	return s.repo.ListExpenses(projectID)
}

// CreateExpense records a field cost and mirrors it into the company cash-out
// ledger (`transactions`) so the finance report stays the single source of truth.
func (s *Service) CreateExpense(projectID string, req CreateExpenseRequest, actor string) (*entities.B2BProjectExpense, error) {
	project, err := s.repo.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("proyek tidak ditemukan")
	}

	expenseDate, err := parseDate(req.ExpenseDate)
	if err != nil {
		return nil, err
	}
	if expenseDate == nil {
		expenseDate = nowPtr()
	}

	expense := &entities.B2BProjectExpense{
		ProjectID:   project.ID,
		Category:    entities.B2BExpenseCategory(req.Category),
		Description: strings.TrimSpace(req.Description),
		Amount:      req.Amount,
		ExpenseDate: expenseDate,
		AccountID:   req.AccountID,
		ReceiptURL:  req.ReceiptURL,
		CreatedBy:   actor,
	}

	if err := s.repo.CreateExpense(expense); err != nil {
		return nil, err
	}

	// Cash-out mirror. AccountID is required by the transactions table, so the
	// ledger row is only written when finance picked a cash account.
	if req.AccountID != "" {
		trx := &entities.Transaction{
			AccountID:   req.AccountID,
			TypeCash:    entities.TransactionsTypeCashCashFlow,
			TypeInOut:   entities.TransactionsTypeInOutOut, // credit = kas keluar
			Description: fmt.Sprintf("[%s] %s", project.Code, expense.Description),
			Amount:      expense.Amount,
			Category:    string(expense.Category),
			Method:      "transfer",
		}
		if err := s.repo.CreateCashOut(trx); err == nil {
			expense.TransactionID = trx.ID
			_ = s.repo.DB().Model(&entities.B2BProjectExpense{}).
				Where("id = ?", expense.ID).
				Update("transaction_id", trx.ID).Error
		}
	}

	return expense, nil
}

// DeleteExpense removes the cost and its mirrored cash-out entry.
func (s *Service) DeleteExpense(expenseID string) error {
	expense, err := s.repo.GetExpense(expenseID)
	if err != nil {
		return fmt.Errorf("pengeluaran tidak ditemukan")
	}
	if expense.TransactionID != "" {
		_ = s.repo.DeleteCashOut(expense.TransactionID)
	}
	return s.repo.DeleteExpense(expenseID)
}

func (s *Service) Margin(projectID string) (MarginSummary, error) {
	project, err := s.repo.GetProject(projectID)
	if err != nil {
		return MarginSummary{}, fmt.Errorf("proyek tidak ditemukan")
	}
	expenses, err := s.repo.ListExpenses(projectID)
	if err != nil {
		return MarginSummary{}, err
	}
	quotations, err := s.repo.ListQuotations(projectID)
	if err != nil {
		return MarginSummary{}, err
	}
	return s.computeMargin(*project, expenses, quotations), nil
}

// computeMargin compares the budget baseline against actual field spending.
// Budget preference: approved quotation total, else project contract value.
func (s *Service) computeMargin(project entities.B2BProject, expenses []entities.B2BProjectExpense, quotations []entities.B2BQuotation) MarginSummary {
	summary := MarginSummary{ContractValue: project.ContractValue}

	budget := project.ContractValue
	summary.BudgetSource = "contract"

	for _, q := range quotations {
		if q.Status == entities.B2BQuotationApproved {
			budget = q.GrandTotal
			summary.BudgetSource = "quotation"
			break
		}
	}
	summary.Budget = budget

	perCategory := map[entities.B2BExpenseCategory]int64{}
	for _, e := range expenses {
		summary.Actual += e.Amount
		perCategory[e.Category] += e.Amount
	}

	summary.Remaining = budget - summary.Actual
	summary.IsOverBudget = summary.Actual > budget && budget > 0

	if budget > 0 {
		summary.UsagePercent = round2(float64(summary.Actual) / float64(budget) * 100)
	}
	summary.MarginAmount = project.ContractValue - summary.Actual
	if project.ContractValue > 0 {
		summary.MarginPercent = round2(float64(summary.MarginAmount) / float64(project.ContractValue) * 100)
	}

	// Stable category order for the chart legend.
	order := []entities.B2BExpenseCategory{
		entities.B2BExpenseMaterial,
		entities.B2BExpenseTransport,
		entities.B2BExpenseWage,
		entities.B2BExpenseOther,
	}
	summary.ByCategory = make([]CategoryMargin, 0, len(order))
	for _, cat := range order {
		amount := perCategory[cat]
		row := CategoryMargin{
			Category: string(cat),
			Label:    categoryLabels[cat],
			Amount:   amount,
		}
		if summary.Actual > 0 {
			row.Percent = round2(float64(amount) / float64(summary.Actual) * 100)
		}
		summary.ByCategory = append(summary.ByCategory, row)
	}

	return summary
}

// ---------------------------------------------------------------------------
// Work order & BAST
// ---------------------------------------------------------------------------

func (s *Service) ListWorkOrders(projectID string) ([]entities.B2BProjectWorkOrder, error) {
	return s.repo.ListWorkOrders(projectID)
}

func (s *Service) CreateWorkOrder(projectID string, req CreateWorkOrderRequest, actor string) (*entities.B2BProjectWorkOrder, error) {
	project, err := s.repo.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("proyek tidak ditemukan")
	}

	number, err := s.buildWorkOrderNumber()
	if err != nil {
		return nil, err
	}

	scheduled, err := parseDate(req.ScheduledDate)
	if err != nil {
		return nil, err
	}

	wo := &entities.B2BProjectWorkOrder{
		ProjectID:     project.ID,
		Number:        number,
		Title:         strings.TrimSpace(req.Title),
		Description:   req.Description,
		Status:        entities.B2BWOOpen,
		ScheduledDate: scheduled,
		AssignedTo:    req.AssignedTo,
		AssigneeName:  req.AssigneeName,
		CreatedBy:     actor,
	}

	if err := s.repo.CreateWorkOrder(wo); err != nil {
		return nil, err
	}

	// Creating the first WO moves a deal into execution.
	if project.Status == entities.B2BStatusDeal {
		previous := project.Status
		project.Status = entities.B2BStatusPengerjaan
		_ = s.repo.SaveProject(project)
		_ = s.repo.AddStatusLog(&entities.B2BProjectStatusLog{
			ProjectID:  project.ID,
			FromStatus: previous,
			ToStatus:   entities.B2BStatusPengerjaan,
			Note:       "Work Order dibuat, pengerjaan dimulai",
			ChangedBy:  actor,
		})
	}

	return wo, nil
}

func (s *Service) UpdateWorkOrder(woID string, req UpdateWorkOrderRequest) (*entities.B2BProjectWorkOrder, error) {
	wo, err := s.repo.GetWorkOrder(woID)
	if err != nil {
		return nil, fmt.Errorf("work order tidak ditemukan")
	}

	if title := strings.TrimSpace(req.Title); title != "" {
		wo.Title = title
	}
	if req.Description != "" {
		wo.Description = req.Description
	}
	if req.AssignedTo != "" {
		wo.AssignedTo = req.AssignedTo
	}
	if req.AssigneeName != "" {
		wo.AssigneeName = req.AssigneeName
	}

	scheduled, err := parseDate(req.ScheduledDate)
	if err != nil {
		return nil, err
	}
	if scheduled != nil {
		wo.ScheduledDate = scheduled
	}

	if req.Status != "" {
		wo.Status = entities.B2BWorkOrderStatus(req.Status)
		if wo.Status == entities.B2BWOCompleted && wo.CompletedDate == nil {
			wo.CompletedDate = nowPtr()
		}
	}

	if err := s.repo.SaveWorkOrder(wo); err != nil {
		return nil, err
	}
	return wo, nil
}

func (s *Service) ListBasts(projectID string) ([]entities.B2BProjectBast, error) {
	return s.repo.ListBasts(projectID)
}

// GenerateBast records the hand-over certificate.
//
// It snapshots the installed items into JSON (so the printed document is stable
// even if the RAB changes later), completes the linked work order and drives the
// parent project to serah_terima — the GORM-style "hook" required by WO-06.
func (s *Service) GenerateBast(projectID string, req GenerateBastRequest, actor string) (*entities.B2BProjectBast, error) {
	project, err := s.repo.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("proyek tidak ditemukan")
	}
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("daftar perangkat terpasang tidak boleh kosong")
	}

	number := strings.TrimSpace(req.Number)
	if number == "" {
		generated, err := s.buildBastNumber()
		if err != nil {
			return nil, err
		}
		number = generated
	}

	snapshot, err := json.Marshal(req.Items)
	if err != nil {
		return nil, err
	}

	signedDate, err := parseDate(req.SignedDate)
	if err != nil {
		return nil, err
	}

	bast := &entities.B2BProjectBast{
		ProjectID:      project.ID,
		WorkOrderID:    req.WorkOrderID,
		Number:         number,
		ItemsSnapshot:  string(snapshot),
		SignedByClient: req.SignedByClient,
		SignedByVendor: req.SignedByVendor,
		SignedDate:     signedDate,
		Notes:          req.Notes,
		CreatedBy:      actor,
	}

	if err := s.repo.CreateBast(bast); err != nil {
		return nil, err
	}

	// Hook 1: complete the referenced work order.
	if req.WorkOrderID != "" {
		if wo, err := s.repo.GetWorkOrder(req.WorkOrderID); err == nil {
			wo.Status = entities.B2BWOCompleted
			if wo.CompletedDate == nil {
				wo.CompletedDate = nowPtr()
			}
			_ = s.repo.SaveWorkOrder(wo)
		}
	}

	// Hook 2: hand-over puts the project into serah_terima.
	if project.Status != entities.B2BStatusSerahTerima && project.Status != entities.B2BStatusPelunasan {
		previous := project.Status
		project.Status = entities.B2BStatusSerahTerima
		_ = s.repo.SaveProject(project)
		_ = s.repo.AddStatusLog(&entities.B2BProjectStatusLog{
			ProjectID:  project.ID,
			FromStatus: previous,
			ToStatus:   entities.B2BStatusSerahTerima,
			Note:       fmt.Sprintf("BAST %s ditandatangani", bast.Number),
			ChangedBy:  actor,
		})
	}

	return bast, nil
}
