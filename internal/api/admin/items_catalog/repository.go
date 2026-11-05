package items_catalog

import (
	"skripsi-be/internal/models/entities"

	"github.com/jinzhu/copier"
	"gorm.io/gorm"
)

type ItemRepositoryInterface interface {
	FindItemsRepository(request GetItemsRequest) ([]entities.Item, error)
	CreateItemRepository(request CreateItemRequest) (entities.Item, error)
	FindByIdItemRepository(request IdItemRequest) (entities.Item, error)
	UpdateItemRepository(request UpdateItemRequest) (entities.Item, error)
	DeleteItemRepository(request IdItemRequest) (entities.Item, error)
}

type ItemRepositoryStruct struct {
	db *gorm.DB
}

func NewItemRepository(db *gorm.DB) *ItemRepositoryStruct {
	return &ItemRepositoryStruct{db}
}

func (r ItemRepositoryStruct) FindItemsRepository(request GetItemsRequest) ([]entities.Item, error) {
	items := []entities.Item{}
	query := r.db.Preload("Asset")

	if request.Category != nil && *request.Category != "" {
		query = query.Where("category = ?", *request.Category)
	}
	if request.AssetID != nil && *request.AssetID != "" {
		query = query.Where("asset_id = ?", *request.AssetID)
	}

	tx := query.Order("name ASC").Find(&items)
	if tx.Error != nil {
		return items, tx.Error
	}

	return items, nil
}

func (r ItemRepositoryStruct) CreateItemRepository(request CreateItemRequest) (entities.Item, error) {
	item := entities.Item{}
	copier.Copy(&item, &request)

	tx := r.db.Create(&item)
	if tx.Error != nil {
		return item, tx.Error
	}

	// Reload with relations
	r.db.Preload("Asset").First(&item, "id = ?", item.ID)

	return item, nil
}

func (r ItemRepositoryStruct) FindByIdItemRepository(request IdItemRequest) (entities.Item, error) {
	item := entities.Item{}
	tx := r.db.Preload("Asset").First(&item, "id = ?", request.Id)
	if tx.Error != nil {
		return item, tx.Error
	}
	return item, nil
}

func (r ItemRepositoryStruct) UpdateItemRepository(request UpdateItemRequest) (entities.Item, error) {
	item := entities.Item{}
	tx := r.db.First(&item, "id = ?", request.Id)
	if tx.Error != nil {
		return item, tx.Error
	}

	copier.Copy(&item, &request.CreateItemRequest)

	tx = r.db.Save(&item)
	if tx.Error != nil {
		return item, tx.Error
	}

	// Reload with relations
	r.db.Preload("Asset").First(&item, "id = ?", item.ID)

	return item, nil
}

func (r ItemRepositoryStruct) DeleteItemRepository(request IdItemRequest) (entities.Item, error) {
	item := entities.Item{}
	tx := r.db.First(&item, "id = ?", request.Id)
	if tx.Error != nil {
		return item, tx.Error
	}

	tx = r.db.Delete(&item)
	if tx.Error != nil {
		return item, tx.Error
	}

	return item, nil
}

