-- 智能巡检模块数据库迁移脚本
-- 执行日期: 2026-03-13
-- 说明: 添加执行策略和并发控制相关字段

-- 为 inspection_groups 表添加执行策略相关字段
ALTER TABLE inspection_groups
ADD COLUMN IF NOT EXISTS execution_strategy VARCHAR(20) DEFAULT 'concurrent' COMMENT '执行策略 concurrent/sequential',
ADD COLUMN IF NOT EXISTS concurrency INT DEFAULT 50 COMMENT '并发数量';

-- 更新 inspection_items 表的 host_match_type 字段注释
ALTER TABLE inspection_items
MODIFY COLUMN host_match_type VARCHAR(20) DEFAULT 'tag' COMMENT '主机匹配方式 tag/name/id';

-- 更新 inspection_items 表的 host_tags 字段注释
ALTER TABLE inspection_items
MODIFY COLUMN host_tags TEXT COMMENT '主机标签或主机名关键词（JSON数组）';

-- 验证迁移结果
SELECT
    TABLE_NAME,
    COLUMN_NAME,
    COLUMN_TYPE,
    COLUMN_DEFAULT,
    COLUMN_COMMENT
FROM
    INFORMATION_SCHEMA.COLUMNS
WHERE
    TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME IN ('inspection_groups', 'inspection_items')
    AND COLUMN_NAME IN ('execution_strategy', 'concurrency', 'host_match_type', 'host_tags')
ORDER BY
    TABLE_NAME, ORDINAL_POSITION;
