// Smoke test customer portal flow against a local backend.
//
// Prerequisites:
//   1. MySQL running + database "iqgncnzy_skripsi" exists
//   2. go run database-seeder.go    (seeded customers now have password "password")
//   3. go run cmd/myapp/main.go      (backend on :3001)
//
// Runs:
//   - customer login (081234567899 / password)
//   - customer dashboard GET
//   - ticket creation
//   - negative test: customer token must be rejected by /api/admin/*
//   - staff login + admin access
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const base = "http://localhost:3001"

func main() {
	fail := func(step string, err error) {
		fmt.Printf("FAIL  %s: %v\n", step, err)
		os.Exit(1)
	}
	ok := func(step string) { fmt.Printf("OK    %s\n", step) }

	client := &http.Client{Timeout: 10 * time.Second}

	post := func(path string, body any) (int, map[string]any) {
		raw, _ := json.Marshal(body)
		resp, err := client.Post(base+path, "application/json", bytes.NewReader(raw))
		if err != nil {
			fail("POST "+path, err)
		}
		defer resp.Body.Close()
		out, _ := io.ReadAll(resp.Body)
		var parsed map[string]any
		_ = json.Unmarshal(out, &parsed)
		return resp.StatusCode, parsed
	}

	get := func(path, token string) (int, map[string]any) {
		req, err := http.NewRequest(http.MethodGet, base+path, nil)
		if err != nil {
			fail("GET "+path, err)
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := client.Do(req)
		if err != nil {
			fail("GET "+path, err)
		}
		defer resp.Body.Close()
		out, _ := io.ReadAll(resp.Body)
		var parsed map[string]any
		_ = json.Unmarshal(out, &parsed)
		return resp.StatusCode, parsed
	}

	// 1. Login customer -----------------------------------------------------
	status, body := post("/api/auth/login-customer", map[string]string{
		"phone": "081234567899", "password": "password",
	})
	if status != 200 {
		fail("login-customer", fmt.Errorf("status=%d body=%v", status, body))
	}
	data, _ := body["data"].(map[string]any)
	custToken, _ := data["token"].(string)
	if custToken == "" {
		fail("login-customer", fmt.Errorf("token kosong: %v", body))
	}
	ok("login customer (081234567899 / password)")

	// 2. Dashboard customer --------------------------------------------------
	status, body = get("/api/customer/dashboard", custToken)
	if status != 200 {
		fail("customer dashboard", fmt.Errorf("status=%d body=%v", status, body))
	}
	ok("dashboard customer")

	// 3. Buat tiket ----------------------------------------------------------
	status, body = post("/api/customer/tickets", map[string]any{
		"title": "Internet mati sejak pagi",
		"description": "Tidak bisa buka situs sama sekali",
		"trouble_type_id": "internet",
	})
	if status != 201 && status != 200 {
		fail("create ticket", fmt.Errorf("status=%d body=%v", status, body))
	}
	ticketData, _ := body["data"].(map[string]any)
	if ticketID, _ := ticketData["id"].(float64); ticketID == 0 {
		fail("create ticket", fmt.Errorf("id kosong: %v", body))
	}
	ok("create ticket customer")

	// 4. Token customer DITOLAK oleh admin (negative test) -------------------
	status, _ = get("/api/admin/company", custToken)
	if status != 403 {
		fail("negative admin guard", fmt.Errorf("status=%d, harap 403", status))
	}
	ok("negative test: customer token ditolak admin (403)")

	// 5. Login staff -----------------------------------------------------------
	status, body = post("/api/auth/login", map[string]string{
		"email": "admin@lillyisp.com", "password": "password",
	})
	if status != 200 {
		fail("login staff", fmt.Errorf("status=%d body=%v", status, body))
	}
	staffData, _ := body["data"].(map[string]any)
	staffToken, _ := staffData["token"].(string)
	if staffToken == "" {
		fail("login staff", fmt.Errorf("token kosong: %v", body))
	}
	ok("login staff")

	// 6. Token staff bisa akses admin -----------------------------------------
	status, _ = get("/api/admin/company", staffToken)
	if status != 200 {
		fail("staff admin access", fmt.Errorf("status=%d, harap 200", status))
	}
	ok("staff token diterima admin")

	fmt.Println("\nSEMUA SMOKE TEST LULUS ✅")
}
