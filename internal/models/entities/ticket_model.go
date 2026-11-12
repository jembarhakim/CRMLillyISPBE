package entities

import (
	"time"

	"gorm.io/gorm"
)

type TroubleType string
type TicketStatus string
type Assignee string

const (
	TroubleWifi     TroubleType = "wifi"
	TroubleInternet TroubleType = "internet"
	TroubleHardware TroubleType = "hardware"
	TroublePower    TroubleType = "power"
	TroubleOther    TroubleType = "other"
	// TroubleNetwork  TroubleType = "network" // New type for Netwatch-triggered tickets - DISABLED FOR TESTING

	// Use role names instead of hardcoded UUIDs - these will be looked up dynamically
	AssignCS   Assignee = "CUSTOMER SERVICE"
	AssignNOC  Assignee = "CUSTOMER SERVICE" // NOC now uses CUSTOMER_SERVICE role
	AssignTech Assignee = "TECHNICIAN"
)

type TroubleTicket struct {
	ID               uint64                `gorm:"primaryKey;autoIncrement" json:"id"`
	CustomerID       string                `gorm:"type:varchar(191)" json:"customer_id"`
	Type             *string               `gorm:"column:type;type:varchar(191)"`
	Title            string                `gorm:"type:longtext" json:"title"`
	Description      *string               `gorm:"type:text" json:"description,omitempty"`
	Status           string                `gorm:"type:varchar(191);default:'unfinished'" json:"status"`
	ClassificationID string                `gorm:"column:classification_id;type:varchar(20);default:'gangguan'" json:"classification_id"` // Reference to ticket_classification
	Classification   *TicketClassification `gorm:"foreignKey:ClassificationID;references:ID" json:"classification,omitempty"`
	AssignedTo       *string               `gorm:"type:varchar(191)" json:"assigned_to,omitempty"`
	CurrentAssignee  string                `gorm:"column:current_assignee_role;type:varchar(191)" json:"current_assignee_role"`
	CustomerNote     *string               `gorm:"type:text" json:"customer_note,omitempty"`
	NOCNote          *string               `gorm:"type:text" json:"noc_note,omitempty"`
	TechnicianNote   *string               `gorm:"type:text" json:"technician_note,omitempty"`

	// Asset reference - links to specific asset/device having trouble
	AssetItemID *string `gorm:"column:asset_item_id;type:varchar(191)" json:"asset_item_id,omitempty"`

	// Image fields
	ImgCS     *string `gorm:"type:varchar(60)" json:"img_cs,omitempty"`
	ImgNOC    *string `gorm:"type:varchar(60)" json:"img_noc,omitempty"`
	ImgTechBF *string `gorm:"type:varchar(60)" json:"img_tech_bf,omitempty"`
	ImgTechAF *string `gorm:"type:varchar(60)" json:"img_tech_af,omitempty"`

	// Technician completion tracking
	TechnicianCompleted *bool `gorm:"type:tinyint(1);default:0" json:"technician_completed,omitempty"`

	// Network architecture type selection
	NetworkArchitecture *string `gorm:"type:varchar(50)" json:"network_architecture,omitempty"` // FTTH or HTB

	// Accumulation tracking - number of customers affected by the same problem
	Accumulation int `gorm:"type:int;default:1" json:"accumulation"` // Default 1 for single customer

	// Netwatch integration fields - DISABLED FOR TESTING
	// CreatedByNetwatch bool            `gorm:"default:false" json:"created_by_netwatch"`
	// NetwatchEventID   *string         `gorm:"type:varchar(36)" json:"netwatch_event_id"`
	// NetwatchEvent     *NetwatchEvent  `json:"netwatch_event,omitempty" gorm:"foreignKey:NetwatchEventID;references:ID"`
	// DeviceID          *string         `gorm:"type:varchar(36)" json:"device_id"`
	// Device            *NetwatchDevice `json:"device,omitempty" gorm:"foreignKey:DeviceID;references:ID"`

	CreatedAt *time.Time `gorm:"column:created_at" json:"created_at,omitempty"`
	UpdatedAt *time.Time `gorm:"column:updated_at" json:"updated_at,omitempty"`
}

func (TroubleTicket) TableName() string { return "trouble_tickets" }

// GetClassificationID returns the ticket classification ID
// Returns: gangguan, psb, lainnya, or dismantle
func (t *TroubleTicket) GetClassificationID() string {
	if t.ClassificationID != "" {
		return t.ClassificationID
	}
	// Default to gangguan if not set
	return ClassificationGangguan
}

// SetClassificationID sets the ticket classification ID
// Valid values: gangguan, psb, lainnya, dismantle
func (t *TroubleTicket) SetClassificationID(classificationID string) {
	// Validate classification
	validClassifications := map[string]bool{
		ClassificationGangguan:  true,
		ClassificationPSB:       true,
		ClassificationLainnya:   true,
		ClassificationDismantle: true,
	}

	if validClassifications[classificationID] {
		t.ClassificationID = classificationID
	} else {
		// Default to gangguan if invalid
		t.ClassificationID = ClassificationGangguan
	}
}

// SetClassification is a legacy method for backward compatibility
// Maps old classification values to new classification system
func (t *TroubleTicket) SetClassification(classification string) {
	// Map legacy values to new classification system
	switch classification {
	case "info", "information":
		// Legacy "info" classification maps to "lainnya" in new system
		t.SetClassificationID(ClassificationLainnya)
	case "gangguan", "trouble":
		t.SetClassificationID(ClassificationGangguan)
	case "psb":
		t.SetClassificationID(ClassificationPSB)
	case "dismantle":
		t.SetClassificationID(ClassificationDismantle)
	case "lainnya":
		t.SetClassificationID(ClassificationLainnya)
	default:
		// Default to gangguan for unknown values
		t.SetClassificationID(ClassificationGangguan)
	}
}

// ShouldShowNOCAction returns whether NOC action button should be visible for this ticket
func (t *TroubleTicket) ShouldShowNOCAction() bool {
	// Only show NOC action for "gangguan" classification
	// Hide for: psb, lainnya, dismantle
	return t.ClassificationID == ClassificationGangguan
}

// ShouldShowInCards returns whether this ticket should be displayed in dashboard cards
func (t *TroubleTicket) ShouldShowInCards() bool {
	// Show in cards: gangguan, psb, dismantle, lainnya
	// All classifications show in cards now
	return true
}

// Enforce sane defaults before insert regardless of caller
func (t *TroubleTicket) BeforeCreate(tx *gorm.DB) (err error) {
	if t.Status == "" || t.Status == "received" {
		t.Status = "unfinished"
	}

	// Set default classification if not provided
	if t.ClassificationID == "" {
		t.ClassificationID = ClassificationGangguan
	}

	// Note: CurrentAssignee will be set by the service layer using dynamic role lookup
	// This hook is kept for other defaults but role assignment is handled in service
	return nil
}
