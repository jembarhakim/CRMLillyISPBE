package entities

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JSONArray is a custom type for JSON array fields
type JSONArray []interface{}

// Scan implements the sql.Scanner interface
func (j *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Value implements the driver.Valuer interface
func (j JSONArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// InstallationProvisioningLog represents audit log for MikroTik RouterOS provisioning
type InstallationProvisioningLog struct {
	ID                     string    `gorm:"column:id;type:varchar(36);primaryKey" json:"id"`
	CustomerInstallationID string    `gorm:"column:customer_installation_id;type:varchar(255);not null;index:idx_ipl_customer_installation_id" json:"customer_installation_id"`
	CustomerID             *string   `gorm:"column:customer_id;type:varchar(255);index:idx_ipl_customer_id" json:"customer_id,omitempty"`
	MACAddress             *string   `gorm:"column:mac_address;type:varchar(17);index:idx_ipl_mac_address" json:"mac_address,omitempty"`
	IPAddress              *string   `gorm:"column:ip_address;type:varchar(15)" json:"ip_address,omitempty"`
	CodeName               *string   `gorm:"column:code_name;type:varchar(255);index:idx_ipl_code_name" json:"code_name,omitempty"`
	Status                 string    `gorm:"column:status;type:enum('queued','running','success','failed','rolled_back');not null;default:'queued';index:idx_ipl_status" json:"status"`
	ProvisioningType       string    `gorm:"column:provisioning_type;type:enum('new','update','dry_run');not null;default:'new'" json:"provisioning_type"`
	CommandsExecuted       JSONArray `gorm:"column:commands_executed;type:json" json:"commands_executed,omitempty"`
	CommandsOutput         JSONArray `gorm:"column:commands_output;type:json" json:"commands_output,omitempty"`
	ErrorMessage           *string   `gorm:"column:error_message;type:text" json:"error_message,omitempty"`
	RetryCount             int       `gorm:"column:retry_count;type:int;default:0" json:"retry_count"`
	ExecutionTimeMs        *int      `gorm:"column:execution_time_ms;type:int" json:"execution_time_ms,omitempty"`
	DryRun                 bool      `gorm:"column:dry_run;type:boolean;default:false" json:"dry_run"`
	CreatedBy              *string   `gorm:"column:created_by;type:varchar(255)" json:"created_by,omitempty"`
	CreatedAt              time.Time `gorm:"column:createdAt;default:CURRENT_TIMESTAMP(3);index:idx_ipl_created_at" json:"createdAt"`
	UpdatedAt              time.Time `gorm:"column:updatedAt;default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)" json:"updatedAt"`

	// Relations
	CustomerInstallation *CustomerInstallation `gorm:"foreignKey:CustomerInstallationID;references:ID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"customer_installation,omitempty"`
	Customer             *Customer             `gorm:"foreignKey:CustomerID;references:ID;constraint:OnDelete:SET NULL,OnUpdate:CASCADE" json:"customer,omitempty"`
	Creator              *User                 `gorm:"foreignKey:CreatedBy;references:ID;constraint:OnDelete:SET NULL,OnUpdate:CASCADE" json:"creator,omitempty"`
}

func (ipl *InstallationProvisioningLog) TableName() string {
	return "installation_provisioning_logs"
}

func (ipl *InstallationProvisioningLog) BeforeCreate(tx *gorm.DB) error {
	if ipl.ID == "" {
		ipl.ID = uuid.New().String()
	}
	return nil
}

// ProvisioningStatus constants
const (
	ProvisioningStatusQueued     = "queued"
	ProvisioningStatusRunning    = "running"
	ProvisioningStatusSuccess    = "success"
	ProvisioningStatusFailed     = "failed"
	ProvisioningStatusRolledBack = "rolled_back"
)

// ProvisioningType constants
const (
	ProvisioningTypeNew    = "new"
	ProvisioningTypeUpdate = "update"
	ProvisioningTypeDryRun = "dry_run"
)

