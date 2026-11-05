package notifications

import (
	"fmt"
	"log"

	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
)

func QueueTicketCreated(db *gorm.DB, ticket *entities.TroubleTicket) {
	if ticket == nil || ticket.CustomerID == "" {
		return
	}
	var customer entities.Customer
	if err := db.First(&customer, "id = ?", ticket.CustomerID).Error; err != nil {
		log.Printf("[notifications] unable to load ticket customer %s: %v", ticket.CustomerID, err)
		return
	}
	vars := map[string]any{
		"nama_pelanggan": customer.Name,
		"no_tiket":       ticket.ID,
		"ringkasan":      ticket.Title,
	}
	deduplicationKey := fmt.Sprintf("ticket:%d:created", ticket.ID)
	logEnqueue(NewService(db).EnqueueCustomer(ticket.CustomerID, "ticket_created", vars, deduplicationKey, ""), fmt.Sprintf("ticket %d created", ticket.ID))
}

func QueueTicketInProgress(db *gorm.DB, ticket *entities.TroubleTicket) {
	if ticket == nil || ticket.CustomerID == "" {
		return
	}
	var customer entities.Customer
	if err := db.First(&customer, "id = ?", ticket.CustomerID).Error; err != nil {
		log.Printf("[notifications] unable to load ticket customer %s: %v", ticket.CustomerID, err)
		return
	}
	vars := map[string]any{"nama_pelanggan": customer.Name, "no_tiket": ticket.ID}
	deduplicationKey := fmt.Sprintf("ticket:%d:in-progress", ticket.ID)
	logEnqueue(NewService(db).EnqueueCustomer(ticket.CustomerID, "ticket_in_progress", vars, deduplicationKey, ""), fmt.Sprintf("ticket %d progress", ticket.ID))
}

func QueueTicketAccepted(db *gorm.DB, ticket *entities.TroubleTicket, technicianID string) {
	if ticket == nil {
		return
	}
	var customer entities.Customer
	if err := db.First(&customer, "id = ?", ticket.CustomerID).Error; err != nil {
		log.Printf("[notifications] unable to load ticket customer %s: %v", ticket.CustomerID, err)
		return
	}
	var technician entities.User
	if err := db.First(&technician, "id = ?", technicianID).Error; err != nil {
		log.Printf("[notifications] unable to load technician %s: %v", technicianID, err)
		return
	}
	mapsLink := ""
	if customer.Latitude != 0 || customer.Longitude != 0 {
		mapsLink = fmt.Sprintf("https://maps.google.com/?q=%f,%f", customer.Latitude, customer.Longitude)
	}
	vars := map[string]any{
		"nama_pelanggan":   customer.Name,
		"nama_teknisi":     technician.Name,
		"no_tiket":         ticket.ID,
		"alamat":           customer.Address,
		"telepon":          customer.Phone,
		"link_maps":        mapsLink,
		"jadwal_kunjungan": "segera dikonfirmasi oleh teknisi",
	}
	service := NewService(db)
	technicianDeduplicationKey := fmt.Sprintf("ticket:%d:assigned:%s", ticket.ID, technicianID)
	customerDeduplicationKey := fmt.Sprintf("ticket:%d:scheduled", ticket.ID)
	logEnqueue(service.Enqueue(technician.Phone, "work_order_assigned", vars, technicianDeduplicationKey, ""), fmt.Sprintf("ticket %d technician assignment", ticket.ID))
	logEnqueue(service.EnqueueCustomer(ticket.CustomerID, "work_order_scheduled", vars, customerDeduplicationKey, ""), fmt.Sprintf("ticket %d schedule confirmation", ticket.ID))
}

func QueueTicketCompleted(db *gorm.DB, ticket *entities.TroubleTicket, templateKey string) {
	if ticket == nil || ticket.CustomerID == "" {
		return
	}
	var customer entities.Customer
	if err := db.First(&customer, "id = ?", ticket.CustomerID).Error; err != nil {
		log.Printf("[notifications] unable to load ticket customer %s: %v", ticket.CustomerID, err)
		return
	}
	vars := map[string]any{"nama_pelanggan": customer.Name, "no_tiket": ticket.ID}
	if templateKey == "ticket_resolved" {
		vars["link_evaluasi"] = ""
	}
	deduplicationKey := fmt.Sprintf("ticket:%d:%s", ticket.ID, templateKey)
	logEnqueue(NewService(db).EnqueueCustomer(ticket.CustomerID, templateKey, vars, deduplicationKey, ""), fmt.Sprintf("ticket %d completed", ticket.ID))
}

func logEnqueue(err error, context string) {
	if err != nil {
		log.Printf("[notifications] %s: %v", context, err)
	}
}
