package dude_status

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/go-ping/ping"
	"skripsi-be/internal/models/entities"
)

const (
	pingWorkerCount = 32
	pingTimeout     = 2 * time.Second
)

var (
	ErrPingAlreadyRunning = errors.New("a ping-all job is already running")
	ErrNoPingTargets      = errors.New("no devices with IP addresses are registered; import a network-map CSV first")
)

type PingJobStatus struct {
	Running    bool      `json:"running"`
	Total      int       `json:"total"`
	Completed  int       `json:"completed"`
	Up         int       `json:"up"`
	Down       int       `json:"down"`
	Failed     int       `json:"failed"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
}

func (s *Service) StartPingAll() (PingJobStatus, error) {
	s.pingMu.Lock()
	defer s.pingMu.Unlock()

	if s.pingState.Running {
		return s.pingState, ErrPingAlreadyRunning
	}

	devices, err := s.repository.List()
	if err != nil {
		return PingJobStatus{}, fmt.Errorf("failed to list registered devices for ping: %w", err)
	}
	if len(devices) == 0 {
		return PingJobStatus{}, ErrNoPingTargets
	}

	s.pingState = PingJobStatus{
		Running:   true,
		Total:     len(devices),
		StartedAt: time.Now(),
	}
	go s.runPingAll(devices)
	return s.pingState, nil
}

func (s *Service) PingAllStatus() PingJobStatus {
	s.pingMu.Lock()
	defer s.pingMu.Unlock()
	return s.pingState
}

func (s *Service) runPingAll(devices []entities.DudeDeviceStatus) {
	tasks := make(chan entities.DudeDeviceStatus)
	var workers sync.WaitGroup
	workerCount := min(pingWorkerCount, len(devices))
	workers.Add(workerCount)

	for range workerCount {
		go func() {
			defer workers.Done()
			for device := range tasks {
				s.checkDevice(device)
			}
		}()
	}

	for _, device := range devices {
		tasks <- device
	}
	close(tasks)
	workers.Wait()

	s.pingMu.Lock()
	s.pingState.Running = false
	s.pingState.FinishedAt = time.Now()
	s.pingMu.Unlock()
}

func (s *Service) checkDevice(device entities.DudeDeviceStatus) {
	ip := net.ParseIP(device.IPAddress)
	if ip == nil {
		log.Printf("[dude-ping] skipped device %q: invalid IP address %q", device.Name, device.IPAddress)
		s.recordPingResult(false, false)
		return
	}

	alive, err := s.pingHost(ip.String())
	if err != nil {
		log.Printf("[dude-ping] failed to check device %q (%s): %v", device.Name, ip, err)
		s.recordPingResult(false, false)
		return
	}

	status := "down"
	if alive {
		status = "up"
	}
	if err := s.ReceiveStatus(DeviceStatus{
		DeviceName: device.Name,
		IPAddress:  ip.String(),
		Status:     status,
	}); err != nil {
		log.Printf("[dude-ping] failed to persist status for device %q (%s): %v", device.Name, ip, err)
		s.recordPingResult(false, false)
		return
	}
	s.recordPingResult(alive, true)
}

func (s *Service) recordPingResult(alive, success bool) {
	s.pingMu.Lock()
	defer s.pingMu.Unlock()
	s.pingState.Completed++
	if !success {
		s.pingState.Failed++
	} else if alive {
		s.pingState.Up++
	} else {
		s.pingState.Down++
	}
}

func pingIP(ip string) (bool, error) {
	probe, err := ping.NewPinger(ip)
	if err != nil {
		return false, fmt.Errorf("create ICMP probe: %w", err)
	}
	probe.SetPrivileged(false)
	probe.Count = 1
	probe.Timeout = pingTimeout
	if err := probe.Run(); err != nil {
		return false, fmt.Errorf("run ICMP probe: %w", err)
	}
	return probe.Statistics().PacketsRecv > 0, nil
}
