//go:build minimum_stock

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/go-sql-driver/mysql"
)

// Migration script: adds `minimum_stock` to the `items` catalog table.
//
// Run with:
//   go run -tags minimum_stock ./scripts/migrate-minimum-stock.go
func main() {
	// Load .env from the project root (best effort)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	dbHost := getEnv("DB_HOST", "127.0.0.1")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "root")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "iqgncnzy_skripsi")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("✅ Database connection successful")
	fmt.Println("🔧 Adding minimum_stock column to items table...")

	// Step 1: Add column only if it does not exist yet
	// (MySQL has no ADD COLUMN IF NOT EXISTS, hence the INFORMATION_SCHEMA check)
	existing, err := columnExists(db, dbName, "items", "minimum_stock")
	if err != nil {
		log.Fatalf("Failed to inspect items table: %v", err)
	}

	if existing {
		fmt.Println("ℹ️  Column minimum_stock already exists, skipping ALTER TABLE")
	} else {
		alterQuery := "ALTER TABLE `items` ADD COLUMN `minimum_stock` INTEGER NOT NULL DEFAULT 0 AFTER `asset_id`"
		if _, err := db.Exec(alterQuery); err != nil {
			log.Fatalf("Failed to add minimum_stock column: %v", err)
		}
		fmt.Println("✅ Column minimum_stock added to items table")
	}

	// Step 2: Seed the test data
	res, err := db.Exec("UPDATE `items` SET `minimum_stock` = 5 WHERE `name` LIKE ?", "%Test Min Stock A%")
	if err != nil {
		log.Fatalf("Failed to update minimum_stock for test items: %v", err)
	}
	affected, _ := res.RowsAffected()
	fmt.Printf("✅ Updated minimum_stock = 5 for %d item(s) matching '%%Test Min Stock A%%'\n", affected)

	// Step 3: Verification
	fmt.Println("\n🔍 Verification (items matching 'Test Min Stock A'):")
	rows, err := db.Query("SELECT `id`, `name`, `minimum_stock` FROM `items` WHERE `name` LIKE ?", "%Test Min Stock A%")
	if err != nil {
		log.Fatalf("Failed to query items: %v", err)
	}
	defer rows.Close()

	found := 0
	for rows.Next() {
		var id, name string
		var minimumStock int
		if err := rows.Scan(&id, &name, &minimumStock); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		fmt.Printf("  - %s | %s | minimum_stock=%d\n", id, name, minimumStock)
		found++
	}
	if found == 0 {
		fmt.Println("  (no items matched 'Test Min Stock A')")
	}

	// Also report the current column definition
	var columnType, isNullable, columnDefault sql.NullString
	colErr := db.QueryRow(`
		SELECT COLUMN_TYPE, IS_NULLABLE, COLUMN_DEFAULT
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = 'items' AND COLUMN_NAME = 'minimum_stock'
	`, dbName).Scan(&columnType, &isNullable, &columnDefault)
	if colErr != nil {
		log.Printf("Could not read column definition: %v", colErr)
	} else {
		fmt.Printf("\n📋 Column definition: minimum_stock %s, nullable=%s, default=%s\n",
			columnType.String, isNullable.String, columnDefault.String)
	}

	fmt.Println("\n🎉 Migration finished!")
}

func columnExists(db *sql.DB, schema, table, column string) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND COLUMN_NAME = ?
	`, schema, table, column).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
