package genieacs

import "time"

// OpticalStatus represents the severity status of an ONT optical reading
type OpticalStatus string

const (
	OpticalStatusNormal   OpticalStatus = "normal"   // -8 to -25 dBm
	OpticalStatusWarning  OpticalStatus = "warning"  // -25 to -27 dBm
	OpticalStatusCritical OpticalStatus = "critical" // < -27 dBm or > -8 dBm (overload)
	OpticalStatusUnknown  OpticalStatus = "unknown"  // No reading available
)

// DeviceParameterValue represents a leaf parameter node from GenieACS NBI
type DeviceParameterValue struct {
	Value     interface{} `json:"_value"`
	Type      string      `json:"_type"`
	Timestamp time.Time   `json:"_timestamp"`
	Writable  bool        `json:"_writable"`
}

// DeviceDoc represents a raw device document returned by GenieACS NBI
type DeviceDoc map[string]interface{}

// OntMetrics represents clean, normalized metrics extracted from an ONT
type OntMetrics struct {
	DeviceID        string        `json:"device_id"`
	SerialNumber    string        `json:"serial_number"`
	Manufacturer    string        `json:"manufacturer"`
	ModelName       string        `json:"model_name"`
	SoftwareVersion string        `json:"software_version"`
	IPAddress       string        `json:"ip_address"`
	MacAddress      string        `json:"mac_address"`
	IsOnline        bool          `json:"is_online"`
	RxPower         *float64      `json:"rx_power"`
	TxPower         *float64      `json:"tx_power"`
	OpticalStatus   OpticalStatus `json:"optical_status"`
	SSID            string        `json:"ssid"`
	ClientCount     int           `json:"client_count"`
	LastInform      *time.Time    `json:"last_inform"`
	IsStale         bool          `json:"is_stale"`
}

