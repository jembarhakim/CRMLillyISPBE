package tickets

import (
	"fmt"
	"strings"
	"time"

	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
)

type TicketRepositoryInterface interface {
	ListByCustomer(customerID string) ([]entities.TroubleTicket, error)
	ByIDForCustomer(customerID string, ticketID uint64) (*entities.TroubleTicket, error)
	Create(ticket *entities.TroubleTicket) error
	Save(ticket *entities.TroubleTicket) error
	TroubleTypes() ([]entities.TroubleTypeRow, error)
	TroubleTypeName(id string) (string, error)
	RoleIDByName(name string) (string, error)
	Steps(ticketID uint64) ([]entities.TicketStep, error)
}

type TicketRepositoryStruct struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) *TicketRepositoryStruct {
	return &TicketRepositoryStruct{db}
}

// ListByCustomer returns every ticket that belongs to the customer, newest first.
func (r *TicketRepositoryStruct) ListByCustomer(customerID string) ([]entities.TroubleTicket, error) {
	tickets := []entities.TroubleTicket{}
	err := r.db.
		Where("customer_id = ?", customerID).
		Order("created_at DESC").
		Find(&tickets).Error
	return tickets, err
}

// ByIDForCustomer fetches a single ticket only when it belongs to the customer.
// The customer_id predicate is the tenant boundary for the portal.
func (r *TicketRepositoryStruct) ByIDForCustomer(customerID string, ticketID uint64) (*entities.TroubleTicket, error) {
	ticket := &entities.TroubleTicket{}
	err := r.db.
		Where("customer_id = ? AND id = ?", customerID, ticketID).
		First(ticket).Error
	if err != nil {
		return nil, err
	}
	return ticket, nil
}

func (r *TicketRepositoryStruct) Create(ticket *entities.TroubleTicket) error {
	return r.db.Create(ticket).Error
}

func (r *TicketRepositoryStruct) Save(ticket *entities.TroubleTicket) error {
	return r.db.Save(ticket).Error
}

// TroubleTypes returns the lookup list used by the complaint form.
func (r *TicketRepositoryStruct) TroubleTypes() ([]entities.TroubleTypeRow, error) {
	rows := []entities.TroubleTypeRow{}
	err := r.db.Table("trouble_type").Select("id, name").Order("name ASC").Scan(&rows).Error
	return rows, err
}

// TroubleTypeName resolves a trouble type id to its display name.
func (r *TicketRepositoryStruct) TroubleTypeName(id string) (string, error) {
	var row entities.TroubleTypeRow
	err := r.db.Table("trouble_type").Select("id, name").Where("id = ?", id).First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", err
	}
	if row.Name == nil {
		return "", nil
	}
	return *row.Name, nil
}

// RoleIDByName mirrors ticketapi.Repo.RoleIDByName so tickets created from the
// portal land in the same CUSTOMER SERVICE queue as tickets created by staff.
// The actual DB may store a role as either "CUSTOMER_SERVICE" or "CUSTOMER SERVICE",
// so we try both variants before failing.
func (r *TicketRepositoryStruct) RoleIDByName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("role tidak boleh kosong")
	}

	candidates := []string{trimmed}
	seen := map[string]struct{}{strings.ToUpper(trimmed): {}}
	addCandidate := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		key := strings.ToUpper(candidate)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, candidate)
	}

	addCandidate(strings.ReplaceAll(trimmed, " ", "_"))
	addCandidate(strings.ReplaceAll(trimmed, "_", " "))
	addCandidate(strings.ReplaceAll(trimmed, " ", ""))
	addCandidate(strings.ReplaceAll(trimmed, "_", ""))

	for _, candidate := range candidates {
		var result struct {
			ID string
		}
		err := r.db.Table("roles").
			Select("id").
			Where("UPPER(name) = ?", strings.ToUpper(candidate)).
			Limit(1).
			Scan(&result).Error
		if err != nil {
			return "", err
		}
		if result.ID != "" {
			return result.ID, nil
		}
	}

	return "", fmt.Errorf("role %q tidak ditemukan", name)
}
// Steps returns the technician progress steps attached to a ticket.
func (r *TicketRepositoryStruct) Steps(ticketID uint64) ([]entities.TicketStep, error) {
	steps := []entities.TicketStep{}
	err := r.db.Where("ticket_id = ?", ticketID).Order("id ASC").Find(&steps).Error
	return steps, err
}

// nowOrEmpty formats an optional timestamp for the tracking timeline.
func nowOrEmpty(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}
