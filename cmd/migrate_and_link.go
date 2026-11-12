//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"

	"skripsi-be/internal/config/database"
	"skripsi-be/internal/models/entities"
	"skripsi-be/internal/services"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")
	db := database.GetDB()
	if db == nil {
		log.Fatal("Could not connect to database")
	}

	fmt.Println("Running AutoMigrate...")
	database.AutoMigrate()

	// Ambil 3 pelanggan pertama yang memiliki koordinat
	var customers []entities.Customer
	if err := db.Where("deleted_at IS NULL AND latitude != 0 AND longitude != 0").
		Order("id ASC").
		Limit(3).
		Find(&customers).Error; err != nil {
		log.Fatalf("Error finding customers: %v", err)
	}

	dummySerials := []string{
		"HWTC12345678", // Huawei (Normal)
		"ZTEG87654321", // ZTE (Warning)
		"FH88889999",   // Fiberhome (Critical)
	}

	fmt.Printf("Found %d customers to link with dummy ONT serials:\n", len(customers))
	for i, c := range customers {
		sn := dummySerials[i]
		if err := db.Model(&entities.Customer{}).Where("id = ?", c.ID).Update("ont_serial_number", sn).Error; err != nil {
			log.Printf("Failed to update customer %s: %v", c.Name, err)
		} else {
			fmt.Printf("  -> Customer: %-25s (ID: %s) => ONT Serial: %s\n", c.Name, c.ID, sn)
		}
	}

	// Inisialisasi poller dan jalankan 1 siklus poll
	fmt.Println("\nTesting OntPollerService.PollOnce()...")
	poller := services.InitSharedOntPoller(db)
	if err := poller.PollOnce(context.Background()); err != nil {
		log.Printf("PollOnce notice: %v", err)
	} else {
		fmt.Println("PollOnce succeeded!")
	}

	// Cek isi tabel ont_statuses
	var statuses []entities.OntStatus
	if err := db.Find(&statuses).Error; err == nil {
		fmt.Printf("\nIsi tabel ont_statuses (%d records):\n", len(statuses))
		for _, s := range statuses {
			rxStr := "nil"
			if s.RxPower != nil {
				rxStr = fmt.Sprintf("%.2f dBm", *s.RxPower)
			}
			fmt.Printf("  - CustomerID: %s | Serial: %-15s | Online: %-5v | Rx: %-10s | Status: %-8s | SSID: %s\n",
				s.CustomerID, s.SerialNumber, s.IsOnline, rxStr, s.OpticalStatus, s.SSID)
		}
	}
}

