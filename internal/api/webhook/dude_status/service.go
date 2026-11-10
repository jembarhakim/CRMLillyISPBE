package dude_status

import (
	"errors"
	"io"
	"math"
	"sort"
	"strings"
	"sync"

	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
)

var ErrInvalidStatus = errors.New("device_name, ip_address, and status are required")
var ErrInvalidDeviceDetails = errors.New("latitude and longitude must both be valid coordinates")

type StatusBroadcaster interface {
	Broadcast(DeviceStatus) error
}

type Service struct {
	repository  StatusRepository
	broadcaster StatusBroadcaster
	pingMu      sync.Mutex
	pingState   PingJobStatus
	pingHost    func(string) (bool, error)
	syncMu      sync.Mutex
	syncState   RouterOSSyncStatus
}

func NewService(repository StatusRepository, broadcaster StatusBroadcaster) *Service {
	return &Service{
		repository:  repository,
		broadcaster: broadcaster,
		pingHost:    pingIP,
	}
}

func (s *Service) ReceiveStatus(status DeviceStatus) error {
	status.DeviceName = strings.TrimSpace(status.DeviceName)
	status.IPAddress = strings.TrimSpace(status.IPAddress)
	status.Status = strings.TrimSpace(status.Status)
	if status.DeviceName == "" || status.IPAddress == "" || status.Status == "" {
		return ErrInvalidStatus
	}

	if err := s.repository.Upsert(status); err != nil {
		return err
	}
	return s.broadcaster.Broadcast(status)
}

func (s *Service) ListDevices() ([]entities.DudeDeviceStatus, error) {
	return s.repository.List()
}

func (s *Service) PreviewCSV(reader io.Reader) (CSVPreview, error) {
	rows, err := parseDudeCSV(reader)
	if err != nil {
		return CSVPreview{}, err
	}

	preview := CSVPreview{Rows: len(rows), Maps: make([]CSVMapPreview, 0)}
	maps := make(map[string]*CSVMapPreview)
	for _, row := range rows {
		if len(row.mapNames) == 0 {
			preview.Skipped++
			continue
		}
		for _, mapName := range row.mapNames {
			entry := maps[mapName]
			if entry == nil {
				entry = &CSVMapPreview{Name: mapName}
				maps[mapName] = entry
			}
			entry.Rows++
			entry.Devices += len(row.devices)
			entry.Skipped += row.skipped
		}
		preview.Skipped += row.skipped
	}
	for _, entry := range maps {
		preview.Maps = append(preview.Maps, *entry)
	}
	sort.Slice(preview.Maps, func(i, j int) bool {
		return preview.Maps[i].Name < preview.Maps[j].Name
	})
	return preview, nil
}

func (s *Service) ImportCSV(reader io.Reader, mapName string) (CSVImportResult, error) {
	mapName = strings.TrimSpace(mapName)
	if mapName == "" {
		return CSVImportResult{}, ErrCSVMapRequired
	}

	rows, err := parseDudeCSV(reader)
	if err != nil {
		return CSVImportResult{}, err
	}

	devicesByIP := make(map[string]entities.DudeDeviceStatus)
	skipped := 0
	for _, row := range rows {
		if !row.hasMap(mapName) {
			continue
		}
		skipped += row.skipped
		for _, device := range row.devices {
			device.MapName = mapName
			devicesByIP[device.IPAddress] = device
		}
	}
	if len(devicesByIP) == 0 {
		return CSVImportResult{}, ErrCSVMapHasNoDevices
	}

	devices := make([]entities.DudeDeviceStatus, 0, len(devicesByIP))
	for _, device := range devicesByIP {
		devices = append(devices, device)
	}
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].IPAddress < devices[j].IPAddress
	})

	result, err := s.repository.ImportDevices(devices)
	if err != nil {
		return CSVImportResult{}, err
	}
	return CSVImportResult{
		ImportResult: result,
		MapName:      mapName,
		Skipped:      skipped,
	}, nil
}

func (s *Service) UpdateDeviceDetails(id string, details UpdateDeviceDetailsRequest) (entities.DudeDeviceStatus, error) {
	details.UserName = strings.TrimSpace(details.UserName)
	details.UserPhone = strings.TrimSpace(details.UserPhone)
	details.Address = strings.TrimSpace(details.Address)
	details.Notes = strings.TrimSpace(details.Notes)

	if (details.Latitude == nil) != (details.Longitude == nil) {
		return entities.DudeDeviceStatus{}, ErrInvalidDeviceDetails
	}
	if details.Latitude != nil && (*details.Latitude < -90 || *details.Latitude > 90 ||
		*details.Longitude < -180 || *details.Longitude > 180 ||
		math.IsNaN(*details.Latitude) || math.IsInf(*details.Latitude, 0) ||
		math.IsNaN(*details.Longitude) || math.IsInf(*details.Longitude, 0)) {
		return entities.DudeDeviceStatus{}, ErrInvalidDeviceDetails
	}

	device, err := s.repository.UpdateDetails(id, details)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return device, gorm.ErrRecordNotFound
	}
	return device, err
}
