package dude_status

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"skripsi-be/internal/helpers"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ReceiveStatus(c *fiber.Ctx) error {
	var status DeviceStatus
	if err := json.Unmarshal(c.Body(), &status); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid JSON request body",
		})
	}

	if err := h.service.ReceiveStatus(status); err != nil {
		if errors.Is(err, ErrInvalidStatus) {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": err.Error(),
			})
		}
		log.Printf("[dude-webhook] failed to process device status: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to process device status",
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    status,
	})
}

func (h *Handler) ListDevices(c *fiber.Ctx) error {
	devices, err := h.service.ListDevices()
	if err != nil {
		log.Printf("[dude-devices] failed to list devices: %v", err)
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, "Failed to load The Dude devices", nil)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "The Dude devices loaded", devices)
}

func (h *Handler) UpdateDeviceDetails(c *fiber.Ctx) error {
	var details UpdateDeviceDetailsRequest
	if err := c.BodyParser(&details); err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "Invalid request body", nil)
	}

	device, err := h.service.UpdateDeviceDetails(c.Params("id"), details)
	if errors.Is(err, ErrInvalidDeviceDetails) {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return helpers.ResponseUtils(c, fiber.StatusNotFound, false, "The Dude device not found", nil)
	}
	if err != nil {
		log.Printf("[dude-devices] failed to update device details: %v", err)
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, "Failed to update device details", nil)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Device details updated", device)
}
