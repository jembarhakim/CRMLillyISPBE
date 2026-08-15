package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NetwatchDevice represents a device being monitored by MikroTik Netwatch
type NetwatchDevice struct {
	ID         string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name       string    `json:"name" gorm:"type:varchar(255);not null"`
	IPAddress  string    `json:"ip_address" gorm:"type:varchar(45);not null;uniqueIndex"`
	CustomerID *string   `json:"customer_id" gorm:"type:varchar(36)"`
	Customer   *Customer `json:"customer,omitempty" gorm:"foreignKey:CustomerID;references:ID"`
	Status     string    `json:"status" gorm:"type:enum('up','down');default:'up'"`
	LastSeen   time.Time `json:"last_seen" gorm:"default:CURRENT_TIMESTAMP(3)"`
	CreatedAt  time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP(3)"`
}

func (n *NetwatchDevice) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	return nil
}

func (NetwatchDevice) TableName() string {
	return "netwatch_devices"
}

// NetwatchEvent represents a single Netwatch event (up/down)
type NetwatchEvent struct {
	ID        string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	DeviceID  string         `json:"device_id" gorm:"type:varchar(36);not null"`
	Device    NetwatchDevice `json:"device,omitempty" gorm:"foreignKey:DeviceID;references:ID"`
	EventType string         `json:"event_type" gorm:"type:enum('up','down');not null"`
	EventTime time.Time      `json:"event_time" gorm:"default:CURRENT_TIMESTAMP(3)"`
	RawData   string         `json:"raw_data" gorm:"type:text"`
	Processed bool           `json:"processed" gorm:"default:false"`
	CreatedAt time.Time      `json:"created_at" gorm:"default:CURRENT_TIMESTAMP(3)"`
}

func (n *NetwatchEvent) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	return nil
}

func (NetwatchEvent) TableName() string {
	return "netwatch_events"
}

// TicketLog represents logs/history for tickets (including Netwatch events)
type TicketLog struct {
	ID        string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TicketID  uint64         `json:"ticket_id" gorm:"not null"`
	Ticket    TroubleTicket  `json:"ticket,omitempty" gorm:"foreignKey:TicketID;references:ID"`
	LogType   string         `json:"log_type" gorm:"type:enum('manual','netwatch','system');default:'manual'"`
	Message   string         `json:"message" gorm:"type:text;not null"`
	EventID   *string        `json:"event_id" gorm:"type:varchar(36)"`
	Event     *NetwatchEvent `json:"event,omitempty" gorm:"foreignKey:EventID;references:ID"`
	CreatedBy *string        `json:"created_by" gorm:"type:varchar(36)"`
	User      *User          `json:"user,omitempty" gorm:"foreignKey:CreatedBy;references:ID"`
	CreatedAt time.Time      `json:"created_at" gorm:"default:CURRENT_TIMESTAMP(3)"`
}

func (t *TicketLog) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

func (TicketLog) TableName() string {
	return "ticket_logs"
}

// NetwatchConfig holds MikroTik connection settings
type NetwatchConfig struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name      string    `json:"name" gorm:"type:varchar(255);not null"`
	Host      string    `json:"host" gorm:"type:varchar(255);not null"`
	Port      int       `json:"port" gorm:"default:8728"`
	Username  string    `json:"username" gorm:"type:varchar(255);not null"`
	Password  string    `json:"password" gorm:"type:varchar(255);not null"`
	UseSSL    bool      `json:"use_ssl" gorm:"default:false"`
	Active    bool      `json:"active" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt time.Time `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP(3)"`
}

func (n *NetwatchConfig) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	return nil
}

func (NetwatchConfig) TableName() string {
	return "netwatch_configs"
}
