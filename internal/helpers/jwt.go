package helpers

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

func loadJWTConfig() {
	_ = godotenv.Load()
	_ = godotenv.Load(".env")
}

func getJWTSecret() []byte {
	loadJWTConfig()

	for _, key := range []string{"JWT_SECRET_KEY", "JWT_SECRET"} {
		secret := strings.TrimSpace(os.Getenv(key))
		if secret != "" {
			return []byte(secret)
		}
	}

	return []byte("lilly-isp-local-dev-secret")
}

// Function to create JWT tokens with claims
func CreateToken(username string, role string) (string, error) {
	secretKey := getJWTSecret()

	claims := jwt.RegisteredClaims{
		Subject:   username,
		Issuer:    "lillyapps",
		Audience:  jwt.ClaimStrings{role},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secretKey)
	if err != nil {
		return "", err
	}

	fmt.Printf("Token claims added: subject=%s role=%s issuer=%s\n", username, role, claims.Issuer)
	return tokenString, nil
}

func parseAndValidateToken(tokenString string) (*jwt.Token, *jwt.RegisteredClaims, error) {
	secretKey := getJWTSecret()

	tokenString = strings.TrimSpace(tokenString)
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return nil, nil, fmt.Errorf("token not provided")
	}

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})
	if err != nil {
		return nil, nil, err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return nil, nil, fmt.Errorf("invalid token claims")
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, nil, fmt.Errorf("token expired")
	}

	if claims.Subject == "" {
		return nil, nil, fmt.Errorf("missing subject")
	}

	if len(claims.Audience) == 0 {
		return nil, nil, fmt.Errorf("missing audience")
	}

	return token, claims, nil
}

// Function to verify JWT tokens
func VerifyToken(c *fiber.Ctx) error {
	tokenString := c.Get("Authorization")
	parsedToken, claims, err := parseAndValidateToken(tokenString)
	if err != nil {
		message := "Invalid Token"
		switch {
		case strings.Contains(err.Error(), "token not provided"):
			message = "Token not provided"
		case strings.Contains(err.Error(), "token expired"):
			message = "Token Expired"
		case strings.Contains(err.Error(), "missing subject") || strings.Contains(err.Error(), "missing audience"):
			message = "Invalid Token Claims"
		default:
			message = "Invalid Token (parse)"
		}
		return ResponseUtils(c, fiber.StatusUnauthorized, false, message, err.Error())
	}

	if parsedToken == nil || claims == nil || !parsedToken.Valid {
		return ResponseUtils(c, fiber.StatusUnauthorized, false, "Invalid Token (not valid)", nil)
	}

	// Customer tokens must never be accepted by staff/admin routes.
	// They are issued separately (role "CUSTOMER") and are only valid for
	// /api/customer/* and /api/auth/verify-customer.
	//
	// A browser that still holds an old customer token while showing the staff
	// dashboard lands here. The message is actionable on purpose: the frontend
	// uses it to clear the stale session and send the user back to /employee.
	if IsCustomerRole(claims.Audience[0]) {
		slog.Warn("customer token used on staff route",
			"path", c.Path(),
			"audience", claims.Audience[0],
			"subject", claims.Subject,
		)
		return ResponseUtils(c, fiber.StatusForbidden, false,
			"Customer token cannot access this resource: sesi portal pelanggan masih aktif. Silakan login ulang sebagai staf.", nil)
	}

	c.Locals("user_id", claims.Subject)
	c.Locals("role", claims.Audience[0])

	return c.Next()
}

// IsCustomerRole reports whether the JWT audience belongs to a customer token.
func IsCustomerRole(role string) bool {
	return NormalizeRole(strings.TrimSpace(role)) == "CUSTOMER"
}

func CustomerVerifyToken(c *fiber.Ctx) error {
	tokenString := c.Get("Authorization")
	parsedToken, claims, err := parseAndValidateToken(tokenString)
	if err != nil {
		message := "Invalid Token"
		switch {
		case strings.Contains(err.Error(), "token not provided"):
			message = "Token not provided"
		case strings.Contains(err.Error(), "token expired"):
			message = "Token Expired"
		case strings.Contains(err.Error(), "missing subject") || strings.Contains(err.Error(), "missing audience"):
			message = "Invalid Token Claims"
		default:
			message = "Invalid Token (parse)"
		}
		return ResponseUtils(c, fiber.StatusUnauthorized, false, message, err.Error())
	}

	if parsedToken == nil || claims == nil || !parsedToken.Valid {
		return ResponseUtils(c, fiber.StatusUnauthorized, false, "Invalid Token", nil)
	}

	// Enforce the split between customer and staff tokens.
	if !IsCustomerRole(claims.Audience[0]) {
		return ResponseUtils(c, fiber.StatusForbidden, false, "This token is not a customer token", nil)
	}

	c.Locals("user_id", claims.Subject)
	c.Locals("role", "CUSTOMER")
	c.Locals("customer_id", claims.Subject)
	return c.Next()
}

func VerifyTokenFromDB(tokenString string) error {
	return nil
}
