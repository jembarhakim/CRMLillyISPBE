package inventory

import (
	"errors"
	"fmt"
	"math/rand"
	"skripsi-be/internal/models/entities"
	"time"
)

type InventoryServiceInterface interface {
	CreatePurchaseService(request CreatePurchaseRequest, createdBy string) (*PurchaseResponse, error)
	CreateDeploymentService(request CreateDeploymentRequest, createdBy string) (*DeploymentResponse, error)
	GetInventoryStatusService(request InventoryStatusRequest) ([]InventoryStatusResponse, error)
}

type InventoryService struct {
	repository InventoryRepositoryInterface
}

func NewInventoryService(repository InventoryRepositoryInterface) InventoryServiceInterface {
	return &InventoryService{repository: repository}
}

// CreatePurchaseService handles the purchase stock workflow
func (s *InventoryService) CreatePurchaseService(request CreatePurchaseRequest, createdBy string) (*PurchaseResponse, error) {
	// Generate purchase ID
	purchaseID, err := s.repository.GetNextBarangMasukID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate purchase ID: %v", err)
	}

	// Parse date
	purchaseDate, err := time.Parse("2006-01-02", request.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %v", err)
	}

	// Create main purchase record
	purchase := &entities.BarangMasuk{
		IdMasuk:   purchaseID,
		Date:      purchaseDate,
		Notes:     request.Notes,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}

	// Create detail records and calculate totals
	var details []entities.DetailBarangMasuk
	var totalItems int
	var totalAmount int

	for _, item := range request.Items {
		// Validate asset exists
		_, err := s.repository.GetAssetByID(item.AssetID)
		if err != nil {
			return nil, fmt.Errorf("asset not found: %v", err)
		}

		detail := entities.DetailBarangMasuk{
			IdMasuk:      purchaseID,
			AssetID:      item.AssetID,
			SerialNumber: item.SerialNumber,
			QtyMasuk:     item.QtyMasuk,
			HargaSatuan:  item.HargaSatuan,
			SubTotal:     item.SubTotal,
			CreatedAt:    time.Now(),
		}

		details = append(details, detail)
		totalItems += item.QtyMasuk
		totalAmount += item.SubTotal
	}

	// Create purchase record
	if err := s.repository.CreatePurchase(purchase, details); err != nil {
		return nil, fmt.Errorf("failed to create purchase: %v", err)
	}

	// Create asset items based on quantities
	var assetItems []entities.AssetItem
	createdItemsCount := 0

	for _, item := range request.Items {
		for i := 0; i < item.QtyMasuk; i++ {
			// Generate unique MAC address (in production, this should be more sophisticated)
			macAddress := s.generateMACAddress()

			assetItem := entities.AssetItem{
				AssetID:    item.AssetID,
				MacAddress: macAddress,
				Status:     "in_stock",
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			assetItems = append(assetItems, assetItem)
			createdItemsCount++
		}
	}

	// Create all asset items
	if err := s.repository.CreateAssetItems(assetItems); err != nil {
		return nil, fmt.Errorf("failed to create asset items: %v", err)
	}

	response := &PurchaseResponse{
		IdMasuk:      purchaseID,
		Date:         request.Date,
		TotalItems:   totalItems,
		TotalAmount:  totalAmount,
		CreatedItems: createdItemsCount,
	}

	return response, nil
}

// CreateDeploymentService handles asset deployment workflow
func (s *InventoryService) CreateDeploymentService(request CreateDeploymentRequest, createdBy string) (*DeploymentResponse, error) {
	// Validate that either customer_installation_id OR trouble_ticket_id is provided
	if request.CustomerInstallationID == nil && request.TroubleTicketID == nil {
		return nil, errors.New("either customer_installation_id or trouble_ticket_id must be provided")
	}

	if request.CustomerInstallationID != nil && request.TroubleTicketID != nil {
		return nil, errors.New("only one of customer_installation_id or trouble_ticket_id should be provided")
	}

	// Get asset item to check current status
	assetItem, err := s.repository.GetAssetItemByID(request.AssetItemID)
	if err != nil {
		return nil, fmt.Errorf("asset item not found: %v", err)
	}

	previousStatus := assetItem.Status
	var newStatus string

	// Determine new status based on transaction type
	if request.TransactionType == "out" {
		if previousStatus != "in_stock" {
			return nil, fmt.Errorf("asset item is not in stock (current status: %s)", previousStatus)
		}
		newStatus = "in_use"
	} else if request.TransactionType == "in" {
		if previousStatus != "in_use" {
			return nil, fmt.Errorf("asset item is not in use (current status: %s)", previousStatus)
		}
		newStatus = "in_stock" // Could also be "maintenance" based on business logic
	} else {
		return nil, errors.New("invalid transaction type")
	}

	// Create transaction record based on type
	var transactionID string
	if request.CustomerInstallationID != nil {
		// Get asset ID from asset item
		assetItem, err := s.repository.GetAssetItemByID(request.AssetItemID)
		if err != nil {
			return nil, fmt.Errorf("failed to get asset item: %v", err)
		}

		// Create asset transaction for customer installation
		transaction := &entities.AssetTransaction{
			CustomerInstallationID: *request.CustomerInstallationID,
			AssetID:                assetItem.AssetID,
			TransactionType:        request.TransactionType,
			Notes:                  request.Notes,
			CreatedBy:              createdBy,
			CreatedAt:              time.Now(),
		}

		if err := s.repository.CreateAssetTransaction(transaction); err != nil {
			return nil, fmt.Errorf("failed to create asset transaction: %v", err)
		}
		transactionID = transaction.ID
	} else {
		// Create ticket asset transaction for trouble ticket
		transaction := &entities.TicketAssetTransaction{
			TroubleTicketID: *request.TroubleTicketID,
			AssetItemID:     request.AssetItemID,
			TransactionType: request.TransactionType,
			Notes:           request.Notes,
			CreatedBy:       createdBy,
			CreatedAt:       time.Now(),
		}

		if err := s.repository.CreateTicketAssetTransaction(transaction); err != nil {
			return nil, fmt.Errorf("failed to create ticket asset transaction: %v", err)
		}
		transactionID = transaction.ID
	}

	// Update asset item status
	if err := s.repository.UpdateAssetItemStatus(request.AssetItemID, newStatus); err != nil {
		return nil, fmt.Errorf("failed to update asset item status: %v", err)
	}

	response := &DeploymentResponse{
		ID:              transactionID,
		AssetItemID:     request.AssetItemID,
		TransactionType: request.TransactionType,
		PreviousStatus:  previousStatus,
		NewStatus:       newStatus,
		CreatedAt:       time.Now().Format("2006-01-02 15:04:05"),
	}

	return response, nil
}

// GetInventoryStatusService retrieves inventory status with filtering
func (s *InventoryService) GetInventoryStatusService(request InventoryStatusRequest) ([]InventoryStatusResponse, error) {
	return s.repository.GetInventoryStatus(request)
}

// generateMACAddress generates a random MAC address for new asset items
func (s *InventoryService) generateMACAddress() string {
	// Generate a random MAC address (in production, this should be more sophisticated)
	rand.Seed(time.Now().UnixNano())

	// Common vendor prefixes for network equipment
	vendorPrefixes := []string{"00:1B:67", "00:0C:29", "00:50:56", "00:1C:42", "00:15:5D"}
	prefix := vendorPrefixes[rand.Intn(len(vendorPrefixes))]

	// Generate random last 3 octets
	octet1 := fmt.Sprintf("%02X", rand.Intn(256))
	octet2 := fmt.Sprintf("%02X", rand.Intn(256))
	octet3 := fmt.Sprintf("%02X", rand.Intn(256))

	return fmt.Sprintf("%s:%s:%s:%s", prefix, octet1, octet2, octet3)
}
