package authapi

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"skripsi-be/internal/helpers"
	"skripsi-be/internal/models/dto"
	"skripsi-be/internal/models/entities"

	"github.com/jinzhu/copier"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthServiceInterface interface {
	LoginAuthService() (*entities.User, error)
}

// OTPChallengeResult is returned by RequestOTPService. It intentionally omits
// the generated code; only metadata useful for the UI is exposed.
type OTPChallengeResult struct {
	PhoneMasked  string `json:"phone_masked"`
	CustomerName string `json:"customer_name,omitempty"`
	ExpiresIn    int    `json:"expires_in"` // seconds
	ResendIn     int    `json:"resend_in"`  // seconds
}

type AuthServiceStruct struct {
	repository *AuthRepositoryStruct
}

func NewAuthService(repository *AuthRepositoryStruct) *AuthServiceStruct {
	return &AuthServiceStruct{repository}
}

func (r *AuthServiceStruct) LoginAuthService(LoginRequest LoginRequest) (*dto.LoginAuthDTO, error) {
	// Service logic here
	user, err := r.repository.GetEmailAuthRepository(LoginRequest)

	if err != nil {
		// Handle error
		return nil, err
	}

	// Check if the user exists
	if user.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	// Check if the password is correct
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(LoginRequest.Password)); err != nil {
		return nil, err
	}

	userDTO := dto.ModelToUser(*user)
	// Return the user data

	token, err := helpers.CreateToken(userDTO.ID, userDTO.Role.Name)
	if err != nil {
		return nil, err
	}
	loginAuthDTO := dto.ModelToLoginAuth(token, *user)

	r.repository.UpdateTokenUserByID(user.ID, token)

	return loginAuthDTO, nil
}

func (r *AuthServiceStruct) LoginCustomerAuthService(LoginRequest LoginCustomerRequest) (*dto.CustomerLoginAuthDTO, error) {
	// Normalise the phone: 0xxx / +62xxx / 62xxx -> 62xxx. The repository also
	// tolerates stored formatting (dashes/spaces).
	phone := NormalizePhone(LoginRequest.Phone)
	if phone == "" {
		return nil, fmt.Errorf("nomor telepon tidak valid")
	}

	user, err := r.repository.GetCustomerByPhoneVariants(phoneVariants(phone))
	if err != nil {
		// Handle error
		return nil, err
	}

	// Check if the user exists
	if user.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	// Check if the password is correct
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(LoginRequest.Password)); err != nil {
		return nil, err
	}

	customerDTO := dto.CustomerDTO{}
	copier.Copy(&customerDTO, user)

	token, err := helpers.CreateToken(customerDTO.ID, "CUSTOMER")
	if err != nil {
		return nil, err
	}
	loginAuthDTO := dto.ModelToCustomerLoginAuth(token, customerDTO)

	// Save token to database for verification
	r.repository.UpdateTokenUserByID(user.ID, token)

	return loginAuthDTO, nil
}

// RequestOTPService validates the phone number, makes sure a customer exists
// and sends a 6 digit OTP over WhatsApp.
//
// The response never reveals whether the phone number belongs to a customer:
// unknown numbers get the same message so the endpoint cannot be used to
// enumerate customer phone numbers.
func (r *AuthServiceStruct) RequestOTPService(request RequestOTPRequest) (OTPChallengeResult, error) {
	result := OTPChallengeResult{}

	phone := NormalizePhone(request.Phone)
	if phone == "" {
		return result, fmt.Errorf("nomor WhatsApp tidak valid. Contoh: 081234567890")
	}

	customer, err := r.repository.GetCustomerByPhoneVariants(phoneVariants(phone))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[OTP] request for unknown phone %s", maskPhone(phone))
			return result, fmt.Errorf("nomor WhatsApp belum terdaftar. Hubungi layanan pelanggan kami")
		}
		return result, err
	}

	code, err := generateOTPCode()
	if err != nil {
		return result, err
	}
	codeHash, err := hashOTP(code)
	if err != nil {
		return result, err
	}

	// Rate limit before delivery so repeated requests cannot be used to spam a
	// customer's WhatsApp.
	if err := issueOTP(phone, codeHash); err != nil {
		return result, err
	}

	if err := sendWhatsAppOTP(phone, code); err != nil {
		log.Printf("[OTP] failed to deliver OTP for %s: %v", maskPhone(phone), err)
		// Roll back the challenge so the customer may retry immediately.
		otpChallenges.mu.Lock()
		delete(otpChallenges.entries, phone)
		if len(otpChallenges.requests[phone]) > 0 {
			otpChallenges.requests[phone] = otpChallenges.requests[phone][:len(otpChallenges.requests[phone])-1]
		}
		otpChallenges.mu.Unlock()
		return result, fmt.Errorf("gagal mengirim kode OTP ke WhatsApp. Silakan coba lagi")
	}

	result.PhoneMasked = maskPhone(phone)
	result.ExpiresIn = int(otpTTL.Seconds())
	result.ResendIn = int(otpResendCooldown.Seconds())
	result.CustomerName = customer.Name

	return result, nil
}

// VerifyOTPService checks the submitted code and issues a customer scoped JWT.
// The token audience is always "CUSTOMER" so it is rejected by every
// admin/staff route (helpers.VerifyToken + RequireRoles).
func (r *AuthServiceStruct) VerifyOTPService(request VerifyOTPRequest) (*dto.CustomerLoginAuthDTO, error) {
	phone := NormalizePhone(request.Phone)
	if phone == "" {
		return nil, fmt.Errorf("nomor WhatsApp tidak valid. Contoh: 081234567890")
	}

	customer, err := r.repository.GetCustomerByPhoneVariants(phoneVariants(phone))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("nomor WhatsApp belum terdaftar. Hubungi layanan pelanggan kami")
		}
		return nil, err
	}

	if err := consumeOTP(phone, strings.TrimSpace(request.Code)); err != nil {
		return nil, err
	}

	customerDTO := dto.CustomerDTO{}
	copier.Copy(&customerDTO, customer)

	// Audience is hardcoded to CUSTOMER - never derived from the request.
	token, err := helpers.CreateToken(customerDTO.ID, "CUSTOMER")
	if err != nil {
		return nil, err
	}

	log.Printf("[OTP] customer %s logged in via WhatsApp OTP", maskPhone(phone))

	return dto.ModelToCustomerLoginAuth(token, customerDTO), nil
}

func (r *AuthServiceStruct) VerifyAuthService(LoginRequest LoginRequest) (*dto.LoginAuthDTO, error) {
	// Service logic here
	user, err := r.repository.GetEmailAuthRepository(LoginRequest)

	if err != nil {
		// Handle error
		return nil, err
	}

	// Check if the user exists
	if user.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	// Check if the password is correct
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(LoginRequest.Password)); err != nil {
		return nil, err
	}

	userDTO := dto.ModelToUser(*user)
	// Return the user data

	token, err := helpers.CreateToken(userDTO.ID, userDTO.Role.Name)
	if err != nil {
		return nil, err
	}
	loginAuthDTO := dto.ModelToLoginAuth(token, *user)

	r.repository.UpdateTokenUserByID(user.ID, token)

	return loginAuthDTO, nil
}
