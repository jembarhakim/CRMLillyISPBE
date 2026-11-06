package entities

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NullableTimeFromVarchar is a custom type that can scan VARCHAR to time.Time
type NullableTimeFromVarchar struct {
	Time  *time.Time
	Valid bool
}

// Scan implements the sql.Scanner interface for NullableTimeFromVarchar
func (nt *NullableTimeFromVarchar) Scan(value interface{}) error {
	if value == nil {
		nt.Time, nt.Valid = nil, false
		return nil
	}

	nt.Valid = true
	switch v := value.(type) {
	case time.Time:
		nt.Time = &v
		return nil
	case []byte:
		// Handle VARCHAR as []byte
		if len(v) == 0 {
			nt.Time, nt.Valid = nil, false
			return nil
		}
		str := string(v)
		if str == "" || str == "NULL" {
			nt.Time, nt.Valid = nil, false
			return nil
		}
		// Try multiple date formats
		formats := []string{
			"2006-01-02 15:04:05",
			time.RFC3339,
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02 15:04:05.000",
			"2006-01-02T15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, str); err == nil {
				nt.Time = &t
				return nil
			}
		}
		return fmt.Errorf("cannot parse %q as time", str)
	case string:
		if v == "" || v == "NULL" {
			nt.Time, nt.Valid = nil, false
			return nil
		}
		// Try multiple date formats
		formats := []string{
			"2006-01-02 15:04:05",
			time.RFC3339,
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02 15:04:05.000",
			"2006-01-02T15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				nt.Time = &t
				return nil
			}
		}
		return fmt.Errorf("cannot parse %q as time", v)
	default:
		return fmt.Errorf("cannot scan %T into NullableTimeFromVarchar", value)
	}
}

// Value implements the driver.Valuer interface for NullableTimeFromVarchar
func (nt NullableTimeFromVarchar) Value() (driver.Value, error) {
	if !nt.Valid || nt.Time == nil {
		return nil, nil
	}
	// Return as string for VARCHAR column
	return nt.Time.Format("2006-01-02 15:04:05"), nil
}

// MarshalJSON implements json.Marshaler for NullableTimeFromVarchar
func (nt NullableTimeFromVarchar) MarshalJSON() ([]byte, error) {
	if !nt.Valid || nt.Time == nil {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf(`"%s"`, nt.Time.Format(time.RFC3339))), nil
}

// UnmarshalJSON implements json.Unmarshaler for NullableTimeFromVarchar
func (nt *NullableTimeFromVarchar) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || string(data) == `""` {
		nt.Time, nt.Valid = nil, false
		return nil
	}

	// Remove quotes
	str := string(data)
	if len(str) > 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	// Try multiple date formats
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05.000",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			nt.Time = &t
			nt.Valid = true
			return nil
		}
	}
	return fmt.Errorf("cannot parse %q as time", str)
}

// Enums
type UserRole string

const (
	RoleAdmin           UserRole = "ADMIN"
	RoleSuperAdmin      UserRole = "SUPERADMIN"
	RoleTechnician      UserRole = "TECHNICIAN"
	RoleFinance         UserRole = "FINANCE"
	RoleCustomerService UserRole = "CUSTOMER_SERVICE"
)

// Accounts model
type Accounts struct {
	ID        string    `json:"id" gorm:"column:id;primaryKey"`
	Name      string    `json:"name" gorm:"column:name"`
	Saldo     int64     `json:"saldo" gorm:"column:saldo;default:0"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:createdAt;type:datetime(3);default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updatedAt;type:datetime(3);default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)"`
}

func (c *Accounts) TableName() string {
	return "accounts"
}

func (u *Accounts) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

// CustomerInstallation model moved to customer_installation_model.go

// Assets model - Catalog only, no company_id or site (these belong to asset_items)
type Asset struct {
	ID           string             `json:"id" gorm:"primaryKey"`
	Type         string             `json:"type" validate:"required"`
	Brand        string             `json:"brand" validate:"required"`
	Model        string             `json:"model" validate:"required"`
	SerialNumber string             `json:"serial_number" validate:"required"`
	Date         string             `json:"date" validate:"required"`
	Price        float64            `json:"price" validate:"required"`
	Description  string             `json:"description"`
	CreatedAt    time.Time          `json:"createdAt" gorm:"column:createdAt;default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt    time.Time          `json:"updatedAt" gorm:"column:updatedAt;"`
	ReportAssets *ReportAssets `json:"report_assets" gorm:"foreignKey:ID;constraint:false"`
	AssetItems   []AssetItem        `json:"asset_items,omitempty" gorm:"foreignKey:AssetID;references:ID"`
	Transactions []AssetTransaction `json:"transactions,omitempty" gorm:"foreignKey:AssetID;references:ID"`
}

func (c *Asset) TableName() string {
	return "assets"
}

func (u *Asset) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

// AssetItem model for tracking individual asset items (instances with company and site)
type AssetItem struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	AssetID      string    `json:"asset_id" gorm:"type:varchar(191);not null"`
	Asset        *Asset    `json:"asset,omitempty" gorm:"foreignKey:AssetID;references:ID"`
	MacAddress   string    `json:"mac_address" gorm:"type:varchar(17);uniqueIndex;not null"`
	SerialNumber *string   `json:"serial_number" gorm:"type:varchar(191)"`
	MacSticker   *string   `json:"mac_sticker" gorm:"type:varchar(191)"`
	Status       string    `json:"status" gorm:"type:enum('in_stock','in_use','maintenance','damaged','retired');default:'in_stock'"`
	CompanyID    *string   `json:"company_id" gorm:"type:varchar(191)"`
	Company      *Company  `json:"company,omitempty" gorm:"foreignKey:CompanyID;references:ID"`
	Site         string    `json:"site" gorm:"type:varchar(191)"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:updated_at;default:CURRENT_TIMESTAMP(3)"`
}

func (ai *AssetItem) TableName() string {
	return "asset_items"
}

func (ai *AssetItem) BeforeCreate(tx *gorm.DB) error {
	if ai.ID == "" {
		ai.ID = uuid.New().String()
	}
	return nil
}

// Company model
type Company struct {
	ID          string     `json:"id" gorm:"primaryKey"`
	Name        string     `json:"name"`
	URL         string     `json:"url"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	LogoURL     string     `json:"logo_url"`
	Description string     `json:"description"`
	Npwp        string     `json:"npwp"`
	Address     string     `json:"address"`
	CreatedAt   time.Time  `json:"createdAt" gorm:"column:createdAt; default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt   time.Time  `json:"updatedAt"  gorm:"column:updatedAt" `
	Customers   []Customer `json:"customers" gorm:"foreignKey:CompanyID;references:ID"`
}

func (c *Company) TableName() string {
	return "company"
}

func (c *Company) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// Customer model
type Customer struct {
	ID                    string     `json:"id" gorm:"primaryKey"`
	Address               string     `gorm:"column:address" json:"address"`
	AreaID                string `gorm:"column:area_id;type:varchar(191);index:idx_customer_area_id,length:191" json:"area_id"`
	Area                  *Areas     `gorm:"foreignKey:AreaID" json:"area"`
	Latitude              float64    `gorm:"column:latitude" json:"latitude"`
	Longitude             float64    `gorm:"column:longitude" json:"longitude"`
	Name                  string     `gorm:"column:name" json:"name"`
	Alias                 string     `gorm:"column:alias" json:"alias"`
	Phone                 string     `gorm:"column:phone" json:"phone"`
	Password              string     `gorm:"column:password" json:"password"`
	ServiceRequestDate    string     `gorm:"column:service_request_date;type:date" json:"service_request_date"`
	CreatedAt             time.Time  `gorm:"column:createdAt;autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time  `gorm:"column:updatedAt;autoUpdateTime" json:"updated_at"`
	DeletedAt             *time.Time `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
	InstallationDate      time.Time  `gorm:"column:installation_date;type:date" json:"installation_date"`
	SalesRepresentativeID string `gorm:"column:sales_representative_id;type:varchar(191);index:idx_customer_sales_rep_id,length:191" json:"sales_representative_id"`
	SalesRepresentative   *User      `gorm:"foreignKey:SalesRepresentativeID" json:"sales_representative"`
	CompanyID             string `gorm:"column:company_id;type:varchar(191);index:idx_customer_company_id,length:191" json:"company_id"`
	Company               *Company   `gorm:"foreignKey:CompanyID" json:"company"`
	ProductID             string     `gorm:"column:product_id;type:varchar(191);index:idx_customer_product_id,length:191" json:"product_id"`
	Product               *Products  `gorm:"foreignKey:ProductID" json:"product"`
	IsInternet            string     `gorm:"column:is_internet;default:yes" json:"is_internet"`
	IsCollaborator        string     `gorm:"column:is_collaborator;default:no" json:"is_collaborator"`

	// One-to-Many relationship with CustomerInstallation
	InstallationReports []CustomerInstallation `gorm:"foreignKey:CustomerID;references:ID" json:"installation_reports,omitempty"`
}

func (u *Customer) TableName() string {
	return "customer"
}
func (c *Customer) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
		c.InstallationDate = time.Now()
	}
	return nil
}

// Device model
type Device struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt" gorm:"default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Groups model
type Areas struct {
	ID              string     `json:"id" gorm:"primaryKey"`
	NameCity        string     `json:"name_city"`
	NameSubdistrict string     `json:"name_subdistrict"`
	NameVillage     string     `json:"name_village"`
	CodeName        string     `json:"code_name" gorm:"column:code_name"`
	CreatedAt       time.Time  `json:"createdAt" gorm:"column:createdAt;default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt       time.Time  `json:"updatedAt" gorm:"column:updatedAt;"`
	Customers       []Customer `json:"customer" gorm:"foreignKey:area_id"`
}

func (u *Areas) TableName() string {
	return "areas"
}
func (u *Areas) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

// Log model
type Log struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	UserID    string    `json:"user_id" gorm:"index:user_id"`
	Action    string    `json:"action"`
	CreatedAt time.Time `json:"createdAt" gorm:"default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt time.Time `json:"updatedAt"`
	User      User      `json:"user" gorm:"foreignKey:UserID;constraint:OnUpdate:RESTRICT"`
}

// Products model
type Products struct {
	ID                string    `json:"id" gorm:"primaryKey"`
	Name              string    `json:"name"`
	Price             int64     `json:"price"`                                                 // BigInt mapped to int64
	Description       string    `json:"description"`                                           // Used as comment for MikroTik provisioning
	DownloadSpeedMbps *int      `json:"download_speed_mbps" gorm:"column:download_speed_mbps"` // Nullable INT field
	UploadSpeedMbps   *int      `json:"upload_speed_mbps" gorm:"column:upload_speed_mbps"`     // Nullable INT field
	CreatedAt         time.Time `json:"createdAt" gorm:"default:CURRENT_TIMESTAMP(3); column:createdAt"`
	UpdatedAt         time.Time `json:"updatedAt" gorm:"column:updatedAt"`
}

func (u *Products) TableName() string {
	return "products"
}
func (p *Products) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

// ReportAssets model
type ReportAssets struct {
	ID          string    `json:"id" gorm:"primaryKey;index:report_assets_ibfk_1"`
	Description string    `json:"description"`
	Quantity    int64     `json:"quantity"` // BigInt mapped to int64
	CreatedAt   time.Time `json:"createdAt" gorm:"default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Assets []Asset `json:"assets" gorm:"foreignKey:ID;constraint:false"`
}

// ReportCash model
type ReportCash struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	CreatedAt   time.Time `json:"createdAt" gorm:"default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Credit      int64     `json:"credit"` // BigInt mapped to int64
	Debit       int64     `json:"debit"`  // BigInt mapped to int64
	Description string    `json:"description"`
}

// Transactions model
// TransactionsTypeCash represents the type of cash transaction
type TransactionsTypeCash string

// Constants for TransactionsTypeCash
const (
	TransactionsTypeCashInternet     TransactionsTypeCash = "internet"
	TransactionsTypeCashCashFlow     TransactionsTypeCash = "cash_flow"
	TransactionsTypeCashCollaborator TransactionsTypeCash = "collaborator"
)

// TransactionsTypeInOut represents the direction of the transaction
type TransactionsTypeInOut string

// Constants for TransactionsTypeInOut
const (
	TransactionsTypeInOutIn  TransactionsTypeInOut = "debit"
	TransactionsTypeInOutOut TransactionsTypeInOut = "credit"
)

// TransactionsType represents the general transaction type
type TransactionsType string

type Transaction struct {
	ID          string                `json:"id" gorm:"column:id;primaryKey;type:varchar(191)"`
	AccountID   string                `json:"account_id" gorm:"column:account_id;index:transactions_account_id_fkey;type:varchar(191)"`
	InvoiceID   string                `json:"invoice_id" gorm:"column:invoice_id;index:transactions_invoice_id_fkey;type:varchar(191)"`
	TypeCash    TransactionsTypeCash  `json:"type_cash" gorm:"column:type_cash;type:varchar(191)"`
	TypeInOut   TransactionsTypeInOut `json:"type_in_out" gorm:"column:type_in_out;type:varchar(191)"`
	Date        string                `json:"date" gorm:"column:date;type:datetime"`
	Description string                `json:"description" gorm:"column:description;type:varchar(191)"`
	Amount      int64                 `json:"amount" gorm:"column:amount;type:bigint"`
	Category    string                `json:"category" gorm:"column:category;type:varchar(191)"`
	Method      string                `json:"method" gorm:"column:method;type:varchar(191)"`
	CreatedAt   time.Time             `json:"createdAt" gorm:"column:createdAt;default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt   time.Time             `json:"updatedAt" gorm:"column:updatedAt;type:datetime"`
	Account     Accounts              `json:"account" gorm:"foreignKey:AccountID;constraint:OnUpdate:RESTRICT"`
}

func (u *Transaction) TableName() string {
	return "transactions"
}
func (u *Transaction) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
		u.Date = time.Now().Format("2006-01-02 15:04:05")
	}
	return nil
}

// Transfers model
type Transfers struct {
	ID            string    `json:"id" gorm:"primaryKey"`
	FromAccountID string    `json:"from_account_id" gorm:"index:transfers_from_account_id_fkey"`
	ToAccountID   string    `json:"to_account_id" gorm:"index:transfers_to_account_id_fkey"`
	Date          time.Time `json:"date"`
	Description   string    `json:"description"`
	Amount        int64     `json:"amount"` // BigInt mapped to int64
	Tags          string    `json:"tags"`
	CreatedAt     time.Time `json:"createdAt" gorm:"default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt     time.Time `json:"updatedAt"`
	FromAccount   Accounts  `json:"accounts_transfers_from_account_idToaccounts" gorm:"foreignKey:FromAccountID;constraint:OnUpdate:RESTRICT"`
	ToAccount     Accounts  `json:"accounts_transfers_to_account_idToaccounts" gorm:"foreignKey:ToAccountID;constraint:OnUpdate:RESTRICT"`
}

// User model
type User struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email" gorm:"unique"`
	Name      string    `json:"name" gorm:"default:null"`
	Password  string    `json:"password"`
	RoleId    string    `gorm:"column:role_id;type:varchar(191);index:idx_users_role_id,length:191"`
	Role      Role      `gorm:"foreignKey:RoleId"`
	Token     string    `json:"token" gorm:"default:null"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"createdAt" gorm:"default:CURRENT_TIMESTAMP(3); column:createdAt"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updatedAt"`
	Log       []Log     `json:"log" gorm:"foreignKey:UserID"`
}

func (u *User) TableName() string {
	return "users"
}
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

type Role struct {
	ID              string           `json:"id" gorm:"primaryKey"`
	Name            string           `json:"name" gorm:"default:null"`
	RolePermissions []RolePermission `json:"role_permissions" gorm:"foreignKey:RoleID"`
	CreatedAt       time.Time        `json:"createdAt" gorm:"default:CURRENT_TIMESTAMP(3); column:createdAt"`
	UpdatedAt       time.Time        `json:"updatedAt" gorm:"column:updatedAt"`
}

func (u *Role) TableName() string {
	return "roles"
}
func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}

type Image struct {
	ID                    string     `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	File                  string     `gorm:"column:file;type:varchar(191);not null" json:"file"`
	FullPath              string     `gorm:"column:full_path;type:varchar(191);not null" json:"full_path"`
	ArchiveInstallationId string     `gorm:"column:archive_installation_id;type:varchar(191);not null" json:"archive_installation_id"`
	CreatedAt             time.Time  `gorm:"column:createdAt;type:datetime(3);not null;default:CURRENT_TIMESTAMP(3)" json:"createdAt"`
	UpdatedAt             *time.Time `gorm:"column:updatedAt;default:null" json:"updatedAt"`
}

func (u *Image) TableName() string {
	return "images"
}

func (u *Image) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

// AssetTransaction model untuk tracking aset keluar/masuk
type AssetTransaction struct {
	ID                     string                `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	CustomerInstallationID string                `gorm:"column:customer_installation_id;type:varchar(191);index:idx_asset_transactions_customer_installation_id" json:"customer_installation_id"`
	AssetItemID            *string               `gorm:"column:asset_item_id;type:varchar(191);index:idx_asset_transactions_asset_item_id" json:"asset_item_id,omitempty"`
	AssetID                string                `gorm:"column:asset_id;type:varchar(191);index:idx_asset_transactions_asset_id" json:"asset_id"`
	TransactionType        string                `gorm:"column:transaction_type;type:enum('out','in')" json:"transaction_type"`
	Quantity               int                   `gorm:"column:quantity;type:int;default:1" json:"quantity"`
	Notes                  *string               `gorm:"column:notes;type:text" json:"notes,omitempty"`
	TransactionDate        time.Time             `gorm:"column:transaction_date;type:datetime(3);default:CURRENT_TIMESTAMP(3)" json:"transaction_date"`
	CreatedBy              string                `gorm:"column:created_by;type:varchar(191);index:idx_asset_transactions_created_by" json:"created_by"`
	CreatedAt              time.Time             `gorm:"column:createdAt;default:CURRENT_TIMESTAMP(3)" json:"createdAt"`
	UpdatedAt              time.Time             `gorm:"column:updatedAt;default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)" json:"updatedAt"`
	CustomerInstallation   *CustomerInstallation `gorm:"foreignKey:CustomerInstallationID;references:ID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"customer_installation,omitempty"`
	AssetItem              *AssetItem            `gorm:"foreignKey:AssetItemID;references:ID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"asset_item,omitempty"`
	Asset                  *Asset                `gorm:"foreignKey:AssetID;references:ID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"asset,omitempty"`
	User                   *User                 `gorm:"foreignKey:CreatedBy;references:ID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"user,omitempty"`
}

func (a *AssetTransaction) TableName() string {
	return "asset_transactions"
}

func (a *AssetTransaction) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// TransactionsTypeInOut represents the direction of the transaction
type InvoiceStatus string

// Constants for TransactionsTypeInOut
const (
	InvoiceStatusPaid    InvoiceStatus = "paid"
	InvoiceStatusUnpaid  InvoiceStatus = "unpaid"
	InvoiceStatusPending InvoiceStatus = "pending"
)

type Invoice struct {
	ID            string                  `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	Amount        int64                   `gorm:"column:amount;type:int;not null" json:"amount"`
	CustomerID    string                  `gorm:"column:customer_id;type:varchar(191);not null" json:"customer_id"`
	Customer      Customer                `gorm:"foreignKey:CustomerID;references:id;constraint:OnUpdate:RESTRICT" json:"customer"`
	Link          string                  `gorm:"column:link;type:varchar(191);not null" json:"link"`
	Status        InvoiceStatus           `gorm:"column:status;type:varchar(191);not null" json:"status"`
	InvoiceDate   *time.Time              `gorm:"column:invoice_date;type:date" json:"invoice_date"`
	DueDate       *time.Time              `gorm:"column:due_date;type:date" json:"due_date"`
	PdfViewed     bool                    `gorm:"column:pdf_viewed;type:boolean;default:false" json:"pdf_viewed"`
	PdfViewedAt   NullableTimeFromVarchar `gorm:"column:pdf_viewed_at;type:varchar(50)" json:"pdf_viewed_at"`
	CreatedAt     time.Time               `gorm:"column:createdAt;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt     time.Time               `gorm:"column:updatedAt;not null" json:"updated_at"`
	InvoiceItems  []InvoiceItems          `gorm:"foreignKey:InvoiceID;constraint:OnUpdate:RESTRICT" json:"invoice_items"`
	Transaction   Transaction             `gorm:"foreignKey:invoice_id;constraint:OnUpdate:RESTRICT" json:"transaction"`
	PendingReason *string                 `gorm:"-" json:"pending_reason,omitempty"` // Virtual field, not stored in DB
}

// InvoicePendingReason stores customer's reason when invoice is pending
type InvoicePendingReason struct {
	ID        string    `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	InvoiceID string    `gorm:"column:invoice_id;type:varchar(191);not null" json:"invoice_id"`
	Reason    string    `gorm:"column:reason;type:text;not null" json:"reason"`
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

func (InvoicePendingReason) TableName() string { return "invoice_pending_reasons" }

func (Invoice) TableName() string {
	return "invoices"
}
func (u *Invoice) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
		u.Status = InvoiceStatusUnpaid
	}

	return nil
}

type InvoiceItems struct {
	ID        string    `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	Name      string    `gorm:"column:name;type:varchar(191);not null" json:"name"`
	Qty       int64     `gorm:"column:qty;type:int;not null" json:"qty"`
	Price     int64     `gorm:"column:price;type:int;not null" json:"price"`
	Total     int64     `gorm:"column:total;type:int;not null" json:"total"`
	InvoiceID string    `gorm:"column:invoices_id;type:varchar(191);not null" json:"invoice_id"`
	Invoice   Invoice   `gorm:"foreignKey:InvoiceID;references:id" json:"invoice"`
	CreatedAt time.Time `gorm:"column:createdAt;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updatedAt;not null" json:"updated_at"`
}

func (InvoiceItems) TableName() string {
	return "invoice_items"
}

// Feature model
type Feature struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"column:name"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:createdAt;default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updatedAt"`
}

func (f *Feature) TableName() string {
	return "features"
}

func (f *Feature) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.New().String()
	}
	return nil
}

// RolePermission model
type RolePermission struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	RoleID string `json:"role_id" gorm:"column:role_id;type:varchar(191);index:idx_role_permissions_role_id,length:191"`
	FeatureID string    `json:"feature_id" gorm:"column:feature_id;type:varchar(191)"`
	CanAccess int       `json:"can_access" gorm:"column:can_access"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:createdAt;default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updatedAt"`
}

func (rp *RolePermission) TableName() string {
	return "role_permissions"
}

func (rp *RolePermission) BeforeCreate(tx *gorm.DB) error {
	if rp.ID == "" {
		rp.ID = uuid.New().String()
	}
	return nil
}

// InstallationHistory model for tracking deleted installation reports
type InstallationHistory struct {
	ID                string                `gorm:"column:id;type:varchar(191);primaryKey" json:"id"`
	InstallationID    string                `gorm:"column:installation_id;type:varchar(191);not null;index:idx_installation_history_installation_id" json:"installation_id"`
	OldIP             *string               `gorm:"column:old_ip;type:varchar(191);default:null" json:"old_ip,omitempty"`
	OldMac            *string               `gorm:"column:old_mac;type:varchar(191);default:null" json:"old_mac,omitempty"`
	ChangeReason      string                `gorm:"column:change_reason;type:enum('router_broken','upgrade','terminated');default:'terminated'" json:"change_reason"`
	TicketID          uint                  `gorm:"column:ticket_id;not null" json:"ticket_id"`
	CreatedAt         time.Time             `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	Installation      *CustomerInstallation `gorm:"foreignKey:InstallationID;references:ID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"installation,omitempty"`
	TroubleTicket     *TroubleTicket        `gorm:"foreignKey:TicketID;references:ID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"trouble_ticket,omitempty"`
}

func (i *InstallationHistory) TableName() string {
	return "installation_history"
}

func (i *InstallationHistory) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.New().String()
	}
	return nil
}

// Removed BeforeCreate hook to prevent UUID conflicts
// UUID generation is now handled explicitly in the repository layer
