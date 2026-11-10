package b2bproject

import (
	"time"

	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
)

type RepositoryInterface interface {
	// Project
	ListProjects(status, keyword string) ([]entities.B2BProject, error)
	GetProject(id string) (*entities.B2BProject, error)
	GetProjectByCode(code string) (*entities.B2BProject, error)
	CreateProject(p *entities.B2BProject) error
	SaveProject(p *entities.B2BProject) error
	DeleteProject(id string) error
	AddStatusLog(l *entities.B2BProjectStatusLog) error
	ListStatusLogs(projectID string) ([]entities.B2BProjectStatusLog, error)

	// Quotation
	ListQuotations(projectID string) ([]entities.B2BQuotation, error)
	GetQuotation(id string) (*entities.B2BQuotation, error)
	NextQuotationVersion(projectID string) (int, error)
	CreateQuotation(q *entities.B2BQuotation, items []entities.B2BQuotationItem) error
	SaveQuotation(q *entities.B2BQuotation) error
	DeleteQuotation(id string) error
	SupersedeOtherQuotations(projectID, keepID string) error
	ApprovedQuotationTotal(projectID string) (int64, error)

	// Milestone
	ListMilestones(projectID string) ([]entities.B2BProjectMilestone, error)
	GetMilestone(id string) (*entities.B2BProjectMilestone, error)
	ReplaceMilestones(projectID string, items []entities.B2BProjectMilestone) error
	CreateMilestone(m *entities.B2BProjectMilestone) error
	SaveMilestone(m *entities.B2BProjectMilestone) error
	AllMilestonesPaid(projectID string) (bool, error)

	// Expense
	ListExpenses(projectID string) ([]entities.B2BProjectExpense, error)
	GetExpense(id string) (*entities.B2BProjectExpense, error)
	CreateExpense(e *entities.B2BProjectExpense) error
	DeleteExpense(id string) error

	// Work order & BAST
	ListWorkOrders(projectID string) ([]entities.B2BProjectWorkOrder, error)
	GetWorkOrder(id string) (*entities.B2BProjectWorkOrder, error)
	CreateWorkOrder(w *entities.B2BProjectWorkOrder) error
	SaveWorkOrder(w *entities.B2BProjectWorkOrder) error
	ListBasts(projectID string) ([]entities.B2BProjectBast, error)
	CreateBast(b *entities.B2BProjectBast) error

	// Cash-out ledger (transactions table)
	CreateCashOut(trx *entities.Transaction) error
	DeleteCashOut(id string) error

	// Sequence helpers
	CountProjects() (int64, error)
	CountBasts() (int64, error)
	CountWorkOrders() (int64, error)
	CountMilestones() (int64, error)

	DB() *gorm.DB
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{db} }

func (r *Repository) DB() *gorm.DB { return r.db }

// ---------------------------------------------------------------------------
// Project
// ---------------------------------------------------------------------------

func (r *Repository) ListProjects(status, keyword string) ([]entities.B2BProject, error) {
	out := []entities.B2BProject{}
	q := r.db.Model(&entities.B2BProject{})

	if status != "" {
		q = q.Where("status = ?", status)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("code LIKE ? OR title LIKE ? OR client_name LIKE ?", like, like, like)
	}

	err := q.Order("created_at DESC").Find(&out).Error
	return out, err
}

func (r *Repository) GetProject(id string) (*entities.B2BProject, error) {
	p := &entities.B2BProject{}
	if err := r.db.Preload("Company").Where("id = ?", id).First(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

func (r *Repository) GetProjectByCode(code string) (*entities.B2BProject, error) {
	p := &entities.B2BProject{}
	err := r.db.Where("code = ?", code).First(p).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *Repository) CreateProject(p *entities.B2BProject) error {
	return r.db.Create(p).Error
}

func (r *Repository) SaveProject(p *entities.B2BProject) error {
	return r.db.Save(p).Error
}

func (r *Repository) DeleteProject(id string) error {
	return r.db.Where("id = ?", id).Delete(&entities.B2BProject{}).Error
}

func (r *Repository) AddStatusLog(l *entities.B2BProjectStatusLog) error {
	return r.db.Create(l).Error
}

func (r *Repository) ListStatusLogs(projectID string) ([]entities.B2BProjectStatusLog, error) {
	out := []entities.B2BProjectStatusLog{}
	err := r.db.Where("project_id = ?", projectID).Order("created_at ASC").Find(&out).Error
	return out, err
}

func (r *Repository) CountProjects() (int64, error) {
	var n int64
	err := r.db.Model(&entities.B2BProject{}).Count(&n).Error
	return n, err
}

// ---------------------------------------------------------------------------
// Quotation
// ---------------------------------------------------------------------------

func (r *Repository) ListQuotations(projectID string) ([]entities.B2BQuotation, error) {
	out := []entities.B2BQuotation{}
	err := r.db.Where("project_id = ?", projectID).Order("version DESC").Find(&out).Error
	return out, err
}

func (r *Repository) GetQuotation(id string) (*entities.B2BQuotation, error) {
	q := &entities.B2BQuotation{}
	if err := r.db.Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC, created_at ASC")
	}).Where("id = ?", id).First(q).Error; err != nil {
		return nil, err
	}
	return q, nil
}

func (r *Repository) NextQuotationVersion(projectID string) (int, error) {
	var max int
	err := r.db.Model(&entities.B2BQuotation{}).
		Select("COALESCE(MAX(version), 0)").
		Where("project_id = ?", projectID).
		Scan(&max).Error
	return max + 1, err
}

// CreateQuotation persists the revision and its RAB lines atomically.
func (r *Repository) CreateQuotation(q *entities.B2BQuotation, items []entities.B2BQuotationItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(q).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].QuotationID = q.ID
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) SaveQuotation(q *entities.B2BQuotation) error {
	return r.db.Save(q).Error
}

func (r *Repository) DeleteQuotation(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("quotation_id = ?", id).Delete(&entities.B2BQuotationItem{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&entities.B2BQuotation{}).Error
	})
}

// SupersedeOtherQuotations marks every other revision as superseded so exactly
// one quotation can stay active after an approval.
func (r *Repository) SupersedeOtherQuotations(projectID, keepID string) error {
	return r.db.Model(&entities.B2BQuotation{}).
		Where("project_id = ? AND id <> ? AND status <> ?", projectID, keepID, entities.B2BQuotationApproved).
		Update("status", entities.B2BQuotationSuperseded).Error
}

// ApprovedQuotationTotal returns the grand total of the approved revision, or 0.
func (r *Repository) ApprovedQuotationTotal(projectID string) (int64, error) {
	var total int64
	err := r.db.Model(&entities.B2BQuotation{}).
		Select("COALESCE(grand_total, 0)").
		Where("project_id = ? AND status = ?", projectID, entities.B2BQuotationApproved).
		Order("version DESC").
		Limit(1).
		Scan(&total).Error
	return total, err
}

// ---------------------------------------------------------------------------
// Milestone
// ---------------------------------------------------------------------------

func (r *Repository) ListMilestones(projectID string) ([]entities.B2BProjectMilestone, error) {
	out := []entities.B2BProjectMilestone{}
	err := r.db.Where("project_id = ?", projectID).
		Order("sort_order ASC, created_at ASC").Find(&out).Error
	return out, err
}

func (r *Repository) GetMilestone(id string) (*entities.B2BProjectMilestone, error) {
	m := &entities.B2BProjectMilestone{}
	if err := r.db.Where("id = ?", id).First(m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

// ReplaceMilestones wipes the schedule and writes the new one atomically.
// Only allowed while no term has been invoiced yet.
func (r *Repository) ReplaceMilestones(projectID string, items []entities.B2BProjectMilestone) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", projectID).
			Delete(&entities.B2BProjectMilestone{}).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].ProjectID = projectID
			if err := tx.Create(&items[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) CreateMilestone(m *entities.B2BProjectMilestone) error {
	return r.db.Create(m).Error
}

func (r *Repository) SaveMilestone(m *entities.B2BProjectMilestone) error {
	return r.db.Save(m).Error
}

// AllMilestonesPaid reports whether every term is settled (drives pelunasan).
func (r *Repository) AllMilestonesPaid(projectID string) (bool, error) {
	var total, paid int64
	if err := r.db.Model(&entities.B2BProjectMilestone{}).
		Where("project_id = ?", projectID).Count(&total).Error; err != nil {
		return false, err
	}
	if total == 0 {
		return false, nil
	}
	if err := r.db.Model(&entities.B2BProjectMilestone{}).
		Where("project_id = ? AND status = ?", projectID, entities.B2BMilestonePaid).
		Count(&paid).Error; err != nil {
		return false, err
	}
	return total == paid, nil
}

func (r *Repository) CountMilestones() (int64, error) {
	var n int64
	err := r.db.Model(&entities.B2BProjectMilestone{}).Count(&n).Error
	return n, err
}

// ---------------------------------------------------------------------------
// Expense
// ---------------------------------------------------------------------------

func (r *Repository) ListExpenses(projectID string) ([]entities.B2BProjectExpense, error) {
	out := []entities.B2BProjectExpense{}
	err := r.db.Where("project_id = ?", projectID).
		Order("expense_date DESC, created_at DESC").Find(&out).Error
	return out, err
}

func (r *Repository) GetExpense(id string) (*entities.B2BProjectExpense, error) {
	e := &entities.B2BProjectExpense{}
	if err := r.db.Where("id = ?", id).First(e).Error; err != nil {
		return nil, err
	}
	return e, nil
}

func (r *Repository) CreateExpense(e *entities.B2BProjectExpense) error {
	return r.db.Create(e).Error
}

func (r *Repository) DeleteExpense(id string) error {
	return r.db.Where("id = ?", id).Delete(&entities.B2BProjectExpense{}).Error
}

// ---------------------------------------------------------------------------
// Work order & BAST
// ---------------------------------------------------------------------------

func (r *Repository) ListWorkOrders(projectID string) ([]entities.B2BProjectWorkOrder, error) {
	out := []entities.B2BProjectWorkOrder{}
	err := r.db.Where("project_id = ?", projectID).Order("created_at DESC").Find(&out).Error
	return out, err
}

func (r *Repository) GetWorkOrder(id string) (*entities.B2BProjectWorkOrder, error) {
	w := &entities.B2BProjectWorkOrder{}
	if err := r.db.Where("id = ?", id).First(w).Error; err != nil {
		return nil, err
	}
	return w, nil
}

func (r *Repository) CreateWorkOrder(w *entities.B2BProjectWorkOrder) error {
	return r.db.Create(w).Error
}

func (r *Repository) SaveWorkOrder(w *entities.B2BProjectWorkOrder) error {
	return r.db.Save(w).Error
}

func (r *Repository) CountWorkOrders() (int64, error) {
	var n int64
	err := r.db.Model(&entities.B2BProjectWorkOrder{}).Count(&n).Error
	return n, err
}

func (r *Repository) ListBasts(projectID string) ([]entities.B2BProjectBast, error) {
	out := []entities.B2BProjectBast{}
	err := r.db.Where("project_id = ?", projectID).Order("created_at DESC").Find(&out).Error
	return out, err
}

func (r *Repository) CreateBast(b *entities.B2BProjectBast) error {
	return r.db.Create(b).Error
}

func (r *Repository) CountBasts() (int64, error) {
	var n int64
	err := r.db.Model(&entities.B2BProjectBast{}).Count(&n).Error
	return n, err
}

// ---------------------------------------------------------------------------
// Cash-out ledger
//
// The expense module writes into the shared `transactions` table so project
// costs appear in the company cash flow report without duplicating logic.
// ---------------------------------------------------------------------------

func (r *Repository) CreateCashOut(trx *entities.Transaction) error {
	return r.db.Create(trx).Error
}

func (r *Repository) DeleteCashOut(id string) error {
	return r.db.Where("id = ?", id).Delete(&entities.Transaction{}).Error
}

// nowPtr is a small helper used across the service layer.
func nowPtr() *time.Time {
	now := time.Now()
	return &now
}
