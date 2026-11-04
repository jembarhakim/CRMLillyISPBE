package wa

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type WAService struct {
	apiURL   string
	apiToken string
	phoneID  string
	driver   string
	client   *http.Client
}

type SendResult struct {
	MessageID string
}

type WhatsAppMessage struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

type WhatsAppAPIRequest struct {
	MessagingProduct string `json:"messaging_product"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Text             struct {
		Body string `json:"body"`
	} `json:"text"`
}

type WhatsAppAPIResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

func NewWAService() *WAService {
	driver := strings.ToLower(strings.TrimSpace(os.Getenv("WHATSAPP_DRIVER")))
	serviceURL := os.Getenv("WHATSAPP_SERVICE_URL")
	if serviceURL == "" {
		serviceURL = "http://localhost:3002" // Default whatsapp-web.js service URL
	}
	if driver == "" {
		driver = "local"
		if strings.Contains(serviceURL, "graph.facebook.com") {
			driver = "meta"
		}
	}
	apiURL := serviceURL
	if driver == "fonnte" && os.Getenv("WHATSAPP_FONNTE_URL") != "" {
		apiURL = os.Getenv("WHATSAPP_FONNTE_URL")
	} else if driver == "meta" {
		if metaURL := os.Getenv("WHATSAPP_META_API_URL"); metaURL != "" {
			apiURL = metaURL
		} else if !strings.Contains(serviceURL, "graph.facebook.com") {
			apiURL = "https://graph.facebook.com/v21.0"
		}
	}
	apiToken := os.Getenv("WHATSAPP_API_TOKEN")
	if driver == "fonnte" {
		apiToken = firstNonEmpty(os.Getenv("WHATSAPP_FONNTE_TOKEN"), apiToken)
	}

	return &WAService{
		apiURL:   strings.TrimRight(apiURL, "/"),
		apiToken: apiToken,
		phoneID:  os.Getenv("WHATSAPP_PHONE_ID"),
		driver:   driver,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *WAService) Driver() string { return s.driver }

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// normalizePhoneNumber normalizes Indonesian phone numbers to WhatsApp format
// Converts: 08970833338 -> 628970833338
// Converts: +628970833338 -> 628970833338
// Converts: 628970833338 -> 628970833338 (already correct)
func NormalizePhoneNumber(phone string) string {
	var digits strings.Builder
	for _, char := range strings.TrimSpace(phone) {
		if char >= '0' && char <= '9' {
			digits.WriteRune(char)
		}
	}
	phone = digits.String()
	if strings.HasPrefix(phone, "00") {
		phone = phone[2:]
	}

	// If starts with 0, replace with 62 (Indonesia country code)
	if strings.HasPrefix(phone, "0") {
		phone = "62" + phone[1:]
	}

	// If doesn't start with 62, add it (assuming Indonesian numbers)
	if !strings.HasPrefix(phone, "62") && len(phone) >= 9 {
		phone = "62" + phone
	}

	return phone
}

func (s *WAService) SendMessage(to, message string) error {
	_, err := s.SendMessageWithResult(to, message)
	return err
}

func (s *WAService) SendMessageWithResult(to, message string) (SendResult, error) {
	// Normalize phone number format (remove +, convert 0 to 62)
	normalizedTo := NormalizePhoneNumber(to)

	// Ensure phone number format (add country code if missing)
	if !s.isValidPhoneNumber(normalizedTo) {
		return SendResult{}, fmt.Errorf("invalid phone number format: %s (normalized: %s, use format: 628123456789)", to, normalizedTo)
	}

	switch s.driver {
	case "local", "whatsapp-web.js", "whatsapp-web":
		return s.sendViaLocalService(normalizedTo, message)
	case "fonnte":
		return s.sendViaFonnte(normalizedTo, message)
	case "meta", "cloud":
		if s.apiToken == "" || s.phoneID == "" {
			return SendResult{}, fmt.Errorf("Meta WhatsApp API requires WHATSAPP_API_TOKEN and WHATSAPP_PHONE_ID")
		}
		return s.sendViaMetaAPI(normalizedTo, message)
	default:
		return SendResult{}, fmt.Errorf("unsupported WHATSAPP_DRIVER %q", s.driver)
	}
}

// sendViaLocalService sends message via local whatsapp-web.js service
func (s *WAService) sendViaLocalService(to, message string) (SendResult, error) {
	// Prepare request for local service
	reqBody := map[string]string{
		"to":      to,
		"message": message,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return SendResult{}, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request to local service
	url := fmt.Sprintf("%s/send-message", s.apiURL)
	fmt.Printf("[WA] Sending via local service to %s\n", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return SendResult{}, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return SendResult{}, fmt.Errorf("failed to send request to local service: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SendResult{}, fmt.Errorf("failed to read response: %v", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorResp struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return SendResult{}, fmt.Errorf("WhatsApp service error: %s", errorResp.Message)
		}
		return SendResult{}, fmt.Errorf("WhatsApp service error: %d - %s", resp.StatusCode, string(body))
	}

	// Parse success response
	var successResp struct {
		Status string `json:"status"`
		Data   struct {
			MessageID string `json:"message_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &successResp); err == nil {
		fmt.Printf("[WA] Message sent successfully to %s (Message ID: %s)\n", to, successResp.Data.MessageID)
		return SendResult{MessageID: successResp.Data.MessageID}, nil
	}

	fmt.Printf("[WA] Message sent successfully to %s\n", to)
	return SendResult{}, nil
}

// sendViaMetaAPI sends message via Meta WhatsApp Business API (legacy)
func (s *WAService) sendViaMetaAPI(to, message string) (SendResult, error) {
	// Prepare the request for Meta WhatsApp Business API
	reqBody := WhatsAppAPIRequest{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "text",
	}
	reqBody.Text.Body = message

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return SendResult{}, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request to Meta Graph API
	url := fmt.Sprintf("%s/%s/messages", s.apiURL, s.phoneID)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return SendResult{}, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers for Meta API
	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return SendResult{}, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SendResult{}, fmt.Errorf("failed to read response: %v", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return SendResult{}, fmt.Errorf("Meta WhatsApp API error: %d - %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp WhatsAppAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return SendResult{}, fmt.Errorf("failed to parse response: %v", err)
	}

	if len(apiResp.Messages) > 0 {
		fmt.Printf("[WA] Message sent successfully to %s (Message ID: %s)\n", to, apiResp.Messages[0].ID)
		return SendResult{MessageID: apiResp.Messages[0].ID}, nil
	}

	return SendResult{}, fmt.Errorf("Meta responded 200 but no message id returned for %s", to)
}

func (s *WAService) sendViaFonnte(to, message string) (SendResult, error) {
	if s.apiToken == "" {
		return SendResult{}, fmt.Errorf("Fonnte requires WHATSAPP_FONNTE_TOKEN")
	}
	apiURL := s.apiURL
	if apiURL == "" || apiURL == "http://localhost:3002" {
		apiURL = "https://api.fonnte.com/send"
	}
	payload, err := json.Marshal(map[string]string{"target": to, "message": message})
	if err != nil {
		return SendResult{}, fmt.Errorf("failed to marshal Fonnte request: %w", err)
	}
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(payload))
	if err != nil {
		return SendResult{}, fmt.Errorf("failed to create Fonnte request: %w", err)
	}
	req.Header.Set("Authorization", s.apiToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return SendResult{}, fmt.Errorf("failed to send request to Fonnte: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SendResult{}, fmt.Errorf("failed to read Fonnte response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return SendResult{}, fmt.Errorf("Fonnte API error: %d - %s", resp.StatusCode, string(body))
	}
	var response struct {
		Status bool     `json:"status"`
		Detail string   `json:"detail"`
		ID     string   `json:"id"`
		IDs    []string `json:"ids"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return SendResult{}, fmt.Errorf("failed to parse Fonnte response: %w", err)
	}
	if !response.Status {
		return SendResult{}, fmt.Errorf("Fonnte rejected the message: %s", response.Detail)
	}
	messageID := response.ID
	if messageID == "" && len(response.IDs) > 0 {
		messageID = response.IDs[0]
	}
	return SendResult{MessageID: messageID}, nil
}

// isValidPhoneNumber checks if the phone number is in the correct format
// WhatsApp Business API requires: 628123456789 (no +, no leading 0, with country code)
func (s *WAService) isValidPhoneNumber(phone string) bool {
	return IsValidPhoneNumber(phone)
}

func IsValidPhoneNumber(phone string) bool {
	// Remove + if present for validation
	phone = strings.TrimSpace(phone)
	if strings.HasPrefix(phone, "+") {
		phone = phone[1:]
	}

	// Basic validation: should be 10-15 digits
	if len(phone) < 10 || len(phone) > 15 {
		return false
	}

	// Check if it's all numeric (after removing +)
	for _, char := range phone {
		if char < '0' || char > '9' {
			return false
		}
	}

	// Should start with country code (62 for Indonesia, or other country codes)
	// Accept 62 (Indonesia), 1 (US/Canada), etc.
	if !strings.HasPrefix(phone, "62") && !strings.HasPrefix(phone, "1") {
		// Allow other country codes but must be at least 2 digits
		if len(phone) < 11 {
			return false
		}
	}

	return true
}
