package servicerequests

// CreateServiceRequestRequest is the payload for
// POST /api/customer/service-requests
type CreateServiceRequestRequest struct {
	// RequestType: upgrade | downgrade | relocation
	RequestType string `json:"request_type" validate:"required,oneof=upgrade downgrade relocation"`

	// RequestedProductID is required for upgrade and downgrade.
	RequestedProductID string `json:"requested_product_id"`

	// Relocation fields (required when request_type = relocation).
	NewAddress   string   `json:"new_address"`
	NewAreaID    string   `json:"new_area_id"`
	NewLatitude  *float64 `json:"new_latitude"`
	NewLongitude *float64 `json:"new_longitude"`

	Note          string `json:"note" validate:"max=1000"`
	AttachmentURL string `json:"attachment_url"`
}

// ReviewServiceRequestRequest is the payload for the admin approval action.
type ReviewServiceRequestRequest struct {
	// Action: approve | reject
	Action     string `json:"action" validate:"required,oneof=approve reject"`
	ReviewNote string `json:"review_note" validate:"max=1000"`
}

// ServiceRequestItem is the representation used by both portals.
type ServiceRequestItem struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id,omitempty"`

	RequestType  string `json:"request_type"`
	TypeLabel    string `json:"type_label"`
	Status       string `json:"status"`
	StatusLabel  string `json:"status_label"`

	CurrentProductID    string `json:"current_product_id"`
	CurrentProductName  string `json:"current_product_name"`
	RequestedProductID  string `json:"requested_product_id"`
	RequestedProductName string `json:"requested_product_name"`

	NewAddress string `json:"new_address,omitempty"`
	NewAreaID  string `json:"new_area_id,omitempty"`

	Note          string `json:"note,omitempty"`
	AttachmentURL string `json:"attachment_url,omitempty"`

	ReviewNote string `json:"review_note,omitempty"`
	ReviewedAt string `json:"reviewed_at,omitempty"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// ServiceSummary is the "current package" summary card of the service page.
type ServiceSummary struct {
	ProductID    string `json:"product_id"`
	ProductName  string `json:"product_name"`
	Description  string `json:"description"`
	Price        int64  `json:"price"`
	DownloadMbps int    `json:"download_speed_mbps"`
	UploadMbps   int    `json:"upload_speed_mbps"`

	// Billing period derived from the installation date (1 month cycle).
	ActiveSince      string `json:"active_since,omitempty"`
	BillingCycleDay  int    `json:"billing_cycle_day,omitempty"`
	NextInvoiceDate  string `json:"next_invoice_date,omitempty"`
	DaysRemaining    int    `json:"days_remaining"`
	IsActive         bool   `json:"is_active"`
	CustomerName     string `json:"customer_name,omitempty"`
	CustomerAddress  string `json:"customer_address,omitempty"`
}

// ServicePageData bundles the summary card, the request history and the list of
// packages the customer may switch to.
type ServicePageData struct {
	Summary         ServiceSummary       `json:"summary"`
	AvailablePlans  []AvailablePlan      `json:"available_plans"`
	Requests        []ServiceRequestItem `json:"requests"`
}

// AvailablePlan is one upgrade/downgrade option.
type AvailablePlan struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Price        int64  `json:"price"`
	DownloadMbps int    `json:"download_speed_mbps"`
	UploadMbps   int    `json:"upload_speed_mbps"`
	// Direction is "upgrade", "downgrade" or "same" relative to the active plan.
	Direction string `json:"direction"`
}
