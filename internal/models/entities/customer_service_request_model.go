package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ServiceRequestType is the kind of change a customer asks for.
type ServiceRequestType string

// ServiceRequestStatus is the approval state inside the CRM.
type ServiceRequestStatus string

const (
	ServiceRequestUpgrade   ServiceRequestType = "upgrade"
	ServiceRequestDowngrade ServiceRequestType = "downgrade"
	ServiceRequestRelocation ServiceRequestType = "relocation"

	ServiceRequestPending  ServiceRequestStatus = "pending"
	ServiceRequestApproved ServiceRequestStatus = "approved"
	ServiceRequestRejected ServiceRequestStatus = "rejected"
)

// CustomerServiceRequest stores a customer's official request (upgrade speed,
// downgrade, or relocation). Requests are queued for the CRM admin team, which
// approves or rejects them from the dashboard.
type CustomerServiceRequest struct {
	ID         string `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	CustomerID string `gorm:"column:customer_id;type:varchar(191);not null;index:idx_svc_req_customer" json:"customer_id"`

	RequestType ServiceRequestType `gorm:"column:request_type;type:varchar(30);not null" json:"request_type"`
	Status      ServiceRequestStatus `gorm:"column:status;type:varchar(30);not null;default:'pending';index:idx_svc_req_status" json:"status"`

	// Current package at the time of the request (snapshot, never mutated).
	CurrentProductID string `gorm:"column:current_product_id;type:varchar(191)" json:"current_product_id"`

	// Requested package for upgrade / downgrade.
	RequestedProductID string `gorm:"column:requested_product_id;type:varchar(191)" json:"requested_product_id"`

	// Relocation payload.
	NewAddress   string   `gorm:"column:new_address;type:text" json:"new_address"`
	NewLatitude  *float64 `gorm:"column:new_latitude" json:"new_latitude,omitempty"`
	NewLongitude *float64 `gorm:"column:new_longitude" json:"new_longitude,omitempty"`
	NewAreaID    string   `gorm:"column:new_area_id;type:varchar(191)" json:"new_area_id"`

	// Customer note + optional attachment (uploaded photo of new location, etc).
	Note          string `gorm:"column:note;type:text" json:"note"`
	AttachmentURL string `gorm:"column:attachment_url;type:varchar(255)" json:"attachment_url"`

	// Admin approval trail.
	ReviewedBy     *string    `gorm:"column:reviewed_by;type:varchar(191)" json:"reviewed_by,omitempty"`
	ReviewedAt     *time.Time `gorm:"column:reviewed_at" json:"reviewed_at,omitempty"`
	ReviewNote     string     `gorm:"column:review_note;type:text" json:"review_note"`

	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP(3)" json:"updated_at"`
}

func (CustomerServiceRequest) TableName() string { return "customer_service_requests" }

func (r *CustomerServiceRequest) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	if r.Status == "" {
		r.Status = ServiceRequestPending
	}
	return nil
}
