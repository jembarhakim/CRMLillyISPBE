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
