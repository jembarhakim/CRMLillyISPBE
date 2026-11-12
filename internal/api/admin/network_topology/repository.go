package networktopology

import (
	"fmt"
	"time"

	"skripsi-be/internal/models/entities"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	List(nodeType string) ([]entities.NetworkNode, error)
	Tree() ([]NodeTreeItem, error)
	Get(id string) (entities.NetworkNode, error)
	Create(req NetworkNodeRequest) (entities.NetworkNode, error)
	Update(id string, req NetworkNodeRequest) (entities.NetworkNode, error)
	Delete(id string) error
	GetPorts(id string) ([]entities.OdpPort, error)
}

type repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &repository{db: db} }

func (r *repository) List(nodeType string) ([]entities.NetworkNode, error) {
	var nodes []entities.NetworkNode
	query := r.db.Order("name ASC")
	if nodeType != "" {
		query = query.Where("type = ?", nodeType)
	}
	err := query.Find(&nodes).Error
	return nodes, err
}

type NodeTreeItem struct {
	entities.NetworkNode
	Children []NodeTreeItem `json:"children"`
}

func (r *repository) Tree() ([]NodeTreeItem, error) {
	var nodes []entities.NetworkNode
	if err := r.db.Order("name ASC").Find(&nodes).Error; err != nil {
		return nil, err
	}

	byID := make(map[string]*NodeTreeItem, len(nodes))
	roots := make([]NodeTreeItem, 0)
	for i := range nodes {
		item := NodeTreeItem{NetworkNode: nodes[i], Children: []NodeTreeItem{}}
		byID[item.ID] = &item
	}
	for i := range nodes {
		item := byID[nodes[i].ID]
		if nodes[i].ParentID != nil {
			if parent, ok := byID[*nodes[i].ParentID]; ok {
				parent.Children = append(parent.Children, *item)
				continue
			}
		}
		roots = append(roots, *item)
	}
	return roots, nil
}

func (r *repository) Get(id string) (entities.NetworkNode, error) {
	var node entities.NetworkNode
	err := r.db.First(&node, "id = ?", id).Error
	return node, err
}

func validateNode(req NetworkNodeRequest) error {
	validTypes := map[entities.NodeType]bool{
		entities.NodeTypeServer: true, entities.NodeTypeRouter: true,
		entities.NodeTypeSwitchHub: true, entities.NodeTypeODC: true,
		entities.NodeTypeODP: true, entities.NodeTypePOP: true,
	}
	if req.Code == "" || req.Name == "" {
		return fmt.Errorf("code dan name wajib diisi")
	}
	if !validTypes[req.Type] {
		return fmt.Errorf("type node tidak valid")
	}
	if req.Type == entities.NodeTypeODP && req.Capacity != 8 && req.Capacity != 16 {
		return fmt.Errorf("kapasitas ODP harus 8 atau 16 port")
	}
	return nil
}

func (r *repository) Create(req NetworkNodeRequest) (entities.NetworkNode, error) {
	if err := validateNode(req); err != nil {
		return entities.NetworkNode{}, err
	}
	if req.Status == "" {
		req.Status = entities.LinkStatusUp
	}
	if req.CoverageRadiusM == 0 {
		req.CoverageRadiusM = 200
	}
	node := entities.NetworkNode{
		ID: uuid.NewString(), Code: req.Code, Name: req.Name, Type: req.Type,
		ParentID: req.ParentID, Latitude: req.Latitude, Longitude: req.Longitude,
		CoverageRadiusM: req.CoverageRadiusM, OLTHost: req.OLTHost, OLTPort: req.OLTPort,
		Capacity: req.Capacity, Status: req.Status, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&node).Error; err != nil {
			return err
		}
		if node.Type == entities.NodeTypeODP {
			for port := 1; port <= node.Capacity; port++ {
				row := entities.OdpPort{ID: uuid.NewString(), ODPNodeID: node.ID, PortNumber: port, Status: entities.OdpPortStatusAvailable}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	return node, err
}

func (r *repository) Update(id string, req NetworkNodeRequest) (entities.NetworkNode, error) {
	if err := validateNode(req); err != nil {
		return entities.NetworkNode{}, err
	}
	node, err := r.Get(id)
	if err != nil {
		return node, err
	}
	updates := map[string]interface{}{
		"code": req.Code, "name": req.Name, "type": req.Type, "parent_id": req.ParentID,
		"latitude": req.Latitude, "longitude": req.Longitude, "coverage_radius_m": req.CoverageRadiusM,
		"olt_host": req.OLTHost, "olt_port": req.OLTPort, "capacity": req.Capacity, "status": req.Status,
		"updated_at": time.Now(),
	}
	err = r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entities.NetworkNode{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}
		if req.Type == entities.NodeTypeODP && (req.Capacity == 8 || req.Capacity == 16) {
			var count int64
			if err := tx.Model(&entities.OdpPort{}).Where("odp_node_id = ?", id).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				for port := 1; port <= req.Capacity; port++ {
					if err := tx.Create(&entities.OdpPort{ID: uuid.NewString(), ODPNodeID: id, PortNumber: port, Status: entities.OdpPortStatusAvailable}).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	if err == nil {
		node, err = r.Get(id)
	}
	return node, err
}

func (r *repository) Delete(id string) error {
	var occupied int64
	if err := r.db.Model(&entities.OdpPort{}).Where("odp_node_id = ? AND status = ?", id, entities.OdpPortStatusOccupied).Count(&occupied).Error; err != nil {
		return err
	}
	if occupied > 0 {
		return fmt.Errorf("ODP tidak dapat dihapus karena masih memiliki port terpakai")
	}
	return r.db.Delete(&entities.NetworkNode{}, "id = ?", id).Error
}

func (r *repository) GetPorts(id string) ([]entities.OdpPort, error) {
	var ports []entities.OdpPort
	err := r.db.Preload("Customer").Where("odp_node_id = ?", id).Order("port_number ASC").Find(&ports).Error
	return ports, err
}
