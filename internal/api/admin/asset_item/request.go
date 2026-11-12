package asset_item

type CreateAssetItemRequest struct {
	AssetID      string  `json:"asset_id" validate:"required"`
	MacAddress   string  `json:"mac_address" validate:"required"`
	SerialNumber *string `json:"serial_number"`
	MacSticker   *string `json:"mac_sticker"`
	Status       string  `json:"status" validate:"required,oneof=in_stock in_use maintenance damaged retired"`
	CompanyID    *string `json:"company_id"`
	Site         *string `json:"site"`
}

type IdAssetItemRequest struct {
	Id string `json:"id" validate:"required"`
}

type UpdateAssetItemRequest struct {
	IdAssetItemRequest
	CreateAssetItemRequest
}

type GetAssetItemsRequest struct {
	AssetID *string `json:"asset_id"`
	Status  *string `json:"status"`
}

// =============================================================================
// Asset Assignment / Router usage tracking
// =============================================================================

// AssetItemUsageResponse answers the question "who is using this asset unit?"
type AssetItemUsageResponse struct {
	AssetItemID     string          `json:"asset_item_id"`
	SerialNumber    *string         `json:"serial_number"`
	MacAddress      string          `json:"mac_address"`
	MacSticker      *string         `json:"mac_sticker"`
	Status          string          `json:"status"`
	Brand           string          `json:"brand,omitempty"`
	Type            string          `json:"type,omitempty"`
	Model           string          `json:"model,omitempty"`
	IsUsed          bool            `json:"is_used"`
	Usage           []AssetItemUser `json:"usage"`
}

// AssetItemUser represents a single customer currently using an asset unit
type AssetItemUser struct {
	CustomerID         string  `json:"customer_id"`
	CustomerName       string  `json:"customer_name"`
	CustomerAddress    string  `json:"customer_address"`
	CustomerPhone      string  `json:"customer_phone"`
	NetworkDeviceID    string  `json:"network_device_id"`
	IPStatic           *string `json:"ip_static"`
	OntSerialNumber    *string `json:"ont_serial_number"`
	Kepemilikan        string  `json:"kepemilikan_perangkat"`
	InstallationID     *string `json:"customer_installation_id,omitempty"`
	InstallationStatus string  `json:"installation_status,omitempty"`
}

// CustomerRouterAssignmentResponse answers "which router does this customer use?"
type CustomerRouterAssignmentResponse struct {
	CustomerID   string                `json:"customer_id"`
	CustomerName string                `json:"customer_name"`
	HasRouter    bool                  `json:"has_router"`
	Routers      []CustomerRouterDetail `json:"routers"`
}

// CustomerRouterDetail is a fully resolved router/ONT unit assigned to a customer
type CustomerRouterDetail struct {
	NetworkDeviceID    string  `json:"network_device_id"`
	AssetItemID        *string `json:"asset_item_id,omitempty"`
	Brand              string  `json:"brand,omitempty"`
	Type               string  `json:"type,omitempty"`
	Model              string  `json:"model,omitempty"`
	SerialNumber       string  `json:"serial_number,omitempty"`
	MacAddress         string  `json:"mac_address,omitempty"`
	MacSticker         string  `json:"mac_sticker,omitempty"`
	AssetItemStatus    string  `json:"asset_item_status,omitempty"`
	OntSerialNumber    string  `json:"ont_serial_number,omitempty"`
	Kepemilikan        string  `json:"kepemilikan_perangkat,omitempty"`
	IPStatic           string  `json:"ip_static,omitempty"`
	SwitchID           string  `json:"switch_id,omitempty"`
	PortNumber         string  `json:"port_number,omitempty"`
	InstallationID     *string `json:"customer_installation_id,omitempty"`
	InstallationStatus string  `json:"installation_status,omitempty"`
}

// LookupAssetItemBySerialRequest is a request parameter wrapper for serial based lookup
type LookupAssetItemBySerialRequest struct {
	SerialNumber string `json:"serial_number"`
}
