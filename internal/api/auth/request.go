package authapi

type LoginRequest struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required,min=6"`
}

// LoginCustomerRequest is the payload for POST /api/auth/login-customer.
// Customers authenticate with their phone number, not email. The backend
// normalises any formatting (0xxx, +62xxx, 62xxx) before looking the row up.
type LoginCustomerRequest struct {
	Phone    string `json:"phone" validate:"required,min=9,max=20"`
	Password string `json:"password" validate:"required,min=6"`
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// RequestOTPRequest is the payload for POST /api/auth/request-otp
type RequestOTPRequest struct {
	Phone string `json:"phone" validate:"required,min=9,max=20"`
}

// VerifyOTPRequest is the payload for POST /api/auth/verify-otp
type VerifyOTPRequest struct {
	Phone string `json:"phone" validate:"required,min=9,max=20"`
	Code  string `json:"code" validate:"required,len=6,numeric"`
}
