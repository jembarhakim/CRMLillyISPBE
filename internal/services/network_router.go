package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"

	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
)

type Router interface {
	Isolate(macAddress string) error
	Unisolate(macAddress string) error
	GetActiveSessions() ([]RouterActiveSession, error)
	BindDHCPLease(macAddress, ipAddress string) error
}

type RouterActiveSession struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	MAC     string `json:"mac"`
	User    string `json:"user"`
	Uptime  string `json:"uptime"`
}

type MockRouter struct {
	mu       sync.Mutex
	isolated map[string]bool
	leases   map[string]string
	sessions []RouterActiveSession
}

func NewMockRouter() *MockRouter {
	return &MockRouter{
		isolated: make(map[string]bool),
		leases:   make(map[string]string),
		sessions: []RouterActiveSession{
			{ID: "mock-1", Address: "10.10.21.10", MAC: "AA:BB:CC:DD:EE:01", User: "demo-user-1", Uptime: "2h10m"},
			{ID: "mock-2", Address: "10.10.21.11", MAC: "AA:BB:CC:DD:EE:02", User: "demo-user-2", Uptime: "35m"},
		},
	}
}

func validateRouterMAC(macAddress string) error {
	if _, err := net.ParseMAC(strings.TrimSpace(macAddress)); err != nil {
		return fmt.Errorf("invalid MAC address: %w", err)
	}
	return nil
}

func validateRouterIP(ipAddress string) error {
	if net.ParseIP(strings.TrimSpace(ipAddress)) == nil {
		return errors.New("invalid IP address")
	}
	return nil
}

func (r *MockRouter) Isolate(macAddress string) error {
	if err := validateRouterMAC(macAddress); err != nil {
		return err
	}
	r.mu.Lock()
	r.isolated[strings.ToUpper(macAddress)] = true
	r.mu.Unlock()
	return nil
}

func (r *MockRouter) Unisolate(macAddress string) error {
	if err := validateRouterMAC(macAddress); err != nil {
		return err
	}
	r.mu.Lock()
	delete(r.isolated, strings.ToUpper(macAddress))
	r.mu.Unlock()
	return nil
}

func (r *MockRouter) GetActiveSessions() ([]RouterActiveSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]RouterActiveSession, len(r.sessions))
	copy(result, r.sessions)
	return result, nil
}

func (r *MockRouter) BindDHCPLease(macAddress, ipAddress string) error {
	if err := validateRouterMAC(macAddress); err != nil {
		return err
	}
	if err := validateRouterIP(ipAddress); err != nil {
		return err
	}
	r.mu.Lock()
	r.leases[strings.ToUpper(macAddress)] = ipAddress
	r.mu.Unlock()
	return nil
}

func (r *MockRouter) IsIsolated(macAddress string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.isolated[strings.ToUpper(macAddress)]
}

type MikroTikRouter struct {
	service *MikroTikService
}

func NewMikroTikRouter(service *MikroTikService) *MikroTikRouter {
	return &MikroTikRouter{service: service}
}

func (r *MikroTikRouter) ensureConnected() error {
	if r.service == nil {
		return errors.New("MikroTik service is not configured")
	}
	if r.service.IsConnected() {
		return nil
	}
	return r.service.ReconnectWithBackoff(3)
}

func (r *MikroTikRouter) Isolate(macAddress string) error {
	if err := validateRouterMAC(macAddress); err != nil {
		return err
	}
	if err := r.ensureConnected(); err != nil {
		return err
	}
	_, err := r.service.ExecuteCommand(fmt.Sprintf("/ip/hotspot/ip-binding set [find mac-address=\"%s\"] type=blocked", macAddress))
	return err
}

func (r *MikroTikRouter) Unisolate(macAddress string) error {
	if err := validateRouterMAC(macAddress); err != nil {
		return err
	}
	if err := r.ensureConnected(); err != nil {
		return err
	}
	_, err := r.service.ExecuteCommand(fmt.Sprintf("/ip/hotspot/ip-binding set [find mac-address=\"%s\"] type=bypassed", macAddress))
	return err
}

func (r *MikroTikRouter) GetActiveSessions() ([]RouterActiveSession, error) {
	if err := r.ensureConnected(); err != nil {
		return nil, err
	}
	output, err := r.service.ExecuteCommand("/ip/hotspot/active print terse")
	if err != nil {
		return nil, err
	}
	sessions := make([]RouterActiveSession, 0)
	for _, line := range strings.Split(output, "\n") {
		session := RouterActiveSession{}
		for _, field := range strings.Fields(line) {
			key, value, ok := strings.Cut(field, "=")
			if !ok {
				continue
			}
			switch key {
			case ".id":
				session.ID = strings.Trim(value, "\"")
			case "address":
				session.Address = strings.Trim(value, "\"")
			case "mac-address":
				session.MAC = strings.Trim(value, "\"")
			case "user":
				session.User = strings.Trim(value, "\"")
			case "uptime":
				session.Uptime = strings.Trim(value, "\"")
			}
		}
		if session.MAC != "" || session.Address != "" {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

func (r *MikroTikRouter) BindDHCPLease(macAddress, ipAddress string) error {
	if err := validateRouterMAC(macAddress); err != nil {
		return err
	}
	if err := validateRouterIP(ipAddress); err != nil {
		return err
	}
	if err := r.ensureConnected(); err != nil {
		return err
	}
	_, err := r.service.ExecuteCommand(fmt.Sprintf("/ip/dhcp-server/lease add address=\"%s\" mac-address=\"%s\" comment=\"CRM-NET-01\"", ipAddress, macAddress))
	return err
}

func NewRouter(service *MikroTikService) Router {
	if !strings.EqualFold(os.Getenv("NETWORK_DATA_MODE"), "real") {
		return NewMockRouter()
	}
	return NewMikroTikRouter(service)
}

func EncryptSecret(plaintext string) (string, error) {
	key := []byte(os.Getenv("NETWORK_CREDENTIAL_KEY"))
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return "", errors.New("NETWORK_CREDENTIAL_KEY must be 16, 24, or 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawStdEncoding.EncodeToString(ciphertext), nil
}

func DecryptSecret(encoded string) (string, error) {
	key := []byte(os.Getenv("NETWORK_CREDENTIAL_KEY"))
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return "", errors.New("NETWORK_CREDENTIAL_KEY must be 16, 24, or 32 bytes")
	}
	blob, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(blob) < gcm.NonceSize() {
		return "", errors.New("encrypted secret is too short")
	}
	nonce, ciphertext := blob[:gcm.NonceSize()], blob[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func SaveRouterCredential(db *gorm.DB, name, host string, port int, username, password string, useTLS bool) error {
	if db == nil {
		return errors.New("database is not configured")
	}
	encrypted, err := EncryptSecret(password)
	if err != nil {
		return err
	}
	credential := entities.NetworkRouterCredential{
		Name:              strings.TrimSpace(name),
		Host:              strings.TrimSpace(host),
		Port:              port,
		Username:          strings.TrimSpace(username),
		EncryptedPassword: encrypted,
		UseTLS:            useTLS,
	}
	if credential.Name == "" {
		credential.Name = credential.Host
	}
	return db.Where("host = ? AND port = ? AND username = ?", credential.Host, credential.Port, credential.Username).
		Assign(map[string]interface{}{
			"name":               credential.Name,
			"encrypted_password": credential.EncryptedPassword,
			"use_tls":            credential.UseTLS,
		}).
		FirstOrCreate(&credential).Error
}
