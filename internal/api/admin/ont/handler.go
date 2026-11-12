package ont

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// GetMapStatusHandler returns summarized ONT status for all customers on the map
// GET /api/map/ont-status or GET /api/admin/ont/map-status
func (h *Handler) GetMapStatusHandler(c *fiber.Ctx) error {
	ctx := c.Context()
	statuses, err := h.service.GetMapStatuses(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to retrieve ONT map statuses: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   statuses,
		"count":  len(statuses),
	})
}

// GetCustomerOntHandler returns full ONT details for a specific customer
// GET /api/customers/:id/ont or GET /api/admin/customers/:id/ont
func (h *Handler) GetCustomerOntHandler(c *fiber.Ctx) error {
	customerID := c.Params("id")
	if customerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Customer ID is required",
		})
	}

	ctx := c.Context()
	ontStatus, err := h.service.GetCustomerOnt(ctx, customerID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   ontStatus,
	})
}

// GetCustomerOntHistoryHandler returns optical history records for graphing
// GET /api/customers/:id/ont/history or GET /api/admin/customers/:id/ont/history
func (h *Handler) GetCustomerOntHistoryHandler(c *fiber.Ctx) error {
	customerID := c.Params("id")
	if customerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Customer ID is required",
		})
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	ctx := c.Context()
	history, err := h.service.GetCustomerHistory(ctx, customerID, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to fetch ONT history: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   history,
		"count":  len(history),
	})
}

// RefreshCustomerOntHandler forces a real-time refresh of ONT parameters via TR-069
// POST /api/customers/:id/ont/refresh or POST /api/admin/customers/:id/ont/refresh
func (h *Handler) RefreshCustomerOntHandler(c *fiber.Ctx) error {
	customerID := c.Params("id")
	if customerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Customer ID is required",
		})
	}

	ctx := c.Context()
	status, err := h.service.RefreshCustomerOnt(ctx, customerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to refresh ONT data: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "ONT data successfully refreshed from device",
		"data":    status,
	})
}

