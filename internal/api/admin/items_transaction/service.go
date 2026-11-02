package items_transaction

import "skripsi-be/internal/models/entities"

type ItemsTransactionServiceInterface interface {
	GetItemsTransactionsService(request GetItemsTransactionsRequest) ([]ItemsTransactionResponse, error)
	CreateItemsTransactionService(request CreateItemsTransactionRequest, createdBy string) (ItemsTransactionResponse, error)
	GetByIdItemsTransactionService(request IdItemsTransactionRequest, transactionType string) (ItemsTransactionResponse, error)
	UpdateItemsTransactionService(request UpdateItemsTransactionRequest) (ItemsTransactionResponse, error)
	DeleteItemsTransactionService(request IdItemsTransactionRequest, transactionType string) error
	DeleteItemsTransactionDetailService(request DeleteItemsTransactionDetailRequest) error
	CreateTransferService(request CreateTransferRequest, createdBy string) (TransferResponse, error)
}

type ItemsTransactionServiceStruct struct {
	repository ItemsTransactionRepositoryInterface
}

func NewItemsTransactionService(repository ItemsTransactionRepositoryInterface) ItemsTransactionServiceStruct {
	return ItemsTransactionServiceStruct{repository}
}

func (s ItemsTransactionServiceStruct) GetItemsTransactionsService(request GetItemsTransactionsRequest) ([]ItemsTransactionResponse, error) {
	return s.repository.FindItemsTransactionsRepository(request)
}

func (s ItemsTransactionServiceStruct) CreateItemsTransactionService(request CreateItemsTransactionRequest, createdBy string) (ItemsTransactionResponse, error) {
	return s.repository.CreateItemsTransactionRepository(request, createdBy)
}

func (s ItemsTransactionServiceStruct) GetByIdItemsTransactionService(request IdItemsTransactionRequest, transactionType string) (ItemsTransactionResponse, error) {
	return s.repository.FindByIdItemsTransactionRepository(request, transactionType)
}

func (s ItemsTransactionServiceStruct) UpdateItemsTransactionService(request UpdateItemsTransactionRequest) (ItemsTransactionResponse, error) {
	return s.repository.UpdateItemsTransactionRepository(request)
}

func (s ItemsTransactionServiceStruct) DeleteItemsTransactionService(request IdItemsTransactionRequest, transactionType string) error {
	return s.repository.DeleteItemsTransactionRepository(request, transactionType)
}

func (s ItemsTransactionServiceStruct) DeleteItemsTransactionDetailService(request DeleteItemsTransactionDetailRequest) error {
	return s.repository.DeleteItemsTransactionDetailRepository(request)
}

func (s ItemsTransactionServiceStruct) CreateTransferService(request CreateTransferRequest, createdBy string) (TransferResponse, error) {
	return s.repository.CreateTransferRepository(request, createdBy)
}

// TransferItemResponse is one (item, unit) line of a transfer.
type TransferItemResponse struct {
	IdItems  string  `json:"id_items"`
	Quantity int     `json:"quantity"`
	Unit     string  `json:"unit"`
	Notes    *string `json:"notes,omitempty"`
}

// TransferResponse describes a stock transfer: one outgoing and one incoming row sharing TransferRef.
type TransferResponse struct {
	TransferRef         string                 `json:"transfer_ref"`
	OutgoingID          string                 `json:"outgoing_id"`
	IncomingID          string                 `json:"incoming_id"`
	Date                string                 `json:"date"`
	SourceHolderID      *string                `json:"source_holder_id"` // null = main warehouse
	DestinationHolderID string                 `json:"destination_holder_id"`
	Notes               *string                `json:"notes,omitempty"`
	CreatedBy           string                 `json:"created_by"`
	Items               []TransferItemResponse `json:"items"`
}

// ItemsTransactionResponse - Unified response structure
type ItemsTransactionResponse struct {
	ID              string                         `json:"id"`
	TransactionType string                         `json:"transaction_type"` // 'out' or 'in'
	Date            string                         `json:"date"`
	Notes           *string                        `json:"notes,omitempty"`
	CreatedBy       string                         `json:"created_by"`
	CreatedAt       string                         `json:"created_at"`
	UpdatedAt       string                         `json:"updated_at"`
	Items           []ItemsTransactionDetailResponse `json:"items"`
	User            *entities.User                 `json:"user,omitempty"`
}

type ItemsTransactionDetailResponse struct {
	IdItems     string      `json:"id_items"`      // item_id from items catalog (preferred) or IdItems (legacy asset_id)
	ItemName    string      `json:"item_name"`    // Name of item from catalog or asset
	Item        interface{} `json:"item,omitempty"` // Item (from catalog) or Asset (legacy) object
	Quantity    int         `json:"quantity"`
	Unit        string      `json:"unit"`
	HargaSatuan *int        `json:"harga_satuan,omitempty"`
	SubTotal    *int        `json:"sub_total,omitempty"`
	Notes       *string     `json:"notes,omitempty"`
	CreatedAt   string      `json:"created_at"`
}

