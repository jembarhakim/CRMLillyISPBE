package networktopology

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{ service Service }

func NewHandler(service Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *fiber.Ctx) error {
	nodes, err := h.service.List(strings.TrimSpace(c.Query("type")))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": nodes})
}

func (h *Handler) Tree(c *fiber.Ctx) error {
	nodes, err := h.service.Tree()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": nodes})
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var req NetworkNodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "payload tidak valid"})
	}
	node, err := h.service.Create(req)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "data": node})
}

func (h *Handler) Update(c *fiber.Ctx) error {
	var req NetworkNodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "payload tidak valid"})
	}
	node, err := h.service.Update(c.Params("id"), req)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": node})
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	if err := h.service.Delete(c.Params("id")); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Node berhasil dihapus"})
}

func (h *Handler) Ports(c *fiber.Ctx) error {
	ports, err := h.service.GetPorts(c.Params("id"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": ports})
}
