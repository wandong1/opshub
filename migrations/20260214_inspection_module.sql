-- ============================================================
-- 智能巡检模块 — 数据库表结构 + 菜单与按钮权限
-- ============================================================

SET FOREIGN_KEY_CHECKS = 0;

-- ============================================================
-- 1. 业务数据表
-- ============================================================

-- 1.1 拨测配置
CREATE TABLE IF NOT EXISTS `probe_configs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '拨测名称',
  `type` varchar(20) NOT NULL COMMENT '拨测类型: ping/tcp/udp',
  `category` varchar(20) NOT NULL DEFAULT 'network' COMMENT '拨测分类: network/layer4/application/workflow/middleware',
  `target` varchar(255) NOT NULL COMMENT '目标地址',
  `port` int NOT NULL DEFAULT 0 COMMENT '端口（TCP/UDP）',
  `group_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '关联 AssetGroup',
  `timeout` int NOT NULL DEFAULT 5 COMMENT '超时秒数',
  `count` int NOT NULL DEFAULT 4 COMMENT 'Ping 次数',
  `packet_size` int NOT NULL DEFAULT 64 COMMENT 'Ping 包大小',
  `description` varchar(500) DEFAULT '' COMMENT '描述',
  `tags` varchar(500) DEFAULT '' COMMENT '逗号分隔标签',
  `status` tinyint NOT NULL DEFAULT 1 COMMENT '1=启用 0=禁用',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_probe_configs_group_id` (`group_id`),
  KEY `idx_probe_configs_category` (`category`),
  KEY `idx_probe_configs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='拨测配置';

-- 1.2 调度任务
CREATE TABLE IF NOT EXISTS `probe_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '任务名称',
  `group_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '业务分组（冗余）',
  `cron_expr` varchar(50) NOT NULL COMMENT '秒级 cron 表达式',
  `pushgateway_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '关联 Pushgateway',
  `concurrency` int NOT NULL DEFAULT 5 COMMENT '并发执行上限',
  `status` tinyint NOT NULL DEFAULT 1 COMMENT '1=启用 0=禁用',
  `last_run_at` datetime(3) DEFAULT NULL COMMENT '最后执行时间',
  `last_result` varchar(20) DEFAULT '' COMMENT 'success/fail',
  `description` varchar(500) DEFAULT '' COMMENT '描述',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_probe_tasks_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='拨测调度任务';

-- 1.2.1 任务-配置关联表（多对多）
CREATE TABLE IF NOT EXISTS `probe_task_configs` (
  `probe_task_id` bigint unsigned NOT NULL COMMENT '任务ID',
  `probe_config_id` bigint unsigned NOT NULL COMMENT '配置ID',
  PRIMARY KEY (`probe_task_id`, `probe_config_id`),
  KEY `idx_probe_task_configs_config_id` (`probe_config_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='任务-拨测配置关联';

-- 1.3 拨测结果
CREATE TABLE IF NOT EXISTS `probe_results` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `probe_task_id` bigint unsigned NOT NULL DEFAULT 0,
  `probe_config_id` bigint unsigned NOT NULL DEFAULT 0,
  `success` tinyint(1) NOT NULL DEFAULT 0,
  `latency` double NOT NULL DEFAULT 0 COMMENT '总延迟 ms',
  `packet_loss` double NOT NULL DEFAULT 0 COMMENT '丢包率（Ping）',
  `ping_rtt_avg` double NOT NULL DEFAULT 0,
  `ping_rtt_min` double NOT NULL DEFAULT 0,
  `ping_rtt_max` double NOT NULL DEFAULT 0,
  `ping_stddev` double NOT NULL DEFAULT 0,
  `ping_packets_sent` int NOT NULL DEFAULT 0,
  `ping_packets_recv` int NOT NULL DEFAULT 0,
  `tcp_connect_time` double NOT NULL DEFAULT 0 COMMENT 'TCP 连接耗时 ms',
  `udp_write_time` double NOT NULL DEFAULT 0,
  `udp_read_time` double NOT NULL DEFAULT 0,
  `error_message` text,
  `detail` text COMMENT 'JSON 扩展详情',
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_probe_results_task_id` (`probe_task_id`),
  KEY `idx_probe_results_config_id` (`probe_config_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='拨测结果';

-- 1.4 Pushgateway 配置
CREATE TABLE IF NOT EXISTS `pushgateway_configs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '名称',
  `url` varchar(255) NOT NULL COMMENT 'Pushgateway 地址',
  `username` varchar(100) DEFAULT '' COMMENT 'Basic Auth 用户名',
  `password` varchar(255) DEFAULT '' COMMENT 'Basic Auth 密码',
  `is_default` tinyint NOT NULL DEFAULT 0 COMMENT '是否默认',
  `status` tinyint NOT NULL DEFAULT 1 COMMENT '1=启用 0=禁用',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_pushgateway_configs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Pushgateway 配置';

-- ============================================================
-- 2. 菜单与按钮权限（ID 从 400 起，避免与已有数据冲突）
-- ============================================================

-- 2.1 目录 (type=1)
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (400, '智能巡检', 'inspection-dir', 1, 0, '', '', 'Monitor', 80, 1, 1, '', '', NOW(), NOW());

-- 2.2 页面菜单 (type=2, parent_id=400)
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (401, '拨测管理',       'inspection:probes',       2, 400, '/inspection/probes',       'inspection/ProbeManagement', 'Connection', 0, 1, 1, '', '', NOW(), NOW()),
  (402, '任务调度',       'inspection:tasks',        2, 400, '/inspection/tasks',        'inspection/TaskSchedule',    'Timer',      1, 1, 1, '', '', NOW(), NOW()),
  (403, 'Pushgateway配置', 'inspection:pushgateways', 2, 400, '/inspection/pushgateways', 'inspection/PushgatewayConfig', 'DataLine', 2, 1, 1, '', '', NOW(), NOW());

-- 2.3 拨测管理按钮 (type=3, parent_id=401)
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (410, '查询拨测', 'inspection:probes:list',    3, 401, '', '', '', 0, 1, 1, '/api/v1/inspection/probes',          'GET',  NOW(), NOW()),
  (411, '新增拨测', 'inspection:probes:create',  3, 401, '', '', '', 1, 1, 1, '/api/v1/inspection/probes',          'POST', NOW(), NOW()),
  (412, '编辑拨测', 'inspection:probes:update',  3, 401, '', '', '', 2, 1, 1, '/api/v1/inspection/probes/:id',      'PUT',  NOW(), NOW()),
  (413, '删除拨测', 'inspection:probes:delete',  3, 401, '', '', '', 3, 1, 1, '/api/v1/inspection/probes/:id',      'DELETE', NOW(), NOW()),
  (414, '导入拨测', 'inspection:probes:import',  3, 401, '', '', '', 4, 1, 1, '/api/v1/inspection/probes/import',   'POST', NOW(), NOW()),
  (415, '导出拨测', 'inspection:probes:export',  3, 401, '', '', '', 5, 1, 1, '/api/v1/inspection/probes/export',   'GET',  NOW(), NOW()),
  (416, '执行拨测', 'inspection:probes:execute', 3, 401, '', '', '', 6, 1, 1, '/api/v1/inspection/probes/:id/run',  'POST', NOW(), NOW());

-- 2.4 任务调度按钮 (type=3, parent_id=402)
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (420, '查询任务', 'inspection:tasks:list',    3, 402, '', '', '', 0, 1, 1, '/api/v1/inspection/tasks',              'GET',  NOW(), NOW()),
  (421, '新增任务', 'inspection:tasks:create',  3, 402, '', '', '', 1, 1, 1, '/api/v1/inspection/tasks',              'POST', NOW(), NOW()),
  (422, '编辑任务', 'inspection:tasks:update',  3, 402, '', '', '', 2, 1, 1, '/api/v1/inspection/tasks/:id',          'PUT',  NOW(), NOW()),
  (423, '删除任务', 'inspection:tasks:delete',  3, 402, '', '', '', 3, 1, 1, '/api/v1/inspection/tasks/:id',          'DELETE', NOW(), NOW()),
  (424, '启停任务', 'inspection:tasks:toggle',  3, 402, '', '', '', 4, 1, 1, '/api/v1/inspection/tasks/:id/toggle',   'PUT',  NOW(), NOW()),
  (425, '查看结果', 'inspection:tasks:results', 3, 402, '', '', '', 5, 1, 1, '/api/v1/inspection/tasks/:id/results',  'GET',  NOW(), NOW());

-- 2.5 Pushgateway 配置按钮 (type=3, parent_id=403)
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (430, '查询Pushgateway', 'inspection:pushgateways:list',   3, 403, '', '', '', 0, 1, 1, '/api/v1/inspection/pushgateways',          'GET',  NOW(), NOW()),
  (431, '新增Pushgateway', 'inspection:pushgateways:create', 3, 403, '', '', '', 1, 1, 1, '/api/v1/inspection/pushgateways',          'POST', NOW(), NOW()),
  (432, '编辑Pushgateway', 'inspection:pushgateways:update', 3, 403, '', '', '', 2, 1, 1, '/api/v1/inspection/pushgateways/:id',      'PUT',  NOW(), NOW()),
  (433, '删除Pushgateway', 'inspection:pushgateways:delete', 3, 403, '', '', '', 3, 1, 1, '/api/v1/inspection/pushgateways/:id',      'DELETE', NOW(), NOW()),
  (434, '测试Pushgateway', 'inspection:pushgateways:test',   3, 403, '', '', '', 4, 1, 1, '/api/v1/inspection/pushgateways/:id/test', 'POST', NOW(), NOW());

-- ============================================================
-- 3. 按钮 API 关联 (sys_menu_api)
-- ============================================================

-- 3.1 拨测管理
INSERT INTO `sys_menu_api` (`menu_id`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (410, '/api/v1/inspection/probes',          'GET',    NOW(), NOW()),
  (411, '/api/v1/inspection/probes',          'POST',   NOW(), NOW()),
  (412, '/api/v1/inspection/probes/:id',      'PUT',    NOW(), NOW()),
  (413, '/api/v1/inspection/probes/:id',      'DELETE',  NOW(), NOW()),
  (414, '/api/v1/inspection/probes/import',   'POST',   NOW(), NOW()),
  (415, '/api/v1/inspection/probes/export',   'GET',    NOW(), NOW()),
  (416, '/api/v1/inspection/probes/:id/run',  'POST',   NOW(), NOW());

-- 3.2 任务调度
INSERT INTO `sys_menu_api` (`menu_id`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (420, '/api/v1/inspection/tasks',              'GET',    NOW(), NOW()),
  (421, '/api/v1/inspection/tasks',              'POST',   NOW(), NOW()),
  (422, '/api/v1/inspection/tasks/:id',          'PUT',    NOW(), NOW()),
  (423, '/api/v1/inspection/tasks/:id',          'DELETE',  NOW(), NOW()),
  (424, '/api/v1/inspection/tasks/:id/toggle',   'PUT',    NOW(), NOW()),
  (425, '/api/v1/inspection/tasks/:id/results',  'GET',    NOW(), NOW());

-- 3.3 Pushgateway 配置
INSERT INTO `sys_menu_api` (`menu_id`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (430, '/api/v1/inspection/pushgateways',          'GET',    NOW(), NOW()),
  (431, '/api/v1/inspection/pushgateways',          'POST',   NOW(), NOW()),
  (432, '/api/v1/inspection/pushgateways/:id',      'PUT',    NOW(), NOW()),
  (433, '/api/v1/inspection/pushgateways/:id',      'DELETE',  NOW(), NOW()),
  (434, '/api/v1/inspection/pushgateways/:id/test', 'POST',   NOW(), NOW());

-- ============================================================
-- 4. 为管理员角色 (role_id=1) 分配所有菜单和按钮权限
-- ============================================================

INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  -- 目录 + 页面菜单
  (1, 400), (1, 401), (1, 402), (1, 403),
  -- 拨测管理按钮
  (1, 410), (1, 411), (1, 412), (1, 413), (1, 414), (1, 415), (1, 416),
  -- 任务调度按钮
  (1, 420), (1, 421), (1, 422), (1, 423), (1, 424), (1, 425),
  -- Pushgateway 配置按钮
  (1, 430), (1, 431), (1, 432), (1, 433), (1, 434);

SET FOREIGN_KEY_CHECKS = 1;
