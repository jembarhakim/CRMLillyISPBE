-- =============================================================================
-- Tabel monitoring ONT (sumber data: GenieACS poller)
-- Dipakai oleh: internal/models/entities/ont_status_model.go
-- =============================================================================

CREATE TABLE IF NOT EXISTS `ont_statuses` (
  `id`              VARCHAR(191) NOT NULL,
  `customer_id`     VARCHAR(191) NOT NULL,
  `serial_number`   VARCHAR(191) NOT NULL,
  `manufacturer`    VARCHAR(100) NULL,
  `model_name`      VARCHAR(100) NULL,
  `is_online`       TINYINT(1)   NOT NULL DEFAULT 0,
  `rx_power`        DOUBLE       NULL,
  `tx_power`        DOUBLE       NULL,
  `software_version` VARCHAR(100) NULL,
  `ip_address`      VARCHAR(45)  NULL,
  `mac_address`     VARCHAR(191) NULL,
  `optical_status`  VARCHAR(50)  NULL DEFAULT 'unknown',
  `ssid`            VARCHAR(191) NULL,
  `client_count`    INT          NOT NULL DEFAULT 0,
  `last_inform`     DATETIME(3)  NULL,
  `is_stale`        TINYINT(1)   NOT NULL DEFAULT 0,
  `created_at`      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at`      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_ont_status_customer_id` (`customer_id`),
  KEY `idx_ont_status_serial` (`serial_number`),
  KEY `idx_ont_status_online` (`is_online`),
  KEY `idx_ont_status_optical` (`optical_status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `ont_status_histories` (
  `id`              VARCHAR(191) NOT NULL,
  `customer_id`     VARCHAR(191) NOT NULL,
  `serial_number`   VARCHAR(191) NOT NULL,
  `is_online`       TINYINT(1)   NOT NULL DEFAULT 0,
  `rx_power`        DOUBLE       NULL,
  `tx_power`        DOUBLE       NULL,
  `optical_status`  VARCHAR(50)  NULL,
  `recorded_at`     DATETIME(3)  NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_ont_hist_customer` (`customer_id`),
  KEY `idx_ont_hist_serial` (`serial_number`),
  KEY `idx_ont_hist_recorded_at` (`recorded_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
