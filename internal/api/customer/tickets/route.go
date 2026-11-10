package tickets

import (
	"skripsi-be/internal/config/database"
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
)

// CustomerTicketRoute registers the complaint portal endpoints.
// Every route is guarded by CustomerVerifyToken, which also rejects staff
// tokens, so a customer can only ever read their own tickets.
func CustomerTicketRoute(app fiber.Router) {
	db := database.GetDB()
	repository := NewTicketRepository(db)
	service := NewTicketService(repository)
	handler := NewTicketHandler(service)

	app.Get("/", helpers.CustomerVerifyToken, handler.List)
	app.Post("/", helpers.CustomerVerifyToken, handler.Create)
	app.Get("/trouble-types", helpers.CustomerVerifyToken, handler.TroubleTypes)
	app.Post("/upload", helpers.CustomerVerifyToken, handler.UploadAttachment)
	app.Get("/:id", helpers.CustomerVerifyToken, handler.Detail)
}
