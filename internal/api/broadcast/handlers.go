package broadcast

import (
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"skripsi-be/internal/models/entities"
	"skripsi-be/internal/notifications"
)

// PersonalizedMessage represents a phone number and its personalized message
type PersonalizedMessage struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

// BroadcastRequest represents the request body for sending a broadcast
type BroadcastRequest struct {
	Target               string                `json:"target"`                // "customers" or "team"
	Phones               []string              `json:"phones"`                // Used only if target is "customers" and no personalized_messages
	Message              string                `json:"message"`               // The message to send (used if no personalized_messages)
	PersonalizedMessages []PersonalizedMessage `json:"personalized_messages"` // Optional: array of phone+message pairs for personalized broadcasts
	TemplateKey          string                `json:"template_key"`          // Optional template key (e.g., "template_outage")
}

// BroadcastResponse represents the response after sending a broadcast
type BroadcastResponse struct {
	Status     string `json:"status"`
	Recipients int    `json:"recipients"`
}

// HandleSendBroadcast handles POST /api/broadcast/send
func HandleSendBroadcast(c *fiber.Ctx, db *gorm.DB) error {
	var request BroadcastRequest

	// Parse JSON body
	if err := c.BodyParser(&request); err != nil {
		log.Printf("[BROADCAST] Error parsing JSON: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot parse JSON",
		})
	}

	// Debug: Log the received request
	log.Printf("[BROADCAST] Received request: Target=%s, Message=%s, PersonalizedMessages=%d, Phones=%d",
		request.Target,
		func() string {
			if len(request.Message) > 50 {
				return request.Message[:50] + "..."
			}
			return request.Message
		}(),
		len(request.PersonalizedMessages),
		len(request.Phones))

	// Validate required fields
	if request.Target == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "target is required",
		})
	}

	// Validate: either message or personalized_messages must be provided
	// For personalized messages, we don't need the base message field
	if request.Message == "" && (request.PersonalizedMessages == nil || len(request.PersonalizedMessages) == 0) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "message or personalized_messages is required",
		})
	}

	// Additional validation: if personalized_messages is provided, ensure it has at least one entry
	if len(request.PersonalizedMessages) > 0 {
		for i, pm := range request.PersonalizedMessages {
			if pm.Phone == "" || pm.Message == "" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fmt.Sprintf("personalized_messages[%d] must have both phone and message", i),
				})
			}
		}
	}

	// Get user ID from middleware (assuming it's set by auth middleware)
	var userID string
	if userIDVal := c.Locals("user_id"); userIDVal != nil {
		userID = fmt.Sprintf("%v", userIDVal)
	} else if userIDVal := c.Locals("userID"); userIDVal != nil {
		userID = fmt.Sprintf("%v", userIDVal)
	} else {
		userID = "" // Set to empty string if not available (will be NULL in DB)
	}

	// Initialize phone list and message map
	var phoneList []string
	var messageMap map[string]string // phone -> personalized message
	var baseMessage string

	// Handle personalized messages
	if len(request.PersonalizedMessages) > 0 {
		messageMap = make(map[string]string)
		for _, pm := range request.PersonalizedMessages {
			phoneList = append(phoneList, pm.Phone)
			messageMap[pm.Phone] = pm.Message
		}
		// Use first message as base message for storage
		baseMessage = request.PersonalizedMessages[0].Message
	} else {
		// Determine phone list based on target
		if request.Target == "team" {
			// Query users table to get all team phone numbers
			var users []entities.User
			if err := db.Where("phone IS NOT NULL AND phone != ''").Find(&users).Error; err != nil {
				log.Printf("Error querying users: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "failed to query team members",
				})
			}

			// Extract phone numbers
			for _, user := range users {
				if user.Phone != "" {
					phoneList = append(phoneList, user.Phone)
				}
			}
		} else if request.Target == "customers" {
			// Use phones from request directly
			if len(request.Phones) == 0 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "phones array or personalized_messages is required for customer broadcasts",
				})
			}
			phoneList = request.Phones
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid target. must be 'customers' or 'team'",
			})
		}
		baseMessage = request.Message
	}

	// Create broadcast history record (ONE record for all recipients)
	broadcastHistory := entities.BroadcastHistory{
		Message:         baseMessage, // Store base message (first personalized message or regular message)
		TargetGroup:     request.Target,
		RecipientCount:  len(phoneList),
		RecipientPhones: phoneList, // Store all phone numbers
		Status:          "pending", // Start as pending, will update after sending
		TemplateKey:     request.TemplateKey,
		SentAt:          time.Now(),
	}

	// Set CreatedBy only if userID is not empty (to avoid foreign key constraint issues)
	if userID != "" {
		broadcastHistory.CreatedBy = userID
	}

	// Save to database first
	if err := db.Create(&broadcastHistory).Error; err != nil {
		log.Printf("Error saving broadcast history: %v", err)
		log.Printf("BroadcastHistory data: Message=%s, TargetGroup=%s, RecipientCount=%d, Status=%s, CreatedBy=%s",
			broadcastHistory.Message, broadcastHistory.TargetGroup, broadcastHistory.RecipientCount, broadcastHistory.Status, broadcastHistory.CreatedBy)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed to save broadcast history",
			"details": err.Error(),
		})
	}

	queueService := notifications.NewService(db)
	queuedCount := 0
	failedCount := 0
	var failedPhones []string

	// Queue messages individually so large broadcasts do not block the HTTP request.
	log.Printf("[BROADCAST] Queueing %d messages for %s", len(phoneList), request.Target)
	for _, phone := range phoneList {
		var messageToSend string
		if len(messageMap) > 0 {
			if personalizedMsg, ok := messageMap[phone]; ok {
				messageToSend = personalizedMsg
			} else {
				messageToSend = baseMessage
			}
		} else {
			messageToSend = request.Message
		}
		templateKey := request.TemplateKey
		if templateKey == "" {
			templateKey = "broadcast"
		}
		if err := queueService.EnqueueRaw(phone, messageToSend, templateKey, broadcastHistory.ID); err != nil {
			log.Printf("[BROADCAST] Failed to queue recipient %s: %v", phone, err)
			failedCount++
			failedPhones = append(failedPhones, phone)
		} else {
			queuedCount++
		}
	}

	if queuedCount == 0 {
		broadcastHistory.Status = "failed"
		if err := db.Model(&broadcastHistory).Update("status", broadcastHistory.Status).Error; err != nil {
			log.Printf("Error updating broadcast history status: %v", err)
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":        "failed",
			"recipients":    len(phoneList),
			"success_count": 0,
			"failed_count":  failedCount,
			"failed_phones": failedPhones,
		})
	}

	return c.JSON(fiber.Map{
		"status":        "queued",
		"recipients":    len(phoneList),
		"success_count": queuedCount,
		"failed_count":  failedCount,
		"failed_phones": failedPhones,
	})
}

// HandleGetBroadcastHistory handles GET /api/broadcast/history
func HandleGetBroadcastHistory(c *fiber.Ctx, db *gorm.DB) error {
	var histories []entities.BroadcastHistory

	// Query broadcast history with user relation
	// Order by sent_at DESC and limit to last 50 records
	if err := db.
		Preload("User").
		Order("sent_at DESC").
		Limit(50).
		Find(&histories).Error; err != nil {
		log.Printf("Error querying broadcast history: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch broadcast history",
		})
	}

	// Format response to match frontend expectations
	type BroadcastHistoryResponse struct {
		ID              string    `json:"id"`
		Message         string    `json:"message"`
		TargetGroup     string    `json:"target_group"`
		Status          string    `json:"status"`
		SentAt          time.Time `json:"sent_at"`
		CreatedBy       string    `json:"created_by,omitempty"`
		UserName        string    `json:"user_name,omitempty"`
		RecipientCount  int       `json:"recipient_count"`
		RecipientPhones []string  `json:"recipient_phones,omitempty"`
	}

	response := make([]BroadcastHistoryResponse, len(histories))
	for i, h := range histories {
		// Convert StringArray to []string, ensure it's never nil
		recipientPhones := []string(h.RecipientPhones)
		if recipientPhones == nil {
			recipientPhones = []string{}
		}

		response[i] = BroadcastHistoryResponse{
			ID:              h.ID,
			Message:         h.Message,
			TargetGroup:     h.TargetGroup,
			Status:          h.Status,
			SentAt:          h.SentAt,
			CreatedBy:       h.CreatedBy,
			RecipientCount:  h.RecipientCount,
			RecipientPhones: recipientPhones, // Always include, even if empty
		}
		// Capitalize status for frontend compatibility
		if h.Status == "sent" {
			response[i].Status = "Sent"
		} else if h.Status == "failed" {
			response[i].Status = "Failed"
		}
		// Add user name if available
		if h.User.Name != "" {
			response[i].UserName = h.User.Name
		}

		// Debug logging
		log.Printf("[BROADCAST HISTORY] ID=%s, RecipientCount=%d, RecipientPhones=%v", h.ID, h.RecipientCount, recipientPhones)
	}

	return c.JSON(response)
}

// HandleGetBroadcastRecipients handles GET /api/broadcast/:id/recipients
// Returns the list of phone numbers that received a specific broadcast
// Useful for sending follow-up broadcasts (e.g., when devices come back online)
func HandleGetBroadcastRecipients(c *fiber.Ctx, db *gorm.DB) error {
	broadcastID := c.Params("id")
	if broadcastID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "broadcast ID is required",
		})
	}

	var broadcastHistory entities.BroadcastHistory
	if err := db.Where("id = ?", broadcastID).First(&broadcastHistory).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "broadcast not found",
			})
		}
		log.Printf("Error querying broadcast: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch broadcast",
		})
	}

	// Convert StringArray to []string for JSON response
	recipientPhones := []string(broadcastHistory.RecipientPhones)
	if recipientPhones == nil {
		recipientPhones = []string{}
	}

	// Return the recipient phones
	response := fiber.Map{
		"broadcast_id":     broadcastID,
		"target_group":     broadcastHistory.TargetGroup,
		"recipient_count":  broadcastHistory.RecipientCount,
		"recipient_phones": recipientPhones, // Convert to []string
		"sent_at":          broadcastHistory.SentAt,
		"message":          broadcastHistory.Message,
	}

	return c.JSON(response)
}
