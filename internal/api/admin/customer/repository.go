package customer

import (
	"fmt"
	"skripsi-be/internal/models/entities"
	"skripsi-be/internal/services"
	"strings"
	"time"

	"github.com/jinzhu/copier"
	"gorm.io/gorm"
)

type AdminCustomerRepositoryInterface interface {
	CreateAdminCustomerRepository(customer entities.Customer) (entities.Customer, error)
	UpdateAdminCustomerRepository(request UpdateAdminCustomerRequest) (entities.Customer, error)
	DeleteAdminCustomerRepository(request IdAdminCustomerRequest) (entities.Customer, error)
	DeleteAdminCustomerWithRelatedRepository(request IdAdminCustomerRequest) (entities.Customer, error)
	GetCustomerRelatedRecordsRepository(request IdAdminCustomerRequest) (map[string]int64, error)
	FindByIdAdminCustomerRepository(request IdAdminCustomerRequest) (entities.Customer, error)
	FindByIdDetailAdminCustomerRepository(request IdAdminCustomerRequest) (*CustomerDetailResponse, error)
	FindAdminCustomerRepository() ([]CustomerListResponse, error)
	FindAdminCustomerRepositoryWithFilter(isInternet, isCollaborator string) ([]CustomerListResponse, error)
}
type AdminCustomerRepositoryStruct struct {
	db *gorm.DB
}

func NewAdminCustomerRepository(db *gorm.DB) AdminCustomerRepositoryStruct {
	return AdminCustomerRepositoryStruct{db}
}

// buildRouterSummary converts a network device (with preloaded AssetItem + Assets)
// into a compact router summary. Returns nil when the device has no asset info at all.
func buildRouterSummary(device entities.NetworkDevice) *CustomerRouterSummary {
	summary := &CustomerRouterSummary{
		NetworkDeviceID: device.ID,
		AssetItemID:     device.AssetItemID,
		Kepemilikan:     device.KepemilikanPerangkat,
	}

	// Catalog info (brand / type / model) lives on the Assets relation
	if device.Assets != nil {
		summary.Brand = device.Assets.Brand
		summary.Type = device.Assets.Type
		summary.Model = device.Assets.Model
	}

	// Physical unit info lives on AssetItem
	if device.AssetItem != nil {
		summary.AssetItemStatus = device.AssetItem.Status
		summary.MacAddress = device.AssetItem.MacAddress
		if device.AssetItem.SerialNumber != nil {
			summary.SerialNumber = *device.AssetItem.SerialNumber
		}
		// Fallback for catalog data when the network_device -> assets link is missing
		if device.AssetItem.Asset != nil {
			if summary.Brand == "" {
				summary.Brand = device.AssetItem.Asset.Brand
			}
			if summary.Type == "" {
				summary.Type = device.AssetItem.Asset.Type
			}
			if summary.Model == "" {
				summary.Model = device.AssetItem.Asset.Model
			}
		}
	}

	if device.MacAddress != nil && *device.MacAddress != "" {
		summary.MacAddress = *device.MacAddress
	}
	if device.OntSerialNumber != nil {
		summary.OntSerialNumber = *device.OntSerialNumber
	}

	// Serial number fallback chain:
	// asset_items.serial_number -> network_devices.ont_serial_number -> assets.serial_number (legacy)
	if summary.SerialNumber == "" {
		summary.SerialNumber = summary.OntSerialNumber
	}
	if summary.SerialNumber == "" && device.Assets != nil {
		summary.SerialNumber = device.Assets.SerialNumber
	}

	if device.IPStatic != nil {
		summary.IPStatic = *device.IPStatic
	}
	if device.SwitchID != nil {
		summary.SwitchID = *device.SwitchID
	}
	if device.PortNumber != nil {
		summary.PortNumber = *device.PortNumber
	}

	// Skip devices that carry no identifying router information
	if summary.SerialNumber == "" && summary.MacAddress == "" &&
		summary.Brand == "" && summary.Model == "" && summary.AssetItemID == nil {
		return nil
	}

	return summary
}

// loadNetworkDevicesWithAsset loads network devices for a customer including the
// physical asset unit (AssetItem) and the asset catalog (Assets).
func (r AdminCustomerRepositoryStruct) loadNetworkDevicesWithAsset(customerID string) []entities.NetworkDevice {
	networkDevices := []entities.NetworkDevice{}
	r.db.Preload("Product").
		Preload("Assets").
		Preload("AssetItem").
		Preload("AssetItem.Asset").
		Where("customer_id = ?", customerID).
		Find(&networkDevices)
	return networkDevices
}

func (r AdminCustomerRepositoryStruct) FindAdminCustomerRepository() ([]CustomerListResponse, error) {
	customers := []entities.Customer{}
	tx := r.db.Preload("Area").Preload("Company").Where("deleted_at IS NULL").Find(&customers)

	if tx.Error != nil {
		return nil, tx.Error
	}

	// Build response with network device data
	var response []CustomerListResponse
	for _, customer := range customers {
		// Get network devices for this customer (with Product, Assets catalog and physical AssetItem unit)
		networkDevices := r.loadNetworkDevicesWithAsset(customer.ID)

		// Note: Product information is now managed through network_devices table
		// Customer no longer has direct Product relationship

		// Aggregate IP and MAC addresses from network devices
		var ipAddresses []string
		var macAddresses []string
		var productName *string
		var productPrice *int64
		var routerSummary *CustomerRouterSummary

		for _, device := range networkDevices {
			if device.IPStatic != nil && *device.IPStatic != "" {
				ipAddresses = append(ipAddresses, *device.IPStatic)
			}
			if device.MacAddress != nil && *device.MacAddress != "" {
				macAddresses = append(macAddresses, *device.MacAddress)
			}
			// Get product information from the first device with a product
			if device.Product != nil && productName == nil {
				productName = &device.Product.Name
				productPrice = &device.Product.Price
			}
			// Get router summary from the first device that has a linked asset unit
			if routerSummary == nil {
				routerSummary = buildRouterSummary(device)
			}
		}

		// Create combined strings
		var combinedIP *string
		var combinedMAC *string

		if len(ipAddresses) > 0 {
			combined := strings.Join(ipAddresses, ", ")
			combinedIP = &combined
		}

		if len(macAddresses) > 0 {
			combined := strings.Join(macAddresses, ", ")
			combinedMAC = &combined
		}

		customerResponse := CustomerListResponse{
			Customer:      customer,
			IPStatic:      combinedIP,
			MacAddress:    combinedMAC,
			ProductName:   productName,
			ProductPrice:  productPrice,
			RouterSummary: routerSummary,
		}

		response = append(response, customerResponse)
	}

	return response, nil
}
func (r AdminCustomerRepositoryStruct) FindByIdAdminCustomerRepository(request IdAdminCustomerRequest) (entities.Customer, error) {
	customer := entities.Customer{}
	tx := r.db.Preload("Area").Preload("Company").Where("deleted_at IS NULL").Find(&customer, "id = ?", request.Id)
	if tx.Error != nil {
		return customer, tx.Error
	}

	return customer, tx.Error
}
func (r AdminCustomerRepositoryStruct) CreateAdminCustomerRepository(customer entities.Customer) (entities.Customer, error) {
	tx := r.db.Create(&customer)
	if tx.Error != nil {
		return customer, tx.Error
	}

	return customer, nil

}
func (r AdminCustomerRepositoryStruct) UpdateAdminCustomerRepository(request UpdateAdminCustomerRequest) (entities.Customer, error) {
	customer := entities.Customer{}
	tx := r.db.Preload("Area").Find(&customer, "id = ?", request.Id)
	if tx.Error != nil {
		return customer, tx.Error
	}

	copier.Copy(&customer, &request)

	tx = r.db.Save(&customer)
	if tx.Error != nil {
		return customer, tx.Error
	}

	return customer, tx.Error
}

func (r AdminCustomerRepositoryStruct) DeleteAdminCustomerRepository(request IdAdminCustomerRequest) (entities.Customer, error) {
	customer := entities.Customer{}
	tx := r.db.Preload("Area").Where("deleted_at IS NULL").Find(&customer, "id = ?", request.Id)
	if tx.Error != nil {
		return customer, tx.Error
	}

	// Check if customer is already soft deleted
	if customer.DeletedAt != nil {
		return customer, fmt.Errorf("customer is already deleted")
	}

	// For soft delete, we preserve all related records for historical data
	// This includes invoices, installations, network devices, services, etc.
	// They will remain in the database but the customer will be marked as deleted

	// Soft delete: Set deleted_at to current timestamp
	now := time.Now()
	customer.DeletedAt = &now
	tx = r.db.Save(&customer)
	if tx.Error != nil {
		return customer, tx.Error
	}

	return customer, nil
}

// DeleteAdminCustomerWithRelatedRepository - Soft delete customer (preserve related records for historical data)
func (r AdminCustomerRepositoryStruct) DeleteAdminCustomerWithRelatedRepository(request IdAdminCustomerRequest) (entities.Customer, error) {
	customer := entities.Customer{}
	tx := r.db.Preload("Area").Where("deleted_at IS NULL").Find(&customer, "id = ?", request.Id)
	if tx.Error != nil {
		return customer, tx.Error
	}

	// Check if customer is already soft deleted
	if customer.DeletedAt != nil {
		return customer, fmt.Errorf("customer is already deleted")
	}

	// For soft delete, we preserve all related records for historical data
	// This includes invoices, installations, network devices, services, etc.
	// They will remain in the database but the customer will be marked as deleted

	// Soft delete the customer only
	now := time.Now()
	customer.DeletedAt = &now
	tx = r.db.Save(&customer)
	if tx.Error != nil {
		return customer, tx.Error
	}

	return customer, nil
}

// GetCustomerRelatedRecordsRepository - Get count of related records for a customer
func (r AdminCustomerRepositoryStruct) GetCustomerRelatedRecordsRepository(request IdAdminCustomerRequest) (map[string]int64, error) {
	relatedCounts := make(map[string]int64)

	// Count invoices
	var invoiceCount int64
	r.db.Model(&entities.Invoice{}).Where("customer_id = ?", request.Id).Count(&invoiceCount)
	relatedCounts["invoices"] = invoiceCount

	// Count installations
	var installationCount int64
	r.db.Model(&entities.CustomerInstallation{}).Where("customer_id = ?", request.Id).Count(&installationCount)
	relatedCounts["installations"] = installationCount

	// Count network devices
	var networkDeviceCount int64
	r.db.Model(&entities.NetworkDevice{}).Where("customer_id = ?", request.Id).Count(&networkDeviceCount)
	relatedCounts["network_devices"] = networkDeviceCount

	// Count customer services
	var serviceCount int64
	r.db.Model(&entities.CustomerService{}).Where("customer_id = ?", request.Id).Count(&serviceCount)
	relatedCounts["customer_services"] = serviceCount

	// Count asset transactions
	var assetTransactionCount int64
	r.db.Model(&entities.AssetTransaction{}).Where("customer_installation_id IN (SELECT id FROM customer_installations WHERE customer_id = ?)", request.Id).Count(&assetTransactionCount)
	relatedCounts["asset_transactions"] = assetTransactionCount

	// Count cables
	var cableCount int64
	r.db.Model(&entities.Cable{}).Where("customer_installation_id IN (SELECT id FROM customer_installations WHERE customer_id = ?)", request.Id).Count(&cableCount)
	relatedCounts["cables"] = cableCount

	// Count images
	var imageCount int64
	r.db.Model(&entities.Image{}).Where("archive_installation_id IN (SELECT id FROM customer_installations WHERE customer_id = ?)", request.Id).Count(&imageCount)
	relatedCounts["images"] = imageCount

	return relatedCounts, nil
}

// CustomerDetailResponse represents comprehensive customer information
type CustomerDetailResponse struct {
	Customer        entities.Customer                 `json:"customer"`
	Installations   []entities.CustomerInstallation   `json:"installations"`
	NetworkDevices  []entities.NetworkDevice          `json:"network_devices"`
	Invoices        []entities.Invoice                `json:"invoices,omitempty"`
	OpticalReadings []entities.PhysicalOpticalReading `json:"optical_readings"`
	LinkStability   *entities.LinkFlappingStatus      `json:"link_stability,omitempty"`
}

// CustomerListResponse represents customer with network device data for list view
type CustomerListResponse struct {
	entities.Customer
	IPStatic     *string `json:"ip_static"`
	MacAddress   *string `json:"mac_address"`
	ProductName  *string `json:"product_name"`
	ProductPrice *int64  `json:"product_price"`

	// Router (physical asset unit) summary for the customer.
	// Sourced from network_devices -> asset_items -> assets.
	RouterSummary *CustomerRouterSummary `json:"router_summary,omitempty"`
}

// CustomerRouterSummary describes the router/ONT unit physically installed at a customer
type CustomerRouterSummary struct {
	NetworkDeviceID string  `json:"network_device_id"`
	AssetItemID     *string `json:"asset_item_id,omitempty"`
	Brand           string  `json:"brand,omitempty"`
	Type            string  `json:"type,omitempty"`
	Model           string  `json:"model,omitempty"`
	SerialNumber    string  `json:"serial_number,omitempty"`
	MacAddress      string  `json:"mac_address,omitempty"`
	OntSerialNumber string  `json:"ont_serial_number,omitempty"`
	AssetItemStatus string  `json:"asset_item_status,omitempty"`
	Kepemilikan     string  `json:"kepemilikan_perangkat,omitempty"`
	IPStatic        string  `json:"ip_static,omitempty"`
	SwitchID        string  `json:"switch_id,omitempty"`
	PortNumber      string  `json:"port_number,omitempty"`
}

func (r AdminCustomerRepositoryStruct) FindByIdDetailAdminCustomerRepository(request IdAdminCustomerRequest) (*CustomerDetailResponse, error) {
	// Get customer with all related data
	customer := entities.Customer{}
	tx := r.db.Preload("Area").Preload("Company").Where("deleted_at IS NULL").Find(&customer, "id = ?", request.Id)
	if tx.Error != nil {
		return nil, tx.Error
	}

	// Check if customer was found
	if customer.ID == "" {
		// Check if customer exists but is soft-deleted
		var deletedCustomer entities.Customer
		deletedCheck := r.db.Where("id = ? AND deleted_at IS NOT NULL", request.Id).First(&deletedCustomer)
		if deletedCheck.Error == nil {
			return nil, fmt.Errorf("customer with ID '%s' has been deleted", request.Id)
		}
		return nil, fmt.Errorf("customer with ID '%s' not found", request.Id)
	}

	// Get customer installations
	installations := []entities.CustomerInstallation{}
	r.db.Preload("Technician").Preload("Images").Where("customer_id = ?", request.Id).Find(&installations)

	// Get network devices for this customer (with Product, Assets catalog and physical AssetItem unit)
	networkDevices := []entities.NetworkDevice{}
	r.db.Preload("Product").
		Preload("Assets").
		Preload("AssetItem").
		Preload("AssetItem.Asset").
		Where("customer_id = ?", request.Id).
		Find(&networkDevices)

	// Get real-time Netwatch status for each network device from MikroTik
	for i := range networkDevices {
		if networkDevices[i].IPStatic != nil && *networkDevices[i].IPStatic != "" {
			// Note: Real-time status is now handled in the frontend via MikroTik API
			// The deleted LastPingStatus and LastPingTimestamp fields are no longer used
			// Real-time status fetching is handled by the frontend getDeviceConnectionStatus method
		}
	}

	// Note: Product information is now managed through network_devices table
	// Customer no longer has direct Product relationship

	// Get recent invoices for this customer (optional)
	invoices := []entities.Invoice{}
	r.db.Where("customer_id = ?", request.Id).Order("createdAt DESC").Limit(10).Find(&invoices)

	opticalReadings := []entities.PhysicalOpticalReading{}
	r.db.Where("customer_id = ?", request.Id).Order("created_at DESC").Limit(50).Find(&opticalReadings)

	var linkStability *entities.LinkFlappingStatus
	var stability entities.LinkFlappingStatus
	if err := r.db.Where("customer_id = ?", request.Id).First(&stability).Error; err == nil {
		linkStability = &stability
	}

	detail := &CustomerDetailResponse{
		Customer:        customer,
		Installations:   installations,
		NetworkDevices:  networkDevices,
		Invoices:        invoices,
		OpticalReadings: opticalReadings,
		LinkStability:   linkStability,
	}

	return detail, nil
}

// getRealTimeNetwatchStatus gets real-time status from MikroTik Netwatch
func (r AdminCustomerRepositoryStruct) getRealTimeNetwatchStatus(ipAddress string) (string, time.Time, error) {
	// Use shared MikroTik service that's already connected
	mikroTikService := services.GetSharedMikroTikService()
	if mikroTikService == nil || !mikroTikService.IsConnected() {
		return "", time.Time{}, fmt.Errorf("MikroTik service not available or not connected")
	}

	// Get Netwatch devices from MikroTik
	devices, err := mikroTikService.GetNetwatchDevices()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to get netwatch devices: %v", err)
	}

	// Find the specific device by IP
	for _, device := range devices {
		if host, ok := device["host"].(string); ok && host == ipAddress {
			if status, ok := device["status"].(string); ok {
				// Convert status to lowercase for consistency
				status = strings.ToLower(status)
				return status, time.Now(), nil
			}
		}
	}

	return "unknown", time.Time{}, fmt.Errorf("device %s not found in MikroTik Netwatch", ipAddress)
}

func (r AdminCustomerRepositoryStruct) FindAdminCustomerRepositoryWithFilter(isInternet, isCollaborator string) ([]CustomerListResponse, error) {
	customers := []entities.Customer{}

	// Build query with filters
	query := r.db.Preload("Area").Preload("Company").Where("deleted_at IS NULL")

	// Apply filters if provided
	if isInternet != "" {
		query = query.Where("is_internet = ?", isInternet)
	}
	if isCollaborator != "" {
		query = query.Where("is_collaborator = ?", isCollaborator)
	}

	tx := query.Find(&customers)

	if tx.Error != nil {
		return nil, tx.Error
	}

	// Build response with network device data (same logic as FindAdminCustomerRepository)
	var response []CustomerListResponse
	for _, customer := range customers {
		// Get network devices for this customer (with Product, Assets catalog and physical AssetItem unit)
		networkDevices := r.loadNetworkDevicesWithAsset(customer.ID)

		// Aggregate IP and MAC addresses from network devices
		var ipAddresses []string
		var macAddresses []string
		var productName *string
		var productPrice *int64
		var routerSummary *CustomerRouterSummary

		for _, device := range networkDevices {
			if device.IPStatic != nil && *device.IPStatic != "" {
				ipAddresses = append(ipAddresses, *device.IPStatic)
			}
			if device.MacAddress != nil && *device.MacAddress != "" {
				macAddresses = append(macAddresses, *device.MacAddress)
			}
			// Get product information from the first device with a product
			if device.Product != nil && productName == nil {
				productName = &device.Product.Name
				productPrice = &device.Product.Price
			}
			// Get router summary from the first device that has a linked asset unit
			if routerSummary == nil {
				routerSummary = buildRouterSummary(device)
			}
		}

		// Create combined strings
		var combinedIP *string
		var combinedMAC *string

		if len(ipAddresses) > 0 {
			combined := strings.Join(ipAddresses, ", ")
			combinedIP = &combined
		}

		if len(macAddresses) > 0 {
			combined := strings.Join(macAddresses, ", ")
			combinedMAC = &combined
		}

		customerResponse := CustomerListResponse{
			Customer:      customer,
			IPStatic:      combinedIP,
			MacAddress:    combinedMAC,
			ProductName:   productName,
			ProductPrice:  productPrice,
			RouterSummary: routerSummary,
		}

		response = append(response, customerResponse)
	}

	return response, nil
}
