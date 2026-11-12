package incidents

import (
	"fmt"
	"log"
	"strings"
	"time"

	"skripsi-be/internal/models/entities"
	"skripsi-be/internal/notifications"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) List(status string) ([]entities.MassIncident, error) {
	var items []entities.MassIncident
	q := s.db.Order("created_at DESC")
	if status = strings.TrimSpace(status); status != "" {
		q = q.Where("status = ?", status)
	}
	return items, q.Find(&items).Error
}

func (s *Service) Create(req CreateRequest) (*entities.MassIncident, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("title wajib diisi")
	}
	if strings.TrimSpace(req.Source) == "" {
		req.Source = "manual"
	}
	var desc *string
	if strings.TrimSpace(req.Description) != "" {
		desc = &req.Description
	}
	item := &entities.MassIncident{
		ID: uuid.NewString(), Title: req.Title, Description: desc, Source: req.Source,
		Status: entities.MassIncidentStatusActive, NodeName: req.NodeName,
		AffectedCustomerIDs: entities.StringArray(req.CustomerIDs),
	}
	if req.NodeID != "" {
		item.NodeID = &req.NodeID
	}
	if req.ODPNodeID != "" {
		item.ODPNodeID = &req.ODPNodeID
	}
	if err := s.db.Create(item).Error; err != nil {
		return nil, err
	}
	s.createAffectedTickets(item)
	s.queueCustomerNotifications(item)
	return item, nil
}

func (s *Service) Update(item *entities.MassIncident) error {
	return s.db.Save(item).Error
}

func (s *Service) Get(id string) (*entities.MassIncident, error) {
	var item entities.MassIncident
	if err := s.db.First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) Resolve(id, note string) (*entities.MassIncident, error) {
	var item entities.MassIncident
	if err := s.db.First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	item.Status = entities.MassIncidentStatusResolved
	item.ResolvedAt = &now
	if strings.TrimSpace(note) != "" {
		item.ResolutionNote = &note
	}
	if err := s.db.Save(&item).Error; err != nil {
		return nil, err
	}
	if len(item.AffectedTicketIDs) > 0 {
		s.db.Model(&entities.TroubleTicket{}).
			Where("id IN ? AND status != ?", []string(item.AffectedTicketIDs), "resolved").
			Updates(map[string]interface{}{"status": "resolved", "updated_at": now})
	}
	return &item, nil
}

func (s *Service) GetActiveForCustomer(customerID string) ([]entities.MassIncident, error) {
	var links []entities.CustomerNetworkLink
	if err := s.db.Where("customer_id = ?", customerID).Find(&links).Error; err != nil {
		return nil, err
	}
	var item []entities.MassIncident
	q := s.db.Where("status = ?", entities.MassIncidentStatusActive)
	if len(links) > 0 {
		odpIDs := make([]string, 0, len(links))
		for _, link := range links {
			odpIDs = append(odpIDs, link.ODPNodeID)
		}
		q = q.Where("odp_node_id IN ? OR affected_customer_ids LIKE ?", odpIDs, "%"+customerID+"%")
	} else {
		q = q.Where("affected_customer_ids LIKE ?", "%"+customerID+"%")
	}
	return item, q.Order("created_at DESC").Find(&item).Error
}

func (s *Service) createAffectedTickets(incident *entities.MassIncident) {
	if len(incident.AffectedCustomerIDs) == 0 {
		return
	}
	var ids []string
	now := time.Now()
	for _, customerID := range incident.AffectedCustomerIDs {
		ticket := entities.TroubleTicket{
			CustomerID: customerID, Title: "Gangguan Massal: " + incident.Title,
			Description: stringPtr("Insiden gangguan massal otomatis. " + incident.Title),
			Status:      "unfinished", ClassificationID: "gangguan",
			CurrentAssignee: "CUSTOMER SERVICE", Accumulation: len(incident.AffectedCustomerIDs),
			MassIncidentID: &incident.ID, CreatedAt: &now, UpdatedAt: &now,
		}
		if err := s.db.Create(&ticket).Error; err != nil {
			log.Printf("[mass-incident] failed to create affected ticket: %v", err)
			continue
		}
		ids = append(ids, fmt.Sprintf("%d", ticket.ID))
	}
	incident.AffectedTicketIDs = entities.StringArray(ids)
	_ = s.db.Model(&entities.MassIncident{}).Where("id = ?", incident.ID).Update("affected_ticket_ids", incident.AffectedTicketIDs)
}

func (s *Service) queueCustomerNotifications(incident *entities.MassIncident) {
	var customers []entities.Customer
	if len(incident.AffectedCustomerIDs) == 0 {
		return
	}
	if err := s.db.Where("id IN ? AND phone <> ''", []string(incident.AffectedCustomerIDs)).Find(&customers).Error; err != nil {
		return
	}
	eta := "menyusul"
	if incident.ETA != nil {
		eta = incident.ETA.Format("02 Jan 2006 15:04")
	}
	area := incident.NodeName
	if area == "" {
		area = "wilayah terdampak"
	}
	ns := notifications.NewService(s.db)
	for _, customer := range customers {
		message := fmt.Sprintf("Pemberitahuan: Terjadi gangguan jaringan di wilayah %s. Tim teknisi sedang melakukan penanganan (Estimasi perbaikan: %s).", area, eta)
		if err := ns.EnqueueRaw(customer.Phone, message, "mass_incident", ""); err != nil {
			log.Printf("[mass-incident] failed queue WA for %s: %v", customer.ID, err)
		}
	}
}

func stringPtr(v string) *string { return &v }

// TriggerKumaDown creates one active incident when a switch/distribution node goes down.
func TriggerKumaDown(db *gorm.DB, monitorName string) (*entities.MassIncident, error) {
	var node entities.NetworkNode
	name := strings.TrimSpace(monitorName)
	if err := db.Where(
		"type = ? AND (LOWER(name) = LOWER(?) OR LOWER(code) = LOWER(?) OR LOWER(name) LIKE ?)",
		entities.NodeTypeSwitchHub, name, name, "%"+name+"%",
	).First(&node).Error; err != nil {
		return nil, nil
	}

	var existing entities.MassIncident
	if err := db.Where("status = ? AND source = ? AND node_id = ?", entities.MassIncidentStatusActive, "kuma", node.ID).First(&existing).Error; err == nil {
		return &existing, nil
	}

	customerIDs := affectedCustomersUnderNode(db, node.ID)
	service := NewService(db)
	return service.Create(CreateRequest{
		Title:       fmt.Sprintf("Gangguan Switch/Distribusi %s", node.Name),
		Description: "Node switch/distribusi terdeteksi down oleh Uptime Kuma.",
		Source:      "kuma", NodeID: node.ID, NodeName: node.Name,
		CustomerIDs: customerIDs,
	})
}

// TriggerODPDown creates a deduplicated incident for an ODP with at least five affected customers.
func TriggerODPDown(db *gorm.DB, odpNodeID, odpName string, customerIDs []string) (*entities.MassIncident, error) {
	if len(customerIDs) < 5 {
		return nil, nil
	}
	var existing entities.MassIncident
	if err := db.Where("status = ? AND source = ? AND odp_node_id = ?", entities.MassIncidentStatusActive, "pppoe_odp", odpNodeID).First(&existing).Error; err == nil {
		return &existing, nil
	}
	service := NewService(db)
	return service.Create(CreateRequest{
		Title:       fmt.Sprintf("Gangguan Massal ODP %s", odpName),
		Description: "Minimal 5 pelanggan pada ODP yang sama terdeteksi tidak memiliki sesi PPPoE aktif.",
		Source:      "pppoe_odp", ODPNodeID: odpNodeID, NodeName: odpName,
		CustomerIDs: customerIDs,
	})
}

func affectedCustomersUnderNode(db *gorm.DB, nodeID string) []string {
	var nodes []entities.NetworkNode
	_ = db.Find(&nodes).Error
	odpIDs := map[string]bool{}
	for _, node := range nodes {
		if node.Type == entities.NodeTypeODP && isDescendant(node, nodeID, nodes) {
			odpIDs[node.ID] = true
		}
	}
	if len(odpIDs) == 0 {
		return nil
	}
	ids := make([]string, 0, len(odpIDs))
	for id := range odpIDs {
		ids = append(ids, id)
	}
	var links []entities.CustomerNetworkLink
	_ = db.Where("odp_node_id IN ?", ids).Find(&links).Error
	customers := make([]string, 0, len(links))
	for _, link := range links {
		customers = append(customers, link.CustomerID)
	}
	return customers
}

func isDescendant(node entities.NetworkNode, ancestorID string, nodes []entities.NetworkNode) bool {
	byID := make(map[string]entities.NetworkNode, len(nodes))
	for _, n := range nodes {
		byID[n.ID] = n
	}
	current := node
	for current.ParentID != nil {
		if *current.ParentID == ancestorID {
			return true
		}
		parent, ok := byID[*current.ParentID]
		if !ok {
			return false
		}
		current = parent
	}
	return false
}
