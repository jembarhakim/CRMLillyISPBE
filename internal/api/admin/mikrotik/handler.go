package mikrotik

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"skripsi-be/internal/helpers"
	"skripsi-be/internal/services"

	"github.com/gofiber/fiber/v2"
)

type MikroTikHandler struct {
	mikroTikService *services.MikroTikService
}

func NewMikroTikHandler(mikroTikService *services.MikroTikService) *MikroTikHandler {
	return &MikroTikHandler{
		mikroTikService: mikroTikService,
	}
}

// Connect to MikroTik device
func (h *MikroTikHandler) Connect(c *fiber.Ctx) error {
	var req struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Invalid request body", nil)
	}

	// Validate required fields
	if req.Host == "" || req.Port == 0 || req.Username == "" || req.Password == "" {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Host, port, username, and password are required", nil)
	}

	// Create new MikroTik service with provided config
	config := &services.MikroTikConfig{
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
	}

	h.mikroTikService = services.NewMikroTikService(config)

	// Attempt to connect
	if err := h.mikroTikService.Connect(); err != nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to connect to MikroTik: "+err.Error(), nil)
	}

	// expose shared instance for background jobs
	services.SetSharedMikroTikService(h.mikroTikService)

	return helpers.ResponseUtils(c, http.StatusOK, true, "Successfully connected to MikroTik", map[string]interface{}{
		"host":   req.Host,
		"port":   req.Port,
		"status": "connected",
	})
}

// Disconnect from MikroTik device
func (h *MikroTikHandler) Disconnect(c *fiber.Ctx) error {
	if h.mikroTikService == nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Not connected to any MikroTik device", nil)
	}

	if err := h.mikroTikService.Disconnect(); err != nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to disconnect: "+err.Error(), nil)
	}

	return helpers.ResponseUtils(c, http.StatusOK, true, "Successfully disconnected from MikroTik", nil)
}

// Get connection status
func (h *MikroTikHandler) GetStatus(c *fiber.Ctx) error {
	// Always check the shared service (set by auto-connect)
	sharedService := services.GetSharedMikroTikService()
	if sharedService == nil {
		return helpers.ResponseUtils(c, http.StatusOK, true, "Status retrieved", map[string]interface{}{
			"status":  "disconnected",
			"message": "Not connected to any MikroTik device",
			"host":    "",
			"port":    0,
		})
	}

	status := sharedService.GetConnectionStatus()
	return helpers.ResponseUtils(c, http.StatusOK, true, "Status retrieved", status)
}

// Get MikroTik logs (Netwatch logs)
func (h *MikroTikHandler) GetLogs(c *fiber.Ctx) error {
	if h.mikroTikService == nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Not connected to any MikroTik device", nil)
	}

	timeRange := c.Query("timeRange", "1d")

	// Use Netwatch logs instead of general logs
	logs, err := h.mikroTikService.GetNetwatchLogs(timeRange)
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to get netwatch logs: "+err.Error(), nil)
	}

	return helpers.ResponseUtils(c, http.StatusOK, true, "Netwatch logs retrieved successfully", map[string]interface{}{
		"logs":      logs,
		"count":     len(logs),
		"timeRange": timeRange,
	})
}

// Execute custom command on MikroTik
func (h *MikroTikHandler) ExecuteCommand(c *fiber.Ctx) error {
	if h.mikroTikService == nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Not connected to any MikroTik device", nil)
	}

	var req struct {
		Command string `json:"command"`
	}

	if err := c.BodyParser(&req); err != nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Invalid request body", nil)
	}

	if req.Command == "" {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Command is required", nil)
	}

	output, err := h.mikroTikService.ExecuteCommand(req.Command)
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to execute command: "+err.Error(), nil)
	}

	return helpers.ResponseUtils(c, http.StatusOK, true, "Command executed successfully", map[string]interface{}{
		"command": req.Command,
		"output":  output,
	})
}

// Get system information
func (h *MikroTikHandler) GetSystemInfo(c *fiber.Ctx) error {
	if h.mikroTikService == nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Not connected to any MikroTik device", nil)
	}

	info, err := h.mikroTikService.GetSystemInfo()
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to get system info: "+err.Error(), nil)
	}

	return helpers.ResponseUtils(c, http.StatusOK, true, "System information retrieved successfully", info)
}

// Set hotspot IP binding type (for customer isolation)
func (h *MikroTikHandler) SetHotspotIPBindingType(c *fiber.Ctx) error {
	router := services.GetSharedRouterOSAPI()
	if router == nil || !router.IsConnected() {
		return helpers.ResponseUtils(c, http.StatusServiceUnavailable, false, "RouterOS API is not connected", nil)
	}

	var req struct {
		MacAddress string `json:"mac_address" validate:"required"`
		Type       string `json:"type" validate:"required,oneof=regular bypassed"`
	}

	if err := c.BodyParser(&req); err != nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Invalid request body", nil)
	}

	if req.MacAddress == "" || req.Type == "" {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "MAC address and type are required", nil)
	}

	err := router.SetHotspotIPBindingType(req.MacAddress, req.Type)
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to set IP binding type: "+err.Error(), nil)
	}

	return helpers.ResponseUtils(c, http.StatusOK, true, "IP binding type updated successfully", map[string]interface{}{
		"mac_address": req.MacAddress,
		"type":        req.Type,
	})
}

// Get hotspot IP bindings
func (h *MikroTikHandler) GetHotspotIPBindings(c *fiber.Ctx) error {
	router := services.GetSharedRouterOSAPI()
	if router == nil || !router.IsConnected() {
		return helpers.ResponseUtils(c, http.StatusServiceUnavailable, false, "RouterOS API is not connected", nil)
	}

	bindings, err := router.ListHotspotIPBindings()
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to get IP bindings: "+err.Error(), nil)
	}

	return helpers.ResponseUtils(c, http.StatusOK, true, "IP bindings retrieved successfully", bindings)
}

// Get specific hotspot IP binding by MAC address
func (h *MikroTikHandler) GetHotspotIPBindingByMAC(c *fiber.Ctx) error {
	router := services.GetSharedRouterOSAPI()
	if router == nil || !router.IsConnected() {
		return helpers.ResponseUtils(c, http.StatusServiceUnavailable, false, "RouterOS API is not connected", nil)
	}

	macAddress := c.Params("mac")
	if macAddress == "" {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "MAC address is required", nil)
	}

	binding, err := router.GetHotspotIPBindingByMAC(macAddress)
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusNotFound, false, "IP binding not found: "+err.Error(), nil)
	}

	return helpers.ResponseUtils(c, http.StatusOK, true, "IP binding retrieved successfully", binding)
}

// Get Netwatch devices
func (h *MikroTikHandler) GetNetwatchDevices(c *fiber.Ctx) error {
	if h.mikroTikService == nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Not connected to any MikroTik device", nil)
	}

	devices, err := h.mikroTikService.GetNetwatchDevices()
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to get netwatch devices: "+err.Error(), nil)
	}

	return helpers.ResponseUtils(c, http.StatusOK, true, "Netwatch devices retrieved successfully", map[string]interface{}{
		"devices": devices,
		"count":   len(devices),
	})
}

// Get real-time logs with WebSocket support
func (h *MikroTikHandler) GetRealTimeLogs(c *fiber.Ctx) error {
	if h.mikroTikService == nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Not connected to any MikroTik device", nil)
	}

	// This would typically be handled by WebSocket middleware
	// For now, we'll return the latest logs
	timeRange := c.Query("timeRange", "1h")

	logs, err := h.mikroTikService.GetNetwatchLogs(timeRange)
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to get logs: "+err.Error(), nil)
	}

	return helpers.ResponseUtils(c, http.StatusOK, true, "Real-time logs retrieved successfully", map[string]interface{}{
		"logs":      logs,
		"count":     len(logs),
		"timeRange": timeRange,
		"timestamp": time.Now(),
	})
}

// getShiftedMac takes "AA:BB:CC:00:00:01" and returns "AA:BB:CC:00:00:02"
func getShiftedMac(originalMac string) (string, error) {
	// 1. Clean the string (remove colons)
	cleanMac := strings.ReplaceAll(originalMac, ":", "")
	cleanMac = strings.ReplaceAll(cleanMac, "-", "") // just in case

	// 2. Parse Hex to Integer
	val, err := strconv.ParseUint(cleanMac, 16, 64)
	if err != nil {
		return "", err
	}

	// 3. Add 1 (The WAN Port Shift)
	val++

	// 4. Format back to Hex String
	newHex := fmt.Sprintf("%012X", val)

	// 5. Re-add colons (XX:XX:XX...)
	var sb strings.Builder
	for i := 0; i < len(newHex); i++ {
		if i > 0 && i%2 == 0 {
			sb.WriteRune(':')
		}
		sb.WriteByte(newHex[i])
	}
	return sb.String(), nil // Returns standard format (e.g., AA:BB:...)
}

// GetDHCPLease finds the router MAC in DHCP leases based on sticker MAC
func (h *MikroTikHandler) GetDHCPLease(c *fiber.Ctx) error {
	var req struct {
		StickerMac string `json:"sticker_mac"`
	}

	if err := c.BodyParser(&req); err != nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Invalid request body", nil)
	}

	if req.StickerMac == "" {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Sticker MAC address is required", nil)
	}

	// Calculate the Shifted (+1) version
	shiftedMac, err := getShiftedMac(req.StickerMac)
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Invalid MAC address format", nil)
	}

	// Always use the shared MikroTik service (which is set by auto-connect)
	sharedService := services.GetSharedMikroTikService()
	if sharedService == nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "MikroTik service not initialized. Please connect to MikroTik first.", nil)
	}

	// Check if MikroTik service is connected
	if !sharedService.IsConnected() {
		return helpers.ResponseUtils(c, http.StatusServiceUnavailable, false, "MikroTik service not connected. Please connect to MikroTik first.", nil)
	}

	// Execute MikroTik command to get all bound DHCP leases
	cmd := "/ip/dhcp-server/lease/print terse where status=bound"
	output, err := sharedService.ExecuteCommand(cmd)
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to query DHCP leases: "+err.Error(), map[string]interface{}{
			"sticker_mac": req.StickerMac,
			"shifted_mac": shiftedMac,
			"command":     cmd,
			"error":       err.Error(),
		})
	}

	// Find the router MAC in DHCP
	foundMac, foundIP, err := findRouterMacInDHCP(output, req.StickerMac, shiftedMac)
	if err != nil {
		return helpers.ResponseUtils(c, http.StatusNotFound, false, err.Error(), map[string]interface{}{
			"sticker_mac":   req.StickerMac,
			"shifted_mac":   shiftedMac,
			"command":       cmd,
			"raw_output":    output,
			"parsing_debug": "Router not found in active DHCP leases",
		})
	}

	return helpers.ResponseUtils(c, http.StatusOK, true, "Router MAC found in DHCP lease", map[string]interface{}{
		"sticker_mac": req.StickerMac,
		"shifted_mac": shiftedMac,
		"mac_address": foundMac,
		"mac_sticker": req.StickerMac,
		"found_ip":    foundIP,
		"command":     cmd,
		"raw_output":  output,
	})
}

// findRouterMacInDHCP finds the router MAC (sticker or shifted) in DHCP leases
func findRouterMacInDHCP(output string, stickerMac string, shiftedMac string) (string, string, error) {
	// Normalize everything to Upper Case to avoid case-sensitivity issues
	stickerMac = strings.ToUpper(stickerMac)
	shiftedMac = strings.ToUpper(shiftedMac)
	output = strings.ToUpper(output)

	// Split the output into lines to process one by one
	lines := strings.Split(output, "\n")

	// The Filtering Loop
	for _, line := range lines {
		// Ignore empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// CHECK 1: Is it the Sticker MAC?
		// We check for "MAC-ADDRESS=XX..." to ensure we don't match partial strings
		if strings.Contains(line, "MAC-ADDRESS="+stickerMac) {
			// Extract IP address
			ipRegex := regexp.MustCompile(`ADDRESS=([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)`)
			ipMatch := ipRegex.FindStringSubmatch(line)
			if len(ipMatch) > 1 {
				return stickerMac, ipMatch[1], nil
			}
			// No IP found, but MAC matched
			return stickerMac, "", nil
		}

		// CHECK 2: Is it the Shifted MAC?
		if strings.Contains(line, "MAC-ADDRESS="+shiftedMac) {
			// Extract IP address
			ipRegex := regexp.MustCompile(`ADDRESS=([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)`)
			ipMatch := ipRegex.FindStringSubmatch(line)
			if len(ipMatch) > 1 {
				return shiftedMac, ipMatch[1], nil
			}
			// No IP found, but MAC matched
			return shiftedMac, "", nil
		}
	}

	// If loop finishes and nothing returned, the device is not online
	return "", "", fmt.Errorf("router not found (checked %s and %s)", stickerMac, shiftedMac)
}
