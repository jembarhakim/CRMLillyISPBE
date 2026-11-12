package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MassIncidentStatus string

const (
	MassIncidentStatusActive   MassIncidentStatus = "active"
	MassIncidentStatusResolved MassIncidentStatus = "resolved"
)

type MassIncident struct {
	ID                  string             `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Title               string             `json:"title" gorm:"type:varchar(191);not null"`
	Description         *string            `json:"description,omitempty" gorm:"type:text"`
	Source              string             `json:"source" gorm:"type:varchar(32);not null;index:idx_mass_incident_source"`
	Status              MassIncidentStatus `json:"status" gorm:"type:varchar(20);not null;default:'active';index:idx_mass_incident_status"`
	NodeID              *string            `json:"node_id,omitempty" gorm:"type:varchar(36);index:idx_mass_incident_node"`
	ODPNodeID           *string            `json:"odp_node_id,omitempty" gorm:"type:varchar(36);index:idx_mass_incident_odp"`
	NodeName            string             `json:"node_name" gorm:"type:varchar(191)"`
	AffectedCustomerIDs StringArray        `json:"affected_customer_ids" gorm:"type:json"`
	AffectedTicketIDs   StringArray        `json:"affected_ticket_ids" gorm:"type:json"`
	ETA                 *time.Time         `json:"eta,omitempty"`
	ResolutionNote      *string            `json:"resolution_note,omitempty" gorm:"type:text"`
	ResolvedAt          *time.Time         `json:"resolved_at,omitempty"`
	CreatedAt           time.Time          `json:"created_at" gorm:"default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt           time.Time          `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)"`
}

func (i *MassIncident) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.NewString()
	}
	return nil
}

func (MassIncident) TableName() string { return "mass_incidents" }
