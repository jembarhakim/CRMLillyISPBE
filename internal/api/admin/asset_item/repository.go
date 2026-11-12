package asset_item

import (
	"skripsi-be/internal/models/entities"

	"github.com/jinzhu/copier"
	"gorm.io/gorm"
)

type AssetItemRepositoryInterface interface {
	FindAssetItemsRepository(request GetAssetItemsRequest) ([]entities.AssetItem, error)
	CreateAssetItemRepository(request CreateAssetItemRequest) (entities.AssetItem, error)
	FindByIdAssetItemRepository(request IdAssetItemRequest) (entities.AssetItem, error)
	UpdateAssetItemRepository(request UpdateAssetItemRequest) (entities.AssetItem, error)
	DeleteAssetItemRepository(request IdAssetItemRequest) (entities.AssetItem, error)
}

type AssetItemRepositoryStruct struct {
	db *gorm.DB
}

func NewAssetItemRepository(db *gorm.DB) *AssetItemRepositoryStruct {
	return &AssetItemRepositoryStruct{db}
}

func (r AssetItemRepositoryStruct) FindAssetItemsRepository(request GetAssetItemsRequest) ([]entities.AssetItem, error) {
	items := []entities.AssetItem{}
	query := r.db.Preload("Asset").Preload("Company")

	if request.AssetID != nil {
		query = query.Where("asset_id = ?", *request.AssetID)
	}
	if request.Status != nil {
		query = query.Where("status = ?", *request.Status)
	}

	tx := query.Find(&items)
	if tx.Error != nil {
		return items, tx.Error
	}

	return items, nil
}

func (r AssetItemRepositoryStruct) CreateAssetItemRepository(request CreateAssetItemRequest) (entities.AssetItem, error) {
	item := entities.AssetItem{}
	copier.Copy(&item, &request)

	tx := r.db.Create(&item)
	if tx.Error != nil {
		return item, tx.Error
	}

	// Reload with relations
	r.db.Preload("Asset").Preload("Company").First(&item, "id = ?", item.ID)

	return item, nil
}

func (r AssetItemRepositoryStruct) FindByIdAssetItemRepository(request IdAssetItemRequest) (entities.AssetItem, error) {
	item := entities.AssetItem{}
	tx := r.db.Preload("Asset").Preload("Company").First(&item, "id = ?", request.Id)
	if tx.Error != nil {
		return item, tx.Error
	}
	return item, nil
}

func (r AssetItemRepositoryStruct) UpdateAssetItemRepository(request UpdateAssetItemRequest) (entities.AssetItem, error) {
	item := entities.AssetItem{}
	tx := r.db.First(&item, "id = ?", request.Id)
	if tx.Error != nil {
		return item, tx.Error
	}
	copier.Copy(&item, &request.CreateAssetItemRequest)

	tx = r.db.Save(&item)
	if tx.Error != nil {
		return item, tx.Error
	}

	// Reload with relations
	r.db.Preload("Asset").Preload("Company").First(&item, "id = ?", item.ID)

	return item, nil
}

func (r AssetItemRepositoryStruct) DeleteAssetItemRepository(request IdAssetItemRequest) (entities.AssetItem, error) {
	item := entities.AssetItem{}
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
