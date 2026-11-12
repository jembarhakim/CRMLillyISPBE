package incidents

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterRoutes(admin fiber.Router, db *gorm.DB) {
	service := NewService(db)
	handler := NewHandler(service)
	admin.Get("/incidents", handler.List)
	admin.Post("/incidents", handler.Create)
	admin.Put("/incidents/:id", handler.Update)
	admin.Put("/incidents/:id/resolve", handler.Resolve)
}
