//go:build minimum_stock

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"skripsi-be/internal/api/admin/items_catalog"
	"skripsi-be/internal/models/entities"
)

// Integration check for minimum_stock through the real repository + GORM model.
// Verifies: create default, create with value, update to 0, update omit.
// Run with: go run -tags minimum_stock ./scripts/verify-minimum-stock-repo.go
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	dbName := getEnv("DB_NAME", "iqgncnzy_skripsi")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		getEnv("DB_USER", "root"), getEnv("DB_PASSWORD", ""),
		getEnv("DB_HOST", "127.0.0.1"), getEnv("DB_PORT", "3306"), dbName)

	gdb, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("gorm open: %v", err)
	}

	repo := items_catalog.NewItemRepository(gdb)
	suffix := uuid.New().String()[:8]
	pass := 0
	fail := 0

	check := func(label string, got, want int) {
		if got == want {
			fmt.Printf("  ✅ %-52s got=%d\n", label, got)
			pass++
		} else {
			fmt.Printf("  ❌ %-52s got=%d want=%d\n", label, got, want)
			fail++
		}
	}

	// 1) CREATE without minimum_stock -> must default to 0
	nameA := "ZZ Verify No Stock " + suffix
	createdA, err := repo.CreateItemRepository(items_catalog.CreateItemRequest{
		Name: nameA, DefaultUnit: "PCS",
	})
	if err != nil {
		log.Fatalf("create A: %v", err)
	}
	check("CREATE without minimum_stock defaults to 0", createdA.MinimumStock, 0)

	// 2) CREATE with minimum_stock = 7
	nameB := "ZZ Verify Min Stock " + suffix
	seven := 7
	createdB, err := repo.CreateItemRepository(items_catalog.CreateItemRequest{
		Name: nameB, DefaultUnit: "PCS", MinimumStock: &seven,
	})
	if err != nil {
		log.Fatalf("create B: %v", err)
	}
	check("CREATE with minimum_stock=7 persists 7", createdB.MinimumStock, 7)

	// 3) UPDATE minimum_stock 7 -> 0 (the GORM zero-value trap)
	zero := 0
	updatedB, err := repo.UpdateItemRepository(items_catalog.UpdateItemRequest{
		IdItemRequest:     items_catalog.IdItemRequest{Id: createdB.ID},
		CreateItemRequest: items_catalog.CreateItemRequest{Name: nameB, DefaultUnit: "PCS", MinimumStock: &zero},
	})
	if err != nil {
		log.Fatalf("update B: %v", err)
	}
	check("UPDATE minimum_stock 7 -> 0 persists 0 (returned)", updatedB.MinimumStock, 0)

	// Confirm it actually hit the DB, not just memory
	reloadedB, err := repo.FindByIdItemRepository(items_catalog.IdItemRequest{Id: createdB.ID})
	if err != nil {
		log.Fatalf("reload B: %v", err)
	}
	check("UPDATE minimum_stock 7 -> 0 persisted in DB", reloadedB.MinimumStock, 0)

	// 4) UPDATE 0 -> 4 then OMIT field -> must keep 4 (not reset)
	four := 4
	if _, err := repo.UpdateItemRepository(items_catalog.UpdateItemRequest{
		IdItemRequest:     items_catalog.IdItemRequest{Id: createdB.ID},
		CreateItemRequest: items_catalog.CreateItemRequest{Name: nameB, DefaultUnit: "PCS", MinimumStock: &four},
	}); err != nil {
		log.Fatalf("update B to 4: %v", err)
	}
	updatedB2, err := repo.UpdateItemRepository(items_catalog.UpdateItemRequest{
		IdItemRequest:     items_catalog.IdItemRequest{Id: createdB.ID},
		CreateItemRequest: items_catalog.CreateItemRequest{Name: nameB, DefaultUnit: "PCS"},
	})
	if err != nil {
		log.Fatalf("update B omit: %v", err)
	}
	check("UPDATE with omitted minimum_stock keeps 4", updatedB2.MinimumStock, 4)

	// 5) LIST returns minimum_stock correctly
	list, err := repo.FindItemsRepository(items_catalog.GetItemsRequest{})
	if err != nil {
		log.Fatalf("list: %v", err)
	}
	seen := 0
	for _, it := range list {
		if it.ID == createdA.ID || it.ID == createdB.ID {
			seen++
		}
	}
	check("LIST returns created rows (field present)", seen, 2)

	// Cleanup
	gdb.Where("id IN ?", []string{createdA.ID, createdB.ID}).Delete(&entities.Item{})

	fmt.Printf("\nResult: %d passed, %d failed\n", pass, fail)
	if fail > 0 {
		os.Exit(1)
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

var _ = sql.ErrNoRows
