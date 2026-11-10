package servicerequests

import (
	"strings"

	"skripsi-be/internal/api/common/validation"
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
)

// AdminServiceRequestHandler exposes the approval queue for the CRM team.
type AdminServiceRequestHandler struct {
	service ServiceRequestServiceInterface
}

func NewAdminServiceRequestHandler(service ServiceRequestServiceInterface) *AdminServiceRequestHandler {
	return &AdminServiceRequestHandler{service}
}

// Pending handles GET /api/admin/service-requests
func (h *AdminServiceRequestHandler) Pending(c *fiber.Ctx) error {
	items, err := h.service.Pending()
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Antrean pengajuan layanan", items)
}

// Review handles POST /api/admin/service-requests/:id/review
func (h *AdminServiceRequestHandler) Review(c *fiber.Ctx) error {
	adminID, _ := c.Locals("user_id").(string)
	if adminID == "" {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Tidak terotorisasi", nil)
	}

	request := new(ReviewServiceRequestRequest)
	if err := c.BodyParser(request); err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	if errorMessages := validation.ValidationRequest(request); errorMessages != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, strings.Join(errorMessages, ", "), nil)
	}

	item, err := h.service.Review(adminID, c.Params("id"), *request)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Pengajuan diproses", item)
}
