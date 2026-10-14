package dude_status

import (
	"errors"
	"math"
	"strings"

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
}

func NewService(repository StatusRepository, broadcaster StatusBroadcaster) *Service {
	return &Service{
		repository:  repository,
		broadcaster: broadcaster,
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
