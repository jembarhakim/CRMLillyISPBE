package b2bproject

import (
	"errors"
	"strings"

	"skripsi-be/internal/api/common/validation"
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
)

// errString wraps a validation message into an error without extra imports at
// every call site.
func errString(msg string) error { return errors.New(msg) }

type Handler struct {
	service ServiceInterface
}

func NewHandler(service ServiceInterface) *Handler {
	return &Handler{service: service}
}

// actorID reads the authenticated staff id (set by helpers.VerifyToken).
func actorID(c *fiber.Ctx) string {
	if v, ok := c.Locals("user_id").(string); ok {
		return v
	}
	return ""
}

// respondError maps service errors to a sensible HTTP status.
func respondError(c *fiber.Ctx, err error, notFound bool) error {
	status := fiber.StatusBadRequest
	if notFound {
		status = fiber.StatusNotFound
	}
	return helpers.ResponseUtils(c, status, false, err.Error(), nil)
}

// ---------------------------------------------------------------------------
// Project
// ---------------------------------------------------------------------------

// List handles GET /api/admin/b2b-project
func (h *Handler) List(c *fiber.Ctx) error {
	projects, err := h.service.ListProjects(c.Query("status"), c.Query("q"))
	if err != nil {
		return respondError(c, err, false)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Daftar proyek B2B", projects)
}

// Create handles POST /api/admin/b2b-project
func (h *Handler) Create(c *fiber.Ctx) error {
	req := new(CreateProjectRequest)
	if err := c.BodyParser(req); err != nil {
		return respondError(c, err, false)
	}
	if msgs := validation.ValidationRequest(req); msgs != nil {
		return respondError(c, errString(strings.Join(msgs, ", ")), false)
	}

	project, err := h.service.CreateProject(*req, actorID(c))
	if err != nil {
		return respondError(c, err, false)
	}
	return helpers.ResponseUtils(c, fiber.StatusCreated, true, "Proyek berhasil dibuat", project)
}

// Detail handles GET /api/admin/b2b-project/:id
func (h *Handler) Detail(c *fiber.Ctx) error {
	detail, err := h.service.GetProjectDetail(c.Params("id"))
	if err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Detail proyek", detail)
}

// Update handles PUT /api/admin/b2b-project/:id
func (h *Handler) Update(c *fiber.Ctx) error {
	req := new(UpdateProjectRequest)
	if err := c.BodyParser(req); err != nil {
		return respondError(c, err, false)
	}
	if msgs := validation.ValidationRequest(req); msgs != nil {
		return respondError(c, errString(strings.Join(msgs, ", ")), false)
	}

	project, err := h.service.UpdateProject(c.Params("id"), *req)
	if err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Proyek diperbarui", project)
}

// Delete handles DELETE /api/admin/b2b-project/:id
func (h *Handler) Delete(c *fiber.Ctx) error {
	if err := h.service.DeleteProject(c.Params("id")); err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Proyek dihapus", nil)
}

// ChangeStatus handles POST /api/admin/b2b-project/:id/status
func (h *Handler) ChangeStatus(c *fiber.Ctx) error {
	req := new(ChangeProjectStatusRequest)
	if err := c.BodyParser(req); err != nil {
		return respondError(c, err, false)
	}
	if msgs := validation.ValidationRequest(req); msgs != nil {
		return respondError(c, errString(strings.Join(msgs, ", ")), false)
	}

	project, err := h.service.ChangeStatus(c.Params("id"), *req, actorID(c))
	if err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Status proyek diperbarui", project)
}

// ---------------------------------------------------------------------------
// Quotation
// ---------------------------------------------------------------------------

// ListQuotations handles GET /api/admin/b2b-project/:id/quotations
func (h *Handler) ListQuotations(c *fiber.Ctx) error {
	items, err := h.service.ListQuotations(c.Params("id"))
	if err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Riwayat penawaran", items)
}

// CreateQuotation handles POST /api/admin/b2b-project/:id/quotations
func (h *Handler) CreateQuotation(c *fiber.Ctx) error {
	req := new(CreateQuotationRequest)
	if err := c.BodyParser(req); err != nil {
		return respondError(c, err, false)
	}
	if msgs := validation.ValidationRequest(req); msgs != nil {
		return respondError(c, errString(strings.Join(msgs, ", ")), false)
	}

	quotation, err := h.service.CreateQuotation(c.Params("id"), *req, actorID(c))
	if err != nil {
		return respondError(c, err, false)
	}
	return helpers.ResponseUtils(c, fiber.StatusCreated, true, "Penawaran dibuat", quotation)
}

// GetQuotation handles GET /api/admin/b2b-project/quotation/:quotationId
func (h *Handler) GetQuotation(c *fiber.Ctx) error {
	quotation, err := h.service.GetQuotation(c.Params("quotationId"))
	if err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Detail penawaran", quotation)
}

// SetQuotationStatus handles PUT /api/admin/b2b-project/quotation/:quotationId/status
func (h *Handler) SetQuotationStatus(c *fiber.Ctx) error {
	req := new(SetQuotationStatusRequest)
	if err := c.BodyParser(req); err != nil {
		return respondError(c, err, false)
	}
	if msgs := validation.ValidationRequest(req); msgs != nil {
		return respondError(c, errString(strings.Join(msgs, ", ")), false)
	}

	quotation, err := h.service.SetQuotationStatus(c.Params("quotationId"), *req)
	if err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Status penawaran diperbarui", quotation)
}

// DeleteQuotation handles DELETE /api/admin/b2b-project/quotation/:quotationId
func (h *Handler) DeleteQuotation(c *fiber.Ctx) error {
	if err := h.service.DeleteQuotation(c.Params("quotationId")); err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Penawaran dihapus", nil)
}

// ---------------------------------------------------------------------------
// Milestone
// ---------------------------------------------------------------------------

// ListMilestones handles GET /api/admin/b2b-project/:id/milestones
func (h *Handler) ListMilestones(c *fiber.Ctx) error {
	items, err := h.service.ListMilestones(c.Params("id"))
	if err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Termin pembayaran", items)
}

// GenerateMilestones handles POST /api/admin/b2b-project/:id/milestones
func (h *Handler) GenerateMilestones(c *fiber.Ctx) error {
	req := new(GenerateMilestonesRequest)
	if err := c.BodyParser(req); err != nil {
		return respondError(c, err, false)
	}
	if msgs := validation.ValidationRequest(req); msgs != nil {
		return respondError(c, errString(strings.Join(msgs, ", ")), false)
	}

	items, err := h.service.GenerateMilestones(c.Params("id"), *req)
	if err != nil {
		return respondError(c, err, false)
	}
	return helpers.ResponseUtils(c, fiber.StatusCreated, true, "Jadwal termin dibuat", items)
}

// UpdateMilestone handles PUT /api/admin/b2b-project/milestone/:milestoneId
func (h *Handler) UpdateMilestone(c *fiber.Ctx) error {
	req := new(UpdateMilestoneRequest)
	if err := c.BodyParser(req); err != nil {
		return respondError(c, err, false)
	}

	m, err := h.service.UpdateMilestone(c.Params("milestoneId"), *req)
	if err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Termin diperbarui", m)
}

// IssueMilestone handles POST /api/admin/b2b-project/milestone/:milestoneId/issue
func (h *Handler) IssueMilestone(c *fiber.Ctx) error {
	m, err := h.service.IssueMilestoneInvoice(c.Params("milestoneId"))
	if err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Invoice termin diterbitkan", m)
}

// PayMilestone handles POST /api/admin/b2b-project/milestone/:milestoneId/pay
func (h *Handler) PayMilestone(c *fiber.Ctx) error {
	m, err := h.service.MarkMilestonePaid(c.Params("milestoneId"))
	if err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Termin ditandai lunas", m)
}

// ---------------------------------------------------------------------------
// Expense & margin
// ---------------------------------------------------------------------------

// ListExpenses handles GET /api/admin/b2b-project/:id/expenses
func (h *Handler) ListExpenses(c *fiber.Ctx) error {
	items, err := h.service.ListExpenses(c.Params("id"))
	if err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Pengeluaran proyek", items)
}

// CreateExpense handles POST /api/admin/b2b-project/:id/expenses
func (h *Handler) CreateExpense(c *fiber.Ctx) error {
	req := new(CreateExpenseRequest)
	if err := c.BodyParser(req); err != nil {
		return respondError(c, err, false)
	}
	if msgs := validation.ValidationRequest(req); msgs != nil {
		return respondError(c, errString(strings.Join(msgs, ", ")), false)
	}

	expense, err := h.service.CreateExpense(c.Params("id"), *req, actorID(c))
	if err != nil {
		return respondError(c, err, false)
	}
	return helpers.ResponseUtils(c, fiber.StatusCreated, true, "Pengeluaran dicatat", expense)
}

// DeleteExpense handles DELETE /api/admin/b2b-project/expense/:expenseId
func (h *Handler) DeleteExpense(c *fiber.Ctx) error {
	if err := h.service.DeleteExpense(c.Params("expenseId")); err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Pengeluaran dihapus", nil)
}

// Margin handles GET /api/admin/b2b-project/:id/margin
func (h *Handler) Margin(c *fiber.Ctx) error {
	summary, err := h.service.Margin(c.Params("id"))
	if err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Ringkasan margin", summary)
}

// ---------------------------------------------------------------------------
// Work order & BAST
// ---------------------------------------------------------------------------

// ListWorkOrders handles GET /api/admin/b2b-project/:id/work-orders
func (h *Handler) ListWorkOrders(c *fiber.Ctx) error {
	items, err := h.service.ListWorkOrders(c.Params("id"))
	if err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Work order proyek", items)
}

// CreateWorkOrder handles POST /api/admin/b2b-project/:id/work-orders
func (h *Handler) CreateWorkOrder(c *fiber.Ctx) error {
	req := new(CreateWorkOrderRequest)
	if err := c.BodyParser(req); err != nil {
		return respondError(c, err, false)
	}
	if msgs := validation.ValidationRequest(req); msgs != nil {
		return respondError(c, errString(strings.Join(msgs, ", ")), false)
	}

	wo, err := h.service.CreateWorkOrder(c.Params("id"), *req, actorID(c))
	if err != nil {
		return respondError(c, err, false)
	}
	return helpers.ResponseUtils(c, fiber.StatusCreated, true, "Work order dibuat", wo)
}

// UpdateWorkOrder handles PUT /api/admin/b2b-project/work-order/:woId
func (h *Handler) UpdateWorkOrder(c *fiber.Ctx) error {
	req := new(UpdateWorkOrderRequest)
	if err := c.BodyParser(req); err != nil {
		return respondError(c, err, false)
	}

	wo, err := h.service.UpdateWorkOrder(c.Params("woId"), *req)
	if err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Work order diperbarui", wo)
}

// ListBasts handles GET /api/admin/b2b-project/:id/basts
func (h *Handler) ListBasts(c *fiber.Ctx) error {
	items, err := h.service.ListBasts(c.Params("id"))
	if err != nil {
		return respondError(c, err, true)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Daftar BAST", items)
}

// GenerateBast handles POST /api/admin/b2b-project/:id/basts
func (h *Handler) GenerateBast(c *fiber.Ctx) error {
	req := new(GenerateBastRequest)
	if err := c.BodyParser(req); err != nil {
		return respondError(c, err, false)
	}
	if msgs := validation.ValidationRequest(req); msgs != nil {
		return respondError(c, errString(strings.Join(msgs, ", ")), false)
	}

	bast, err := h.service.GenerateBast(c.Params("id"), *req, actorID(c))
	if err != nil {
		return respondError(c, err, false)
	}
	return helpers.ResponseUtils(c, fiber.StatusCreated, true, "BAST dibuat", bast)
}
