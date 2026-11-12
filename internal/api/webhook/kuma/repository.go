package kuma

import (
	"log"
	"strings"
	"time"

	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
)

type Repository interface {
	SaveStatusLog(logEntry *entities.DeviceStatusLog) error
	UpdateRelatedNodeStatus(monitorID string, name string, status string) (int, error)
	GetRecentLogs(limit int) ([]entities.DeviceStatusLog, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) SaveStatusLog(logEntry *entities.DeviceStatusLog) error {
	return r.db.Create(logEntry).Error
}

func (r *repository) UpdateRelatedNodeStatus(monitorID string, name string, status string) (int, error) {
	updatedCount := 0
	cleanName := strings.TrimSpace(name)
	linkStatus := entities.LinkStatusUp
	if status == "down" {
		linkStatus = entities.LinkStatusDown
	} else if status == "degraded" || status == "pending" {
		linkStatus = entities.LinkStatusDegraded
	}

	// 1. Update matching network_nodes
	if cleanName != "" {
		res := r.db.Model(&entities.NetworkNode{}).
			Where("LOWER(name) = LOWER(?) OR LOWER(code) = LOWER(?) OR id = ?", cleanName, cleanName, monitorID).
			Updates(map[string]interface{}{
				"status":     linkStatus,
				"updated_at": time.Now(),
			})
		if res.Error == nil && res.RowsAffected > 0 {
			updatedCount += int(res.RowsAffected)
			log.Printf("[Kuma Webhook] Updated %d network_nodes for '%s' to %s", res.RowsAffected, cleanName, linkStatus)
		}
	}

	// 2. Update matching dude_device_statuses
	if cleanName != "" {
		resDude := r.db.Model(&entities.DudeDeviceStatus{}).
			Where("LOWER(device_name) = LOWER(?) OR LOWER(device_name) LIKE ?", cleanName, "%"+cleanName+"%").
			Updates(map[string]interface{}{
				"status":     status,
				"last_seen":  time.Now(),
				"updated_at": time.Now(),
			})
		if resDude.Error == nil && resDude.RowsAffected > 0 {
			updatedCount += int(resDude.RowsAffected)
			log.Printf("[Kuma Webhook] Updated %d dude_device_statuses for '%s' to %s", resDude.RowsAffected, cleanName, status)
		}
	}

	// 3. Update matching netwatch_devices
	if cleanName != "" {
		resNetwatch := r.db.Model(&entities.NetwatchDevice{}).
			Where("LOWER(name) = LOWER(?) OR LOWER(name) LIKE ?", cleanName, "%"+cleanName+"%").
			Updates(map[string]interface{}{
				"status":     status,
				"last_seen":  time.Now(),
				"updated_at": time.Now(),
			})
		if resNetwatch.Error == nil && resNetwatch.RowsAffected > 0 {
			updatedCount += int(resNetwatch.RowsAffected)
		}
	}

	return updatedCount, nil
}

func (r *repository) GetRecentLogs(limit int) ([]entities.DeviceStatusLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var logs []entities.DeviceStatusLog
	err := r.db.Order("created_at DESC").Limit(limit).Find(&logs).Error
	return logs, err
}
