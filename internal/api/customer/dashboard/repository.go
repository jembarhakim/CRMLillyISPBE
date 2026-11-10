package dashboard

import (
	"fmt"

	"github.com/jinzhu/copier"
	"gorm.io/gorm"

	"skripsi-be/internal/models/dto"
	"skripsi-be/internal/models/entities"
)

type CustomerDashboardRepositoryInterface interface {
	MyUserCustomerDashboard(request string) (dto.CustomerDTO, error)
	MyProductCustomerDashboard(request string) (entities.Products, error)
	MyInvoiceCustomerDashboard(request string) ([]entities.Invoice, error)
	MyInvoiceIdCustomerDashboard(request SearchInvoice) (entities.Invoice, error)
	MyInvoiceUpdatePaymentCustomerDashboard(request SearchInvoice, link string) (entities.Invoice, error)
	GetProductIDFromNetworkDevice(customerID string) (string, error)
	GetNetworkDeviceData(customerID string) (entities.NetworkDevice, error)
	CheckDeviceStatus(macAddress string) (string, error)
	GetCustomerWithProduct(customerID string) (entities.Customer, error)
	GetAvailableProducts() ([]entities.Products, error)
	MyInvoiceDetailCustomerDashboard(customerID string, invoiceID string) (entities.Invoice, error)
}

type CustomerDashboardRepositoryStruct struct {
	db *gorm.DB
}

func NewCustomerDashboardRepository(db *gorm.DB) CustomerDashboardRepositoryStruct {
	return CustomerDashboardRepositoryStruct{db}
}

func (r CustomerDashboardRepositoryStruct) MyUserCustomerDashboard(request string) (dto.CustomerDTO, error) {
	myAccount := entities.Customer{}
	customerDto := dto.CustomerDTO{}
	err := r.db.Where("id = ?", &request).First(&myAccount).Error
	if err != nil {
		return customerDto, err
	}

	copier.Copy(&customerDto, &myAccount)

	// Add debug logging
	fmt.Printf("MyUserCustomerDashboard - Customer: ID=%s, Name=%s\n",
		myAccount.ID, myAccount.Name)

	return customerDto, nil
}

func (r CustomerDashboardRepositoryStruct) MyProductCustomerDashboard(request string) (entities.Products, error) {
	product := entities.Products{}
	err := r.db.Where("id = ?", &request).First(&product).Error
	if err != nil {
		return product, err
	}

	return product, nil
}

func (r CustomerDashboardRepositoryStruct) MyInvoiceCustomerDashboard(request string) ([]entities.Invoice, error) {
	invoice := []entities.Invoice{}
	err := r.db.Where("customer_id = ?", &request).Order("createdAt desc").Find(&invoice).Error
	if err != nil {
		return invoice, err
	}

	return invoice, nil
}

func (r CustomerDashboardRepositoryStruct) MyInvoiceIdCustomerDashboard(request SearchInvoice) (entities.Invoice, error) {
	invoice := entities.Invoice{}
	err := r.db.Where("customer_id = ?", request.UserId).Where("id = ?", request.InvoiceId).First(&invoice).Error
	if err != nil {
		return invoice, err
	}

	return invoice, nil
}

func (r CustomerDashboardRepositoryStruct) MyInvoiceUpdatePaymentCustomerDashboard(request SearchInvoice, link string) (entities.Invoice, error) {
	invoice := entities.Invoice{}
	err := r.db.Where("customer_id = ?", request.UserId).Where("id = ?", request.InvoiceId).First(&invoice).Error
	if err != nil {
		return invoice, err
	}
	invoice.Link = link
	invoice.Status = entities.InvoiceStatusPending
	err = r.db.Save(&invoice).Error
	if err != nil {
		return invoice, err
	}

	return invoice, nil
}

func (r CustomerDashboardRepositoryStruct) GetProductIDFromNetworkDevice(customerID string) (string, error) {
	networkDevice := entities.NetworkDevice{}
	err := r.db.Where("customer_id = ?", customerID).First(&networkDevice).Error
	if err != nil {
		// If no network device found, return empty string instead of error
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", err
	}

	if networkDevice.ProductID == nil {
		return "", nil
	}

	return *networkDevice.ProductID, nil
}

// GetNetworkDeviceData returns network device data with mac_address for a customer
func (r CustomerDashboardRepositoryStruct) GetNetworkDeviceData(customerID string) (entities.NetworkDevice, error) {
	networkDevice := entities.NetworkDevice{}
	fmt.Printf("GetNetworkDeviceData - Looking for customer_id: %s\n", customerID)

	err := r.db.Preload("Product").Where("customer_id = ?", customerID).First(&networkDevice).Error
	if err != nil {
		// If no network device found, return empty struct instead of error
		if err == gorm.ErrRecordNotFound {
			fmt.Printf("GetNetworkDeviceData - No network device found for customer: %s\n", customerID)
			// Initialize with empty values to avoid JSON serialization issues
			networkDevice = entities.NetworkDevice{
				ID:         "",
				CustomerID: customerID,
				MacAddress: nil,
				IPStatic:   nil,
				ProductID:  nil,
				Product:    nil,
			}
			return networkDevice, nil
		}
		fmt.Printf("GetNetworkDeviceData - Database error: %v\n", err)
		return networkDevice, err
	}

	fmt.Printf("GetNetworkDeviceData - Found network device: ID=%s, MAC=%v, ProductID=%v, Product=%v\n",
		networkDevice.ID, networkDevice.MacAddress, networkDevice.ProductID, networkDevice.Product)

	return networkDevice, nil
}

// CheckDeviceStatus checks device status via Mikrotik using mac_address
func (r CustomerDashboardRepositoryStruct) CheckDeviceStatus(macAddress string) (string, error) {
	// This will be implemented to call Mikrotik API
	// For now, return a mock status based on mac_address
	if macAddress == "" {
		return "off", nil
	}

	// Mock logic - in real implementation, this would call Mikrotik API
	// Check if mac_address contains certain patterns to simulate status
	if len(macAddress) > 10 {
		return "up", nil
	}

	return "down", nil
}

// GetCustomerWithProduct gets customer data (placeholder for future product relationship)
func (r CustomerDashboardRepositoryStruct) GetCustomerWithProduct(customerID string) (entities.Customer, error) {
	customer := entities.Customer{}
	fmt.Printf("GetCustomerWithProduct - Looking for customer_id: %s\n", customerID)

	err := r.db.Where("id = ?", customerID).First(&customer).Error
	if err != nil {
		fmt.Printf("GetCustomerWithProduct - Error: %v\n", err)
		return customer, err
	}

	fmt.Printf("GetCustomerWithProduct - Found customer: ID=%s, Name=%s\n",
		customer.ID, customer.Name)

	return customer, nil
}

// GetAvailableProducts gets all available products
func (r CustomerDashboardRepositoryStruct) GetAvailableProducts() ([]entities.Products, error) {
	var products []entities.Products
	fmt.Printf("GetAvailableProducts - Fetching all products\n")

	err := r.db.Find(&products).Error
	if err != nil {
		fmt.Printf("GetAvailableProducts - Error: %v\n", err)
		return products, err
	}

	fmt.Printf("GetAvailableProducts - Found %d products\n", len(products))
	return products, nil
}

// MyInvoiceDetailCustomerDashboard returns a single invoice (with items)
// belonging to the customer. The customer_id filter is what prevents a
// customer from reading somebody else's invoice by guessing an id.
func (r CustomerDashboardRepositoryStruct) MyInvoiceDetailCustomerDashboard(customerID string, invoiceID string) (entities.Invoice, error) {
	invoice := entities.Invoice{}
	err := r.db.
		Preload("InvoiceItems").
		Preload("Customer").
		Where("customer_id = ? AND id = ?", customerID, invoiceID).
		First(&invoice).Error
	if err != nil {
		return invoice, err
	}
	return invoice, nil
}
