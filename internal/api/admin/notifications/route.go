package notificationsapi

import (
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterAdminRoutes(router fiber.Router, db *gorm.DB) {
	handler := NewHandler(db)
	router.Use(helpers.VerifyToken, helpers.RequireRoles("ADMIN", "FINANCE", "CUSTOMER_SERVICE"))

	router.Get("/summary", handler.Summary)
	router.Get("/whatsapp/connection", handler.WhatsAppConnection)
	router.Get("/templates", handler.Templates)
	router.Post("/templates/preview", handler.PreviewTemplate)
	router.Post("/templates", handler.SaveTemplate)
	router.Put("/templates/:key", handler.SaveTemplate)
	router.Delete("/templates/:key", handler.DeleteTemplate)
	router.Get("/queue", handler.ListNotifications)
	router.Get("/logs", handler.ListLogs)
	router.Post("/queue", handler.Enqueue)
	router.Get("/preferences", handler.ListPreferences)
	router.Put("/preferences/:customerId", handler.SetPreference)
}

func RegisterWebhookRoutes(router fiber.Router, db *gorm.DB) {
	router.Get("/status", MetaWebhookVerification)
	router.Post("/status", DeliveryWebhook(db))
}
