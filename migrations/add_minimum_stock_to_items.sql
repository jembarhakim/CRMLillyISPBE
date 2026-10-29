-- Migration: Add minimum_stock column to items table (INV-01)
-- Safe to re-run: checks INFORMATION_SCHEMA first, MySQL has no ADD COLUMN IF NOT EXISTS
-- Runs against the currently selected database. Do not add a USE statement.

SET @dbname = DATABASE();
SET @tablename = 'items';
SET @columnname = 'minimum_stock';

SET @preparedStatement = (SELECT IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
   WHERE table_name = @tablename AND table_schema = @dbname AND column_name = @columnname) > 0,
  'SELECT 1',
  CONCAT('ALTER TABLE `', @tablename, '` ADD COLUMN `', @columnname, '` INT NOT NULL DEFAULT 0 AFTER `asset_id`')
));
PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;

SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_DEFAULT
FROM INFORMATION_SCHEMA.COLUMNS
WHERE table_schema = DATABASE() AND table_name = 'items' AND column_name = 'minimum_stock';

-- Rollback (manual, only if needed): ALTER TABLE items DROP COLUMN minimum_stock
