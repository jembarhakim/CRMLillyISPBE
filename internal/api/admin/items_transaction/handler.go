package items_transaction

import (
	"errors"
	"fmt"
	"skripsi-be/internal/api/common/validation"
	"skripsi-be/internal/helpers"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type ItemsTransactionHandlerInterface interface {
	GetAllItemsTransactionHandler(c *fiber.Ctx) error
	GetByIdItemsTransactionHandler(c *fiber.Ctx) error
	CreateItemsTransactionHandler(c *fiber.Ctx) error
	UpdateItemsTransactionHandler(c *fiber.Ctx) error
	DeleteItemsTransactionHandler(c *fiber.Ctx) error
	CreateTransferHandler(c *fiber.Ctx) error
}

type ItemsTransactionHandlerStruct struct {
	service ItemsTransactionServiceInterface
}

func NewItemsTransactionHandler(service ItemsTransactionServiceInterface) ItemsTransactionHandlerInterface {
	return &ItemsTransactionHandlerStruct{service}
}

func (h *ItemsTransactionHandlerStruct) GetAllItemsTransactionHandler(c *fiber.Ctx) error {
	request := GetItemsTransactionsRequest{}

	// Parse query parameters
	if transactionType := c.Query("transaction_type"); transactionType != "" {
		request.TransactionType = &transactionType
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		request.DateFrom = &dateFrom
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		request.DateTo = &dateTo
	}
	if assetID := c.Query("asset_id"); assetID != "" {
		request.AssetID = &assetID
	}

	transactions, err := h.service.GetItemsTransactionsService(request)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success get items transactions", transactions)
}

func (h *ItemsTransactionHandlerStruct) GetByIdItemsTransactionHandler(c *fiber.Ctx) error {
	transactionType := c.Query("transaction_type", "out")
	if transactionType != "out" && transactionType != "in" {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "Invalid transaction_type. Must be 'out' or 'in'", nil)
	}

	request := IdItemsTransactionRequest{
		Id: c.Params("id"),
	}

	transaction, err := h.service.GetByIdItemsTransactionService(request, transactionType)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success get items transaction", transaction)
}

func (h *ItemsTransactionHandlerStruct) CreateItemsTransactionHandler(c *fiber.Ctx) error {
	request := CreateItemsTransactionRequest{}

	if err := c.BodyParser(&request); err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	errValidation := validation.ValidationRequest(&request)
	if errValidation != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, strings.Join(errValidation, ", "), nil)
	}

	// Get the authenticated user ID from the request context
	var createdBy string
	if userID := c.Locals("user_id"); userID != nil {
		createdBy = fmt.Sprintf("%v", userID)
	} else {
		// Fallback to a known admin user ID
		createdBy = "b6cbf8b6-6a2e-45f3-a9ab-e30e5941bf5a"
	}

	transaction, err := h.service.CreateItemsTransactionService(request, createdBy)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success create items transaction", transaction)
}

// CreateTransferHandler records an INV-02 stock transfer between two holders.
// The token owner is only the actor (created_by). There is deliberately no hardcoded fallback
// user here: a request without a user id in the context is rejected.
func (h *ItemsTransactionHandlerStruct) CreateTransferHandler(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || strings.TrimSpace(userID) == "" {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Unauthorized", nil)
	}

	request := CreateTransferRequest{}
	if err := c.BodyParser(&request); err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	errValidation := validation.ValidationRequest(&request)
	if errValidation != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, strings.Join(errValidation, ", "), nil)
	}

	transfer, err := h.service.CreateTransferService(request, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrTransferInvalidDate):
			return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
		case errors.Is(err, ErrTransferSameHolder),
			errors.Is(err, ErrTransferBadSource),
			errors.Is(err, ErrTransferBadDestination),
			errors.Is(err, ErrTransferUnknownItem),
			errors.Is(err, ErrTransferInsufficientStock):
			return helpers.ResponseUtils(c, fiber.StatusUnprocessableEntity, false, err.Error(), nil)
		default:
			return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
		}
	}

	return helpers.ResponseUtils(c, fiber.StatusCreated, true, "Success create stock transfer", transfer)
}

func (h *ItemsTransactionHandlerStruct) UpdateItemsTransactionHandler(c *fiber.Ctx) error {
	request := UpdateItemsTransactionRequest{
		IdItemsTransactionRequest: IdItemsTransactionRequest{
			Id: c.Params("id"),
		},
	}

	if err := c.BodyParser(&request.CreateItemsTransactionRequest); err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	errValidation := validation.ValidationRequest(&request)
	if errValidation != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, strings.Join(errValidation, ", "), nil)
	}

	transaction, err := h.service.UpdateItemsTransactionService(request)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success update items transaction", transaction)
}

func (h *ItemsTransactionHandlerStruct) DeleteItemsTransactionHandler(c *fiber.Ctx) error {
	transactionType := c.Query("transaction_type", "out")
	if transactionType != "out" && transactionType != "in" {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "Invalid transaction_type. Must be 'out' or 'in'", nil)
	}

	request := IdItemsTransactionRequest{
		Id: c.Params("id"),
	}

	err := h.service.DeleteItemsTransactionService(request, transactionType)
	if err != nil {
		// A row that belongs to a transfer pair cannot be removed on its own.
		if errors.Is(err, ErrTransferPairDelete) {
			return helpers.ResponseUtils(c, fiber.StatusConflict, false, err.Error(), nil)
		}
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Success delete items transaction", nil)
}

