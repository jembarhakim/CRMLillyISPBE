package dude_status

import (
	"errors"
	"fmt"
	"log"
	"time"

	"skripsi-be/internal/models/entities"
	"skripsi-be/internal/services"
)

var (
	ErrRouterOSSyncRunning  = errors.New("RouterOS status sync is already running")
	ErrRouterOSNotConnected = errors.New("RouterOS API is not connected")
)

type RouterOSSyncStatus struct {
	Enabled         bool      `json:"enabled"`
	Running         bool      `json:"running"`
	IntervalSeconds int       `json:"interval_seconds"`
	Total           int       `json:"total"`
	Completed       int       `json:"completed"`
	Up              int       `json:"up"`
	Down            int       `json:"down"`
	Failed          int       `json:"failed"`
	LastStartedAt   time.Time `json:"last_started_at,omitempty"`
	LastFinishedAt  time.Time `json:"last_finished_at,omitempty"`
	LastError       string    `json:"last_error,omitempty"`
}

func (s *Service) RouterOSSyncStatus() RouterOSSyncStatus {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	return s.syncState
}

func (s *Service) StartRouterOSSync(client *services.RouterOSAPI) (RouterOSSyncStatus, error) {
	if client == nil {
		return RouterOSSyncStatus{}, ErrRouterOSNotConnected
	}
	if err := client.EnsureConnected(); err != nil {
		return RouterOSSyncStatus{}, fmt.Errorf("%w: %v", ErrRouterOSNotConnected, err)
	}

	devices, err := s.repository.ListImported()
	if err != nil {
		return RouterOSSyncStatus{}, fmt.Errorf("list imported devices for RouterOS sync: %w", err)
	}
	if len(devices) == 0 {
		return RouterOSSyncStatus{}, ErrNoPingTargets
	}

	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	if s.syncState.Running {
		return s.syncState, ErrRouterOSSyncRunning
	}
	s.syncState = RouterOSSyncStatus{
		Enabled:         true,
		Running:         true,
		IntervalSeconds: s.syncState.IntervalSeconds,
		Total:           len(devices),
		LastStartedAt:   time.Now(),
	}
	state := s.syncState
	go s.runRouterOSSync(client, devices)
	return state, nil
}

func (s *Service) StartRouterOSSyncLoop(interval time.Duration, clientProvider func() *services.RouterOSAPI) {
	if interval <= 0 {
		interval = time.Minute
	}
	s.syncMu.Lock()
	s.syncState.Enabled = true
	s.syncState.IntervalSeconds = int(interval.Seconds())
	s.syncMu.Unlock()

	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			s.startScheduledRouterOSSync(clientProvider)
		case <-ticker.C:
			s.startScheduledRouterOSSync(clientProvider)
		}
	}
}

func (s *Service) startScheduledRouterOSSync(clientProvider func() *services.RouterOSAPI) {
	client := clientProvider()
	if client == nil {
		s.recordRouterOSSyncFailure(ErrRouterOSNotConnected)
		return
	}
	if _, err := s.StartRouterOSSync(client); err != nil && !errors.Is(err, ErrRouterOSSyncRunning) {
		s.recordRouterOSSyncFailure(err)
	}
}

func (s *Service) runRouterOSSync(client *services.RouterOSAPI, devices []entities.DudeDeviceStatus) {
	var firstErr error
	for _, device := range devices {
		alive, err := client.Ping(device.IPAddress)
		if err != nil {
			log.Printf("[dude-sync] failed to check %q (%s): %v", device.Name, device.IPAddress, err)
			s.recordRouterOSSyncResult(false, false, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		status := "down"
		if alive {
			status = "up"
		}
		if err := s.ReceiveStatus(DeviceStatus{
			DeviceName: device.Name,
			IPAddress:  device.IPAddress,
			Status:     status,
		}); err != nil {
			log.Printf("[dude-sync] failed to persist status for %q (%s): %v", device.Name, device.IPAddress, err)
			s.recordRouterOSSyncResult(false, false, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		s.recordRouterOSSyncResult(alive, true, nil)
	}

	s.syncMu.Lock()
	s.syncState.Running = false
	s.syncState.LastFinishedAt = time.Now()
	if firstErr != nil {
		s.syncState.LastError = firstErr.Error()
	}
	s.syncMu.Unlock()
}

func (s *Service) recordRouterOSSyncFailure(err error) {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	s.syncState.LastError = err.Error()
	s.syncState.LastFinishedAt = time.Now()
}

func (s *Service) recordRouterOSSyncResult(alive, success bool, err error) {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	s.syncState.Completed++
	if !success {
		s.syncState.Failed++
		if err != nil && s.syncState.LastError == "" {
			s.syncState.LastError = err.Error()
		}
		return
	}
	if alive {
		s.syncState.Up++
	} else {
		s.syncState.Down++
	}
}
