-- Migration: Add minimum_stock column to items catalog table
-- Adds a per-item minimum stock threshold used for low-stock alerts.
--
-- Note: MySQL does not support `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`,
-- so this script checks INFORMATION_SCHEMA before altering (same pattern as
-- add_unit_notes_to_items_tables.sql).

SET @dbname = DATABASE();
SET @tablename = "items";
SET @columnname = "minimum_stock";

SET @preparedStatement = (SELECT IF(
  (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
    WHERE
      (table_name = @tablename)
      AND (table_schema = @dbname)
      AND (column_name = @columnname)
  ) > 0,
  "SELECT 1",
  CONCAT("ALTER TABLE ", @tablename, " ADD COLUMN `", @columnname, "` INTEGER NOT NULL DEFAULT 0 AFTER `asset_id`")
));
PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;

-- Test data: set a minimum stock of 5 for the test items
UPDATE `items` SET `minimum_stock` = 5 WHERE `name` LIKE '%Test Min Stock A%';

-- Verification
SELECT `name`, `minimum_stock` FROM `items` WHERE `name` LIKE '%Test Min Stock A%';
