CREATE TABLE IF NOT EXISTS `physical_optical_readings` (
  `id` VARCHAR(36) NOT NULL,
  `customer_id` VARCHAR(191) NOT NULL,
  `source_event` VARCHAR(32) NOT NULL,
  `technician_id` VARCHAR(191) NOT NULL,
  `technician_name` VARCHAR(191) NULL,
  `rx_power` DECIMAL(6,2) NOT NULL,
  `optical_status` VARCHAR(20) NOT NULL,
  `notes` TEXT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_physical_optical_customer` (`customer_id`),
  KEY `idx_physical_optical_source` (`source_event`),
  KEY `idx_physical_optical_status` (`optical_status`),
  CONSTRAINT `fk_physical_optical_customer` FOREIGN KEY (`customer_id`) REFERENCES `customer` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `link_flapping_status` (
  `customer_id` VARCHAR(191) NOT NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'stable',
  `disconnect_count` INT NOT NULL DEFAULT 0,
  `window_start` DATETIME(3) NULL,
  `last_detected_at` DATETIME(3) NULL,
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`customer_id`),
  KEY `idx_link_flapping_status` (`status`),
  CONSTRAINT `fk_link_flapping_customer` FOREIGN KEY (`customer_id`) REFERENCES `customer` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
