package customer

import (
	"database/sql"
	"testing"

	"skripsi-be/internal/models/entities"
)

func strPtr(v string) *string { return &v }

// TestBuildRouterSummary_PrefersAssetItemSerial ensures the physical unit serial
// (asset_items.serial_number) wins over the asset catalog serial (assets.serial_number).
func TestBuildRouterSummary_PrefersAssetItemSerial(t *testing.T) {
	device := entities.NetworkDevice{
		ID:          "nd-1",
		AssetItemID: strPtr("ai-1"),
		AssetsID:    sql.NullString{String: "asset-1", Valid: true},
		Assets: &entities.Asset{
			ID:           "asset-1",
			Brand:        "Huawei",
			Type:         "ONT",
			Model:        "HG8245H",
			SerialNumber: "CATALOG-SERIAL",
		},
		AssetItem: &entities.AssetItem{
			ID:           "ai-1",
			MacAddress:   "AA:BB:CC:DD:EE:FF",
			SerialNumber: strPtr("REAL-UNIT-SERIAL"),
			Status:       "in_use",
		},
	}

	got := buildRouterSummary(device)
	if got == nil {
		t.Fatal("expected non-nil router summary")
	}
	if got.SerialNumber != "REAL-UNIT-SERIAL" {
		t.Errorf("serial number = %q, want %q (physical unit must win)", got.SerialNumber, "REAL-UNIT-SERIAL")
	}
	if got.Brand != "Huawei" || got.Type != "ONT" || got.Model != "HG8245H" {
		t.Errorf("catalog info mismatch: %+v", got)
	}
	if got.MacAddress != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("mac address = %q, want AA:BB:CC:DD:EE:FF", got.MacAddress)
	}
	if got.AssetItemStatus != "in_use" {
		t.Errorf("asset item status = %q, want in_use", got.AssetItemStatus)
	}
}

// TestBuildRouterSummary_FallsBackToCatalogWhenNoAssetItem verifies graceful
// degradation for legacy rows that only have the catalog relation populated.
func TestBuildRouterSummary_FallsBackToCatalogWhenNoAssetItem(t *testing.T) {
	device := entities.NetworkDevice{
		ID:       "nd-legacy",
		AssetsID: sql.NullString{String: "asset-1", Valid: true},
		Assets: &entities.Asset{
			ID:           "asset-1",
			Brand:        "ZTE",
			Type:         "ONT",
			Model:        "F609",
			SerialNumber: "LEGACY-SERIAL",
		},
	}

	got := buildRouterSummary(device)
	if got == nil {
		t.Fatal("expected non-nil router summary for legacy device")
	}
	if got.SerialNumber != "LEGACY-SERIAL" {
		t.Errorf("serial number = %q, want LEGACY-SERIAL fallback", got.SerialNumber)
	}
	if got.Brand != "ZTE" {
		t.Errorf("brand = %q, want ZTE", got.Brand)
	}
}

// TestBuildRouterSummary_UsesOntSerialWhenAssetHasNoSerial covers the intermediate fallback.
func TestBuildRouterSummary_UsesOntSerialWhenAssetHasNoSerial(t *testing.T) {
	device := entities.NetworkDevice{
		ID:              "nd-2",
		AssetItemID:     strPtr("ai-2"),
		OntSerialNumber: strPtr("ONT-SN-002"),
		AssetItem: &entities.AssetItem{
			ID:         "ai-2",
			MacAddress: "11:22:33:44:55:66",
			Status:     "in_use",
		},
	}

	got := buildRouterSummary(device)
	if got == nil {
		t.Fatal("expected non-nil router summary")
	}
	if got.SerialNumber != "ONT-SN-002" {
		t.Errorf("serial number = %q, want ONT-SN-002", got.SerialNumber)
	}
	if got.OntSerialNumber != "ONT-SN-002" {
		t.Errorf("ont serial number = %q, want ONT-SN-002", got.OntSerialNumber)
	}
}

// TestBuildRouterSummary_ReturnsNilWhenNoAssetInfo ensures we do not emit empty
// router summaries that would confuse the UI.
func TestBuildRouterSummary_ReturnsNilWhenNoAssetInfo(t *testing.T) {
	device := entities.NetworkDevice{ID: "nd-empty"}

	if got := buildRouterSummary(device); got != nil {
		t.Errorf("expected nil summary for device without asset info, got %+v", got)
	}
}

// TestBuildRouterSummary_FallsBackToAssetItemAsset verifies catalog data is still
// resolved through asset_item.asset when the network_device -> assets link is missing.
func TestBuildRouterSummary_FallsBackToAssetItemAsset(t *testing.T) {
	device := entities.NetworkDevice{
		ID:          "nd-3",
		AssetItemID: strPtr("ai-3"),
		AssetsID:    sql.NullString{String: "", Valid: false},
		AssetItem: &entities.AssetItem{
			ID:           "ai-3",
			MacAddress:   "DE:AD:BE:EF:00:01",
			SerialNumber: strPtr("SN-AI-3"),
			Status:       "in_use",
			Asset: &entities.Asset{
				Brand: "Fiberhome",
				Type:  "ONT",
				Model: "AN5506-04-F",
			},
		},
	}

	got := buildRouterSummary(device)
	if got == nil {
		t.Fatal("expected non-nil router summary")
	}
	if got.Brand != "Fiberhome" || got.Model != "AN5506-04-F" {
		t.Errorf("expected catalog fallback from AssetItem.Asset, got %+v", got)
	}
	if got.SerialNumber != "SN-AI-3" {
		t.Errorf("serial number = %q, want SN-AI-3", got.SerialNumber)
	}
}
