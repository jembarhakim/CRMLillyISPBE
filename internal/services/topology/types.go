// Package topology berisi logika penentuan status efektif pelanggan.
//
// Status efektif = status ONU pelanggan + status SEMUA link di atasnya
// (drop -> distribusi POP->ODP -> backbone server->POP).
//
// Ini satu-satunya sumber kebenaran untuk kategori status; frontend hanya
// menampilkan apa yang dikirim backend, sehingga tidak ada duplikasi logika.
package topology

import "time"

// Category adalah kategori status yang ditampilkan di peta.
type Category string

const (
	// CategoryUp — online dan redaman normal.
	CategoryUp Category = "up"
	// CategoryWarn — online tapi redaman mendekati batas (-25 s/d -27 dBm).
	CategoryWarn Category = "warn"
	// CategoryBad — online tapi redaman buruk (di bawah -27 dBm).
	CategoryBad Category = "bad"
	// CategoryDown — ONU LOS/mati, atau link di atasnya putus.
	CategoryDown Category = "down"
)

// Source menunjukkan dari mana data mentah berasal, agar UI bisa jujur
// menandai data nyata vs simulasi.
type Source string

const (
	SourceGenieACS Source = "genieacs"
	SourceMikroTik Source = "mikrotik"
	SourceMock     Source = "mock"
	SourceUnknown  Source = "unknown"
	SourceMixed    Source = "mixed"
)

// Thresholds adalah ambang redaman (dBm). Nilai negatif.
//
//	Rx >= WarnMin            -> normal      (mis. -19.5)
//	BadMin <= Rx < WarnMin   -> warning     (mis. -26.0)
//	Rx < BadMin              -> bad/critical (mis. -28.4)
//
// Default mengikuti config/ont-models.yaml: WarnMin=-25, BadMin=-27.
type Thresholds struct {
	WarnMin float64 // -25
	BadMin  float64 // -27
	// OverloadMax menandai sinyal terlalu kuat (mis. > -8 dBm) sebagai bad.
	OverloadMax float64 // -8
}

// DefaultThresholds mengembalikan ambang yang sama dengan config/ont-models.yaml.
func DefaultThresholds() Thresholds {
	return Thresholds{WarnMin: -25, BadMin: -27, OverloadMax: -8}
}

// OnuState adalah kondisi ONU milik satu pelanggan pada satu waktu.
type OnuState struct {
	// OwnUp false berarti ONU LOS / mati listrik (terlepas dari redaman).
	OwnUp bool
	// Rx nil berarti redaman belum terdata; TIDAK dianggap bad.
	Rx *float64
	Tx *float64
	// SerialNumber, Manufacturer, ModelName untuk ditampilkan di panel detail.
	SerialNumber string
	Manufacturer string
	ModelName    string
	// LastInform adalah waktu kontak terakhir ONU ke ACS.
	LastInform *time.Time
	// Source asal data ONU (genieacs / mock / unknown).
	Source Source
}

// LinkState adalah kondisi satu segmen fiber.
type LinkState struct {
	// Up false berarti segmen ini putus.
	Up bool
	// Source asal data link (mikrotik / mock / unknown).
	Source Source
}

// Hop adalah satu titik pada jalur pelanggan -> server, untuk panel detail.
type Hop struct {
	// Kind: "onu" | "drop" | "distribution" | "backbone" | "server"
	Kind   string `json:"kind"`
	Label  string `json:"label"`
	OK     bool   `json:"ok"`
	Source Source `json:"source"`
	// LinkCode diisi untuk hop bertipe link (drop/distribution/backbone).
	LinkCode string `json:"link_code,omitempty"`
}

// Result adalah keluaran resolver untuk satu pelanggan.
type Result struct {
	Category Category `json:"category"`
	// Reason adalah kalimat penjelas yang tampil di UI.
	Reason string `json:"reason"`
	// Rx diisi hanya bila redaman tersedia (nil bila down atau belum terdata).
	Rx *float64 `json:"rx"`
	// Source asal data yang menentukan status akhir.
	Source Source `json:"source"`
	// Hops adalah jalur pelanggan ke server (opsional, untuk endpoint detail).
	Hops []Hop `json:"hops,omitempty"`
}
