//go:build minimum_stock

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/go-sql-driver/mysql"

	"skripsi-be/internal/helpers"
)

// End-to-end HTTP check of minimum_stock against the running backend.
// Covers the edge cases a positive-value UI test cannot reach:
// update -> 0, update with field omitted, and negative rejection.
//
// Run with: go run -tags minimum_stock ./scripts/verify-minimum-stock-http.go
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	baseURL := getEnv("APP_BASE_URL", "http://127.0.0.1:3001/api")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		getEnv("DB_USER", "root"), getEnv("DB_PASSWORD", ""),
		getEnv("DB_HOST", "127.0.0.1"), getEnv("DB_PORT", "3306"),
		getEnv("DB_NAME", "iqgncnzy_skripsi"))

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	token, err := helpers.CreateToken("verify-script", "admin")
	if err != nil {
		log.Fatalf("create token: %v", err)
	}

	pass, fail := 0, 0
	check := func(label string, got, want int) {
		if got == want {
			fmt.Printf("  ✅ %-56s got=%d\n", label, got)
			pass++
		} else {
			fmt.Printf("  ❌ %-56s got=%d want=%d\n", label, got, want)
			fail++
		}
	}

	dbValue := func(id string) int {
		var v int
		if err := db.QueryRow("SELECT minimum_stock FROM items WHERE id = ?", id).Scan(&v); err != nil {
			log.Fatalf("db read (%s): %v", id, err)
		}
		return v
	}

	itemName := "ZZ HTTP Min Stock " + token[len(token)-6:]
	url := baseURL + "/admin/items-catalog/"

	// 1) CREATE via HTTP with minimum_stock = 12
	respBody, status := doJSON("POST", url, token, map[string]any{
		"name": itemName, "default_unit": "PCS", "minimum_stock": 12,
	})
	if status != http.StatusOK {
		log.Fatalf("create failed (status %d): %s", status, respBody)
	}
	var created struct {
		Data struct {
			ID           string `json:"id"`
			MinimumStock int    `json:"minimum_stock"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(respBody), &created); err != nil {
		log.Fatalf("decode create: %v (body: %s)", err, respBody)
	}
	id := created.Data.ID
	fmt.Printf("Created item id=%s\n", id)
	check("POST create minimum_stock=12 (response)", created.Data.MinimumStock, 12)
	check("POST create minimum_stock=12 (in DB)", dbValue(id), 12)

	// 2) GET list contains the field
	var listEnvelope struct {
		Data []struct {
			ID           string `json:"id"`
			MinimumStock int    `json:"minimum_stock"`
		} `json:"data"`
	}
	respBody, status = doJSON("GET", url, token, nil)
	if status != http.StatusOK {
		log.Fatalf("list failed (status %d): %s", status, respBody)
	}
	if err := json.Unmarshal([]byte(respBody), &listEnvelope); err != nil {
		log.Fatalf("decode list: %v", err)
	}
	foundList := -1
	for _, it := range listEnvelope.Data {
		if it.ID == id {
			foundList = it.MinimumStock
		}
	}
	check("GET list returns minimum_stock=12", foundList, 12)

	// 3) GET detail contains the field
	var detail struct {
		Data struct {
			MinimumStock int `json:"minimum_stock"`
		} `json:"data"`
	}
	respBody, status = doJSON("GET", url+id, token, nil)
	if status != http.StatusOK {
		log.Fatalf("detail failed (status %d): %s", status, respBody)
	}
	if err := json.Unmarshal([]byte(respBody), &detail); err != nil {
		log.Fatalf("decode detail: %v", err)
	}
	check("GET detail returns minimum_stock=12", detail.Data.MinimumStock, 12)

	// 4) PUT minimum_stock 12 -> 0 (the GORM zero-value edge case)
	respBody, status = doJSON("PUT", url+id, token, map[string]any{
		"name": itemName, "default_unit": "PCS", "minimum_stock": 0,
	})
	if status != http.StatusOK {
		log.Fatalf("put 0 failed (status %d): %s", status, respBody)
	}
	check("PUT minimum_stock 12 -> 0 (in DB)", dbValue(id), 0)

	respBody, status = doJSON("GET", url+id, token, nil)
	if status != http.StatusOK {
		log.Fatalf("detail after put failed: %s", respBody)
	}
	if err := json.Unmarshal([]byte(respBody), &detail); err != nil {
		log.Fatalf("decode detail: %v", err)
	}
	check("PUT minimum_stock 12 -> 0 (GET detail)", detail.Data.MinimumStock, 0)

	// 5) PUT 0 -> 9, then PUT omitting the field -> must stay 9
	if _, status = doJSON("PUT", url+id, token, map[string]any{
		"name": itemName, "default_unit": "PCS", "minimum_stock": 9,
	}); status != http.StatusOK {
		log.Fatalf("put 9 failed: %d", status)
	}
	check("PUT minimum_stock 0 -> 9 (in DB)", dbValue(id), 9)

	respBody, status = doJSON("PUT", url+id, token, map[string]any{
		"name": itemName, "default_unit": "PCS", // minimum_stock omitted
	})
	if status != http.StatusOK {
		log.Fatalf("put omit failed (status %d): %s", status, respBody)
	}
	check("PUT without minimum_stock keeps 9 (in DB)", dbValue(id), 9)

	// 6) Negative value must be rejected by validation (gte=0)
	_, status = doJSON("PUT", url+id, token, map[string]any{
		"name": itemName, "default_unit": "PCS", "minimum_stock": -1,
	})
	if status == http.StatusBadRequest {
		fmt.Printf("  ✅ %-56s status=%d\n", "PUT negative rejected with 400", status)
		pass++
	} else {
		fmt.Printf("  ❌ %-56s got status=%d want=400\n", "PUT negative rejected with 400", status)
		fail++
	}

	// Cleanup
	if _, err := db.Exec("DELETE FROM items WHERE id = ?", id); err != nil {
		log.Printf("cleanup failed: %v", err)
	}

	fmt.Printf("\nResult: %d passed, %d failed\n", pass, fail)
	if fail > 0 {
		os.Exit(1)
	}
}

func doJSON(method, url, token string, payload any) (string, int) {
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("http %s %s: %v", method, url, err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return string(b), res.StatusCode
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
