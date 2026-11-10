package authapi

import (
	"skripsi-be/internal/config/database"
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
)

func AuthRoute(app fiber.Router) {
	dbmysql := database.GetDB()
	repository := NewAuthRepository(dbmysql)
	service := NewAuthService(repository)
	handler := NewAuthHandler(service)

	app.Post("/login", handler.LoginAuthHandler)
	app.Post("/login-customer", handler.LoginCustomerAuthHandler)
	app.Post("/verify", helpers.VerifyToken, handler.VerifyAuthHandler)
	app.Post("/verify-customer", helpers.CustomerVerifyToken, handler.VerifyAuthHandler)

	// WhatsApp OTP login for customers (see waotp.go for rate limiting rules).
	app.Post("/request-otp", handler.RequestOTPHandler)
	app.Post("/verify-otp", handler.VerifyOTPHandler)
}
