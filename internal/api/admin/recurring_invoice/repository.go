package recurring_invoice

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"skripsi-be/internal/models/entities"
	"skripsi-be/internal/services"

	"github.com/google/uuid"
)

// custID dereferences a nullable customer id ("" when nil).
func custID(id *string) string {
	if id == nil {
		return ""
	}
	return *id
}

// strPtr wraps a string into a pointer, or returns nil when empty.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

type AdminRecurringInvoiceRepositoryInterface interface {
	FindAllRecurringInvoices() ([]entities.RecurringInvoice, error)
	FindRecurringInvoiceByID(request IdRecurringInvoiceRequest) (entities.RecurringInvoice, error)
	CreateRecurringInvoice(request CreateRecurringInvoiceRequest, userID string) (entities.RecurringInvoice, error)
	UpdateRecurringInvoice(request UpdateRecurringInvoiceRequest) (entities.RecurringInvoice, error)
	UpdateRecurringInvoiceStatus(request UpdateRecurringInvoiceStatusRequest) (entities.RecurringInvoice, error)
	DeleteRecurringInvoice(request IdRecurringInvoiceRequest) error
	GenerateInvoiceFromRecurring(request GenerateInvoiceRequest) (entities.Invoice, error)
	GetRecurringInvoiceHistory(request IdRecurringInvoiceRequest) ([]entities.RecurringInvoiceHistory, error)
	ProcessDueRecurringInvoices() (int, error)
}

type AdminRecurringInvoiceRepositoryStruct struct {
	db *gorm.DB
}

func NewAdminRecurringInvoiceRepository(db *gorm.DB) *AdminRecurringInvoiceRepositoryStruct {
	return &AdminRecurringInvoiceRepositoryStruct{db}
}

// clampToMonth moves base by addMonths and returns a date at the same time-of-day
// with the preferred day-of-month, clamped to the last valid day of the target month.
func clampToMonth(base time.Time, addMonths int, preferredDay int) time.Time {
	y, m, _ := base.Date()
	idx := int(m) - 1 + addMonths
	ty := y + idx/12
	tm := time.Month(idx%12 + 1)
	first := time.Date(ty, tm, 1, base.Hour(), base.Minute(), base.Second(), base.Nanosecond(), base.Location())
	last := first.AddDate(0, 1, -1).Day()
	d := preferredDay
	if d > last {
		d = last
	}
	return time.Date(ty, tm, d, base.Hour(), base.Minute(), base.Second(), base.Nanosecond(), base.Location())
}

func (r AdminRecurringInvoiceRepositoryStruct) FindAllRecurringInvoices() ([]entities.RecurringInvoice, error) {
	var recurringInvoices []entities.RecurringInvoice

	tx := r.db.Preload("Customer").
		Preload("CreatedByUser").
		Preload("History.GeneratedInvoice").
		Order("created_at DESC").
		Find(&recurringInvoices)

	if tx.Error != nil {
		return recurringInvoices, tx.Error
	}

	// Parse JSON invoice items for each recurring invoice
	for i := range recurringInvoices {
		if recurringInvoices[i].InvoiceItems != "" {
			var items []entities.RecurringInvoiceItem
			if err := json.Unmarshal([]byte(recurringInvoices[i].InvoiceItems), &items); err == nil {
				recurringInvoices[i].InvoiceItemsData = items
			}
		}
	}

	return recurringInvoices, nil
}

func (r AdminRecurringInvoiceRepositoryStruct) FindRecurringInvoiceByID(request IdRecurringInvoiceRequest) (entities.RecurringInvoice, error) {
	var recurringInvoice entities.RecurringInvoice

	tx := r.db.Preload("Customer").
		Preload("CreatedByUser").
		Preload("History.GeneratedInvoice").
		Where("id = ?", request.Id).
		First(&recurringInvoice)

	if tx.Error != nil {
		return recurringInvoice, tx.Error
	}

	// Parse JSON invoice items
	if recurringInvoice.InvoiceItems != "" {
		var items []entities.RecurringInvoiceItem
		if err := json.Unmarshal([]byte(recurringInvoice.InvoiceItems), &items); err == nil {
			recurringInvoice.InvoiceItemsData = items
		}
	}

	return recurringInvoice, nil
}

func (r AdminRecurringInvoiceRepositoryStruct) CreateRecurringInvoice(request CreateRecurringInvoiceRequest, userID string) (entities.RecurringInvoice, error) {
	// Next invoice should be the same as provided due_date
	nextInvoiceDate := request.DueDate

	// Marshal invoice items to JSON
	itemsJSON, err := json.Marshal(request.InvoiceItems)
	if err != nil {
		return entities.RecurringInvoice{}, fmt.Errorf("failed to marshal invoice items: %v", err)
	}

	// Determine original day from the provided invoice_date (EOM-friendly)
	originalDay := request.InvoiceDate.Day()
	if originalDay >= 30 {
		originalDay = 31
	}

	recurringInvoice := entities.RecurringInvoice{
		CustomerID:           request.CustomerID,
		CustomerInstallation: request.CustomerInstallation,
		Amount:               request.Amount,
		InvoiceDate:          request.InvoiceDate,
		DueDate:              request.DueDate,
		NextInvoiceDate:      nextInvoiceDate,
		Frequency:            entities.RecurringInvoiceFrequency(request.Frequency),
		Status:               entities.RecurringInvoiceStatusActive,
		Description:          request.Description,
		InvoiceItems:         string(itemsJSON),
		CreatedBy:            &userID,
		OriginalDay:          originalDay,
	}

	tx := r.db.Create(&recurringInvoice)
	if tx.Error != nil {
		return recurringInvoice, tx.Error
	}

	// Parse items for response
	var items []entities.RecurringInvoiceItem
	for _, item := range request.InvoiceItems {
		items = append(items, entities.RecurringInvoiceItem{
			Name:  item.Name,
			Price: item.Price,
			Qty:   item.Qty,
			Total: item.Total,
		})
	}
	recurringInvoice.InvoiceItemsData = items

	return recurringInvoice, nil
}

func (r AdminRecurringInvoiceRepositoryStruct) UpdateRecurringInvoice(request UpdateRecurringInvoiceRequest) (entities.RecurringInvoice, error) {
	var recurringInvoice entities.RecurringInvoice

	// Find existing recurring invoice
	tx := r.db.Where("id = ?", request.Id).First(&recurringInvoice)
	if tx.Error != nil {
		return recurringInvoice, tx.Error
	}

	// Next invoice should mirror the provided due_date
	nextInvoiceDate := request.DueDate

	// Marshal invoice items to JSON
	itemsJSON, err := json.Marshal(request.InvoiceItems)
	if err != nil {
		return entities.RecurringInvoice{}, fmt.Errorf("failed to marshal invoice items: %v", err)
	}

	// Update fields
	recurringInvoice.CustomerID = request.CustomerID
	recurringInvoice.Amount = request.Amount
	recurringInvoice.InvoiceDate = request.InvoiceDate
	recurringInvoice.DueDate = request.DueDate
	recurringInvoice.NextInvoiceDate = nextInvoiceDate
	recurringInvoice.Frequency = entities.RecurringInvoiceFrequency(request.Frequency)
	recurringInvoice.Description = request.Description
	recurringInvoice.InvoiceItems = string(itemsJSON)
	// Keep original template day in sync with new invoice_date
	od := request.InvoiceDate.Day()
	if od >= 30 {
		od = 31
	}
	recurringInvoice.OriginalDay = od

	tx = r.db.Save(&recurringInvoice)
	if tx.Error != nil {
		return recurringInvoice, tx.Error
	}

	// Parse items for response
	var items []entities.RecurringInvoiceItem
	for _, item := range request.InvoiceItems {
		items = append(items, entities.RecurringInvoiceItem{
			Name:  item.Name,
			Price: item.Price,
			Qty:   item.Qty,
			Total: item.Total,
		})
	}
	recurringInvoice.InvoiceItemsData = items

	return recurringInvoice, nil
}

func (r AdminRecurringInvoiceRepositoryStruct) UpdateRecurringInvoiceStatus(request UpdateRecurringInvoiceStatusRequest) (entities.RecurringInvoice, error) {
	var recurringInvoice entities.RecurringInvoice

	tx := r.db.Where("id = ?", request.Id).First(&recurringInvoice)
	if tx.Error != nil {
		return recurringInvoice, tx.Error
	}

	recurringInvoice.Status = entities.RecurringInvoiceStatus(request.Status)

	tx = r.db.Save(&recurringInvoice)
	if tx.Error != nil {
		return recurringInvoice, tx.Error
	}

	// Parse items for response
	if recurringInvoice.InvoiceItems != "" {
		var items []entities.RecurringInvoiceItem
		if err := json.Unmarshal([]byte(recurringInvoice.InvoiceItems), &items); err == nil {
			recurringInvoice.InvoiceItemsData = items
		}
	}

	return recurringInvoice, nil
}

func (r AdminRecurringInvoiceRepositoryStruct) DeleteRecurringInvoice(request IdRecurringInvoiceRequest) error {
	tx := r.db.Where("id = ?", request.Id).Delete(&entities.RecurringInvoice{})
	return tx.Error
}

func (r AdminRecurringInvoiceRepositoryStruct) GenerateInvoiceFromRecurring(request GenerateInvoiceRequest) (entities.Invoice, error) {
	var out entities.Invoice
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// lock the recurring record to avoid double generation
		var recurringInvoice entities.RecurringInvoice
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("Customer").First(&recurringInvoice, "id = ?", request.Id).Error; err != nil {
			return err
		}

		// Parse items
		var items []entities.RecurringInvoiceItem
		if err := json.Unmarshal([]byte(recurringInvoice.InvoiceItems), &items); err != nil {
			return fmt.Errorf("failed to parse invoice items: %v", err)
		}

		// Compute dates with end-of-month clamping
		invoiceDate := request.InvoiceDate
		dueDate := request.DueDate
		log.Printf("[recurring-gen] request id=%s invoice_date=%v due_date=%v", request.Id, invoiceDate, dueDate)
		if invoiceDate == nil {
			invoiceDate = &recurringInvoice.NextInvoiceDate
		}

		// Helper removed: use clampToMonth directly

		if dueDate == nil {
			// Due date should be next period relative to invoice_date
			// (monthly/quarterly/yearly) and clamped to the last valid day
			// using the preferred day from the template's original due date.
			monthsToAdd := 1
			switch recurringInvoice.Frequency {
			case entities.RecurringInvoiceFrequencyQuarterly:
				monthsToAdd = 3
			case entities.RecurringInvoiceFrequencyYearly:
				monthsToAdd = 12
			default:
				monthsToAdd = 1
			}
			// Prefer template original day when available; fallback to invoice day.
			preferredDay := recurringInvoice.OriginalDay
			if preferredDay <= 0 {
				preferredDay = invoiceDate.Day()
			}
			if preferredDay >= 30 {
				preferredDay = 31
			}
			dv := clampToMonth(*invoiceDate, monthsToAdd, preferredDay)
			dueDate = &dv
		}
		//log.Printf("[recurring-gen] computed dates id=%s invoice_date=%s due_date=%s (freq=%s preferredDay=%d)", request.Id, invoiceDate.Format("2006-01-02"), dueDate.Format("2006-01-02"), string(recurringInvoice.Frequency), recurringInvoice.DueDate.Day())

		// Consolidate: collapse all unpaid months into a single invoice for the customer
		var inv entities.Invoice
		if err := tx.Where("customer_id = ? AND status IN (?)",
			recurringInvoice.CustomerID, []entities.InvoiceStatus{entities.InvoiceStatusUnpaid, entities.InvoiceStatusPending},
		).Order("invoice_date ASC, createdAt ASC").First(&inv).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// No existing unpaid/pending invoice found - this is normal, create new one
				log.Printf("[recurring-gen] no existing unpaid invoice found for customer=%s, creating new invoice", recurringInvoice.CustomerID)
				inv = entities.Invoice{
					CustomerID:  strPtr(recurringInvoice.CustomerID),
					Amount:      0,
					Link:        fmt.Sprintf("/invoice/%s", recurringInvoice.ID),
					Status:      entities.InvoiceStatusUnpaid,
					InvoiceDate: invoiceDate,
					DueDate:     dueDate,
				}
				if err := tx.Create(&inv).Error; err != nil {
					return err
				}
				// Ensure dates are persisted even if the ORM omitted pointer fields
				if err := tx.Model(&inv).Updates(map[string]interface{}{
					"invoice_date": *invoiceDate,
					"due_date":     *dueDate,
				}).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			// Merge dates: keep earliest invoice_date, latest due_date
			if inv.InvoiceDate != nil && invoiceDate.Before(*inv.InvoiceDate) {
				if err := tx.Model(&inv).Update("invoice_date", *invoiceDate).Error; err != nil {
					return err
				}
			}
			if inv.DueDate != nil && dueDate.After(*inv.DueDate) {
				if err := tx.Model(&inv).Update("due_date", *dueDate).Error; err != nil {
					return err
				}
			}
		}

		// Items
		var addedTotal int64 = 0
		for _, it := range items {
			name := it.Name
			// Product information is now managed through network_devices
			// Use the item name as provided, or use a default if empty
			if name == "" {
				name = "Internet Service"
			}
			qty := it.Qty
			if qty <= 0 {
				qty = 1
			}
			price := it.Price
			total := it.Total
			if total <= 0 {
				total = price * qty
			}
			ii := entities.InvoiceItems{InvoiceID: inv.ID, Name: name, Price: price, Qty: qty, Total: total}
			// Ensure primary key is set (ID is varchar PK, no BeforeCreate hook)
			ii.ID = uuid.New().String()
			if err := tx.Create(&ii).Error; err != nil {
				return err
			}
			addedTotal += total
		}

		// Update invoice amount by added items
		if addedTotal > 0 {
			if err := tx.Model(&inv).Update("amount", gorm.Expr("amount + ?", addedTotal)).Error; err != nil {
				return err
			}
		}

		// History
		h := entities.RecurringInvoiceHistory{RecurringInvoiceID: recurringInvoice.ID, GeneratedInvoiceID: inv.ID, InvoiceDate: *invoiceDate, DueDate: *dueDate}
		if err := tx.Create(&h).Error; err != nil {
			return err
		}

		// Update recurring dates (force), with fallback when RowsAffected == 0
		// Business rule: next_invoice_date should equal computed dueDate
		newNext := *dueDate
		res := tx.Model(&entities.RecurringInvoice{}).Where("id = ?", recurringInvoice.ID).
			Updates(map[string]interface{}{
				"invoice_date":      *invoiceDate,
				"due_date":          *dueDate,
				"next_invoice_date": newNext,
				"updated_at":        time.Now(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			if err := tx.Exec(
				"UPDATE recurring_invoices SET invoice_date = ?, due_date = ?, next_invoice_date = ?, updated_at = ? WHERE id = ?",
				*invoiceDate, *dueDate, newNext, time.Now(), recurringInvoice.ID,
			).Error; err != nil {
				return err
			}
		}

		out = inv
		return nil
	})
	if err != nil {
		return entities.Invoice{}, err
	}
	// reload invoice with relations for response (explicit condition to avoid malformed WHERE)
	r.db.Preload("Customer").Preload("InvoiceItems").First(&out, "id = ?", out.ID)

	// Enqueue router job for unpaid scheduler after generation (best-effort)
	if out.Status == entities.InvoiceStatusUnpaid {
		_, _ = services.EnqueueRouterJob(r.db, out.ID, services.RouterActionSetUnpaidScheduler, 0)
	}
	return out, nil
}

func (r AdminRecurringInvoiceRepositoryStruct) GetRecurringInvoiceHistory(request IdRecurringInvoiceRequest) ([]entities.RecurringInvoiceHistory, error) {
	var history []entities.RecurringInvoiceHistory

	tx := r.db.Preload("GeneratedInvoice").
		Where("recurring_invoice_id = ?", request.Id).
		Order("generated_at DESC").
		Find(&history)

	return history, tx.Error
}

// Helper function to calculate next invoice date
func calculateNextInvoiceDate(currentDate time.Time, frequency string) time.Time {
	months := 1
	switch frequency {
	case "monthly":
		months = 1
	case "quarterly":
		months = 3
	case "yearly":
		months = 12
	default:
		months = 1
	}

	// Determine preferred day: keep the same day-of-month if possible.
	preferred := currentDate.Day()

	// For end-of-month behavior: if the source day is near the month end (>=30),
	// request day 31 so clampToMonth yields the last valid day of the target month
	// (31 → 31/30/29/28; 30 → 30/29/28; others keep same day).
	if preferred >= 30 {
		preferred = 31
	}
	return clampToMonth(currentDate, months, preferred)
}

// ProcessDueRecurringInvoices finds all active recurring invoices with next_invoice_date <= now
// and generates invoices for them. Returns the number of invoices generated.
func (r AdminRecurringInvoiceRepositoryStruct) ProcessDueRecurringInvoices() (int, error) {
	var due []entities.RecurringInvoice
	tx := r.db.Where("status = ? AND next_invoice_date <= ?", entities.RecurringInvoiceStatusActive, time.Now()).Find(&due)
	if tx.Error != nil {
		return 0, tx.Error
	}

	generated := 0
	for _, rec := range due {
		_, err := r.GenerateInvoiceFromRecurring(GenerateInvoiceRequest{IdRecurringInvoiceRequest: IdRecurringInvoiceRequest{Id: rec.ID}})
		if err == nil {
			generated++
		}
	}
	return generated, nil
}
