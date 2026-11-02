package items_transaction

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
