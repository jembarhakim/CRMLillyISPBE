package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"skripsi-be/internal/api/broadcast"
	"skripsi-be/internal/models/entities"
)

func main() {
	// Database connection string
	// Replace with your actual database credentials
	dsn := "user:password@tcp(localhost:3306)/iqgncnzy_skripsi?charset=utf8mb4&parseTime=True&loc=Local"

	// Connect to database
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate tables (optional - ensures tables exist)
	// Make sure to run the SQL migration first!
	err = db.AutoMigrate(
		&entities.User{},
		&entities.Role{},
		&entities.BroadcastHistory{},
	)
	if err != nil {
		log.Printf("Warning: Auto-migrate error (table may already exist): %v", err)
	}

	// Create Fiber app
	app := fiber.New()

	// Setup CORS (if needed)
	// app.Use(cors.New(cors.Config{
	// 	AllowOrigins: "*",
	// 	AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
	// 	AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	// }))

	// Setup middleware (example - you should have your own auth middleware)
	// app.Use(func(c *fiber.Ctx) error {
	// 	// Extract user ID from JWT token or session
	// 	// For example:
	// 	// token := c.Get("Authorization")
	// 	// userID := extractUserIDFromToken(token)
	// 	// c.Locals("userID", userID)
	// 	return c.Next()
	// })

	// Setup broadcast routes
	broadcast.SetupBroadcastRoutes(app, db)

	// Start server
	log.Println("Server starting on :8080")
	if err := app.Listen(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

