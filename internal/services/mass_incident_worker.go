package services

import (
	"log"
	"time"

	incidentsapi "skripsi-be/internal/api/admin/incidents"
	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
)

func StartMassIncidentWorker(db *gorm.DB) {
	go func() {
		observations := make(map[string]map[string][]time.Time)
		run := func() {
			mt := GetSharedMikroTikService()
			if mt == nil {
				return
			}
			sessions, err := mt.GetActivePPPoESessions(db)
			if err != nil {
				return
			}
			active := make(map[string]bool)
			for _, session := range sessions {
				if session.IsMatched && session.CustomerID != "" {
					active[session.CustomerID] = true
				}
			}

			var links []entities.CustomerNetworkLink
			if err := db.Preload("ODPNode").Find(&links).Error; err != nil {
				log.Printf("[mass-incident] failed to load ODP customer links: %v", err)
				return
			}
			byODP := make(map[string][]entities.CustomerNetworkLink)
			for _, link := range links {
				byODP[link.ODPNodeID] = append(byODP[link.ODPNodeID], link)
			}

			now := time.Now()
			cutoff := now.Add(-3 * time.Minute)
			for odpID, odpLinks := range byODP {
				if observations[odpID] == nil {
					observations[odpID] = make(map[string][]time.Time)
				}
				for _, link := range odpLinks {
					if !active[link.CustomerID] {
						observations[odpID][link.CustomerID] = append(observations[odpID][link.CustomerID], now)
					}
				}
				unique := make(map[string]bool)
				uniqueTimes := make(map[string][]time.Time)
				for customerID, times := range observations[odpID] {
					filtered := times[:0]
					for _, at := range times {
						if at.After(cutoff) {
							filtered = append(filtered, at)
						}
					}
					if len(filtered) > 0 {
						unique[customerID] = true
						uniqueTimes[customerID] = filtered
					}
				}
				observations[odpID] = uniqueTimes
				if len(unique) < 5 {
					continue
				}
				var odp entities.NetworkNode
				if err := db.First(&odp, "id = ? AND type = ?", odpID, entities.NodeTypeODP).Error; err != nil {
					continue
				}
				customerIDs := make([]string, 0, len(unique))
				for customerID := range unique {
					customerIDs = append(customerIDs, customerID)
				}
				if _, err := incidentsapi.TriggerODPDown(db, odpID, odp.Name, customerIDs); err != nil {
					log.Printf("[mass-incident] failed to trigger ODP incident: %v", err)
				}
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
