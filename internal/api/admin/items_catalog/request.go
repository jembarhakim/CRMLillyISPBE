package items_catalog

type CreateItemRequest struct {
	Name         string  `json:"name" validate:"required"`
	DefaultUnit  string  `json:"default_unit" validate:"required,oneof=PCS M KG BOX ROLL"`
	Category     *string `json:"category"`
	Description  *string `json:"description"`
	AssetID      *string `json:"asset_id"`
	MinimumStock *int    `json:"minimum_stock" validate:"omitempty,gte=0"`
}

type IdItemRequest struct {
	Id string `json:"id" validate:"required"`
}

type UpdateItemRequest struct {
	IdItemRequest
	CreateItemRequest
}

type GetItemsRequest struct {
	Category *string `json:"category"`
	AssetID  *string `json:"asset_id"`
}
