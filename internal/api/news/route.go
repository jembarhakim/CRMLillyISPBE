package news

import (
	"skripsi-be/internal/config/database"
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
)

func PublicRoutes(app fiber.Router) {
	handler := NewHandler(NewRepository(database.GetDB()))
	app.Get("/", handler.ListPublic)
	app.Get("/:id", handler.GetPublic)
}

func AdminRoutes(app fiber.Router) {
	handler := NewHandler(NewRepository(database.GetDB()))
	app.Use(helpers.VerifyToken)
	app.Get("/", handler.ListAdmin)
	app.Get("/:id", handler.GetAdmin)
	app.Post("/", handler.Create)
	app.Put("/:id", handler.Update)
	app.Delete("/:id", handler.Delete)
}
