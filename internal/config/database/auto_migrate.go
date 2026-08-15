package database

import (
	"log"

	"skripsi-be/internal/models/entities"
)

func AutoMigrate() {
	db := GetDB()

	err := db.AutoMigrate(
		// Master
		&entities.Accounts{},
		&entities.Company{},
		&entities.Role{},
		&entities.Feature{},
		&entities.RolePermission{},
		&entities.User{},

		// Customer
		&entities.Customer{},
		&entities.CustomerInstallation{},
		&entities.CustomerService{},

		// Invoice
		&entities.Invoice{},
		&entities.InvoiceItems{},
		&entities.InvoicePendingReason{},

		// Device
		&entities.Device{},
		&entities.NetworkDevice{},
		&entities.Cable{},

		// Ticket
		&entities.TroubleTicket{},
		&entities.TicketStep{},
		&entities.TicketStepImage{},
		&entities.TechnicianStep{},
		&entities.TechnicianTeamMember{},
		&entities.TicketClassification{},
		&entities.TroubleTypeRow{},

		// Broadcast
		&entities.BroadcastHistory{},

		// Inventory
		&entities.Asset{},
		&entities.AssetItem{},
		&entities.AssetTransaction{},
		&entities.Item{},
		&entities.ItemsMasuk{},
		&entities.ItemsKeluar{},
		&entities.DetailItemsMasuk{},
		&entities.DetailItemsKeluar{},
		&entities.BarangMasuk{},
		&entities.DetailBarangMasuk{},
		&entities.TicketAssetTransaction{},

		// Recurring Invoice
		&entities.RecurringInvoice{},
		&entities.RecurringInvoiceItem{},
		&entities.RecurringInvoiceHistory{},

		// Netwatch
		&entities.NetwatchDevice{},
		&entities.NetwatchEvent{},
		&entities.NetwatchConfig{},
		&entities.TicketLog{},

		// Router
		&entities.RouterJob{},

		// Report
		&entities.InstallationReportTechnician{},
		&entities.InstallationProvisioningLog{},
		&entities.InstallationHistory{},

		// Others
		&entities.Areas{},
		&entities.Products{},
		&entities.Log{},
		&entities.Image{},
		&entities.Transaction{},
		&entities.Transfers{},
		&entities.ReportAssets{},
		&entities.ReportCash{},
	)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("AutoMigrate selesai")
}