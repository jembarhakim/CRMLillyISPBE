package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DetailBarangMasuk represents the detail of incoming goods/purchases
type DetailBarangMasuk struct {
	IdMasuk      string    `json:"id_masuk" gorm:"column:IdMasuk;type:varchar(6);not null"`
	AssetID      string    `json:"asset_id" gorm:"column:asset_id;type:varchar(191);not null"`
	Asset        *Asset    `json:"asset,omitempty" gorm:"foreignKey:AssetID;references:ID"`
	SerialNumber *string   `json:"serial_number" gorm:"column:serial_number;type:varchar(191)"`
	QtyMasuk     int       `json:"qty_masuk" gorm:"column:QtyMasuk;type:int"`
	HargaSatuan  int       `json:"harga_satuan" gorm:"column:HargaSatuan;type:int;not null"`
	SubTotal     int       `json:"sub_total" gorm:"column:SubTotal;type:int;not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)"`
}

func (d *DetailBarangMasuk) TableName() string {
	return "detail_barangmasuk"
}

// BarangMasuk represents the main purchase/incoming goods record
type BarangMasuk struct {
	IdMasuk   string              `json:"id_masuk" gorm:"column:IdMasuk;type:varchar(6);primaryKey"`
	Date      time.Time           `json:"date" gorm:"column:date;type:date"`
	Notes     *string             `json:"notes" gorm:"column:notes;type:text"`
	CreatedBy string              `json:"created_by" gorm:"column:created_by;type:varchar(191);not null"`
	CreatedAt time.Time           `json:"created_at" gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)"`
	Details   []DetailBarangMasuk `json:"details,omitempty" gorm:"foreignKey:IdMasuk;references:IdMasuk"`
}

func (b *BarangMasuk) TableName() string {
	return "barangmasuk"
}

func (b *BarangMasuk) BeforeCreate(tx *gorm.DB) error {
	if b.IdMasuk == "" {
		// Generate a 6-character ID like "BM0001"
		b.IdMasuk = generateBarangMasukID()
	}
	return nil
}

// Note: AssetItem and AssetTransaction models are already defined in user_model.go
// This file contains only the new models specific to inventory management

// TicketAssetTransaction represents transactions for trouble tickets
type TicketAssetTransaction struct {
	ID              string     `json:"id" gorm:"column:id;type:varchar(191);primaryKey"`
	TroubleTicketID uint64     `json:"trouble_ticket_id" gorm:"column:trouble_ticket_id;type:bigint unsigned;not null"`
	AssetItemID     string     `json:"asset_item_id" gorm:"column:asset_item_id;type:varchar(191);not null"`
	AssetItem       *AssetItem `json:"asset_item,omitempty" gorm:"foreignKey:AssetItemID;references:ID"`
	TransactionType string     `json:"transaction_type" gorm:"column:transaction_type;type:enum('out','in');not null"`
	Notes           *string    `json:"notes" gorm:"column:notes;type:text"`
	CreatedBy       string     `json:"created_by" gorm:"column:created_by;type:varchar(191);not null"`
	CreatedAt       time.Time  `json:"created_at" gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)"`
}

func (tat *TicketAssetTransaction) TableName() string {
	return "ticket_asset_transactions"
}

func (tat *TicketAssetTransaction) BeforeCreate(tx *gorm.DB) error {
	if tat.ID == "" {
		tat.ID = uuid.New().String()
	}
	return nil
}

// Helper function to generate BarangMasuk ID
func generateBarangMasukID() string {
	// This should be implemented to generate unique IDs like "BM0001", "BM0002", etc.
	// For now, we'll use UUID and truncate to 6 characters
	return "BM" + uuid.New().String()[:4]
}
