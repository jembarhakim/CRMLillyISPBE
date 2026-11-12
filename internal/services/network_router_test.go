package services

import (
	"strings"
	"testing"
)

func TestMockRouterActions(t *testing.T) {
	router := NewMockRouter()
	mac := "AA:BB:CC:DD:EE:01"

	if err := router.Isolate(mac); err != nil {
		t.Fatalf("Isolate() error = %v", err)
	}
	if !router.IsIsolated(mac) {
		t.Fatal("Isolate() did not mark MAC as isolated")
	}
	if err := router.Unisolate(mac); err != nil {
		t.Fatalf("Unisolate() error = %v", err)
	}
	if router.IsIsolated(mac) {
		t.Fatal("Unisolate() did not clear isolation")
	}
	if err := router.BindDHCPLease(mac, "10.10.21.10"); err != nil {
		t.Fatalf("BindDHCPLease() error = %v", err)
	}
	sessions, err := router.GetActiveSessions()
	if err != nil {
		t.Fatalf("GetActiveSessions() error = %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("GetActiveSessions() count = %d, want 2", len(sessions))
	}
}

func TestRouterRejectsInvalidAddresses(t *testing.T) {
	router := NewMockRouter()
	if err := router.Isolate("not-a-mac"); err == nil {
		t.Fatal("Isolate() accepted invalid MAC")
	}
	if err := router.BindDHCPLease("AA:BB:CC:DD:EE:01", "invalid"); err == nil {
		t.Fatal("BindDHCPLease() accepted invalid IP")
	}
}

func TestSecretEncryptionRoundTrip(t *testing.T) {
	t.Setenv("NETWORK_CREDENTIAL_KEY", "0123456789abcdef0123456789abcdef")
	encrypted, err := EncryptSecret("router-password")
	if err != nil {
		t.Fatalf("EncryptSecret() error = %v", err)
	}
	if strings.Contains(encrypted, "router-password") {
		t.Fatal("encrypted secret contains plaintext")
	}
	decrypted, err := DecryptSecret(encrypted)
	if err != nil {
		t.Fatalf("DecryptSecret() error = %v", err)
	}
	if decrypted != "router-password" {
		t.Fatalf("decrypted secret = %q, want router-password", decrypted)
	}
}
