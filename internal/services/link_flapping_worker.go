package services

import (
	"log"
	"time"

	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
)

func StartLinkFlappingWorker(db *gorm.DB) {
	go func() {
		var previous map[string]string
		disconnects := make(map[string][]time.Time)

		run := func() {
			mt := GetSharedMikroTikService()
			if mt == nil {
				return
			}
			sessions, err := mt.GetActivePPPoESessions(db)
			if err != nil {
				return
			}

			current := make(map[string]string)
			for _, session := range sessions {
				if session.IsMatched && session.CustomerID != "" {
					current[session.Name] = session.CustomerID
				}
			}

			if previous != nil {
				now := time.Now()
				for name, customerID := range previous {
					if _, active := current[name]; !active {
						cutoff := now.Add(-15 * time.Minute)
						filtered := disconnects[customerID][:0]
						for _, at := range disconnects[customerID] {
							if at.After(cutoff) {
								filtered = append(filtered, at)
							}
						}
						filtered = append(filtered, now)
						disconnects[customerID] = filtered

						status := "stable"
						count := len(filtered)
						windowStart := now.Add(-15 * time.Minute)
						if count > 3 {
							status = "link_flapping"
						}
						linkStatus := entities.LinkFlappingStatus{
							CustomerID: customerID, Status: status,
							DisconnectCount: count, WindowStart: windowStart,
						}
						if status == "link_flapping" {
							last := now
							linkStatus.LastDetectedAt = &last
						}
						if err := db.Save(&linkStatus).Error; err != nil {
							log.Printf("[link-flapping] failed to save status for %s: %v", customerID, err)
						}
					}
				}
			}

			previous = make(map[string]string, len(current))
			for name, customerID := range current {
				previous[name] = customerID
			}
		}

		run()
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			run()
		}
	}()
}
