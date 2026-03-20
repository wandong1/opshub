-- 集成管理功能迁移脚本
-- 新增：集成管理菜单、监控大屏菜单、菜单管理查看按钮

-- 1. 集成管理菜单（系统管理子菜单，parent_id=1，sort=8）
INSERT IGNORE INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES (14, '集成管理', 'integrations', 2, 1, '/system/integrations', 'system/Integrations', 'Link', 8, 1, 1, '', '', NOW(), NOW());

-- 2. 集成管理按钮权限
INSERT IGNORE INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (400, '查看集成配置', 'integrations:view', 3, 14, '', '', '', 1, 1, 1, '/api/v1/system/integrations', 'GET', NOW(), NOW()),
  (401, '保存集成配置', 'integrations:save', 3, 14, '', '', '', 2, 1, 1, '/api/v1/system/integrations', 'PUT', NOW(), NOW());

-- 3. 监控大屏菜单（监控中心子菜单，parent_id=42，sort=0 排第一）
INSERT IGNORE INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES (402, '监控大屏', 'monitor_dashboard', 2, 42, '/monitor/grafana', 'monitor/GrafanaDashboard', 'Dashboard', 0, 1, 1, '', '', NOW(), NOW());

-- 4. 菜单管理补充「查看」按钮（parent_id=5）
INSERT IGNORE INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES (403, '查看菜单', 'menus:view', 3, 5, '', '', '', 9, 1, 1, '/api/v1/menus', 'GET', NOW(), NOW());
