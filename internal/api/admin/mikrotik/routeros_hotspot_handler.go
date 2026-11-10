package mikrotik

import (
	"log"
	"net/http"
	"runtime/debug"
	"strings"

	"skripsi-be/internal/helpers"
	"skripsi-be/internal/services"

	"github.com/gofiber/fiber/v2"
)

func (h *RouterOSHandler) ListHotspotUsers(c *fiber.Ctx) error {
	client, err := h.requireClient(c)
	if err != nil {
		return err
	}
	users, err := client.ListHotspotUsers()
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusBadGateway, false, "Failed to list Hotspot users: "+err.Error(), nil)
	}
	return helpers.ResponseUtils(c, http.StatusOK, true, "Hotspot users retrieved", fiber.Map{
		"items": users,
		"count": len(users),
	})
}

func (h *RouterOSHandler) ListHotspotUserProfiles(c *fiber.Ctx) error {
	client, err := h.requireClient(c)
	if err != nil {
		return err
	}
	profiles, err := client.ListHotspotUserProfiles()
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusBadGateway, false, "Failed to list Hotspot profiles: "+err.Error(), nil)
	}
	return helpers.ResponseUtils(c, http.StatusOK, true, "Hotspot profiles retrieved", fiber.Map{
		"items": profiles,
		"count": len(profiles),
	})
}

func (h *RouterOSHandler) ListHotspotActiveSessions(c *fiber.Ctx) error {
	client, err := h.requireClient(c)
	if err != nil {
		return err
	}
	sessions, err := client.ListHotspotActiveSessions()
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusBadGateway, false, "Failed to list active Hotspot sessions: "+err.Error(), nil)
	}
	return helpers.ResponseUtils(c, http.StatusOK, true, "Active Hotspot sessions retrieved", fiber.Map{
		"items": sessions,
		"count": len(sessions),
	})
}

func (h *RouterOSHandler) ResolveHotspotDeviceBindings(c *fiber.Ctx) (handlerErr error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("[mikrotik-api] panic while resolving Hotspot device bindings: %v\n%s", recovered, debug.Stack())
			handlerErr = helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Unexpected server error while resolving Hotspot device bindings; backend stack trace was logged", nil)
		}
	}()
	client, err := h.requireClient(c)
	if err != nil {
		return err
	}
	var req struct {
		Addresses []string `json:"addresses"`
	}
	if err := c.BodyParser(&req); err != nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Invalid request body", nil)
	}
	if len(req.Addresses) > 200 {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "At most 200 device addresses can be resolved at once", nil)
	}
	resolutions, err := client.ResolveHotspotDeviceBindings(req.Addresses)
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusBadGateway, false, "Failed to resolve device IP bindings: "+err.Error(), nil)
	}
	return helpers.ResponseUtils(c, http.StatusOK, true, "Device IP bindings resolved", fiber.Map{
		"items": resolutions,
		"count": len(resolutions),
	})
}

func (h *RouterOSHandler) SetHotspotDeviceBindingType(c *fiber.Ctx) error {
	client, err := h.requireClient(c)
	if err != nil {
		return err
	}
	var req struct {
		Address string `json:"address"`
		Type    string `json:"type"`
	}
	if err := c.BodyParser(&req); err != nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Invalid request body", nil)
	}
	req.Address = strings.TrimSpace(req.Address)
	if req.Address == "" {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Device IP address is required", nil)
	}
	if err := client.SetHotspotDeviceBindingType(req.Address, req.Type); err != nil {
		return helpers.ResponseUtils(c, http.StatusBadGateway, false, "Failed to update device isolation: "+err.Error(), nil)
	}
	return helpers.ResponseUtils(c, http.StatusOK, true, "Device isolation updated", fiber.Map{
		"address": req.Address,
		"type":    strings.ToLower(strings.TrimSpace(req.Type)),
	})
}

func (h *RouterOSHandler) CreateHotspotUser(c *fiber.Ctx) error {
	client, err := h.requireClient(c)
	if err != nil {
		return err
	}
	var req struct {
		Name     string `json:"name"`
		Password string `json:"password"`
		Profile  string `json:"profile"`
		Comment  string `json:"comment"`
	}
	if err := c.BodyParser(&req); err != nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Invalid request body", nil)
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || strings.TrimSpace(req.Password) == "" {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Username and password are required", nil)
	}
	if err := client.CreateHotspotUser(req.Name, req.Password, req.Profile, req.Comment); err != nil {
		return helpers.ResponseUtils(c, http.StatusBadGateway, false, "Failed to create Hotspot user: "+err.Error(), nil)
	}
	return helpers.ResponseUtils(c, http.StatusCreated, true, "Hotspot user created", fiber.Map{"name": req.Name})
}

func (h *RouterOSHandler) UpdateHotspotUser(c *fiber.Ctx) error {
	client, err := h.requireClient(c)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(c.Params("name"))
	if name == "" {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Hotspot username is required", nil)
	}
	var req services.HotspotUserInput
	if err := c.BodyParser(&req); err != nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Invalid request body", nil)
	}
	if err := client.UpdateHotspotUser(name, req); err != nil {
		return helpers.ResponseUtils(c, http.StatusBadGateway, false, "Failed to update Hotspot user: "+err.Error(), nil)
	}
	return helpers.ResponseUtils(c, http.StatusOK, true, "Hotspot user updated", fiber.Map{"name": name})
}

func (h *RouterOSHandler) DeleteHotspotUser(c *fiber.Ctx) error {
	client, err := h.requireClient(c)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(c.Params("name"))
	if name == "" {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Hotspot username is required", nil)
	}
	if err := client.DeleteHotspotUser(name); err != nil {
		return helpers.ResponseUtils(c, http.StatusBadGateway, false, "Failed to delete Hotspot user: "+err.Error(), nil)
	}
	return helpers.ResponseUtils(c, http.StatusOK, true, "Hotspot user deleted", fiber.Map{"name": name})
}

func (h *RouterOSHandler) KickHotspotUser(c *fiber.Ctx) error {
	client, err := h.requireClient(c)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(c.Params("name"))
	if name == "" {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Hotspot username is required", nil)
	}
	kicked, err := client.KickHotspotUser(name)
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusBadGateway, false, "Failed to disconnect Hotspot user: "+err.Error(), nil)
	}
	return helpers.ResponseUtils(c, http.StatusOK, true, "Hotspot sessions disconnected", fiber.Map{
		"name":            name,
		"sessions_kicked": kicked,
	})
}
