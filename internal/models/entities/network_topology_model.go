package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NodeType identifies the role of a node inside the fiber topology.
type NodeType string

const (
	NodeTypeServer    NodeType = "server"
	NodeTypeRouter    NodeType = "router"
	NodeTypeSwitchHub NodeType = "switch_hub"
	NodeTypeODC       NodeType = "odc"
	NodeTypeODP       NodeType = "odp"
	NodeTypePOP       NodeType = "pop" // backward-compatible legacy value
)

// LinkType identifies the fiber segment type.
type LinkType string

const (
	LinkTypeBackbone     LinkType = "backbone"     // server -> POP
	LinkTypeDistribution LinkType = "distribution" // POP -> ODP
	LinkTypeDrop         LinkType = "drop"         // ODP -> customer
)

// LinkStatus is the monitored state of a fiber segment.
type LinkStatus string

const (
	LinkStatusUp       LinkStatus = "up"
	LinkStatusDown     LinkStatus = "down"
	LinkStatusDegraded LinkStatus = "degraded"
)

// NetworkNode represents infrastructure equipment: server, POP or ODP.
//
// The topology is a tree: server -> POP -> ODP. ParentID points one level up
// so the status resolver can walk ancestors to detect a broken backbone.
type NetworkNode struct {
	ID       string   `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Code     string   `json:"code" gorm:"type:varchar(32);not null;uniqueIndex:idx_network_nodes_code"`
	Name     string   `json:"name" gorm:"type:varchar(191);not null"`
	Type     NodeType `json:"type" gorm:"type:varchar(32);not null;index:idx_network_nodes_type"`
	ParentID *string  `json:"parent_id,omitempty" gorm:"type:varchar(36);index:idx_network_nodes_parent"`

	Latitude  float64 `json:"latitude" gorm:"type:double;not null"`
	Longitude float64 `json:"longitude" gorm:"type:double;not null"`

	// CoverageRadiusM draws the "area cakupan" circle on the map (default 200 m).
	CoverageRadiusM int `json:"coverage_radius_m" gorm:"type:int;not null;default:200"`

	// OLT host/port for ODP nodes (e.g. 10.10.9.203 / gpon-onu_1).
	OLTHost *string `json:"olt_host,omitempty" gorm:"type:varchar(191)"`
	OLTPort *string `json:"olt_port,omitempty" gorm:"type:varchar(191)"`

	Capacity int        `json:"capacity" gorm:"type:int;default:0"`
	Status   LinkStatus `json:"status" gorm:"type:enum('up','down','degraded');default:'up'"`

	CreatedAt time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt time.Time `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)"`
}

func (n *NetworkNode) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	if n.CoverageRadiusM == 0 {
		n.CoverageRadiusM = 200
	}
	return nil
}

func (NetworkNode) TableName() string {
	return "network_nodes"
}

type OdpPortStatus string

const (
	OdpPortStatusAvailable OdpPortStatus = "available"
	OdpPortStatusOccupied  OdpPortStatus = "occupied"
	OdpPortStatusDamaged   OdpPortStatus = "damaged"
)

// OdpPort tracks physical splitter port capacity for an ODP node.
type OdpPort struct {
	ID         string        `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ODPNodeID  string        `json:"odp_node_id" gorm:"type:varchar(36);not null;uniqueIndex:idx_odp_ports_node_port,priority:1"`
	ODPNode    *NetworkNode  `json:"odp_node,omitempty" gorm:"foreignKey:ODPNodeID;references:ID"`
	PortNumber int           `json:"port_number" gorm:"not null;uniqueIndex:idx_odp_ports_node_port,priority:2"`
	Status     OdpPortStatus `json:"status" gorm:"type:varchar(20);not null;default:'available';index:idx_odp_ports_status"`
	CustomerID *string       `json:"customer_id,omitempty" gorm:"type:varchar(191);index:idx_odp_ports_customer"`
	Customer   *Customer     `json:"customer,omitempty" gorm:"foreignKey:CustomerID;references:ID"`
	Notes      string        `json:"notes,omitempty" gorm:"type:text"`
	CreatedAt  time.Time     `json:"created_at" gorm:"default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt  time.Time     `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)"`
}

func (p *OdpPort) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

func (OdpPort) TableName() string {
	return "odp_ports"
}

// NetworkLink represents a fiber segment between two nodes, or the drop cable
// from an ODP to a customer.
//
// Geometry stores the GeoJSON-ordered coordinate list [[lat,lng],...] produced
// by the seeder, so the frontend only draws polylines and never computes paths.
type NetworkLink struct {
	ID   string   `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Code string   `json:"code" gorm:"type:varchar(48);not null;uniqueIndex:idx_network_links_code"`
	Name string   `json:"name" gorm:"type:varchar(191);not null"`
	Type LinkType `json:"type" gorm:"type:enum('backbone','distribution','drop');not null;index:idx_network_links_type"`

	FromNodeID *string `json:"from_node_id,omitempty" gorm:"type:varchar(36);index:idx_network_links_from"`
	ToNodeID   *string `json:"to_node_id,omitempty" gorm:"type:varchar(36);index:idx_network_links_to"`

	// ToCustomerID is only set for Type == drop.
	ToCustomerID *string `json:"to_customer_id,omitempty" gorm:"type:varchar(191);index:idx_network_links_customer"`

	// Geometry holds [[lat,lng],...]. MySQL JSON column.
	Geometry []byte `json:"geometry,omitempty" gorm:"type:json"`

	LengthM      *float64 `json:"length_m,omitempty" gorm:"type:decimal(10,2)"`
	CapacityMbps *int     `json:"capacity_mbps,omitempty" gorm:"type:int"`
	UtilizedMbps *int     `json:"utilized_mbps,omitempty" gorm:"type:int"`

	Status LinkStatus `json:"status" gorm:"type:enum('up','down','degraded');default:'up';index:idx_network_links_status"`

	// NetwatchIP is the address MikroTik Netwatch pings to determine link health.
	// MikroTikCollector joins netwatch_devices.ip_address against this column.
	NetwatchIP *string `json:"netwatch_ip,omitempty" gorm:"type:varchar(45);index:idx_network_links_netwatch_ip"`

	CreatedAt time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt time.Time `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)"`
}

func (l *NetworkLink) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	return nil
}

func (NetworkLink) TableName() string {
	return "network_links"
}

// CustomerNetworkLink maps a customer to the ODP that serves it, including the
// OLT port used for diagnostics ("port OLT" in the detail panel).
type CustomerNetworkLink struct {
	ID         string  `json:"id" gorm:"primaryKey;type:varchar(36)"`
	CustomerID string  `json:"customer_id" gorm:"type:varchar(191);not null;uniqueIndex:idx_cnl_customer"`
	ODPNodeID  string  `json:"odp_node_id" gorm:"type:varchar(36);not null;index:idx_cnl_odp"`
	OLTPort    *string `json:"olt_port,omitempty" gorm:"type:varchar(64)"`
	SplitterID *string `json:"splitter_port,omitempty" gorm:"type:varchar(32)"`

	CreatedAt time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt time.Time `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)"`
}

func (c *CustomerNetworkLink) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

func (CustomerNetworkLink) TableName() string {
	return "customer_network_links"
}

// NetworkCableRoute stores a geographic cable path for Backbone, Distribution,
// or Dropcore visualization.
type NetworkCableRoute struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name      string    `json:"name" gorm:"type:varchar(191);not null"`
	CableType string    `json:"cable_type" gorm:"type:varchar(32);not null;index:idx_cable_routes_type"`
	TotalCore int       `json:"total_core" gorm:"type:int;not null;default:12"`
	UsedCore  int       `json:"used_core" gorm:"type:int;not null;default:0"`
	Geometry  []byte    `json:"-" gorm:"type:json;not null"`
	Status    string    `json:"status" gorm:"type:varchar(20);not null;default:'active'"`
	CreatedAt time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt time.Time `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)"`
}

func (r *NetworkCableRoute) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}

func (NetworkCableRoute) TableName() string {
	return "network_cable_routes"
}
