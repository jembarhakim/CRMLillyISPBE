package authapi

import (
	"skripsi-be/internal/models/entities"

	"gorm.io/gorm"
)

type AuthRepositoryInterface interface {
	GetUserByID(id string) (*entities.User, error)
}

type AuthRepositoryStruct struct {
	DB *gorm.DB
}

func NewAuthRepository(db *gorm.DB) *AuthRepositoryStruct {
	return &AuthRepositoryStruct{db}
}

func (db *AuthRepositoryStruct) GetEmailAuthRepository(LoginRequest LoginRequest) (*entities.User, error) {
	// Implement the logic to get a user by ID from the database

	user := &entities.User{}
	if err := db.DB.Preload("Role").First(user, "email = ?", LoginRequest.Email).Error; err != nil {
		return nil, err
	}

	// Check if the user exists
	if user.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (db *AuthRepositoryStruct) GetPhoneCustomerAuthRepository(LoginRequest LoginRequest) (*entities.Customer, error) {
	// Implement the logic to get a user by ID from the database

	user := &entities.Customer{}
	if err := db.DB.First(user, "phone = ?", LoginRequest.Email).Error; err != nil {
		return nil, err
	}

	// Check if the user exists
	if user.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

// GetCustomerByPhoneVariants looks a customer up by any of the phone
// representations produced by phoneVariants (62xxx / 0xxx / +62xxx).
// It also tolerates formatting characters stored in the phone column.
func (db *AuthRepositoryStruct) GetCustomerByPhoneVariants(variants []string) (*entities.Customer, error) {
	if len(variants) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	user := &entities.Customer{}

	// Primary lookup: exact match on any stored representation.
	if err := db.DB.First(user, "phone IN ?", variants).Error; err == nil {
		return user, nil
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Fallback: phone numbers stored with spaces/dashes. Compare digits only.
	for _, variant := range variants {
		err := db.DB.
			Where("REPLACE(REPLACE(REPLACE(REPLACE(phone, ' ', ''), '-', ''), '(', ''), ')', '') = ?", variant).
			First(user).Error
		if err == nil {
			return user, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	return nil, gorm.ErrRecordNotFound
}

// UpdateTokenCustomerByID stores the issued JWT on the customer row so the
// token can be revoked/inspected later (mirrors UpdateTokenUserByID).
func (db *AuthRepositoryStruct) UpdateTokenCustomerByID(id string, token string) (*entities.Customer, error) {
	customer := &entities.Customer{}
	if err := db.DB.First(customer, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return customer, nil
}

func (db *AuthRepositoryStruct) GetTokenAuthRepository(token string) (*entities.User, error) {
	// Implement the logic to get a user by ID from the database

	user := &entities.User{}
	if err := db.DB.First(user, "token = ?", token).Error; err != nil {
		return nil, err
	}

	// Check if the user exists
	if user.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (db *AuthRepositoryStruct) UpdateTokenUserByID(id string, token string) (*entities.User, error) {
	user := &entities.User{}
	if err := db.DB.First(user, "id = ?", id).Error; err != nil {
		return nil, err
	}

	// Check if the user exists
	if user.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}

	// Update the token field
	user.Token = token
	// Save the updated user back to the database
	if err := db.DB.Save(user).Error; err != nil {
		return nil, err
	}
	return user, nil

}
