package notifications

import (
	"fmt"
	"log"
	"time"

	"skripsi-be/internal/api/webhook/wa"
	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func StartWorker(db *gorm.DB) {
	service := NewService(db)
	if err := service.SeedDefaultTemplates(); err != nil {
		log.Printf("[notifications] failed to seed default templates: %v", err)
		return
	}

	perMinute := configuredInt("WHATSAPP_RATE_LIMIT_PER_MINUTE", 10)
	interval := time.Minute / time.Duration(perMinute)
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	log.Printf("[notifications] queue worker started, rate_limit=%d messages/minute", perMinute)

	for range ticker.C {
		if err := processNext(db); err != nil {
			log.Printf("[notifications] queue worker error: %v", err)
		}
	}
}

func processNext(db *gorm.DB) error {
	var notification entities.Notification
	err := db.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("(status = ? AND next_attempt_at <= ?) OR (status = ? AND updated_at <= ?)",
				StatusQueued, time.Now(), StatusSending, time.Now().Add(-2*time.Minute)).
			Order("created_at ASC").
			Limit(1).
			Find(&notification)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		notification.Attempts++
		if err := tx.Model(&notification).Updates(map[string]any{
			"status": StatusSending, "attempts": notification.Attempts, "updated_at": time.Now(),
		}).Error; err != nil {
			return err
		}
		return tx.Create(&entities.NotificationLog{
			NotificationID: notification.ID,
			Status:         StatusSending,
			Attempt:        notification.Attempts,
			Details:        "notification delivery attempt started",
		}).Error
	})
	if err != nil {
		return err
	}
	if notification.ID == "" {
		return nil
	}

	if notification.InvoiceID != nil {
		active, err := NewService(db).CheckInvoiceStillUnpaid(*notification.InvoiceID)
		if err != nil {
			return finishFailed(db, notification, fmt.Errorf("check invoice payment status: %w", err))
		}
		if !active {
			if err := db.Model(&notification).Updates(map[string]any{
				"status": StatusCancelled, "updated_at": time.Now(),
			}).Error; err != nil {
				return err
			}
			if err := NewService(db).Log(notification.ID, StatusCancelled, notification.Attempts, "invoice is already paid; reminder cancelled"); err != nil {
				return err
			}
			return syncBroadcastStatus(db, notification.BroadcastID)
		}
	}

	result, err := wa.NewWAService().SendMessageWithResult(notification.Recipient, notification.Message)
	if err != nil {
		return finishFailed(db, notification, err)
	}
	now := time.Now()
	updates := map[string]any{
		"status": StatusSent, "sent_at": now, "updated_at": now, "last_error": nil,
	}
	if result.MessageID != "" {
		updates["gateway_message_id"] = result.MessageID
	}
	if err := db.Model(&notification).Updates(updates).Error; err != nil {
		return fmt.Errorf("mark notification sent: %w", err)
	}
	details := "WhatsApp gateway accepted the message"
	if result.MessageID != "" {
		details += " (message id: " + result.MessageID + ")"
	}
	if err := NewService(db).Log(notification.ID, StatusSent, notification.Attempts, details); err != nil {
		return err
	}
	return syncBroadcastStatus(db, notification.BroadcastID)
}

func finishFailed(db *gorm.DB, notification entities.Notification, sendErr error) error {
	now := time.Now()
	status := StatusQueued
	updates := map[string]any{
		"status": StatusQueued, "updated_at": now, "last_error": sendErr.Error(),
	}
	details := "send failed; retry scheduled: " + sendErr.Error()
	if notification.Attempts >= notification.MaxAttempts {
		status = StatusFailed
		updates["status"] = StatusFailed
		details = "send failed; retry limit reached: " + sendErr.Error()
	} else {
		delay := 30 * time.Second * time.Duration(1<<(notification.Attempts-1))
		if delay > time.Hour {
			delay = time.Hour
		}
		updates["next_attempt_at"] = now.Add(delay)
	}
	if err := db.Model(&notification).Updates(updates).Error; err != nil {
		return fmt.Errorf("save failed notification state after %q: %w", sendErr, err)
	}
	if err := NewService(db).Log(notification.ID, status, notification.Attempts, details); err != nil {
		return fmt.Errorf("save failed notification log after %q: %w", sendErr, err)
	}
	return syncBroadcastStatus(db, notification.BroadcastID)
}
