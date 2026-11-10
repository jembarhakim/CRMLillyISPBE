package servicerequests

import (
	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
)

type ServiceRequestRepositoryInterface interface {
	Customer(customerID string) (*entities.Customer, error)
	Product(productID string) (*entities.Products, error)
	Products() ([]entities.Products, error)
	CreateRequest(request *entities.CustomerServiceRequest) error
	ListByCustomer(customerID string) ([]entities.CustomerServiceRequest, error)
	LatestByCustomer(customerID string) (*entities.CustomerServiceRequest, error)
	ByID(id string) (*entities.CustomerServiceRequest, error)
	SaveRequest(request *entities.CustomerServiceRequest) error
	ListPending() ([]entities.CustomerServiceRequest, error)
}

type ServiceRequestRepositoryStruct struct {
	db *gorm.DB
}

func NewServiceRequestRepository(db *gorm.DB) *ServiceRequestRepositoryStruct {
	return &ServiceRequestRepositoryStruct{db}
}

func (r *ServiceRequestRepositoryStruct) Customer(customerID string) (*entities.Customer, error) {
	customer := &entities.Customer{}
	if err := r.db.Preload("Product").Where("id = ?", customerID).First(customer).Error; err != nil {
		return nil, err
	}
	return customer, nil
}

func (r *ServiceRequestRepositoryStruct) Product(productID string) (*entities.Products, error) {
	product := &entities.Products{}
	if err := r.db.Where("id = ?", productID).First(product).Error; err != nil {
		return nil, err
	}
	return product, nil
}

func (r *ServiceRequestRepositoryStruct) Products() ([]entities.Products, error) {
	products := []entities.Products{}
	err := r.db.Order("price ASC").Find(&products).Error
	return products, err
}

func (r *ServiceRequestRepositoryStruct) CreateRequest(request *entities.CustomerServiceRequest) error {
	return r.db.Create(request).Error
}

func (r *ServiceRequestRepositoryStruct) ListByCustomer(customerID string) ([]entities.CustomerServiceRequest, error) {
	requests := []entities.CustomerServiceRequest{}
	err := r.db.
		Where("customer_id = ?", customerID).
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

// LatestByCustomer returns the newest request, used to block duplicate pending
// submissions for the same request type.
func (r *ServiceRequestRepositoryStruct) LatestByCustomer(customerID string) (*entities.CustomerServiceRequest, error) {
	request := &entities.CustomerServiceRequest{}
	err := r.db.
		Where("customer_id = ?", customerID).
		Order("created_at DESC").
		First(request).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return request, nil
}

func (r *ServiceRequestRepositoryStruct) ByID(id string) (*entities.CustomerServiceRequest, error) {
	request := &entities.CustomerServiceRequest{}
	if err := r.db.Where("id = ?", id).First(request).Error; err != nil {
		return nil, err
	}
	return request, nil
}

func (r *ServiceRequestRepositoryStruct) SaveRequest(request *entities.CustomerServiceRequest) error {
	return r.db.Save(request).Error
}

// ListPending powers the admin approval queue.
func (r *ServiceRequestRepositoryStruct) ListPending() ([]entities.CustomerServiceRequest, error) {
	requests := []entities.CustomerServiceRequest{}
	err := r.db.
		Where("status = ?", entities.ServiceRequestPending).
		Order("created_at ASC").
		Find(&requests).Error
	return requests, err
}
