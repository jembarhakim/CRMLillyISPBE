package tickets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"skripsi-be/internal/api/common/validation"
	"skripsi-be/internal/helpers"

	"github.com/gofiber/fiber/v2"
)

type TicketHandlerStruct struct {
	service TicketServiceInterface
}

func NewTicketHandler(service TicketServiceInterface) *TicketHandlerStruct {
	return &TicketHandlerStruct{service}
}

// customerIDFromContext is set by helpers.CustomerVerifyToken.
func customerIDFromContext(c *fiber.Ctx) string {
	if v, ok := c.Locals("user_id").(string); ok {
		return v
	}
	return ""
}

// List handles GET /api/customer/tickets
func (h *TicketHandlerStruct) List(c *fiber.Ctx) error {
	customerID := customerIDFromContext(c)
	if customerID == "" {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Tidak terotorisasi", nil)
	}

	items, err := h.service.List(customerID)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Riwayat keluhan", items)
}

// Create handles POST /api/customer/tickets
func (h *TicketHandlerStruct) Create(c *fiber.Ctx) error {
	customerID := customerIDFromContext(c)
	if customerID == "" {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Tidak terotorisasi", nil)
	}

	request := new(CreateCustomerTicketRequest)
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

	return helpers.ResponseUtils(c, fiber.StatusCreated, true, "Keluhan berhasil dikirim", item)
}

// Detail handles GET /api/customer/tickets/:id
func (h *TicketHandlerStruct) Detail(c *fiber.Ctx) error {
	customerID := customerIDFromContext(c)
	if customerID == "" {
		return helpers.ResponseUtils(c, fiber.StatusUnauthorized, false, "Tidak terotorisasi", nil)
	}

	ticketID, err := ParseTicketID(c.Params("id"))
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}

	detail, err := h.service.Detail(customerID, ticketID)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusNotFound, false, err.Error(), nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Detail keluhan", detail)
}

// TroubleTypes handles GET /api/customer/tickets/trouble-types
func (h *TicketHandlerStruct) TroubleTypes(c *fiber.Ctx) error {
	types, err := h.service.TroubleTypes()
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Jenis kendala", types)
}

// UploadAttachment handles POST /api/customer/tickets/upload (multipart, field
// name "image") and returns the stored file name. The same convention as the
// staff upload endpoint is used: files land in ./uploads/cs-images and only the
// file name is persisted on the ticket row (trouble_tickets.img_cs).
func (h *TicketHandlerStruct) UploadAttachment(c *fiber.Ctx) error {
	file, err := c.FormFile("image")
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "File gambar tidak disediakan", nil)
	}

	if !isValidImageType(file.Header.Get("Content-Type")) {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "Tipe file tidak valid. Hanya gambar yang diizinkan", nil)
	}

	// 5MB limit: attachments come from low end phones.
	if file.Size > 5*1024*1024 {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "Ukuran file terlalu besar. Maksimum 5MB", nil)
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "File tidak memiliki ekstensi", nil)
	}

	uploadDir := "./uploads/cs-images"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, "Gagal membuat direktori upload", nil)
	}

	filename := fmt.Sprintf("customer-%d%s", time.Now().UnixNano(), ext)
	if err := c.SaveFile(file, filepath.Join(uploadDir, filename)); err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, "Gagal menyimpan file", nil)
	}

	return helpers.ResponseUtils(c, fiber.StatusOK, true, "Gambar berhasil diunggah", map[string]string{
		"filename": filename,
		"url":      "/uploads/cs-images/" + filename,
	})
}

func isValidImageType(contentType string) bool {
	for _, valid := range []string{"image/jpeg", "image/jpg", "image/png", "image/webp", "image/gif"} {
		if strings.HasPrefix(contentType, valid) {
			return true
		}
	}
	return false
}
