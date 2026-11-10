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
	ListImported() ([]entities.DudeDeviceStatus, error)
	ImportDevices([]entities.DudeDeviceStatus) (ImportResult, error)
	UpdateDetails(string, UpdateDeviceDetailsRequest) (entities.DudeDeviceStatus, error)
}

type ImportResult struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Total   int `json:"total"`
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

func (r *gormStatusRepository) ListImported() ([]entities.DudeDeviceStatus, error) {
	var devices []entities.DudeDeviceStatus
	err := r.db.Where("map_name <> ''").Order("name ASC").Find(&devices).Error
	return devices, err
}

func (r *gormStatusRepository) ImportDevices(devices []entities.DudeDeviceStatus) (ImportResult, error) {
	result := ImportResult{Total: len(devices)}
	if len(devices) == 0 {
		return result, nil
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		addresses := make([]string, 0, len(devices))
		for _, device := range devices {
			addresses = append(addresses, device.IPAddress)
		}

		var existing []entities.DudeDeviceStatus
		if err := tx.Select("ip_address").Where("ip_address IN ?", addresses).Find(&existing).Error; err != nil {
			return err
		}
		result.Updated = len(existing)
		result.Created = len(devices) - result.Updated

		now := time.Now()
		for i := range devices {
			devices[i].LastSeen = now
		}
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "ip_address"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name",
				"map_name",
				"status",
				"last_seen",
				"updated_at",
			}),
		}).Create(&devices).Error
	})
	return result, err
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
