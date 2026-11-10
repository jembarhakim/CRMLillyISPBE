package monitoring

import (
	"skripsi-be/internal/config/database"
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func RouteCustomerMonitoring(app *fiber.App) {
	// Initialize dependencies
	db := database.GetDB()
	service := NewService(db)
	handler := NewHandler(service)

	customerMonitoring := app.Group("/api/customer/monitoring")

	// Public health probe (no auth: used by uptime monitoring).
	customerMonitoring.Get("/health", func(c *fiber.Ctx) error {
		return helpers.ResponseUtils(c, 200, true, "Customer monitoring service is healthy", map[string]interface{}{
			"service": "customer-monitoring",
			"version": "1.0.0",
			"status":  "active",
		})
	})

	// Connection history for the authenticated customer.
	//
	// Must use the CUSTOMER token guard: the handler reads c.Locals("customer_id"),
	// which only helpers.CustomerVerifyToken sets. Using AdminAuthMiddleware here
	// made every customer request fail with 401 and force a logout in the portal.
	customerMonitoring.Get("/history", helpers.CustomerVerifyToken, handler.GetConnectionHistory)

	// WebSocket endpoint for real-time monitoring. Browsers cannot attach an
	// Authorization header to a WS handshake, so this stays unauthenticated.
	customerMonitoring.Get("/ws", websocket.New(WebSocketHandler))
}
