package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"skripsi-be/internal/api/admin/recurring_invoice"
	"skripsi-be/internal/config/database"
	"skripsi-be/internal/routes"
	"skripsi-be/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	recovermw "github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Initialize slog with LOG_LEVEL environment variable
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO" // default to INFO
	}

	var level slog.Level
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN", "WARNING":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	database.GetDB()
	database.AutoMigrate()
	app := fiber.New(fiber.Config{
		JSONEncoder:  json.Marshal,
		JSONDecoder:  json.Unmarshal,
		BodyLimit:    50 * 1024 * 1024,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	})
	// CORS configuration
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000, http://127.0.0.1:3000, http://192.168.1.7:3000, http://192.168.1.11:3000, http://localhost:3002, http://127.0.0.1:3002, http://192.168.1.11:3002, https://menara.net.id, http://rndpolije.lilly.net.id, https://rndpolije.lilly.net.id, http://rndpolije.lilly.net.id:3000, https://rndpolije.lilly.net.id:3000",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	// Recovery and request logging for diagnostics
	app.Use(recovermw.New())
	app.Use(customLoggerMiddleware())

	// Initialize routes
	routes.RouteFiber(app)

	// Auto-connect to MikroTik on startup if credentials are available
	go func() {
		// Wait a bit for the server to fully start
		time.Sleep(2 * time.Second)

		host := os.Getenv("MIKROTIK_HOST")
		port := os.Getenv("MIKROTIK_PORT")
		username := os.Getenv("MIKROTIK_USERNAME")
		password := os.Getenv("MIKROTIK_PASSWORD")

		if host != "" && port != "" && username != "" && password != "" {
			log.Printf("[mikrotik] auto-connecting to %s:%s", host, port)

			// Create MikroTik service directly
			config := &services.MikroTikConfig{
				Host:     host,
				Port:     parsePort(port),
				Username: username,
				Password: password,
			}

			mtService := services.NewMikroTikService(config)
			if err := mtService.Connect(); err != nil {
				log.Printf("[mikrotik] auto-connect failed: %v", err)
			} else {
				services.SetSharedMikroTikService(mtService)
				log.Printf("[mikrotik] auto-connected successfully")
			}
		} else {
			log.Printf("[mikrotik] no credentials found in environment variables")
		}
	}()

	// Background scheduler: ONLY recurring invoice generation (MikroTik enforcement removed)
	go func() {
		run := func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[scheduler] panic: %v", r)
				}
			}()
			start := time.Now()
			db := database.GetDB()
			repo := recurring_invoice.NewAdminRecurringInvoiceRepository(db)
			n, err := repo.ProcessDueRecurringInvoices()
			dur := time.Since(start)
			if err != nil {
				log.Printf("[recurring] run error after %s: %v", dur, err)
			} else if n > 0 {
				log.Printf("[recurring] generated %d invoices in %s", n, dur)
			}
		}

		run()
		ticker := time.NewTicker(1 * time.Minute)
		log.Printf("[recurring] scheduler started, interval=1m (mikrotik enforcement disabled)")
		defer ticker.Stop()
		for range ticker.C {
			run()
		}
	}()

	// Start RouterJobs worker
	go func() {
		db := database.GetDB()
		services.StartRouterJobWorker(db)
	}()

	log.Fatal(app.Listen(":3001"))
}

func parsePort(portStr string) int {
	if port, err := strconv.Atoi(portStr); err == nil {
		return port
	}
	return 22 // default SSH port
}

// customLoggerMiddleware creates a Fiber middleware that uses slog for HTTP request logging
func customLoggerMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process request
		err := c.Next()

		// Log request details
		slog.Info("HTTP Request",
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"latency", time.Since(start),
			"ip", c.IP(),
			"user_agent", c.Get("User-Agent"),
		)

		return err
	}
}
