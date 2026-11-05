package notificationsapi

import (
	"crypto/subtle"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"

	"skripsi-be/internal/api/webhook/wa"
	"skripsi-be/internal/helpers"
	"skripsi-be/internal/notifications"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Handler struct {
	service *notifications.Service
	db      *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{service: notifications.NewService(db), db: db}
}

func (h *Handler) Templates(c *fiber.Ctx) error {
	templates, err := h.service.ListTemplates()
	if err != nil {
		return helpers.ResponseUtils(c, 500, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, 200, true, "ok", templates)
}

func (h *Handler) SaveTemplate(c *fiber.Ctx) error {
	var input notifications.TemplateInput
	if err := c.BodyParser(&input); err != nil {
		return helpers.ResponseUtils(c, 400, false, "Invalid request body", nil)
	}
	if key := c.Params("key"); key != "" {
		input.Key = key
	}
	template, err := h.service.SaveTemplate(input)
	if err != nil {
		return helpers.ResponseUtils(c, 400, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, 200, true, "Template saved", template)
}

func (h *Handler) DeleteTemplate(c *fiber.Ctx) error {
	if err := h.service.DeleteTemplate(c.Params("key")); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helpers.ResponseUtils(c, 404, false, "Template not found", nil)
		}
		return helpers.ResponseUtils(c, 500, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, 200, true, "Template deleted", nil)
}

func (h *Handler) PreviewTemplate(c *fiber.Ctx) error {
	var input struct {
		Body      string         `json:"body"`
		Variables map[string]any `json:"variables"`
	}
	if err := c.BodyParser(&input); err != nil {
		return helpers.ResponseUtils(c, 400, false, "Invalid request body", nil)
	}
	return helpers.ResponseUtils(c, 200, true, "Preview generated", fiber.Map{
		"message": notifications.Render(input.Body, input.Variables),
	})
}

func (h *Handler) ListNotifications(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	items, err := h.service.ListNotifications(c.Query("status"), limit)
	if err != nil {
		return helpers.ResponseUtils(c, 500, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, 200, true, "ok", items)
}

func (h *Handler) ListLogs(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	logs, err := h.service.ListLogs(c.Query("notification_id"), limit)
	if err != nil {
		return helpers.ResponseUtils(c, 500, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, 200, true, "ok", logs)
}

func (h *Handler) Enqueue(c *fiber.Ctx) error {
	var input notifications.NotificationInput
	if err := c.BodyParser(&input); err != nil {
		return helpers.ResponseUtils(c, 400, false, "Invalid request body", nil)
	}
	if err := h.service.Enqueue(input.To, input.TemplateKey, input.Variables, "", ""); err != nil {
		return helpers.ResponseUtils(c, 400, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, 202, true, "Notification queued", nil)
}

func (h *Handler) ListPreferences(c *fiber.Ctx) error {
	preferences, err := h.service.ListPreferences(strings.TrimSpace(c.Query("search")))
	if err != nil {
		return helpers.ResponseUtils(c, 500, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, 200, true, "ok", preferences)
}

func (h *Handler) SetPreference(c *fiber.Ctx) error {
	var input notifications.PreferenceInput
	if err := c.BodyParser(&input); err != nil {
		return helpers.ResponseUtils(c, 400, false, "Invalid request body", nil)
	}
	if input.CustomerID == "" {
		input.CustomerID = c.Params("customerId")
	}
	preference, err := h.service.SetPreference(input)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helpers.ResponseUtils(c, 404, false, "Customer not found", nil)
		}
		return helpers.ResponseUtils(c, 400, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, 200, true, "Preference updated", preference)
}

func (h *Handler) Summary(c *fiber.Ctx) error {
	counts, err := h.service.CountByStatus()
	if err != nil {
		return helpers.ResponseUtils(c, 500, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, 200, true, "ok", fiber.Map{
		"counts":                counts,
		"gateway":               wa.NewWAService().Driver(),
		"rate_limit_per_minute": envInt("WHATSAPP_RATE_LIMIT_PER_MINUTE", 10),
		"max_retries":           envInt("WHATSAPP_MAX_RETRIES", 5),
	})
}

func DeliveryWebhook(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		configuredToken := strings.TrimSpace(os.Getenv("WHATSAPP_STATUS_WEBHOOK_TOKEN"))
		providedToken := strings.TrimSpace(c.Get("X-WhatsApp-Webhook-Token"))
		customTokenValid := configuredToken != "" && len(configuredToken) == len(providedToken) &&
			subtle.ConstantTimeCompare([]byte(configuredToken), []byte(providedToken)) == 1
		if customTokenValid {
			return applyCustomDeliveryStatus(c, db)
		}
		if !verifyMetaSignature(c.Body(), c.Get("X-Hub-Signature-256")) {
			return helpers.ResponseUtils(c, 401, false, "Invalid webhook authentication", nil)
		}
		var payload struct {
			Entry []struct {
				Changes []struct {
					Value struct {
						Statuses []struct {
							ID     string `json:"id"`
							Status string `json:"status"`
							Errors any    `json:"errors"`
						} `json:"statuses"`
					} `json:"value"`
				} `json:"changes"`
			} `json:"entry"`
		}
		if err := json.Unmarshal(c.Body(), &payload); err != nil {
			return helpers.ResponseUtils(c, 400, false, "Invalid Meta webhook body", nil)
		}
		service := notifications.NewService(db)
		for _, entry := range payload.Entry {
			for _, change := range entry.Changes {
				for _, status := range change.Value.Statuses {
					details, _ := json.Marshal(status.Errors)
					if err := service.ApplyDeliveryStatus(status.ID, status.Status, string(details)); err != nil {
						if errors.Is(err, gorm.ErrRecordNotFound) {
							log.Printf("[notifications] Meta status for untracked message %s ignored", status.ID)
							continue
						}
						return helpers.ResponseUtils(c, 500, false, err.Error(), nil)
					}
				}
			}
		}
		return helpers.ResponseUtils(c, 200, true, "Delivery statuses updated", nil)
	}
}

func applyCustomDeliveryStatus(c *fiber.Ctx, db *gorm.DB) error {
	var input struct {
		MessageID string `json:"message_id"`
		Status    string `json:"status"`
		Details   string `json:"details"`
	}
	if err := c.BodyParser(&input); err != nil {
		return helpers.ResponseUtils(c, 400, false, "Invalid request body", nil)
	}
	if strings.TrimSpace(input.MessageID) == "" {
		return helpers.ResponseUtils(c, 400, false, "message_id is required", nil)
	}
	if err := notifications.NewService(db).ApplyDeliveryStatus(input.MessageID, input.Status, input.Details); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return helpers.ResponseUtils(c, 404, false, "Notification not found", nil)
		}
		return helpers.ResponseUtils(c, 400, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, 200, true, "Delivery status updated", nil)
}

func MetaWebhookVerification(c *fiber.Ctx) error {
	verifyToken := strings.TrimSpace(os.Getenv("WHATSAPP_WEBHOOK_VERIFY_TOKEN"))
	providedToken := c.Query("hub.verify_token")
	if c.Query("hub.mode") != "subscribe" || verifyToken == "" || len(verifyToken) != len(providedToken) ||
		subtle.ConstantTimeCompare([]byte(verifyToken), []byte(providedToken)) != 1 {
		return c.SendStatus(403)
	}
	return c.SendString(c.Query("hub.challenge"))
}

func verifyMetaSignature(body []byte, signature string) bool {
	appSecret := strings.TrimSpace(os.Getenv("WHATSAPP_APP_SECRET"))
	if appSecret == "" || !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	provided, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(appSecret))
	if _, err := mac.Write(body); err != nil {
		return false
	}
	return hmac.Equal(provided, mac.Sum(nil))
}

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
