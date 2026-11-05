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

// Diagnostic: dumps the items table minimum_stock state.
// Run with: go run -tags minimum_stock ./scripts/check-minimum-stock.go
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		getEnv("DB_USER", "root"), getEnv("DB_PASSWORD", ""),
		getEnv("DB_HOST", "127.0.0.1"), getEnv("DB_PORT", "3306"),
		getEnv("DB_NAME", "iqgncnzy_skripsi"))

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()

	fmt.Println("items table columns:")
	rows, err := db.Query(`
		SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, IFNULL(COLUMN_DEFAULT,'NULL'), EXTRA
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'items'
		ORDER BY ORDINAL_POSITION
	`)
	if err != nil {
		log.Fatalf("describe: %v", err)
	}
	for rows.Next() {
		var n, t, nullable, def, extra string
		rows.Scan(&n, &t, &nullable, &def, &extra)
		fmt.Printf("  %-14s %-14s null=%-3s default=%-8s %s\n", n, t, nullable, def, extra)
	}
	rows.Close()

	fmt.Println("\nitems rows:")
	r2, err := db.Query("SELECT name, minimum_stock FROM items ORDER BY name")
	if err != nil {
		log.Fatalf("select: %v", err)
	}
	defer r2.Close()
	total := 0
	for r2.Next() {
		var name string
		var min int
		r2.Scan(&name, &min)
		fmt.Printf("  %-30s minimum_stock=%d\n", name, min)
		total++
	}
	fmt.Printf("\nTotal items: %d\n", total)
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
