package news

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"skripsi-be/internal/api/common/validation"
	"skripsi-be/internal/helpers"
	"skripsi-be/internal/models/entities"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxNewsImageSize = 5 * 1024 * 1024

type Handler struct {
	repository *Repository
}

func NewHandler(repository *Repository) *Handler {
	return &Handler{repository: repository}
}

func (h *Handler) ListPublic(c *fiber.Ctx) error {
	page, limit := pagination(c)
	items, total, err := h.repository.ListPublished(page, limit)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}
	response := PublicNewsPage{Page: page, Limit: limit, Total: total, Items: make([]PublicNewsItem, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, toPublicItem(item))
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "", response)
}

func (h *Handler) GetPublic(c *fiber.Ctx) error {
	item, err := h.repository.GetPublishedByID(c.Params("id"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return helpers.ResponseUtils(c, fiber.StatusNotFound, false, "News not found", nil)
	}
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}
	response := PublicNewsDetail{PublicNewsItem: toPublicItem(item), Content: item.Content}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "", response)
}

func (h *Handler) ListAdmin(c *fiber.Ctx) error {
	items, err := h.repository.ListAdmin()
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "", items)
}

func (h *Handler) GetAdmin(c *fiber.Ctx) error {
	item, err := h.repository.GetByID(c.Params("id"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return helpers.ResponseUtils(c, fiber.StatusNotFound, false, "News not found", nil)
	}
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "", item)
}

func (h *Handler) Create(c *fiber.Ctx) error {
	request := newsWriteRequestFromForm(c)
	if messages := validation.ValidationRequest(request); len(messages) > 0 {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "Invalid news data", messages)
	}
	imagePath, err := saveNewsImage(c)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}
	if imagePath == "" {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, "Image file is required", nil)
	}
	request.ImageURL = imagePath
	item, err := h.repository.Create(request)
	if err != nil {
		removeNewsImage(imagePath)
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}
	return helpers.ResponseUtils(c, fiber.StatusCreated, true, "News created", item)
}

func (h *Handler) Update(c *fiber.Ctx) error {
	current, err := h.repository.GetByID(c.Params("id"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return helpers.ResponseUtils(c, fiber.StatusNotFound, false, "News not found", nil)
	}
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}
	request := NewsWriteRequest{
		Title:       current.Title,
		Summary:     current.Summary,
		Content:     current.Content,
		ImageURL:    current.ImageURL,
		IsPublished: current.IsPublished,
	}
	if err := applyNewsUpdateFields(c, &request); err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}
	imagePath, err := saveNewsImage(c)
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusBadRequest, false, err.Error(), nil)
	}
	if imagePath != "" {
		request.ImageURL = imagePath
	} else {
		request.ImageURL = current.ImageURL
	}
	item, err := h.repository.Update(c.Params("id"), request)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		removeNewsImage(imagePath)
		return helpers.ResponseUtils(c, fiber.StatusNotFound, false, "News not found", nil)
	}
	if err != nil {
		removeNewsImage(imagePath)
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}
	if imagePath != "" && current.ImageURL != imagePath {
		removeNewsImage(current.ImageURL)
	}
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "News updated", item)
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	item, err := h.repository.GetByID(c.Params("id"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return helpers.ResponseUtils(c, fiber.StatusNotFound, false, "News not found", nil)
	}
	if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}
	if err := h.repository.Delete(c.Params("id")); errors.Is(err, gorm.ErrRecordNotFound) {
		return helpers.ResponseUtils(c, fiber.StatusNotFound, false, "News not found", nil)
	} else if err != nil {
		return helpers.ResponseUtils(c, fiber.StatusInternalServerError, false, err.Error(), nil)
	}
	removeNewsImage(item.ImageURL)
	return helpers.ResponseUtils(c, fiber.StatusOK, true, "News deleted", nil)
}

func newsWriteRequestFromForm(c *fiber.Ctx) NewsWriteRequest {
	published, _ := publishedValue(c.FormValue("is_published"))
	return NewsWriteRequest{
		Title:       c.FormValue("title"),
		Summary:     c.FormValue("summary"),
		Content:     c.FormValue("content"),
		IsPublished: published,
	}
}

func applyNewsUpdateFields(c *fiber.Ctx, request *NewsWriteRequest) error {
	if strings.HasPrefix(c.Get(fiber.HeaderContentType), fiber.MIMEApplicationJSON) {
		fields := map[string]json.RawMessage{}
		if err := c.BodyParser(&fields); err != nil {
			return fmt.Errorf("invalid JSON body: %w", err)
		}
		for key, target := range map[string]*string{
			"title": &request.Title, "summary": &request.Summary, "content": &request.Content,
		} {
			if raw, ok := fields[key]; ok {
				if err := json.Unmarshal(raw, target); err != nil {
					return fmt.Errorf("field %s must be a string", key)
				}
			}
		}
		for _, key := range []string{"is_published", "isPublished", "published", "status", "visibility"} {
			if raw, ok := fields[key]; ok {
				var value any
				if err := json.Unmarshal(raw, &value); err != nil {
					return fmt.Errorf("invalid publication status")
				}
				published, err := parsePublishedValue(value)
				if err != nil {
					return err
				}
				request.IsPublished = published
				break
			}
		}
		return nil
	}

	form, err := c.MultipartForm()
	if err != nil {
		return fmt.Errorf("invalid multipart form: %w", err)
	}
	for key, target := range map[string]*string{
		"title": &request.Title, "summary": &request.Summary, "content": &request.Content,
	} {
		if values, ok := form.Value[key]; ok && len(values) > 0 {
			*target = values[0]
		}
	}
	for _, key := range []string{"is_published", "isPublished", "published", "status", "visibility"} {
		if values, ok := form.Value[key]; ok && len(values) > 0 {
			published, err := publishedValue(values[0])
			if err != nil {
				return err
			}
			request.IsPublished = published
			break
		}
	}
	return nil
}

func publishedValue(value string) (bool, error) {
	return parsePublishedValue(value)
}

func parsePublishedValue(value any) (bool, error) {
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "published", "publish", "visible", "show", "tampilkan":
			return true, nil
		case "false", "0", "draft", "hidden", "hide", "sembunyikan":
			return false, nil
		}
	}
	return false, fmt.Errorf("status must be true/false or published/hidden")
}

func saveNewsImage(c *fiber.Ctx) (string, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return "", fmt.Errorf("invalid multipart form: %w", err)
	}
	files := form.File["image"]
	if len(files) == 0 {
		return "", nil
	}
	file := files[0]
	if file.Size <= 0 || file.Size > maxNewsImageSize {
		return "", fmt.Errorf("image must be between 1 byte and 5 MB")
	}

	extension := strings.ToLower(filepath.Ext(file.Filename))
	allowedExtensions := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
	if !allowedExtensions[extension] {
		return "", fmt.Errorf("image must be a JPG, PNG, WEBP, or GIF file")
	}
	input, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("cannot read uploaded image: %w", err)
	}
	defer input.Close()
	header := make([]byte, 512)
	count, err := input.Read(header)
	if err != nil && count == 0 {
		return "", fmt.Errorf("cannot read uploaded image: %w", err)
	}
	contentType := http.DetectContentType(header[:count])
	allowedTypes := map[string]bool{
		".jpg": "image/jpeg" == contentType, ".jpeg": "image/jpeg" == contentType,
		".png": "image/png" == contentType, ".webp": "image/webp" == contentType,
		".gif": "image/gif" == contentType,
	}
	if !allowedTypes[extension] {
		return "", fmt.Errorf("uploaded file is not a valid image")
	}

	filename := uuid.NewString() + extension
	if err := os.MkdirAll(filepath.Join("uploads", "news"), 0755); err != nil {
		return "", fmt.Errorf("cannot create image directory: %w", err)
	}
	destination := filepath.Join("uploads", "news", filename)
	if err := c.SaveFile(file, destination); err != nil {
		return "", fmt.Errorf("cannot save uploaded image: %w", err)
	}
	return "/uploads/news/" + filename, nil
}

func removeNewsImage(imagePath string) {
	const prefix = "/uploads/news/"
	if !strings.HasPrefix(imagePath, prefix) {
		return
	}
	filename := strings.TrimPrefix(imagePath, prefix)
	if filename == "" || filepath.Base(filename) != filename {
		return
	}
	_ = os.Remove(filepath.Join("uploads", "news", filename))
}

func pagination(c *fiber.Ctx) (int, int) {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.Query("limit", "9"))
	if err != nil || limit < 1 {
		limit = 9
	}
	if limit > 50 {
		limit = 50
	}
	return page, limit
}

func toPublicItem(item entities.News) PublicNewsItem {
	return PublicNewsItem{
		ID:          item.ID,
		Title:       item.Title,
		Summary:     item.Summary,
		ImageURL:    item.ImageURL,
		PublishedAt: item.PublishedAt,
	}
}
