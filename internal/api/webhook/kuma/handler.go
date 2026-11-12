package kuma

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// HandleWebhook processes incoming POST requests from Uptime Kuma
func (h *Handler) HandleWebhook(c *fiber.Ctx) error {
	rawBody := c.Body()

	var payload KumaWebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		log.Printf("[Kuma Handler] Failed to parse JSON webhook: %v, raw: %s", err, string(rawBody))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid JSON payload",
			"error":   err.Error(),
		})
	}

	logEntry, updatedNodes, err := h.service.ProcessWebhook(&payload, rawBody)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to process Kuma webhook event",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":       true,
		"message":       "Kuma webhook event processed successfully",
		"device_name":   logEntry.DeviceName,
		"status":        logEntry.Status,
		"ping_ms":       logEntry.PingMs,
		"nodes_updated": updatedNodes,
		"log_id":        logEntry.ID,
	})
}

// GetLogs returns the latest device status logs recorded from Kuma
func (h *Handler) GetLogs(c *fiber.Ctx) error {
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	logs, err := h.service.GetRecentLogs(limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to retrieve status logs",
			"error":   err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    logs,
		"total":   len(logs),
	})
}
