package asset_item

import (
	"skripsi-be/internal/config/database"
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
)

func AssetItemRoute(app fiber.Router) {
	db := database.GetDB()
	repository := NewAssetItemRepository(db)
	service := NewAssetItemService(repository)
	handler := NewAssetItemHandler(service)

	app.Use(helpers.VerifyToken)
	app.Get("/", handler.GetAllAssetItemHandler)
	app.Get("/:id", handler.GetByIdAssetItemHandler)
	app.Post("/", handler.CreateAssetItemHandler)
	app.Put("/:id", handler.UpdateAssetItemHandler)
	app.Delete("/:id", handler.DeleteAssetItemHandler)
}

