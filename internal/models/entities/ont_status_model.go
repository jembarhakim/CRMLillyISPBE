package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OntStatus menyimpan status terkini ONT pelanggan (hasil monitoring GenieACS)
type OntStatus struct {
	ID              string     `json:"id" gorm:"primaryKey;type:varchar(191)"`
	CustomerID      string     `json:"customer_id" gorm:"type:varchar(191);uniqueIndex:idx_ont_status_customer_id;not null"`
	Customer        *Customer  `json:"customer,omitempty" gorm:"foreignKey:CustomerID;references:ID;constraint:OnDelete:CASCADE"`
	SerialNumber    string     `json:"serial_number" gorm:"type:varchar(191);index:idx_ont_status_serial;not null"`
	Manufacturer    string     `json:"manufacturer" gorm:"type:varchar(100)"`
	ModelName       string     `json:"model_name" gorm:"type:varchar(100)"`
	IsOnline        bool       `json:"is_online" gorm:"index:idx_ont_status_online"`
	RxPower         *float64   `json:"rx_power" gorm:"type:double"`
	TxPower         *float64   `json:"tx_power" gorm:"type:double"`
	SoftwareVersion string     `json:"software_version" gorm:"type:varchar(100)"`
	IPAddress       string     `json:"ip_address" gorm:"type:varchar(45)"`
	MacAddress      string     `json:"mac_address" gorm:"type:varchar(191)"`
	OpticalStatus   string     `json:"optical_status" gorm:"type:varchar(50);default:'unknown';index:idx_ont_status_optical"` // normal, warning, critical, unknown
	SSID            string     `json:"ssid" gorm:"column:ssid;type:varchar(191)"`
	ClientCount     int        `json:"client_count" gorm:"default:0"`
	LastInform      *time.Time `json:"last_inform"`
	IsStale         bool       `json:"is_stale" gorm:"default:false"` // true jika offline tapi menampilkan data pembacaan terakhir
	CreatedAt       time.Time  `json:"created_at" gorm:"default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)"`
}

func (o *OntStatus) BeforeCreate(tx *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.New().String()
	}
	return nil
}

func (OntStatus) TableName() string {
	return "ont_statuses"
}

// OntStatusHistory menyimpan riwayat pembacaan optik & status untuk visualisasi grafik time-series
type OntStatusHistory struct {
	ID            string    `json:"id" gorm:"primaryKey;type:varchar(191)"`
	CustomerID    string    `json:"customer_id" gorm:"type:varchar(191);index:idx_ont_hist_customer;not null"`
	SerialNumber  string    `json:"serial_number" gorm:"type:varchar(191);index:idx_ont_hist_serial;not null"`
	IsOnline      bool      `json:"is_online"`
	RxPower       *float64  `json:"rx_power" gorm:"type:double"`
	TxPower       *float64  `json:"tx_power" gorm:"type:double"`
	OpticalStatus string    `json:"optical_status" gorm:"type:varchar(50)"`
	RecordedAt    time.Time `json:"recorded_at" gorm:"index:idx_ont_hist_recorded_at;not null"`
}

func (h *OntStatusHistory) BeforeCreate(tx *gorm.DB) error {
	if h.ID == "" {
		h.ID = uuid.New().String()
	}
	if h.RecordedAt.IsZero() {
		h.RecordedAt = time.Now()
	}
	return nil
}

func (OntStatusHistory) TableName() string {
	return "ont_status_histories"
}
