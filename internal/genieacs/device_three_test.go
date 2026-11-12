package genieacs

import (
	"testing"
	"time"
)


// buildSimulatorDoc creates a GenieACS-style document for one of the simulator devices
func buildSimulatorDoc(manufacturer, productClass, serial, rx, tx, path string) map[string]interface{} {
	recent := time.Now().Add(-30 * time.Second).Format(time.RFC3339Nano)

	rxLeaf := map[string]interface{}{"_value": rx}
	txLeaf := map[string]interface{}{"_value": tx}

	wanDevice := map[string]interface{}{
		"1": map[string]interface{}{
			path: map[string]interface{}{
				"RXPower": rxLeaf,
				"TXPower": txLeaf,
			},
		},
	}

	return map[string]interface{}{
		"_id":         serial,
		"_lastInform": recent,
		"_deviceId": map[string]interface{}{
			"_Manufacturer": manufacturer,
			"_ProductClass": productClass,
			"_SerialNumber": serial,
		},
		"InternetGatewayDevice": map[string]interface{}{
			"WANDevice": wanDevice,
		},
	}
}

// TestExtractMetrics_HuaweiNormal covers the Huawei device from the simulator (normal range)
func TestExtractMetrics_HuaweiNormal(t *testing.T) {
	cfg, err := LoadConfig(findConfigPath(t))
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	doc := buildSimulatorDoc("Huawei", "HG8245H", "HWTC12345678", "-19.40", "2.30",
		"X_GponInterfaceConfig")

	metrics := ExtractMetrics(doc, cfg)
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}
	if metrics.RxPower == nil || *metrics.RxPower != -19.4 {
		t.Fatalf("RxPower = %v, want -19.4", metrics.RxPower)
	}
	if metrics.OpticalStatus != OpticalStatusNormal {
		t.Errorf("OpticalStatus = %v, want normal", metrics.OpticalStatus)
	}
	if !metrics.IsOnline {
		t.Error("expected device to be online")
	}
}

// TestExtractMetrics_ZteWarning covers the ZTE device (warning range)
func TestExtractMetrics_ZteWarning(t *testing.T) {
	cfg, err := LoadConfig(findConfigPath(t))
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	doc := buildSimulatorDoc("ZTE", "F609", "ZTEG87654321", "-26.20", "1.80",
		"X_ZTE-COM_WANPONInterfaceConfig")

	metrics := ExtractMetrics(doc, cfg)
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}
	if metrics.RxPower == nil || *metrics.RxPower != -26.2 {
		t.Fatalf("RxPower = %v, want -26.2", metrics.RxPower)
	}
	if metrics.OpticalStatus != OpticalStatusWarning {
		t.Errorf("OpticalStatus = %v, want warning", metrics.OpticalStatus)
	}
}

// TestExtractMetrics_FiberhomeMilliDbm exercises the TR-181 profile end-to-end,
// including the 0.001 milli-dBm scale.
func TestExtractMetrics_FiberhomeMilliDbm(t *testing.T) {
	cfg, err := LoadConfig(findConfigPath(t))
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	recent := time.Now().Add(-30 * time.Second).Format(time.RFC3339Nano)
	doc := map[string]interface{}{
		"_id":         "FH88889999",
		"_lastInform": recent,
		"_deviceId": map[string]interface{}{
			"_Manufacturer": "Fiberhome",
			"_ProductClass": "AN5506-04-F",
			"_SerialNumber": "FH88889999",
		},
		"Device": map[string]interface{}{
			"Optical": map[string]interface{}{
				"Interface": map[string]interface{}{
					"1": map[string]interface{}{
						"OpticalSignalLevel":   map[string]interface{}{"_value": "-28800"},
						"TransmitOpticalLevel": map[string]interface{}{"_value": "2100"},
					},
				},
			},
		},
	}

	metrics := ExtractMetrics(doc, cfg)
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}
	if metrics.RxPower == nil {
		t.Fatal("expected RxPower to be parsed for Fiberhome TR-181")
	}
	if *metrics.RxPower > -28.0 || *metrics.RxPower < -29.5 {
		t.Errorf("RxPower = %v dBm, want ~-28.8 (milli-dBm scaled)", *metrics.RxPower)
	}
	if metrics.OpticalStatus != OpticalStatusCritical {
		t.Errorf("OpticalStatus = %v, want critical", metrics.OpticalStatus)
	}
	if metrics.SerialNumber != "FH88889999" {
		t.Errorf("SerialNumber = %q, want FH88889999", metrics.SerialNumber)
	}
}

// TestExtractNestedValue_Wildcard verifies wildcard paths resolve across branches.
func TestExtractNestedValue_Wildcard(t *testing.T) {
	doc := map[string]interface{}{
		"InternetGatewayDevice": map[string]interface{}{
			"WANDevice": map[string]interface{}{
				"2": map[string]interface{}{
					"X_GponInterfaceConfig": map[string]interface{}{
						"RXPower": map[string]interface{}{"_value": "-21.75"},
					},
				},
			},
		},
	}

	// Index 1 does not exist, only index 2
	if _, _, ok := ExtractNestedValue(doc, "InternetGatewayDevice.WANDevice.1.X_GponInterfaceConfig.RXPower"); ok {
		t.Error("expected indexed lookup at 1 to fail")
	}

	val, _, ok := ExtractNestedValueAny(doc, "InternetGatewayDevice.WANDevice.*.X_GponInterfaceConfig.RXPower")
	if !ok {
		t.Fatal("wildcard lookup failed")
	}

	parsed, err := ParseFloatValue(val, 1.0)
	if err != nil {
		t.Fatalf("ParseFloatValue error: %v", err)
	}
	if *parsed != -21.75 {
		t.Errorf("wildcard parsed value = %v, want -21.75", *parsed)
	}
}
