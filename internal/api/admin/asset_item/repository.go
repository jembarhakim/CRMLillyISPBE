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

	// Router / customer integration
	FindAssetItemUsageRepository(assetItemID string) (AssetItemUsageResponse, error)
	FindAssetItemUsagesBySerialRepository(serialNumber string) ([]AssetItemUsageResponse, error)
	FindCustomerRoutersRepository(customerID string) (CustomerRouterAssignmentResponse, error)
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

// =============================================================================
// Router / customer integration
// =============================================================================

// buildUsageFor resolves the customers currently using an asset unit by joining
// network_devices (customer_id + asset_item_id).
func (r AssetItemRepositoryStruct) buildUsageFor(item entities.AssetItem) (AssetItemUsageResponse, error) {
	response := AssetItemUsageResponse{
		AssetItemID:  item.ID,
		SerialNumber: item.SerialNumber,
		MacAddress:   item.MacAddress,
		MacSticker:   item.MacSticker,
		Status:       item.Status,
		Usage:        []AssetItemUser{},
	}

	if item.Asset != nil {
		response.Brand = item.Asset.Brand
		response.Type = item.Asset.Type
		response.Model = item.Asset.Model
	}

	type usageRow struct {
		CustomerID         string
		CustomerName       string
		CustomerAddress    string
		CustomerPhone      string
		NetworkDeviceID    string
		IPStatic           *string
		OntSerialNumber    *string
		Kepemilikan        string
		InstallationID     *string
		InstallationStatus string
	}

	var rows []usageRow
	err := r.db.Table("network_devices nd").
		Select(`nd.customer_id, c.name AS customer_name, c.address AS customer_address, c.phone AS customer_phone,
		        nd.id AS network_device_id, nd.ip_static, nd.ont_serial_number,
		        COALESCE(nd.kepemilikan_perangkat, '') AS kepemilikan,
		        nd.customer_installation_id AS installation_id,
		        COALESCE(ci.status, '') AS installation_status`).
		Joins("JOIN customer c ON c.id = nd.customer_id AND c.deleted_at IS NULL").
		Joins("LEFT JOIN customer_installations ci ON ci.id = nd.customer_installation_id").
		Where("nd.asset_item_id = ?", item.ID).
		Scan(&rows).Error
	if err != nil {
		return response, err
	}

	for _, row := range rows {
		response.Usage = append(response.Usage, AssetItemUser{
			CustomerID:         row.CustomerID,
			CustomerName:       row.CustomerName,
			CustomerAddress:    row.CustomerAddress,
			CustomerPhone:      row.CustomerPhone,
			NetworkDeviceID:    row.NetworkDeviceID,
			IPStatic:           row.IPStatic,
			OntSerialNumber:    row.OntSerialNumber,
			Kepemilikan:        row.Kepemilikan,
			InstallationID:     row.InstallationID,
			InstallationStatus: row.InstallationStatus,
		})
	}

	response.IsUsed = len(response.Usage) > 0
	return response, nil
}

// FindAssetItemUsageRepository resolves which customer(s) are using a specific asset unit
func (r AssetItemRepositoryStruct) FindAssetItemUsageRepository(assetItemID string) (AssetItemUsageResponse, error) {
	item := entities.AssetItem{}
	tx := r.db.Preload("Asset").First(&item, "id = ?", assetItemID)
	if tx.Error != nil {
		return AssetItemUsageResponse{}, tx.Error
	}

	return r.buildUsageFor(item)
}

// FindAssetItemUsagesBySerialRepository resolves usage by physical unit serial number.
// Used to answer "SN <x> is installed at which customer?".
func (r AssetItemRepositoryStruct) FindAssetItemUsagesBySerialRepository(serialNumber string) ([]AssetItemUsageResponse, error) {
	items := []entities.AssetItem{}
	tx := r.db.Preload("Asset").
		Where("serial_number = ? OR TRIM(serial_number) = ?", serialNumber, serialNumber).
		Find(&items)
	if tx.Error != nil {
		return nil, tx.Error
	}

	results := make([]AssetItemUsageResponse, 0, len(items))
	for _, item := range items {
		usage, err := r.buildUsageFor(item)
		if err != nil {
			return results, err
		}
		results = append(results, usage)
	}

	return results, nil
}

// FindCustomerRoutersRepository returns every router/ONT unit assigned to a customer,
// fully resolved from network_devices -> asset_items -> assets.
func (r AssetItemRepositoryStruct) FindCustomerRoutersRepository(customerID string) (CustomerRouterAssignmentResponse, error) {
	response := CustomerRouterAssignmentResponse{
		CustomerID: customerID,
		Routers:    []CustomerRouterDetail{},
	}

	var customer entities.Customer
	if err := r.db.First(&customer, "id = ?", customerID).Error; err != nil {
		return response, err
	}
	response.CustomerName = customer.Name

	devices := []entities.NetworkDevice{}
	err := r.db.Preload("Assets").
		Preload("AssetItem").
		Preload("AssetItem.Asset").
		Preload("CustomerInstallation").
		Where("customer_id = ?", customerID).
		Find(&devices).Error
	if err != nil {
		return response, err
	}

	for _, device := range devices {
		detail := CustomerRouterDetail{
			NetworkDeviceID: device.ID,
			AssetItemID:     device.AssetItemID,
			Kepemilikan:     device.KepemilikanPerangkat,
		}

		// Catalog info (brand / type / model) from the Assets relation
		if device.Assets != nil {
			detail.Brand = device.Assets.Brand
			detail.Type = device.Assets.Type
			detail.Model = device.Assets.Model
		}

		// Physical unit info from AssetItem
		if device.AssetItem != nil {
			detail.AssetItemStatus = device.AssetItem.Status
			detail.MacAddress = device.AssetItem.MacAddress
			if device.AssetItem.MacSticker != nil {
				detail.MacSticker = *device.AssetItem.MacSticker
			}
			if device.AssetItem.SerialNumber != nil {
				detail.SerialNumber = *device.AssetItem.SerialNumber
			}
			if device.AssetItem.Asset != nil {
				if detail.Brand == "" {
					detail.Brand = device.AssetItem.Asset.Brand
				}
				if detail.Type == "" {
					detail.Type = device.AssetItem.Asset.Type
				}
				if detail.Model == "" {
					detail.Model = device.AssetItem.Asset.Model
				}
			}
		}

		if device.MacAddress != nil && *device.MacAddress != "" {
			detail.MacAddress = *device.MacAddress
		}
		if device.OntSerialNumber != nil {
			detail.OntSerialNumber = *device.OntSerialNumber
		}
		if device.IPStatic != nil {
			detail.IPStatic = *device.IPStatic
		}
		if device.SwitchID != nil {
			detail.SwitchID = *device.SwitchID
		}
		if device.PortNumber != nil {
			detail.PortNumber = *device.PortNumber
		}
		if device.CustomerInstallationID != nil {
			detail.InstallationID = device.CustomerInstallationID
			if device.CustomerInstallation != nil {
				detail.InstallationStatus = device.CustomerInstallation.Status
			}
		}

		response.Routers = append(response.Routers, detail)
	}

	response.HasRouter = len(response.Routers) > 0
	return response, nil
}
