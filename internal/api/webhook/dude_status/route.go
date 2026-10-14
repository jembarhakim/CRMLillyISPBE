package dude_status

import (
	"skripsi-be/internal/helpers"

	"gorm.io/gorm"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func Routes(app fiber.Router, db *gorm.DB) {
	hub := NewHub()
	repository := NewStatusRepository(db)
	service := NewService(repository, hub)
	handler := NewHandler(service)

	app.Post("/", handler.ReceiveStatus)
	app.Get("/ws", websocket.New(hub.HandleConnection, websocket.Config{
		Origins: []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"https://menara.net.id",
			"http://rndpolije.lilly.net.id",
			"https://rndpolije.lilly.net.id",
		},
	}))
}

func AdminRoutes(app fiber.Router, db *gorm.DB) {
	repository := NewStatusRepository(db)
	service := NewService(repository, NewHub())
	handler := NewHandler(service)

	app.Get("/", helpers.VerifyToken, helpers.RequireRoles("ADMIN", "CUSTOMER_SERVICE"), handler.ListDevices)
	app.Put("/:id", helpers.VerifyToken, helpers.RequireRoles("ADMIN", "CUSTOMER_SERVICE"), handler.UpdateDeviceDetails)
}
