package topology

import (
	"testing"
)

func f(v float64) *float64 { return &v }

func th() Thresholds { return DefaultThresholds() }

// onlineRx membuat OnuState online dengan redaman tertentu.
func onlineRx(rx float64) OnuState {
	return OnuState{OwnUp: true, Rx: f(rx), Source: SourceGenieACS}
}

func upLink(src Source) *LinkState { return &LinkState{Up: true, Source: src} }

func downLink(src Source) *LinkState {
	return &LinkState{Up: false, Source: src}
}

// ---------------------------------------------------------------------------
// 1. Klasifikasi redaman & batas -25 / -27
// ---------------------------------------------------------------------------

func TestResolve_OpticalBoundaries(t *testing.T) {
	cases := []struct {
		name string
		rx   float64
		want Category
	}{
		{"normal kuat", -19.50, CategoryUp},
		{"normal tepat di WarnMin", -25.00, CategoryUp}, // -25 masih normal
		{"warning tepat di bawah WarnMin", -25.01, CategoryWarn},
		{"warning di tengah", -26.00, CategoryWarn},
		{"warning tepat di BadMin", -27.00, CategoryWarn}, // -27 masih warning
		{"bad tepat di bawah BadMin", -27.01, CategoryBad},
		{"bad dalam", -28.40, CategoryBad},
		{"bad sangat rendah", -31.00, CategoryBad},
		{"overload di atas OverloadMax", -7.50, CategoryBad},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Resolve(onlineRx(tc.rx), Chain{}, th())
			if got.Category != tc.want {
				t.Errorf("Rx %.2f: category = %s, want %s (reason: %s)",
					tc.rx, got.Category, tc.want, got.Reason)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 2. ONU down (LOS) mengalahkan redaman
// ---------------------------------------------------------------------------

func TestResolve_OnuDownOverridesRedaman(t *testing.T) {
	onu := OnuState{OwnUp: false, Rx: f(-19.5), Source: SourceGenieACS}
	got := Resolve(onu, Chain{}, th())

	if got.Category != CategoryDown {
		t.Fatalf("category = %s, want down", got.Category)
	}
	if got.Reason != "ONU LOS / mati listrik" {
		t.Errorf("reason = %q", got.Reason)
	}
	if got.Rx != nil {
		t.Errorf("Rx harus nil saat down, didapat %v", *got.Rx)
	}
}

// ---------------------------------------------------------------------------
// 3. Rx nil TIDAK boleh dianggap bad
// ---------------------------------------------------------------------------

func TestResolve_RxNilIsNotBad(t *testing.T) {
	onu := OnuState{OwnUp: true, Rx: nil, Source: SourceMikroTik}
	got := Resolve(onu, Chain{}, th())

	if got.Category != CategoryUp {
		t.Fatalf("category = %s, want up (redaman belum terdata)", got.Category)
	}
	if got.Rx != nil {
		t.Errorf("Rx harus nil, didapat %v", *got.Rx)
	}
	if got.Reason != "Online (redaman belum terdata)" {
		t.Errorf("reason = %q", got.Reason)
	}
}

// ---------------------------------------------------------------------------
// 4. Link putus -> down, dengan reason menyebut penyebab paling hulu
// ---------------------------------------------------------------------------

func TestResolve_BackboneDownWinsOverDistribution(t *testing.T) {
	onu := onlineRx(-19.5)
	chain := Chain{
		Backbone:          downLink(SourceMikroTik),
		BackboneLabel:     "POP-SBR",
		Distribution:      downLink(SourceMikroTik),
		DistributionLabel: "ODP-SBR-01",
	}

	got := Resolve(onu, chain, th())

	if got.Category != CategoryDown {
		t.Fatalf("category = %s, want down", got.Category)
	}
	// Backbone dievaluasi lebih dulu: penyebab paling hulu yang dilaporkan.
	if got.Reason != "Link backbone ke POP-SBR putus" {
		t.Errorf("reason = %q, want backbone first", got.Reason)
	}
	if got.Source != SourceMikroTik {
		t.Errorf("source = %s, want mikrotik", got.Source)
	}
}

func TestResolve_DistributionDown(t *testing.T) {
	chain := Chain{
		Backbone:          upLink(SourceMikroTik),
		Distribution:      downLink(SourceMikroTik),
		DistributionLabel: "ODP-PTG-01",
	}

	got := Resolve(onlineRx(-19.5), chain, th())

	if got.Category != CategoryDown {
		t.Fatalf("category = %s, want down", got.Category)
	}
	if got.Reason != "Link distribusi ke ODP-PTG-01 putus" {
		t.Errorf("reason = %q", got.Reason)
	}
}

func TestResolve_DropDown(t *testing.T) {
	chain := Chain{
		Backbone:     upLink(SourceMikroTik),
		Distribution: upLink(SourceMikroTik),
		Drop:         downLink(SourceMikroTik),
	}

	got := Resolve(onlineRx(-19.5), chain, th())

	if got.Category != CategoryDown {
		t.Fatalf("category = %s, want down", got.Category)
	}
	if got.Reason != "Kabel drop ke pelanggan putus" {
		t.Errorf("reason = %q", got.Reason)
	}
}

func TestResolve_LinkDownBeatsOnuDown(t *testing.T) {
	// ONU juga mati, tapi penyebab infrastruktur lebih informatif.
	onu := OnuState{OwnUp: false, Rx: nil, Source: SourceGenieACS}
	chain := Chain{Backbone: downLink(SourceMikroTik), BackboneLabel: "POP-KLW"}

	got := Resolve(onu, chain, th())

	if got.Category != CategoryDown {
		t.Fatalf("category = %s, want down", got.Category)
	}
	if got.Reason != "Link backbone ke POP-KLW putus" {
		t.Errorf("reason = %q, want backbone cause", got.Reason)
	}
}

func TestResolve_AllLinksUpStillEvaluatesOptics(t *testing.T) {
	chain := Chain{
		Backbone:     upLink(SourceMikroTik),
		Distribution: upLink(SourceMikroTik),
		Drop:         upLink(SourceMikroTik),
		DropLabel:    "DROP-DEMO-001",
	}

	got := Resolve(onlineRx(-28.4), chain, th())

	if got.Category != CategoryBad {
		t.Fatalf("category = %s, want bad", got.Category)
	}
}

// ---------------------------------------------------------------------------
// 5. Hops (jalur pelanggan -> server)
// ---------------------------------------------------------------------------

func TestResolve_HopsOrderAndStatus(t *testing.T) {
	onu := onlineRx(-19.5)
	chain := Chain{
		Drop:              upLink(SourceMikroTik),
		DropLabel:         "Kabel drop",
		Distribution:      downLink(SourceMikroTik),
		DistributionLabel: "Link distribusi ke ODP-SBR-01",
		Backbone:          upLink(SourceMikroTik),
		BackboneLabel:     "Link backbone ke POP-SBR",
		ServerName:        "Server NOC",
	}

	got := Resolve(onu, chain, th())

	if len(got.Hops) != 5 {
		t.Fatalf("jumlah hop = %d, want 5 (onu, drop, distribution, backbone, server)", len(got.Hops))
	}

	wantKinds := []string{"onu", "drop", "distribution", "backbone", "server"}
	for i, want := range wantKinds {
		if got.Hops[i].Kind != want {
			t.Errorf("hop[%d].Kind = %q, want %q", i, got.Hops[i].Kind, want)
		}
	}

	if !got.Hops[0].OK {
		t.Error("hop ONU harus OK")
	}
	if got.Hops[2].OK {
		t.Error("hop distribusi harus down")
	}
	if !got.Hops[3].OK {
		t.Error("hop backbone harus OK")
	}
	if !got.Hops[4].OK {
		t.Error("hop server harus OK")
	}
}

func TestResolve_HopsWithoutChain(t *testing.T) {
	got := Resolve(onlineRx(-19.5), Chain{}, th())

	// Tanpa topologi: hanya ONU + server.
	if len(got.Hops) != 2 {
		t.Fatalf("jumlah hop = %d, want 2", len(got.Hops))
	}
	if got.Hops[0].Kind != "onu" || got.Hops[1].Kind != "server" {
		t.Errorf("hop kinds = %q,%q", got.Hops[0].Kind, got.Hops[1].Kind)
	}
	if got.Hops[1].Label != "Server NOC" {
		t.Errorf("default server label = %q, want 'Server NOC'", got.Hops[1].Label)
	}
}

// ---------------------------------------------------------------------------
// 6. Source tracking & threshold validation
// ---------------------------------------------------------------------------

func TestResolve_SourcePropagates(t *testing.T) {
	onu := OnuState{OwnUp: true, Rx: f(-19.5), Source: SourceMock}
	if got := Resolve(onu, Chain{}, th()); got.Source != SourceMock {
		t.Errorf("source = %s, want mock", got.Source)
	}

	// Source kosong dinormalkan menjadi unknown.
	onu2 := OnuState{OwnUp: true, Rx: f(-19.5)}
	if got := Resolve(onu2, Chain{}, th()); got.Source != SourceUnknown {
		t.Errorf("source = %s, want unknown", got.Source)
	}
}

func TestThresholds_Validate(t *testing.T) {
	if err := DefaultThresholds().Validate(); err != nil {
		t.Fatalf("default thresholds harus valid: %v", err)
	}

	bad := []Thresholds{
		{WarnMin: -25, BadMin: -20}, // BadMin > WarnMin
		{WarnMin: 25, BadMin: -27},  // WarnMin positif
		{WarnMin: -25, BadMin: 27},  // BadMin positif
	}
	for i, tc := range bad {
		if err := tc.Validate(); err == nil {
			t.Errorf("kasus %d harus gagal validasi: %+v", i, tc)
		}
	}
}

func TestDefaultThresholdsMatchesConfig(t *testing.T) {
	got := DefaultThresholds()
	// Harus sama dengan config/ont-models.yaml (normal_min -25, warning_min -27).
	if got.WarnMin != -25 {
		t.Errorf("WarnMin = %.1f, want -25", got.WarnMin)
	}
	if got.BadMin != -27 {
		t.Errorf("BadMin = %.1f, want -27", got.BadMin)
	}
	if got.OverloadMax != -8 {
		t.Errorf("OverloadMax = %.1f, want -8", got.OverloadMax)
	}
}

func TestLabelForCategory(t *testing.T) {
	cases := map[Category]string{
		CategoryUp:   "Online",
		CategoryWarn: "Redaman tinggi",
		CategoryBad:  "Redaman buruk",
		CategoryDown: "Down",
	}
	for cat, want := range cases {
		if got := LabelForCategory(cat); got != want {
			t.Errorf("LabelForCategory(%s) = %q, want %q", cat, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// 7. Skenario gabungan
// ---------------------------------------------------------------------------

func TestResolve_MixedPopulation(t *testing.T) {
	type tc struct {
		name  string
		onu   OnuState
		chain Chain
		want  Category
	}

	upL := func() *LinkState { return upLink(SourceMikroTik) }

	cases := []tc{
		{"normal", onlineRx(-19.5), Chain{Backbone: upL(), Distribution: upL()}, CategoryUp},
		{"warning", onlineRx(-26.0), Chain{Backbone: upL(), Distribution: upL()}, CategoryWarn},
		{"bad", onlineRx(-28.4), Chain{Backbone: upL(), Distribution: upL()}, CategoryBad},
		{"onu los", OnuState{OwnUp: false, Source: SourceGenieACS}, Chain{Backbone: upL()}, CategoryDown},
		{"backbone putus", onlineRx(-19.5), Chain{Backbone: downLink(SourceMikroTik), BackboneLabel: "POP-SBR"}, CategoryDown},
		{"tanpa redaman", OnuState{OwnUp: true, Rx: nil, Source: SourceMikroTik}, Chain{}, CategoryUp},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Resolve(c.onu, c.chain, th()); got.Category != c.want {
				t.Errorf("category = %s, want %s (reason: %s)", got.Category, c.want, got.Reason)
			}
		})
	}
}
