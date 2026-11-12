//go:build ignore

// Seeder data DUMMY untuk monitoring pelanggan + topologi jaringan.
//
// Jalankan dari root CRMLillyISPBE:
//
//	go run cmd/seed_dummy_monitoring.go -dry-run   # pratinjau, tanpa DB
//	go run cmd/seed_dummy_monitoring.go            # tulis ke DB
//	go run cmd/seed_dummy_monitoring.go -clean     # hapus semua data [DEMO]
//
// Yang dibuat:
//   - 1 area [DEMO] Jember
//   - 60 pelanggan [DEMO] (tersebar di 6 ODP)
//   - 60 baris customer_network_links (pelanggan -> ODP + port OLT)
//   - 60 baris network_links bertipe 'drop' (ODP -> pelanggan, GeoJSON)
//   - 60 baris ont_statuses (Rx/Tx, online, serial ONU)
//   - 60 x 40 baris ont_status_histories (riwayat Rx untuk sparkline)
//   - 3 baris netwatch_devices (IP POP, penghubung ke network_links.netwatch_ip)
//
// SEMUA data pelanggan memakai prefix "[DEMO] " sehingga -clean hanya
// menghapus data demo dan TIDAK menyentuh pelanggan nyata.
//
// Prasyarat: jalankan cmd/seed_network_topology.go lebih dulu agar node
// SRV / POP-* / ODP-* sudah ada.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"time"

	"skripsi-be/internal/config/database"
	"skripsi-be/internal/models/entities"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

const (
	demoPrefix    = "[DEMO] "
	demoSeed      = 20261012
	customersXODP = 10 // 6 ODP x 10 = 60 pelanggan
	historyPoints = 40 // titik riwayat Rx (sesuai referensi demo)
)

// ---------------------------------------------------------------------------
// Referensi data
// ---------------------------------------------------------------------------

var odpCodes = []string{
	"ODP-SBR-01", "ODP-SBR-02",
	"ODP-PTG-01", "ODP-PTG-02",
	"ODP-KLW-01", "ODP-KLW-02",
}

// odpToPOP memetakan ODP ke POP di atasnya (mencari baris netwatch POP).
var odpToPOP = map[string]string{
	"ODP-SBR-01": "POP-SBR", "ODP-SBR-02": "POP-SBR",
	"ODP-PTG-01": "POP-PTG", "ODP-PTG-02": "POP-PTG",
	"ODP-KLW-01": "POP-KLW", "ODP-KLW-02": "POP-KLW",
}

var firstNames = []string{
	"Budi", "Siti", "Agus", "Dewi", "Rizky", "Ayu", "Hendra", "Fitri", "Joko", "Lestari",
	"Andi", "Nur", "Eko", "Rina", "Dimas", "Wulan", "Yusuf", "Putri", "Fajar", "Intan",
}

var lastNames = []string{
	"Santoso", "Wijaya", "Hidayat", "Rahma", "Pratama",
	"Saputra", "Utami", "Kurniawan", "Handayani", "Firmansyah",
}

var plans = []string{"10 Mbps", "20 Mbps", "30 Mbps", "50 Mbps"}

// ---------------------------------------------------------------------------
// Generator acak deterministik (mulberry32, sama seperti referensi demo)
// ---------------------------------------------------------------------------

func mulberry32(a uint32) func() float64 {
	return func() float64 {
		a += 0x6D2B79F5
		t := a
		t = (t ^ (t >> 15)) * (t | 1)
		t ^= t + (t^(t>>7))*(t|61)
		return float64((t^(t>>14))>>0) / 4294967296.0
	}
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func round6(v float64) float64 {
	return math.Round(v*1e6) / 1e6
}

// ---------------------------------------------------------------------------
// Model dummy (generator, sebelum ditulis ke DB)
// ---------------------------------------------------------------------------

type demoCustomer struct {
	CustomerID string
	Name       string
	Alias      string
	Phone      string
	Address    string
	Lat, Lng   float64
	ODPCode    string
	Plan       string
	ONUSerial  string
	OLTPort    string

	// State simulasi
	OrigBase float64
	Base     float64
	Rx       float64
	OwnUp    bool
	History  []*float64
	LastUp   *time.Time
}

type nodeGeo struct {
	ID   string
	Code string
	Lat  float64
	Lng  float64
}

type latLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// bendPathForDrop membuat polyline melengkung ODP -> pelanggan.
func bendPathForDrop(aLat, aLng, bLat, bLng float64) []latLng {
	const count = 1
	const amp = 0.02

	pts := []latLng{{Lat: aLat, Lng: aLng}}
	dx := bLng - aLng
	dy := bLat - aLat
	for i := 1; i <= count; i++ {
		t := float64(i) / float64(count+1)
		s := amp
		if i%2 == 0 {
			s = -amp
		}
		pts = append(pts, latLng{Lat: aLat + dy*t - dx*s, Lng: aLng + dx*t + dy*s})
	}
	pts = append(pts, latLng{Lat: bLat, Lng: bLng})
	return pts
}

func haversineM(aLat, aLng, bLat, bLng float64) float64 {
	const earthRadiusM = 6371000.0
	toRad := func(d float64) float64 { return d * math.Pi / 180.0 }
	dLat := toRad(bLat - aLat)
	dLng := toRad(bLng - aLng)
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(aLat))*math.Cos(toRad(bLat))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusM * math.Asin(math.Sqrt(h))
}

func pathLengthM(pts []latLng) float64 {
	total := 0.0
	for i := 1; i < len(pts); i++ {
		total += haversineM(pts[i-1].Lat, pts[i-1].Lng, pts[i].Lat, pts[i].Lng)
	}
	return total
}

// opticalStatusFromRx mencerminkan ambang di config/ont-models.yaml.
func opticalStatusFromRx(rx *float64) string {
	if rx == nil {
		return "unknown"
	}
	v := *rx
	switch {
	case v > -8:
		return "critical" // overload optik
	case v >= -25:
		return "normal"
	case v >= -27:
		return "warning"
	default:
		return "critical"
	}
}

// ---------------------------------------------------------------------------
// Dataset builder
// ---------------------------------------------------------------------------

func buildDataset() []demoCustomer {
	rnd := mulberry32(demoSeed)
	out := make([]demoCustomer, 0, len(odpCodes)*customersXODP)

	idx := 0
	for _, odp := range odpCodes {
		for k := 0; k < customersXODP; k++ {
			i := idx
			idx++

			base := -(17 + rnd()*8.5) // -17 .. -25.5 dBm

			out = append(out, demoCustomer{
				CustomerID: uuid.New().String(),
				Name:       demoPrefix + firstNames[i%len(firstNames)] + " " + lastNames[(i*7+(i/20)*3)%len(lastNames)],
				Alias:      fmt.Sprintf("DEMO-%03d", i+1),
				Phone:      fmt.Sprintf("0812%08d", 30000000+i*137),
				Address:    fmt.Sprintf("Jl. Demo No. %d, Jember (data dummy)", i+1),
				ODPCode:    odp,
				Plan:       plans[i%len(plans)],
				ONUSerial:  fmt.Sprintf("ZTEG%08X", 0xA1B2C300+i*7919),
				OLTPort:    fmt.Sprintf("gpon-onu_1/%d/%d:%d", 1+(i%2), 1+(k%8), k+1),
				OrigBase:   base,
				Base:       base,
				Rx:         base,
				OwnUp:      true,
			})
		}
	}

	// Kasus khusus agar demo menarik (index sama dengan referensi).
	for _, i := range []int{7, 23, 41} {
		if i < len(out) {
			v := -26.1 - rnd()*0.6 // warning
			out[i].OrigBase, out[i].Base, out[i].Rx = v, v, v
		}
	}
	for _, i := range []int{13, 50} {
		if i < len(out) {
			v := -28.2 - rnd()*0.8 // critical
			out[i].OrigBase, out[i].Base, out[i].Rx = v, v, v
		}
	}
	for _, i := range []int{5, 33, 47} {
		if i < len(out) {
			out[i].OwnUp = false // ONU LOS
		}
	}

	// Bangun riwayat Rx awal (40 titik) per pelanggan.
	now := time.Now()
	for ci := range out {
		c := &out[ci]
		hist := make([]*float64, 0, historyPoints)
		for p := 0; p < historyPoints; p++ {
			c.Rx = clamp(c.Rx+(c.Base-c.Rx)*0.25+(rnd()-0.5)*0.5, -32, -14)
			if !c.OwnUp {
				hist = append(hist, nil)
				continue
			}
			v := math.Round(c.Rx*100) / 100
			hist = append(hist, &v)
		}
		c.History = hist
		if c.OwnUp {
			t := now.Add(-time.Duration(rnd()*300) * time.Second)
			c.LastUp = &t
		}
	}

	return out
}

func printPlan(dataset []demoCustomer) {
	fmt.Println("=== DRY RUN seeder data DEMO (tidak menulis ke DB) ===")
	fmt.Printf("\nArea      : %sJember (dummy)\n", demoPrefix)
	fmt.Printf("Pelanggan : %d (prefix %q)\n", len(dataset), demoPrefix)

	counts := map[string]int{"normal": 0, "warning": 0, "critical": 0, "down": 0}
	for _, c := range dataset {
		switch {
		case !c.OwnUp:
			counts["down"]++
		case c.Rx < -27:
			counts["critical"]++
		case c.Rx < -25:
			counts["warning"]++
		default:
			counts["normal"]++
		}
	}
	fmt.Printf("Status    : normal=%d warning=%d critical=%d down=%d\n",
		counts["normal"], counts["warning"], counts["critical"], counts["down"])

	fmt.Println("\nContoh 10 pelanggan:")
	for i, c := range dataset {
		if i >= 10 {
			break
		}
		state := "online"
		if !c.OwnUp {
			state = "DOWN (LOS)"
		}
		fmt.Printf("  %-28s %-12s Rx=%7.2f dBm  %-10s %s\n",
			c.Name, c.ODPCode, c.Rx, state, c.OLTPort)
	}
	fmt.Println("  ...")
	fmt.Printf("\nTotal: %d pelanggan, %d drop-link, %d ont_statuses, %d riwayat Rx\n",
		len(dataset), len(dataset), len(dataset), len(dataset)*historyPoints)
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	dryRun := flag.Bool("dry-run", false, "tampilkan rencana tanpa menulis ke database")
	clean := flag.Bool("clean", false, "hapus seluruh data pelanggan & monitoring [DEMO]")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("note: .env not loaded (using existing environment)")
	}

	// -dry-run tidak memerlukan koneksi database.
	if *dryRun {
		printPlan(buildDataset())
		return
	}

	db := database.GetDB()
	if db == nil {
		log.Fatal("tidak bisa terhubung ke database")
	}

	if !db.Migrator().HasTable(&entities.OntStatus{}) ||
		!db.Migrator().HasTable(&entities.OntStatusHistory{}) {
		log.Fatal("tabel ont_statuses / ont_status_histories belum ada — jalankan backend sekali agar AutoMigrate membuatnya")
	}

	if *clean {
		if err := cleanDemo(db); err != nil {
			log.Fatalf("gagal membersihkan data demo: %v", err)
		}
		fmt.Println("Data demo dihapus (pelanggan nyata tidak disentuh).")
		return
	}

	dataset := buildDataset()
	if err := insertDemo(db, dataset); err != nil {
		log.Fatalf("gagal menulis data demo: %v", err)
	}

	fmt.Printf("\nSelesai. %d pelanggan demo, %d drop-link, %d ont_statuses, %d riwayat Rx.\n",
		len(dataset), len(dataset), len(dataset), len(dataset)*historyPoints)
	fmt.Println("Hapus kapan saja dengan: go run cmd/seed_dummy_monitoring.go -clean")
}

// ---------------------------------------------------------------------------
// Helper DB
// ---------------------------------------------------------------------------

// loadNodes membaca node dari DB (ODP + POP) untuk koordinat & netwatch.
func loadNodes(db *gorm.DB) (odp map[string]nodeGeo, popNetwatch map[string]string, err error) {
	var nodes []entities.NetworkNode
	if e := db.Find(&nodes).Error; e != nil {
		return nil, nil, e
	}
	if len(nodes) == 0 {
		return nil, nil, errors.New("tabel network_nodes kosong — jalankan cmd/seed_network_topology.go dulu")
	}

	odp = map[string]nodeGeo{}
	codeByID := map[string]string{}
	for _, n := range nodes {
		codeByID[n.ID] = n.Code
		if n.Type == entities.NodeTypeODP {
			odp[n.Code] = nodeGeo{ID: n.ID, Code: n.Code, Lat: n.Latitude, Lng: n.Longitude}
		}
	}

	// netwatch_ip per POP diambil dari network_links bertipe backbone.
	var links []entities.NetworkLink
	if e := db.Where("type = ?", entities.LinkTypeBackbone).Find(&links).Error; e != nil {
		return nil, nil, e
	}
	popNetwatch = map[string]string{}
	for _, l := range links {
		if l.ToNodeID != nil && l.NetwatchIP != nil {
			popNetwatch[codeByID[*l.ToNodeID]] = *l.NetwatchIP
		}
	}

	return odp, popNetwatch, nil
}

// cleanDemo menghapus seluruh data demo (prefix [DEMO]).
func cleanDemo(db *gorm.DB) error {
	return db.Transaction(cleanDemoTx)
}

func cleanDemoTx(tx *gorm.DB) error {
	var ids []string
	if err := tx.Model(&entities.Customer{}).
		Where("name LIKE ?", demoPrefix+"%").
		Pluck("id", &ids).Error; err != nil {
		return err
	}

	if len(ids) > 0 {
		if err := tx.Where("customer_id IN ?", ids).Delete(&entities.OntStatusHistory{}).Error; err != nil {
			return err
		}
		if err := tx.Where("customer_id IN ?", ids).Delete(&entities.OntStatus{}).Error; err != nil {
			return err
		}
		if err := tx.Where("customer_id IN ?", ids).Delete(&entities.CustomerNetworkLink{}).Error; err != nil {
			return err
		}
		if err := tx.Where("to_customer_id IN ?", ids).Delete(&entities.NetworkLink{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", ids).Delete(&entities.Customer{}).Error; err != nil {
			return err
		}
	}

	if err := tx.Where("name LIKE ?", demoPrefix+"Netwatch%").Delete(&entities.NetwatchDevice{}).Error; err != nil {
		return err
	}
	if err := tx.Where("name_city = ?", demoPrefix+"Jember").Delete(&entities.Areas{}).Error; err != nil {
		return err
	}

	return nil
}

func insertDemo(db *gorm.DB, dataset []demoCustomer) error {
	odpByCode, popNetwatch, err := loadNodes(db)
	if err != nil {
		return err
	}

	for _, code := range odpCodes {
		if _, ok := odpByCode[code]; !ok {
			return fmt.Errorf("ODP %q tidak ditemukan di network_nodes", code)
		}
	}

	// Area dummy dibuat DI DALAM transaksi, setelah cleanDemoTx, agar tidak
	// ikut terhapus oleh pembersihan data demo sebelumnya.

	rnd := mulberry32(demoSeed + 1)
	now := time.Now()

	return db.Transaction(func(tx *gorm.DB) error {
		// Idempoten: bersihkan data demo sebelumnya.
		if err := cleanDemoTx(tx); err != nil {
			return err
		}

		// Area dummy (idempoten berdasarkan name_city).
		area := entities.Areas{
			ID:              uuid.New().String(),
			NameCity:        demoPrefix + "Jember",
			NameSubdistrict: "Sumbersari",
			NameVillage:     "Demo",
			CodeName:        "DEMO-JBR",
		}
		var existingArea entities.Areas
		if e := tx.Where("name_city = ?", area.NameCity).First(&existingArea).Error; e == nil {
			area.ID = existingArea.ID
		} else if e := tx.Create(&area).Error; e != nil {
			return fmt.Errorf("gagal membuat area demo: %w", e)
		}

		for _, c := range dataset {
			installDate := now.AddDate(0, -3, 0)
			requestDate := installDate.AddDate(0, 0, -7)
			lat := round6(odpByCode[c.ODPCode].Lat + (rnd()-0.5)*0.009)
			lng := round6(odpByCode[c.ODPCode].Lng + (rnd()-0.5)*0.010)

			// Insert via Exec: kolom FK (sales_representative_id, company_id)
			// bertipe string pada entity tetapi harus NULL di DB.
			if e := tx.Exec(`
				INSERT INTO customer
					(id, name, alias, phone, address, area_id, latitude, longitude,
					 password, is_internet, is_collaborator, ont_serial_number,
					 installation_date, service_request_date, createdAt, updatedAt)
				VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				c.CustomerID, c.Name, c.Alias, c.Phone, c.Address, area.ID, lat, lng,
				"demo", "yes", "no", c.ONUSerial,
				installDate, requestDate, now, now,
			).Error; e != nil {
				return fmt.Errorf("gagal membuat customer %s: %w", c.Name, e)
			}

			// Pelanggan -> ODP + port OLT.
			cnl := entities.CustomerNetworkLink{
				CustomerID: c.CustomerID,
				ODPNodeID:  odpByCode[c.ODPCode].ID,
				OLTPort:    strPtr(c.OLTPort),
			}
			if e := tx.Create(&cnl).Error; e != nil {
				return fmt.Errorf("gagal membuat customer_network_link: %w", e)
			}

			// Kabel drop ODP -> pelanggan (GeoJSON).
			pts := bendPathForDrop(
				odpByCode[c.ODPCode].Lat, odpByCode[c.ODPCode].Lng,
				lat, lng,
			)
			geom, e := json.Marshal(pts)
			if e != nil {
				return e
			}
			length := pathLengthM(pts)
			drop := entities.NetworkLink{
				Code:         "DROP-" + c.Alias,
				Name:         "Drop " + c.ODPCode + " ke " + c.Name,
				Type:         entities.LinkTypeDrop,
				FromNodeID:   strPtr(odpByCode[c.ODPCode].ID),
				ToCustomerID: strPtr(c.CustomerID),
				Geometry:     geom,
				LengthM:      &length,
				Status:       entities.LinkStatusUp,
			}
			if !c.OwnUp {
				drop.Status = entities.LinkStatusDown
			}
			if ip, ok := popNetwatch[odpToPOP[c.ODPCode]]; ok {
				drop.NetwatchIP = strPtr(ip)
			}
			if e := tx.Create(&drop).Error; e != nil {
				return fmt.Errorf("gagal membuat drop link: %w", e)
			}

			// ont_statuses (status terkini).
			var rx, txPower *float64
			if c.OwnUp {
				v := math.Round(c.Rx*100) / 100
				rx = &v
				t := math.Round((c.Rx+21.7)*100) / 100
				txPower = &t
			}
			ont := entities.OntStatus{
				CustomerID:    c.CustomerID,
				SerialNumber:  c.ONUSerial,
				Manufacturer:  "ZTE",
				ModelName:     "F609",
				IsOnline:      c.OwnUp,
				RxPower:       rx,
				TxPower:       txPower,
				OpticalStatus: opticalStatusFromRx(rx),
				SSID:          "LillyNet_" + c.Alias,
				ClientCount:   int(rnd() * 6),
				LastInform:    c.LastUp,
				IsStale:       !c.OwnUp,
			}
			if e := tx.Create(&ont).Error; e != nil {
				return fmt.Errorf("gagal membuat ont_status: %w", e)
			}

			// ont_status_histories (40 titik untuk sparkline).
			for p, rxv := range c.History {
				ts := now.Add(-time.Duration((len(c.History)-p)*60) * time.Second)
				h := entities.OntStatusHistory{
					CustomerID:    c.CustomerID,
					SerialNumber:  c.ONUSerial,
					IsOnline:      rxv != nil,
					RxPower:       rxv,
					OpticalStatus: opticalStatusFromRx(rxv),
					RecordedAt:    ts,
				}
				if e := tx.Create(&h).Error; e != nil {
					return fmt.Errorf("gagal membuat ont history: %w", e)
				}
			}
		}

		// netwatch_devices untuk POP (penghubung ke MikroTik).
		for popCode, ip := range popNetwatch {
			var existing entities.NetwatchDevice
			if e := tx.Where("ip_address = ?", ip).First(&existing).Error; e == nil {
				continue
			}
			nw := entities.NetwatchDevice{
				Name:      demoPrefix + "Netwatch " + popCode,
				IPAddress: ip,
				Status:    "up",
				LastSeen:  now,
			}
			if e := tx.Create(&nw).Error; e != nil {
				return fmt.Errorf("gagal membuat netwatch %s: %w", popCode, e)
			}
		}

		return nil
	})
}
