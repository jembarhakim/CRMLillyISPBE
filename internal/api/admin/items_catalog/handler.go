package items_catalog

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"skripsi-be/internal/api/common/validation"
	"skripsi-be/internal/helpers"
)

type ItemHandlerStruct struct {
	service ItemServiceInterface
}

func NewItemHandler(service ItemServiceInterface) *ItemHandlerStruct {
	return &ItemHandlerStruct{service}
}

func (h ItemHandlerStruct) GetAllItemsHandler(c *fiber.Ctx) error {
	request := GetItemsRequest{}

	// Parse query parameters
	if category := c.Query("category"); category != "" {
		request.Category = &category
	}
	if assetID := c.Query("asset_id"); assetID != "" {
		request.AssetID = &assetID
	}

	items, err := h.service.GetAllItemsService(request)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success Get Data", items)
}

func (h ItemHandlerStruct) GetByIdItemHandler(c *fiber.Ctx) error {
	request := IdItemRequest{}
	request.Id = c.Params("id")

	errValidation := validation.ValidationRequest(request)
	if errValidation != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, strings.Join(errValidation, ", "), nil)
	}

	item, err := h.service.GetByIdItemService(request)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success Get Data", item)
}

func (h ItemHandlerStruct) CreateItemHandler(c *fiber.Ctx) error {
	request := CreateItemRequest{}
	err := c.BodyParser(&request)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	errValidation := validation.ValidationRequest(&request)
	if errValidation != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, strings.Join(errValidation, ", "), nil)
	}

	item, err := h.service.CreateItemService(request)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success create item", item)
}

func (h ItemHandlerStruct) UpdateItemHandler(c *fiber.Ctx) error {
	request := UpdateItemRequest{
		IdItemRequest: IdItemRequest{
			Id: c.Params("id"),
		},
	}

	err := c.BodyParser(&request.CreateItemRequest)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	errValidation := validation.ValidationRequest(&request)
	if errValidation != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, strings.Join(errValidation, ", "), nil)
	}

	item, err := h.service.UpdateItemService(request)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success update item", item)
}

func (h ItemHandlerStruct) DeleteItemHandler(c *fiber.Ctx) error {
	request := IdItemRequest{
		Id: c.Params("id"),
	}

	errValidation := validation.ValidationRequest(&request)
	if errValidation != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, strings.Join(errValidation, ", "), nil)
	}

	item, err := h.service.DeleteItemService(request)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success delete item", item)
}

