package networktopology

import "skripsi-be/internal/models/entities"

type NetworkNodeRequest struct {
	Code            string              `json:"code"`
	Name            string              `json:"name"`
	Type            entities.NodeType   `json:"type"`
	ParentID        *string             `json:"parent_id"`
	Latitude        float64             `json:"latitude"`
	Longitude       float64             `json:"longitude"`
	CoverageRadiusM int                 `json:"coverage_radius_m"`
	OLTHost         *string             `json:"olt_host"`
	OLTPort         *string             `json:"olt_port"`
	Capacity        int                 `json:"capacity"`
	Status          entities.LinkStatus `json:"status"`
}
