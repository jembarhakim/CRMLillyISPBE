package payments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"skripsi-be/internal/models/entities"
)

func TestPendingVAIsActiveOnlyBeforeExpiry(t *testing.T) {
	now := time.Now()
	attempt := entities.PaymentAttempt{
		Status:     "pending",
		ExpiryTime: now.Add(time.Hour),
	}
	if !pendingVAIsActive(attempt, now) {
		t.Fatal("unexpired pending VA should be active")
	}

	attempt.ExpiryTime = now
	if pendingVAIsActive(attempt, now) {
		t.Fatal("VA expiring now should not be active")
	}

	attempt.Status = "expired"
	attempt.ExpiryTime = now.Add(time.Hour)
	if pendingVAIsActive(attempt, now) {
		t.Fatal("non-pending VA should not be active")
	}
}

func TestMidtransOrderNotFoundIsRecognized(t *testing.T) {
	if !isMidtransOrderNotFound(&ProviderError{StatusCode: http.StatusNotFound, Body: `{"status_code":"404"}`}) {
		t.Fatal("Midtrans 404 should identify a stale order")
	}
	if isMidtransOrderNotFound(&ProviderError{StatusCode: http.StatusUnauthorized, Body: "unauthorized"}) {
		t.Fatal("unauthorized response must not be treated as a stale order")
	}
}

func TestProviderAPIErrorUsesFriendlyStatuses(t *testing.T) {
	if err := providerAPIError(&ProviderError{StatusCode: http.StatusUnauthorized, Body: "unauthorized"}, "fallback"); err.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized should return 401, got %d", err.StatusCode)
	}
	if err := providerAPIError(&ProviderError{StatusCode: http.StatusPaymentRequired, Body: "channel disabled"}, "fallback"); err.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("402 should be preserved, got %d", err.StatusCode)
	}
	if err := providerAPIError(errMidtransKeyMissing, "fallback"); err.StatusCode != http.StatusUnauthorized {
		t.Fatalf("missing server key should return 401, got %d", err.StatusCode)
	}
}

func TestMidtransChargeUsesCoreAPIPayload(t *testing.T) {
	serverKey := "test-server-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v2/charge" {
			t.Errorf("request = %s %s, want POST /v2/charge", r.Method, r.URL.Path)
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != serverKey || password != "" {
			t.Errorf("unexpected Basic Auth credentials")
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		details := payload["transaction_details"].(map[string]any)
		bank := payload["bank_transfer"].(map[string]any)
		expiry := payload["custom_expiry"].(map[string]any)
		if payload["payment_type"] != "bank_transfer" ||
			details["order_id"] != "va-test-order" ||
			details["gross_amount"] != float64(275000) ||
			bank["bank"] != "bri" ||
			expiry["expiry_duration"] != float64(24) ||
			expiry["unit"] != "hour" {
			t.Errorf("unexpected Core API payload: %#v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status_code":"201","expiry_time":"2026-11-13 12:00:00","va_numbers":[{"va_number":"1234567890"}]}`))
	}))
	defer server.Close()

	gateway := &MidtransGateway{
		serverKey: serverKey,
		baseURL:   server.URL,
		client:    server.Client(),
	}
	result, err := gateway.Charge(context.Background(), ChargeRequest{
		OrderID: "va-test-order", Amount: 275000, Bank: "bri", FirstName: "Siti",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != "201" || result.VANumber != "1234567890" {
		t.Fatalf("unexpected charge result: %+v", result)
	}
}
