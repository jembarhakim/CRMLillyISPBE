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
