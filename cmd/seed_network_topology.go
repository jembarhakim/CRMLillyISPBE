//go:build ignore

// Seeder infrastruktur jaringan (server -> POP -> ODP) + link fiber.
//
// Jalankan dari root CRMLillyISPBE:
//
//	go run cmd/seed_network_topology.go
//	go run cmd/seed_network_topology.go -dry-run
//	go run cmd/seed_network_topology.go -clean
//
// PENTING: seeder ini TIDAK membuat data pelanggan. Hanya:
//   - 1 node server (NOC)
//   - 3 node POP   (Sumbersari, Patrang, Kaliwates)
//   - 6 node ODP
//   - 3 link backbone   (server -> POP)
//   - 6 link distribusi (POP -> ODP)
//
// Link bertipe 'drop' (ODP -> pelanggan) diisi terpisah saat pelanggan nyata
// sudah dipetakan ke ODP lewat customer_network_links.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"

	"skripsi-be/internal/config/database"
	"skripsi-be/internal/models/entities"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Definisi topologi (koordinat area Jember, sama dengan referensi demo)
// ---------------------------------------------------------------------------

type nodeDef struct {
	Code     string
	Name     string
	Type     entities.NodeType
	Parent   string // kode parent ("" untuk server)
	Lat      float64
	Lng      float64
	RadiusM  int
	OLTHost  string
	OLTPort  string
	Capacity int
	// NetwatchIP: IP yang di-ping MikroTik untuk menentukan status node ini.
	// Nilai contoh dummy (10.255.x.x) — ganti dengan IP MikroTik Netwatch nyata.
	NetwatchIP string
}

var nodeDefs = []nodeDef{
	// --- Server / NOC -------------------------------------------------------
	{Code: "SRV", Name: "Server NOC", Type: entities.NodeTypeServer,
		Parent: "", Lat: -8.1722, Lng: 113.6995, RadiusM: 200},

	// --- POP ---------------------------------------------------------------
	{Code: "POP-SBR", Name: "POP Sumbersari", Type: entities.NodeTypePOP,
		Parent: "SRV", Lat: -8.1650, Lng: 113.7160, RadiusM: 200, Capacity: 1024,
		NetwatchIP: "10.255.0.11"},
	{Code: "POP-PTG", Name: "POP Patrang", Type: entities.NodeTypePOP,
		Parent: "SRV", Lat: -8.1330, Lng: 113.7060, RadiusM: 200, Capacity: 1024,
		NetwatchIP: "10.255.0.12"},
	{Code: "POP-KLW", Name: "POP Kaliwates", Type: entities.NodeTypePOP,
		Parent: "SRV", Lat: -8.2120, Lng: 113.6900, RadiusM: 200, Capacity: 1024,
		NetwatchIP: "10.255.0.13"},

	// --- ODP ---------------------------------------------------------------
	{Code: "ODP-SBR-01", Name: "ODP Sumbersari 01", Type: entities.NodeTypeODP,
		Parent: "POP-SBR", Lat: -8.1600, Lng: 113.7230, RadiusM: 100, Capacity: 64,
		OLTHost: "10.10.9.203", OLTPort: "gpon-onu_1/1/1"},
	{Code: "ODP-SBR-02", Name: "ODP Sumbersari 02", Type: entities.NodeTypeODP,
		Parent: "POP-SBR", Lat: -8.1710, Lng: 113.7110, RadiusM: 100, Capacity: 64,
		OLTHost: "10.10.9.203", OLTPort: "gpon-onu_1/1/2"},
	{Code: "ODP-PTG-01", Name: "ODP Patrang 01", Type: entities.NodeTypeODP,
		Parent: "POP-PTG", Lat: -8.1280, Lng: 113.7000, RadiusM: 100, Capacity: 64,
		OLTHost: "10.10.9.203", OLTPort: "gpon-onu_1/2/1"},
	{Code: "ODP-PTG-02", Name: "ODP Patrang 02", Type: entities.NodeTypeODP,
		Parent: "POP-PTG", Lat: -8.1390, Lng: 113.7130, RadiusM: 100, Capacity: 64,
		OLTHost: "10.10.9.203", OLTPort: "gpon-onu_1/2/2"},
	{Code: "ODP-KLW-01", Name: "ODP Kaliwates 01", Type: entities.NodeTypeODP,
		Parent: "POP-KLW", Lat: -8.2060, Lng: 113.6830, RadiusM: 100, Capacity: 64,
		OLTHost: "10.10.9.203", OLTPort: "gpon-onu_1/3/1"},
	{Code: "ODP-KLW-02", Name: "ODP Kaliwates 02", Type: entities.NodeTypeODP,
		Parent: "POP-KLW", Lat: -8.2200, Lng: 113.6960, RadiusM: 100, Capacity: 64,
		OLTHost: "10.10.9.203", OLTPort: "gpon-onu_1/3/2"},
}

// ---------------------------------------------------------------------------
// Geometri: kurva halus antar dua titik (dihitung di backend, disimpan sebagai JSON)
// ---------------------------------------------------------------------------

type latLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// bendPath membuat polyline melengkung dari a ke b dengan `count` titik sisipan
// dan amplitudo `amp` (derajat). Sama seperti referensi demo, tetapi dihitung
// sekali saat seeding sehingga frontend tinggal menggambar.
func bendPath(aLat, aLng, bLat, bLng float64, count int, amp float64) []latLng {
	pts := []latLng{{Lat: aLat, Lng: aLng}}
	dx := bLng - aLng
	dy := bLat - aLat

	for i := 1; i <= count; i++ {
		t := float64(i) / float64(count+1)
		s := amp
		if i%2 == 0 {
			s = -amp
		}
		pts = append(pts, latLng{
			Lat: aLat + dy*t - dx*s,
			Lng: aLng + dx*t + dy*s,
		})
	}

	pts = append(pts, latLng{Lat: bLat, Lng: bLng})
	return pts
}

// ---------------------------------------------------------------------------
// Utilitas
// ---------------------------------------------------------------------------

// haversineM menghitung jarak dua koordinat dalam meter (untuk length_m).
func haversineM(aLat, aLng, bLat, bLng float64) float64 {
	const earthRadiusM = 6371000.0
	toRad := func(d float64) float64 { return d * math.Pi / 180.0 }

	dLat := toRad(bLat - aLat)
	dLng := toRad(bLng - aLng)
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(aLat))*math.Cos(toRad(bLat))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusM * math.Asin(math.Sqrt(h))
}

// pathLengthM menjumlahkan panjang seluruh segmen polyline.
func pathLengthM(pts []latLng) float64 {
	total := 0.0
	for i := 1; i < len(pts); i++ {
		total += haversineM(pts[i-1].Lat, pts[i-1].Lng, pts[i].Lat, pts[i].Lng)
	}
	return total
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// cleanTopology menghapus data topologi dengan urutan aman terhadap foreign key.
func cleanTopology(db *gorm.DB) error {
	if err := db.Exec("DELETE FROM customer_network_links").Error; err != nil {
		return err
	}
	if err := db.Exec("DELETE FROM network_links").Error; err != nil {
		return err
	}
	return db.Exec("DELETE FROM network_nodes").Error
}

// printDryRun menampilkan rencana seeding tanpa menyentuh database.
func printDryRun() {
	nodeByCode := map[string]nodeDef{}
	for _, d := range nodeDefs {
		nodeByCode[d.Code] = d
	}

	fmt.Println("=== DRY RUN (tidak menulis ke database) ===")
	fmt.Printf("\nNode (%d):\n", len(nodeDefs))
	for _, d := range nodeDefs {
		parent := d.Parent
		if parent == "" {
			parent = "-"
		}
		fmt.Printf("  %-12s %-22s type=%-6s parent=%-8s (%.4f, %.4f) r=%dm\n",
			d.Code, d.Name, d.Type, parent, d.Lat, d.Lng, d.RadiusM)
	}

	linkDefs := buildLinkDefs()
	fmt.Printf("\nLink backbone + distribusi (%d):\n", len(linkDefs))
	for _, ld := range linkDefs {
		from := nodeByCode[ld.FromCode]
		to := nodeByCode[ld.ToCode]
		pts := bendPath(from.Lat, from.Lng, to.Lat, to.Lng, ld.BendCount, ld.BendAmp)
		fmt.Printf("  %-16s type=%-13s %-40s %6.0f m, %d titik\n",
			ld.Code, ld.Type, ld.Name, pathLengthM(pts), len(pts))
	}

	fmt.Printf("\nTotal: %d node, %d link\n", len(nodeDefs), len(linkDefs))
	fmt.Println("Data pelanggan tidak dibuat oleh seeder ini.")
}

// linkDef mendeskripsikan satu segmen fiber untuk seeding.
type linkDef struct {
	Code, Name, FromCode, ToCode string
	Type                         entities.LinkType
	BendCount                    int
	BendAmp                      float64
	NetwatchIP                   string
}

// buildLinkDefs mengembalikan daftar link yang akan di-seed.
func buildLinkDefs() []linkDef {
	return []linkDef{
		// Backbone: server -> POP
		{"BB-POP-SBR", "Backbone Server ke POP Sumbersari", "SRV", "POP-SBR", entities.LinkTypeBackbone, 2, 0.12, "10.255.0.11"},
		{"BB-POP-PTG", "Backbone Server ke POP Patrang", "SRV", "POP-PTG", entities.LinkTypeBackbone, 2, 0.12, "10.255.0.12"},
		{"BB-POP-KLW", "Backbone Server ke POP Kaliwates", "SRV", "POP-KLW", entities.LinkTypeBackbone, 2, 0.12, "10.255.0.13"},
		// Distribusi: POP -> ODP (netwatch dummy per ODP)
		{"DL-ODP-SBR-01", "Distribusi POP Sumbersari ke ODP-SBR-01", "POP-SBR", "ODP-SBR-01", entities.LinkTypeDistribution, 1, 0.10, "10.255.0.21"},
		{"DL-ODP-SBR-02", "Distribusi POP Sumbersari ke ODP-SBR-02", "POP-SBR", "ODP-SBR-02", entities.LinkTypeDistribution, 1, 0.10, "10.255.0.22"},
		{"DL-ODP-PTG-01", "Distribusi POP Patrang ke ODP-PTG-01", "POP-PTG", "ODP-PTG-01", entities.LinkTypeDistribution, 1, 0.10, "10.255.0.23"},
		{"DL-ODP-PTG-02", "Distribusi POP Patrang ke ODP-PTG-02", "POP-PTG", "ODP-PTG-02", entities.LinkTypeDistribution, 1, 0.10, "10.255.0.24"},
		{"DL-ODP-KLW-01", "Distribusi POP Kaliwates ke ODP-KLW-01", "POP-KLW", "ODP-KLW-01", entities.LinkTypeDistribution, 1, 0.10, "10.255.0.25"},
		{"DL-ODP-KLW-02", "Distribusi POP Kaliwates ke ODP-KLW-02", "POP-KLW", "ODP-KLW-02", entities.LinkTypeDistribution, 1, 0.10, "10.255.0.26"},
	}
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	dryRun := flag.Bool("dry-run", false, "tampilkan rencana tanpa menulis ke database")
	clean := flag.Bool("clean", false, "hapus seluruh data topologi lebih dulu")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("note: .env not loaded (using existing environment)")
	}

	// -dry-run tidak memerlukan koneksi database agar bisa dites tanpa MySQL.
	if *dryRun {
		printDryRun()
		return
	}

	db := database.GetDB()
	if db == nil {
		log.Fatal("tidak bisa terhubung ke database")
	}

	if *clean {
		fmt.Println("Menghapus data topologi lama (customer_network_links -> network_links -> network_nodes)...")
		if !*dryRun {
			if err := cleanTopology(db); err != nil {
				log.Fatalf("gagal membersihkan topologi: %v", err)
			}
		}
		fmt.Println("  selesai.")
		fmt.Println()
	}

	if !*dryRun {
		if err := db.AutoMigrate(&entities.NetworkNode{}, &entities.NetworkLink{}, &entities.CustomerNetworkLink{}); err != nil {
			log.Fatalf("AutoMigrate gagal: %v", err)
		}
	}

	// 1. Buat / perbarui node.
	codeToID := map[string]string{}
	for _, d := range nodeDefs {
		var existing entities.NetworkNode
		err := db.Where("code = ?", d.Code).First(&existing).Error

		node := entities.NetworkNode{
			Code:            d.Code,
			Name:            d.Name,
			Type:            d.Type,
			Latitude:        d.Lat,
			Longitude:       d.Lng,
			CoverageRadiusM: d.RadiusM,
			OLTHost:         strPtr(d.OLTHost),
			OLTPort:         strPtr(d.OLTPort),
			Capacity:        d.Capacity,
			Status:          entities.LinkStatusUp,
		}
		if d.Parent != "" {
			pid, ok := codeToID[d.Parent]
			if !ok {
				log.Fatalf("parent %q untuk %q belum dibuat (urutan nodeDefs salah?)", d.Parent, d.Code)
			}
			node.ParentID = &pid
		}

		switch {
		case err == nil:
			node.ID = existing.ID
			if e := db.Model(&existing).Updates(node).Error; e != nil {
				log.Fatalf("gagal update node %s: %v", d.Code, e)
			}
			codeToID[d.Code] = existing.ID
		default:
			if e := db.Create(&node).Error; e != nil {
				log.Fatalf("gagal membuat node %s: %v", d.Code, e)
			}
			codeToID[d.Code] = node.ID
		}
	}

	// 2. Buat link backbone & distribusi.
	linkDefs := buildLinkDefs()

	nodeByCode := map[string]nodeDef{}
	for _, d := range nodeDefs {
		nodeByCode[d.Code] = d
	}

	created, updated := 0, 0
	for _, ld := range linkDefs {
		from := nodeByCode[ld.FromCode]
		to := nodeByCode[ld.ToCode]

		pts := bendPath(from.Lat, from.Lng, to.Lat, to.Lng, ld.BendCount, ld.BendAmp)
		geom, err := json.Marshal(pts)
		if err != nil {
			log.Fatalf("gagal marshal geometry %s: %v", ld.Code, err)
		}
		length := pathLengthM(pts)

		var existing entities.NetworkLink
		err = db.Where("code = ?", ld.Code).First(&existing).Error

		link := entities.NetworkLink{
			Code:       ld.Code,
			Name:       ld.Name,
			Type:       ld.Type,
			FromNodeID: strPtr(codeToID[ld.FromCode]),
			ToNodeID:   strPtr(codeToID[ld.ToCode]),
			Geometry:   geom,
			LengthM:    &length,
			Status:     entities.LinkStatusUp,
			NetwatchIP: strPtr(ld.NetwatchIP),
		}

		switch {
		case err == nil:
			link.ID = existing.ID
			if e := db.Model(&existing).Updates(link).Error; e != nil {
				log.Fatalf("gagal update link %s: %v", ld.Code, e)
			}
			updated++
		default:
			if e := db.Create(&link).Error; e != nil {
				log.Fatalf("gagal membuat link %s: %v", ld.Code, e)
			}
			created++
		}
	}

	fmt.Printf("\nRingkasan: %d node, %d link dibuat, %d link diperbarui\n",
		len(nodeDefs), created, updated)
	fmt.Println("Link 'drop' (ODP -> pelanggan) dibuat otomatis saat pelanggan dipetakan ke ODP.")
}
