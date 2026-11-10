package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =============================================================================
// B2B Quotation (Penawaran Harga / RAB) — versioned
//
// One project can have many quotations. Each save of a revision creates a new
// row with an incremented Version so negotiation history is never lost.
// =============================================================================

// B2BQuotationStatus is the negotiation state of one revision.
type B2BQuotationStatus string

const (
	B2BQuotationDraft    B2BQuotationStatus = "draft"
	B2BQuotationSent     B2BQuotationStatus = "sent"
	B2BQuotationApproved B2BQuotationStatus = "approved"
	B2BQuotationRejected B2BQuotationStatus = "rejected"
	B2BQuotationSuperseded B2BQuotationStatus = "superseded"
)

// B2BQuotationItemCategory groups RAB lines exactly like the commercial form.
type B2BQuotationItemCategory string

const (
	B2BItemDevice  B2BQuotationItemCategory = "perangkat"
	B2BItemCable   B2BQuotationItemCategory = "kabel"
	B2BItemService B2BQuotationItemCategory = "jasa"
	B2BItemOther   B2BQuotationItemCategory = "lainnya"
)

// B2BQuotation is one revision of the price offer.
type B2BQuotation struct {
	ID        string `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	ProjectID string `gorm:"column:project_id;type:varchar(191);not null;index:idx_b2b_quotation_project" json:"project_id"`

	Version     int                `gorm:"column:version;type:int;not null;default:1" json:"version"`
	Status      B2BQuotationStatus `gorm:"column:status;type:varchar(20);not null;default:'draft'" json:"status"`
	Subject     string             `gorm:"column:subject;type:varchar(191)" json:"subject"`
	Notes       string             `gorm:"column:notes;type:text" json:"notes"`
	Terms       string             `gorm:"column:terms;type:text" json:"terms"`

	// Money. Subtotal is the sum of item subtotals; tax is applied on top.
	Subtotal   int64   `gorm:"column:subtotal;type:bigint;not null;default:0" json:"subtotal"`
	TaxPercent float64 `gorm:"column:tax_percent;type:decimal(5,2);not null;default:11" json:"tax_percent"`
	TaxAmount  int64   `gorm:"column:tax_amount;type:bigint;not null;default:0" json:"tax_amount"`
	GrandTotal int64   `gorm:"column:grand_total;type:bigint;not null;default:0" json:"grand_total"`

	ValidUntil *time.Time `gorm:"column:valid_until;type:date" json:"valid_until"`

	CreatedBy string    `gorm:"column:created_by;type:varchar(191)" json:"created_by"`
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP(3)" json:"updated_at"`

	Items []B2BQuotationItem `gorm:"foreignKey:QuotationID;references:ID" json:"items,omitempty"`
}

func (B2BQuotation) TableName() string { return "b2b_quotations" }

func (q *B2BQuotation) BeforeCreate(tx *gorm.DB) error {
	if q.ID == "" {
		q.ID = uuid.New().String()
	}
	if q.Status == "" {
		q.Status = B2BQuotationDraft
	}
	if q.Version == 0 {
		q.Version = 1
	}
	return nil
}

// B2BQuotationItem is one RAB line: device, cable or installation service.
type B2BQuotationItem struct {
	ID          string `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	QuotationID string `gorm:"column:quotation_id;type:varchar(191);not null;index:idx_b2b_qitem_quotation" json:"quotation_id"`

	Category  B2BQuotationItemCategory `gorm:"column:category;type:varchar(20);not null;default:'perangkat'" json:"category"`
	Name      string                   `gorm:"column:name;type:varchar(191);not null" json:"name"`
	Unit      string                   `gorm:"column:unit;type:varchar(30)" json:"unit"`
	Qty       float64                  `gorm:"column:qty;type:decimal(12,2);not null;default:1" json:"qty"`
	UnitPrice int64                    `gorm:"column:unit_price;type:bigint;not null;default:0" json:"unit_price"`
	Subtotal  int64                    `gorm:"column:subtotal;type:bigint;not null;default:0" json:"subtotal"`
	Note      string                   `gorm:"column:note;type:varchar(191)" json:"note"`

	SortOrder int       `gorm:"column:sort_order;type:int;default:0" json:"sort_order"`
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
}

func (B2BQuotationItem) TableName() string { return "b2b_quotation_items" }

func (i *B2BQuotationItem) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.New().String()
	}
	return nil
}
