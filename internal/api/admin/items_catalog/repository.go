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

	// minimum_stock is optional on create; fall back to 0 when not provided
	// so the not-null column always receives an explicit value.
	item.MinimumStock = 0
	if request.MinimumStock != nil {
		item.MinimumStock = *request.MinimumStock
	}

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

	// GORM skips fields whose value equals the Go zero value when the column has a
	// default tag, so a minimum_stock of 0 would be silently ignored by Save().
	// Assign the column explicitly via Select so that 0 is always persisted, while
	// leaving the stored value untouched when the field is omitted from the payload.
	updateColumns := []string{"name", "default_unit", "category", "description", "asset_id"}
	if request.MinimumStock != nil {
		item.MinimumStock = *request.MinimumStock
		updateColumns = append(updateColumns, "minimum_stock")
	}

	tx = r.db.Model(&item).Select(updateColumns).Updates(&item)
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
