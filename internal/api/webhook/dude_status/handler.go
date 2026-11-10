package dude_status

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"skripsi-be/internal/helpers"
	"skripsi-be/internal/services"
)

const maxDudeCSVSize = 5 * 1024 * 1024

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

func (h *Handler) PreviewCSV(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "CSV file is required", nil)
	}
	if fileHeader.Size > maxDudeCSVSize {
		return helpers.ResponseUtils(c, fiber.StatusRequestEntityTooLarge, false, "CSV file exceeds the 5 MiB limit", nil)
	}
	file, err := fileHeader.Open()
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "Failed to read CSV file", nil)
	}
	defer file.Close()

	preview, err := h.service.PreviewCSV(io.LimitReader(file, maxDudeCSVSize+1))
	if err != nil {
		if errors.Is(err, ErrInvalidCSV) {
			return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
		}
		log.Printf("[dude-import] failed to preview CSV: %v", err)
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "Failed to parse CSV: "+err.Error(), nil)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "The Dude CSV preview is ready", preview)
}

func (h *Handler) ImportCSV(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "CSV file is required", nil)
	}
	if fileHeader.Size > maxDudeCSVSize {
		return helpers.ResponseUtils(c, fiber.StatusRequestEntityTooLarge, false, "CSV file exceeds the 5 MiB limit", nil)
	}
	file, err := fileHeader.Open()
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "Failed to read CSV file", nil)
	}
	defer file.Close()

	result, err := h.service.ImportCSV(io.LimitReader(file, maxDudeCSVSize+1), c.FormValue("map_name"))
	if errors.Is(err, ErrCSVMapRequired) || errors.Is(err, ErrCSVMapHasNoDevices) || errors.Is(err, ErrInvalidCSV) {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}
	if err != nil {
		log.Printf("[dude-import] failed to import CSV: %v", err)
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, "Failed to import The Dude CSV", nil)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "The Dude devices imported", result)
}

func (h *Handler) StartRouterOSSync(c *fiber.Ctx) error {
	job, err := h.service.StartRouterOSSync(services.GetSharedRouterOSAPI())
	if errors.Is(err, ErrRouterOSNotConnected) {
		return helpers.ResponseUtils(c, fiber.StatusServiceUnavailable, false, err.Error(), nil)
	}
	if errors.Is(err, ErrNoPingTargets) {
		return helpers.ResponseUtils(c, fiber.StatusConflict, false, err.Error(), nil)
	}
	if errors.Is(err, ErrRouterOSSyncRunning) {
		return helpers.ResponseUtils(c, fiber.StatusConflict, false, err.Error(), job)
	}
	if err != nil {
		log.Printf("[dude-sync] failed to start RouterOS sync: %v", err)
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, "Failed to start RouterOS status sync", nil)
	}
	return helpers.ResponseUtils(c, fiber.StatusAccepted, true, "RouterOS status sync started", job)
}

func (h *Handler) RouterOSSyncStatus(c *fiber.Ctx) error {
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "RouterOS status sync retrieved", h.service.RouterOSSyncStatus())
}

func (h *Handler) StartPingAll(c *fiber.Ctx) error {
	job, err := h.service.StartPingAll()
	if errors.Is(err, ErrNoPingTargets) {
		return helpers.ResponseUtils(c, fiber.StatusConflict, false, err.Error(), nil)
	}
	if errors.Is(err, ErrPingAlreadyRunning) {
		return helpers.ResponseUtils(c, fiber.StatusConflict, false, err.Error(), job)
	}
	if err != nil {
		log.Printf("[dude-ping] failed to start ping-all job: %v", err)
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, "Failed to start ping-all job", nil)
	}
	return helpers.ResponseUtils(c, fiber.StatusAccepted, true, "Ping-all job started", job)
}

func (h *Handler) PingAllStatus(c *fiber.Ctx) error {
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Ping-all job status retrieved", h.service.PingAllStatus())
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
