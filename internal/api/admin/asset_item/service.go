package asset_item

import "skripsi-be/internal/models/entities"

type AssetItemServiceInterface interface {
	GetAssetItemsService(request GetAssetItemsRequest) ([]entities.AssetItem, error)
	CreateAssetItemService(request CreateAssetItemRequest) (entities.AssetItem, error)
	GetByIdAssetItemService(request IdAssetItemRequest) (entities.AssetItem, error)
	UpdateAssetItemService(request UpdateAssetItemRequest) (entities.AssetItem, error)
	DeleteAssetItemService(request IdAssetItemRequest) (entities.AssetItem, error)
}

type AssetItemServiceStruct struct {
	repository AssetItemRepositoryInterface
}

func NewAssetItemService(repository AssetItemRepositoryInterface) AssetItemServiceStruct {
	return AssetItemServiceStruct{repository}
}

func (s AssetItemServiceStruct) GetAssetItemsService(request GetAssetItemsRequest) ([]entities.AssetItem, error) {
	items, err := s.repository.FindAssetItemsRepository(request)
	if err != nil {
		return items, err
	}
	return items, err
}

func (s AssetItemServiceStruct) GetByIdAssetItemService(request IdAssetItemRequest) (entities.AssetItem, error) {
	item, err := s.repository.FindByIdAssetItemRepository(request)
	if err != nil {
		return item, err
	}
	return item, err
}

func (s AssetItemServiceStruct) CreateAssetItemService(request CreateAssetItemRequest) (entities.AssetItem, error) {
	item, err := s.repository.CreateAssetItemRepository(request)
	if err != nil {
		return item, err
	}
	return item, err
}

func (s AssetItemServiceStruct) UpdateAssetItemService(request UpdateAssetItemRequest) (entities.AssetItem, error) {
	item, err := s.repository.UpdateAssetItemRepository(request)
	if err != nil {
		return item, err
	}
	return item, err
}

func (s AssetItemServiceStruct) DeleteAssetItemService(request IdAssetItemRequest) (entities.AssetItem, error) {
	item, err := s.repository.DeleteAssetItemRepository(request)
	if err != nil {
		return item, err
	}
	return item, err
}

