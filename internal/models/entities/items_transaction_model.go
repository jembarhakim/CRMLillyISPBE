package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ItemsKeluar (Items OUT) - Parent transaction for items going out.
// HolderID is the holder the goods leave from (NULL = main warehouse, which is also what every
// row written by the legacy POST / endpoint means). TransferRef links the row to its incoming
// counterpart through a shared 32-char hex id and is NULL when the row is not part of a transfer.
type ItemsKeluar struct {
	ID          string              `gorm:"column:id;type:varchar(6);primaryKey" json:"id"`
	Date        time.Time           `gorm:"column:date;type:date;not null" json:"date"`
	Notes       *string             `gorm:"column:notes;type:text" json:"notes,omitempty"`
	HolderID    *string             `gorm:"column:holder_id;type:varchar(191);index" json:"holder_id,omitempty"`
	TransferRef *string             `gorm:"column:transfer_ref;type:char(32);index" json:"transfer_ref,omitempty"`
	CreatedBy   string              `gorm:"column:created_by;type:varchar(191);index" json:"created_by"`
	CreatedAt   time.Time           `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time           `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
	Details     []DetailItemsKeluar `gorm:"foreignKey:IdKeluar;references:ID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"details,omitempty"`
	User        *User               `gorm:"foreignKey:CreatedBy;references:ID" json:"user,omitempty"`
}

func (i *ItemsKeluar) TableName() string {
	return "itemskeluar"
}

func (i *ItemsKeluar) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		// Generate short ID like "IK0001"
		i.ID = generateItemsTransactionID("IK")
	}
	return nil
}

// ItemsMasuk (Items IN) - Parent transaction for items coming in.
// HolderID is the holder that receives the goods (NULL = main warehouse, which is also what every
// row written by the legacy POST / endpoint means). TransferRef links the row to its outgoing
// counterpart through a shared 32-char hex id and is NULL when the row is not part of a transfer.
type ItemsMasuk struct {
	ID          string             `gorm:"column:id;type:varchar(6);primaryKey" json:"id"`
	Date        time.Time          `gorm:"column:date;type:date;not null" json:"date"`
	Notes       *string            `gorm:"column:notes;type:text" json:"notes,omitempty"`
	HolderID    *string            `gorm:"column:holder_id;type:varchar(191);index" json:"holder_id,omitempty"`
	TransferRef *string            `gorm:"column:transfer_ref;type:char(32);index" json:"transfer_ref,omitempty"`
	CreatedBy   string             `gorm:"column:created_by;type:varchar(191);index" json:"created_by"`
	CreatedAt   time.Time          `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time          `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
	Details     []DetailItemsMasuk `gorm:"foreignKey:IdMasuk;references:ID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"details,omitempty"`
	User        *User              `gorm:"foreignKey:CreatedBy;references:ID" json:"user,omitempty"`
}

func (i *ItemsMasuk) TableName() string {
	return "itemsmasuk"
}

func (i *ItemsMasuk) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		// Generate short ID like "IM0001"
		i.ID = generateItemsTransactionID("IM")
	}
	return nil
}

// Item - Item catalog (master data)
type Item struct {
	ID          string    `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	Name        string    `gorm:"column:name;type:varchar(191);uniqueIndex;not null" json:"name"`
	DefaultUnit string    `gorm:"column:default_unit;type:varchar(20);not null;default:'PCS'" json:"default_unit"`
	Category    *string   `gorm:"column:category;type:varchar(100)" json:"category,omitempty"`
	Description *string   `gorm:"column:description;type:text" json:"description,omitempty"`
	AssetID     *string   `gorm:"column:asset_id;type:varchar(191);index" json:"asset_id,omitempty"`
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`

	// Relationships
	Asset *Asset `gorm:"foreignKey:AssetID;references:ID" json:"asset,omitempty"`
}

func (i *Item) TableName() string {
	return "items"
}

func (i *Item) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.New().String()
	}
	return nil
}

// DetailItemsKeluar - Detail records for items going out
type DetailItemsKeluar struct {
	IdKeluar  string    `gorm:"column:IdKeluar;type:varchar(6);index" json:"id_keluar"`
	ItemID    string    `gorm:"column:item_id;type:varchar(191);index" json:"item_id"` // References items catalog (renamed from IdItems)
	QtyKeluar int       `gorm:"column:QtyKeluar;type:int" json:"qty_keluar"`
	Unit      string    `gorm:"column:unit;type:varchar(20);default:'PCS'" json:"unit"` // Unit used in this transaction (can differ from item.default_unit)
	Notes     *string   `gorm:"column:notes;type:text" json:"notes,omitempty"`          // Notes for this specific line item (e.g., "Used for customer ABC", "Damaged")
	CreatedAt time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`

	// Relationships
	ItemsKeluar *ItemsKeluar `gorm:"foreignKey:IdKeluar;references:ID" json:"items_keluar,omitempty"`
	Item        *Item        `gorm:"foreignKey:ItemID;references:ID" json:"item,omitempty"` // Reference to items catalog
}

func (d *DetailItemsKeluar) TableName() string {
	return "detail_itemskeluar"
}

// DetailItemsMasuk - Detail records for items coming in
type DetailItemsMasuk struct {
	IdMasuk     string    `gorm:"column:IdMasuk;type:varchar(6);index" json:"id_masuk"`
	ItemID      string    `gorm:"column:item_id;type:varchar(191);index" json:"item_id"` // References items catalog (renamed from IdItems)
	QtyMasuk    int       `gorm:"column:QtyMasuk;type:int" json:"qty_masuk"`
	HargaSatuan int       `gorm:"column:HargaSatuan;type:int;not null;default:0" json:"harga_satuan"`
	SubTotal    int       `gorm:"column:SubTotal;type:int;not null;default:0" json:"sub_total"`
	Unit        string    `gorm:"column:unit;type:varchar(20);default:'PCS'" json:"unit"` // Unit used in this transaction (can differ from item.default_unit)
	Notes       *string   `gorm:"column:notes;type:text" json:"notes,omitempty"`          // Notes for this specific line item (e.g., "Purchase order #123", "Return from customer")
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`

	// Relationships
	ItemsMasuk *ItemsMasuk `gorm:"foreignKey:IdMasuk;references:ID" json:"items_masuk,omitempty"`
	Item       *Item       `gorm:"foreignKey:ItemID;references:ID" json:"item,omitempty"` // Reference to items catalog
}

func (d *DetailItemsMasuk) TableName() string {
	return "detail_itemsmasuk"
}

// Helper function to generate transaction IDs (IK0001, IM0001, etc.)
func generateItemsTransactionID(prefix string) string {
	// For now, use UUID truncated to 6 characters
	// In production, you might want to use a sequence or auto-increment
	uuid := uuid.New().String()
	return prefix + uuid[:4]
}
