package routes

import (
	"log"
	"skripsi-be/internal/api/admin/account"
	"skripsi-be/internal/api/admin/area"
	"skripsi-be/internal/api/admin/asset"
	assetitem "skripsi-be/internal/api/admin/asset_item"
	assettransaction "skripsi-be/internal/api/admin/asset_transaction"
	"skripsi-be/internal/api/admin/company"
	"skripsi-be/internal/api/admin/customer"
	customerinstallation "skripsi-be/internal/api/admin/customer/installation"
	"skripsi-be/internal/api/admin/dashboard"
	"skripsi-be/internal/api/admin/feature"
	"skripsi-be/internal/api/admin/geocoding"
	"skripsi-be/internal/api/admin/inventory"
	"skripsi-be/internal/api/admin/invoice"
	itemscatalog "skripsi-be/internal/api/admin/items_catalog"
	itemstransaction "skripsi-be/internal/api/admin/items_transaction"
	"skripsi-be/internal/api/admin/mikrotik"
	networkdevice "skripsi-be/internal/api/admin/network-device"
	network_monitoring "skripsi-be/internal/api/admin/network_monitoring"
	"skripsi-be/internal/api/admin/product"
	"skripsi-be/internal/api/admin/recurring_invoice"
	"skripsi-be/internal/api/admin/report"
	"skripsi-be/internal/api/admin/role"
	"skripsi-be/internal/api/admin/transaction"
	usermanagement "skripsi-be/internal/api/admin/user-management"
	authapi "skripsi-be/internal/api/auth"
	broadcastapi "skripsi-be/internal/api/broadcast"
	upload_file "skripsi-be/internal/api/common/upload_file"
	customerdashboard "skripsi-be/internal/api/customer/dashboard"
	"skripsi-be/internal/api/customer/monitoring"
	newsapi "skripsi-be/internal/api/news"
	telegramapi "skripsi-be/internal/api/telegram"
	ticketapi "skripsi-be/internal/api/ticket"
	midtrans "skripsi-be/internal/api/webhook/midtrans"
	"skripsi-be/internal/api/webhook/moota"
	waapi "skripsi-be/internal/api/webhook/wa"
	"skripsi-be/internal/config/database"
	"skripsi-be/internal/services"

	"github.com/gofiber/fiber/v2"
)

func RouteFiber(app *fiber.App) {
	app.Static("/", "./public")
	app.Static("/uploads", "./uploads")

	// Add a test endpoint to verify server is working
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Server is running",
			"status":  "ok",
		})
	})

	api := app.Group("/api")

	upload_file.CommonUploadFileRoute(api.Group("/file-upload"))
	product.PublicProductRoute(api.Group("/product"))

	auth := api.Group("/auth")
	authapi.AuthRoute(auth)
	newsapi.PublicRoutes(api.Group("/news"))

	admin := api.Group("/admin")
	dashboard.AdminDashboardRoute(admin.Group("/dashboard"))
	company.AdminCompanyRoute(admin.Group("/company"))
	account.AdminAccountRoute(admin.Group("/account"))
	usermanagement.AdminUserManagementRoute(admin.Group("/user-management"))
	role.AdminRoleRoute(admin.Group("/role"))
	feature.AdminFeatureRoute(admin.Group("/feature"))
	product.AdminProductRoute(admin.Group("/product"))
	newsapi.AdminRoutes(admin.Group("/news"))
	report.AdminReportRoute(admin.Group("/report"))
	area.AdminAreaRoute(admin.Group("/area"))
	customer.AdminCustomerRoute(admin.Group("/customer"))
	customerinstallation.AdminCustomerInstallationRoute(admin.Group("/customer-installation"))
	asset.AdminAssetRoute(admin.Group("/asset"))
	assetitem.AssetItemRoute(admin.Group("/asset-item"))
	assettransaction.AssetTransactionRoute(admin.Group("/asset-transaction"))
	itemstransaction.ItemsTransactionRoute(admin.Group("/items-transaction"))
	itemscatalog.ItemsCatalogRoute(admin.Group("/items-catalog"))
	transaction.AdminTrasactionRoute(admin.Group("/transaction"))
	invoice.AdminInvoiceRoute(admin.Group("/invoice"))
	recurring_invoice.AdminRecurringInvoiceRoute(admin.Group("/recurring-invoice"))
	mikrotik.MikroTikRoutes(admin.Group("/mikrotik"))
	geocoding.AdminGeocodingRoute(admin.Group("/geocoding"))
	inventory.InventoryRoute(admin)

	// Network Monitoring routes
	networkMonitoringHandler := network_monitoring.NewNetworkMonitoringHandler(
		services.NewNetworkMonitoringAssistant(database.GetDB(), services.GetSharedMikroTikService()),
	)
	// Update the handler in the route registration
	admin.Group("/network-monitoring").Use(func(c *fiber.Ctx) error {
		c.Locals("networkMonitoringHandler", networkMonitoringHandler)
		return c.Next()
	})
	network_monitoring.NetworkMonitoringRoutes(admin.Group("/network-monitoring"))

	// Network Device routes
	networkdeviceHandler := networkdevice.NewAdminNetworkDeviceHandler(
		networkdevice.NewAdminNetworkDeviceService(
			networkdevice.NewAdminNetworkDeviceRepository(database.GetDB()),
		),
	)
	networkdevice.AdminNetworkDeviceRoutes(admin, networkdeviceHandler)

	customer := api.Group("/customer")
	customerdashboard.CustomerDashboardRoute(customer.Group("/dashboard"))

	// Customer monitoring routes
	monitoring.RouteCustomerMonitoring(app)

	// tickets module (shared access beneath /api)
	log.Println("Registering ticket routes...")
	ticketapi.TicketRoutes(api)
	log.Println("Ticket routes registered successfully")

	// telegram module for testing notifications
	log.Println("Registering telegram routes...")
	telegramapi.TelegramRoutes(api)
	log.Println("Telegram routes registered successfully")

	webhook := api.Group("/webhook")
	moota.WebhookMootaRoute(webhook.Group("/moota"))

	webhookPlain := app.Group("/webhook")
	midtrans.WebhookMidtransRoute(webhookPlain.Group("/midtrans"))

	// WhatsApp routes
	waapi.WARoutes(app)

	// Broadcast routes
	log.Println("Registering broadcast routes...")
	broadcastapi.SetupBroadcastRoutes(app, database.GetDB())
	log.Println("Broadcast routes registered successfully")

}
