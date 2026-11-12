package ont

import (
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
)

// RegisterOntRoutes registers the ONT monitoring endpoints
func RegisterOntRoutes(app *fiber.App, handler *Handler) {
	api := app.Group("/api", helpers.VerifyToken)

	// Specification exact routes:
	// GET  /api/map/ont-status
	// GET  /api/customers/:id/ont
	// GET  /api/customers/:id/ont/history
	// POST /api/customers/:id/ont/refresh
	api.Get("/map/ont-status", handler.GetMapStatusHandler)
	api.Get("/customers/:id/ont", handler.GetCustomerOntHandler)
	api.Get("/customers/:id/ont/history", handler.GetCustomerOntHistoryHandler)
	api.Post("/customers/:id/ont/refresh", handler.RefreshCustomerOntHandler)

	// Admin namespace aliases for dashboard consistency:
	admin := api.Group("/admin")
	admin.Get("/ont/map-status", handler.GetMapStatusHandler)
	admin.Get("/customers/:id/ont", handler.GetCustomerOntHandler)
	admin.Get("/customers/:id/ont/history", handler.GetCustomerOntHistoryHandler)
	admin.Post("/customers/:id/ont/refresh", handler.RefreshCustomerOntHandler)
}

