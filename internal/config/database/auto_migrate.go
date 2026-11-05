package database

import (
	"log"

	"skripsi-be/internal/models/entities"
)

func AutoMigrate() {
	db := GetDB()
	if db == nil {
		log.Println("Database connection is nil; skipping AutoMigrate")
		return
	}

	models := []interface{}{
		// Master
		&entities.Accounts{},
		&entities.Company{},
		&entities.Role{},
		&entities.Feature{},
		&entities.RolePermission{},
		&entities.User{},
		&entities.News{},

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
		&entities.Notification{},
		&entities.NotificationLog{},
		&entities.NotificationTemplate{},
		&entities.NotificationPreference{},

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
		&entities.DudeDeviceStatus{},
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
	}

	for _, model := range models {
		if db.Migrator().HasTable(model) {
			log.Printf("Skipping schema migration for existing table: %T", model)
			continue
		}

		if err := db.Migrator().CreateTable(model); err != nil {
			log.Printf("Failed to create table for %T: %v", model, err)
		}
	}

	if err := db.AutoMigrate(&entities.DudeDeviceStatus{}); err != nil {
		log.Printf("Failed to migrate The Dude device status table: %v", err)
	}

	// Additive column migrations for existing tables.
	// The loop above only creates missing tables, so new columns on pre-existing
	// tables must be added here (Migrator().AddColumn is a no-op if it exists).
	if db.Migrator().HasTable(&entities.Item{}) {
		if !db.Migrator().HasColumn(&entities.Item{}, "MinimumStock") {
			if err := db.Migrator().AddColumn(&entities.Item{}, "MinimumStock"); err != nil {
				log.Printf("Failed to add minimum_stock column to items table: %v", err)
			}
		}
	}

	log.Println("Schema verification complete")
}
