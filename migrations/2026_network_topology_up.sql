-- =============================================================================
-- Network Topology for Customer Mapping & Monitoring
-- Fase 1B: tabel infrastruktur jaringan (server -> POP -> ODP) dan link fiber.
--
-- Catatan: tabel ini TIDAK menyimpan data pelanggan dummy. Pelanggan nyata
-- tetap di tabel `customer`. Baris `network_links` bertipe 'drop' menunjuk ke
-- customer yang sudah ada.
--
-- Rollback: 2026_network_topology_down.sql
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 1. network_nodes : server / POP / ODP
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `network_nodes` (
  `id`                VARCHAR(36)  NOT NULL,
  `code`              VARCHAR(32)  NOT NULL,
  `name`              VARCHAR(191) NOT NULL,
  `type`              VARCHAR(32) NOT NULL,
  `parent_id`         VARCHAR(36)  NULL,
  `latitude`          DOUBLE       NOT NULL,
  `longitude`         DOUBLE       NOT NULL,
  `coverage_radius_m` INT          NOT NULL DEFAULT 200,
  `olt_host`          VARCHAR(191) NULL,
  `olt_port`          VARCHAR(191) NULL,
  `capacity`          INT          NULL DEFAULT 0,
  `status`            ENUM('up','down','degraded') NOT NULL DEFAULT 'up',
  `created_at`        DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at`        DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_network_nodes_code` (`code`),
  KEY `idx_network_nodes_type` (`type`),
  KEY `idx_network_nodes_parent` (`parent_id`),
  CONSTRAINT `fk_network_nodes_parent` FOREIGN KEY (`parent_id`) REFERENCES `network_nodes` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Port capacity for physical ODP splitter devices.
CREATE TABLE IF NOT EXISTS `odp_ports` (
  `id`          VARCHAR(36) NOT NULL,
  `odp_node_id` VARCHAR(36) NOT NULL,
  `port_number` INT NOT NULL,
  `status`      VARCHAR(20) NOT NULL DEFAULT 'available',
  `customer_id` VARCHAR(191) NULL,
  `notes`       TEXT NULL,
  `created_at`  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at`  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_odp_ports_node_port` (`odp_node_id`, `port_number`),
  KEY `idx_odp_ports_status` (`status`),
  KEY `idx_odp_ports_customer` (`customer_id`),
  CONSTRAINT `fk_odp_ports_node` FOREIGN KEY (`odp_node_id`) REFERENCES `network_nodes` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `network_cable_routes` (
  `id`         VARCHAR(36) NOT NULL,
  `name`       VARCHAR(191) NOT NULL,
  `cable_type` VARCHAR(32) NOT NULL,
  `total_core` INT NOT NULL DEFAULT 12,
  `used_core`  INT NOT NULL DEFAULT 0,
  `geometry`   JSON NOT NULL,
  `status`     VARCHAR(20) NOT NULL DEFAULT 'active',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_cable_routes_type` (`cable_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- -----------------------------------------------------------------------------
-- 2. network_links : backbone (server->POP), distribution (POP->ODP), drop (ODP->customer)
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `network_links` (
  `id`              VARCHAR(36)  NOT NULL,
  `code`            VARCHAR(48)  NOT NULL,
  `name`            VARCHAR(191) NOT NULL,
  `type`            ENUM('backbone','distribution','drop') NOT NULL,
  `from_node_id`    VARCHAR(36)  NULL,
  `to_node_id`      VARCHAR(36)  NULL,
  `to_customer_id`  VARCHAR(191) NULL,
  `geometry`        JSON         NULL,
  `length_m`        DECIMAL(10,2) NULL,
  `capacity_mbps`   INT          NULL,
  `utilized_mbps`   INT          NULL,
  `status`          ENUM('up','down','degraded') NOT NULL DEFAULT 'up',
  `netwatch_ip`     VARCHAR(45)  NULL,
  `created_at`      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at`      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_network_links_code` (`code`),
  KEY `idx_network_links_type` (`type`),
  KEY `idx_network_links_status` (`status`),
  KEY `idx_network_links_from` (`from_node_id`),
  KEY `idx_network_links_to` (`to_node_id`),
  KEY `idx_network_links_customer` (`to_customer_id`),
  KEY `idx_network_links_netwatch_ip` (`netwatch_ip`),
  CONSTRAINT `fk_network_links_from` FOREIGN KEY (`from_node_id`) REFERENCES `network_nodes` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_network_links_to`   FOREIGN KEY (`to_node_id`)   REFERENCES `network_nodes` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- -----------------------------------------------------------------------------
-- 3. customer_network_links : pelanggan -> ODP + port OLT
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `customer_network_links` (
  `id`            VARCHAR(36)  NOT NULL,
  `customer_id`   VARCHAR(191) NOT NULL,
  `odp_node_id`   VARCHAR(36)  NOT NULL,
  `olt_port`      VARCHAR(64)  NULL,
  `splitter_port` VARCHAR(32)  NULL,
  `created_at`    DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at`    DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_cnl_customer` (`customer_id`),
  KEY `idx_cnl_odp` (`odp_node_id`),
  CONSTRAINT `fk_cnl_odp` FOREIGN KEY (`odp_node_id`) REFERENCES `network_nodes` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
