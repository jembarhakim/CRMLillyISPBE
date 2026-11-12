package genieacs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Client is the HTTP client for interacting with GenieACS Northbound Interface (NBI)
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new GenieACS NBI client
func NewClient(baseURL string, timeout time.Duration) *Client {
	if baseURL == "" {
		baseURL = os.Getenv("GENIEACS_NBI_URL")
		if baseURL == "" {
			baseURL = "http://127.0.0.1:7557"
		}
	}
	baseURL = strings.TrimRight(baseURL, "/")

	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetDevices fetches multiple devices from GenieACS NBI with optional MongoDB query and projection.
// Performs ONE single bulk query, avoiding 1-call-per-device overhead.
func (c *Client) GetDevices(ctx context.Context, mongoQuery string, projection string) ([]DeviceDoc, error) {
	endpoint := fmt.Sprintf("%s/devices", c.BaseURL)

	params := url.Values{}
	if mongoQuery != "" {
		params.Set("query", mongoQuery)
	}
	if projection != "" {
		params.Set("projection", projection)
	}

	if len(params) > 0 {
		endpoint = fmt.Sprintf("%s?%s", endpoint, params.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create NBI request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GenieACS NBI unreachable at %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("NBI returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var devices []DeviceDoc
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		return nil, fmt.Errorf("failed to decode NBI devices response: %w", err)
	}

	return devices, nil
}

// GetDeviceBySerial finds a single device by its ONT serial number
func (c *Client) GetDeviceBySerial(ctx context.Context, serialNumber string) (*DeviceDoc, error) {
	if serialNumber == "" {
		return nil, fmt.Errorf("serial number cannot be empty")
	}

	// Query by _deviceId._SerialNumber
	queryJSON, err := json.Marshal(map[string]string{
		"_deviceId._SerialNumber": serialNumber,
	})
	if err != nil {
		return nil, err
	}

	devices, err := c.GetDevices(ctx, string(queryJSON), "")
	if err != nil {
		return nil, err
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("device with serial number %s not found in GenieACS", serialNumber)
	}

	return &devices[0], nil
}

// GetDeviceByID finds a single device by its full GenieACS _id
func (c *Client) GetDeviceByID(ctx context.Context, deviceID string) (*DeviceDoc, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device ID cannot be empty")
	}

	queryJSON, err := json.Marshal(map[string]string{
		"_id": deviceID,
	})
	if err != nil {
		return nil, err
	}

	devices, err := c.GetDevices(ctx, string(queryJSON), "")
	if err != nil {
		return nil, err
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("device with ID %s not found in GenieACS", deviceID)
	}

	return &devices[0], nil
}

// RefreshDevice enqueues a refreshObject task with Connection Request to force ONT to report fresh values
func (c *Client) RefreshDevice(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return fmt.Errorf("device ID cannot be empty")
	}

	escapedID := url.PathEscape(deviceID)
	endpoint := fmt.Sprintf("%s/devices/%s/tasks?connection_request", c.BaseURL, escapedID)

	taskPayload := map[string]interface{}{
		"name":       "refreshObject",
		"objectName": "",
	}
	bodyBytes, err := json.Marshal(taskPayload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create refresh task request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Use specific timeout for connection request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to contact GenieACS NBI for refresh: %w", err)
	}
	defer resp.Body.Close()

	// 200 OK or 202 Accepted indicates task was accepted
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GenieACS task refresh failed with HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

