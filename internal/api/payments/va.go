package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"skripsi-be/internal/helpers"
	"skripsi-be/internal/models/entities"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errMidtransKeyMissing = errors.New("MIDTRANS_SERVER_KEY is not configured")

type ProviderError struct {
	StatusCode int
	Body       string
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("Midtrans returned HTTP %d: %s", e.StatusCode, e.Body)
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string { return e.Message }

var vaRateLimits = struct {
	sync.Mutex
	requests    map[string][]time.Time
	lastCleanup time.Time
}{requests: make(map[string][]time.Time)}

type VAResult struct {
	InvoiceID  string    `json:"invoice_id"`
	OrderID    string    `json:"order_id"`
	Bank       string    `json:"bank"`
	VANumber   string    `json:"va_number"`
	Amount     int64     `json:"amount"`
	ExpiryTime time.Time `json:"expiry_time"`
	Status     string    `json:"status"`
}

type ChargeRequest struct {
	OrderID   string
	Amount    int64
	Bank      string
	FirstName string
}

type ChargeResult struct {
	StatusCode string
	VANumber   string
	ExpiryTime string
}

type TransactionStatus struct {
	TransactionStatus string `json:"transaction_status"`
}

type Gateway interface {
	Charge(context.Context, ChargeRequest) (ChargeResult, error)
	Expire(context.Context, string) error
	Status(context.Context, string) (TransactionStatus, error)
}

type MidtransGateway struct {
	serverKey string
	baseURL   string
	client    *http.Client
}

func NewMidtransGatewayFromEnv() *MidtransGateway {
	baseURL := "https://api.sandbox.midtrans.com"
	if strings.EqualFold(os.Getenv("MIDTRANS_IS_PRODUCTION"), "true") {
		baseURL = "https://api.midtrans.com"
	}
	return &MidtransGateway{
		serverKey: strings.TrimSpace(os.Getenv("MIDTRANS_SERVER_KEY")),
		baseURL:   baseURL,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (g *MidtransGateway) Charge(ctx context.Context, charge ChargeRequest) (ChargeResult, error) {
	if g.serverKey == "" {
		return ChargeResult{}, errMidtransKeyMissing
	}

	payload := map[string]any{
		"payment_type": "bank_transfer",
		"transaction_details": map[string]any{
			"order_id":     charge.OrderID,
			"gross_amount": charge.Amount,
		},
		"bank_transfer":    map[string]string{"bank": charge.Bank},
		"customer_details": map[string]string{"first_name": charge.FirstName},
		"custom_expiry": map[string]any{
			"expiry_duration": 24,
			"unit":            "hour",
		},
	}
	var response struct {
		StatusCode string `json:"status_code"`
		ExpiryTime string `json:"expiry_time"`
		VANumbers  []struct {
			VANumber string `json:"va_number"`
		} `json:"va_numbers"`
	}
	if err := g.request(ctx, http.MethodPost, "/v2/charge", payload, &response); err != nil {
		return ChargeResult{}, err
	}
	result := ChargeResult{StatusCode: response.StatusCode, ExpiryTime: response.ExpiryTime}
	if len(response.VANumbers) > 0 {
		result.VANumber = response.VANumbers[0].VANumber
	}
	return result, nil
}

func (g *MidtransGateway) Expire(ctx context.Context, orderID string) error {
	if g.serverKey == "" {
		return errMidtransKeyMissing
	}
	return g.request(ctx, http.MethodPost, "/v2/"+url.PathEscape(orderID)+"/expire", nil, nil)
}

func (g *MidtransGateway) Status(ctx context.Context, orderID string) (TransactionStatus, error) {
	if g.serverKey == "" {
		return TransactionStatus{}, errMidtransKeyMissing
	}
	var response TransactionStatus
	err := g.request(ctx, http.MethodGet, "/v2/"+url.PathEscape(orderID)+"/status", nil, &response)
	return response, err
}

func (g *MidtransGateway) request(ctx context.Context, method, path string, payload any, output any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, g.baseURL+path, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(g.serverKey, "")
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	response, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	if err != nil {
		return err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return &ProviderError{StatusCode: response.StatusCode, Body: strings.TrimSpace(string(responseBody))}
	}
	if output != nil {
		if err := json.Unmarshal(responseBody, output); err != nil {
			return fmt.Errorf("decode Midtrans response: %w", err)
		}
	}
	return nil
}

type Service struct {
	db      *gorm.DB
	gateway Gateway
}

func NewService(db *gorm.DB, gateway Gateway) *Service {
	return &Service{db: db, gateway: gateway}
}

func (s *Service) CreateVA(ctx context.Context, customerID, invoiceID, bank string) (VAResult, error) {
	bank = strings.ToLower(strings.TrimSpace(bank))
	if bank != "bca" && bank != "bni" && bank != "bri" {
		return VAResult{}, &APIError{StatusCode: fiber.StatusBadRequest, Message: "Pilihan bank tidak valid. Pilih BCA, BNI, atau BRI."}
	}
	if invoiceID == "" {
		return VAResult{}, &APIError{StatusCode: fiber.StatusBadRequest, Message: "Tagihan harus dipilih."}
	}

	var result VAResult
	var operationErr error
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var invoice entities.Invoice
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND customer_id = ?", invoiceID, customerID).
			First(&invoice).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &APIError{StatusCode: fiber.StatusNotFound, Message: "Tagihan tidak ditemukan atau bukan milik akun Anda."}
		}
		if err != nil {
			return err
		}
		if invoice.Status == entities.InvoiceStatusPaid {
			return &APIError{StatusCode: fiber.StatusConflict, Message: "Tagihan ini sudah lunas."}
		}
		if invoice.Amount <= 0 {
			return &APIError{StatusCode: fiber.StatusBadRequest, Message: "Nominal tagihan tidak valid. Silakan hubungi admin."}
		}

		now := time.Now()
		var attempts []entities.PaymentAttempt
		if err := tx.Where("invoice_id = ? AND status = ?", invoice.ID, "pending").
			Order("created_at DESC").Find(&attempts).Error; err != nil {
			return err
		}

		var customer entities.Customer
		if err := tx.Select("name").Where("id = ?", customerID).First(&customer).Error; err != nil {
			return err
		}
		firstName := strings.Fields(strings.TrimSpace(customer.Name))
		customerName := "Pelanggan"
		if len(firstName) > 0 {
			customerName = firstName[0]
		}

		for _, attempt := range attempts {
			if pendingVAIsActive(attempt, now) {
				status, statusErr := s.gateway.Status(ctx, attempt.OrderID)
				if statusErr != nil {
					if !isMidtransOrderNotFound(statusErr) {
						log.Printf("Midtrans status request failed for existing order %s: %v", attempt.OrderID, statusErr)
						operationErr = providerAPIError(statusErr, "Status VA lama tidak dapat diperiksa. Silakan coba lagi.")
						return nil
					}
					if err := tx.Model(&attempt).Update("status", "expired").Error; err != nil {
						return err
					}
					continue
				}

				switch strings.ToLower(status.TransactionStatus) {
				case "pending":
					result = resultFromAttempt(attempt)
					return nil
				case "settlement", "capture":
					if err := tx.Model(&attempt).Update("status", strings.ToLower(status.TransactionStatus)).Error; err != nil {
						return err
					}
					operationErr = &APIError{
						StatusCode: fiber.StatusConflict,
						Message:    "Pembayaran ini terdeteksi berhasil. Silakan muat ulang status tagihan.",
					}
					return nil
				case "expire", "cancel", "deny", "failure":
					if err := tx.Model(&attempt).Update("status", strings.ToLower(status.TransactionStatus)).Error; err != nil {
						return err
					}
					continue
				default:
					result = resultFromAttempt(attempt)
					return nil
				}
			}
			if err := tx.Model(&attempt).Update("status", "expired").Error; err != nil {
				return err
			}
		}

		orderID := "va-" + uuid.NewString()
		charge, err := s.gateway.Charge(ctx, ChargeRequest{
			OrderID: orderID, Amount: invoice.Amount, Bank: bank, FirstName: customerName,
		})
		if err != nil {
			log.Printf("Midtrans VA charge failed for invoice %s: %v", invoice.ID, err)
			operationErr = providerAPIError(err, "VA belum dapat dibuat. Silakan coba lagi.")
			return nil
		}
		if charge.StatusCode != "201" || charge.VANumber == "" {
			log.Printf("Midtrans VA charge returned unexpected response for invoice %s (status %q)", invoice.ID, charge.StatusCode)
			operationErr = &APIError{StatusCode: fiber.StatusBadGateway, Message: "VA belum dapat dibuat. Silakan coba lagi."}
			return nil
		}

		expiryTime := parseExpiryTime(charge.ExpiryTime, now.Add(24*time.Hour))
		attempt := entities.PaymentAttempt{
			InvoiceID:  invoice.ID,
			OrderID:    orderID,
			Bank:       bank,
			VANumber:   charge.VANumber,
			Amount:     invoice.Amount,
			ExpiryTime: expiryTime,
			Status:     "pending",
		}
		if err := tx.Create(&attempt).Error; err != nil {
			log.Printf("Failed to store Midtrans VA for invoice %s: %v", invoice.ID, err)
			operationErr = &APIError{StatusCode: fiber.StatusInternalServerError, Message: "VA berhasil dibuat tetapi belum dapat disimpan. Silakan hubungi admin."}
			return nil
		}
		result = resultFromAttempt(attempt)
		return nil
	})
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			return VAResult{}, apiErr
		}
		log.Printf("Failed to create VA for invoice %s: %v", invoiceID, err)
		return VAResult{}, &APIError{StatusCode: fiber.StatusInternalServerError, Message: "Terjadi kesalahan saat memproses tagihan."}
	}
	if operationErr != nil {
		return VAResult{}, operationErr
	}
	return result, nil
}

func pendingVAIsActive(attempt entities.PaymentAttempt, now time.Time) bool {
	return attempt.Status == "pending" && attempt.ExpiryTime.After(now)
}

func isMidtransOrderNotFound(err error) bool {
	var providerErr *ProviderError
	return errors.As(err, &providerErr) && providerErr.StatusCode == http.StatusNotFound
}

func (s *Service) CheckStatus(ctx context.Context, customerID, orderID string) (TransactionStatus, error) {
	var attempt entities.PaymentAttempt
	err := s.db.WithContext(ctx).Where("order_id = ?", orderID).First(&attempt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return TransactionStatus{}, &APIError{StatusCode: fiber.StatusNotFound, Message: "Pembayaran tidak ditemukan."}
	}
	if err != nil {
		log.Printf("Failed to find VA order %s: %v", orderID, err)
		return TransactionStatus{}, &APIError{StatusCode: fiber.StatusInternalServerError, Message: "Status pembayaran belum dapat diperiksa."}
	}
	var invoice entities.Invoice
	err = s.db.WithContext(ctx).Where("id = ? AND customer_id = ?", attempt.InvoiceID, customerID).First(&invoice).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return TransactionStatus{}, &APIError{StatusCode: fiber.StatusNotFound, Message: "Pembayaran tidak ditemukan."}
	}
	if err != nil {
		log.Printf("Failed to verify ownership for VA order %s: %v", orderID, err)
		return TransactionStatus{}, &APIError{StatusCode: fiber.StatusInternalServerError, Message: "Status pembayaran belum dapat diperiksa."}
	}

	status, err := s.gateway.Status(ctx, orderID)
	if err != nil {
		if isMidtransOrderNotFound(err) {
			return TransactionStatus{TransactionStatus: "expire"}, nil
		}
		log.Printf("Midtrans status request failed for order %s: %v", orderID, err)
		return TransactionStatus{}, providerAPIError(err, "Status pembayaran belum dapat diperiksa.")
	}
	return status, nil
}

func resultFromAttempt(attempt entities.PaymentAttempt) VAResult {
	return VAResult{
		InvoiceID:  attempt.InvoiceID,
		OrderID:    attempt.OrderID,
		Bank:       attempt.Bank,
		VANumber:   attempt.VANumber,
		Amount:     attempt.Amount,
		ExpiryTime: attempt.ExpiryTime,
		Status:     attempt.Status,
	}
}

func parseExpiryTime(value string, fallback time.Time) time.Time {
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return fallback
}

func providerAPIError(err error, fallback string) *APIError {
	if errors.Is(err, errMidtransKeyMissing) {
		return &APIError{StatusCode: http.StatusUnauthorized, Message: "Konfigurasi pembayaran belum tersedia. Silakan hubungi admin."}
	}
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		switch providerErr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return &APIError{StatusCode: http.StatusUnauthorized, Message: "Konfigurasi pembayaran tidak valid. Silakan hubungi admin."}
		case http.StatusPaymentRequired:
			return &APIError{StatusCode: http.StatusPaymentRequired, Message: "Kanal VA bank ini belum aktif. Silakan pilih bank lain."}
		}
	}
	return &APIError{StatusCode: fiber.StatusBadGateway, Message: fallback}
}

type vaRequest struct {
	InvoiceID string `json:"invoiceId"`
	Bank      string `json:"bank"`
}

func RegisterRoutes(router fiber.Router, db *gorm.DB) {
	service := NewService(db, NewMidtransGatewayFromEnv())
	routes := router.Group("", helpers.CustomerVerifyToken)
	routes.Post("/va", limitVARequests, func(c *fiber.Ctx) error {
		var request vaRequest
		if err := c.BodyParser(&request); err != nil {
			return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "Data permintaan tidak valid.", nil)
		}
		customerID, _ := c.Locals("user_id").(string)
		result, err := service.CreateVA(c.UserContext(), customerID, request.InvoiceID, request.Bank)
		if err != nil {
			return respondError(c, err)
		}
		return helpers.ResponseUtils(c, fiber.StatusOK, true, "VA berhasil disiapkan.", result)
	})
	routes.Get("/va/:orderId/status", func(c *fiber.Ctx) error {
		customerID, _ := c.Locals("user_id").(string)
		status, err := service.CheckStatus(c.UserContext(), customerID, c.Params("orderId"))
		if err != nil {
			return respondError(c, err)
		}
		return helpers.ResponseUtils(c, fiber.StatusOK, true, "Status pembayaran.", status)
	})
}

func limitVARequests(c *fiber.Ctx) error {
	customerID := fmt.Sprint(c.Locals("user_id"))
	now := time.Now()
	cutoff := now.Add(-time.Minute)

	vaRateLimits.Lock()
	if now.Sub(vaRateLimits.lastCleanup) >= time.Minute {
		for key, requests := range vaRateLimits.requests {
			active := requests[:0]
			for _, requestTime := range requests {
				if requestTime.After(cutoff) {
					active = append(active, requestTime)
				}
			}
			if len(active) == 0 {
				delete(vaRateLimits.requests, key)
			} else {
				vaRateLimits.requests[key] = active
			}
		}
		vaRateLimits.lastCleanup = now
	}

	requests := vaRateLimits.requests[customerID]
	active := requests[:0]
	for _, requestTime := range requests {
		if requestTime.After(cutoff) {
			active = append(active, requestTime)
		}
	}
	if len(active) >= 5 {
		vaRateLimits.requests[customerID] = active
		vaRateLimits.Unlock()
		return helpers.ResponseUtils(c, fiber.StatusTooManyRequests, false, "Terlalu banyak permintaan. Silakan coba lagi dalam satu menit.", nil)
	}
	vaRateLimits.requests[customerID] = append(active, now)
	vaRateLimits.Unlock()
	return c.Next()
}

func respondError(c *fiber.Ctx, err error) error {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return helpers.ResponseUtils(c, apiErr.StatusCode, false, apiErr.Message, nil)
	}
	return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, "Terjadi kesalahan pada server.", nil)
}
