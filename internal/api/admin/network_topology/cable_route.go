package networktopology

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"skripsi-be/internal/models/entities"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CableRouteRequest struct {
	Name        string      `json:"name"`
	CableType   string      `json:"cable_type"`
	Type        string      `json:"type"`
	TotalCore   int         `json:"total_core"`
	UsedCore    int         `json:"used_core"`
	Geometry    [][]float64 `json:"geometry"`
	Coordinates [][]float64 `json:"coordinates"`
}

type CableRouteResponse struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	CableType string      `json:"cable_type"`
	TotalCore int         `json:"total_core"`
	UsedCore  int         `json:"used_core"`
	Geometry  [][]float64 `json:"geometry"`
	Status    string      `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

func RegisterCableRouteHandlers(admin fiber.Router, db *gorm.DB) {
	admin.Get("/cable-routes", func(c *fiber.Ctx) error {
		var routes []entities.NetworkCableRoute
		if err := db.Order("created_at DESC").Find(&routes).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		result := make([]CableRouteResponse, 0, len(routes))
		for _, route := range routes {
			var geometry [][]float64
			_ = json.Unmarshal(route.Geometry, &geometry)
			result = append(result, CableRouteResponse{
				ID: route.ID, Name: route.Name, CableType: route.CableType,
				TotalCore: route.TotalCore, UsedCore: route.UsedCore, Geometry: geometry,
				Status: route.Status, CreatedAt: route.CreatedAt, UpdatedAt: route.UpdatedAt,
			})
		}
		return c.JSON(fiber.Map{"success": true, "data": result})
	})

	admin.Post("/cable-routes", func(c *fiber.Ctx) error {
		var req CableRouteRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "Payload tidak valid"})
		}
		coordinates := req.Geometry
		if len(coordinates) == 0 {
			coordinates = req.Coordinates
		}
		cableType := strings.ToLower(strings.TrimSpace(req.CableType))
		if cableType == "" {
			cableType = strings.ToLower(strings.TrimSpace(req.Type))
		}
		if req.Name == "" || len(coordinates) < 2 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "Nama dan minimal dua titik koordinat wajib diisi"})
		}
		if cableType != "backbone" && cableType != "distribution" && cableType != "dropcore" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "cable_type harus backbone, distribution, atau dropcore"})
		}
		for _, point := range coordinates {
			if len(point) != 2 || point[0] < -90 || point[0] > 90 || point[1] < -180 || point[1] > 180 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "Format koordinat harus [lat, lng] yang valid"})
			}
		}
		if req.TotalCore <= 0 {
			req.TotalCore = 12
		}
		if req.UsedCore < 0 || req.UsedCore > req.TotalCore {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "Jumlah core terpakai tidak valid"})
		}
		raw, err := json.Marshal(coordinates)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		route := entities.NetworkCableRoute{
			ID: uuid.NewString(), Name: req.Name, CableType: cableType,
			TotalCore: req.TotalCore, UsedCore: req.UsedCore, Geometry: raw, Status: "active",
		}
		if err := db.Create(&route).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		return c.Status(201).JSON(fiber.Map{"success": true, "data": CableRouteResponse{
			ID: route.ID, Name: route.Name, CableType: route.CableType,
			TotalCore: route.TotalCore, UsedCore: route.UsedCore, Geometry: coordinates,
			Status: route.Status, CreatedAt: route.CreatedAt, UpdatedAt: route.UpdatedAt,
		}})
	})

	admin.Delete("/cable-routes/:id", func(c *fiber.Ctx) error {
		result := db.Delete(&entities.NetworkCableRoute{}, "id = ?", c.Params("id"))
		if result.Error != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": result.Error.Error()})
		}
		if result.RowsAffected == 0 {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": fmt.Sprintf("Cable route tidak ditemukan")})
		}
		return c.JSON(fiber.Map{"success": true, "message": "Cable route berhasil dihapus"})
	})
}
