package kuma

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"skripsi-be/internal/models/entities"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Skipf("SQLite integration test unavailable in this Go environment: %v", err)
		return nil
	}
	if err := db.AutoMigrate(&entities.DeviceStatusLog{}, &entities.DudeDeviceStatus{}); err != nil {
		t.Fatalf("failed to automigrate: %v", err)
	}
	return db
}

func TestKumaWebhookHandler(t *testing.T) {
	db := setupTestDB(t)

	// Seed a test device in dude_device_statuses
	db.Create(&entities.DudeDeviceStatus{
		Name:      "Router Core BGP",
		IPAddress: "10.10.10.1",
		Status:    "up",
	})

	app := fiber.New()
	RegisterRoutes(app.Group("/webhook/kuma"), db)

	// Test 1: Standard Uptime Kuma down alert
	kumaPayload := KumaWebhookPayload{
		Heartbeat: &KumaHeartbeat{
			MonitorID: 42,
			Status:    0, // DOWN
			Time:      "2026-11-07 12:00:00",
			Msg:       "Connection refused",
			Ping:      0.0,
		},
		Monitor: &KumaMonitor{
			ID:   42,
			Name: "Router Core BGP",
		},
		Msg: "[🔴 Down] Router Core BGP",
	}

	body, _ := json.Marshal(kumaPayload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/kuma", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, 3000)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	// Verify log recorded in database
	var count int64
	db.Model(&entities.DeviceStatusLog{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 log entry, got %d", count)
	}

	var savedLog entities.DeviceStatusLog
	db.First(&savedLog)
	if savedLog.DeviceName != "Router Core BGP" || savedLog.Status != "down" {
		t.Fatalf("unexpected log data: %+v", savedLog)
	}

	// Verify node status in dude_device_statuses was updated to "down"
	var updatedDevice entities.DudeDeviceStatus
	db.Where("device_name = ?", "Router Core BGP").First(&updatedDevice)
	if updatedDevice.Status != "down" {
		t.Fatalf("expected device status down, got %s", updatedDevice.Status)
	}

	// Test 2: Query /webhook/kuma/logs
	logReq := httptest.NewRequest(http.MethodGet, "/webhook/kuma/logs", nil)
	logResp, err := app.Test(logReq, 3000)
	if err != nil {
		t.Fatalf("logs request failed: %v", err)
	}
	if logResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", logResp.StatusCode)
	}
}
