-- 多条断言支持数据库迁移
-- 执行时间：2026-05-17

-- 1. 修改 inspection_items 表
ALTER TABLE inspection_items
  DROP COLUMN assertion_type,
  DROP COLUMN assertion_value,
  ADD COLUMN assertions JSON COMMENT '断言规则列表（JSON数组）',
  ADD COLUMN assertion_logic VARCHAR(10) DEFAULT 'and' COMMENT '断言逻辑：and/or';

-- 2. 修改 inspection_execution_details 表
ALTER TABLE inspection_execution_details
  DROP COLUMN assertion_type,
  DROP COLUMN assertion_value,
  ADD COLUMN assertions JSON COMMENT '断言规则列表（JSON数组）',
  ADD COLUMN assertion_logic VARCHAR(10) COMMENT '断言逻辑：and/or';
