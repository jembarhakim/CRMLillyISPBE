-- =============================================================================
-- Rollback: Network Topology for Customer Mapping & Monitoring
-- Hapus urutan terbalik dari foreign key.
-- =============================================================================

DROP TABLE IF EXISTS `customer_network_links`;
DROP TABLE IF EXISTS `network_cable_routes`;
DROP TABLE IF EXISTS `odp_ports`;
DROP TABLE IF EXISTS `network_links`;
DROP TABLE IF EXISTS `network_nodes`;
