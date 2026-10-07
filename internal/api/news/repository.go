package news

import (
	"time"

	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListAdmin() ([]entities.News, error) {
	items := []entities.News{}
	err := r.db.Order("created_at DESC").Find(&items).Error
	return items, err
}

func (r *Repository) GetByID(id string) (entities.News, error) {
	var item entities.News
	err := r.db.First(&item, "id = ?", id).Error
	return item, err
}

func (r *Repository) ListPublished(page, limit int) ([]entities.News, int64, error) {
	items := []entities.News{}
	var total int64
	if err := r.db.Model(&entities.News{}).Where("is_published = ?", true).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := r.db.Where("is_published = ?", true).
		Order("published_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&items).Error
	return items, total, err
}

func (r *Repository) GetPublishedByID(id string) (entities.News, error) {
	var item entities.News
	err := r.db.First(&item, "id = ? AND is_published = ?", id, true).Error
	return item, err
}

func (r *Repository) Create(request NewsWriteRequest) (entities.News, error) {
	item := entities.News{
		Title:       request.Title,
		Summary:     request.Summary,
		Content:     request.Content,
		ImageURL:    request.ImageURL,
		IsPublished: request.IsPublished,
	}
	if item.IsPublished {
		now := time.Now()
		item.PublishedAt = &now
	}
	err := r.db.Create(&item).Error
	return item, err
}

func (r *Repository) Update(id string, request NewsWriteRequest) (entities.News, error) {
	item, err := r.GetByID(id)
	if err != nil {
		return entities.News{}, err
	}
	item.Title = request.Title
	item.Summary = request.Summary
	item.Content = request.Content
	item.ImageURL = request.ImageURL
	item.IsPublished = request.IsPublished
	if item.IsPublished && item.PublishedAt == nil {
		now := time.Now()
		item.PublishedAt = &now
	} else if !item.IsPublished {
		item.PublishedAt = nil
	}
	err = r.db.Save(&item).Error
	return item, err
}

func (r *Repository) Delete(id string) error {
	result := r.db.Delete(&entities.News{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
