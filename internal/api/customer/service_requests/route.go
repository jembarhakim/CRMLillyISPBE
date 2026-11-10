package servicerequests

import (
	"skripsi-be/internal/config/database"
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
)

// CustomerServiceRequestRoute registers the customer side of the
// upgrade / downgrade / relocation workflow.
func CustomerServiceRequestRoute(app fiber.Router) {
	db := database.GetDB()
	repository := NewServiceRequestRepository(db)
	service := NewServiceRequestService(repository)
	handler := NewServiceRequestHandler(service)

	app.Get("/", helpers.CustomerVerifyToken, handler.Page)
	app.Post("/", helpers.CustomerVerifyToken, handler.Create)
	app.Get("/history", helpers.CustomerVerifyToken, handler.History)
}

// AdminServiceRequestRoute registers the CRM approval queue.
func AdminServiceRequestRoute(app fiber.Router) {
	db := database.GetDB()
	repository := NewServiceRequestRepository(db)
	service := NewServiceRequestService(repository)
	handler := NewAdminServiceRequestHandler(service)

	app.Get("/", helpers.VerifyToken, helpers.RequireRoles("ADMIN", "CUSTOMER_SERVICE"), handler.Pending)
	app.Post("/:id/review", helpers.VerifyToken, helpers.RequireRoles("ADMIN", "CUSTOMER_SERVICE"), handler.Review)
}
