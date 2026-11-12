package networktopology

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterRoutes(admin fiber.Router, db *gorm.DB) {
	repo := NewRepository(db)
	service := NewService(repo)
	handler := NewHandler(service)

	admin.Get("/network-nodes", handler.List)
	admin.Get("/network-nodes/tree", handler.Tree)
	admin.Post("/network-nodes", handler.Create)
	admin.Put("/network-nodes/:id", handler.Update)
	admin.Delete("/network-nodes/:id", handler.Delete)
	admin.Get("/odp/:id/ports", handler.Ports)
	RegisterCableRouteHandlers(admin, db)
}
