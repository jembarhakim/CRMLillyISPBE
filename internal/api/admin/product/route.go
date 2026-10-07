package product

import (
	"skripsi-be/internal/config/database"
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
)

func AdminProductRoute(app fiber.Router) {

	db := database.GetDB()
	repository := NewAdminProductRepository(db)
	service := NewAdminProductServiceStruct(repository)
	handler := NewAdminProductHandlerStruct(service)

	app.Use(helpers.VerifyToken)
	app.Get("", handler.GetAllAdminProductHandler)
	app.Get("/:id", handler.GetByIdAdminProductHandler)
	app.Post("", handler.CreateAdminProductHandler)
	app.Put("/:id", handler.UpdateAdminProductHandler)
	app.Delete("/:id", handler.DeleteAdminProductHandler)

}

func PublicProductRoute(app fiber.Router) {
	db := database.GetDB()
	repository := NewAdminProductRepository(db)
	service := NewAdminProductServiceStruct(repository)
	handler := NewAdminProductHandlerStruct(service)

	app.Get("", handler.GetAllAdminProductHandler)
	app.Get("/:id", handler.GetByIdAdminProductHandler)
}
