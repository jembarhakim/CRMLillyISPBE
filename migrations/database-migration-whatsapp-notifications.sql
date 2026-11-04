CREATE TABLE IF NOT EXISTS notification_templates (
  `key` VARCHAR(100) NOT NULL,
  name VARCHAR(191) NOT NULL,
  category VARCHAR(50) NOT NULL,
  body TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`key`)
);

CREATE TABLE IF NOT EXISTS notifications (
  id VARCHAR(191) NOT NULL,
  recipient VARCHAR(32) NOT NULL,
  template_key VARCHAR(100) NOT NULL,
  message TEXT NOT NULL,
  status VARCHAR(20) NOT NULL,
  attempts INT NOT NULL DEFAULT 0,
  max_attempts INT NOT NULL DEFAULT 5,
  next_attempt_at DATETIME(3) NOT NULL,
  deduplication_key VARCHAR(191) NULL,
  invoice_id VARCHAR(191) NULL,
  broadcast_id VARCHAR(191) NULL,
  gateway_message_id VARCHAR(191) NULL,
  last_error TEXT NULL,
  sent_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY idx_notifications_deduplication_key (deduplication_key),
  KEY idx_notifications_status_next_attempt (status, next_attempt_at),
  KEY idx_notifications_invoice_id (invoice_id),
  KEY idx_notifications_broadcast_id (broadcast_id),
  KEY idx_notifications_gateway_message_id (gateway_message_id)
);

CREATE TABLE IF NOT EXISTS notification_logs (
  id VARCHAR(191) NOT NULL,
  notification_id VARCHAR(191) NOT NULL,
  status VARCHAR(20) NOT NULL,
  attempt INT NOT NULL DEFAULT 0,
  details TEXT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_notification_logs_notification_id (notification_id)
);

CREATE TABLE IF NOT EXISTS notification_preferences (
  customer_id VARCHAR(191) NOT NULL,
  invoice_reminders_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (customer_id)
);
