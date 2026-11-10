package dude_status

import (
	"skripsi-be/internal/helpers"
	"skripsi-be/internal/services"
	"time"

	"gorm.io/gorm"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func RegisterRoutes(webhookApp, adminApp fiber.Router, db *gorm.DB) {
	hub := NewHub()
	repository := NewStatusRepository(db)
	service := NewService(repository, hub)
	handler := NewHandler(service)
	go service.StartRouterOSSyncLoop(time.Minute, services.GetSharedRouterOSAPI)

	webhookApp.Post("/", handler.ReceiveStatus)
	webhookApp.Get("/ws", websocket.New(hub.HandleConnection, websocket.Config{
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

	adminApp.Get("/", helpers.VerifyToken, helpers.RequireRoles("ADMIN", "CUSTOMER_SERVICE"), handler.ListDevices)
	adminApp.Post("/import/preview", helpers.VerifyToken, helpers.RequireRoles("ADMIN"), handler.PreviewCSV)
	adminApp.Post("/import", helpers.VerifyToken, helpers.RequireRoles("ADMIN"), handler.ImportCSV)
	adminApp.Put("/:id", helpers.VerifyToken, helpers.RequireRoles("ADMIN", "CUSTOMER_SERVICE"), handler.UpdateDeviceDetails)
	adminApp.Post("/ping-all", helpers.VerifyToken, helpers.RequireRoles("ADMIN"), handler.StartPingAll)
	adminApp.Get("/ping-all", helpers.VerifyToken, helpers.RequireRoles("ADMIN"), handler.PingAllStatus)
	adminApp.Post("/sync", helpers.VerifyToken, helpers.RequireRoles("ADMIN"), handler.StartRouterOSSync)
	adminApp.Get("/sync", helpers.VerifyToken, helpers.RequireRoles("ADMIN", "CUSTOMER_SERVICE"), handler.RouterOSSyncStatus)
}
