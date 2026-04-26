-- PromQL 巡检功能数据库迁移脚本
-- 执行时间：2026-04-25

-- ===== 1. 巡检组表：删除旧字段，新增单数据源关联 =====
ALTER TABLE inspection_groups
DROP COLUMN prometheus_url,
DROP COLUMN prometheus_username,
DROP COLUMN prometheus_password;

-- 新增单数据源 ID 字段
ALTER TABLE inspection_groups
ADD COLUMN datasource_id INT UNSIGNED DEFAULT 0 COMMENT '关联的数据源ID（单个）' AFTER labels,
ADD INDEX idx_datasource_id (datasource_id);

-- ===== 2. 巡检项表：新增 PromQL 查询类型字段 =====
-- 检查字段是否已存在
ALTER TABLE inspection_items
ADD COLUMN promql_query_type VARCHAR(20) DEFAULT 'instant' COMMENT 'PromQL查询类型: instant/range' AFTER prom_ql_query;

-- 扩展 assertion_value 字段长度（支持 JSON 配置）
ALTER TABLE inspection_items
MODIFY COLUMN assertion_value TEXT COMMENT '断言配置（支持 JSON 格式）';

-- ===== 3. 巡检执行详情表：新增 PromQL 相关字段 =====
ALTER TABLE inspection_execution_details
ADD COLUMN datasource_id INT UNSIGNED DEFAULT 0 COMMENT '使用的数据源ID' AFTER risk_level,
ADD COLUMN promql TEXT COMMENT '实际执行的PromQL（变量已替换）' AFTER datasource_id,
ADD COLUMN promql_result LONGTEXT COMMENT '原始查询结果（JSON）' AFTER promql,
ADD COLUMN metric_value DECIMAL(20,4) DEFAULT NULL COMMENT '提取的指标值' AFTER promql_result,
ADD COLUMN metric_labels JSON COMMENT '指标标签（JSON）' AFTER metric_value,
ADD COLUMN assertion_pass BOOLEAN DEFAULT NULL COMMENT '断言是否通过' AFTER metric_labels,
ADD COLUMN assertion_rule TEXT COMMENT '应用的断言规则（JSON）' AFTER assertion_pass,
ADD COLUMN failure_reason TEXT COMMENT '失败原因详情' AFTER assertion_rule,
ADD INDEX idx_datasource_id (datasource_id);

-- ===== 4. 数据迁移说明 =====
-- 如果现有巡检组配置了 prometheus_url，需要手动在告警管理中创建对应的数据源，
-- 然后更新 inspection_groups.datasource_id 字段
