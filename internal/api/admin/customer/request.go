package customer

type CreateAdminCustomerRequest struct {
	Name                  string  `json:"name" validate:"required"`
	Alias                 string  `json:"alias"`
	Address               string  `json:"address" validate:"required"`
	AreaID                string  `json:"area_id" validate:"required"`
	Phone                 string  `json:"phone" validate:"required"`
	Latitude              float64 `json:"latitude" validate:"required"`
	Longitude             float64 `json:"longitude" validate:"required"`
	ServiceRequestDate    string  `json:"service_request_date" validate:"required"`
	SalesRepresentativeID *string `json:"sales_representative_id"`
	CompanyID             *string `json:"company_id"`
	IsInternet            string  `json:"is_internet"`
	IsCollaborator        string  `json:"is_collaborator"`
}

type IdAdminCustomerRequest struct {
	Id string `json:"id"`
}

type UpdateAdminCustomerRequest struct {
	IdAdminCustomerRequest
	CreateAdminCustomerRequest
}
