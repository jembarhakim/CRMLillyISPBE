package incidents

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *fiber.Ctx) error {
	items, err := h.service.List(c.Query("status"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": items, "total": len(items)})
}

type CreateRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Source      string   `json:"source"`
	NodeID      string   `json:"node_id"`
	ODPNodeID   string   `json:"odp_node_id"`
	NodeName    string   `json:"node_name"`
	CustomerIDs []string `json:"affected_customer_ids"`
	ETA         string   `json:"eta"`
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var req CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "payload tidak valid"})
	}
	var eta *time.Time
	if strings.TrimSpace(req.ETA) != "" {
		if parsed, err := time.Parse(time.RFC3339, req.ETA); err == nil {
			eta = &parsed
		}
	}
	item, err := h.service.Create(CreateRequest{
		Title: req.Title, Description: req.Description, Source: req.Source,
		NodeID: req.NodeID, ODPNodeID: req.ODPNodeID, NodeName: req.NodeName,
		CustomerIDs: req.CustomerIDs, ETA: func() string { return "" }(),
	})
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	if eta != nil {
		item.ETA = eta
		_ = h.service.Update(item)
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "data": item})
}

type ResolveRequest struct {
	ResolutionNote string `json:"resolution_note"`
}

func (h *Handler) Resolve(c *fiber.Ctx) error {
	var req ResolveRequest
	_ = c.BodyParser(&req)
	item, err := h.service.Resolve(c.Params("id"), req.ResolutionNote)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": item})
}

type UpdateRequest struct {
	ETA            string `json:"eta"`
	Status         string `json:"status"`
	ResolutionNote string `json:"resolution_note"`
}

func (h *Handler) Update(c *fiber.Ctx) error {
	var req UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "payload tidak valid"})
	}
	item, err := h.service.Get(c.Params("id"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	if strings.TrimSpace(req.ETA) != "" {
		if parsed, err := time.Parse(time.RFC3339, req.ETA); err == nil {
			item.ETA = &parsed
		}
	}
	if req.ResolutionNote != "" {
		item.ResolutionNote = &req.ResolutionNote
	}
	if req.Status == "resolved" {
		resolved, err := h.service.Resolve(item.ID, req.ResolutionNote)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return c.JSON(fiber.Map{"success": true, "data": resolved})
	}
	if err := h.service.Update(item); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": item})
}
