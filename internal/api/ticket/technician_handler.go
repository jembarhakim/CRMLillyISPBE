package ticketapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"skripsi-be/internal/helpers"
	"skripsi-be/internal/models/entities"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// GetTechnicianSteps gets all predefined technician steps
func (h *Handler) GetTechnicianSteps(c *fiber.Ctx) error {
	// Optional network architecture filter from query
	networkArch := c.Query("network_architecture")
	var networkArchPtr *string
	if networkArch != "" {
		networkArchPtr = &networkArch
	}

	steps, err := h.svc.GetTechnicianSteps(networkArchPtr)
	if err != nil {
		return helpers.ResponseUtils(c, 500, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, 200, true, "technician steps retrieved", steps)
}

// GetSpareParts gets all available spare parts
func (h *Handler) GetSpareParts(c *fiber.Ctx) error {
	parts, err := h.svc.GetSpareParts()
	if err != nil {
		return helpers.ResponseUtils(c, 500, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, 200, true, "spare parts retrieved", parts)
}

// GetTicketTechnicianSteps gets technician progress for a specific ticket
func (h *Handler) GetTicketTechnicianSteps(c *fiber.Ctx) error {
	id, err := idParam(c)
	if err != nil {
		return helpers.ResponseUtils(c, 400, false, err.Error(), nil)
	}

	technicianID := c.Query("technician_id")
	if technicianID == "" || technicianID == "TECHNICIAN" || technicianID == "ADMIN" || technicianID == "CUSTOMER_SERVICE" {
		if uid, ok := c.Locals("user_id").(string); ok && uid != "" {
			technicianID = uid
		}
	}

	steps, err := h.svc.GetTicketTechnicianSteps(id, technicianID)
	if err != nil {
		return helpers.ResponseUtils(c, 500, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, 200, true, "ticket technician steps retrieved", steps)
}

// UpdateTechnicianStep updates a technician's progress on a specific step
func (h *Handler) UpdateTechnicianStep(c *fiber.Ctx) error {
	id, err := idParam(c)
	if err != nil {
		return helpers.ResponseUtils(c, 400, false, err.Error(), nil)
	}

	// Handle multipart form data for images
	form, err := c.MultipartForm()
	if err != nil {
		return helpers.ResponseUtils(c, 400, false, "failed to parse multipart form", nil)
	}

	var req struct {
		StepID         uint64  `form:"step_id"`
		TechnicianID   string  `form:"technician_id"`
		Status         string  `form:"status"`
		Notes          *string `form:"notes"`
		SparePartsUsed *string `form:"spare_parts_used"`
	}

	// Parse form fields
	if stepIDStr := form.Value["step_id"]; len(stepIDStr) > 0 {
		if stepID, err := strconv.ParseUint(stepIDStr[0], 10, 64); err == nil {
			req.StepID = stepID
		}
	}
	if technicianID := form.Value["technician_id"]; len(technicianID) > 0 {
		req.TechnicianID = technicianID[0]
	}
	if status := form.Value["status"]; len(status) > 0 {
		req.Status = status[0]
	}
	if notes := form.Value["notes"]; len(notes) > 0 && notes[0] != "" {
		req.Notes = &notes[0]
	}
	if spareParts := form.Value["spare_parts_used"]; len(spareParts) > 0 && spareParts[0] != "" {
		req.SparePartsUsed = &spareParts[0]
	}

	// Resolve technician ID from JWT if missing or looks like a role label
	resolvedTechID := req.TechnicianID
	if resolvedTechID == "" || resolvedTechID == "TECHNICIAN" || resolvedTechID == "ADMIN" || resolvedTechID == "CUSTOMER_SERVICE" {
		if auth := c.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			parts := strings.Split(strings.TrimPrefix(auth, "Bearer "), ".")
			if len(parts) == 3 {
				if payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
					var payload map[string]interface{}
					if err := json.Unmarshal(payloadBytes, &payload); err == nil {
						if sub, ok := payload["sub"].(string); ok && sub != "" {
							resolvedTechID = sub
						}
					}
				}
			}
		}
	}

	// Handle uploaded images
	var imagePaths []string
	// Ensure upload directory exists (under public so it can be served statically)
	uploadDir := filepath.Join("public", "uploads", "technician-progress")
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		_ = os.MkdirAll(uploadDir, 0755)
	}
	if images := form.File["images"]; len(images) > 0 {
		for _, file := range images {
			// Save file to public/uploads/technician-progress/
			filename := fmt.Sprintf("%d_%s_%s", time.Now().Unix(), resolvedTechID, file.Filename)
			fullpath := filepath.Join(uploadDir, filename)

			if err := c.SaveFile(file, fullpath); err != nil {
				return helpers.ResponseUtils(c, 500, false, fmt.Sprintf("failed to save image: %v", err), nil)
			}

			imagePaths = append(imagePaths, filename)
		}
	}

	// Persist JSON image paths in step record
	var imagePathsJSON *string
	if len(imagePaths) > 0 {
		b, _ := json.Marshal(imagePaths)
		s := string(b)
		imagePathsJSON = &s
	}

	// Basic required fields validation
	if req.StepID == 0 || resolvedTechID == "" || req.Status == "" {
		return helpers.ResponseUtils(c, 400, false, "step_id, technician_id, and status are required", nil)
	}

	if err := h.svc.UpdateTechnicianStep(id, req.StepID, resolvedTechID, req.Status, req.Notes, req.SparePartsUsed, imagePathsJSON); err != nil {
		return helpers.ResponseUtils(c, 400, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, 200, true, "technician step updated successfully", nil)
}

// SetNetworkArchitecture sets FTTH/HTB for a ticket
func (h *Handler) SetNetworkArchitecture(c *fiber.Ctx) error {
	id, err := idParam(c)
	if err != nil {
		return helpers.ResponseUtils(c, 400, false, err.Error(), nil)
	}
	var req struct {
		Architecture string `json:"architecture"`
	}
	if err := c.BodyParser(&req); err != nil || req.Architecture == "" {
		return helpers.ResponseUtils(c, 400, false, "architecture is required", nil)
	}
	if err := h.svc.SetNetworkArchitecture(id, req.Architecture); err != nil {
		return helpers.ResponseUtils(c, 400, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, 200, true, "network architecture updated", nil)
}

// UpsertTechnicianTeam assigns Senior/Junior/Helper ensuring unique users
func (h *Handler) UpsertTechnicianTeam(c *fiber.Ctx) error {
	id, err := idParam(c)
	if err != nil {
		return helpers.ResponseUtils(c, 400, false, err.Error(), nil)
	}
	var req struct {
		Members []struct {
			UserID string `json:"user_id"`
			Role   string `json:"role"`
		} `json:"members"`
	}
	if err := c.BodyParser(&req); err != nil {
		return helpers.ResponseUtils(c, 400, false, "invalid body", nil)
	}
	members := make([]entities.TechnicianTeamMember, 0, len(req.Members))
	for _, m := range req.Members {
		members = append(members, entities.TechnicianTeamMember{TicketID: id, UserID: m.UserID, Role: strings.ToLower(m.Role)})
	}
	if err := h.svc.UpsertTechnicianTeam(id, members); err != nil {
		return helpers.ResponseUtils(c, 400, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, 200, true, "technician team updated", nil)
}

// GetTechnicianStepProgress gets progress summary for a technician on a ticket
func (h *Handler) GetTechnicianStepProgress(c *fiber.Ctx) error {
	id, err := idParam(c)
	if err != nil {
		return helpers.ResponseUtils(c, 400, false, err.Error(), nil)
	}

	technicianID := c.Query("technician_id")
	if technicianID == "" || technicianID == "TECHNICIAN" || technicianID == "ADMIN" || technicianID == "CUSTOMER_SERVICE" {
		if uid, ok := c.Locals("user_id").(string); ok && uid != "" {
			technicianID = uid
		}
	}

	progress, err := h.svc.GetTechnicianStepProgress(id, technicianID)
	if err != nil {
		return helpers.ResponseUtils(c, 500, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, 200, true, "technician step progress retrieved", progress)
}

// MarkTechnicianJobCompleted marks the entire technician job as completed
func (h *Handler) MarkTechnicianJobCompleted(c *fiber.Ctx) error {
	id, err := idParam(c)
	if err != nil {
		return helpers.ResponseUtils(c, 400, false, err.Error(), nil)
	}

	if err := h.svc.MarkTechnicianJobCompleted(id); err != nil {
		return helpers.ResponseUtils(c, 400, false, err.Error(), nil)
	}

	// Return updated ticket so clients can refresh UI status
	ticket, _ := h.svc.repo.GetTicketByID(id)
	return helpers.ResponseUtils(c, 200, true, "technician job marked as completed", ticket)
}

// GetTechnicianChecklist gets the complete checklist for a technician on a ticket
func (h *Handler) GetTechnicianChecklist(c *fiber.Ctx) error {
	id, err := idParam(c)
	if err != nil {
		return helpers.ResponseUtils(c, 400, false, err.Error(), nil)
	}

	technicianID := c.Query("technician_id")
	if technicianID == "" || technicianID == "TECHNICIAN" || technicianID == "SUPERADMIN" || technicianID == "CUSTOMER_SERVICE" {
		if uid, ok := c.Locals("user_id").(string); ok && uid != "" {
			technicianID = uid
		}
	}

	// Get ticket to retrieve network architecture and check if it's a dismantle ticket
	ticket, terr := h.svc.repo.GetTicketByID(id)
	var networkArchPtr *string
	isDismantleTicket := false

	if terr == nil {
		// Check if this is a dismantle ticket by checking classification_id field
		classificationStr := strings.ToLower(ticket.ClassificationID)
		if classificationStr == "dismantle" {
			isDismantleTicket = true
		}

		// Set network architecture
		if ticket.NetworkArchitecture != nil && *ticket.NetworkArchitecture != "" {
			networkArchPtr = ticket.NetworkArchitecture
		}

		// Auto-set network architecture to DISMANTLE for dismantle tickets
		if isDismantleTicket && (networkArchPtr == nil || *networkArchPtr == "") {
			dismantleArch := "DISMANTLE"
			networkArchPtr = &dismantleArch
			// Update ticket with DISMANTLE architecture
			_ = h.svc.repo.SetNetworkArchitecture(id, "DISMANTLE")
		}
	}

	// Get all predefined steps filtered by network architecture
	steps, err := h.svc.GetTechnicianSteps(networkArchPtr)
	if err != nil {
		return helpers.ResponseUtils(c, 500, false, err.Error(), nil)
	}

	// Filter to only step ID 12 and selfie step (step_order = 0) for dismantle tickets
	const DISMANTLE_STEP_ID = uint64(12)
	if isDismantleTicket {
		var dismantleSteps []entities.TechnicianStep
		for _, step := range steps {
			// Include selfie step (step_order = 0) and dismantle photo step (id = 12)
			if step.StepOrder == 0 || step.ID == DISMANTLE_STEP_ID {
				dismantleSteps = append(dismantleSteps, step)
			}
		}
		steps = dismantleSteps
	}

	// Get technician progress
	progress, err := h.svc.GetTicketTechnicianSteps(id, technicianID)
	if err != nil {
		return helpers.ResponseUtils(c, 500, false, err.Error(), nil)
	}

	// Create a map of progress for quick lookup
	progressMap := make(map[uint64]entities.TicketTechnicianStep)
	for _, p := range progress {
		progressMap[p.StepID] = p
	}

	// Combine steps with progress
	var checklist []map[string]interface{}
	for _, step := range steps {
		item := map[string]interface{}{
			"step_id":          step.ID,
			"step_order":       step.StepOrder,
			"title":            step.Title,
			"description":      step.Description,
			"tools":            step.Tools,
			"spare_parts":      step.SpareParts,
			"procedure":        step.Procedure,
			"solution":         step.Solution,
			"status":           "pending",
			"notes":            nil,
			"spare_parts_used": nil,
			"completed_at":     nil,
			"image_paths":      nil,
		}

		// Add progress if exists
		if p, exists := progressMap[step.ID]; exists {
			item["status"] = p.Status
			item["notes"] = p.Notes
			item["spare_parts_used"] = p.SparePartsUsed
			item["completed_at"] = p.CompletedAt
			item["image_paths"] = p.ImagePaths
		}

		checklist = append(checklist, item)
	}

	// Return checklist with network architecture and ticket metadata
	// Include classification info so frontend can also detect dismantle tickets
	responseData := fiber.Map{
		"checklist":            checklist,
		"network_architecture": networkArchPtr,
	}

	// Add ticket metadata for frontend to detect dismantle status
	if terr == nil {
		responseData["ticket"] = fiber.Map{
			"classification_id": ticket.ClassificationID,
			"status":            ticket.Status,
		}
	}

	return helpers.ResponseUtils(c, 200, true, "technician checklist retrieved", responseData)
}

// SaveSelfieStep saves a selfie photo as step 0 for a ticket
func (h *Handler) SaveSelfieStep(c *fiber.Ctx) error {
	id, err := idParam(c)
	if err != nil {
		return helpers.ResponseUtils(c, 400, false, err.Error(), nil)
	}

	// Handle multipart form data for selfie image
	form, err := c.MultipartForm()
	if err != nil {
		fmt.Printf("Error parsing multipart form: %v\n", err)
		return helpers.ResponseUtils(c, 400, false, fmt.Sprintf("failed to parse multipart form: %v", err), nil)
	}

	// Get technician ID from JWT or form
	technicianID := ""
	if techID := form.Value["technician_id"]; len(techID) > 0 && techID[0] != "" {
		technicianID = techID[0]
	}

	// Resolve technician ID from JWT if missing or looks like a role label
	if technicianID == "" || technicianID == "TECHNICIAN" || technicianID == "ADMIN" || technicianID == "CUSTOMER_SERVICE" {
		if uid, ok := c.Locals("user_id").(string); ok && uid != "" {
			technicianID = uid
		} else if auth := c.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			parts := strings.Split(strings.TrimPrefix(auth, "Bearer "), ".")
			if len(parts) == 3 {
				if payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
					var payload map[string]interface{}
					if err := json.Unmarshal(payloadBytes, &payload); err == nil {
						if sub, ok := payload["sub"].(string); ok && sub != "" {
							technicianID = sub
						}
					}
				}
			}
		}
	}

	if technicianID == "" {
		fmt.Printf("Technician ID not found. Locals user_id: %v\n", c.Locals("user_id"))
		return helpers.ResponseUtils(c, 400, false, "technician_id is required. Please ensure you are authenticated.", nil)
	}

	// Handle uploaded selfie image
	var savedImagePath string
	uploadDir := filepath.Join("public", "uploads", "technician-selfie")
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		_ = os.MkdirAll(uploadDir, 0755)
	}

	// Get selfie file (expecting field name "selfie" or "image")
	var selfieFile *multipart.FileHeader
	if files := form.File["selfie"]; len(files) > 0 {
		selfieFile = files[0]
	} else if files := form.File["image"]; len(files) > 0 {
		selfieFile = files[0]
	} else if files := form.File["images"]; len(files) > 0 {
		selfieFile = files[0]
	}

	if selfieFile == nil {
		// Log available form fields for debugging
		var fileKeys []string
		for k := range form.File {
			fileKeys = append(fileKeys, k)
		}
		fmt.Printf("No selfie file found. Available file fields: %v\n", fileKeys)
		return helpers.ResponseUtils(c, 400, false, "selfie image is required. Please ensure the file field is named 'selfie', 'image', or 'images'", nil)
	}

	// Validate file size (max 20MB)
	if selfieFile.Size > 20*1024*1024 {
		return helpers.ResponseUtils(c, 400, false, "image size exceeds 20MB limit", nil)
	}

	// Save file
	filename := fmt.Sprintf("%d_%s_selfie_%s", time.Now().Unix(), technicianID, selfieFile.Filename)
	fullpath := filepath.Join(uploadDir, filename)

	if err := c.SaveFile(selfieFile, fullpath); err != nil {
		return helpers.ResponseUtils(c, 500, false, fmt.Sprintf("failed to save image: %v", err), nil)
	}

	// Save full path to database (relative to public directory)
	savedImagePath = filepath.Join("uploads", "technician-selfie", filename)

	// Save to database as step 0
	if err := h.svc.SaveSelfieStep(id, technicianID, savedImagePath); err != nil {
		// Clean up saved file on error
		_ = os.Remove(fullpath)
		// Log the error for debugging
		fmt.Printf("Error saving selfie step: %v\n", err)
		return helpers.ResponseUtils(c, 400, false, fmt.Sprintf("Failed to save selfie: %v", err), nil)
	}

	return helpers.ResponseUtils(c, 200, true, "Foto selfie disimpan", fiber.Map{
		"selfie_path": filename, // Return just filename for frontend URL construction
		"step_order":  0,
	})
}

// GetSelfieStep retrieves the selfie photo (step 0) for a ticket
func (h *Handler) GetSelfieStep(c *fiber.Ctx) error {
	id, err := idParam(c)
	if err != nil {
		return helpers.ResponseUtils(c, 400, false, err.Error(), nil)
	}

	// Get user role to determine if admin/customer service can view any technician's selfie
	roleVal := c.Locals("role")
	role, _ := roleVal.(string)
	normalizedRole := helpers.NormalizeRole(role)

	// Get technician ID from query parameter
	technicianID := c.Query("technician_id")

	// For technicians, use their own ID if not provided
	// For admin/customer service, allow querying any technician or find any selfie on the ticket
	if normalizedRole == "TECHNICIAN" {
		if technicianID == "" || technicianID == "TECHNICIAN" || technicianID == "ADMIN" || technicianID == "CUSTOMER_SERVICE" {
			if uid, ok := c.Locals("user_id").(string); ok && uid != "" {
				technicianID = uid
			} else if auth := c.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				parts := strings.Split(strings.TrimPrefix(auth, "Bearer "), ".")
				if len(parts) == 3 {
					if payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
						var payload map[string]interface{}
						if err := json.Unmarshal(payloadBytes, &payload); err == nil {
							if sub, ok := payload["sub"].(string); ok && sub != "" {
								technicianID = sub
							}
						}
					}
				}
			}
		}
		if technicianID == "" {
			return helpers.ResponseUtils(c, 400, false, "technician_id is required", nil)
		}
	}

	// Get step with step_order = 0
	step, err := h.svc.repo.GetStepByOrder(0)
	if err != nil {
		return helpers.ResponseUtils(c, 404, false, "step 0 not found", nil)
	}

	var ticketStep *entities.TicketTechnicianStep

	// For admin/customer service: if technician_id is provided, use it; otherwise find any selfie on the ticket
	if normalizedRole == "ADMIN" || normalizedRole == "CUSTOMER_SERVICE" || normalizedRole == "SUPERADMIN" {
		if technicianID != "" && technicianID != "ADMIN" && technicianID != "CUSTOMER_SERVICE" && technicianID != "SUPERADMIN" {
			// Query specific technician
			ticketSteps, err := h.svc.GetTicketTechnicianSteps(id, technicianID)
			if err != nil {
				return helpers.ResponseUtils(c, 500, false, err.Error(), nil)
			}
			for i := range ticketSteps {
				if ticketSteps[i].StepID == step.ID {
					ticketStep = &ticketSteps[i]
					break
				}
			}
		} else {
			// Find any selfie on the ticket (query all technician steps for step 0)
			// Get all technician team members for this ticket to find their selfies
			var teamMembers []entities.TechnicianTeamMember
			if err := h.svc.repo.DB.Where("ticket_id = ?", id).Find(&teamMembers).Error; err == nil {
				// Try each team member
				for _, member := range teamMembers {
					steps, err := h.svc.GetTicketTechnicianSteps(id, member.UserID)
					if err == nil {
						for i := range steps {
							if steps[i].StepID == step.ID {
								ticketStep = &steps[i]
								break
							}
						}
						if ticketStep != nil {
							break
						}
					}
				}
			}
			// Fallback: direct query if team members approach doesn't work
			if ticketStep == nil {
				var allSteps []entities.TicketTechnicianStep
				if err := h.svc.repo.DB.Where("ticket_id = ? AND step_id = ?", id, step.ID).Find(&allSteps).Error; err == nil {
					if len(allSteps) > 0 {
						// Return the first selfie found
						ticketStep = &allSteps[0]
					}
				}
			}
		}
	} else {
		// For technicians: must query their own steps
		if technicianID == "" {
			return helpers.ResponseUtils(c, 400, false, "technician_id is required", nil)
		}
		ticketSteps, err := h.svc.GetTicketTechnicianSteps(id, technicianID)
		if err != nil {
			return helpers.ResponseUtils(c, 500, false, err.Error(), nil)
		}
		for i := range ticketSteps {
			if ticketSteps[i].StepID == step.ID {
				ticketStep = &ticketSteps[i]
				break
			}
		}
	}

	if ticketStep == nil {
		return helpers.ResponseUtils(c, 404, false, "selfie not found", nil)
	}

	// Extract image path from JSON array
	var imagePaths []string
	if ticketStep.ImagePaths != nil && *ticketStep.ImagePaths != "" {
		if err := json.Unmarshal([]byte(*ticketStep.ImagePaths), &imagePaths); err == nil && len(imagePaths) > 0 {
			// Extract just the filename from the path (handle both old and new path formats)
			fullPath := imagePaths[0]
			// Normalize path separators (handle Windows backslashes)
			fullPath = strings.ReplaceAll(fullPath, "\\", "/")
			// Remove any path prefixes (uploads/technician-progress/ or uploads/technician-selfie/)
			fullPath = strings.TrimPrefix(fullPath, "uploads/technician-progress/")
			fullPath = strings.TrimPrefix(fullPath, "uploads/technician-selfie/")
			// Extract just the filename
			filename := filepath.Base(fullPath)
			// Debug: log the extracted filename
			fmt.Printf("Extracted filename from path '%s': '%s'\n", imagePaths[0], filename)
			// Ensure we only return the filename, not the path
			return helpers.ResponseUtils(c, 200, true, "selfie retrieved", fiber.Map{
				"selfie_path": filename,
			})
		}
	}

	return helpers.ResponseUtils(c, 404, false, "selfie image not found", nil)
}
