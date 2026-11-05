package items_catalog

import "skripsi-be/internal/models/entities"

type ItemServiceInterface interface {
	GetAllItemsService(request GetItemsRequest) ([]entities.Item, error)
	CreateItemService(request CreateItemRequest) (entities.Item, error)
	GetByIdItemService(request IdItemRequest) (entities.Item, error)
	UpdateItemService(request UpdateItemRequest) (entities.Item, error)
	DeleteItemService(request IdItemRequest) (entities.Item, error)
}

type ItemServiceStruct struct {
	repository ItemRepositoryInterface
}

func NewItemService(repository ItemRepositoryInterface) ItemServiceStruct {
	return ItemServiceStruct{repository}
}

func (s ItemServiceStruct) GetAllItemsService(request GetItemsRequest) ([]entities.Item, error) {
	items, err := s.repository.FindItemsRepository(request)
	if err != nil {
		return items, err
	}
	return items, err
}

func (s ItemServiceStruct) GetByIdItemService(request IdItemRequest) (entities.Item, error) {
	item, err := s.repository.FindByIdItemRepository(request)
	if err != nil {
		return item, err
	}
	return item, err
}

func (s ItemServiceStruct) CreateItemService(request CreateItemRequest) (entities.Item, error) {
	item, err := s.repository.CreateItemRepository(request)
	if err != nil {
		return item, err
	}
	return item, err
}

func (s ItemServiceStruct) UpdateItemService(request UpdateItemRequest) (entities.Item, error) {
	item, err := s.repository.UpdateItemRepository(request)
	if err != nil {
		return item, err
	}
	return item, err
}

func (s ItemServiceStruct) DeleteItemService(request IdItemRequest) (entities.Item, error) {
	item, err := s.repository.DeleteItemRepository(request)
	if err != nil {
		return item, err
	}
	return item, err
}

