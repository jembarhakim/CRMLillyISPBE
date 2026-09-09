package middleware

import (
	"fmt"
	"log"
	"os"
	"skripsi-be/internal/config/database"
	"skripsi-be/internal/helpers"
	"skripsi-be/internal/models/entities"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

// CustomerAuthMiddleware validates JWT tokens for customer access
func CustomerAuthMiddleware(c *fiber.Ctx) error {
	// Try to load .env file, but don't fail if it doesn't exist (for production)
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	secretKey := []byte(os.Getenv("JWT_SECRET_KEY"))
	if len(secretKey) == 0 {
		secretKey = []byte(os.Getenv("JWT_SECRET"))
	}
	if len(secretKey) == 0 {
		secretKey = []byte("lilly-isp-local-dev-secret")
	}

	tokenString := c.Get("Authorization")
	log.Println("Customer token from header:", tokenString)
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	if tokenString == "" {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Token not provided", nil)
	}

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Invalid Token", err.Error())
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Invalid Token", nil)
	}
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Token Expired", nil)
	}
	if claims.Subject == "" || len(claims.Audience) == 0 {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Invalid Token Claims", nil)
	}

	subject := claims.Subject
	// For customer authentication, we need to verify the customer exists
	db := database.GetDB()
	var customer entities.Customer
	if err := db.Where("id = ? OR email = ?", subject, subject).First(&customer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Customer not found", nil)
		}
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, "Database error", nil)
	}

	// Set customer information in context
	c.Locals("customer_id", customer.ID)
	c.Locals("customer_phone", customer.Phone)
	c.Locals("customer_name", customer.Name)
	c.Locals("user_id", subject)
	c.Locals("role", claims.Audience[0])

	return c.Next()
}

// AdminAuthMiddleware validates JWT tokens for admin/staff access
func AdminAuthMiddleware(c *fiber.Ctx) error {
	// Try to load .env file, but don't fail if it doesn't exist (for production)
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	secretKey := []byte(os.Getenv("JWT_SECRET_KEY"))
	if len(secretKey) == 0 {
		secretKey = []byte(os.Getenv("JWT_SECRET"))
	}
	if len(secretKey) == 0 {
		secretKey = []byte("lilly-isp-local-dev-secret")
	}

	tokenString := c.Get("Authorization")
	log.Println("Admin token from header:", tokenString)
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	if tokenString == "" {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Token not provided", nil)
	}

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Invalid Token", err.Error())
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Invalid Token", nil)
	}
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Token Expired", nil)
	}
	if claims.Subject == "" || len(claims.Audience) == 0 {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Invalid Token Claims", nil)
	}

	subject := claims.Subject
	// For admin authentication, verify the user exists in the users table
	db := database.GetDB()
	var user entities.User
	if err := db.Where("id = ? OR email = ?", subject, subject).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "User not found", nil)
		}
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, "Database error", nil)
	}

	// Set user information in context
	c.Locals("user_id", user.ID)
	c.Locals("user_email", user.Email)
	c.Locals("user_name", user.Name)
	c.Locals("role", claims.Audience[0])

	return c.Next()
}
