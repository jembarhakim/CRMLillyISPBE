package notificationsapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"skripsi-be/internal/api/webhook/wa"
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
)

// WhatsAppConnection mengembalikan status koneksi layanan whatsapp-web.js
// dan, bila belum terhubung, teks QR untuk ditampilkan di frontend.
// Isi QR sengaja tidak pernah ditulis ke log.
func (h *Handler) WhatsAppConnection(c *fiber.Ctx) error {
	c.Set("Cache-Control", "no-store")

	driver := wa.NewWAService().Driver()
	if driver != "local" {
		return helpers.ResponseUtils(c, 200, true, "ok", fiber.Map{
			"driver":      driver,
			"connected":   false,
			"qr_required": false,
			"qr":          "",
			"message":     "Driver " + driver + " tidak memakai QR.",
		})
	}

	baseURL := strings.TrimRight(os.Getenv("WHATSAPP_SERVICE_URL"), "/")
	if baseURL == "" {
		baseURL = "http://localhost:3002"
	}
	client := &http.Client{Timeout: 5 * time.Second}

	var status struct {
		Status      string `json:"status"`
		ClientReady bool   `json:"clientReady"`
		QRNeeded    bool   `json:"qr_needed"`
		Message     string `json:"message"`
	}
	if err := fetchWAJSON(client, baseURL+"/status", &status); err != nil {
		return helpers.ResponseUtils(c, 502, false, "Layanan WhatsApp tidak dapat dihubungi", nil)
	}

	connected := status.ClientReady
	qr := ""
	if !connected {
		var q struct {
			QR string `json:"qr"`
		}
		if err := fetchWAJSON(client, baseURL+"/qr", &q); err == nil {
			qr = q.QR
		}
	}

	return helpers.ResponseUtils(c, 200, true, "ok", fiber.Map{
		"driver":      driver,
		"connected":   connected,
		"qr_required": !connected,
		"qr":          qr,
		"message":     status.Message,
	})
}

func fetchWAJSON(client *http.Client, url string, out any) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("layanan WhatsApp membalas status %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}
