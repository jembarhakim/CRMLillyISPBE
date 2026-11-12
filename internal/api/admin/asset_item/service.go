package asset_item

import "skripsi-be/internal/models/entities"

type AssetItemServiceInterface interface {
	GetAssetItemsService(request GetAssetItemsRequest) ([]entities.AssetItem, error)
	CreateAssetItemService(request CreateAssetItemRequest) (entities.AssetItem, error)
	GetByIdAssetItemService(request IdAssetItemRequest) (entities.AssetItem, error)
	UpdateAssetItemService(request UpdateAssetItemRequest) (entities.AssetItem, error)
	DeleteAssetItemService(request IdAssetItemRequest) (entities.AssetItem, error)

	// Router / customer integration
	GetAssetItemUsageService(assetItemID string) (AssetItemUsageResponse, error)
	GetAssetItemUsagesBySerialService(serialNumber string) ([]AssetItemUsageResponse, error)
	GetCustomerRoutersService(customerID string) (CustomerRouterAssignmentResponse, error)
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

// =============================================================================
// Router / customer integration
// =============================================================================

// GetAssetItemUsageService returns the customers currently using a specific asset unit
func (s AssetItemServiceStruct) GetAssetItemUsageService(assetItemID string) (AssetItemUsageResponse, error) {
	return s.repository.FindAssetItemUsageRepository(assetItemID)
}

// GetAssetItemUsagesBySerialService returns usage info for every unit matching a serial number
func (s AssetItemServiceStruct) GetAssetItemUsagesBySerialService(serialNumber string) ([]AssetItemUsageResponse, error) {
	return s.repository.FindAssetItemUsagesBySerialRepository(serialNumber)
}

// GetCustomerRoutersService returns all router/ONT units assigned to a customer
func (s AssetItemServiceStruct) GetCustomerRoutersService(customerID string) (CustomerRouterAssignmentResponse, error) {
	return s.repository.FindCustomerRoutersRepository(customerID)
}

