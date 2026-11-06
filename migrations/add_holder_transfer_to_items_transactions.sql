-- Migration: INV-02 - add holder_id and transfer_ref to itemskeluar and itemsmasuk
-- Safe to re-run: each change is guarded by an INFORMATION_SCHEMA check.
-- Runs against the currently selected database. Do not add a USE statement.
-- holder_id copies charset/collation/length from users.id so the foreign key is valid.

SET @ucs  = (SELECT CHARACTER_SET_NAME FROM INFORMATION_SCHEMA.COLUMNS
             WHERE table_schema = DATABASE() AND table_name = 'users' AND column_name = 'id');
SET @ucol = (SELECT COLLATION_NAME FROM INFORMATION_SCHEMA.COLUMNS
             WHERE table_schema = DATABASE() AND table_name = 'users' AND column_name = 'id');
SET @ulen = (SELECT CHARACTER_MAXIMUM_LENGTH FROM INFORMATION_SCHEMA.COLUMNS
             WHERE table_schema = DATABASE() AND table_name = 'users' AND column_name = 'id');

-- If any of these three values is NULL, STOP and do not run the rest.
SELECT @ucs AS users_id_charset, @ucol AS users_id_collation, @ulen AS users_id_length;

SET @sql = (SELECT IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
   WHERE table_schema = DATABASE() AND table_name = 'itemskeluar' AND column_name = 'holder_id') > 0,
  'SELECT 1',
  CONCAT('ALTER TABLE `itemskeluar` ADD COLUMN `holder_id` VARCHAR(', @ulen, ') CHARACTER SET ', @ucs,
         ' COLLATE ', @ucol, ' NULL DEFAULT NULL AFTER `notes`, ADD KEY `idx_itemskeluar_holder_id` (`holder_id`)')
));
PREPARE s FROM @sql; EXECUTE s; DEALLOCATE PREPARE s;

SET @sql = (SELECT IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
   WHERE table_schema = DATABASE() AND table_name = 'itemsmasuk' AND column_name = 'holder_id') > 0,
  'SELECT 1',
  CONCAT('ALTER TABLE `itemsmasuk` ADD COLUMN `holder_id` VARCHAR(', @ulen, ') CHARACTER SET ', @ucs,
         ' COLLATE ', @ucol, ' NULL DEFAULT NULL AFTER `notes`, ADD KEY `idx_itemsmasuk_holder_id` (`holder_id`)')
));
PREPARE s FROM @sql; EXECUTE s; DEALLOCATE PREPARE s;

SET @sql = (SELECT IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
   WHERE table_schema = DATABASE() AND table_name = 'itemskeluar' AND column_name = 'transfer_ref') > 0,
  'SELECT 1',
  'ALTER TABLE `itemskeluar` ADD COLUMN `transfer_ref` CHAR(32) CHARACTER SET ascii COLLATE ascii_bin NULL DEFAULT NULL, ADD KEY `idx_itemskeluar_transfer_ref` (`transfer_ref`)'
));
PREPARE s FROM @sql; EXECUTE s; DEALLOCATE PREPARE s;

SET @sql = (SELECT IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
   WHERE table_schema = DATABASE() AND table_name = 'itemsmasuk' AND column_name = 'transfer_ref') > 0,
  'SELECT 1',
  'ALTER TABLE `itemsmasuk` ADD COLUMN `transfer_ref` CHAR(32) CHARACTER SET ascii COLLATE ascii_bin NULL DEFAULT NULL, ADD KEY `idx_itemsmasuk_transfer_ref` (`transfer_ref`)'
));
PREPARE s FROM @sql; EXECUTE s; DEALLOCATE PREPARE s;

SET @sql = (SELECT IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS
   WHERE table_schema = DATABASE() AND table_name = 'itemskeluar'
     AND constraint_name = 'fk_itemskeluar_holder' AND constraint_type = 'FOREIGN KEY') > 0,
  'SELECT 1',
  'ALTER TABLE `itemskeluar` ADD CONSTRAINT `fk_itemskeluar_holder` FOREIGN KEY (`holder_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT'
));
PREPARE s FROM @sql; EXECUTE s; DEALLOCATE PREPARE s;

SET @sql = (SELECT IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS
   WHERE table_schema = DATABASE() AND table_name = 'itemsmasuk'
     AND constraint_name = 'fk_itemsmasuk_holder' AND constraint_type = 'FOREIGN KEY') > 0,
  'SELECT 1',
  'ALTER TABLE `itemsmasuk` ADD CONSTRAINT `fk_itemsmasuk_holder` FOREIGN KEY (`holder_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT'
));
PREPARE s FROM @sql; EXECUTE s; DEALLOCATE PREPARE s;

SELECT table_name, column_name, column_type, collation_name, is_nullable
FROM INFORMATION_SCHEMA.COLUMNS
WHERE table_schema = DATABASE() AND table_name IN ('itemskeluar','itemsmasuk')
  AND column_name IN ('holder_id','transfer_ref')
ORDER BY table_name, column_name;

SELECT table_name, constraint_name, delete_rule, update_rule
FROM INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS
WHERE constraint_schema = DATABASE()
  AND constraint_name IN ('fk_itemskeluar_holder','fk_itemsmasuk_holder');

-- Rollback (manual, only if needed):
-- ALTER TABLE itemskeluar DROP FOREIGN KEY fk_itemskeluar_holder, DROP COLUMN holder_id, DROP COLUMN transfer_ref;
-- ALTER TABLE itemsmasuk  DROP FOREIGN KEY fk_itemsmasuk_holder,  DROP COLUMN holder_id, DROP COLUMN transfer_ref;
