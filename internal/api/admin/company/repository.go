package company

import (
	"gorm.io/gorm"

	"skripsi-be/internal/models/entities"
)

type AdminCompanyRepositoryInterface interface {
	GetCompanyByID(id int) (*entities.Company, error)
	DeleteCompanyByID(id int) error
}
type AdminCompanyRepositoryStruct struct {
	db *gorm.DB
}

func NewAdminCompanyRepository(db *gorm.DB) *AdminCompanyRepositoryStruct {
	return &AdminCompanyRepositoryStruct{db}
}

func (r *AdminCompanyRepositoryStruct) GetAllAdminCompanyRepository() (*[]entities.Company, error) {
	companies := &[]entities.Company{}
	r.db.Find(&companies)

	return companies, nil
}

func (r *AdminCompanyRepositoryStruct) GetByIDAdminCompanyRepository(id IdAdminCompanyRequest) (*entities.Company, error) {
	company := &entities.Company{}
	r.db.First(&company, id)
	if company.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}

	return company, nil
}

func (r *AdminCompanyRepositoryStruct) CreateAdminCompanyRepository(request CreateAdminCompanyRequest) (*entities.Company, error) {
	company := &entities.Company{}
	company.Name = request.Name
	company.Email = request.Email
	company.Phone = request.Phone
	company.URL = request.URL
	company.LogoURL = request.LogoUrl
	company.Npwp = request.Npwp
	company.Address = request.Address
	company.Description = request.Description

	tx := r.db.Create(company)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return nil, nil
}

func (r *AdminCompanyRepositoryStruct) UpdateAdminCompanyRepository(request UpdateAdminCompanyRequest) (*entities.Company, error) {
	company := &entities.Company{}

	r.db.First(&company, "id = ?", request.Id)

	company.Name = request.Name
	company.Email = request.Email
	company.Phone = request.Phone
	company.URL = request.URL
	company.LogoURL = request.LogoUrl
	company.Npwp = request.Npwp
	company.Address = request.Address
	company.Description = request.Description

	tx := r.db.Save(company)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return nil, nil
}

func (r *AdminCompanyRepositoryStruct) DeleteCompanyByID(request IdAdminCompanyRequest) (*entities.Company, error) {
	company := &entities.Company{}

	r.db.First(&company, "id = ?", request.Id)

	tx := r.db.Delete(company)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return nil, nil
}
