package genieacs

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestParseLiveGenieACSDevice is an opt-in integration test that parses a real
// device document straight from a running GenieACS NBI instance.
// Enable with: GODEBUG=  set GENIEACS_LIVE_TEST=1 && go test ./internal/genieacs/ -run Live -v
func TestParseLiveGenieACSDevice(t *testing.T) {
	if os.Getenv("GENIEACS_LIVE_TEST") == "" {
		t.Skip("set GENIEACS_LIVE_TEST=1 to run against a live GenieACS NBI")
	}

	baseURL := os.Getenv("GENIEACS_NBI_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:7557"
	}

	client := NewClient(baseURL, 5*time.Second)
	devices, err := client.GetDevices(context.Background(), "", "")
	if err != nil {
		t.Skipf("GenieACS not reachable: %v", err)
	}
	if len(devices) == 0 {
		t.Skip("no devices registered in GenieACS")
	}

	cfg, err := LoadConfig(findConfigPath(t))
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	for _, doc := range devices {
		metrics := ExtractMetrics(doc, cfg)
		if metrics == nil {
			t.Errorf("device %v: nil metrics", doc["_id"])
			continue
		}

		t.Logf("device=%s serial=%s ip=%s mac=%s rx=%v status=%s online=%v",
			metrics.DeviceID, metrics.SerialNumber, metrics.IPAddress,
			metrics.MacAddress, metrics.RxPower, metrics.OpticalStatus, metrics.IsOnline)

		if metrics.SerialNumber == "" {
			t.Errorf("device %v: empty serial number", doc["_id"])
		}
		if metrics.IPAddress == "" {
			t.Errorf("device %s: empty IP address", metrics.SerialNumber)
		}
		if metrics.RxPower == nil {
			t.Errorf("device %s: empty RxPower", metrics.SerialNumber)
		}
	}
}

// TestHostFromURL ensures the IP is extracted from a ConnectionRequestURL,
// which is what makes the GenieACS "IP" column and our map work.
func TestHostFromURL(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"http://10.10.9.101:7547/cwmp", "10.10.9.101"},
		{"http://10.10.9.102:7547/cwmp", "10.10.9.102"},
		{"10.10.9.103:7547/cwmp", "10.10.9.103"},
		{"http://10.10.9.104/cwmp", "10.10.9.104"},
		{"", ""},
		{"not a url", ""},
		{"http://0.0.0.0:7547/cwmp", ""},
	}

	for _, c := range cases {
		if got := hostFromURL(c.input); got != c.want {
			t.Errorf("hostFromURL(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

// TestExtractMetrics_IPFromConnectionRequestURL mirrors the real GenieACS document
// where _ipAddress is empty but ConnectionRequestURL carries the device IP.
func TestExtractMetrics_IPFromConnectionRequestURL(t *testing.T) {
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
		"InternetGatewayDevice": map[string]interface{}{
			"ManagementServer": map[string]interface{}{
				"ConnectionRequestURL": map[string]interface{}{
					"_value": "http://10.10.9.103:7547/cwmp",
				},
			},
		},
	}

	metrics := ExtractMetrics(doc, cfg)
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}
	if metrics.IPAddress != "10.10.9.103" {
		t.Errorf("IPAddress = %q, want 10.10.9.103 from ConnectionRequestURL", metrics.IPAddress)
	}
}

// TestExtractMetrics_PrefersRootIPAddress ensures the root _ipAddress wins when present.
func TestExtractMetrics_PrefersRootIPAddress(t *testing.T) {
	cfg, err := LoadConfig(findConfigPath(t))
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	recent := time.Now().Add(-30 * time.Second).Format(time.RFC3339Nano)
	doc := map[string]interface{}{
		"_id":         "HWTC12345678",
		"_lastInform": recent,
		"_ipAddress":  map[string]interface{}{"_value": "10.10.9.101"},
		"_deviceId": map[string]interface{}{
			"_Manufacturer": "Huawei",
			"_ProductClass": "HG8245H",
			"_SerialNumber": "HWTC12345678",
		},
		"InternetGatewayDevice": map[string]interface{}{
			"ManagementServer": map[string]interface{}{
				"ConnectionRequestURL": map[string]interface{}{
					"_value": "http://10.99.99.99:7547/cwmp",
				},
			},
		},
	}

	metrics := ExtractMetrics(doc, cfg)
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}
	if metrics.IPAddress != "10.10.9.101" {
		t.Errorf("IPAddress = %q, want root _ipAddress 10.10.9.101", metrics.IPAddress)
	}
}

// buildSimulatorDocWithIP creates a GenieACS doc that also reports IP + MAC,
// mirroring the payload produced by infra/genieacs/simulator/simulator.js
func buildSimulatorDocWithIP(manufacturer, productClass, serial, rx, tx, path,
	wanIP, lanIP, mac string) map[string]interface{} {
	doc := buildSimulatorDoc(manufacturer, productClass, serial, rx, tx, path)

	doc["InternetGatewayDevice"].(map[string]interface{})["WANDevice"].
		(map[string]interface{})["1"].(map[string]interface{})["WANConnectionDevice"] =
		map[string]interface{}{
			"1": map[string]interface{}{
				"WANIPConnection": map[string]interface{}{
					"1": map[string]interface{}{
						"ExternalIPAddress": map[string]interface{}{"_value": wanIP},
					},
				},
			},
		}

	doc["InternetGatewayDevice"].(map[string]interface{})["LANDevice"] =
		map[string]interface{}{
			"1": map[string]interface{}{
				"LANHostConfigManagement": map[string]interface{}{
					"IPInterface": map[string]interface{}{
						"1": map[string]interface{}{
							"IPInterfaceIPAddress": map[string]interface{}{"_value": lanIP},
						},
					},
				},
				"LANDeviceInfo": map[string]interface{}{
					"1": map[string]interface{}{
						"LANEthernetInterfaceConfig": map[string]interface{}{
							"1": map[string]interface{}{
								"MACAddress": map[string]interface{}{"_value": mac},
							},
						},
					},
				},
			},
		}

	return doc
}

// TestExtractMetrics_DummyIPAddresses verifies the dummy WAN IP, LAN IP and MAC
// reported by the three simulator devices are parsed correctly.
func TestExtractMetrics_DummyIPAddresses(t *testing.T) {
	cfg, err := LoadConfig(findConfigPath(t))
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	cases := []struct {
		manufacturer string
		productClass string
		serial       string
		rx           string
		tx           string
		path         string
		wanIP        string
		lanIP        string
		mac          string
	}{
		{"Huawei", "HG8245H", "HWTC12345678", "-19.40", "2.30", "X_GponInterfaceConfig",
			"10.10.9.101", "192.168.100.11", "AA:BB:CC:10:20:31"},
		{"ZTE", "F609", "ZTEG87654321", "-26.20", "1.80", "X_ZTE-COM_WANPONInterfaceConfig",
			"10.10.9.102", "192.168.100.12", "AA:BB:CC:20:30:42"},
		{"Fiberhome", "AN5506-04-F", "FH88889999", "-28.80", "2.10", "X_FH_PONInterfaceConfig",
			"10.10.9.103", "192.168.100.13", "AA:BB:CC:30:40:53"},
	}

	for _, c := range cases {
		doc := buildSimulatorDocWithIP(c.manufacturer, c.productClass, c.serial,
			c.rx, c.tx, c.path, c.wanIP, c.lanIP, c.mac)

		metrics := ExtractMetrics(doc, cfg)
		if metrics == nil {
			t.Fatalf("%s: expected non-nil metrics", c.serial)
		}

		// WAN IP must win over LAN IP
		if metrics.IPAddress != c.wanIP {
			t.Errorf("%s: IPAddress = %q, want %q (WAN IP)", c.serial, metrics.IPAddress, c.wanIP)
		}
		if metrics.MacAddress != c.mac {
			t.Errorf("%s: MacAddress = %q, want %q", c.serial, metrics.MacAddress, c.mac)
		}
	}
}

// TestExtractMetrics_FallsBackToLanIP ensures the LAN IP is used when no WAN IP exists.
func TestExtractMetrics_FallsBackToLanIP(t *testing.T) {
	cfg, err := LoadConfig(findConfigPath(t))
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	recent := time.Now().Add(-30 * time.Second).Format(time.RFC3339Nano)
	doc := map[string]interface{}{
		"_id":         "HWTC12345678",
		"_lastInform": recent,
		"_deviceId": map[string]interface{}{
			"_Manufacturer": "Huawei",
			"_ProductClass": "HG8245H",
			"_SerialNumber": "HWTC12345678",
		},
		"InternetGatewayDevice": map[string]interface{}{
			"LANDevice": map[string]interface{}{
				"1": map[string]interface{}{
					"LANHostConfigManagement": map[string]interface{}{
						"IPInterface": map[string]interface{}{
							"1": map[string]interface{}{
								"IPInterfaceIPAddress": map[string]interface{}{"_value": "192.168.100.11"},
							},
						},
					},
				},
			},
		},
	}

	metrics := ExtractMetrics(doc, cfg)
	if metrics == nil {
		t.Fatal("expected non-nil metrics")
	}
	if metrics.IPAddress != "192.168.100.11" {
		t.Errorf("IPAddress = %q, want LAN fallback 192.168.100.11", metrics.IPAddress)
	}
}

// TestExtractMetrics_RejectsInvalidIP guards against 0.0.0.0 placeholders.
func TestExtractMetrics_RejectsInvalidIP(t *testing.T) {
	if isValidIPv4("0.0.0.0") {
		t.Error("0.0.0.0 must be rejected as a usable IP")
	}
	if isValidIPv4("") {
		t.Error("empty string must be rejected")
	}
	if isValidIPv4("not-an-ip") {
		t.Error("garbage must be rejected")
	}
	if !isValidIPv4("10.10.9.101") {
		t.Error("10.10.9.101 must be accepted")
	}
}

// TestLookLikeMac validates MAC detection used to reject bogus values.
func TestLookLikeMac(t *testing.T) {
	if !looksLikeMac("AA:BB:CC:10:20:31") {
		t.Error("colon MAC should be accepted")
	}
	if !looksLikeMac("aa-bb-cc-10-20-31") {
		t.Error("dash MAC should be accepted")
	}
	if looksLikeMac("AA:BB:CC") {
		t.Error("partial MAC must be rejected")
	}
	if looksLikeMac("") {
		t.Error("empty must be rejected")
	}
}
