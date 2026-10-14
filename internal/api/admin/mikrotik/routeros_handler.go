package mikrotik

import (
	"log"
	"net/http"
	"sync"

	"skripsi-be/internal/helpers"
	"skripsi-be/internal/services"

	"github.com/gofiber/fiber/v2"
)

type RouterOSHandler struct {
	mu     sync.RWMutex
	client *services.RouterOSAPI
}

func NewRouterOSHandler() *RouterOSHandler {
	return &RouterOSHandler{}
}

func (h *RouterOSHandler) Connect(c *fiber.Ctx) error {
	var req struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		UseTLS   bool   `json:"use_tls"`
	}
	if err := c.BodyParser(&req); err != nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Invalid request body", nil)
	}

	client, err := services.ConnectRouterOSAPI(services.RouterOSAPIConfig{
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
		UseTLS:   req.UseTLS,
	})
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusBadGateway, false, err.Error(), nil)
	}

	h.mu.Lock()
	oldClient := h.client
	h.client = client
	h.mu.Unlock()
	if oldClient != nil {
		if err := oldClient.Close(); err != nil {
			log.Printf("[mikrotik-api] failed to close replaced RouterOS connection: %v", err)
		}
	}

	return helpers.ResponseUtils(c, http.StatusOK, true, "Connected to RouterOS API", fiber.Map{
		"host":    req.Host,
		"port":    req.Port,
		"use_tls": req.UseTLS,
		"status":  "connected",
	})
}

func (h *RouterOSHandler) Disconnect(c *fiber.Ctx) error {
	h.mu.Lock()
	client := h.client
	h.client = nil
	h.mu.Unlock()
	if client == nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "RouterOS API is not connected", nil)
	}
	if err := client.Close(); err != nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to disconnect from RouterOS API: "+err.Error(), nil)
	}
	return helpers.ResponseUtils(c, http.StatusOK, true, "Disconnected from RouterOS API", nil)
}

func (h *RouterOSHandler) Status(c *fiber.Ctx) error {
	client := h.getClient()
	status := "disconnected"
	var host string
	if client != nil && client.IsConnected() {
		status = "connected"
		host = client.Host()
	}
	return helpers.ResponseUtils(c, http.StatusOK, true, "RouterOS API status retrieved", fiber.Map{
		"status": status,
		"host":   host,
	})
}

func (h *RouterOSHandler) ListDudeDevices(c *fiber.Ctx) error {
	client, err := h.requireClient(c)
	if err != nil {
		return err
	}
	devices, err := client.Command("/dude/device/print")
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusBadGateway, false, "Failed to read The Dude devices: "+err.Error(), nil)
	}
	return helpers.ResponseUtils(c, http.StatusOK, true, "The Dude devices retrieved", fiber.Map{
		"items": devices,
		"count": len(devices),
	})
}

func (h *RouterOSHandler) ListDudeServices(c *fiber.Ctx) error {
	client, err := h.requireClient(c)
	if err != nil {
		return err
	}
	servicesList, err := client.Command("/dude/service/print")
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusBadGateway, false, "Failed to read The Dude services: "+err.Error(), nil)
	}
	return helpers.ResponseUtils(c, http.StatusOK, true, "The Dude services retrieved", fiber.Map{
		"items": servicesList,
		"count": len(servicesList),
	})
}

func (h *RouterOSHandler) requireClient(c *fiber.Ctx) (*services.RouterOSAPI, error) {
	client := h.getClient()
	if client == nil || !client.IsConnected() {
		return nil, helpers.ResponseUtils(c, http.StatusServiceUnavailable, false, "RouterOS API is not connected", nil)
	}
	return client, nil
}

func (h *RouterOSHandler) getClient() *services.RouterOSAPI {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.client
}
