package field_payment

import (
	"skripsi-be/internal/config/database"
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
)

func FieldPaymentRoutes(app fiber.Router) {
	handler := NewFieldPaymentHandler(database.GetDB())
	app.Use(helpers.VerifyToken)

	app.Get("/invoices", helpers.RequireRoles("TECHNICIAN"), handler.GetOpenInvoices)
	app.Post("/payments", helpers.RequireRoles("TECHNICIAN"), handler.CreateFieldPayment)
	app.Get("/payments", helpers.RequireRoles("ADMIN", "FINANCE"), handler.GetPendingPayments)
	app.Post("/closings", helpers.RequireRoles("ADMIN", "FINANCE"), handler.CloseCollectorPayments)
}
