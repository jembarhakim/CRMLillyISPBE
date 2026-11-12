CREATE TABLE IF NOT EXISTS `network_router_credentials` (
  `id`                 VARCHAR(36)  NOT NULL,
  `name`               VARCHAR(191) NOT NULL,
  `host`               VARCHAR(191) NOT NULL,
  `port`               INT          NOT NULL,
  `username`           VARCHAR(191) NOT NULL,
  `encrypted_password` TEXT         NOT NULL,
  `use_tls`            BOOLEAN      NOT NULL DEFAULT FALSE,
  `created_at`         DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at`         DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_router_credentials_host_port_user` (`host`, `port`, `username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
