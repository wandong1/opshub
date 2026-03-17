-- 为巡检项表添加拨测执行类型相关字段
-- 执行时间: 2026-03-17

ALTER TABLE `inspection_items`
ADD COLUMN `probe_category` VARCHAR(20) DEFAULT '' COMMENT '拨测分类: network/layer4/application/workflow',
ADD COLUMN `probe_type` VARCHAR(20) DEFAULT '' COMMENT '拨测类型: ping/tcp/udp/http/https/websocket',
ADD COLUMN `probe_config_id` BIGINT UNSIGNED DEFAULT 0 COMMENT '关联拨测配置ID',
ADD INDEX `idx_inspection_items_probe_config` (`probe_config_id`);
