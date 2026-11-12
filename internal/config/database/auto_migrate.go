package database

import (
	"log"
	"strings"

	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
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
		&entities.CustomerServiceRequest{},

		// Invoice
		&entities.Invoice{},
		&entities.InvoiceItems{},
		&entities.InvoicePendingReason{},
		&entities.InvoicePayment{},
		&entities.PaymentAttempt{},
		&entities.FieldPaymentClosing{},
		&entities.CustomerIsolationLog{},

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

		// B2B Incidental Project (Sprint 6)
		&entities.B2BProject{},
		&entities.B2BProjectStatusLog{},
		&entities.B2BProjectMilestone{},
		&entities.B2BQuotation{},
		&entities.B2BQuotationItem{},
		&entities.B2BProjectExpense{},
		&entities.B2BProjectWorkOrder{},
		&entities.B2BProjectBast{},

		// Netwatch & Monitoring Logs
		&entities.NetwatchDevice{},
		&entities.NetwatchEvent{},
		&entities.NetwatchConfig{},
		&entities.DudeDeviceStatus{},
		&entities.TicketLog{},
		&entities.DeviceStatusLog{},

		// Router
		&entities.RouterJob{},
		&entities.NetworkRouterCredential{},

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

		// Network topology (customer mapping & monitoring)
		&entities.NetworkNode{},
		&entities.OdpPort{},
		&entities.NetworkLink{},
		&entities.CustomerNetworkLink{},
		&entities.NetworkCableRoute{},
		&entities.PhysicalOpticalReading{},
		&entities.LinkFlappingStatus{},
		&entities.MassIncident{},
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

	migrateInvoiceStatusColumn(db)

	if err := db.AutoMigrate(&entities.DudeDeviceStatus{}); err != nil {
		log.Printf("Failed to migrate The Dude device status table: %v", err)
	} else if err := db.Model(&entities.DudeDeviceStatus{}).
		Where("last_down_at IS NULL AND LOWER(status) IN ?", []string{"down", "offline"}).
		UpdateColumn("last_down_at", gorm.Expr("last_seen")).Error; err != nil {
		log.Printf("Failed to backfill last-down timestamps for The Dude devices: %v", err)
	}
	if err := db.AutoMigrate(
		&entities.NetworkNode{},
		&entities.OdpPort{},
		&entities.NetworkLink{},
		&entities.CustomerNetworkLink{},
		&entities.MassIncident{},
		&entities.TroubleTicket{},
	); err != nil {
		log.Printf("Failed to migrate network topology tables: %v", err)
	}

	// Additive column migrations for existing tables.
	if db.Migrator().HasTable(&entities.Item{}) {
		if !db.Migrator().HasColumn(&entities.Item{}, "MinimumStock") {
			if err := db.Migrator().AddColumn(&entities.Item{}, "MinimumStock"); err != nil {
				log.Printf("Failed to add minimum_stock column to items table: %v", err)
			}
		}
	}

	// Migrate ONT monitoring tables and columns
	if err := db.AutoMigrate(&entities.OntStatus{}, &entities.OntStatusHistory{}); err != nil {
		log.Printf("Failed to migrate ONT status tables: %v", err)
	}
	if !db.Migrator().HasColumn(&entities.Customer{}, "ont_serial_number") {
		_ = db.Migrator().AddColumn(&entities.Customer{}, "ont_serial_number")
	}
	if !db.Migrator().HasColumn(&entities.NetworkDevice{}, "ont_serial_number") {
		_ = db.Migrator().AddColumn(&entities.NetworkDevice{}, "ont_serial_number")
	}

	log.Println("Schema verification complete")
}

func migrateInvoiceStatusColumn(db *gorm.DB) {
	if !db.Migrator().HasTable(&entities.Invoice{}) {
		return
	}

	columnTypes, err := db.Migrator().ColumnTypes(&entities.Invoice{})
	if err != nil {
		log.Printf("Failed to inspect invoice status column: %v", err)
		return
	}

	for _, column := range columnTypes {
		if !strings.EqualFold(column.Name(), "status") {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(column.DatabaseTypeName()), "enum") {
			return
		}

		if err := db.Exec("ALTER TABLE `invoices` MODIFY COLUMN `status` VARCHAR(191) NOT NULL").Error; err != nil {
			log.Printf("Failed to migrate invoices.status to VARCHAR for partially paid invoices: %v", err)
			return
		}
		log.Println("Migrated invoices.status from ENUM to VARCHAR")
		return
	}
}
