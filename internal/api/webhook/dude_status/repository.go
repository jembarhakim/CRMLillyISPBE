package dude_status

import (
	"time"

	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StatusRepository interface {
	Upsert(DeviceStatus) error
	List() ([]entities.DudeDeviceStatus, error)
	UpdateDetails(string, UpdateDeviceDetailsRequest) (entities.DudeDeviceStatus, error)
}

type gormStatusRepository struct {
	db *gorm.DB
}

func NewStatusRepository(db *gorm.DB) StatusRepository {
	return &gormStatusRepository{db: db}
}

func (r *gormStatusRepository) Upsert(status DeviceStatus) error {
	now := time.Now()
	device := entities.DudeDeviceStatus{
		Name:      status.DeviceName,
		IPAddress: status.IPAddress,
		Status:    status.Status,
		LastSeen:  now,
	}

	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ip_address"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name",
			"status",
			"last_seen",
			"updated_at",
		}),
	}).Create(&device).Error
}

func (r *gormStatusRepository) List() ([]entities.DudeDeviceStatus, error) {
	var devices []entities.DudeDeviceStatus
	err := r.db.Order("name ASC").Find(&devices).Error
	return devices, err
}

func (r *gormStatusRepository) UpdateDetails(id string, details UpdateDeviceDetailsRequest) (entities.DudeDeviceStatus, error) {
	var device entities.DudeDeviceStatus
	if err := r.db.First(&device, "id = ?", id).Error; err != nil {
		return device, err
	}

	err := r.db.Model(&device).Updates(map[string]interface{}{
		"user_name":  details.UserName,
		"user_phone": details.UserPhone,
		"address":    details.Address,
		"notes":      details.Notes,
		"latitude":   details.Latitude,
		"longitude":  details.Longitude,
	}).Error
	if err != nil {
		return device, err
	}

	err = r.db.First(&device, "id = ?", id).Error
	return device, err
}
