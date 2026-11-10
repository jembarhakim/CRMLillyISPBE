package servicerequests

import (
	"strings"

	"skripsi-be/internal/api/common/validation"
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
)

type ServiceRequestHandlerStruct struct {
	service ServiceRequestServiceInterface
}

func NewServiceRequestHandler(service ServiceRequestServiceInterface) *ServiceRequestHandlerStruct {
	return &ServiceRequestHandlerStruct{service}
}

func customerIDFromContext(c *fiber.Ctx) string {
	if v, ok := c.Locals("user_id").(string); ok {
		return v
	}
	return ""
}

// Page handles GET /api/customer/service-requests
// Returns the current package summary, available plans and request history.
func (h *ServiceRequestHandlerStruct) Page(c *fiber.Ctx) error {
	customerID := customerIDFromContext(c)
	if customerID == "" {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Tidak terotorisasi", nil)
	}

	page, err := h.service.Page(customerID)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Layanan Saya", page)
}

// Create handles POST /api/customer/service-requests
func (h *ServiceRequestHandlerStruct) Create(c *fiber.Ctx) error {
	customerID := customerIDFromContext(c)
	if customerID == "" {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Tidak terotorisasi", nil)
	}

	request := new(CreateServiceRequestRequest)
	if err := c.BodyParser(request); err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	if errorMessages := validation.ValidationRequest(request); errorMessages != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, strings.Join(errorMessages, ", "), nil)
	}

	item, err := h.service.Create(customerID, *request)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusCreated, true, "Pengajuan berhasil dikirim dan menunggu persetujuan", item)
}

// History handles GET /api/customer/service-requests/history
func (h *ServiceRequestHandlerStruct) History(c *fiber.Ctx) error {
	customerID := customerIDFromContext(c)
	if customerID == "" {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Tidak terotorisasi", nil)
	}

	items, err := h.service.List(customerID)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Riwayat pengajuan", items)
}
