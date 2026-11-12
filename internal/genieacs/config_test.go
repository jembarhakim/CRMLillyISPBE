package genieacs

import (
	"os"
	"path/filepath"
	"testing"
)

// findConfigPath locates config/ont-models.yaml relative to the package directory
func findConfigPath(t *testing.T) string {
	t.Helper()

	candidates := []string{
		"../../../config/ont-models.yaml", // from internal/genieacs -> repo root
		"../../config/ont-models.yaml",
		"./config/ont-models.yaml",
	}

	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if _, err := os.Stat(abs); err == nil {
			return abs
		}
	}

	t.Skip("config/ont-models.yaml not found, skipping config integration test")
	return ""
}

// TestLoadConfig_ReadsVendorProfiles is the regression test that was missing:
// it loads the REAL YAML file and asserts that vendor profiles and their
// parameters are actually decoded (previously Parameters was
// map[string]interface{} and the profile branch in ExtractMetrics never ran).
func TestLoadConfig_ReadsVendorProfiles(t *testing.T) {
	path := findConfigPath(t)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if len(cfg.Models) == 0 {
		t.Fatal("expected model profiles to be loaded from YAML, got none")
	}

	// Thresholds must come from the YAML file, not from defaults
	if cfg.Thresholds.OpticalRx.NormalMin != -25.0 {
		t.Errorf("thresholds.optical_rx.normal_min = %v, want -25.0", cfg.Thresholds.OpticalRx.NormalMin)
	}
	if cfg.Thresholds.DefaultInformInterval != 300 {
		t.Errorf("thresholds.default_inform_interval = %v, want 300", cfg.Thresholds.DefaultInformInterval)
	}

	// The three devices used by the GenieACS simulator must resolve to a profile
	expected := []string{"Huawei", "ZTE", "Fiberhome"}
	for _, vendor := range expected {
		profile := MatchModelProfile(vendor, "", cfg)
		if profile == nil {
			t.Errorf("no profile matched manufacturer %q", vendor)
			continue
		}
		if !profile.HasMetric("rx_power") {
			t.Errorf("profile %q has no rx_power parameter", profile.Key)
		}
	}

	huawei := cfg.FindModel("huawei_tr098")
	if huawei == nil {
		t.Fatal("huawei_tr098 profile not found")
	}
	rx := huawei.Parameter("rx_power")
	if rx.Path != "InternetGatewayDevice.WANDevice.1.X_GponInterfaceConfig.RXPower" {
		t.Errorf("huawei rx_power path = %q", rx.Path)
	}
	if rx.Scale != 1.0 {
		t.Errorf("huawei rx_power scale = %v, want 1.0", rx.Scale)
	}

	fiberhome := cfg.FindModel("fiberhome_tr181")
	if fiberhome == nil {
		t.Fatal("fiberhome_tr181 profile not found")
	}
	fiberhomeRx := fiberhome.Parameter("rx_power")
	if fiberhomeRx.Scale != 0.001 {
		t.Errorf("fiberhome rx_power scale = %v, want 0.001 (milli-dBm)", fiberhomeRx.Scale)
	}
}

// TestMatchModelProfile_DeterministicAndFallbackLast ensures a specific vendor
// profile is never shadowed by the catch-all generic fallback.
func TestMatchModelProfile_DeterministicAndFallbackLast(t *testing.T) {
	cfg, err := LoadConfig(findConfigPath(t))
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}

	cases := []struct {
		manufacturer string
		productClass string
		wantKey      string
	}{
		{"Huawei", "HG8245H", "huawei_tr098"},
		{"ZTE", "F609", "zte_tr098"},
		{"Fiberhome", "AN5506-04-F", "fiberhome_tr181"},
	}

	for _, c := range cases {
		got := MatchModelProfile(c.manufacturer, c.productClass, cfg)
		if got == nil {
			t.Errorf("%s: got nil profile", c.manufacturer)
			continue
		}
		if got.Key != c.wantKey {
			t.Errorf("%s %s: matched %q, want %q", c.manufacturer, c.productClass, got.Key, c.wantKey)
		}
	}

	// Repeated calls must always yield the same profile (no map ordering randomness)
	first := MatchModelProfile("Huawei", "HG8245H", cfg)
	for i := 0; i < 50; i++ {
		again := MatchModelProfile("Huawei", "HG8245H", cfg)
		if again == nil || first == nil || again.Key != first.Key {
			t.Fatalf("non-deterministic match on iteration %d", i)
		}
	}

	// Unknown vendor must fall back to the generic profile
	generic := MatchModelProfile("TotallyUnknownVendor", "X100", cfg)
	if generic == nil {
		t.Fatal("expected generic fallback profile for unknown vendor")
	}
	if generic.Key != "generic_fallback" {
		t.Errorf("unknown vendor matched %q, want generic_fallback", generic.Key)
	}
}
