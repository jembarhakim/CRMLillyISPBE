package asset_item

import (
	"skripsi-be/internal/api/common/validation"
	"skripsi-be/internal/helpers"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type AssetItemHandlerInterface interface {
	GetAllAssetItemHandler(c *fiber.Ctx) error
	GetByIdAssetItemHandler(c *fiber.Ctx) error
	CreateAssetItemHandler(c *fiber.Ctx) error
	UpdateAssetItemHandler(c *fiber.Ctx) error
	DeleteAssetItemHandler(c *fiber.Ctx) error

	// Router / customer integration
	GetAssetItemUsageHandler(c *fiber.Ctx) error
	GetAssetItemUsageBySerialHandler(c *fiber.Ctx) error
	GetCustomerRoutersHandler(c *fiber.Ctx) error
}

type AssetItemHandlerStruct struct {
	service AssetItemServiceInterface
}

func NewAssetItemHandler(service AssetItemServiceInterface) AssetItemHandlerInterface {
	return &AssetItemHandlerStruct{service}
}

func (h *AssetItemHandlerStruct) GetAllAssetItemHandler(c *fiber.Ctx) error {
	request := GetAssetItemsRequest{}

	// Parse query parameters
	if assetID := c.Query("asset_id"); assetID != "" {
		request.AssetID = &assetID
	}
	if status := c.Query("status"); status != "" {
		request.Status = &status
	}

	items, err := h.service.GetAssetItemsService(request)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success get asset items", items)
}

func (h *AssetItemHandlerStruct) GetByIdAssetItemHandler(c *fiber.Ctx) error {
	request := IdAssetItemRequest{
		Id: c.Params("id"),
	}

	item, err := h.service.GetByIdAssetItemService(request)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success get asset item", item)
}

func (h *AssetItemHandlerStruct) CreateAssetItemHandler(c *fiber.Ctx) error {
	request := CreateAssetItemRequest{}

	if err := c.BodyParser(&request); err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	errValidation := validation.ValidationRequest(&request)
	if errValidation != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, strings.Join(errValidation, ", "), nil)
	}

	item, err := h.service.CreateAssetItemService(request)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success create asset item", item)
}

func (h *AssetItemHandlerStruct) UpdateAssetItemHandler(c *fiber.Ctx) error {
	request := UpdateAssetItemRequest{
		IdAssetItemRequest: IdAssetItemRequest{
			Id: c.Params("id"),
		},
	}

	if err := c.BodyParser(&request.CreateAssetItemRequest); err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	errValidation := validation.ValidationRequest(&request)
	if errValidation != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, strings.Join(errValidation, ", "), nil)
	}

	item, err := h.service.UpdateAssetItemService(request)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success update asset item", item)
}

func (h *AssetItemHandlerStruct) DeleteAssetItemHandler(c *fiber.Ctx) error {
	request := IdAssetItemRequest{
		Id: c.Params("id"),
	}

	item, err := h.service.DeleteAssetItemService(request)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success delete asset item", item)
}

// =============================================================================
// Router / customer integration
// =============================================================================

// GetAssetItemUsageHandler answers: "asset unit ini dipakai oleh customer siapa?"
// GET /api/admin/asset-item/:id/usage
func (h *AssetItemHandlerStruct) GetAssetItemUsageHandler(c *fiber.Ctx) error {
	assetItemID := c.Params("id")
	if assetItemID == "" {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "Asset item ID is required", nil)
	}

	usage, err := h.service.GetAssetItemUsageService(assetItemID)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusNotFound, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success get asset item usage", usage)
}

// GetAssetItemUsageBySerialHandler answers: "router dengan serial ini terpasang di customer siapa?"
// GET /api/admin/asset-item/usage/by-serial?serial_number=XXXX
func (h *AssetItemHandlerStruct) GetAssetItemUsageBySerialHandler(c *fiber.Ctx) error {
	serialNumber := strings.TrimSpace(c.Query("serial_number"))
	if serialNumber == "" {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "Query parameter serial_number is required", nil)
	}

	usages, err := h.service.GetAssetItemUsagesBySerialService(serialNumber)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}

	if len(usages) == 0 {
		return helpers.ResponseUtils(c, fiber.StatusNotFound, false, "No asset item found with serial number "+serialNumber, nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success get asset item usage by serial number", usages)
}

// GetCustomerRoutersHandler answers: "customer ini memakai router apa saja?"
// GET /api/admin/asset-item/customer/:customerId/routers
func (h *AssetItemHandlerStruct) GetCustomerRoutersHandler(c *fiber.Ctx) error {
	customerID := c.Params("customerId")
	if customerID == "" {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "Customer ID is required", nil)
	}

	result, err := h.service.GetCustomerRoutersService(customerID)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusNotFound, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success get customer routers", result)
}
