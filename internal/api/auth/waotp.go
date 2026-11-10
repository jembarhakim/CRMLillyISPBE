package authapi

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// =============================================================================
// WhatsApp OTP login support
//
// Flow:
//  1. POST /api/auth/request-otp  {phone}
//     - normalise phone -> find customer -> generate 6 digit OTP
//     - store hash + expiry in process memory (otpChallenges)
//     - deliver OTP via WhatsApp gateway (whatsapp-service / Fonnte compatible)
//  2. POST /api/auth/verify-otp   {phone, code}
//     - validate OTP (single use, max 5 attempts, 5 minutes TTL)
//     - issue JWT with role "CUSTOMER" so customer token can never reach admin routes
//
// Customers are keyed by phone number because that is the only identity a
// customer is guaranteed to know (see LoginCustomerAuthService).
// =============================================================================

const (
	otpTTL            = 5 * time.Minute
	otpResendCooldown = 60 * time.Second
	otpMaxAttempts    = 5
	otpRequestPerHour = 5 // per phone number, per hour

	otpHashCost = 10
)

// otpEntry is the in-memory state of a single OTP challenge.
type otpEntry struct {
	Hash          string
	ExpiresAt     time.Time
	LastRequestAt time.Time
	Attempts      int
}

// otpStore is a tiny in-process store. A process restart invalidates pending
// OTPs, which is acceptable for a 5 minute challenge. Swap for Redis if the
// backend is ever scaled horizontally.
type otpStore struct {
	mu       sync.Mutex
	entries  map[string]*otpEntry
	requests map[string][]time.Time // phone -> request timestamps (rate limit)
}

var otpChallenges = &otpStore{
	entries:  make(map[string]*otpEntry),
	requests: make(map[string][]time.Time),
}

// NormalizePhone converts a local Indonesian number into the canonical
// digits-only form expected by the WhatsApp gateway (e.g. 0812-3456-7890 ->
// 6281234567890). Returns an empty string when the input cannot be a phone.
func NormalizePhone(raw string) string {
	digits := regexp.MustCompile(`[^0-9]`).ReplaceAllString(raw, "")
	if digits == "" {
		return ""
	}
	// Remove international "00" prefix (0062... -> 62...)
	digits = strings.TrimPrefix(digits, "00")
	switch {
	case strings.HasPrefix(digits, "62"):
		// already international
	case strings.HasPrefix(digits, "0"):
		digits = "62" + strings.TrimPrefix(digits, "0")
	case strings.HasPrefix(digits, "8"):
		digits = "62" + digits
	}
	if len(digits) < 10 || len(digits) > 15 {
		return ""
	}
	return digits
}

// phoneVariants returns the possible stored representations of a phone number
// so repository lookups work regardless of how the customer record was saved.
func phoneVariants(normalized string) []string {
	variants := []string{normalized}
	if strings.HasPrefix(normalized, "62") {
		local := "0" + strings.TrimPrefix(normalized, "62")
		variants = append(variants, local, "+"+normalized)
	}
	return variants
}

func maskPhone(normalized string) string {
	if len(normalized) <= 6 {
		return normalized
	}
	return normalized[:4] + strings.Repeat("*", len(normalized)-6) + normalized[len(normalized)-2:]
}

// ---------------------------------------------------------------------------
// OTP lifecycle
// ---------------------------------------------------------------------------

func generateOTPCode() (string, error) {
	limit := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func hashOTP(code string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(code), otpHashCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// recentRequestCount counts OTP requests for the phone inside the last hour.
// Caller must hold otpChallenges.mu.
func recentRequestCount(phone string, now time.Time) int {
	cutoff := now.Add(-time.Hour)
	kept := make([]time.Time, 0, len(otpChallenges.requests[phone]))
	for _, ts := range otpChallenges.requests[phone] {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	otpChallenges.requests[phone] = kept
	return len(kept)
}

// issueOTP creates (or replaces) the challenge for a phone number.
// Errors are user-facing messages already localised in Bahasa Indonesia.
func issueOTP(phone string, codeHash string) error {
	now := time.Now()

	otpChallenges.mu.Lock()
	defer otpChallenges.mu.Unlock()

	if existing := otpChallenges.entries[phone]; existing != nil {
		if elapsed := now.Sub(existing.LastRequestAt); elapsed < otpResendCooldown {
			remaining := int((otpResendCooldown - elapsed).Seconds())
			if remaining < 1 {
				remaining = 1
			}
			return fmt.Errorf("mohon tunggu %d detik sebelum meminta kode baru", remaining)
		}
	}

	if recentRequestCount(phone, now) >= otpRequestPerHour {
		return fmt.Errorf("terlalu banyak permintaan kode OTP. Silakan coba lagi dalam 1 jam")
	}

	otpChallenges.entries[phone] = &otpEntry{
		Hash:          codeHash,
		ExpiresAt:     now.Add(otpTTL),
		LastRequestAt: now,
		Attempts:      0,
	}
	otpChallenges.requests[phone] = append(otpChallenges.requests[phone], now)

	return nil
}

// consumeOTP validates the submitted code and invalidates the challenge.
func consumeOTP(phone string, code string) error {
	otpChallenges.mu.Lock()
	defer otpChallenges.mu.Unlock()

	entry := otpChallenges.entries[phone]
	if entry == nil {
		return fmt.Errorf("kode OTP tidak ditemukan. Silakan minta kode baru")
	}
	if time.Now().After(entry.ExpiresAt) {
		delete(otpChallenges.entries, phone)
		return fmt.Errorf("kode OTP sudah kedaluwarsa. Silakan minta kode baru")
	}
	if entry.Attempts >= otpMaxAttempts {
		delete(otpChallenges.entries, phone)
		return fmt.Errorf("terlalu banyak percobaan. Silakan minta kode baru")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(entry.Hash), []byte(code)); err != nil {
		entry.Attempts++
		remaining := otpMaxAttempts - entry.Attempts
		if remaining <= 0 {
			delete(otpChallenges.entries, phone)
			return fmt.Errorf("terlalu banyak percobaan. Silakan minta kode baru")
		}
		return fmt.Errorf("kode OTP salah. Sisa percobaan: %d", remaining)
	}

	// success -> single use
	delete(otpChallenges.entries, phone)
	return nil
}

// resetOTPStore clears all challenges. Used by tests.
func resetOTPStore() {
	otpChallenges.mu.Lock()
	defer otpChallenges.mu.Unlock()
	otpChallenges.entries = make(map[string]*otpEntry)
	otpChallenges.requests = make(map[string][]time.Time)
}

// ---------------------------------------------------------------------------
// WhatsApp delivery
// ---------------------------------------------------------------------------

// sendWhatsAppOTP delivers the OTP through the configured WhatsApp gateway.
// Supported providers (env WA_OTP_PROVIDER):
//   - "fonnte"  : Fonnte API (https://api.fonnte.com/send)
//   - "service" : local whatsapp-service exposed by this backend (/api/wa/send-message)
//   - unset     : OTP is only written to the log (development fallback)
func sendWhatsAppOTP(phone string, code string) error {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("WA_OTP_PROVIDER")))
	message := fmt.Sprintf(
		"Kode OTP Lilly Networks Anda: %s\n\nBerlaku %d menit. JANGAN bagikan kode ini kepada siapa pun, termasuk petugas kami.",
		code, int(otpTTL.Minutes()),
	)

	switch provider {
	case "fonnte":
		return sendViaFonnte(phone, message)
	case "service":
		return sendViaWAService(phone, message)
	default:
		log.Printf("[OTP] WA_OTP_PROVIDER not configured, OTP for %s is: %s", maskPhone(phone), code)
		return nil
	}
}

func sendViaFonnte(phone string, message string) error {
	token := strings.TrimSpace(os.Getenv("FONNTE_TOKEN"))
	if token == "" {
		return fmt.Errorf("FONNTE_TOKEN belum dikonfigurasi")
	}
	endpoint := strings.TrimSpace(os.Getenv("FONNTE_API_URL"))
	if endpoint == "" {
		endpoint = "https://api.fonnte.com/send"
	}

	form := url.Values{}
	form.Set("target", phone)
	form.Set("message", message)

	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", token)

	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("whatsapp gateway menolak permintaan (status %d)", res.StatusCode)
	}
	return nil
}

func sendViaWAService(phone string, message string) error {
	baseURL := strings.TrimSpace(os.Getenv("WA_SERVICE_URL"))
	if baseURL == "" {
		baseURL = "http://127.0.0.1:3001"
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/api/wa/send-message"

	payload := fmt.Sprintf(`{"phone":%q,"message":%q}`, phone, message)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("whatsapp service menolak permintaan (status %d)", res.StatusCode)
	}
	return nil
}
