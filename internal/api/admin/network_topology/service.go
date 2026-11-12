package networktopology

import (
	"fmt"

	"skripsi-be/internal/models/entities"
)

type Service interface {
	List(nodeType string) ([]entities.NetworkNode, error)
	Tree() ([]NodeTreeItem, error)
	Create(req NetworkNodeRequest) (entities.NetworkNode, error)
	Update(id string, req NetworkNodeRequest) (entities.NetworkNode, error)
	Delete(id string) error
	GetPorts(id string) ([]entities.OdpPort, error)
}

type service struct{ repo Repository }

func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) List(nodeType string) ([]entities.NetworkNode, error) {
	return s.repo.List(nodeType)
}
func (s *service) Tree() ([]NodeTreeItem, error) { return s.repo.Tree() }
func (s *service) Create(req NetworkNodeRequest) (entities.NetworkNode, error) {
	if req.ParentID != nil {
		parent, err := s.repo.Get(*req.ParentID)
		if err != nil {
			return entities.NetworkNode{}, fmt.Errorf("parent node tidak ditemukan")
		}
		validHierarchy := map[entities.NodeType][]entities.NodeType{
			entities.NodeTypeRouter:    {entities.NodeTypeServer},
			entities.NodeTypeSwitchHub: {entities.NodeTypeRouter},
			entities.NodeTypeODC:       {entities.NodeTypeSwitchHub},
			entities.NodeTypeODP:       {entities.NodeTypeODC},
		}
		if allowed, ok := validHierarchy[req.Type]; ok {
			found := false
			for _, allowedType := range allowed {
				if parent.Type == allowedType {
					found = true
					break
				}
			}
			if !found {
				return entities.NetworkNode{}, fmt.Errorf("parent node tidak sesuai hierarki jaringan")
			}
		}
	}
	return s.repo.Create(req)
}
func (s *service) Update(id string, req NetworkNodeRequest) (entities.NetworkNode, error) {
	return s.repo.Update(id, req)
}
func (s *service) Delete(id string) error { return s.repo.Delete(id) }
func (s *service) GetPorts(id string) ([]entities.OdpPort, error) {
	return s.repo.GetPorts(id)
}
