package genieacs

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseFloatValue(t *testing.T) {
	tests := []struct {
		name      string
		raw       interface{}
		scale     float64
		expected  float64
		expectErr bool
	}{
		{
			name:      "Huawei clean string",
			raw:       "-19.50",
			scale:     1.0,
			expected:  -19.50,
			expectErr: false,
		},
		{
			name:      "ZTE string with unit suffix",
			raw:       "-26.14 dBm",
			scale:     1.0,
			expected:  -26.14,
			expectErr: false,
		},
		{
			name:      "TR-181 integer milli-dBm with scale 0.001",
			raw:       -28760,
			scale:     0.001,
			expected:  -28.76,
			expectErr: false,
		},
		{
			name:      "Positive Tx power",
			raw:       "2.35 dBm",
			scale:     1.0,
			expected:  2.35,
			expectErr: false,
		},
		{
			name:      "Nil value",
			raw:       nil,
			scale:     1.0,
			expectErr: true,
		},
		{
			name:      "Empty string",
			raw:       "",
			scale:     1.0,
			expectErr: true,
		},
		{
			name:      "Non-numeric string",
			raw:       "N/A",
			scale:     1.0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := ParseFloatValue(tt.raw, tt.scale)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if val == nil {
				t.Fatalf("expected non-nil value")
			}
			// Compare with small tolerance
			diff := *val - tt.expected
			if diff < -0.001 || diff > 0.001 {
				t.Errorf("expected %.3f, got %.3f", tt.expected, *val)
			}
		})
	}
}

func TestEvaluateOpticalStatus(t *testing.T) {
	thresholds := OpticalThresholds{
		NormalMin:   -25.0,
		NormalMax:   -8.0,
		WarningMin:  -27.0,
		CriticalMin: -27.0,
	}

	val := func(f float64) *float64 { return &f }

	tests := []struct {
		name     string
		rx       *float64
		expected OpticalStatus
	}{
		{"Nil reading", nil, OpticalStatusUnknown},
		{"Normal optimal", val(-19.5), OpticalStatusNormal},
		{"Normal boundary min", val(-25.0), OpticalStatusNormal},
		{"Normal boundary max", val(-8.0), OpticalStatusNormal},
		{"Warning moderate", val(-26.2), OpticalStatusWarning},
		{"Warning boundary min", val(-27.0), OpticalStatusWarning},
		{"Critical high attenuation", val(-28.5), OpticalStatusCritical},
		{"Critical optical overload", val(-6.5), OpticalStatusCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := EvaluateOpticalStatus(tt.rx, thresholds)
			if res != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, res)
			}
		})
	}
}

func TestExtractNestedValue(t *testing.T) {
	jsonRaw := `{
		"_id": "202BC1-HG8245H-HWTC12345678",
		"InternetGatewayDevice": {
			"WANDevice": {
				"1": {
					"X_GponInterfaceConfig": {
						"RXPower": {
							"_value": "-19.45",
							"_type": "xsd:string",
							"_timestamp": "2026-10-26T15:00:00.000Z"
						}
					}
				}
			}
		}
	}`

	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(jsonRaw), &doc); err != nil {
		t.Fatalf("failed to unmarshal test JSON: %v", err)
	}

	val, ts, ok := ExtractNestedValue(doc, "InternetGatewayDevice.WANDevice.1.X_GponInterfaceConfig.RXPower")
	if !ok {
		t.Fatalf("expected ExtractNestedValue to return true")
	}
	if val != "-19.45" {
		t.Errorf("expected '-19.45', got %v", val)
	}
	if ts.IsZero() {
		t.Errorf("expected valid timestamp, got zero")
	}

	// Test non-existent path
	_, _, ok = ExtractNestedValue(doc, "InternetGatewayDevice.NonExistent.Path")
	if ok {
		t.Errorf("expected false for non-existent path")
	}
}

func TestExtractMetricsFromDoc(t *testing.T) {
	cfg := DefaultConfig()

	// Simulate NBI document for Huawei
	recentTime := time.Now().Add(-30 * time.Second).Format(time.RFC3339Nano)
	doc := map[string]interface{}{
		"_id":         "202BC1-HG8245H-HWTC12345678",
		"_lastInform": recentTime,
		"_deviceId": map[string]interface{}{
			"_Manufacturer": "Huawei",
			"_OUI":          "202BC1",
			"_ProductClass": "HG8245H",
			"_SerialNumber": "HWTC12345678",
		},
		"InternetGatewayDevice": map[string]interface{}{
			"WANDevice": map[string]interface{}{
				"1": map[string]interface{}{
					"X_GponInterfaceConfig": map[string]interface{}{
						"RXPower": map[string]interface{}{"_value": "-19.35"},
						"TXPower": map[string]interface{}{"_value": "2.25"},
					},
				},
			},
			"LANDevice": map[string]interface{}{
				"1": map[string]interface{}{
					"WLANConfiguration": map[string]interface{}{
						"1": map[string]interface{}{
							"SSID": map[string]interface{}{"_value": "LillyNet_Test"},
							"AssociatedDeviceNumberOfEntries": map[string]interface{}{"_value": 3},
						},
					},
				},
			},
		},
	}

	metrics := ExtractMetrics(doc, cfg)
	if metrics == nil {
		t.Fatalf("expected non-nil metrics")
	}

	if metrics.SerialNumber != "HWTC12345678" {
		t.Errorf("expected serial HWTC12345678, got %s", metrics.SerialNumber)
	}
	if !metrics.IsOnline {
		t.Errorf("expected device to be online")
	}
	if metrics.RxPower == nil || *metrics.RxPower != -19.35 {
		t.Errorf("expected RxPower -19.35, got %v", metrics.RxPower)
	}
	if metrics.OpticalStatus != OpticalStatusNormal {
		t.Errorf("expected normal optical status, got %s", metrics.OpticalStatus)
	}
	if metrics.SSID != "LillyNet_Test" {
		t.Errorf("expected SSID 'LillyNet_Test', got %s", metrics.SSID)
	}
	if metrics.ClientCount != 3 {
		t.Errorf("expected ClientCount 3, got %d", metrics.ClientCount)
	}
	if metrics.IsStale {
		t.Errorf("expected not stale when online")
	}
}

