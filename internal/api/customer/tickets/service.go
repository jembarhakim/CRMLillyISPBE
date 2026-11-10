package tickets

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"skripsi-be/internal/models/entities"
)

type TicketServiceInterface interface {
	Create(customerID string, request CreateCustomerTicketRequest) (*CustomerTicketItem, error)
	List(customerID string) ([]CustomerTicketItem, error)
	Detail(customerID string, ticketID uint64) (*CustomerTicketDetail, error)
	TroubleTypes() ([]entities.TroubleTypeRow, error)
}

type TicketServiceStruct struct {
	repository TicketRepositoryInterface
}

func NewTicketService(repository TicketRepositoryInterface) *TicketServiceStruct {
	return &TicketServiceStruct{repository}
}

// ---------------------------------------------------------------------------
// Status mapping
//
// The staff workflow stores technical statuses ("unfinished", "ongoing",
// "resolved", ...). Customers are shown four friendly stages instead:
//   received  -> Tiket Diterima
//   scheduled -> Teknisi Dijadwalkan
//   handling  -> Dalam Penanganan
//   done      -> Selesai
// ---------------------------------------------------------------------------

const (
	stageReceived  = "received"
	stageScheduled = "scheduled"
	stageHandling  = "handling"
	stageDone      = "done"
)

var stageLabels = map[string]string{
	stageReceived:  "Tiket Diterima",
	stageScheduled: "Teknisi Dijadwalkan",
	stageHandling:  "Dalam Penanganan",
	stageDone:      "Selesai",
}

var stageOrder = []string{stageReceived, stageScheduled, stageHandling, stageDone}

// resolveStage maps a ticket row to a customer facing stage.
func resolveStage(ticket entities.TroubleTicket) string {
	status := strings.ToLower(strings.TrimSpace(ticket.Status))
	hasTechnician := ticket.AssignedTo != nil && *ticket.AssignedTo != ""
	technicianCompleted := ticket.TechnicianCompleted != nil && *ticket.TechnicianCompleted

	switch status {
	case "resolved", "closed", "done", "completed", "finished":
		return stageDone
	case "on_progress", "in_progress", "handling", "working", "processing":
		return stageHandling
	case "scheduled", "assigned", "dispatched":
		return stageScheduled
	case "ongoing":
		// "ongoing" is used for both NOC review and technician work; a finished
		// technician job counts as done.
		if technicianCompleted {
			return stageDone
		}
		return stageHandling
	case "unfinished", "new", "received", "":
		// Still in the CS queue but already dispatched to a technician.
		if hasTechnician {
			return stageScheduled
		}
		return stageReceived
	}

	if technicianCompleted {
		return stageDone
	}
	return stageReceived
}

func stageStep(stage string) int {
	for i, s := range stageOrder {
		if s == stage {
			return i + 1
		}
	}
	return 1
}

// toCustomerTicketItem converts a ticket row into the portal representation.
func (s *TicketServiceStruct) toCustomerTicketItem(ticket entities.TroubleTicket) CustomerTicketItem {
	stage := resolveStage(ticket)

	troubleTypeName := ""
	if ticket.Type != nil && *ticket.Type != "" {
		if name, err := s.repository.TroubleTypeName(*ticket.Type); err == nil {
			troubleTypeName = name
		}
	}
	// Fall back to the raw type value (the ML classifier also stores names).
	if troubleTypeName == "" && ticket.Type != nil {
		troubleTypeName = *ticket.Type
	}

	item := CustomerTicketItem{
		ID:              ticket.ID,
		TicketNumber:    fmt.Sprintf("TKT-%05d", ticket.ID),
		Title:           ticket.Title,
		TroubleTypeName: troubleTypeName,
		Status:          ticket.Status,
		Stage:           stage,
		StageLabel:      stageLabels[stage],
		StageStep:       stageStep(stage),
		CreatedAt:       nowOrEmpty(ticket.CreatedAt),
		UpdatedAt:       nowOrEmpty(ticket.UpdatedAt),
	}
	if ticket.Description != nil {
		item.Description = *ticket.Description
	}
	if ticket.ImgCS != nil && *ticket.ImgCS != "" {
		item.ImageURL = "/uploads/cs-images/" + *ticket.ImgCS
	}
	if ticket.TechnicianNote != nil {
		item.TechnicianNote = *ticket.TechnicianNote
	}
	return item
}

// buildTimeline produces the four step tracker for the detail view.
func buildTimeline(ticket entities.TroubleTicket, item CustomerTicketItem) []CustomerTicketTimelineEntry {
	createdAt := nowOrEmpty(ticket.CreatedAt)
	updatedAt := nowOrEmpty(ticket.UpdatedAt)

	timeline := make([]CustomerTicketTimelineEntry, 0, len(stageOrder))
	for i, stage := range stageOrder {
		step := i + 1
		entry := CustomerTicketTimelineEntry{
			Stage:      stage,
			StageLabel: stageLabels[stage],
			Reached:    item.StageStep >= step,
		}
		if step == 1 && entry.Reached {
			entry.At = createdAt
			entry.Note = "Keluhan Anda sudah kami terima."
		}
		if step == len(stageOrder) && entry.Reached {
			entry.At = updatedAt
			if ticket.TechnicianNote != nil {
				entry.Note = *ticket.TechnicianNote
			}
		}
		timeline = append(timeline, entry)
	}
	return timeline
}

// Create registers a new complaint for the customer. The ticket enters the
// CUSTOMER SERVICE queue exactly like a ticket created by a staff member.
func (s *TicketServiceStruct) Create(customerID string, request CreateCustomerTicketRequest) (*CustomerTicketItem, error) {
	if strings.TrimSpace(customerID) == "" {
		return nil, fmt.Errorf("customer tidak dikenali")
	}

	title := strings.TrimSpace(request.Title)
	description := strings.TrimSpace(request.Description)
	if len([]rune(title)) < 5 {
		return nil, fmt.Errorf("judul keluhan minimal 5 karakter")
	}
	if len([]rune(description)) < 10 {
		return nil, fmt.Errorf("deskripsi keluhan minimal 10 karakter")
	}

	csRoleID, err := s.repository.RoleIDByName(string(entities.AssignCS))
	if err != nil {
		return nil, err
	}

	ticket := entities.TroubleTicket{
		CustomerID:       customerID,
		Title:            title,
		Description:      &description,
		Status:           "unfinished",
		CurrentAssignee:  csRoleID,
		ClassificationID: entities.ClassificationGangguan,
	}

	if typeID := strings.TrimSpace(request.TroubleTypeID); typeID != "" {
		ticket.Type = &typeID
	}
	if filename := strings.TrimSpace(request.ImageFilename); filename != "" {
		ticket.ImgCS = &filename
	}

	if err := s.repository.Create(&ticket); err != nil {
		return nil, err
	}

	log.Printf("[customer-ticket] ticket #%d created for customer %s", ticket.ID, customerID)

	item := s.toCustomerTicketItem(ticket)
	return &item, nil
}

// List returns every complaint that belongs to the customer.
func (s *TicketServiceStruct) List(customerID string) ([]CustomerTicketItem, error) {
	tickets, err := s.repository.ListByCustomer(customerID)
	if err != nil {
		return nil, err
	}

	items := make([]CustomerTicketItem, 0, len(tickets))
	for _, ticket := range tickets {
		items = append(items, s.toCustomerTicketItem(ticket))
	}
	return items, nil
}

// Detail returns one complaint together with its tracking timeline.
func (s *TicketServiceStruct) Detail(customerID string, ticketID uint64) (*CustomerTicketDetail, error) {
	ticket, err := s.repository.ByIDForCustomer(customerID, ticketID)
	if err != nil {
		return nil, fmt.Errorf("tiket tidak ditemukan")
	}

	item := s.toCustomerTicketItem(*ticket)
	detail := CustomerTicketDetail{
		CustomerTicketItem: item,
		Timeline:           buildTimeline(*ticket, item),
	}
	return &detail, nil
}

func (s *TicketServiceStruct) TroubleTypes() ([]entities.TroubleTypeRow, error) {
	return s.repository.TroubleTypes()
}

// ParseTicketID validates the :id path parameter.
func ParseTicketID(raw string) (uint64, error) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("id tiket tidak valid")
	}
	return id, nil
}
