package kuma

import (
	"log"
	"time"

	incidentsapi "skripsi-be/internal/api/admin/incidents"
	"skripsi-be/internal/api/webhook/dude_status"
	"skripsi-be/internal/config/database"
	"skripsi-be/internal/models/entities"
)

type Service interface {
	ProcessWebhook(payload *KumaWebhookPayload, rawBody []byte) (*entities.DeviceStatusLog, int, error)
	GetRecentLogs(limit int) ([]entities.DeviceStatusLog, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) ProcessWebhook(payload *KumaWebhookPayload, rawBody []byte) (*entities.DeviceStatusLog, int, error) {
	monitorID := payload.GetMonitorID()
	monitorName := payload.GetMonitorName()
	status := payload.GetStatus()
	pingMs := payload.GetPingMs()
	msg := payload.GetMessage()

	logEntry := &entities.DeviceStatusLog{
		Source:     "kuma",
		MonitorID:  monitorID,
		DeviceName: monitorName,
		Status:     status,
		PingMs:     pingMs,
		Message:    msg,
		RawPayload: string(rawBody),
		CreatedAt:  time.Now(),
	}

	// 1. Save to device_status_logs
	if err := s.repo.SaveStatusLog(logEntry); err != nil {
		log.Printf("[Kuma Service] Error saving device status log: %v", err)
		return nil, 0, err
	}

	// 2. Update status of corresponding node in network_nodes and dude_device_statuses
	updatedNodes, err := s.repo.UpdateRelatedNodeStatus(monitorID, monitorName, status)
	if err != nil {
		log.Printf("[Kuma Service] Warning updating related nodes: %v", err)
	}

	if status == "down" {
		if db := database.GetDB(); db != nil {
			if _, triggerErr := incidentsapi.TriggerKumaDown(db, monitorName); triggerErr != nil {
				log.Printf("[Kuma Service] Mass incident trigger failed: %v", triggerErr)
			}
		}
	}

	// 3. Broadcast real-time status update to frontend via WebSocket Hub if connected
	if hub := dude_status.GetSharedHub(); hub != nil {
		_ = hub.Broadcast(dude_status.DeviceStatus{
			DeviceName: monitorName,
			Status:     status,
		})
	}

	log.Printf("[Kuma Webhook] Processed monitor '%s' (ID: %s) -> Status: %s, Ping: %.2f ms, Nodes Updated: %d",
		monitorName, monitorID, status, pingMs, updatedNodes)

	return logEntry, updatedNodes, nil
}

func (s *service) GetRecentLogs(limit int) ([]entities.DeviceStatusLog, error) {
	return s.repo.GetRecentLogs(limit)
}
