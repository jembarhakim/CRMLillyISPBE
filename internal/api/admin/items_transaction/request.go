package items_transaction

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type CreateItemsTransactionRequest struct {
	TransactionType string                         `json:"transaction_type" validate:"required,oneof=out in"` // 'out' for keluar, 'in' for masuk
	Date            string                         `json:"date" validate:"required"`                           // Date in YYYY-MM-DD format
	Notes           *string                        `json:"notes"`
	Items           []CreateItemsTransactionDetail `json:"items" validate:"required,min=1"` // At least one item required
}

type CreateItemsTransactionDetail struct {
	IdItems     string  `json:"id_items" validate:"required"`      // Asset ID
	Quantity    int     `json:"quantity" validate:"required,min=1"` // Quantity
	Unit        string  `json:"unit" validate:"required"`         // Unit (PCS, M, KG, etc.)
	HargaSatuan *int    `json:"harga_satuan"`                     // Unit price (required for masuk, optional for keluar)
	Notes       *string `json:"notes"`                            // Item-specific notes
}

type GetItemsTransactionsRequest struct {
	TransactionType *string `json:"transaction_type"` // 'out' or 'in'
	DateFrom        *string `json:"date_from"`        // Filter from date
	DateTo          *string `json:"date_to"`          // Filter to date
	AssetID         *string `json:"asset_id"`         // Filter by asset
}

type IdItemsTransactionRequest struct {
	Id string `json:"id" validate:"required"`
}

type UpdateItemsTransactionRequest struct {
	IdItemsTransactionRequest
	CreateItemsTransactionRequest
}

type DeleteItemsTransactionDetailRequest struct {
	TransactionType string `json:"transaction_type" validate:"required,oneof=out in"`
	TransactionID   string `json:"transaction_id" validate:"required"`
	DetailID       string `json:"detail_id" validate:"required"` // For now, we'll use composite key (IdKeluar/IdMasuk + IdItems)
}

// CreateTransferItemRequest is intentionally separate from CreateItemsTransactionDetail so the
// unit rule below applies to transfers only and the legacy POST / route keeps its behaviour.
type CreateTransferItemRequest struct {
	IdItems  string  `json:"id_items" validate:"required"`
	Quantity int     `json:"quantity" validate:"required,min=1"`
	Unit     string  `json:"unit" validate:"required,oneof=PCS M KG BOX ROLL"`
	Notes    *string `json:"notes"`
}

// CreateTransferRequest - INV-02 stock transfer between two holders.
// SourceHolderID empty/absent means the main warehouse. The user who records the transfer
// (the token owner) is only the actor and is stored in created_by, never as the source holder.
type CreateTransferRequest struct {
	SourceHolderID      *string                     `json:"source_holder_id"`
	DestinationHolderID string                      `json:"destination_holder_id" validate:"required"` // must be a TECHNICIAN user
	Date                string                      `json:"date" validate:"required"`                  // YYYY-MM-DD
	Notes               *string                     `json:"notes"`
	Items               []CreateTransferItemRequest `json:"items" validate:"required,min=1,dive"`
}

// ErrStockQueryInvalid wraps every problem found in the query string of the stock endpoints (HTTP 400).
var ErrStockQueryInvalid = errors.New("invalid query parameter")

// HolderWarehouse is the holder_id filter value that selects the main warehouse (holder_id IS NULL).
const HolderWarehouse = "warehouse"

const (
	defaultMutationLimit = 50
	maxMutationLimit     = 200
)

var (
	stockUnits    = []string{"PCS", "M", "KG", "BOX", "ROLL"}
	mutationKinds = []string{"in", "out", "transfer_in", "transfer_out"}
)

func oneOf(value string, allowed []string) bool {
	for _, candidate := range allowed {
		if candidate == value {
			return true
		}
	}
	return false
}

// GetStockBalanceRequest filters the stock balance per holder.
// HolderID: empty = every holder, HolderWarehouse = main warehouse, otherwise a users.id.
type GetStockBalanceRequest struct {
	ItemID      string
	HolderID    string
	Unit        string
	IncludeZero bool // rows whose balance is 0 are hidden unless this is true
}

// NewGetStockBalanceRequest builds and validates the filters from raw query-string values.
func NewGetStockBalanceRequest(itemID, holderID, unit, includeZero string) (GetStockBalanceRequest, error) {
	request := GetStockBalanceRequest{
		ItemID:   strings.TrimSpace(itemID),
		HolderID: strings.TrimSpace(holderID),
		Unit:     strings.TrimSpace(unit),
	}
	if request.Unit != "" && !oneOf(request.Unit, stockUnits) {
		return request, fmt.Errorf("%w: unit must be one of %s", ErrStockQueryInvalid, strings.Join(stockUnits, ", "))
	}
	switch strings.ToLower(strings.TrimSpace(includeZero)) {
	case "", "false", "0":
	case "true", "1":
		request.IncludeZero = true
	default:
		return request, fmt.Errorf("%w: include_zero must be true or false", ErrStockQueryInvalid)
	}
	return request, nil
}

// GetStockMutationsRequest filters the mutation history. Kind is one of in, out, transfer_in, transfer_out.
type GetStockMutationsRequest struct {
	ItemID   string
	HolderID string
	Kind     string
	DateFrom string // YYYY-MM-DD, inclusive
	DateTo   string // YYYY-MM-DD, inclusive
	Limit    int
	Offset   int
}

// NewGetStockMutationsRequest builds and validates the filters from raw query-string values.
func NewGetStockMutationsRequest(itemID, holderID, kind, dateFrom, dateTo, limit, offset string) (GetStockMutationsRequest, error) {
	request := GetStockMutationsRequest{
		ItemID:   strings.TrimSpace(itemID),
		HolderID: strings.TrimSpace(holderID),
		Kind:     strings.TrimSpace(kind),
		DateFrom: strings.TrimSpace(dateFrom),
		DateTo:   strings.TrimSpace(dateTo),
		Limit:    defaultMutationLimit,
	}
	if request.Kind != "" && !oneOf(request.Kind, mutationKinds) {
		return request, fmt.Errorf("%w: kind must be one of %s", ErrStockQueryInvalid, strings.Join(mutationKinds, ", "))
	}
	for name, value := range map[string]string{"date_from": request.DateFrom, "date_to": request.DateTo} {
		if value == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return request, fmt.Errorf("%w: %s must use the format YYYY-MM-DD", ErrStockQueryInvalid, name)
		}
	}
	if request.DateFrom != "" && request.DateTo != "" && request.DateFrom > request.DateTo {
		return request, fmt.Errorf("%w: date_from is after date_to", ErrStockQueryInvalid)
	}
	if value := strings.TrimSpace(limit); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > maxMutationLimit {
			return request, fmt.Errorf("%w: limit must be between 1 and %d", ErrStockQueryInvalid, maxMutationLimit)
		}
		request.Limit = n
	}
	if value := strings.TrimSpace(offset); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return request, fmt.Errorf("%w: offset must be 0 or more", ErrStockQueryInvalid)
		}
		request.Offset = n
	}
	return request, nil
}
