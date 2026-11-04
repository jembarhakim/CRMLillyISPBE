package ticketapi

import (
	"fmt"
	"log"
	"skripsi-be/internal/models/entities"
	"skripsi-be/internal/notifications"
	"skripsi-be/internal/telegram"
	"time"
)

type Service struct {
	repo            *Repo
	telegramService *telegram.Service
}

func NewService(r *Repo) *Service {
	return &Service{
		repo:            r,
		telegramService: telegram.NewService(),
	}
}

func (s *Service) CreateCS(input entities.TroubleTicket, classification string) (*entities.TroubleTicket, error) {
	// Force initial state to match DB enum values
	input.Status = "unfinished"
	if input.CurrentAssignee == "" {
		// Look up CS role ID dynamically
		csRoleID, err := s.repo.RoleIDByName(string(entities.AssignCS))
		if err != nil {
			return nil, err
		}
		if csRoleID == "" {
			return nil, fmt.Errorf("cs role ID is empty")
		}
		input.CurrentAssignee = csRoleID
	}

	// Set classification based on CS input
	input.SetClassification(classification)

	if err := s.repo.Create(&input); err != nil {
		return nil, err
	}
	notifications.QueueTicketCreated(s.repo.DB, &input)

	// Send Telegram notification to CS channel
	if err := s.telegramService.SendTicketNotification("CUSTOMER SERVICE", &input, "Tiket Baru Dibuat"); err != nil {
		log.Printf("Failed to send Telegram notification: %v", err)
	}

	// Auto assign technician AND return updated ticket for PSB, LAINNYA, DISMANTLE
	if input.ClassificationID == entities.ClassificationPSB ||
		input.ClassificationID == entities.ClassificationLainnya ||
		input.ClassificationID == entities.ClassificationDismantle {
		log.Printf("[Auto Assign] Ticket %d is %s. Auto-assigning technician...", input.ID, input.ClassificationID)
		if _, err := s.AssignTechnician(input.ID); err != nil {
			log.Printf("[Auto Assign] Failed to auto-assign technician for ticket %d: %v", input.ID, err)
			// Still proceed to try reloading and return
		}
		// Reload ticket from DB to get updated assignee
		updated, err := s.repo.ByID(input.ID)
		if err != nil {
			log.Printf("[Auto Assign] Failed to reload ticket %d after assignment: %v", input.ID, err)
			return &input, nil // fallback to pre-assign ticket
		}
		return updated, nil
	}

	// For all other types, return the CS-assigned ticket
	return &input, nil
}

// CreateTicketFromNetwatch create a ticket from Netwatch monitoring
// DISABLED FOR TESTING - Netwatch integration is currently disabled
func (s *Service) CreateTicketFromNetwatch(input *entities.TroubleTicket) (*entities.TroubleTicket, error) {
	// Force initial state to match DB enum values
	input.Status = "ongoing" // Netwatch tickets start as ongoing

	// Look up CS role ID dynamically for assignment
	csRoleID, err := s.repo.RoleIDByName(string(entities.AssignCS))
	if err != nil {
		return nil, err
	}
	if csRoleID == "" {
		return nil, fmt.Errorf("cs role ID is empty")
	}
	input.CurrentAssignee = csRoleID

	if err := s.repo.Create(input); err != nil {
		return nil, err
	}
	return input, nil
}

func (s *Service) SendToNOC(id uint64, note string, imageFilename *string) (*entities.TroubleTicket, error) {
	t, err := s.repo.ByID(id)
	if err != nil {
		return nil, err
	}
	t.Status = "ongoing"
	// Look up NOC role ID dynamically (NOC now uses CUSTOMER_SERVICE role)
	nocRoleName := string(entities.AssignNOC)
	log.Printf("SendToNOC: Looking up role ID for name: '%s' (NOC now uses CUSTOMER_SERVICE)", nocRoleName)
	nocRoleID, err := s.repo.RoleIDByName(nocRoleName)
	if err != nil {
		log.Printf("SendToNOC: Error looking up NOC role ID: %v", err)
		return nil, err
	}
	log.Printf("SendToNOC: Found NOC role ID: '%s' (using CUSTOMER_SERVICE role)", nocRoleID)
	if nocRoleID == "" {
		return nil, fmt.Errorf("noc role ID is empty for role name: %s", nocRoleName)
	}
	t.CurrentAssignee = nocRoleID
	t.CustomerNote = &note
	log.Printf("SendToNOC: Setting CustomerNote to: %s", note)
	if imageFilename != nil && *imageFilename != "" {
		t.ImgCS = imageFilename
		log.Printf("SendToNOC: Setting ImgCS to: %s", *imageFilename)
	} else {
		log.Printf("SendToNOC: No image provided")
	}
	log.Printf("SendToNOC: About to save ticket with CustomerNote=%v, ImgCS=%v", t.CustomerNote, t.ImgCS)
	if err := s.repo.Save(t); err != nil {
		return nil, err
	}
	notifications.QueueTicketInProgress(s.repo.DB, t)

	// Send Telegram notification to NOC channel
	if err := s.telegramService.SendTicketNotification("NOC", t, "Tiket Ditugaskan ke NOC"); err != nil {
		log.Printf("Failed to send Telegram notification to NOC: %v", err)
	}

	log.Printf("SendToNOC: Successfully saved ticket")
	return t, nil
}

// SendToCS moves the ticket back to Customer Service with an optional NOC note and image
func (s *Service) SendToCS(id uint64, note string, tType *string, imageFilename *string) (*entities.TroubleTicket, error) {
	t, err := s.repo.ByID(id)
	if err != nil {
		return nil, err
	}
	// keep status ongoing while reassigning
	t.Status = "ongoing"
	// If NOC provided a diagnosed trouble type, persist it
	if tType != nil && *tType != "" {
		t.Type = tType
	}
	// Look up CS role ID dynamically
	csRoleName := string(entities.AssignCS)
	log.Printf("SendToCS: Looking up role ID for name: '%s'", csRoleName)
	csRoleID, err := s.repo.RoleIDByName(csRoleName)
	if err != nil {
		return nil, err
	}
	if csRoleID == "" {
		return nil, fmt.Errorf("cs role ID is empty for role name: %s", csRoleName)
	}
	t.CurrentAssignee = csRoleID
	// reuse NOC note field for context when NOC sends back to CS
	t.NOCNote = &note

	// Save image filename if provided
	if imageFilename != nil && *imageFilename != "" {
		t.ImgNOC = imageFilename
	}

	if err := s.repo.Save(t); err != nil {
		return nil, err
	}

	// Send Telegram notification to CS channel
	if err := s.telegramService.SendTicketNotification("CUSTOMER SERVICE", t, "Tiket Dikembalikan ke Customer Service"); err != nil {
		log.Printf("Failed to send Telegram notification to CS: %v", err)
	}

	return t, nil
}

func (s *Service) NOCSolved(id uint64, note string) (*entities.TroubleTicket, error) {
	t, err := s.repo.ByID(id)
	if err != nil {
		return nil, err
	}
	t.Status = "finished"
	// Look up CS role ID dynamically
	csRoleID, err := s.repo.RoleIDByName(string(entities.AssignCS))
	if err != nil {
		return nil, err
	}
	if csRoleID == "" {
		return nil, fmt.Errorf("cs role ID is empty")
	}
	t.CurrentAssignee = csRoleID
	t.NOCNote = &note
	if err := s.repo.Save(t); err != nil {
		return nil, err
	}

	// Send Telegram notification to CS channel
	if err := s.telegramService.SendTicketNotification("CUSTOMER SERVICE", t, "Tiket Diselesaikan oleh NOC"); err != nil {
		log.Printf("Failed to send Telegram notification to CS: %v", err)
	}

	return t, nil
}

func (s *Service) NOCPhysical(id uint64, note string) (*entities.TroubleTicket, error) {
	t, err := s.repo.ByID(id)
	if err != nil {
		return nil, err
	}
	t.Status = "ongoing"
	// Look up CS role ID dynamically
	csRoleID, err := s.repo.RoleIDByName(string(entities.AssignCS))
	if err != nil {
		return nil, err
	}
	if csRoleID == "" {
		return nil, fmt.Errorf("cs role ID is empty")
	}
	t.CurrentAssignee = csRoleID
	t.NOCNote = &note
	if err := s.repo.Save(t); err != nil {
		return nil, err
	}

	// Send Telegram notification to CS channel
	if err := s.telegramService.SendTicketNotification("CUSTOMER SERVICE", t, "Tiket Memerlukan Pemeriksaan Fisik"); err != nil {
		log.Printf("Failed to send Telegram notification to CS: %v", err)
	}

	return t, nil
}

func (s *Service) AssignTechnician(id uint64) (*entities.TroubleTicket, error) {
	t, err := s.repo.ByID(id)
	if err != nil {
		return nil, err
	}
	// Don't assign to specific technician - leave AssignedTo as NULL so all technicians can see it
	t.AssignedTo = nil // Set to NULL so all technicians can see it
	// Look up Technician role ID dynamically
	techRoleID, err := s.repo.RoleIDByName(string(entities.AssignTech))
	if err != nil {
		return nil, err
	}
	if techRoleID == "" {
		return nil, fmt.Errorf("technician role ID is empty")
	}
	t.CurrentAssignee = techRoleID
	if t.Status == "unfinished" {
		t.Status = "ongoing"
	}
	if err := s.repo.Save(t); err != nil {
		return nil, err
	}

	// Send Telegram notification to Technician channel
	if err := s.telegramService.SendTicketNotification("TECHNICIAN", t, "Tiket Ditugaskan ke Teknisi"); err != nil {
		log.Printf("Failed to send Telegram notification to Technician: %v", err)
	}

	return t, nil
}

func (s *Service) TechnicianResolve(id uint64, note string) (*entities.TroubleTicket, error) {
	t, err := s.repo.ByID(id)
	if err != nil {
		return nil, err
	}
	t.Status = "finished"
	// Look up CS role ID dynamically
	csRoleID, err := s.repo.RoleIDByName(string(entities.AssignCS))
	if err != nil {
		return nil, err
	}
	if csRoleID == "" {
		return nil, fmt.Errorf("cs role ID is empty")
	}
	t.CurrentAssignee = csRoleID
	t.TechnicianNote = &note
	if err := s.repo.Save(t); err != nil {
		return nil, err
	}

	// Send Telegram notification to CS channel
	if err := s.telegramService.SendTicketNotification("CUSTOMER SERVICE", t, "Tiket Diselesaikan oleh Teknisi"); err != nil {
		log.Printf("Failed to send Telegram notification to CS: %v", err)
	}

	return t, nil
}

// CSResolve allows Customer Service to resolve tickets with a customer note
func (s *Service) CSResolve(id uint64, note string) (*entities.TroubleTicket, error) {
	t, err := s.repo.ByID(id)
	if err != nil {
		return nil, err
	}
	// Enforce technician completion before CS can resolve
	if t.TechnicianCompleted == nil || (t.TechnicianCompleted != nil && !*t.TechnicianCompleted) {
		return nil, fmt.Errorf("teknisi belum menyelesaikan pekerjaan")
	}
	t.Status = "finished"
	// Keep assigned to CS since they're resolving it
	csRoleID, err := s.repo.RoleIDByName(string(entities.AssignCS))
	if err != nil {
		return nil, err
	}
	if csRoleID == "" {
		return nil, fmt.Errorf("cs role ID is empty")
	}
	t.CurrentAssignee = csRoleID
	t.CustomerNote = &note
	if err := s.repo.Save(t); err != nil {
		return nil, err
	}
	notifications.QueueTicketCompleted(s.repo.DB, t, "ticket_resolved")
	return t, nil
}

// AddTechnicianNote adds a technician note to a ticket without changing status
func (s *Service) AddTechnicianNote(id uint64, note string, imgTechBf *string, imgTechAf *string) (*entities.TroubleTicket, error) {
	t, err := s.repo.ByID(id)
	if err != nil {
		return nil, err
	}

	// Update the technician note field
	t.TechnicianNote = &note

	// Update image fields if provided
	if imgTechBf != nil {
		t.ImgTechBF = imgTechBf
	}
	if imgTechAf != nil {
		t.ImgTechAF = imgTechAf
	}

	if err := s.repo.Save(t); err != nil {
		return nil, err
	}

	// Send Telegram notification to CS channel about technician note
	if err := s.telegramService.SendTicketNotification("CUSTOMER SERVICE", t, "Catatan Teknisi Ditambahkan"); err != nil {
		log.Printf("Failed to send Telegram notification to CS: %v", err)
	}

	return t, nil
}

// AcceptTicket allows a technician to accept a ticket (locks assignment to a single tech)
func (s *Service) AcceptTicket(id uint64, technicianUserID string) (*entities.TroubleTicket, error) {
	t, err := s.repo.ByID(id)
	if err != nil {
		return nil, err
	}
	if t.AssignedTo != nil && *t.AssignedTo != "" && *t.AssignedTo != technicianUserID {
		return nil, fmt.Errorf("tiket sudah diterima oleh teknisi lain")
	}
	t.AssignedTo = &technicianUserID
	if t.Status == "unfinished" {
		t.Status = "ongoing"
	}
	if err := s.repo.Save(t); err != nil {
		return nil, err
	}
	notifications.QueueTicketAccepted(s.repo.DB, t, technicianUserID)
	return t, nil
}

// SetTeam sets the team composition for a ticket
func (s *Service) SetTeam(ticketID uint64, team []entities.TechnicianTeamMember) error {
	// Validate roles
	for _, m := range team {
		switch m.Role {
		case "senior", "junior", "helper":
		default:
			return fmt.Errorf("peran tim tidak valid: %s", m.Role)
		}
	}
	return s.repo.ReplaceTeamMembers(ticketID, team)
}

// AddStep adds a troubleshooting step (max 7)
func (s *Service) AddStep(ticketID uint64, step entities.TicketStep) error {
	cnt, err := s.repo.CountTicketSteps(ticketID)
	if err != nil {
		return err
	}
	if cnt >= 7 {
		return fmt.Errorf("maksimum 7 langkah telah tercapai")
	}
	step.TicketID = ticketID
	step.StepOrder = int(cnt) + 1
	return s.repo.AddTicketStep(&step)
}

// AddStepImages attaches multiple images to a step
func (s *Service) AddStepImages(stepID uint64, paths []string) error {
	for _, p := range paths {
		if p == "" {
			continue
		}
		if err := s.repo.AddStepImage(&entities.TicketStepImage{StepID: stepID, Path: p}); err != nil {
			return err
		}
	}
	return nil
}

// VerifyAndClose allows CS to verify the report and close the ticket
func (s *Service) VerifyAndClose(ticketID uint64, csUserID string) (*entities.TroubleTicket, error) {
	t, err := s.repo.ByID(ticketID)
	if err != nil {
		return nil, err
	}
	now := time.Now() // get current time
	t.Status = "finished"
	// mark verified fields if present in DB
	// using raw SQL update for portability if fields exist
	if err := s.repo.DB.Model(&entities.TroubleTicket{}).Where("id = ?", ticketID).
		Updates(map[string]interface{}{
			"status":                "finished",
			"verified_by_cs":        1,
			"verified_at":           now,
			"current_assignee_role": t.CurrentAssignee, // keep
		}).Error; err != nil {
		return nil, err
	}
	notifications.QueueTicketCompleted(s.repo.DB, t, "ticket_resolved")
	return t, nil
}

// MarkTechnicianCompleted allows technician to mark their work as completed
func (s *Service) MarkTechnicianCompleted(ticketID uint64, technicianUserID string) (*entities.TroubleTicket, error) {
	// Use repository method to mark technician completed
	ticket, err := s.repo.MarkTechnicianCompleted(ticketID, technicianUserID)
	if err != nil {
		return nil, err
	}

	// Send Telegram notification to CS channel
	if err := s.telegramService.SendTicketNotification("CUSTOMER SERVICE", ticket, "Pekerjaan Teknisi Selesai - Siap untuk Review CS"); err != nil {
		log.Printf("Failed to send Telegram notification: %v", err)
	}
	notifications.QueueTicketCompleted(s.repo.DB, ticket, "work_order_completed")

	return ticket, nil
}

// (removed duplicate SetNetworkArchitecture; see technician_service.go)

// ValidateAndSetTeam validates technician assignments and sets team composition
func (s *Service) ValidateAndSetTeam(ticketID uint64, teamMembers []entities.TechnicianTeamMember) error {
	// Validate each team member assignment
	for _, member := range teamMembers {
		if err := s.repo.ValidateTechnicianAssignment(ticketID, member.UserID); err != nil {
			return err
		}
	}

	// Clear existing team members for this ticket
	if err := s.repo.DB.Where("ticket_id = ?", ticketID).Delete(&entities.TechnicianTeamMember{}).Error; err != nil {
		return fmt.Errorf("gagal menghapus tim yang ada: %v", err)
	}

	// Add new team members
	for _, member := range teamMembers {
		member.TicketID = ticketID
		if err := s.repo.DB.Create(&member).Error; err != nil {
			return fmt.Errorf("gagal menambahkan anggota tim: %v", err)
		}
	}

	return nil
}

// GetTechnicianTeamMembers gets all team members for a ticket
func (s *Service) GetTechnicianTeamMembers(ticketID uint64) ([]entities.TechnicianTeamMember, error) {
	return s.repo.GetTechnicianTeamMembers(ticketID)
}
