package dashboard

import (
	"skripsi-be/internal/config/database"
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
)

func CustomerDashboardRoute(app fiber.Router) {

	db := database.GetDB()
	repository := NewCustomerDashboardRepository(db)
	service := NewCustomerDashboardService(repository)
	handler := NewCustomerDashboardHandler(service)

	// Route
	app.Get("/", helpers.CustomerVerifyToken, handler.MyUserCustomerDashboard)
	app.Post("/", helpers.CustomerVerifyToken, handler.CreatePaymentCustomerDashboard)
	app.Get("/device-status", helpers.CustomerVerifyToken, handler.CheckDeviceStatus)
	app.Get("/products", helpers.CustomerVerifyToken, handler.GetAvailableProducts)
	app.Get("/test", handler.TestEndpoint)

	// Billing: monthly bill table, Midtrans Snap token and PDF source data.
	app.Get("/bills", helpers.CustomerVerifyToken, handler.MyBills)
	app.Post("/payment/snap", helpers.CustomerVerifyToken, handler.CreateSnapToken)
	app.Get("/invoice/:id", helpers.CustomerVerifyToken, handler.MyInvoiceDetail)
}
