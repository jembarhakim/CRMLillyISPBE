package servicerequests

import (
	"fmt"
	"strings"
	"time"

	"skripsi-be/internal/models/entities"
)

type ServiceRequestServiceInterface interface {
	Summary(customerID string) (ServiceSummary, error)
	Page(customerID string) (ServicePageData, error)
	Create(customerID string, request CreateServiceRequestRequest) (*ServiceRequestItem, error)
	List(customerID string) ([]ServiceRequestItem, error)
	Review(adminID string, requestID string, request ReviewServiceRequestRequest) (*ServiceRequestItem, error)
	Pending() ([]ServiceRequestItem, error)
}

type ServiceRequestServiceStruct struct {
	repository ServiceRequestRepositoryInterface
}

func NewServiceRequestService(repository ServiceRequestRepositoryInterface) *ServiceRequestServiceStruct {
	return &ServiceRequestServiceStruct{repository}
}

var typeLabels = map[entities.ServiceRequestType]string{
	entities.ServiceRequestUpgrade:    "Upgrade Kecepatan",
	entities.ServiceRequestDowngrade:  "Downgrade Paket",
	entities.ServiceRequestRelocation: "Relokasi Alamat",
}

var statusLabels = map[entities.ServiceRequestStatus]string{
	entities.ServiceRequestPending:  "Menunggu Persetujuan",
	entities.ServiceRequestApproved: "Disetujui",
	entities.ServiceRequestRejected: "Ditolak",
}

// Summary builds the "current package" card for the service page.
//
// The billing period is derived from the customer installation date using a
// monthly cycle; this mirrors how recurring invoices are generated.
func (s *ServiceRequestServiceStruct) Summary(customerID string) (ServiceSummary, error) {
	summary := ServiceSummary{}

	customer, err := s.repository.Customer(customerID)
	if err != nil {
		return summary, err
	}

	summary.CustomerName = customer.Name
	summary.CustomerAddress = customer.Address

	productID := customer.ProductID
	if productID == "" {
		return summary, nil
	}

	var product *entities.Products
	if customer.Product != nil {
		product = customer.Product
	} else {
		product, err = s.repository.Product(productID)
		if err != nil {
			return summary, nil
		}
	}

	summary.ProductID = product.ID
	summary.ProductName = product.Name
	summary.Description = product.Description
	summary.Price = product.Price
	if product.DownloadSpeedMbps != nil {
		summary.DownloadMbps = *product.DownloadSpeedMbps
	}
	if product.UploadSpeedMbps != nil {
		summary.UploadMbps = *product.UploadSpeedMbps
	}

	// Active since installation date; next invoice on the same day next month.
	if !customer.InstallationDate.IsZero() {
		installed := customer.InstallationDate
		summary.ActiveSince = installed.Format("2006-01-02")
		summary.BillingCycleDay = installed.Day()

		now := time.Now()
		next := time.Date(now.Year(), now.Month(), installed.Day(), 0, 0, 0, 0, now.Location())
		if !next.After(now) {
			next = next.AddDate(0, 1, 0)
		}
		summary.NextInvoiceDate = next.Format("2006-01-02")
		summary.DaysRemaining = int(next.Sub(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())).Hours() / 24)
		summary.IsActive = summary.DaysRemaining >= 0
	}

	return summary, nil
}

// AvailablePlans lists the packages a customer may switch to, tagged with the
// direction relative to the active plan (price based).
func (s *ServiceRequestServiceStruct) availablePlans(customerID string, summary ServiceSummary) ([]AvailablePlan, error) {
	products, err := s.repository.Products()
	if err != nil {
		return nil, err
	}

	plans := make([]AvailablePlan, 0, len(products))
	for _, product := range products {
		if product.ID == summary.ProductID {
			continue // never offer the active plan
		}

		plan := AvailablePlan{
			ID:          product.ID,
			Name:        product.Name,
			Description: product.Description,
			Price:       product.Price,
			Direction:   "upgrade",
		}
		if product.DownloadSpeedMbps != nil {
			plan.DownloadMbps = *product.DownloadSpeedMbps
		}
		if product.UploadSpeedMbps != nil {
			plan.UploadMbps = *product.UploadSpeedMbps
		}
		if summary.ProductID != "" && product.Price <= summary.Price {
			plan.Direction = "downgrade"
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

// toItem converts the entity into the shared portal representation.
func (s *ServiceRequestServiceStruct) toItem(request entities.CustomerServiceRequest) ServiceRequestItem {
	item := ServiceRequestItem{
		ID:                 request.ID,
		CustomerID:         request.CustomerID,
		RequestType:        string(request.RequestType),
		TypeLabel:          typeLabels[request.RequestType],
		Status:             string(request.Status),
		StatusLabel:        statusLabels[request.Status],
		CurrentProductID:   request.CurrentProductID,
		RequestedProductID: request.RequestedProductID,
		NewAddress:         request.NewAddress,
		NewAreaID:          request.NewAreaID,
		Note:               request.Note,
		AttachmentURL:      request.AttachmentURL,
		ReviewNote:         request.ReviewNote,
		CreatedAt:          request.CreatedAt.Format("2006-01-02 15:04"),
		UpdatedAt:          request.UpdatedAt.Format("2006-01-02 15:04"),
	}

	// Resolve package names for display; failures are non fatal.
	if request.CurrentProductID != "" {
		if product, err := s.repository.Product(request.CurrentProductID); err == nil {
			item.CurrentProductName = product.Name
		}
	}
	if request.RequestedProductID != "" {
		if product, err := s.repository.Product(request.RequestedProductID); err == nil {
			item.RequestedProductName = product.Name
		}
	}
	if request.ReviewedAt != nil {
		item.ReviewedAt = request.ReviewedAt.Format("2006-01-02 15:04")
	}

	return item
}

// Page returns everything the service page needs in one round trip.
func (s *ServiceRequestServiceStruct) Page(customerID string) (ServicePageData, error) {
	page := ServicePageData{}

	summary, err := s.Summary(customerID)
	if err != nil {
		return page, err
	}
	page.Summary = summary

	plans, err := s.availablePlans(customerID, summary)
	if err != nil {
		return page, err
	}
	page.AvailablePlans = plans

	requests, err := s.List(customerID)
	if err != nil {
		return page, err
	}
	page.Requests = requests

	return page, nil
}

// List returns the request history of a customer.
func (s *ServiceRequestServiceStruct) List(customerID string) ([]ServiceRequestItem, error) {
	requests, err := s.repository.ListByCustomer(customerID)
	if err != nil {
		return nil, err
	}

	items := make([]ServiceRequestItem, 0, len(requests))
	for _, request := range requests {
		items = append(items, s.toItem(request))
	}
	return items, nil
}

// Create validates and stores a new service request in the admin approval queue.
func (s *ServiceRequestServiceStruct) Create(customerID string, request CreateServiceRequestRequest) (*ServiceRequestItem, error) {
	requestType := entities.ServiceRequestType(strings.ToLower(strings.TrimSpace(request.RequestType)))

	summary, err := s.Summary(customerID)
	if err != nil {
		return nil, err
	}

	newRequest := entities.CustomerServiceRequest{
		CustomerID:       customerID,
		RequestType:      requestType,
		Status:           entities.ServiceRequestPending,
		CurrentProductID: summary.ProductID,
		Note:             strings.TrimSpace(request.Note),
		AttachmentURL:    strings.TrimSpace(request.AttachmentURL),
	}

	switch requestType {
	case entities.ServiceRequestUpgrade, entities.ServiceRequestDowngrade:
		productID := strings.TrimSpace(request.RequestedProductID)
		if productID == "" {
			return nil, fmt.Errorf("paket yang diajukan wajib dipilih")
		}
		if productID == summary.ProductID {
			return nil, fmt.Errorf("paket yang diajukan sama dengan paket aktif Anda")
		}
		product, err := s.repository.Product(productID)
		if err != nil {
			return nil, fmt.Errorf("paket yang diajukan tidak ditemukan")
		}
		if requestType == entities.ServiceRequestUpgrade && product.Price < summary.Price {
			return nil, fmt.Errorf("paket %s bukan upgrade dari paket aktif Anda", product.Name)
		}
		if requestType == entities.ServiceRequestDowngrade && product.Price > summary.Price {
			return nil, fmt.Errorf("paket %s bukan downgrade dari paket aktif Anda", product.Name)
		}
		newRequest.RequestedProductID = productID

	case entities.ServiceRequestRelocation:
		address := strings.TrimSpace(request.NewAddress)
		if len([]rune(address)) < 10 {
			return nil, fmt.Errorf("alamat baru wajib diisi minimal 10 karakter")
		}
		newRequest.NewAddress = address
		newRequest.NewAreaID = strings.TrimSpace(request.NewAreaID)
		newRequest.NewLatitude = request.NewLatitude
		newRequest.NewLongitude = request.NewLongitude
	}

	// Block duplicate pending submissions of the same type.
	latest, err := s.repository.LatestByCustomer(customerID)
	if err != nil {
		return nil, err
	}
	if latest != nil &&
		latest.Status == entities.ServiceRequestPending &&
		latest.RequestType == requestType {
		return nil, fmt.Errorf("Anda masih memiliki pengajuan yang menunggu persetujuan")
	}

	if err := s.repository.CreateRequest(&newRequest); err != nil {
		return nil, err
	}

	item := s.toItem(newRequest)
	return &item, nil
}

// Review lets the admin team approve or reject a queued request.
func (s *ServiceRequestServiceStruct) Review(adminID string, requestID string, request ReviewServiceRequestRequest) (*ServiceRequestItem, error) {
	stored, err := s.repository.ByID(requestID)
	if err != nil {
		return nil, fmt.Errorf("pengajuan tidak ditemukan")
	}
	if stored.Status != entities.ServiceRequestPending {
		return nil, fmt.Errorf("pengajuan ini sudah diproses")
	}

	now := time.Now()
	stored.ReviewNote = strings.TrimSpace(request.ReviewNote)
	stored.ReviewedBy = &adminID
	stored.ReviewedAt = &now

	if request.Action == "approve" {
		stored.Status = entities.ServiceRequestApproved
	} else {
		stored.Status = entities.ServiceRequestRejected
	}

	if err := s.repository.SaveRequest(stored); err != nil {
		return nil, err
	}

	item := s.toItem(*stored)
	return &item, nil
}

// Pending returns the admin approval queue.
func (s *ServiceRequestServiceStruct) Pending() ([]ServiceRequestItem, error) {
	requests, err := s.repository.ListPending()
	if err != nil {
		return nil, err
	}

	items := make([]ServiceRequestItem, 0, len(requests))
	for _, request := range requests {
		items = append(items, s.toItem(request))
	}
	return items, nil
}

