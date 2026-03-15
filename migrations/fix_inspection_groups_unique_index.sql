-- 修复巡检组唯一索引，支持软删除
-- 删除旧的唯一索引
ALTER TABLE `inspection_groups` DROP INDEX IF EXISTS `idx_inspection_groups_name`;

-- 创建新的复合唯一索引（name + deleted_at）
-- 这样可以允许相同名称的记录，只要其中一个是软删除的
ALTER TABLE `inspection_groups` ADD UNIQUE INDEX `idx_inspection_groups_name_deleted` (`name`, `deleted_at`);
