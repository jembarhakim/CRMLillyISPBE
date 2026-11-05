package wa

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizePhoneNumber(t *testing.T) {
	tests := map[string]string{
		"0812 3456-7890":      "6281234567890",
		"+62 (812) 3456-7890": "6281234567890",
		"006281234567890":     "6281234567890",
	}
	for input, expected := range tests {
		if got := NormalizePhoneNumber(input); got != expected {
			t.Errorf("NormalizePhoneNumber(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestSendMessageWithResultLocalGateway(t *testing.T) {
	t.Setenv("WHATSAPP_DRIVER", "local")
	t.Setenv("WHATSAPP_FONNTE_TOKEN", "")
	t.Setenv("WHATSAPP_API_TOKEN", "")
	t.Setenv("WHATSAPP_SERVICE_URL", "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/send-message" {
			t.Errorf("unexpected request path %q", r.URL.Path)
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if payload["to"] != "6281234567890" {
			t.Errorf("request recipient = %q", payload["to"])
		}
		_, _ = w.Write([]byte(`{"status":"success","data":{"message_id":"local-message-1"}}`))
	}))
	defer server.Close()
	t.Setenv("WHATSAPP_SERVICE_URL", server.URL)

	result, err := NewWAService().SendMessageWithResult("081234567890", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if result.MessageID != "local-message-1" {
		t.Fatalf("message id = %q", result.MessageID)
	}
}

func TestSendMessageWithResultFonnteGateway(t *testing.T) {
	t.Setenv("WHATSAPP_DRIVER", "fonnte")
	t.Setenv("WHATSAPP_FONNTE_TOKEN", "test-token")
	t.Setenv("WHATSAPP_API_TOKEN", "")
	t.Setenv("WHATSAPP_FONNTE_URL", "")
	t.Setenv("WHATSAPP_SERVICE_URL", "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "test-token" {
			t.Errorf("unexpected authorization header")
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if payload["target"] != "6281234567890" {
			t.Errorf("request recipient = %q", payload["target"])
		}
		_, _ = w.Write([]byte(`{"status":true,"id":"fonnte-message-1"}`))
	}))
	defer server.Close()
	t.Setenv("WHATSAPP_FONNTE_URL", server.URL)

	result, err := NewWAService().SendMessageWithResult("+62 812-3456-7890", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if result.MessageID != "fonnte-message-1" {
		t.Fatalf("message id = %q", result.MessageID)
	}
}

func TestSendMessageWithResultMetaGateway(t *testing.T) {
	t.Setenv("WHATSAPP_DRIVER", "meta")
	t.Setenv("WHATSAPP_API_TOKEN", "meta-token")
	t.Setenv("WHATSAPP_FONNTE_TOKEN", "")
	t.Setenv("WHATSAPP_PHONE_ID", "phone-id")
	t.Setenv("WHATSAPP_SERVICE_URL", "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v21.0/phone-id/messages" {
			t.Errorf("unexpected request path %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer meta-token" {
			t.Errorf("unexpected authorization header")
		}
		_, _ = w.Write([]byte(`{"messages":[{"id":"meta-message-1"}]}`))
	}))
	defer server.Close()
	t.Setenv("WHATSAPP_META_API_URL", server.URL+"/v21.0")

	result, err := NewWAService().SendMessageWithResult("081234567890", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if result.MessageID != "meta-message-1" {
		t.Fatalf("message id = %q", result.MessageID)
	}
}
