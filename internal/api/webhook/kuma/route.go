package kuma

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// RegisterRoutes registers the Uptime Kuma webhook endpoints
func RegisterRoutes(router fiber.Router, db *gorm.DB) {
	repo := NewRepository(db)
	service := NewService(repo)
	handler := NewHandler(service)

	router.Post("/", handler.HandleWebhook)
	router.Get("/logs", handler.GetLogs)
}
