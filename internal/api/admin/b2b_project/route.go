package b2bproject

import (
	"skripsi-be/internal/config/database"
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
)

// AdminB2BProjectRoute registers the B2B incidental project module.
//
// Access: the project manager roles (SUPERADMIN, CUSTOMER_SERVICE) manage the
// project itself, while FINANCE also needs the milestone/expense endpoints.
func AdminB2BProjectRoute(app fiber.Router) {
	db := database.GetDB()
	repository := NewRepository(db)
	service := NewService(repository)
	handler := NewHandler(service)

	app.Use(helpers.VerifyToken)

	manage := helpers.RequireRoles("ADMIN", "CUSTOMER_SERVICE")
	finance := helpers.RequireRoles("ADMIN", "CUSTOMER_SERVICE", "FINANCE")

	// --- Project master ---
	app.Get("/", finance, handler.List)
	app.Post("/", manage, handler.Create)
	app.Get("/:id", finance, handler.Detail)
	app.Put("/:id", manage, handler.Update)
	app.Delete("/:id", manage, handler.Delete)
	app.Post("/:id/status", manage, handler.ChangeStatus)

	// --- Quotation (RAB) ---
	app.Get("/:id/quotations", finance, handler.ListQuotations)
	app.Post("/:id/quotations", manage, handler.CreateQuotation)
	app.Get("/quotation/:quotationId", finance, handler.GetQuotation)
	app.Put("/quotation/:quotationId/status", manage, handler.SetQuotationStatus)
	app.Delete("/quotation/:quotationId", manage, handler.DeleteQuotation)

	// --- Milestone invoicing ---
	app.Get("/:id/milestones", finance, handler.ListMilestones)
	app.Post("/:id/milestones", finance, handler.GenerateMilestones)
	app.Put("/milestone/:milestoneId", finance, handler.UpdateMilestone)
	app.Post("/milestone/:milestoneId/issue", finance, handler.IssueMilestone)
	app.Post("/milestone/:milestoneId/pay", finance, handler.PayMilestone)

	// --- Expense & margin ---
	app.Get("/:id/expenses", finance, handler.ListExpenses)
	app.Post("/:id/expenses", finance, handler.CreateExpense)
	app.Delete("/expense/:expenseId", finance, handler.DeleteExpense)
	app.Get("/:id/margin", finance, handler.Margin)

	// --- Work order & BAST ---
	app.Get("/:id/work-orders", finance, handler.ListWorkOrders)
	app.Post("/:id/work-orders", manage, handler.CreateWorkOrder)
	app.Put("/work-order/:woId", manage, handler.UpdateWorkOrder)
	app.Get("/:id/basts", finance, handler.ListBasts)
	app.Post("/:id/basts", manage, handler.GenerateBast)
}
