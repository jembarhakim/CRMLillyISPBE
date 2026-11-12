CREATE TABLE IF NOT EXISTS `payment_attempts` (
  `id`          VARCHAR(191) NOT NULL,
  `invoice_id`  VARCHAR(191) NOT NULL,
  `order_id`    VARCHAR(50)  NOT NULL,
  `bank`        VARCHAR(10)  NOT NULL,
  `va_number`   VARCHAR(50)  NOT NULL,
  `amount`      BIGINT       NOT NULL,
  `expiry_time` DATETIME(3)  NOT NULL,
  `status`      VARCHAR(32)  NOT NULL,
  `created_at`  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at`  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_payment_attempts_order_id` (`order_id`),
  KEY `idx_payment_attempt_invoice_status` (`invoice_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
