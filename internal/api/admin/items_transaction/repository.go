package items_transaction

import (
	"errors"
	"fmt"
	"skripsi-be/internal/models/entities"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Sentinel errors for INV-02 stock transfers. The handler maps them to HTTP status codes.
var (
	ErrTransferInvalidDate       = errors.New("invalid date, expected format YYYY-MM-DD")
	ErrTransferSameHolder        = errors.New("source and destination holder are identical")
	ErrTransferBadSource         = errors.New("source holder does not exist")
	ErrTransferBadDestination    = errors.New("destination holder must be an existing TECHNICIAN user")
	ErrTransferUnknownItem       = errors.New("item not found in the items catalog")
	ErrTransferInsufficientStock = errors.New("insufficient stock at the source holder")
	ErrTransferPairDelete        = errors.New("transaction belongs to a transfer pair and cannot be deleted on its own")
)

type ItemsTransactionRepositoryInterface interface {
	FindItemsTransactionsRepository(request GetItemsTransactionsRequest) ([]ItemsTransactionResponse, error)
	CreateItemsTransactionRepository(request CreateItemsTransactionRequest, createdBy string) (ItemsTransactionResponse, error)
	FindByIdItemsTransactionRepository(request IdItemsTransactionRequest, transactionType string) (ItemsTransactionResponse, error)
	UpdateItemsTransactionRepository(request UpdateItemsTransactionRequest) (ItemsTransactionResponse, error)
	DeleteItemsTransactionRepository(request IdItemsTransactionRequest, transactionType string) error
	DeleteItemsTransactionDetailRepository(request DeleteItemsTransactionDetailRequest) error
	CreateTransferRepository(request CreateTransferRequest, createdBy string) (TransferResponse, error)
	GetStockBalancesRepository(request GetStockBalanceRequest) ([]StockBalanceResponse, error)
	GetStockMutationsRepository(request GetStockMutationsRequest) (StockMutationListResponse, error)
}

type ItemsTransactionRepositoryStruct struct {
	db *gorm.DB
}

func NewItemsTransactionRepository(db *gorm.DB) *ItemsTransactionRepositoryStruct {
	return &ItemsTransactionRepositoryStruct{db}
}

func (r ItemsTransactionRepositoryStruct) FindItemsTransactionsRepository(request GetItemsTransactionsRequest) ([]ItemsTransactionResponse, error) {
	var responses []ItemsTransactionResponse

	// Query items keluar (out)
	if request.TransactionType == nil || *request.TransactionType == "out" {
		var itemsKeluar []entities.ItemsKeluar
		query := r.db.Preload("Details.Item").Preload("User") // Preload Item from catalog

		if request.DateFrom != nil {
			query = query.Where("date >= ?", *request.DateFrom)
		}
		if request.DateTo != nil {
			query = query.Where("date <= ?", *request.DateTo)
		}
		if request.AssetID != nil {
			// Filter by asset_id through items catalog (items can have asset_id)
			query = query.Joins("JOIN detail_itemskeluar ON detail_itemskeluar.IdKeluar = itemskeluar.id").
				Joins("JOIN items ON items.id = detail_itemskeluar.item_id").
				Where("items.asset_id = ?", *request.AssetID)
		}

		if err := query.Order("date DESC, created_at DESC").Find(&itemsKeluar).Error; err != nil {
			return nil, err
		}

		for _, ik := range itemsKeluar {
			items := make([]ItemsTransactionDetailResponse, 0)
			for _, detail := range ik.Details {
				// Use Item from catalog
				var itemName string
				var itemInfo interface{}
				if detail.Item != nil {
					itemName = detail.Item.Name
					itemInfo = detail.Item
				} else {
					itemName = "Unknown Item"
					itemInfo = nil
				}

				items = append(items, ItemsTransactionDetailResponse{
					IdItems:   detail.ItemID,
					ItemName:  itemName,
					Item:      itemInfo, // Item from catalog
					Quantity:  detail.QtyKeluar,
					Unit:      detail.Unit,
					Notes:     detail.Notes,
					CreatedAt: detail.CreatedAt.Format(time.RFC3339),
				})
			}

			responses = append(responses, ItemsTransactionResponse{
				ID:              ik.ID,
				TransactionType: "out",
				Date:            ik.Date.Format("2006-01-02"),
				Notes:           ik.Notes,
				CreatedBy:       ik.CreatedBy,
				CreatedAt:       ik.CreatedAt.Format(time.RFC3339),
				UpdatedAt:       ik.UpdatedAt.Format(time.RFC3339),
				Items:           items,
				User:            ik.User,
			})
		}
	}

	// Query items masuk (in)
	if request.TransactionType == nil || *request.TransactionType == "in" {
		var itemsMasuk []entities.ItemsMasuk
		query := r.db.Preload("Details.Item").Preload("User") // Preload Item from catalog

		if request.DateFrom != nil {
			query = query.Where("date >= ?", *request.DateFrom)
		}
		if request.DateTo != nil {
			query = query.Where("date <= ?", *request.DateTo)
		}
		if request.AssetID != nil {
			// Filter by asset_id through items catalog (items can have asset_id)
			query = query.Joins("JOIN detail_itemsmasuk ON detail_itemsmasuk.IdMasuk = itemsmasuk.id").
				Joins("JOIN items ON items.id = detail_itemsmasuk.item_id").
				Where("items.asset_id = ?", *request.AssetID)
		}

		if err := query.Order("date DESC, created_at DESC").Find(&itemsMasuk).Error; err != nil {
			return nil, err
		}

		for _, im := range itemsMasuk {
			items := make([]ItemsTransactionDetailResponse, 0)
			for _, detail := range im.Details {
				subTotal := detail.SubTotal
				// Use Item from catalog
				var itemName string
				var itemInfo interface{}
				if detail.Item != nil {
					itemName = detail.Item.Name
					itemInfo = detail.Item
				} else {
					itemName = "Unknown Item"
					itemInfo = nil
				}

				items = append(items, ItemsTransactionDetailResponse{
					IdItems:     detail.ItemID,
					ItemName:    itemName,
					Item:        itemInfo, // Item from catalog
					Quantity:    detail.QtyMasuk,
					Unit:        detail.Unit,
					HargaSatuan: &detail.HargaSatuan,
					SubTotal:    &subTotal,
					Notes:       detail.Notes,
					CreatedAt:   detail.CreatedAt.Format(time.RFC3339),
				})
			}

			responses = append(responses, ItemsTransactionResponse{
				ID:              im.ID,
				TransactionType: "in",
				Date:            im.Date.Format("2006-01-02"),
				Notes:           im.Notes,
				CreatedBy:       im.CreatedBy,
				CreatedAt:       im.CreatedAt.Format(time.RFC3339),
				UpdatedAt:       im.UpdatedAt.Format(time.RFC3339),
				Items:           items,
				User:            im.User,
			})
		}
	}

	return responses, nil
}

func (r ItemsTransactionRepositoryStruct) CreateItemsTransactionRepository(request CreateItemsTransactionRequest, createdBy string) (ItemsTransactionResponse, error) {
	// Parse date
	date, err := time.Parse("2006-01-02", request.Date)
	if err != nil {
		return ItemsTransactionResponse{}, fmt.Errorf("invalid date format: %v", err)
	}

	var response ItemsTransactionResponse

	if request.TransactionType == "out" {
		// Create ItemsKeluar
		itemsKeluar := entities.ItemsKeluar{
			ID:        generateItemsTransactionID("IK"),
			Date:      date,
			Notes:     request.Notes,
			CreatedBy: createdBy,
		}

		// Create details
		details := make([]entities.DetailItemsKeluar, 0)
		for _, item := range request.Items {
			// item.IdItems should be item_id from items catalog
			details = append(details, entities.DetailItemsKeluar{
				IdKeluar:  itemsKeluar.ID,
				ItemID:    item.IdItems, // Reference to items catalog (item_id from items table)
				QtyKeluar: item.Quantity,
				Unit:      item.Unit,
				Notes:     item.Notes,
			})
		}

		// Save in transaction
		tx := r.db.Begin()
		if err := tx.Create(&itemsKeluar).Error; err != nil {
			tx.Rollback()
			return ItemsTransactionResponse{}, err
		}

		for _, detail := range details {
			if err := tx.Create(&detail).Error; err != nil {
				tx.Rollback()
				return ItemsTransactionResponse{}, err
			}
		}

		if err := tx.Commit().Error; err != nil {
			return ItemsTransactionResponse{}, err
		}

		// Reload with relations
		r.db.Preload("Details.Item").Preload("User").First(&itemsKeluar, "id = ?", itemsKeluar.ID)

		// Build response
		items := make([]ItemsTransactionDetailResponse, 0)
		for _, detail := range itemsKeluar.Details {
			var itemName string
			var itemInfo interface{}
			if detail.Item != nil {
				itemName = detail.Item.Name
				itemInfo = detail.Item
			} else {
				itemName = "Unknown Item"
				itemInfo = nil
			}

			items = append(items, ItemsTransactionDetailResponse{
				IdItems:   detail.ItemID,
				ItemName:  itemName,
				Item:      itemInfo,
				Quantity:  detail.QtyKeluar,
				Unit:      detail.Unit,
				Notes:     detail.Notes,
				CreatedAt: detail.CreatedAt.Format(time.RFC3339),
			})
		}

		response = ItemsTransactionResponse{
			ID:              itemsKeluar.ID,
			TransactionType: "out",
			Date:            itemsKeluar.Date.Format("2006-01-02"),
			Notes:           itemsKeluar.Notes,
			CreatedBy:       itemsKeluar.CreatedBy,
			CreatedAt:       itemsKeluar.CreatedAt.Format(time.RFC3339),
			UpdatedAt:       itemsKeluar.UpdatedAt.Format(time.RFC3339),
			Items:           items,
			User:            itemsKeluar.User,
		}
	} else {
		// Create ItemsMasuk
		itemsMasuk := entities.ItemsMasuk{
			ID:        generateItemsTransactionID("IM"),
			Date:      date,
			Notes:     request.Notes,
			CreatedBy: createdBy,
		}

		// Create details
		details := make([]entities.DetailItemsMasuk, 0)
		for _, item := range request.Items {
			hargaSatuan := 0
			if item.HargaSatuan != nil {
				hargaSatuan = *item.HargaSatuan
			}
			subTotal := hargaSatuan * item.Quantity

			// item.IdItems should be item_id from items catalog
			details = append(details, entities.DetailItemsMasuk{
				IdMasuk:     itemsMasuk.ID,
				ItemID:      item.IdItems, // Reference to items catalog (item_id from items table)
				QtyMasuk:    item.Quantity,
				HargaSatuan: hargaSatuan,
				SubTotal:    subTotal,
				Unit:        item.Unit,
				Notes:       item.Notes,
			})
		}

		// Save in transaction
		tx := r.db.Begin()
		if err := tx.Create(&itemsMasuk).Error; err != nil {
			tx.Rollback()
			return ItemsTransactionResponse{}, err
		}

		for _, detail := range details {
			if err := tx.Create(&detail).Error; err != nil {
				tx.Rollback()
				return ItemsTransactionResponse{}, err
			}
		}

		if err := tx.Commit().Error; err != nil {
			return ItemsTransactionResponse{}, err
		}

		// Reload with relations
		r.db.Preload("Details.Item").Preload("User").First(&itemsMasuk, "id = ?", itemsMasuk.ID)

		// Build response
		items := make([]ItemsTransactionDetailResponse, 0)
		for _, detail := range itemsMasuk.Details {
			subTotal := detail.SubTotal
			var itemName string
			var itemInfo interface{}
			if detail.Item != nil {
				itemName = detail.Item.Name
				itemInfo = detail.Item
			} else {
				itemName = "Unknown Item"
				itemInfo = nil
			}

			items = append(items, ItemsTransactionDetailResponse{
				IdItems:     detail.ItemID,
				ItemName:    itemName,
				Item:        itemInfo,
				Quantity:    detail.QtyMasuk,
				Unit:        detail.Unit,
				HargaSatuan: &detail.HargaSatuan,
				SubTotal:    &subTotal,
				Notes:       detail.Notes,
				CreatedAt:   detail.CreatedAt.Format(time.RFC3339),
			})
		}

		response = ItemsTransactionResponse{
			ID:              itemsMasuk.ID,
			TransactionType: "in",
			Date:            itemsMasuk.Date.Format("2006-01-02"),
			Notes:           itemsMasuk.Notes,
			CreatedBy:       itemsMasuk.CreatedBy,
			CreatedAt:       itemsMasuk.CreatedAt.Format(time.RFC3339),
			UpdatedAt:       itemsMasuk.UpdatedAt.Format(time.RFC3339),
			Items:           items,
			User:            itemsMasuk.User,
		}
	}

	return response, nil
}

// CreateTransferRepository records an INV-02 stock transfer as one atomic pair: an itemskeluar row
// for the source holder and an itemsmasuk row for the destination holder, sharing one transfer_ref.
// The user who records it is only the actor and goes into created_by.
//
// Concurrency: the involved items rows are locked with SELECT ... FOR UPDATE as the FIRST statement
// of the transaction, in ascending id order, so two transfers of the same item queue up and cannot
// deadlock. The stock read happens after the lock is held, so it sees every committed transfer.
// LIMITATION: the stock guarantee only holds between transfers. The legacy POST / endpoint does not
// take this lock and can still push a balance below zero.
func (r ItemsTransactionRepositoryStruct) CreateTransferRepository(request CreateTransferRequest, createdBy string) (TransferResponse, error) {
	date, err := time.Parse("2006-01-02", request.Date)
	if err != nil {
		return TransferResponse{}, ErrTransferInvalidDate
	}

	destination := strings.TrimSpace(request.DestinationHolderID)
	var source *string // nil = main warehouse
	if request.SourceHolderID != nil {
		if v := strings.TrimSpace(*request.SourceHolderID); v != "" {
			source = &v
		}
	}
	if source != nil && *source == destination {
		return TransferResponse{}, ErrTransferSameHolder
	}

	// Sum the requested quantity per (item, unit) BEFORE checking stock, otherwise two lines of
	// 5 against a stock of 8 would each pass on their own.
	type stockKey struct {
		ItemID string
		Unit   string
	}
	needed := make(map[stockKey]int, len(request.Items))
	notes := make(map[stockKey]*string, len(request.Items))
	keys := make([]stockKey, 0, len(request.Items))
	idSet := make(map[string]struct{}, len(request.Items))
	for _, item := range request.Items {
		key := stockKey{ItemID: strings.TrimSpace(item.IdItems), Unit: strings.TrimSpace(item.Unit)}
		if key.ItemID == "" {
			return TransferResponse{}, ErrTransferUnknownItem
		}
		if _, seen := needed[key]; !seen {
			keys = append(keys, key)
			notes[key] = item.Notes
		}
		needed[key] += item.Quantity
		idSet[key.ItemID] = struct{}{}
	}
	itemIDs := make([]string, 0, len(idSet))
	for id := range idSet {
		itemIDs = append(itemIDs, id)
	}
	sort.Strings(itemIDs)
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].ItemID != keys[j].ItemID {
			return keys[i].ItemID < keys[j].ItemID
		}
		return keys[i].Unit < keys[j].Unit
	})

	tx := r.db.Begin()
	if tx.Error != nil {
		return TransferResponse{}, tx.Error
	}
	defer func() {
		if rec := recover(); rec != nil {
			tx.Rollback()
			panic(rec)
		}
	}()
	fail := func(err error) (TransferResponse, error) {
		tx.Rollback()
		return TransferResponse{}, err
	}

	// 1. Lock the item rows (first statement of the transaction) and use the result as the
	// existence check: every requested id must come back.
	var lockedItems []entities.Item
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").
		Where("id IN ?", itemIDs).
		Order("id ASC").
		Find(&lockedItems).Error; err != nil {
		return fail(err)
	}
	if len(lockedItems) != len(itemIDs) {
		return fail(ErrTransferUnknownItem)
	}

	// 2. Destination must be an existing TECHNICIAN user.
	var destinationCount int64
	if err := tx.Table("users AS u").
		Joins("JOIN roles AS r ON r.id = u.role_id").
		Where("u.id = ? AND r.name = ?", destination, "TECHNICIAN").
		Count(&destinationCount).Error; err != nil {
		return fail(err)
	}
	if destinationCount == 0 {
		return fail(ErrTransferBadDestination)
	}

	// 3. A named source must be an existing user (the foreign key would reject it anyway, but
	// that would surface as a 500).
	if source != nil {
		var sourceCount int64
		if err := tx.Table("users").Where("id = ?", *source).Count(&sourceCount).Error; err != nil {
			return fail(err)
		}
		if sourceCount == 0 {
			return fail(ErrTransferBadSource)
		}
	}

	// 4. Stock check at the SOURCE holder, per (item, unit).
	for _, key := range keys {
		available, err := transferHolderStock(tx, source, key.ItemID, key.Unit)
		if err != nil {
			return fail(err)
		}
		if available < int64(needed[key]) {
			return fail(fmt.Errorf("%w: item %s (%s) requested %d, available %d",
				ErrTransferInsufficientStock, key.ItemID, key.Unit, needed[key], available))
		}
	}

	// 5. Write the pair.
	transferRef := strings.ReplaceAll(uuid.New().String(), "-", "") // 32 hex chars = CHAR(32)
	outgoing := entities.ItemsKeluar{
		ID:          generateItemsTransactionID("IK"),
		Date:        date,
		Notes:       request.Notes,
		HolderID:    source,
		TransferRef: &transferRef,
		CreatedBy:   createdBy,
	}
	incoming := entities.ItemsMasuk{
		ID:          generateItemsTransactionID("IM"),
		Date:        date,
		Notes:       request.Notes,
		HolderID:    &destination,
		TransferRef: &transferRef,
		CreatedBy:   createdBy,
	}
	if err := tx.Create(&outgoing).Error; err != nil {
		return fail(err)
	}
	if err := tx.Create(&incoming).Error; err != nil {
		return fail(err)
	}

	outDetails := make([]entities.DetailItemsKeluar, 0, len(keys))
	inDetails := make([]entities.DetailItemsMasuk, 0, len(keys))
	responseItems := make([]TransferItemResponse, 0, len(keys))
	for _, key := range keys {
		qty := needed[key]
		outDetails = append(outDetails, entities.DetailItemsKeluar{
			IdKeluar:  outgoing.ID,
			ItemID:    key.ItemID,
			QtyKeluar: qty,
			Unit:      key.Unit,
			Notes:     notes[key],
		})
		// HargaSatuan and SubTotal are NOT NULL in the database; a transfer carries no price.
		inDetails = append(inDetails, entities.DetailItemsMasuk{
			IdMasuk:     incoming.ID,
			ItemID:      key.ItemID,
			QtyMasuk:    qty,
			HargaSatuan: 0,
			SubTotal:    0,
			Unit:        key.Unit,
			Notes:       notes[key],
		})
		responseItems = append(responseItems, TransferItemResponse{
			IdItems:  key.ItemID,
			Quantity: qty,
			Unit:     key.Unit,
			Notes:    notes[key],
		})
	}
	if err := tx.Create(&outDetails).Error; err != nil {
		return fail(err)
	}
	if err := tx.Create(&inDetails).Error; err != nil {
		return fail(err)
	}

	if err := tx.Commit().Error; err != nil {
		return TransferResponse{}, err
	}

	return TransferResponse{
		TransferRef:         transferRef,
		OutgoingID:          outgoing.ID,
		IncomingID:          incoming.ID,
		Date:                date.Format("2006-01-02"),
		SourceHolderID:      source,
		DestinationHolderID: destination,
		Notes:               request.Notes,
		CreatedBy:           createdBy,
		Items:               responseItems,
	}, nil
}

// transferHolderStock returns the stock of (itemID, unit) held by one holder: everything received
// (itemsmasuk) minus everything released (itemskeluar). holder == nil is the main warehouse and
// matches holder_id IS NULL, which also covers every legacy row. NULL quantities count as 0.
// Stock is never summed across holders or across units.
func transferHolderStock(tx *gorm.DB, holder *string, itemID, unit string) (int64, error) {
	var totalIn, totalOut int64

	incoming := tx.Table("detail_itemsmasuk AS d").
		Joins("JOIN itemsmasuk AS h ON h.id = d.IdMasuk").
		Where("d.item_id = ? AND d.unit = ?", itemID, unit)
	if holder == nil {
		incoming = incoming.Where("h.holder_id IS NULL")
	} else {
		incoming = incoming.Where("h.holder_id = ?", *holder)
	}
	if err := incoming.Select("COALESCE(SUM(COALESCE(d.QtyMasuk, 0)), 0)").Scan(&totalIn).Error; err != nil {
		return 0, err
	}

	outgoing := tx.Table("detail_itemskeluar AS d").
		Joins("JOIN itemskeluar AS h ON h.id = d.IdKeluar").
		Where("d.item_id = ? AND d.unit = ?", itemID, unit)
	if holder == nil {
		outgoing = outgoing.Where("h.holder_id IS NULL")
	} else {
		outgoing = outgoing.Where("h.holder_id = ?", *holder)
	}
	if err := outgoing.Select("COALESCE(SUM(COALESCE(d.QtyKeluar, 0)), 0)").Scan(&totalOut).Error; err != nil {
		return 0, err
	}

	return totalIn - totalOut, nil
}

// WarehouseLabel is the holder name shown for rows whose holder_id is NULL.
const WarehouseLabel = "Gudang Utama"

// stockBalanceUnionSQL lists every stock movement as a signed quantity (+ received, - released)
// together with its holder. holder_id NULL = main warehouse, which also covers every legacy row.
// NULL quantities count as 0 and a NULL unit counts as the column default (PCS).
const stockBalanceUnionSQL = `
SELECT d.item_id AS item_id, COALESCE(d.unit, 'PCS') AS unit, h.holder_id AS holder_id, COALESCE(d.QtyMasuk, 0) AS qty
FROM detail_itemsmasuk AS d
JOIN itemsmasuk AS h ON h.id = d.IdMasuk
UNION ALL
SELECT d.item_id, COALESCE(d.unit, 'PCS'), h.holder_id, -COALESCE(d.QtyKeluar, 0)
FROM detail_itemskeluar AS d
JOIN itemskeluar AS h ON h.id = d.IdKeluar`

// stockMutationUnionSQL lists every line of every transaction. A row that carries a transfer_ref is
// one half of a transfer. Quantities are positive; the kind tells the direction.
const stockMutationUnionSQL = `
SELECT h.id AS ref,
	   CASE WHEN h.transfer_ref IS NULL THEN 'in' ELSE 'transfer_in' END AS kind,
	   h.transfer_ref AS transfer_ref,
	   h.date AS tx_date,
	   d.item_id AS item_id,
	   COALESCE(d.unit, 'PCS') AS unit,
	   COALESCE(d.QtyMasuk, 0) AS quantity,
	   h.holder_id AS holder_id,
	   h.created_by AS recorded_by_id,
	   h.notes AS notes,
	   d.notes AS item_notes,
	   h.created_at AS created_at
FROM detail_itemsmasuk AS d
JOIN itemsmasuk AS h ON h.id = d.IdMasuk
UNION ALL
SELECT h.id,
	   CASE WHEN h.transfer_ref IS NULL THEN 'out' ELSE 'transfer_out' END,
	   h.transfer_ref,
	   h.date,
	   d.item_id,
	   COALESCE(d.unit, 'PCS'),
	   COALESCE(d.QtyKeluar, 0),
	   h.holder_id,
	   h.created_by,
	   h.notes,
	   d.notes,
	   h.created_at
FROM detail_itemskeluar AS d
JOIN itemskeluar AS h ON h.id = d.IdKeluar`

// holderCondition builds the SQL for a holder filter. column is always a constant from this file,
// never user input; the holder value itself is passed as a bound argument.
func holderCondition(column string, holder string) (string, []interface{}) {
	switch holder {
	case "":
		return "", nil
	case HolderWarehouse:
		return " AND " + column + " IS NULL", nil
	default:
		return " AND " + column + " = ?", []interface{}{holder}
	}
}

// loadNames returns id -> name for the given ids of one table ("items" or "users").
func (r ItemsTransactionRepositoryStruct) loadNames(table string, ids []string) (map[string]string, error) {
	names := make(map[string]string, len(ids))
	if len(ids) == 0 {
		return names, nil
	}

	var rows []struct {
		ID   string `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	if err := r.db.Table(table).Select("id, COALESCE(name, '') AS name").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		names[row.ID] = row.Name
	}
	return names, nil
}

// displayName falls back to the id when the name is unknown or empty.
func displayName(id string, names map[string]string) string {
	if name := names[id]; name != "" {
		return name
	}
	return id
}

func holderDisplayName(holderID *string, names map[string]string) string {
	if holderID == nil {
		return WarehouseLabel
	}
	return displayName(*holderID, names)
}

func uniqueIDs(groups ...[]string) []string {
	seen := make(map[string]struct{})
	ids := make([]string, 0)
	for _, group := range groups {
		for _, id := range group {
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

type stockBalanceRow struct {
	ItemID   string  `gorm:"column:item_id"`
	Unit     string  `gorm:"column:unit"`
	HolderID *string `gorm:"column:holder_id"`
	Balance  int64   `gorm:"column:balance"`
}

// GetStockBalancesRepository returns the stock per (holder, item, unit): everything the holder
// received minus everything it released. Units and holders are never summed together. Rows whose
// item no longer exists (item_id NULL) are left out.
func (r ItemsTransactionRepositoryStruct) GetStockBalancesRepository(request GetStockBalanceRequest) ([]StockBalanceResponse, error) {
	query := "SELECT m.item_id AS item_id, m.unit AS unit, m.holder_id AS holder_id, SUM(m.qty) AS balance FROM (" +
		stockBalanceUnionSQL + ") AS m WHERE m.item_id IS NOT NULL"
	args := make([]interface{}, 0, 4)

	if request.ItemID != "" {
		query += " AND m.item_id = ?"
		args = append(args, request.ItemID)
	}
	if request.Unit != "" {
		query += " AND m.unit = ?"
		args = append(args, request.Unit)
	}
	condition, conditionArgs := holderCondition("m.holder_id", request.HolderID)
	query += condition
	args = append(args, conditionArgs...)

	query += " GROUP BY m.item_id, m.unit, m.holder_id"
	if !request.IncludeZero {
		query += " HAVING SUM(m.qty) <> 0"
	}

	var rows []stockBalanceRow
	if err := r.db.Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	itemIDs := make([]string, 0, len(rows))
	holderIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		itemIDs = append(itemIDs, row.ItemID)
		if row.HolderID != nil {
			holderIDs = append(holderIDs, *row.HolderID)
		}
	}
	itemNames, err := r.loadNames("items", uniqueIDs(itemIDs))
	if err != nil {
		return nil, err
	}
	userNames, err := r.loadNames("users", uniqueIDs(holderIDs))
	if err != nil {
		return nil, err
	}

	balances := make([]StockBalanceResponse, 0, len(rows))
	for _, row := range rows {
		balances = append(balances, StockBalanceResponse{
			ItemID:     row.ItemID,
			ItemName:   displayName(row.ItemID, itemNames),
			Unit:       row.Unit,
			HolderID:   row.HolderID,
			HolderName: holderDisplayName(row.HolderID, userNames),
			Balance:    row.Balance,
		})
	}
	sort.SliceStable(balances, func(i, j int) bool {
		a, b := balances[i], balances[j]
		if a.ItemName != b.ItemName {
			return a.ItemName < b.ItemName
		}
		if a.ItemID != b.ItemID {
			return a.ItemID < b.ItemID
		}
		if a.Unit != b.Unit {
			return a.Unit < b.Unit
		}
		if (a.HolderID == nil) != (b.HolderID == nil) {
			return a.HolderID == nil // main warehouse first
		}
		return a.HolderName < b.HolderName
	})
	return balances, nil
}

type stockMutationRow struct {
	Ref          string  `gorm:"column:ref"`
	Kind         string  `gorm:"column:kind"`
	TransferRef  *string `gorm:"column:transfer_ref"`
	TxDate       string  `gorm:"column:tx_date"`
	ItemID       string  `gorm:"column:item_id"`
	Unit         string  `gorm:"column:unit"`
	Quantity     int64   `gorm:"column:quantity"`
	HolderID     *string `gorm:"column:holder_id"`
	RecordedByID string  `gorm:"column:recorded_by_id"`
	Notes        *string `gorm:"column:notes"`
	ItemNotes    *string `gorm:"column:item_notes"`
	CreatedAt    string  `gorm:"column:created_at"`
}

// GetStockMutationsRepository returns one page of the mutation history, newest first, with the
// names of the item, the holder and the person who recorded the transaction.
func (r ItemsTransactionRepositoryStruct) GetStockMutationsRepository(request GetStockMutationsRequest) (StockMutationListResponse, error) {
	where := " WHERE 1 = 1"
	args := make([]interface{}, 0, 8)

	if request.ItemID != "" {
		where += " AND t.item_id = ?"
		args = append(args, request.ItemID)
	}
	condition, conditionArgs := holderCondition("t.holder_id", request.HolderID)
	where += condition
	args = append(args, conditionArgs...)
	if request.Kind != "" {
		where += " AND t.kind = ?"
		args = append(args, request.Kind)
	}
	if request.DateFrom != "" {
		where += " AND t.tx_date >= ?"
		args = append(args, request.DateFrom)
	}
	if request.DateTo != "" {
		where += " AND t.tx_date <= ?"
		args = append(args, request.DateTo)
	}

	result := StockMutationListResponse{Limit: request.Limit, Offset: request.Offset, Items: []StockMutationResponse{}}

	if err := r.db.Raw("SELECT COUNT(*) FROM ("+stockMutationUnionSQL+") AS t"+where, args...).Scan(&result.Total).Error; err != nil {
		return StockMutationListResponse{}, err
	}
	if result.Total == 0 {
		return result, nil
	}

	listQuery := "SELECT t.ref AS ref, t.kind AS kind, t.transfer_ref AS transfer_ref, " +
		"DATE_FORMAT(t.tx_date, '%Y-%m-%d') AS tx_date, t.item_id AS item_id, t.unit AS unit, " +
		"t.quantity AS quantity, t.holder_id AS holder_id, t.recorded_by_id AS recorded_by_id, " +
		"t.notes AS notes, t.item_notes AS item_notes, " +
		"COALESCE(DATE_FORMAT(t.created_at, '%Y-%m-%dT%H:%i:%s'), '') AS created_at " +
		"FROM (" + stockMutationUnionSQL + ") AS t" + where +
		" ORDER BY t.tx_date DESC, t.created_at DESC, t.ref DESC, t.item_id ASC LIMIT ? OFFSET ?"
	listArgs := append(append(make([]interface{}, 0, len(args)+2), args...), request.Limit, request.Offset)

	var rows []stockMutationRow
	if err := r.db.Raw(listQuery, listArgs...).Scan(&rows).Error; err != nil {
		return StockMutationListResponse{}, err
	}

	itemIDs := make([]string, 0, len(rows))
	userIDs := make([]string, 0, len(rows)*2)
	for _, row := range rows {
		itemIDs = append(itemIDs, row.ItemID)
		userIDs = append(userIDs, row.RecordedByID)
		if row.HolderID != nil {
			userIDs = append(userIDs, *row.HolderID)
		}
	}
	itemNames, err := r.loadNames("items", uniqueIDs(itemIDs))
	if err != nil {
		return StockMutationListResponse{}, err
	}
	userNames, err := r.loadNames("users", uniqueIDs(userIDs))
	if err != nil {
		return StockMutationListResponse{}, err
	}

	for _, row := range rows {
		result.Items = append(result.Items, StockMutationResponse{
			Ref:            row.Ref,
			Kind:           row.Kind,
			TransferRef:    row.TransferRef,
			Date:           row.TxDate,
			ItemID:         row.ItemID,
			ItemName:       displayName(row.ItemID, itemNames),
			Unit:           row.Unit,
			Quantity:       row.Quantity,
			HolderID:       row.HolderID,
			HolderName:     holderDisplayName(row.HolderID, userNames),
			RecordedByID:   row.RecordedByID,
			RecordedByName: displayName(row.RecordedByID, userNames),
			Notes:          row.Notes,
			ItemNotes:      row.ItemNotes,
			CreatedAt:      row.CreatedAt,
		})
	}
	return result, nil
}

func (r ItemsTransactionRepositoryStruct) FindByIdItemsTransactionRepository(request IdItemsTransactionRequest, transactionType string) (ItemsTransactionResponse, error) {
	// Implementation similar to FindItemsTransactionsRepository but for single record
	// For brevity, I'll implement a simplified version
	return ItemsTransactionResponse{}, fmt.Errorf("not implemented yet")
}

func (r ItemsTransactionRepositoryStruct) UpdateItemsTransactionRepository(request UpdateItemsTransactionRequest) (ItemsTransactionResponse, error) {
	// Implementation for update
	return ItemsTransactionResponse{}, fmt.Errorf("not implemented yet")
}

// DeleteItemsTransactionRepository deletes one transaction row but never one that belongs to a
// transfer pair. The guard is part of the DELETE itself (transfer_ref IS NULL), so a concurrent
// request cannot slip in between a check and the delete. Zero deleted rows means either the row is
// one half of a transfer (ErrTransferPairDelete) or it does not exist (nil, as before).
func (r ItemsTransactionRepositoryStruct) DeleteItemsTransactionRepository(request IdItemsTransactionRequest, transactionType string) error {
	table := "itemsmasuk"
	var result *gorm.DB
	if transactionType == "out" {
		table = "itemskeluar"
		result = r.db.Where("id = ? AND transfer_ref IS NULL", request.Id).Delete(&entities.ItemsKeluar{})
	} else {
		result = r.db.Where("id = ? AND transfer_ref IS NULL", request.Id).Delete(&entities.ItemsMasuk{})
	}
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}

	var remaining int64
	if err := r.db.Table(table).Where("id = ?", request.Id).Count(&remaining).Error; err != nil {
		return err
	}
	if remaining > 0 {
		return ErrTransferPairDelete
	}
	return nil
}

func (r ItemsTransactionRepositoryStruct) DeleteItemsTransactionDetailRepository(request DeleteItemsTransactionDetailRequest) error {
	if request.TransactionType == "out" {
		return r.db.Where("IdKeluar = ? AND IdItems = ?", request.TransactionID, request.DetailID).
			Delete(&entities.DetailItemsKeluar{}).Error
	}
	return r.db.Where("IdMasuk = ? AND IdItems = ?", request.TransactionID, request.DetailID).
		Delete(&entities.DetailItemsMasuk{}).Error
}

// Helper function to generate transaction IDs
func generateItemsTransactionID(prefix string) string {
	// Simple ID generation - in production, use proper sequence
	uuid := uuid.New().String()
	return prefix + uuid[:4]
}
