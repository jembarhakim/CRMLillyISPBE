package field_payment

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	admininvoice "skripsi-be/internal/api/admin/invoice"
	"skripsi-be/internal/helpers"
	"skripsi-be/internal/models/entities"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FieldPaymentHandler struct {
	db *gorm.DB
}

func NewFieldPaymentHandler(db *gorm.DB) *FieldPaymentHandler {
	return &FieldPaymentHandler{db: db}
}

type invoiceOption struct {
	ID            string `json:"id"`
	Amount        int64  `json:"amount"`
	PaidAmount    int64  `json:"paid_amount"`
	Outstanding   int64  `json:"outstanding"`
	Status        string `json:"status"`
	CustomerName  string `json:"customer_name"`
	CustomerPhone string `json:"customer_phone"`
}

type fieldPaymentResponse struct {
	ID            string    `json:"id"`
	InvoiceID     string    `json:"invoice_id"`
	InvoiceAmount int64     `json:"invoice_amount"`
	CustomerName  string    `json:"customer_name"`
	Amount        int64     `json:"amount"`
	ReceiptPath   string    `json:"receipt_path"`
	CollectorID   string    `json:"collector_id"`
	CollectorName string    `json:"collector_name"`
	CreatedAt     time.Time `json:"created_at"`
}

type closePaymentsRequest struct {
	CollectorID string   `json:"collector_id" validate:"required"`
	AccountID   string   `json:"account_id" validate:"required"`
	PaymentIDs  []string `json:"payment_ids" validate:"required,min=1"`
}

func calculatePaid(db *gorm.DB, invoiceID string) (int64, error) {
	var transactionTotal, fieldPaymentTotal int64
	if err := db.Model(&entities.Transaction{}).
		Where("invoice_id = ?", invoiceID).
		Select("COALESCE(SUM(amount), 0)").Scan(&transactionTotal).Error; err != nil {
		return 0, err
	}
	if err := db.Model(&entities.InvoicePayment{}).
		Where("invoice_id = ? AND payment_type = ? AND closing_id IS NULL", invoiceID, "field_cash").
		Select("COALESCE(SUM(amount), 0)").Scan(&fieldPaymentTotal).Error; err != nil {
		return 0, err
	}
	return transactionTotal + fieldPaymentTotal, nil
}

func (h *FieldPaymentHandler) GetOpenInvoices(c *fiber.Ctx) error {
	query := strings.TrimSpace(c.Query("query"))
	if query == "" {
		return helpers.ResponseUtils(c, http.StatusOK, true, "Search by exact invoice ID", []invoiceOption{})
	}
	var invoices []entities.Invoice
	dbQuery := h.db.Preload("Customer").
		Where("status IN ?", []entities.InvoiceStatus{
			entities.InvoiceStatusUnpaid,
			entities.InvoiceStatusPending,
			entities.InvoiceStatusPartiallyPaid,
		}).
		Order("createdAt DESC").
		Limit(1).
		Where("id = ?", query)
	if err := dbQuery.Find(&invoices).Error; err != nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to load open invoices", nil)
	}

	result := make([]invoiceOption, 0, len(invoices))
	for _, invoice := range invoices {
		paid, err := calculatePaid(h.db, invoice.ID)
		if err != nil {
			return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to calculate invoice balance", nil)
		}
		outstanding := invoice.Amount - paid
		if outstanding <= 0 {
			continue
		}
		result = append(result, invoiceOption{
			ID:            invoice.ID,
			Amount:        invoice.Amount,
			PaidAmount:    paid,
			Outstanding:   outstanding,
			Status:        string(invoice.Status),
			CustomerName:  invoice.Customer.Name,
			CustomerPhone: invoice.Customer.Phone,
		})
	}
	return helpers.ResponseUtils(c, http.StatusOK, true, "Open invoices loaded", result)
}

func (h *FieldPaymentHandler) CreateFieldPayment(c *fiber.Ctx) error {
	var request struct {
		InvoiceID   string `json:"invoice_id" validate:"required"`
		Amount      int64  `json:"amount" validate:"required,min=1"`
		ReceiptPath string `json:"receipt_path" validate:"required"`
	}
	if err := c.BodyParser(&request); err != nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Invalid request body", nil)
	}
	if request.InvoiceID == "" || request.Amount < 1 || strings.TrimSpace(request.ReceiptPath) == "" {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Invoice, positive amount, and receipt are required", nil)
	}
	if !strings.HasPrefix(request.ReceiptPath, "cs-images/payment-receipts/") || strings.Contains(request.ReceiptPath, "..") {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "A valid payment receipt path is required", nil)
	}

	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return helpers.ResponseUtils(c, http.StatusUnauthorized, false, "User session is missing", nil)
	}

	tx := h.db.Begin()
	if tx.Error != nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Unable to start payment transaction", nil)
	}
	var invoice entities.Invoice
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&invoice, "id = ?", request.InvoiceID).Error; err != nil {
		tx.Rollback()
		return helpers.ResponseUtils(c, http.StatusNotFound, false, "Invoice not found", nil)
	}
	if invoice.Status == entities.InvoiceStatusPaid {
		tx.Rollback()
		return helpers.ResponseUtils(c, http.StatusConflict, false, "Invoice is already paid", nil)
	}
	paid, err := calculatePaid(tx, invoice.ID)
	if err != nil {
		tx.Rollback()
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to calculate invoice balance", nil)
	}
	if request.Amount != invoice.Amount-paid {
		tx.Rollback()
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, fmt.Sprintf("Cash collection must cover the outstanding balance of %d", invoice.Amount-paid), nil)
	}

	payment := entities.InvoicePayment{
		InvoiceID:   invoice.ID,
		Amount:      request.Amount,
		PaymentType: "field_cash",
		Method:      "cash",
		ReceiptPath: strings.TrimSpace(request.ReceiptPath),
		CollectedBy: userID,
	}
	if err := tx.Create(&payment).Error; err != nil {
		tx.Rollback()
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to record field payment", nil)
	}
	if err := tx.Model(&invoice).Update("status", entities.InvoiceStatusPaid).Error; err != nil {
		tx.Rollback()
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to update invoice status", nil)
	}
	if err := tx.Commit().Error; err != nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to save field payment", nil)
	}
	invoice.Status = entities.InvoiceStatusPaid
	admininvoice.EnqueuePaidInvoiceActions(h.db, invoice)
	return helpers.ResponseUtils(c, http.StatusCreated, true, "Cash payment recorded", payment)
}

func (h *FieldPaymentHandler) GetPendingPayments(c *fiber.Ctx) error {
	var payments []entities.InvoicePayment
	if err := h.db.Preload("Invoice").Preload("Invoice.Customer").
		Where("payment_type = ? AND closing_id IS NULL", "field_cash").
		Order("created_at ASC").Find(&payments).Error; err != nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to load pending cash payments", nil)
	}

	result := make([]fieldPaymentResponse, 0, len(payments))
	for _, payment := range payments {
		var collector entities.User
		if err := h.db.Select("id", "name").First(&collector, "id = ?", payment.CollectedBy).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to load collector information", nil)
		}
		item := fieldPaymentResponse{
			ID:            payment.ID,
			InvoiceID:     payment.InvoiceID,
			Amount:        payment.Amount,
			ReceiptPath:   payment.ReceiptPath,
			CollectorID:   payment.CollectedBy,
			CollectorName: collector.Name,
			CreatedAt:     payment.CreatedAt,
		}
		if payment.Invoice != nil {
			item.InvoiceAmount = payment.Invoice.Amount
			item.CustomerName = payment.Invoice.Customer.Name
		}
		result = append(result, item)
	}
	return helpers.ResponseUtils(c, http.StatusOK, true, "Pending cash payments loaded", result)
}

func (h *FieldPaymentHandler) CloseCollectorPayments(c *fiber.Ctx) error {
	var request closePaymentsRequest
	if err := c.BodyParser(&request); err != nil {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Invalid request body", nil)
	}
	if request.CollectorID == "" || request.AccountID == "" || len(request.PaymentIDs) == 0 {
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Collector, account, and payments are required", nil)
	}
	verifierID, ok := c.Locals("user_id").(string)
	if !ok || verifierID == "" {
		return helpers.ResponseUtils(c, http.StatusUnauthorized, false, "User session is missing", nil)
	}

	tx := h.db.Begin()
	if tx.Error != nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Unable to start closing transaction", nil)
	}
	var payments []entities.InvoicePayment
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ? AND collected_by = ? AND payment_type = ? AND closing_id IS NULL", request.PaymentIDs, request.CollectorID, "field_cash").
		Find(&payments).Error; err != nil {
		tx.Rollback()
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to load selected payments", nil)
	}
	if len(payments) != len(request.PaymentIDs) {
		tx.Rollback()
		return helpers.ResponseUtils(c, http.StatusConflict, false, "Some selected payments are already closed or do not belong to this collector", nil)
	}

	var total int64
	for _, payment := range payments {
		total += payment.Amount
	}
	var account entities.Accounts
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account, "id = ?", request.AccountID).Error; err != nil {
		tx.Rollback()
		return helpers.ResponseUtils(c, http.StatusBadRequest, false, "Finance account not found", nil)
	}
	closing := entities.FieldPaymentClosing{
		CollectorID: request.CollectorID,
		AccountID:   request.AccountID,
		VerifiedBy:  verifierID,
		Total:       total,
	}
	if err := tx.Create(&closing).Error; err != nil {
		tx.Rollback()
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to create closing record", nil)
	}

	for _, payment := range payments {
		transaction := entities.Transaction{
			AccountID:   request.AccountID,
			InvoiceID:   payment.InvoiceID,
			TypeCash:    entities.TransactionsTypeCashInternet,
			TypeInOut:   entities.TransactionsTypeInOutIn,
			Description: fmt.Sprintf("Field cash payment closing %s", closing.ID),
			Amount:      payment.Amount,
			Category:    "field_payment",
			Method:      "cash",
		}
		if err := tx.Create(&transaction).Error; err != nil {
			tx.Rollback()
			return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to record cash in account ledger", nil)
		}
		transactionID := transaction.ID
		if err := tx.Model(&payment).Updates(map[string]interface{}{
			"closing_id":     closing.ID,
			"transaction_id": transactionID,
		}).Error; err != nil {
			tx.Rollback()
			return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to mark payment as closed", nil)
		}
	}

	if err := tx.Model(&account).UpdateColumn("saldo", gorm.Expr("saldo + ?", total)).Error; err != nil {
		tx.Rollback()
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to update account balance", nil)
	}
	if err := tx.Commit().Error; err != nil {
		return helpers.ResponseUtils(c, http.StatusInternalServerError, false, "Failed to complete cash closing", nil)
	}
	return helpers.ResponseUtils(c, http.StatusOK, true, "Collector cash closing completed", closing)
}
