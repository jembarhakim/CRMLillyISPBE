package items_catalog

import (
	"skripsi-be/internal/config/database"
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
)

func ItemsCatalogRoute(app fiber.Router) {
	db := database.GetDB()
	repository := NewItemRepository(db)
	service := NewItemService(repository)
	handler := NewItemHandler(service)

	app.Use(helpers.VerifyToken)
	app.Get("/", handler.GetAllItemsHandler)
	app.Get("/:id", handler.GetByIdItemHandler)
	app.Post("/", handler.CreateItemHandler)
	app.Put("/:id", handler.UpdateItemHandler)
	app.Delete("/:id", handler.DeleteItemHandler)
}

