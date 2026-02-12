-- Copyright (c) 2026 YDCloud
--
-- Permission is hereby granted, free of charge, to any person obtaining a copy of
-- this software and associated documentation files (the "Software"), to deal in
-- the Software without restriction, including without limitation the rights to
-- use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
-- the Software, and to permit persons to whom the Software is furnished to do so,
-- subject to the following conditions:
--
-- The above copyright notice and this permission notice shall be included in all
-- copies or substantial portions of the Software.
--
-- THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
-- IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
-- FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
-- COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
-- IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
-- CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

-- OpsHub Database Initialization Script
-- 创建数据库的所有必要表和初始化数据
-- 执行前请确保数据库已创建: CREATE DATABASE opshub CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ============================================================
-- 1. RBAC 系统表
-- ============================================================
--
-- 用户表
CREATE TABLE IF NOT EXISTS `sys_user` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(50) NOT NULL COMMENT '用户名',
  `password` varchar(255) NOT NULL COMMENT '密码',
  `real_name` varchar(50) COMMENT '真实姓名',
  `email` varchar(100) COMMENT '邮箱',
  `phone` varchar(20) COMMENT '手机号',
  `avatar` varchar(255) COMMENT '头像',
  `status` tinyint DEFAULT 1 COMMENT '状态 1:启用 0:禁用',
  `department_id` bigint unsigned DEFAULT 0 COMMENT '部门ID',
  `bio` text COMMENT '个人简介',
  `last_login_at` datetime COMMENT '最后登录时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username_deleted` (`username`, `deleted_at`),
  KEY `idx_department_id` (`department_id`),
  KEY `idx_status` (`status`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 角色表
CREATE TABLE IF NOT EXISTS `sys_role` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL COMMENT '角色名称',
  `code` varchar(50) NOT NULL COMMENT '角色编码',
  `description` varchar(200) COMMENT '角色描述',
  `sort` int DEFAULT 0 COMMENT '排序',
  `status` tinyint DEFAULT 1 COMMENT '状态 1:启用 0:禁用',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`, `deleted_at`),
  UNIQUE KEY `uk_code` (`code`, `deleted_at`),
  KEY `idx_sort` (`sort`),
  KEY `idx_status` (`status`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 部门表
CREATE TABLE IF NOT EXISTS `sys_department` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL COMMENT '部门名称',
  `code` varchar(50) COMMENT '部门编码',
  `parent_id` bigint unsigned DEFAULT 0 COMMENT '父部门ID',
  `dept_type` tinyint DEFAULT 3 COMMENT '部门类型 1:公司 2:中心 3:部门',
  `sort` int DEFAULT 0 COMMENT '排序',
  `status` tinyint DEFAULT 1 COMMENT '状态 1:启用 0:禁用',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`, `deleted_at`),
  KEY `idx_parent_id` (`parent_id`),
  KEY `idx_dept_type` (`dept_type`),
  KEY `idx_sort` (`sort`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 菜单表
CREATE TABLE IF NOT EXISTS `sys_menu` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL COMMENT '菜单名称',
  `code` varchar(50) COMMENT '菜单编码',
  `type` tinyint COMMENT '菜单类型 1:目录 2:菜单 3:按钮',
  `parent_id` bigint unsigned DEFAULT 0 COMMENT '父菜单ID',
  `path` varchar(200) COMMENT '路由路径',
  `component` varchar(200) COMMENT '组件路径',
  `icon` varchar(100) COMMENT '图标',
  `sort` int DEFAULT 0 COMMENT '排序',
  `visible` tinyint DEFAULT 1 COMMENT '是否显示 1:显示 0:隐藏',
  `status` tinyint DEFAULT 1 COMMENT '状态 1:启用 0:禁用',
  `api_path` varchar(200) COMMENT '后端API路径',
  `api_method` varchar(10) COMMENT 'HTTP方法(GET/POST/PUT/DELETE)',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`, `deleted_at`),
  KEY `idx_parent_id` (`parent_id`),
  KEY `idx_type` (`type`),
  KEY `idx_sort` (`sort`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 职位表
CREATE TABLE IF NOT EXISTS `sys_position` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `post_name` varchar(50) NOT NULL COMMENT '职位名称',
  `post_code` varchar(50) NOT NULL COMMENT '职位编码',
  `post_status` tinyint DEFAULT 1 COMMENT '职位状态 1:启用 2:禁用',
  `remark` varchar(200) COMMENT '备注',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_post_code` (`post_code`, `deleted_at`),
  KEY `idx_post_status` (`post_status`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户-角色关联表
CREATE TABLE IF NOT EXISTS `sys_user_role` (
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `role_id` bigint unsigned NOT NULL COMMENT '角色ID',
  PRIMARY KEY (`user_id`, `role_id`),
  KEY `idx_role_id` (`role_id`),
  CONSTRAINT `fk_user_role_user` FOREIGN KEY (`user_id`) REFERENCES `sys_user` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_user_role_role` FOREIGN KEY (`role_id`) REFERENCES `sys_role` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 角色-菜单关联表
CREATE TABLE IF NOT EXISTS `sys_role_menu` (
  `role_id` bigint unsigned NOT NULL COMMENT '角色ID',
  `menu_id` bigint unsigned NOT NULL COMMENT '菜单ID',
  PRIMARY KEY (`role_id`, `menu_id`),
  KEY `idx_menu_id` (`menu_id`),
  CONSTRAINT `fk_role_menu_role` FOREIGN KEY (`role_id`) REFERENCES `sys_role` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_role_menu_menu` FOREIGN KEY (`menu_id`) REFERENCES `sys_menu` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 菜单-API关联表（一个按钮可绑定多个API）
CREATE TABLE IF NOT EXISTS `sys_menu_api` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `menu_id` bigint unsigned NOT NULL COMMENT '菜单ID',
  `api_path` varchar(200) NOT NULL COMMENT 'API路径',
  `api_method` varchar(10) NOT NULL COMMENT 'HTTP方法',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_menu_id` (`menu_id`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户-职位关联表
CREATE TABLE IF NOT EXISTS `sys_user_position` (
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `position_id` bigint unsigned NOT NULL COMMENT '职位ID',
  PRIMARY KEY (`user_id`, `position_id`),
  KEY `idx_position_id` (`position_id`),
  CONSTRAINT `fk_user_position_user` FOREIGN KEY (`user_id`) REFERENCES `sys_user` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_user_position_position` FOREIGN KEY (`position_id`) REFERENCES `sys_position` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 系统配置表
CREATE TABLE IF NOT EXISTS `sys_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `key` varchar(100) NOT NULL COMMENT '配置键',
  `value` text COMMENT '配置值',
  `type` varchar(20) DEFAULT 'string' COMMENT '配置类型(string/int/bool/json)',
  `group` varchar(50) COMMENT '配置分组(basic/security)',
  `remark` varchar(200) COMMENT '备注说明',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_key` (`key`),
  KEY `idx_group` (`group`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户登录失败记录表
CREATE TABLE IF NOT EXISTS `sys_user_login_attempt` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(50) NOT NULL COMMENT '用户名',
  `fail_count` int DEFAULT 0 COMMENT '失败次数',
  `last_fail_at` datetime COMMENT '最后失败时间',
  `locked_until` datetime COMMENT '锁定截止时间',
  PRIMARY KEY (`id`),
  KEY `idx_username` (`username`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- 2. 审计日志表
-- ============================================================

-- 操作审计日志表
CREATE TABLE IF NOT EXISTS `sys_operation_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned COMMENT '用户ID',
  `username` varchar(50) COMMENT '用户名',
  `real_name` varchar(50) COMMENT '真实姓名',
  `module` varchar(50) COMMENT '操作模块',
  `action` varchar(50) COMMENT '操作动作',
  `description` varchar(200) COMMENT '操作描述',
  `method` varchar(10) COMMENT '请求方法',
  `path` varchar(200) COMMENT '请求路径',
  `params` text COMMENT '请求参数',
  `status` int COMMENT '响应状态码',
  `error_msg` text COMMENT '错误信息',
  `cost_time` bigint COMMENT '耗时(毫秒)',
  `ip` varchar(50) COMMENT '客户端IP',
  `user_agent` varchar(500) COMMENT '用户代理',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_username` (`username`),
  KEY `idx_action` (`action`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 登录审计日志表
CREATE TABLE IF NOT EXISTS `sys_login_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned COMMENT '用户ID',
  `username` varchar(50) COMMENT '用户名',
  `real_name` varchar(50) COMMENT '真实姓名',
  `login_type` varchar(20) COMMENT '登录类型',
  `login_status` varchar(20) COMMENT '登录状态',
  `login_time` datetime COMMENT '登录时间',
  `logout_time` datetime COMMENT '登出时间',
  `ip` varchar(50) COMMENT '登录IP',
  `location` varchar(100) COMMENT '登录地点',
  `user_agent` varchar(500) COMMENT '用户代理',
  `fail_reason` varchar(200) COMMENT '失败原因',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_username` (`username`),
  KEY `idx_login_time` (`login_time`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 数据变更审计日志表
CREATE TABLE IF NOT EXISTS `sys_data_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned COMMENT '用户ID',
  `username` varchar(50) COMMENT '用户名',
  `real_name` varchar(50) COMMENT '真实姓名',
  `table_name` varchar(50) COMMENT '操作表名',
  `record_id` bigint unsigned COMMENT '记录ID',
  `action` varchar(20) COMMENT '操作类型',
  `old_data` longtext COMMENT '旧数据',
  `new_data` longtext COMMENT '新数据',
  `diff_fields` text COMMENT '变更字段',
  `ip` varchar(50) COMMENT '客户端IP',
  `user_agent` varchar(500) COMMENT '用户代理',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_table_name` (`table_name`),
  KEY `idx_record_id` (`record_id`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- 3. 资产管理表
-- ============================================================

-- 资产组表
CREATE TABLE IF NOT EXISTS `asset_group` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '组名称',
  `code` varchar(50) COMMENT '组编码',
  `parent_id` bigint unsigned DEFAULT 0 COMMENT '父组ID',
  `description` varchar(500) COMMENT '描述',
  `sort` int DEFAULT 0 COMMENT '排序',
  `status` tinyint DEFAULT 1 COMMENT '状态 1:启用 0:禁用',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`, `deleted_at`),
  KEY `idx_parent_id` (`parent_id`),
  KEY `idx_sort` (`sort`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 凭证表
CREATE TABLE IF NOT EXISTS `credentials` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '凭证名称',
  `type` varchar(20) NOT NULL COMMENT '凭证类型 password/key',
  `username` varchar(100) COMMENT '用户名',
  `password` varchar(500) COMMENT '密码(加密)',
  `private_key` text COMMENT '私钥(加密)',
  `passphrase` varchar(500) COMMENT '私钥密码(加密)',
  `description` varchar(500) COMMENT '描述',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_type` (`type`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 主机表
CREATE TABLE IF NOT EXISTS `hosts` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '主机名称',
  `group_id` bigint unsigned COMMENT '所属组ID',
  `type` varchar(20) DEFAULT 'self' COMMENT '主机类型 self:自建 cloud:云实例',
  `cloud_provider` varchar(50) COMMENT '云厂商',
  `cloud_instance_id` varchar(100) COMMENT '云实例ID',
  `cloud_account_id` bigint unsigned COMMENT '云账户ID',
  `ssh_user` varchar(50) NOT NULL COMMENT 'SSH用户',
  `ip` varchar(50) NOT NULL COMMENT 'IP地址',
  `port` int DEFAULT 22 COMMENT 'SSH端口',
  `credential_id` bigint unsigned COMMENT '凭证ID',
  `tags` varchar(500) COMMENT '标签',
  `description` varchar(500) COMMENT '描述',
  `status` tinyint DEFAULT -1 COMMENT '状态 1:在线 0:离线 -1:未知',
  `last_seen` datetime COMMENT '最后看到时间',
  `os` varchar(100) COMMENT '操作系统',
  `kernel` varchar(100) COMMENT '内核版本',
  `arch` varchar(50) COMMENT '架构',
  `cpu_info` text COMMENT 'CPU信息',
  `cpu_cores` int COMMENT 'CPU核心数',
  `cpu_usage` float COMMENT 'CPU使用率',
  `memory_total` bigint COMMENT '总内存',
  `memory_used` bigint COMMENT '已用内存',
  `memory_usage` float COMMENT '内存使用率',
  `disk_total` bigint COMMENT '总磁盘',
  `disk_used` bigint COMMENT '已用磁盘',
  `disk_usage` float COMMENT '磁盘使用率',
  `uptime` varchar(100) COMMENT '运行时间',
  `hostname` varchar(100) COMMENT '主机名',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_group_id` (`group_id`),
  KEY `idx_ip` (`ip`),
  KEY `idx_status` (`status`),
  KEY `idx_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_hosts_group` FOREIGN KEY (`group_id`) REFERENCES `asset_group` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 云账户表
CREATE TABLE IF NOT EXISTS `cloud_accounts` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '账户名称',
  `provider` varchar(50) NOT NULL COMMENT '云厂商',
  `access_key` varchar(200) NOT NULL COMMENT 'AccessKey',
  `secret_key` varchar(500) NOT NULL COMMENT 'SecretKey',
  `region` varchar(100) COMMENT '默认地域',
  `description` varchar(500) COMMENT '描述',
  `status` tinyint DEFAULT 1 COMMENT '状态 1:启用 0:禁用',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_provider` (`provider`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 角色资产权限表
CREATE TABLE IF NOT EXISTS `sys_role_asset_permission` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `role_id` bigint unsigned NOT NULL COMMENT '角色ID',
  `asset_group_id` bigint unsigned NOT NULL COMMENT '资产组ID',
  `host_ids` json COMMENT '主机ID列表',
  `permissions` int unsigned DEFAULT 63 COMMENT '权限位',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_asset` (`role_id`, `asset_group_id`, `deleted_at`),
  KEY `idx_asset_group_id` (`asset_group_id`),
  KEY `idx_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_role_asset_perm_role` FOREIGN KEY (`role_id`) REFERENCES `sys_role` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_role_asset_perm_group` FOREIGN KEY (`asset_group_id`) REFERENCES `asset_group` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 中间件管理表
CREATE TABLE IF NOT EXISTS `middlewares` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '中间件名称',
  `type` varchar(20) NOT NULL COMMENT '类型: mysql/redis/clickhouse/mongodb/kafka/milvus',
  `group_id` bigint unsigned DEFAULT 0 COMMENT '所属业务分组',
  `host_id` bigint unsigned DEFAULT 0 COMMENT '关联主机（可选）',
  `host` varchar(255) NOT NULL COMMENT '连接地址',
  `port` int NOT NULL COMMENT '连接端口',
  `username` varchar(100) COMMENT '用户名',
  `password` varchar(500) COMMENT '密码（加密存储）',
  `database_name` varchar(100) COMMENT '默认数据库/索引',
  `connection_params` text COMMENT '额外连接参数（JSON）',
  `tags` varchar(500) COMMENT '标签',
  `description` varchar(500) COMMENT '备注',
  `status` tinyint DEFAULT -1 COMMENT '状态 1:在线 0:离线 -1:未知',
  `version` varchar(50) COMMENT '中间件版本',
  `last_checked` datetime COMMENT '最后检测时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_type` (`type`),
  KEY `idx_group_id` (`group_id`),
  KEY `idx_host_id` (`host_id`),
  KEY `idx_status` (`status`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 角色中间件权限表
CREATE TABLE IF NOT EXISTS `sys_role_middleware_permission` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `role_id` bigint unsigned NOT NULL COMMENT '角色ID',
  `asset_group_id` bigint unsigned NOT NULL COMMENT '资产组ID',
  `middleware_ids` json COMMENT '中间件ID列表（空=整个分组）',
  `permissions` int unsigned DEFAULT 31 COMMENT '权限位',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_mw_group` (`role_id`, `asset_group_id`, `deleted_at`),
  KEY `idx_asset_group_id` (`asset_group_id`),
  KEY `idx_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_role_mw_perm_role` FOREIGN KEY (`role_id`) REFERENCES `sys_role` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_role_mw_perm_group` FOREIGN KEY (`asset_group_id`) REFERENCES `asset_group` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- SSH终端会话记录表（资产管理-终端审计）
CREATE TABLE IF NOT EXISTS `ssh_terminal_sessions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `host_id` bigint unsigned NOT NULL COMMENT '主机ID',
  `host_name` varchar(100) COMMENT '主机名称',
  `host_ip` varchar(50) COMMENT '主机IP',
  `user_id` bigint unsigned NOT NULL COMMENT '操作用户ID',
  `username` varchar(100) COMMENT '用户名',
  `recording_path` varchar(500) COMMENT '录制文件路径',
  `duration` int COMMENT '会话时长(秒)',
  `file_size` bigint COMMENT '文件大小(字节)',
  `status` varchar(20) DEFAULT 'recording' COMMENT '会话状态 recording/completed/failed',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_host_id` (`host_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- 4. 任务管理表 (Task Plugin)
-- ============================================================

-- 任务模板表
CREATE TABLE IF NOT EXISTS `job_templates` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL COMMENT '模板名称',
  `code` varchar(100) NOT NULL COMMENT '模板编码',
  `description` text COMMENT '模板描述',
  `content` longtext NOT NULL COMMENT '模板内容',
  `variables` json COMMENT '变量定义',
  `category` varchar(50) NOT NULL COMMENT '分类 script/ansible/module',
  `platform` varchar(50) COMMENT '平台 linux/windows',
  `timeout` int DEFAULT 300 COMMENT '超时时间(秒)',
  `sort` int DEFAULT 0 COMMENT '排序',
  `status` tinyint DEFAULT 1 COMMENT '状态 0:禁用 1:启用',
  `created_by` bigint unsigned NOT NULL COMMENT '创建者ID',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`, `deleted_at`),
  KEY `idx_category` (`category`),
  KEY `idx_sort` (`sort`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 任务执行表
CREATE TABLE IF NOT EXISTS `job_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL COMMENT '任务名称',
  `template_id` bigint unsigned COMMENT '模板ID',
  `task_type` varchar(50) NOT NULL COMMENT '任务类型 manual/ansible/cron',
  `status` varchar(50) DEFAULT 'pending' COMMENT '状态 pending/running/success/failed',
  `target_hosts` text COMMENT '目标主机列表(JSON)',
  `parameters` json COMMENT '执行参数',
  `execute_time` datetime COMMENT '执行时间',
  `result` json COMMENT '执行结果',
  `error_message` text COMMENT '错误信息',
  `created_by` bigint unsigned NOT NULL COMMENT '创建者ID',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_template_id` (`template_id`),
  KEY `idx_task_type` (`task_type`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Ansible任务表
CREATE TABLE IF NOT EXISTS `ansible_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL COMMENT '任务名称',
  `playbook_content` longtext COMMENT 'Playbook内容',
  `playbook_path` varchar(500) COMMENT 'Playbook路径',
  `inventory` text COMMENT '清单(JSON)',
  `extra_vars` json COMMENT '额外变量',
  `tags` varchar(500) COMMENT '标签',
  `fork` int DEFAULT 5 COMMENT '并发数',
  `timeout` int DEFAULT 600 COMMENT '超时时间(秒)',
  `verbose` varchar(20) DEFAULT 'v' COMMENT '日志级别',
  `status` varchar(50) DEFAULT 'pending' COMMENT '状态 pending/running/success/failed/cancelled',
  `last_run_time` datetime COMMENT '最后执行时间',
  `last_run_result` json COMMENT '最后执行结果',
  `created_by` bigint unsigned NOT NULL COMMENT '创建者ID',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_status` (`status`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- 5. Kubernetes 插件表
-- ============================================================

-- Kubernetes集群表
CREATE TABLE IF NOT EXISTS `k8s_clusters` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '集群名称',
  `alias` varchar(100) COMMENT '集群别名',
  `api_endpoint` varchar(500) NOT NULL COMMENT 'API地址',
  `kube_config` text NOT NULL COMMENT 'kubeconfig(加密)',
  `version` varchar(50) COMMENT 'K8S版本',
  `status` int DEFAULT 1 COMMENT '状态 1:正常 2:连接失败 3:不可用',
  `region` varchar(100) COMMENT '地域',
  `provider` varchar(50) COMMENT '云厂商',
  `description` varchar(500) COMMENT '描述',
  `created_by` bigint unsigned COMMENT '创建者ID',
  `node_count` int DEFAULT 0 COMMENT '节点数',
  `pod_count` int DEFAULT 0 COMMENT 'Pod数',
  `status_synced_at` datetime COMMENT '状态同步时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`),
  KEY `idx_status` (`status`),
  KEY `idx_provider` (`provider`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户kubeconfig表
CREATE TABLE IF NOT EXISTS `k8s_user_kube_configs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` bigint unsigned NOT NULL COMMENT '集群ID',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `service_account` varchar(255) NOT NULL COMMENT 'ServiceAccount名称',
  `namespace` varchar(255) DEFAULT 'default' COMMENT '命名空间',
  `is_active` tinyint DEFAULT 1 COMMENT '是否激活',
  `created_by` bigint unsigned NOT NULL COMMENT '创建者ID',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `revoked_at` datetime COMMENT '撤销时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_cluster_user_sa` (`cluster_id`, `user_id`, `service_account`),
  KEY `idx_cluster_id` (`cluster_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户K8S角色绑定表
CREATE TABLE IF NOT EXISTS `k8s_user_role_bindings` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` bigint unsigned NOT NULL COMMENT '集群ID',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `role_name` varchar(255) NOT NULL COMMENT '角色名称',
  `role_namespace` varchar(255) DEFAULT '' COMMENT '命名空间(空=ClusterRole)',
  `role_type` varchar(50) NOT NULL COMMENT '角色类型 ClusterRole/Role',
  `bound_by` bigint unsigned NOT NULL COMMENT '绑定者ID',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_cluster_user_role` (`cluster_id`, `user_id`, `role_name`, `role_namespace`),
  KEY `idx_cluster_id` (`cluster_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 集群巡检记录表
CREATE TABLE IF NOT EXISTS `k8s_cluster_inspections` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` bigint unsigned NOT NULL COMMENT '集群ID',
  `cluster_name` varchar(100) COMMENT '集群名称',
  `status` varchar(20) COMMENT '状态 running/completed/failed',
  `score` int COMMENT '健康评分',
  `check_count` int COMMENT '检查项总数',
  `pass_count` int COMMENT '通过项数',
  `warning_count` int COMMENT '警告项数',
  `fail_count` int COMMENT '失败项数',
  `duration` int COMMENT '耗时(秒)',
  `report_data` longtext COMMENT '巡检报告',
  `user_id` bigint unsigned COMMENT '执行者ID',
  `start_time` datetime COMMENT '开始时间',
  `end_time` datetime COMMENT '结束时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_cluster_id` (`cluster_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 终端会话记录表
CREATE TABLE IF NOT EXISTS `k8s_terminal_sessions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` bigint unsigned NOT NULL COMMENT '集群ID',
  `cluster_name` varchar(100) COMMENT '集群名称',
  `namespace` varchar(100) NOT NULL COMMENT '命名空间',
  `pod_name` varchar(200) NOT NULL COMMENT 'Pod名称',
  `container_name` varchar(100) NOT NULL COMMENT '容器名称',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `username` varchar(100) COMMENT '用户名',
  `recording_path` varchar(500) NOT NULL COMMENT '录制文件路径',
  `duration` int COMMENT '会话时长(秒)',
  `file_size` bigint COMMENT '文件大小(字节)',
  `status` varchar(20) DEFAULT 'completed' COMMENT '状态',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_cluster_id` (`cluster_id`),
  KEY `idx_namespace` (`namespace`),
  KEY `idx_pod_name` (`pod_name`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- 6. 监控插件表
-- ============================================================

-- 域名监控表
CREATE TABLE IF NOT EXISTS `domain_monitors` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `domain` varchar(255) NOT NULL COMMENT '监控域名',
  `status` varchar(20) DEFAULT 'unknown' COMMENT '状态',
  `response_time` int DEFAULT 0 COMMENT '响应时间(ms)',
  `ssl_valid` tinyint DEFAULT 0 COMMENT 'SSL是否有效',
  `ssl_expiry` datetime COMMENT 'SSL过期时间',
  `check_interval` int DEFAULT 300 COMMENT '检查间隔(秒)',
  `enable_ssl` tinyint DEFAULT 1 COMMENT '是否启用SSL检查',
  `enable_alert` tinyint DEFAULT 0 COMMENT '是否启用告警',
  `last_check` datetime COMMENT '最后检查时间',
  `next_check` datetime COMMENT '下次检查时间',
  `alert_config_id` bigint unsigned COMMENT '告警配置ID',
  `response_threshold` int DEFAULT 1000 COMMENT '响应时间阈值(ms)',
  `ssl_expiry_days` int DEFAULT 30 COMMENT '证书过期天数阈值',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_domain` (`domain`),
  KEY `idx_status` (`status`),
  KEY `idx_next_check` (`next_check`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 告警配置表
CREATE TABLE IF NOT EXISTS `alert_configs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '告警名称',
  `alert_type` varchar(20) NOT NULL COMMENT '告警类型',
  `enabled` tinyint DEFAULT 1 COMMENT '是否启用',
  `threshold` int COMMENT '阈值',
  `domain_monitor_id` bigint unsigned COMMENT '域名监控ID',
  `enable_email` tinyint DEFAULT 0 COMMENT '邮件告警',
  `enable_webhook` tinyint DEFAULT 0 COMMENT 'Webhook告警',
  `enable_wechat` tinyint DEFAULT 0 COMMENT '企业微信告警',
  `enable_dingtalk` tinyint DEFAULT 0 COMMENT '钉钉告警',
  `enable_feishu` tinyint DEFAULT 0 COMMENT '飞书告警',
  `enable_system_msg` tinyint DEFAULT 0 COMMENT '系统消息告警',
  `alert_interval` int DEFAULT 600 COMMENT '告警间隔(秒)',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_alert_type` (`alert_type`),
  KEY `idx_domain_monitor_id` (`domain_monitor_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 告警渠道表
CREATE TABLE IF NOT EXISTS `alert_channels` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '渠道名称',
  `channel_type` varchar(20) NOT NULL COMMENT '渠道类型',
  `enabled` tinyint DEFAULT 1 COMMENT '是否启用',
  `config` text COMMENT '渠道配置(JSON)',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_channel_type` (`channel_type`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 告警接收人表
CREATE TABLE IF NOT EXISTS `alert_receivers` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '接收人名称',
  `email` varchar(100) COMMENT '邮箱',
  `phone` varchar(20) COMMENT '电话',
  `wechat_id` varchar(100) COMMENT '企业微信ID',
  `dingtalk_id` varchar(100) COMMENT '钉钉ID',
  `feishu_id` varchar(100) COMMENT '飞书ID',
  `user_id` bigint unsigned COMMENT '关联用户ID',
  `enable_email` tinyint DEFAULT 1 COMMENT '启用邮件',
  `enable_webhook` tinyint DEFAULT 0 COMMENT '启用webhook',
  `enable_wechat` tinyint DEFAULT 0 COMMENT '启用企业微信',
  `enable_dingtalk` tinyint DEFAULT 0 COMMENT '启用钉钉',
  `enable_feishu` tinyint DEFAULT 0 COMMENT '启用飞书',
  `enable_system_msg` tinyint DEFAULT 1 COMMENT '启用系统消息',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 告警接收人-渠道关联表
CREATE TABLE IF NOT EXISTS `alert_receiver_channels` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `receiver_id` bigint unsigned NOT NULL COMMENT '接收人ID',
  `channel_id` bigint unsigned NOT NULL COMMENT '渠道ID',
  `config` text COMMENT '渠道特定配置',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_receiver_channel` (`receiver_id`, `channel_id`),
  KEY `idx_receiver_id` (`receiver_id`),
  KEY `idx_channel_id` (`channel_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 告警日志表
CREATE TABLE IF NOT EXISTS `alert_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `alert_type` varchar(50) NOT NULL COMMENT '告警类型',
  `domain_monitor_id` bigint unsigned NOT NULL COMMENT '监控ID',
  `domain` varchar(255) NOT NULL COMMENT '域名',
  `status` varchar(20) NOT NULL COMMENT '发送状态',
  `message` text COMMENT '告警消息',
  `channel_type` varchar(20) COMMENT '渠道类型',
  `error_msg` text COMMENT '错误信息',
  `sent_at` datetime COMMENT '发送时间',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_alert_type` (`alert_type`),
  KEY `idx_domain_monitor_id` (`domain_monitor_id`),
  KEY `idx_sent_at` (`sent_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- 12. SSL证书管理插件
-- ============================================================

-- SSL证书表
CREATE TABLE IF NOT EXISTS `ssl_certificates` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL,
  `name` varchar(100) NOT NULL COMMENT '证书名称',
  `domain` varchar(255) NOT NULL COMMENT '主域名',
  `san_domains` text COMMENT 'SAN域名(JSON数组)',
  `acme_email` varchar(255) COMMENT 'ACME注册邮箱',
  `ca_provider` varchar(20) COMMENT 'CA提供商: letsencrypt/zerossl/google/buypass',
  `key_algorithm` varchar(20) COMMENT '密钥算法: rsa2048/rsa3072/rsa4096/ec256/ec384',
  `source_type` varchar(20) COMMENT '证书来源: acme/aliyun/manual',
  `cloud_account_id` bigint unsigned DEFAULT NULL COMMENT '云账号ID',
  `cloud_cert_id` varchar(100) COMMENT '云厂商证书ID',
  `certificate` text COMMENT '证书PEM',
  `private_key` text COMMENT '私钥PEM(加密)',
  `cert_chain` text COMMENT '证书链',
  `issuer` varchar(255) COMMENT '签发机构',
  `not_before` datetime COMMENT '生效时间',
  `not_after` datetime COMMENT '过期时间',
  `fingerprint` varchar(100) COMMENT '指纹',
  `status` varchar(20) DEFAULT 'pending' COMMENT '状态: pending/active/expiring/expired/error',
  `auto_renew` tinyint(1) DEFAULT 1 COMMENT '自动续期',
  `renew_days_before` int DEFAULT 30 COMMENT '提前续期天数',
  `dns_provider_id` bigint unsigned DEFAULT NULL COMMENT 'DNS服务商ID',
  `last_renew_at` datetime COMMENT '最后续期时间',
  `last_error` text COMMENT '最后错误信息',
  PRIMARY KEY (`id`),
  KEY `idx_ssl_certificates_deleted_at` (`deleted_at`),
  KEY `idx_ssl_certificates_domain` (`domain`),
  KEY `idx_ssl_certificates_not_after` (`not_after`),
  KEY `idx_ssl_certificates_cloud_account_id` (`cloud_account_id`),
  KEY `idx_ssl_certificates_dns_provider_id` (`dns_provider_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- SSL DNS服务商配置表
CREATE TABLE IF NOT EXISTS `ssl_dns_providers` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL,
  `name` varchar(100) NOT NULL COMMENT '名称',
  `provider` varchar(50) NOT NULL COMMENT 'DNS服务商类型: aliyun/cloudflare/huawei/aws_route53',
  `config` text NOT NULL COMMENT '配置JSON(加密)',
  `email` varchar(255) COMMENT '联系邮箱',
  `phone` varchar(50) COMMENT '联系电话',
  `enabled` tinyint(1) DEFAULT 1 COMMENT '是否启用',
  `last_test_at` datetime COMMENT '最后测试时间',
  `last_test_ok` tinyint(1) DEFAULT 0 COMMENT '最后测试结果',
  PRIMARY KEY (`id`),
  KEY `idx_ssl_dns_providers_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- SSL部署配置表
CREATE TABLE IF NOT EXISTS `ssl_deploy_configs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL,
  `certificate_id` bigint unsigned NOT NULL COMMENT '关联证书ID',
  `name` varchar(100) NOT NULL COMMENT '配置名称',
  `deploy_type` varchar(20) NOT NULL COMMENT '部署类型: nginx_ssh/k8s_secret',
  `target_config` text NOT NULL COMMENT '目标配置JSON',
  `auto_deploy` tinyint(1) DEFAULT 1 COMMENT '续期后自动部署',
  `enabled` tinyint(1) DEFAULT 1 COMMENT '是否启用',
  `last_deploy_at` datetime COMMENT '最后部署时间',
  `last_deploy_ok` tinyint(1) DEFAULT 0 COMMENT '最后部署结果',
  `last_error` text COMMENT '最后错误信息',
  PRIMARY KEY (`id`),
  KEY `idx_ssl_deploy_configs_deleted_at` (`deleted_at`),
  KEY `idx_ssl_deploy_configs_certificate_id` (`certificate_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- SSL续期任务表
CREATE TABLE IF NOT EXISTS `ssl_renew_tasks` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL,
  `certificate_id` bigint unsigned NOT NULL COMMENT '关联证书ID',
  `task_type` varchar(20) NOT NULL COMMENT '任务类型: issue/renew/deploy',
  `status` varchar(20) DEFAULT 'pending' COMMENT '状态: pending/running/success/failed',
  `trigger_type` varchar(20) NOT NULL COMMENT '触发类型: auto/manual',
  `started_at` datetime COMMENT '开始时间',
  `finished_at` datetime COMMENT '完成时间',
  `error_message` text COMMENT '错误信息',
  `result` text COMMENT '结果JSON',
  PRIMARY KEY (`id`),
  KEY `idx_ssl_renew_tasks_deleted_at` (`deleted_at`),
  KEY `idx_ssl_renew_tasks_certificate_id` (`certificate_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- 7. Nginx 日志分析插件表
-- ============================================================

-- Nginx 数据源配置表
CREATE TABLE IF NOT EXISTS `nginx_sources` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '数据源名称',
  `type` varchar(20) NOT NULL COMMENT '数据源类型 host/k8s_ingress',
  `description` varchar(500) COMMENT '描述',
  `status` tinyint DEFAULT 1 COMMENT '状态 1:启用 0:禁用',
  `host_id` bigint unsigned COMMENT '主机ID(host类型)',
  `log_path` varchar(500) COMMENT '日志路径(host类型)',
  `log_format` varchar(50) DEFAULT 'combined' COMMENT '日志格式',
  `cluster_id` bigint unsigned COMMENT 'K8s集群ID(k8s_ingress类型)',
  `namespace` varchar(100) COMMENT 'K8s命名空间',
  `ingress_name` varchar(100) COMMENT 'Ingress名称',
  `k8s_pod_selector` varchar(200) COMMENT 'Pod标签选择器',
  `k8s_container_name` varchar(100) COMMENT '容器名称',
  `log_format_config` text COMMENT '自定义日志格式配置',
  `geo_enabled` tinyint DEFAULT 1 COMMENT '是否启用地理位置解析',
  `session_enabled` tinyint DEFAULT 0 COMMENT '是否启用会话跟踪',
  `collect_interval` int DEFAULT 60 COMMENT '采集间隔(秒)',
  `retention_days` int DEFAULT 30 COMMENT '数据保留天数',
  `last_collect_at` datetime COMMENT '最后采集时间',
  `last_collect_logs` bigint DEFAULT 0 COMMENT '最后采集日志数',
  `last_error` varchar(500) COMMENT '最后错误信息',
  `last_file_size` bigint DEFAULT 0 COMMENT '上次文件大小',
  `last_file_offset` bigint DEFAULT 0 COMMENT '上次读取偏移量',
  `last_file_inode` bigint unsigned DEFAULT 0 COMMENT '文件inode',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_host_id` (`host_id`),
  KEY `idx_cluster_id` (`cluster_id`),
  KEY `idx_status` (`status`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx IP 维度表
CREATE TABLE IF NOT EXISTS `nginx_dim_ip` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `ip_address` varchar(50) NOT NULL COMMENT 'IP地址',
  `country` varchar(50) COMMENT '国家',
  `province` varchar(50) COMMENT '省份',
  `city` varchar(50) COMMENT '城市',
  `isp` varchar(100) COMMENT '运营商',
  `is_bot` tinyint DEFAULT 0 COMMENT '是否机器人',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ip_address` (`ip_address`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx URL 维度表
CREATE TABLE IF NOT EXISTS `nginx_dim_url` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `url_hash` varchar(64) NOT NULL COMMENT 'URL哈希',
  `url_path` varchar(2000) COMMENT 'URL路径',
  `url_normalized` varchar(500) COMMENT '规范化路径',
  `host` varchar(255) COMMENT '主机名',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_url_hash` (`url_hash`),
  KEY `idx_url_normalized` (`url_normalized`),
  KEY `idx_host` (`host`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx Referer 维度表
CREATE TABLE IF NOT EXISTS `nginx_dim_referer` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `referer_hash` varchar(64) NOT NULL COMMENT 'Referer哈希',
  `referer_url` varchar(2000) COMMENT 'Referer URL',
  `referer_domain` varchar(255) COMMENT 'Referer域名',
  `referer_type` varchar(20) COMMENT '来源类型 direct/search/social/other',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_referer_hash` (`referer_hash`),
  KEY `idx_referer_domain` (`referer_domain`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx User-Agent 维度表
CREATE TABLE IF NOT EXISTS `nginx_dim_user_agent` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `ua_hash` varchar(64) NOT NULL COMMENT 'UA哈希',
  `user_agent` varchar(500) COMMENT 'User-Agent',
  `browser` varchar(50) COMMENT '浏览器',
  `browser_version` varchar(20) COMMENT '浏览器版本',
  `os` varchar(50) COMMENT '操作系统',
  `os_version` varchar(20) COMMENT '系统版本',
  `device_type` varchar(20) COMMENT '设备类型 desktop/mobile/tablet/bot',
  `is_bot` tinyint DEFAULT 0 COMMENT '是否机器人',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ua_hash` (`ua_hash`),
  KEY `idx_browser` (`browser`),
  KEY `idx_os` (`os`),
  KEY `idx_device_type` (`device_type`),
  KEY `idx_is_bot` (`is_bot`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx 访问日志事实表 (星型模型)
CREATE TABLE IF NOT EXISTS `nginx_fact_access_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_id` bigint unsigned NOT NULL COMMENT '数据源ID',
  `timestamp` datetime NOT NULL COMMENT '访问时间',
  `ip_id` bigint unsigned COMMENT 'IP维度ID',
  `url_id` bigint unsigned COMMENT 'URL维度ID',
  `referer_id` bigint unsigned COMMENT 'Referer维度ID',
  `ua_id` bigint unsigned COMMENT 'UA维度ID',
  `method` varchar(20) COMMENT '请求方法',
  `protocol` varchar(50) COMMENT '协议',
  `status` int COMMENT '状态码',
  `body_bytes_sent` bigint COMMENT '响应大小',
  `request_time` decimal(10,3) COMMENT '请求耗时',
  `upstream_time` decimal(10,3) COMMENT '上游耗时',
  `ingress_name` varchar(100) COMMENT 'Ingress名称',
  `service_name` varchar(100) COMMENT '服务名称',
  `pod_name` varchar(100) COMMENT 'Pod名称',
  `is_pv` tinyint DEFAULT 1 COMMENT '是否页面访问',
  `session_id` varchar(64) COMMENT '会话ID',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_source_time` (`source_id`, `timestamp`),
  KEY `idx_ip_id` (`ip_id`),
  KEY `idx_url_id` (`url_id`),
  KEY `idx_referer_id` (`referer_id`),
  KEY `idx_ua_id` (`ua_id`),
  KEY `idx_method` (`method`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx 访问日志表 (兼容旧版，扁平化存储)
CREATE TABLE IF NOT EXISTS `nginx_access_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_id` bigint unsigned NOT NULL COMMENT '数据源ID',
  `timestamp` datetime NOT NULL COMMENT '访问时间',
  `remote_addr` varchar(50) COMMENT '客户端IP',
  `remote_user` varchar(100) COMMENT '远程用户',
  `request` varchar(2000) COMMENT '请求行',
  `method` varchar(20) COMMENT '请求方法',
  `uri` varchar(1000) COMMENT '请求URI',
  `protocol` varchar(50) COMMENT '协议',
  `status` int COMMENT '状态码',
  `body_bytes_sent` bigint COMMENT '响应大小',
  `http_referer` varchar(1000) COMMENT 'Referer',
  `http_user_agent` varchar(500) COMMENT 'User-Agent',
  `request_time` decimal(10,3) COMMENT '请求耗时',
  `upstream_time` decimal(10,3) COMMENT '上游耗时',
  `host` varchar(255) COMMENT '主机名',
  `country` varchar(50) COMMENT '国家',
  `province` varchar(50) COMMENT '省份',
  `city` varchar(50) COMMENT '城市',
  `isp` varchar(100) COMMENT '运营商',
  `browser` varchar(50) COMMENT '浏览器',
  `browser_version` varchar(20) COMMENT '浏览器版本',
  `os` varchar(50) COMMENT '操作系统',
  `os_version` varchar(20) COMMENT '系统版本',
  `device_type` varchar(20) COMMENT '设备类型',
  `ingress_name` varchar(100) COMMENT 'Ingress名称',
  `service_name` varchar(100) COMMENT '服务名称',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_source_time` (`source_id`, `timestamp`),
  KEY `idx_source_ip` (`source_id`, `remote_addr`),
  KEY `idx_source_status` (`source_id`, `status`),
  KEY `idx_source_country` (`source_id`, `country`),
  KEY `idx_source_device` (`source_id`, `device_type`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx 小时聚合统计表
CREATE TABLE IF NOT EXISTS `nginx_agg_hourly` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_id` bigint unsigned NOT NULL COMMENT '数据源ID',
  `hour` datetime NOT NULL COMMENT '小时时间点',
  `total_requests` bigint DEFAULT 0 COMMENT '总请求数',
  `pv_count` bigint DEFAULT 0 COMMENT 'PV数',
  `unique_ips` bigint DEFAULT 0 COMMENT '独立IP数',
  `total_bandwidth` bigint DEFAULT 0 COMMENT '总带宽',
  `avg_response_time` decimal(10,3) DEFAULT 0 COMMENT '平均响应时间',
  `max_response_time` decimal(10,3) DEFAULT 0 COMMENT '最大响应时间',
  `min_response_time` decimal(10,3) DEFAULT 0 COMMENT '最小响应时间',
  `status_2xx` bigint DEFAULT 0 COMMENT '2xx状态码数',
  `status_3xx` bigint DEFAULT 0 COMMENT '3xx状态码数',
  `status_4xx` bigint DEFAULT 0 COMMENT '4xx状态码数',
  `status_5xx` bigint DEFAULT 0 COMMENT '5xx状态码数',
  `method_distribution` text COMMENT '方法分布JSON',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_source_hour` (`source_id`, `hour`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx 日聚合统计表
CREATE TABLE IF NOT EXISTS `nginx_agg_daily` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_id` bigint unsigned NOT NULL COMMENT '数据源ID',
  `date` date NOT NULL COMMENT '日期',
  `total_requests` bigint DEFAULT 0 COMMENT '总请求数',
  `pv_count` bigint DEFAULT 0 COMMENT 'PV数',
  `unique_ips` bigint DEFAULT 0 COMMENT '独立IP数',
  `total_bandwidth` bigint DEFAULT 0 COMMENT '总带宽',
  `avg_response_time` decimal(10,3) DEFAULT 0 COMMENT '平均响应时间',
  `max_response_time` decimal(10,3) DEFAULT 0 COMMENT '最大响应时间',
  `min_response_time` decimal(10,3) DEFAULT 0 COMMENT '最小响应时间',
  `status_2xx` bigint DEFAULT 0 COMMENT '2xx状态码数',
  `status_3xx` bigint DEFAULT 0 COMMENT '3xx状态码数',
  `status_4xx` bigint DEFAULT 0 COMMENT '4xx状态码数',
  `status_5xx` bigint DEFAULT 0 COMMENT '5xx状态码数',
  `top_urls` text COMMENT 'Top URL JSON',
  `top_ips` text COMMENT 'Top IP JSON',
  `top_referers` text COMMENT 'Top Referer JSON',
  `top_countries` text COMMENT 'Top 国家 JSON',
  `top_browsers` text COMMENT 'Top 浏览器 JSON',
  `top_devices` text COMMENT 'Top 设备 JSON',
  `hourly_traffic` text COMMENT '每小时流量分布 JSON',
  `method_distribution` text COMMENT '方法分布 JSON',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_source_date` (`source_id`, `date`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx 日统计表 (兼容旧版)
CREATE TABLE IF NOT EXISTS `nginx_daily_stats` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_id` bigint unsigned NOT NULL COMMENT '数据源ID',
  `date` date NOT NULL COMMENT '日期',
  `total_requests` bigint DEFAULT 0 COMMENT '总请求数',
  `unique_visitors` bigint DEFAULT 0 COMMENT '独立访客数',
  `total_bandwidth` bigint DEFAULT 0 COMMENT '总带宽',
  `avg_response_time` decimal(10,3) DEFAULT 0 COMMENT '平均响应时间',
  `status_2xx` bigint DEFAULT 0 COMMENT '2xx状态码数',
  `status_3xx` bigint DEFAULT 0 COMMENT '3xx状态码数',
  `status_4xx` bigint DEFAULT 0 COMMENT '4xx状态码数',
  `status_5xx` bigint DEFAULT 0 COMMENT '5xx状态码数',
  `top_ur_is` text COMMENT 'Top URI JSON',
  `top_i_ps` text COMMENT 'Top IP JSON',
  `top_referers` text COMMENT 'Top Referer JSON',
  `top_user_agents` text COMMENT 'Top UA JSON',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_source_id` (`source_id`),
  KEY `idx_date` (`date`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Nginx 小时统计表 (兼容旧版)
CREATE TABLE IF NOT EXISTS `nginx_hourly_stats` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `source_id` bigint unsigned NOT NULL COMMENT '数据源ID',
  `hour` datetime NOT NULL COMMENT '小时时间点',
  `total_requests` bigint DEFAULT 0 COMMENT '总请求数',
  `unique_visitors` bigint DEFAULT 0 COMMENT '独立访客数',
  `total_bandwidth` bigint DEFAULT 0 COMMENT '总带宽',
  `avg_response_time` decimal(10,3) DEFAULT 0 COMMENT '平均响应时间',
  `status_2xx` bigint DEFAULT 0 COMMENT '2xx状态码数',
  `status_3xx` bigint DEFAULT 0 COMMENT '3xx状态码数',
  `status_4xx` bigint DEFAULT 0 COMMENT '4xx状态码数',
  `status_5xx` bigint DEFAULT 0 COMMENT '5xx状态码数',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_source_id` (`source_id`),
  KEY `idx_hour` (`hour`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================
-- 初始化数据
-- ============================================================

-- 插入默认部门
INSERT INTO `sys_department` (`id`, `name`, `code`, `parent_id`, `dept_type`, `sort`, `status`, `created_at`, `updated_at`)
VALUES (1, '总公司', 'head', 0, 1, 0, 1, NOW(), NOW());

-- 插入默认角色
INSERT INTO `sys_role` (`id`, `name`, `code`, `description`, `sort`, `status`, `created_at`, `updated_at`)
VALUES
  (1, '管理员', 'admin', '系统管理员，拥有所有权限', 0, 1, NOW(), NOW()),
  (2, '普通用户', 'user', '普通用户，具有基本操作权限', 1, 1, NOW(), NOW());

-- 插入默认菜单（从当前数据库导出的完整菜单结构）
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  -- ========== 顶级菜单 ==========
  (10, '仪表盘', 'dashboard', 1, 0, '/dashboard', '', 'HomeFilled', 0, 1, 1, '', '', NOW(), NOW()),
  (15, '资产管理', 'asset-management', 1, 0, '/asset', '', 'Coin', 1, 1, 1, '', '', NOW(), NOW()),
  (23, '操作审计', 'audit', 1, 0, '/audit', '', 'Document', 50, 1, 1, '', '', NOW(), NOW()),
  (30, '插件管理', 'plugin', 1, 0, '/plugin', '', 'Grid', 80, 1, 1, '', '', NOW(), NOW()),
  (42, '监控中心', '_monitor', 1, 0, '/monitor', '', 'Monitor', 80, 1, 1, '', '', NOW(), NOW()),
  (61, '任务中心', '_task', 1, 0, '/task', '', 'Grid', 90, 1, 1, '', '', NOW(), NOW()),
  (1, '系统管理', 'system', 1, 0, '', '', 'Setting', 100, 1, 1, '', '', NOW(), NOW()),
  (29, '个人信息', 'profile', 2, 0, '/profile', 'Profile', 'UserFilled', 100, 0, 1, '', '', NOW(), NOW()),
  (36, '容器管理', '_kubernetes', 1, 0, '/kubernetes', '', 'Platform', 100, 1, 1, '', '', NOW(), NOW()),

  -- ========== 系统管理子菜单 (parent_id=1) ==========
  (2, '用户管理', 'users', 2, 1, '/users', 'system/Users', 'User', 1, 1, 1, '', '', NOW(), NOW()),
  (3, '角色管理', 'roles', 2, 1, '/roles', 'system/Roles', 'UserFilled', 2, 1, 1, '', '', NOW(), NOW()),
  (5, '菜单管理', 'menus', 2, 1, '/menus', 'system/Menus', 'Menu', 4, 1, 1, '', '', NOW(), NOW()),
  (11, '部门信息', 'dept-info', 2, 1, '/dept-info', 'system/DeptInfo', 'OfficeBuilding', 5, 1, 1, '', '', NOW(), NOW()),
  (12, '岗位信息', 'position-info', 2, 1, '/position-info', 'system/PositionInfo', 'Avatar', 6, 1, 1, '', '', NOW(), NOW()),
  (13, '系统配置', 'system-config', 2, 1, '/system-config', 'system/SystemConfig', 'Setting', 7, 1, 1, '', '', NOW(), NOW()),

  -- ========== 资产管理子菜单 (parent_id=15) ==========
  (16, '主机管理', 'host-management', 2, 15, '/asset/hosts', 'asset/Hosts', 'Monitor', 1, 1, 1, '', '', NOW(), NOW()),
  (19, '凭据管理', 'asset:credentials', 2, 15, '/asset/credentials', 'asset/Credentials', 'Lock', 2, 1, 1, '', '', NOW(), NOW()),
  (17, '业务分组', 'business-group', 2, 15, '/asset/groups', 'asset/Groups', 'Collection', 3, 1, 1, '', '', NOW(), NOW()),
  (27, '云账号管理', 'cloud-accounts', 2, 15, '/asset/cloud-accounts', 'asset/CloudAccounts', 'Cloudy', 5, 1, 1, '', '', NOW(), NOW()),
  (34, '终端审计', 'asset_terminal_audit', 2, 15, '/asset/terminal-audit', '', 'View', 5, 1, 1, '', '', NOW(), NOW()),
  (65, '权限配置', 'asset_permission', 2, 15, '/asset/permissions', 'views/asset/AssetPermission.vue', 'Lock', 6, 1, 1, '', '', NOW(), NOW()),

  -- ========== 操作审计子菜单 (parent_id=23) ==========
  (24, '操作日志', 'operation-logs', 2, 23, '/audit/operation-logs', 'audit/OperationLogs', 'Document', 1, 1, 1, '', '', NOW(), NOW()),
  (25, '登录日志', 'login-logs', 2, 23, '/audit/login-logs', 'audit/LoginLogs', 'CircleCheck', 2, 1, 1, '', '', NOW(), NOW()),
  (26, '数据日志', 'data-logs', 2, 23, '/audit/data-logs', 'audit/DataLogs', 'Notebook', 3, 1, 1, '', '', NOW(), NOW()),

  -- ========== 插件管理子菜单 (parent_id=30) ==========
  (32, '插件列表', 'plugin-list', 2, 30, '/plugin/list', 'plugin/PluginList', 'Grid', 1, 1, 1, '', '', NOW(), NOW()),
  (33, '插件安装', 'plugin-install', 2, 30, '/plugin/install', 'plugin/PluginInstall', 'Upload', 2, 1, 1, '', '', NOW(), NOW()),

  -- ========== 容器管理子菜单 (parent_id=36) ==========
  (69, '集群管理', 'kubernetes_clusters', 2, 36, '/kubernetes/clusters', '', 'Connection', 1, 1, 1, '', '', NOW(), NOW()),
  (70, '节点管理', 'kubernetes_nodes', 2, 36, '/kubernetes/nodes', '', 'Monitor', 2, 1, 1, '', '', NOW(), NOW()),
  (71, '命名空间', 'kubernetes_namespaces', 2, 36, '/kubernetes/namespaces', '', 'FolderOpened', 3, 1, 1, '', '', NOW(), NOW()),
  (72, '工作负载', 'kubernetes_workloads', 2, 36, '/kubernetes/workloads', '', 'Grid', 4, 1, 1, '', '', NOW(), NOW()),
  (73, '网络管理', 'kubernetes_network', 2, 36, '/kubernetes/network', '', 'Connection', 5, 1, 1, '', '', NOW(), NOW()),
  (74, '配置管理', 'kubernetes_config', 2, 36, '/kubernetes/config', '', 'Tools', 6, 1, 1, '', '', NOW(), NOW()),
  (75, '存储管理', 'kubernetes_storage', 2, 36, '/kubernetes/storage', '', 'Files', 7, 1, 1, '', '', NOW(), NOW()),
  (76, '访问控制', 'kubernetes_access', 2, 36, '/kubernetes/access', '', 'Lock', 8, 1, 1, '', '', NOW(), NOW()),
  (77, '终端审计', 'kubernetes_audit', 2, 36, '/kubernetes/audit', '', 'Monitor', 9, 1, 1, '', '', NOW(), NOW()),
  (85, '应用诊断', 'kubernetes_application_diagnosis', 2, 36, '/kubernetes/application-diagnosis', '', 'Grid', 10, 1, 1, '', '', NOW(), NOW()),
  (86, '集群巡检', 'kubernetes_cluster_inspection', 2, 36, '/kubernetes/cluster-inspection', '', 'Grid', 11, 1, 1, '', '', NOW(), NOW()),

  -- ========== 监控中心子菜单 (parent_id=42) ==========
  (78, '域名监控', 'monitor_domain', 2, 42, '/monitor/domain', '', 'Monitor', 1, 1, 1, '', '', NOW(), NOW()),
  (79, '告警通道', 'monitor_alert_channels', 2, 42, '/monitor/alert-channels', '', 'Grid', 2, 1, 1, '', '', NOW(), NOW()),
  (80, '告警接收人', 'monitor_alert_receivers', 2, 42, '/monitor/alert-receivers', '', 'User', 3, 1, 1, '', '', NOW(), NOW()),
  (81, '告警日志', 'monitor_alert_logs', 2, 42, '/monitor/alert-logs', '', 'Document', 4, 1, 1, '', '', NOW(), NOW()),

  -- ========== 任务中心子菜单 (parent_id=61) ==========
  (82, '任务模板', 'task_templates', 2, 61, '/task/templates', '', 'Document', 1, 1, 1, '', '', NOW(), NOW()),
  (83, '执行任务', 'task_execute', 2, 61, '/task/execute', '', 'Tools', 2, 1, 1, '', '', NOW(), NOW()),
  (84, '文件分发', 'task_file_distribution', 2, 61, '/task/file-distribution', '', 'Files', 3, 1, 1, '', '', NOW(), NOW()),
  (87, '执行历史', 'task_execution_history', 2, 61, '/task/execution-history', '', 'Timer', 4, 1, 1, '', '', NOW(), NOW());

-- 为管理员角色分配所有菜单权限
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (1, 1), (1, 2), (1, 3), (1, 5), (1, 10), (1, 11), (1, 12), (1, 13), (1, 15), (1, 16), (1, 17), (1, 19),
  (1, 23), (1, 24), (1, 25), (1, 26), (1, 27), (1, 29), (1, 30), (1, 32), (1, 33), (1, 34), (1, 36),
  (1, 42), (1, 61), (1, 65), (1, 69), (1, 70), (1, 71), (1, 72), (1, 73), (1, 74), (1, 75), (1, 76), (1, 77),
  (1, 78), (1, 79), (1, 80), (1, 81), (1, 82), (1, 83), (1, 84), (1, 85), (1, 86), (1, 87);

-- 为普通用户角色分配基础菜单权限
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (2, 10), (2, 15), (2, 16), (2, 17), (2, 19), (2, 27), (2, 34), (2, 65),
  (2, 23), (2, 24), (2, 25), (2, 36), (2, 42), (2, 61);

-- ============================================================
-- 按钮权限数据 (type=3) — 前端 v-permission 对应的按钮级权限
-- ============================================================

-- 插入按钮权限菜单
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (200, '新增用户', 'users:create', 3, 2, '', '', '', 1, 1, 1, '/api/v1/users', 'POST', NOW(), NOW()),
  (201, '编辑用户', 'users:update', 3, 2, '', '', '', 2, 1, 1, '/api/v1/users/:id', 'PUT', NOW(), NOW()),
  (202, '删除用户', 'users:delete', 3, 2, '', '', '', 3, 1, 1, '/api/v1/users/:id', 'DELETE', NOW(), NOW()),
  (203, '解锁用户', 'users:unlock', 3, 2, '', '', '', 4, 1, 1, '/api/v1/users/:id/unlock', 'PUT', NOW(), NOW()),
  (204, '重置密码', 'users:reset-pwd', 3, 2, '', '', '', 5, 1, 1, '/api/v1/users/:id/reset-password', 'PUT', NOW(), NOW()),
  (205, '新增角色', 'roles:create', 3, 3, '', '', '', 6, 1, 1, '/api/v1/roles', 'POST', NOW(), NOW()),
  (206, '编辑角色', 'roles:update', 3, 3, '', '', '', 7, 1, 1, '/api/v1/roles/:id', 'PUT', NOW(), NOW()),
  (207, '删除角色', 'roles:delete', 3, 3, '', '', '', 8, 1, 1, '/api/v1/roles/:id', 'DELETE', NOW(), NOW()),
  (208, '分配菜单权限', 'roles:assign-menus', 3, 3, '', '', '', 9, 1, 1, '/api/v1/roles/:id/menus', 'PUT', NOW(), NOW()),
  (209, '新增菜单', 'menus:create', 3, 5, '', '', '', 10, 1, 1, '/api/v1/menus', 'POST', NOW(), NOW()),
  (210, '编辑菜单', 'menus:update', 3, 5, '', '', '', 11, 1, 1, '/api/v1/menus/:id', 'PUT', NOW(), NOW()),
  (211, '删除菜单', 'menus:delete', 3, 5, '', '', '', 12, 1, 1, '/api/v1/menus/:id', 'DELETE', NOW(), NOW()),
  (212, '新增部门', 'depts:create', 3, 11, '', '', '', 13, 1, 1, '/api/v1/departments', 'POST', NOW(), NOW()),
  (213, '编辑部门', 'depts:update', 3, 11, '', '', '', 14, 1, 1, '/api/v1/departments/:id', 'PUT', NOW(), NOW()),
  (214, '删除部门', 'depts:delete', 3, 11, '', '', '', 15, 1, 1, '/api/v1/departments/:id', 'DELETE', NOW(), NOW()),
  (215, '新增岗位', 'positions:create', 3, 12, '', '', '', 16, 1, 1, '/api/v1/positions', 'POST', NOW(), NOW()),
  (216, '编辑岗位', 'positions:update', 3, 12, '', '', '', 17, 1, 1, '/api/v1/positions/:id', 'PUT', NOW(), NOW()),
  (217, '删除岗位', 'positions:delete', 3, 12, '', '', '', 18, 1, 1, '/api/v1/positions/:id', 'DELETE', NOW(), NOW()),
  (218, '分配岗位用户', 'positions:assign-users', 3, 12, '', '', '', 19, 1, 1, '/api/v1/positions/:id/users', 'POST', NOW(), NOW()),
  (219, '保存配置', 'system-config:update', 3, 13, '', '', '', 20, 1, 1, '/api/v1/system/config/basic', 'PUT', NOW(), NOW()),
  (220, '新增主机', 'hosts:import', 3, 16, '', '', '', 21, 1, 1, '/api/v1/hosts', 'POST', NOW(), NOW()),
  (221, '编辑主机', 'hosts:update', 3, 16, '', '', '', 22, 1, 1, '/api/v1/hosts/:id', 'PUT', NOW(), NOW()),
  (222, '删除主机', 'hosts:delete', 3, 16, '', '', '', 23, 1, 1, '/api/v1/hosts/:id', 'DELETE', NOW(), NOW()),
  (223, '批量删除', 'hosts:batch-delete', 3, 16, '', '', '', 24, 1, 1, '/api/v1/hosts/batch-delete', 'POST', NOW(), NOW()),
  (224, '采集信息', 'hosts:collect', 3, 16, '', '', '', 25, 1, 1, '/api/v1/hosts/:id/collect', 'POST', NOW(), NOW()),
  (225, '文件管理', 'hosts:file-manage', 3, 16, '', '', '', 26, 1, 1, '/api/v1/hosts/:id/files', 'GET', NOW(), NOW()),
  (226, '新增分组', 'asset-groups:create', 3, 17, '', '', '', 27, 1, 1, '/api/v1/asset-groups', 'POST', NOW(), NOW()),
  (227, '编辑分组', 'asset-groups:update', 3, 17, '', '', '', 28, 1, 1, '/api/v1/asset-groups/:id', 'PUT', NOW(), NOW()),
  (228, '删除分组', 'asset-groups:delete', 3, 17, '', '', '', 29, 1, 1, '/api/v1/asset-groups/:id', 'DELETE', NOW(), NOW()),
  (229, '新增凭据', 'credentials:create', 3, 19, '', '', '', 30, 1, 1, '/api/v1/credentials', 'POST', NOW(), NOW()),
  (230, '编辑凭据', 'credentials:update', 3, 19, '', '', '', 31, 1, 1, '/api/v1/credentials/:id', 'PUT', NOW(), NOW()),
  (231, '删除凭据', 'credentials:delete', 3, 19, '', '', '', 32, 1, 1, '/api/v1/credentials/:id', 'DELETE', NOW(), NOW()),
  (232, '新增云账号', 'cloud-accounts:create', 3, 27, '', '', '', 33, 1, 1, '/api/v1/cloud-accounts', 'POST', NOW(), NOW()),
  (233, '导入主机', 'cloud-accounts:import', 3, 27, '', '', '', 34, 1, 1, '/api/v1/cloud-accounts/import', 'POST', NOW(), NOW()),
  (234, '编辑云账号', 'cloud-accounts:update', 3, 27, '', '', '', 35, 1, 1, '/api/v1/cloud-accounts/:id', 'PUT', NOW(), NOW()),
  (235, '删除云账号', 'cloud-accounts:delete', 3, 27, '', '', '', 36, 1, 1, '/api/v1/cloud-accounts/:id', 'DELETE', NOW(), NOW()),
  (236, '删除会话', 'terminal-sessions:delete', 3, 34, '', '', '', 37, 1, 1, '/api/v1/terminal-sessions/:id', 'DELETE', NOW(), NOW()),
  (237, '添加权限', 'asset-perms:create', 3, 65, '', '', '', 38, 1, 1, '/api/v1/asset-permissions', 'POST', NOW(), NOW()),
  (238, '编辑权限', 'asset-perms:update', 3, 65, '', '', '', 39, 1, 1, '/api/v1/asset-permissions/:id', 'PUT', NOW(), NOW()),
  (239, '删除权限', 'asset-perms:delete', 3, 65, '', '', '', 40, 1, 1, '/api/v1/asset-permissions/:id', 'DELETE', NOW(), NOW()),
  (240, '批量删除', 'op-logs:batch-delete', 3, 24, '', '', '', 41, 1, 1, '/api/v1/audit/operation-logs/batch-delete', 'POST', NOW(), NOW()),
  (241, '删除日志', 'op-logs:delete', 3, 24, '', '', '', 42, 1, 1, '/api/v1/audit/operation-logs/:id', 'DELETE', NOW(), NOW()),
  (242, '批量删除', 'login-logs:batch-delete', 3, 25, '', '', '', 43, 1, 1, '/api/v1/audit/login-logs/batch-delete', 'POST', NOW(), NOW()),
  (243, '删除日志', 'login-logs:delete', 3, 25, '', '', '', 44, 1, 1, '/api/v1/audit/login-logs/:id', 'DELETE', NOW(), NOW()),
  (244, '启用插件', 'plugins:enable', 3, 32, '', '', '', 45, 1, 1, '/api/v1/plugins/:name/enable', 'POST', NOW(), NOW()),
  (245, '禁用插件', 'plugins:disable', 3, 32, '', '', '', 46, 1, 1, '/api/v1/plugins/:name/disable', 'POST', NOW(), NOW()),
  (246, '卸载插件', 'plugins:uninstall', 3, 32, '', '', '', 47, 1, 1, '/api/v1/plugins/:name/uninstall', 'DELETE', NOW(), NOW()),
  (247, '安装插件', 'plugins:install', 3, 33, '', '', '', 48, 1, 1, '/api/v1/plugins/upload', 'POST', NOW(), NOW()),
  (248, '注册集群', 'k8s-clusters:create', 3, 69, '', '', '', 49, 1, 1, '/api/v1/plugins/kubernetes/clusters', 'POST', NOW(), NOW()),
  (249, '编辑集群', 'k8s-clusters:update', 3, 69, '', '', '', 50, 1, 1, '/api/v1/plugins/kubernetes/clusters/:id', 'PUT', NOW(), NOW()),
  (250, '删除集群', 'k8s-clusters:delete', 3, 69, '', '', '', 51, 1, 1, '/api/v1/plugins/kubernetes/clusters/:id', 'DELETE', NOW(), NOW()),
  (251, '同步状态', 'k8s-clusters:sync', 3, 69, '', '', '', 52, 1, 1, '/api/v1/plugins/kubernetes/clusters/:id/sync', 'POST', NOW(), NOW()),
  (252, '批量同步', 'k8s-clusters:batch-sync', 3, 69, '', '', '', 53, 1, 1, '/api/v1/plugins/kubernetes/clusters/:id/sync', 'POST', NOW(), NOW()),
  (253, '批量删除', 'k8s-clusters:batch-delete', 3, 69, '', '', '', 54, 1, 1, '/api/v1/plugins/kubernetes/clusters/:id', 'DELETE', NOW(), NOW()),
  (254, '凭据申请', 'k8s-clusters:apply-credential', 3, 69, '', '', '', 55, 1, 1, '/api/v1/plugins/kubernetes/clusters/kubeconfig', 'POST', NOW(), NOW()),
  (255, '吊销凭据', 'k8s-clusters:revoke-credential', 3, 69, '', '', '', 56, 1, 1, '/api/v1/plugins/kubernetes/clusters/kubeconfig', 'DELETE', NOW(), NOW()),
  (256, '新建命名空间', 'k8s-namespaces:create', 3, 71, '', '', '', 57, 1, 1, '/api/v1/plugins/kubernetes/resources/namespaces', 'POST', NOW(), NOW()),
  (257, '创建工作负载', 'k8s-workloads:create', 3, 72, '', '', '', 58, 1, 1, '/api/v1/plugins/kubernetes/workloads/update', 'POST', NOW(), NOW()),
  (258, '批量重启', 'k8s-workloads:batch-restart', 3, 72, '', '', '', 59, 1, 1, '/api/v1/plugins/kubernetes/workloads/update', 'POST', NOW(), NOW()),
  (259, '批量停止', 'k8s-workloads:batch-stop', 3, 72, '', '', '', 60, 1, 1, '/api/v1/plugins/kubernetes/workloads/update', 'POST', NOW(), NOW()),
  (260, '批量恢复', 'k8s-workloads:batch-resume', 3, 72, '', '', '', 61, 1, 1, '/api/v1/plugins/kubernetes/workloads/update', 'POST', NOW(), NOW()),
  (261, '批量删除', 'k8s-workloads:batch-delete', 3, 72, '', '', '', 62, 1, 1, '/api/v1/plugins/kubernetes/workloads/delete', 'DELETE', NOW(), NOW()),
  (262, '创建服务', 'k8s-services:create', 3, 73, '', '', '', 63, 1, 1, '/api/v1/plugins/kubernetes/resources/services/:ns/:name', 'POST', NOW(), NOW()),
  (263, '编辑服务', 'k8s-services:update', 3, 73, '', '', '', 64, 1, 1, '/api/v1/plugins/kubernetes/resources/services/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (264, '删除服务', 'k8s-services:delete', 3, 73, '', '', '', 65, 1, 1, '/api/v1/plugins/kubernetes/resources/services/:ns/:name', 'DELETE', NOW(), NOW()),
  (265, '创建Ingress', 'k8s-ingresses:create', 3, 73, '', '', '', 66, 1, 1, '/api/v1/plugins/kubernetes/resources/ingresses/:ns/:name', 'POST', NOW(), NOW()),
  (266, '编辑Ingress', 'k8s-ingresses:update', 3, 73, '', '', '', 67, 1, 1, '/api/v1/plugins/kubernetes/resources/ingresses/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (267, '删除Ingress', 'k8s-ingresses:delete', 3, 73, '', '', '', 68, 1, 1, '/api/v1/plugins/kubernetes/resources/ingresses/:ns/:name', 'DELETE', NOW(), NOW()),
  (268, '创建Endpoints', 'k8s-endpoints:create', 3, 73, '', '', '', 69, 1, 1, '/api/v1/plugins/kubernetes/resources/endpoints/:ns/yaml', 'POST', NOW(), NOW()),
  (269, '编辑Endpoints', 'k8s-endpoints:update', 3, 73, '', '', '', 70, 1, 1, '/api/v1/plugins/kubernetes/resources/endpoints/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (270, '删除Endpoints', 'k8s-endpoints:delete', 3, 73, '', '', '', 71, 1, 1, '/api/v1/plugins/kubernetes/resources/endpoints/:ns/:name', 'DELETE', NOW(), NOW()),
  (271, '创建网络策略', 'k8s-networkpolicies:create', 3, 73, '', '', '', 72, 1, 1, '/api/v1/plugins/kubernetes/resources/networkpolicies/:ns/yaml', 'POST', NOW(), NOW()),
  (272, '编辑网络策略', 'k8s-networkpolicies:update', 3, 73, '', '', '', 73, 1, 1, '/api/v1/plugins/kubernetes/resources/networkpolicies/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (273, '删除网络策略', 'k8s-networkpolicies:delete', 3, 73, '', '', '', 74, 1, 1, '/api/v1/plugins/kubernetes/resources/networkpolicies/:ns/:name', 'DELETE', NOW(), NOW()),
  (274, '创建ConfigMap', 'k8s-configmaps:create', 3, 74, '', '', '', 75, 1, 1, '/api/v1/plugins/kubernetes/resources/configmaps/:ns/yaml', 'POST', NOW(), NOW()),
  (275, '编辑ConfigMap', 'k8s-configmaps:update', 3, 74, '', '', '', 76, 1, 1, '/api/v1/plugins/kubernetes/resources/configmaps/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (276, '删除ConfigMap', 'k8s-configmaps:delete', 3, 74, '', '', '', 77, 1, 1, '/api/v1/plugins/kubernetes/resources/configmaps/:ns/:name', 'DELETE', NOW(), NOW()),
  (277, '创建Secret', 'k8s-secrets:create', 3, 74, '', '', '', 78, 1, 1, '/api/v1/plugins/kubernetes/resources/secrets/:ns/yaml', 'POST', NOW(), NOW()),
  (278, '编辑Secret', 'k8s-secrets:update', 3, 74, '', '', '', 79, 1, 1, '/api/v1/plugins/kubernetes/resources/secrets/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (279, '删除Secret', 'k8s-secrets:delete', 3, 74, '', '', '', 80, 1, 1, '/api/v1/plugins/kubernetes/resources/secrets/:ns/:name', 'DELETE', NOW(), NOW()),
  (280, '创建ResourceQuota', 'k8s-resourcequotas:create', 3, 74, '', '', '', 81, 1, 1, '/api/v1/plugins/kubernetes/resources/resourcequotas/:ns/yaml', 'POST', NOW(), NOW()),
  (281, '编辑ResourceQuota', 'k8s-resourcequotas:update', 3, 74, '', '', '', 82, 1, 1, '/api/v1/plugins/kubernetes/resources/resourcequotas/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (282, '删除ResourceQuota', 'k8s-resourcequotas:delete', 3, 74, '', '', '', 83, 1, 1, '/api/v1/plugins/kubernetes/resources/resourcequotas/:ns/:name', 'DELETE', NOW(), NOW()),
  (283, '创建LimitRange', 'k8s-limitranges:create', 3, 74, '', '', '', 84, 1, 1, '/api/v1/plugins/kubernetes/resources/limitranges/:ns/yaml', 'POST', NOW(), NOW()),
  (284, '编辑LimitRange', 'k8s-limitranges:update', 3, 74, '', '', '', 85, 1, 1, '/api/v1/plugins/kubernetes/resources/limitranges/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (285, '删除LimitRange', 'k8s-limitranges:delete', 3, 74, '', '', '', 86, 1, 1, '/api/v1/plugins/kubernetes/resources/limitranges/:ns/:name', 'DELETE', NOW(), NOW()),
  (286, '创建HPA', 'k8s-hpa:create', 3, 74, '', '', '', 87, 1, 1, '/api/v1/plugins/kubernetes/resources/horizontalpodautoscalers/:ns/yaml', 'POST', NOW(), NOW()),
  (287, '编辑HPA', 'k8s-hpa:update', 3, 74, '', '', '', 88, 1, 1, '/api/v1/plugins/kubernetes/resources/horizontalpodautoscalers/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (288, '删除HPA', 'k8s-hpa:delete', 3, 74, '', '', '', 89, 1, 1, '/api/v1/plugins/kubernetes/resources/horizontalpodautoscalers/:ns/:name', 'DELETE', NOW(), NOW()),
  (289, '创建PDB', 'k8s-pdb:create', 3, 74, '', '', '', 90, 1, 1, '/api/v1/plugins/kubernetes/resources/poddisruptionbudgets/:ns/yaml', 'POST', NOW(), NOW()),
  (290, '编辑PDB', 'k8s-pdb:update', 3, 74, '', '', '', 91, 1, 1, '/api/v1/plugins/kubernetes/resources/poddisruptionbudgets/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (291, '删除PDB', 'k8s-pdb:delete', 3, 74, '', '', '', 92, 1, 1, '/api/v1/plugins/kubernetes/resources/poddisruptionbudgets/:ns/:name', 'DELETE', NOW(), NOW()),
  (292, '创建PVC', 'k8s-pvc:create', 3, 75, '', '', '', 93, 1, 1, '/api/v1/plugins/kubernetes/resources/persistentvolumeclaims/:ns/yaml', 'POST', NOW(), NOW()),
  (293, '编辑PVC', 'k8s-pvc:update', 3, 75, '', '', '', 94, 1, 1, '/api/v1/plugins/kubernetes/resources/persistentvolumeclaims/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (294, '删除PVC', 'k8s-pvc:delete', 3, 75, '', '', '', 95, 1, 1, '/api/v1/plugins/kubernetes/resources/persistentvolumeclaims/:ns/:name', 'DELETE', NOW(), NOW()),
  (295, '创建PV', 'k8s-pv:create', 3, 75, '', '', '', 96, 1, 1, '/api/v1/plugins/kubernetes/resources/persistentvolumes/yaml', 'POST', NOW(), NOW()),
  (296, '编辑PV', 'k8s-pv:update', 3, 75, '', '', '', 97, 1, 1, '/api/v1/plugins/kubernetes/resources/persistentvolumes/:name/yaml', 'PUT', NOW(), NOW()),
  (297, '删除PV', 'k8s-pv:delete', 3, 75, '', '', '', 98, 1, 1, '/api/v1/plugins/kubernetes/resources/persistentvolumes/:name', 'DELETE', NOW(), NOW()),
  (298, '创建StorageClass', 'k8s-storageclasses:create', 3, 75, '', '', '', 99, 1, 1, '/api/v1/plugins/kubernetes/resources/storageclasses/yaml', 'POST', NOW(), NOW()),
  (299, '编辑StorageClass', 'k8s-storageclasses:update', 3, 75, '', '', '', 100, 1, 1, '/api/v1/plugins/kubernetes/resources/storageclasses/:name/yaml', 'PUT', NOW(), NOW()),
  (300, '删除StorageClass', 'k8s-storageclasses:delete', 3, 75, '', '', '', 101, 1, 1, '/api/v1/plugins/kubernetes/resources/storageclasses/:name', 'DELETE', NOW(), NOW()),
  (301, '编辑ServiceAccount', 'k8s-serviceaccounts:update', 3, 76, '', '', '', 102, 1, 1, '/api/v1/plugins/kubernetes/resources/serviceaccounts/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (302, '删除ServiceAccount', 'k8s-serviceaccounts:delete', 3, 76, '', '', '', 103, 1, 1, '/api/v1/plugins/kubernetes/resources/serviceaccounts/:ns/:name', 'DELETE', NOW(), NOW()),
  (303, '编辑Role', 'k8s-roles:update', 3, 76, '', '', '', 104, 1, 1, '/api/v1/plugins/kubernetes/resources/roles/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (304, '删除Role', 'k8s-roles:delete', 3, 76, '', '', '', 105, 1, 1, '/api/v1/plugins/kubernetes/resources/roles/:ns/:name', 'DELETE', NOW(), NOW()),
  (305, '编辑ClusterRole', 'k8s-clusterroles:update', 3, 76, '', '', '', 106, 1, 1, '/api/v1/plugins/kubernetes/resources/clusterroles/:name/yaml', 'PUT', NOW(), NOW()),
  (306, '删除ClusterRole', 'k8s-clusterroles:delete', 3, 76, '', '', '', 107, 1, 1, '/api/v1/plugins/kubernetes/resources/clusterroles/:name', 'DELETE', NOW(), NOW()),
  (307, '编辑RoleBinding', 'k8s-rolebindings:update', 3, 76, '', '', '', 108, 1, 1, '/api/v1/plugins/kubernetes/resources/rolebindings/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (308, '删除RoleBinding', 'k8s-rolebindings:delete', 3, 76, '', '', '', 109, 1, 1, '/api/v1/plugins/kubernetes/resources/rolebindings/:ns/:name', 'DELETE', NOW(), NOW()),
  (309, '编辑ClusterRoleBinding', 'k8s-clusterrolebindings:update', 3, 76, '', '', '', 110, 1, 1, '/api/v1/plugins/kubernetes/resources/clusterrolebindings/:name/yaml', 'PUT', NOW(), NOW()),
  (310, '删除ClusterRoleBinding', 'k8s-clusterrolebindings:delete', 3, 76, '', '', '', 111, 1, 1, '/api/v1/plugins/kubernetes/resources/clusterrolebindings/:name', 'DELETE', NOW(), NOW()),
  (311, '开始巡检', 'k8s-inspection:start', 3, 86, '', '', '', 112, 1, 1, '/api/v1/plugins/kubernetes/inspection/start', 'POST', NOW(), NOW()),
  (312, '创建模板', 'templates:create', 3, 82, '', '', '', 113, 1, 1, '/api/v1/plugins/task/templates', 'POST', NOW(), NOW()),
  (313, '编辑模板', 'templates:update', 3, 82, '', '', '', 114, 1, 1, '/api/v1/plugins/task/templates/:id', 'PUT', NOW(), NOW()),
  (314, '删除模板', 'templates:delete', 3, 82, '', '', '', 115, 1, 1, '/api/v1/plugins/task/templates/:id', 'DELETE', NOW(), NOW()),
  (315, '执行任务', 'tasks:execute', 3, 83, '', '', '', 116, 1, 1, '/api/v1/plugins/task/execute', 'POST', NOW(), NOW()),
  (316, '执行分发', 'task-distribute:execute', 3, 84, '', '', '', 117, 1, 1, '/api/v1/plugins/task/distribute', 'POST', NOW(), NOW()),
  (317, '创建身份源', 'identity-sources:create', 3, 102, '', '', '', 118, 1, 1, '/api/v1/identity/sources', 'POST', NOW(), NOW()),
  (318, '编辑身份源', 'identity-sources:update', 3, 102, '', '', '', 119, 1, 1, '/api/v1/identity/sources/:id', 'PUT', NOW(), NOW()),
  (319, '删除身份源', 'identity-sources:delete', 3, 102, '', '', '', 120, 1, 1, '/api/v1/identity/sources/:id', 'DELETE', NOW(), NOW()),
  (320, '创建应用', 'identity-apps:create', 3, 103, '', '', '', 121, 1, 1, '/api/v1/identity/apps', 'POST', NOW(), NOW()),
  (321, '编辑应用', 'identity-apps:update', 3, 103, '', '', '', 122, 1, 1, '/api/v1/identity/apps/:id', 'PUT', NOW(), NOW()),
  (322, '删除应用', 'identity-apps:delete', 3, 103, '', '', '', 123, 1, 1, '/api/v1/identity/apps/:id', 'DELETE', NOW(), NOW()),
  (323, '创建凭证', 'identity-creds:create', 3, 104, '', '', '', 124, 1, 1, '/api/v1/identity/credentials', 'POST', NOW(), NOW()),
  (324, '编辑凭证', 'identity-creds:update', 3, 104, '', '', '', 125, 1, 1, '/api/v1/identity/credentials/:id', 'PUT', NOW(), NOW()),
  (325, '删除凭证', 'identity-creds:delete', 3, 104, '', '', '', 126, 1, 1, '/api/v1/identity/credentials/:id', 'DELETE', NOW(), NOW()),
  (326, '创建权限', 'identity-perms:create', 3, 105, '', '', '', 127, 1, 1, '/api/v1/identity/permissions', 'POST', NOW(), NOW()),
  (327, '删除权限', 'identity-perms:delete', 3, 105, '', '', '', 128, 1, 1, '/api/v1/identity/permissions/:id', 'DELETE', NOW(), NOW()),
  -- 数据日志按钮 (parent_id=26)
  (328, '批量删除', 'data-logs:batch-delete', 3, 26, '', '', '', 1, 1, 1, '/api/v1/audit/data-logs/batch-delete', 'POST', NOW(), NOW()),
  (329, '删除日志', 'data-logs:delete', 3, 26, '', '', '', 2, 1, 1, '/api/v1/audit/data-logs/:id', 'DELETE', NOW(), NOW()),
  -- 操作日志查询按钮 (parent_id=24)
  (330, '查询日志', 'op-logs:search', 3, 24, '', '', '', 0, 1, 1, '/api/v1/audit/operation-logs', 'GET', NOW(), NOW()),
  -- 登录日志查询按钮 (parent_id=25)
  (331, '查询日志', 'login-logs:search', 3, 25, '', '', '', 0, 1, 1, '/api/v1/audit/login-logs', 'GET', NOW(), NOW()),
  -- 执行历史按钮 (parent_id=87)
  (332, '批量删除', 'task-history:batch-delete', 3, 87, '', '', '', 1, 1, 1, '/api/v1/plugins/task/execution-history/batch-delete', 'POST', NOW(), NOW()),
  (333, '导出记录', 'task-history:export', 3, 87, '', '', '', 2, 1, 1, '/api/v1/plugins/task/execution-history/export', 'POST', NOW(), NOW()),
  (334, '删除记录', 'task-history:delete', 3, 87, '', '', '', 3, 1, 1, '/api/v1/plugins/task/execution-history/:id', 'DELETE', NOW(), NOW());

-- 插入菜单API关联
INSERT INTO `sys_menu_api` (`menu_id`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (200, '/api/v1/users', 'POST', NOW(), NOW()),
  (201, '/api/v1/users/:id', 'PUT', NOW(), NOW()),
  (202, '/api/v1/users/:id', 'DELETE', NOW(), NOW()),
  (203, '/api/v1/users/:id/unlock', 'PUT', NOW(), NOW()),
  (204, '/api/v1/users/:id/reset-password', 'PUT', NOW(), NOW()),
  (205, '/api/v1/roles', 'POST', NOW(), NOW()),
  (206, '/api/v1/roles/:id', 'PUT', NOW(), NOW()),
  (207, '/api/v1/roles/:id', 'DELETE', NOW(), NOW()),
  (208, '/api/v1/roles/:id/menus', 'PUT', NOW(), NOW()),
  (209, '/api/v1/menus', 'POST', NOW(), NOW()),
  (210, '/api/v1/menus/:id', 'PUT', NOW(), NOW()),
  (211, '/api/v1/menus/:id', 'DELETE', NOW(), NOW()),
  (212, '/api/v1/departments', 'POST', NOW(), NOW()),
  (213, '/api/v1/departments/:id', 'PUT', NOW(), NOW()),
  (214, '/api/v1/departments/:id', 'DELETE', NOW(), NOW()),
  (215, '/api/v1/positions', 'POST', NOW(), NOW()),
  (216, '/api/v1/positions/:id', 'PUT', NOW(), NOW()),
  (217, '/api/v1/positions/:id', 'DELETE', NOW(), NOW()),
  (218, '/api/v1/positions/:id/users', 'POST', NOW(), NOW()),
  (219, '/api/v1/system/config/basic', 'PUT', NOW(), NOW()),
  (219, '/api/v1/system/config/security', 'PUT', NOW(), NOW()),
  (220, '/api/v1/hosts', 'POST', NOW(), NOW()),
  (220, '/api/v1/hosts/import', 'POST', NOW(), NOW()),
  (220, '/api/v1/cloud-accounts/import', 'POST', NOW(), NOW()),
  (221, '/api/v1/hosts/:id', 'PUT', NOW(), NOW()),
  (222, '/api/v1/hosts/:id', 'DELETE', NOW(), NOW()),
  (223, '/api/v1/hosts/batch-delete', 'POST', NOW(), NOW()),
  (224, '/api/v1/hosts/:id/collect', 'POST', NOW(), NOW()),
  (225, '/api/v1/hosts/:id/files', 'GET', NOW(), NOW()),
  (226, '/api/v1/asset-groups', 'POST', NOW(), NOW()),
  (227, '/api/v1/asset-groups/:id', 'PUT', NOW(), NOW()),
  (228, '/api/v1/asset-groups/:id', 'DELETE', NOW(), NOW()),
  (229, '/api/v1/credentials', 'POST', NOW(), NOW()),
  (230, '/api/v1/credentials/:id', 'PUT', NOW(), NOW()),
  (231, '/api/v1/credentials/:id', 'DELETE', NOW(), NOW()),
  (232, '/api/v1/cloud-accounts', 'POST', NOW(), NOW()),
  (233, '/api/v1/cloud-accounts/import', 'POST', NOW(), NOW()),
  (234, '/api/v1/cloud-accounts/:id', 'PUT', NOW(), NOW()),
  (235, '/api/v1/cloud-accounts/:id', 'DELETE', NOW(), NOW()),
  (236, '/api/v1/terminal-sessions/:id', 'DELETE', NOW(), NOW()),
  (237, '/api/v1/asset-permissions', 'POST', NOW(), NOW()),
  (238, '/api/v1/asset-permissions/:id', 'PUT', NOW(), NOW()),
  (239, '/api/v1/asset-permissions/:id', 'DELETE', NOW(), NOW()),
  (240, '/api/v1/audit/operation-logs/batch-delete', 'POST', NOW(), NOW()),
  (241, '/api/v1/audit/operation-logs/:id', 'DELETE', NOW(), NOW()),
  (242, '/api/v1/audit/login-logs/batch-delete', 'POST', NOW(), NOW()),
  (243, '/api/v1/audit/login-logs/:id', 'DELETE', NOW(), NOW()),
  (244, '/api/v1/plugins/:name/enable', 'POST', NOW(), NOW()),
  (245, '/api/v1/plugins/:name/disable', 'POST', NOW(), NOW()),
  (246, '/api/v1/plugins/:name/uninstall', 'DELETE', NOW(), NOW()),
  (247, '/api/v1/plugins/upload', 'POST', NOW(), NOW()),
  (248, '/api/v1/plugins/kubernetes/clusters', 'POST', NOW(), NOW()),
  (249, '/api/v1/plugins/kubernetes/clusters/:id', 'PUT', NOW(), NOW()),
  (250, '/api/v1/plugins/kubernetes/clusters/:id', 'DELETE', NOW(), NOW()),
  (251, '/api/v1/plugins/kubernetes/clusters/:id/sync', 'POST', NOW(), NOW()),
  (251, '/api/v1/plugins/kubernetes/clusters/sync-all', 'POST', NOW(), NOW()),
  (252, '/api/v1/plugins/kubernetes/clusters/:id/sync', 'POST', NOW(), NOW()),
  (253, '/api/v1/plugins/kubernetes/clusters/:id', 'DELETE', NOW(), NOW()),
  (254, '/api/v1/plugins/kubernetes/clusters/kubeconfig', 'POST', NOW(), NOW()),
  (255, '/api/v1/plugins/kubernetes/clusters/kubeconfig', 'DELETE', NOW(), NOW()),
  (256, '/api/v1/plugins/kubernetes/resources/namespaces', 'POST', NOW(), NOW()),
  (257, '/api/v1/plugins/kubernetes/workloads/update', 'POST', NOW(), NOW()),
  (258, '/api/v1/plugins/kubernetes/workloads/update', 'POST', NOW(), NOW()),
  (259, '/api/v1/plugins/kubernetes/workloads/update', 'POST', NOW(), NOW()),
  (260, '/api/v1/plugins/kubernetes/workloads/update', 'POST', NOW(), NOW()),
  (261, '/api/v1/plugins/kubernetes/workloads/delete', 'DELETE', NOW(), NOW()),
  (262, '/api/v1/plugins/kubernetes/resources/services/:ns/:name', 'POST', NOW(), NOW()),
  (263, '/api/v1/plugins/kubernetes/resources/services/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (264, '/api/v1/plugins/kubernetes/resources/services/:ns/:name', 'DELETE', NOW(), NOW()),
  (265, '/api/v1/plugins/kubernetes/resources/ingresses/:ns/:name', 'POST', NOW(), NOW()),
  (266, '/api/v1/plugins/kubernetes/resources/ingresses/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (267, '/api/v1/plugins/kubernetes/resources/ingresses/:ns/:name', 'DELETE', NOW(), NOW()),
  (268, '/api/v1/plugins/kubernetes/resources/endpoints/:ns/yaml', 'POST', NOW(), NOW()),
  (269, '/api/v1/plugins/kubernetes/resources/endpoints/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (270, '/api/v1/plugins/kubernetes/resources/endpoints/:ns/:name', 'DELETE', NOW(), NOW()),
  (271, '/api/v1/plugins/kubernetes/resources/networkpolicies/:ns/yaml', 'POST', NOW(), NOW()),
  (272, '/api/v1/plugins/kubernetes/resources/networkpolicies/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (273, '/api/v1/plugins/kubernetes/resources/networkpolicies/:ns/:name', 'DELETE', NOW(), NOW()),
  (274, '/api/v1/plugins/kubernetes/resources/configmaps/:ns/yaml', 'POST', NOW(), NOW()),
  (275, '/api/v1/plugins/kubernetes/resources/configmaps/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (276, '/api/v1/plugins/kubernetes/resources/configmaps/:ns/:name', 'DELETE', NOW(), NOW()),
  (277, '/api/v1/plugins/kubernetes/resources/secrets/:ns/yaml', 'POST', NOW(), NOW()),
  (278, '/api/v1/plugins/kubernetes/resources/secrets/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (279, '/api/v1/plugins/kubernetes/resources/secrets/:ns/:name', 'DELETE', NOW(), NOW()),
  (280, '/api/v1/plugins/kubernetes/resources/resourcequotas/:ns/yaml', 'POST', NOW(), NOW()),
  (281, '/api/v1/plugins/kubernetes/resources/resourcequotas/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (282, '/api/v1/plugins/kubernetes/resources/resourcequotas/:ns/:name', 'DELETE', NOW(), NOW()),
  (283, '/api/v1/plugins/kubernetes/resources/limitranges/:ns/yaml', 'POST', NOW(), NOW()),
  (284, '/api/v1/plugins/kubernetes/resources/limitranges/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (285, '/api/v1/plugins/kubernetes/resources/limitranges/:ns/:name', 'DELETE', NOW(), NOW()),
  (286, '/api/v1/plugins/kubernetes/resources/horizontalpodautoscalers/:ns/yaml', 'POST', NOW(), NOW()),
  (287, '/api/v1/plugins/kubernetes/resources/horizontalpodautoscalers/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (288, '/api/v1/plugins/kubernetes/resources/horizontalpodautoscalers/:ns/:name', 'DELETE', NOW(), NOW()),
  (289, '/api/v1/plugins/kubernetes/resources/poddisruptionbudgets/:ns/yaml', 'POST', NOW(), NOW()),
  (290, '/api/v1/plugins/kubernetes/resources/poddisruptionbudgets/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (291, '/api/v1/plugins/kubernetes/resources/poddisruptionbudgets/:ns/:name', 'DELETE', NOW(), NOW()),
  (292, '/api/v1/plugins/kubernetes/resources/persistentvolumeclaims/:ns/yaml', 'POST', NOW(), NOW()),
  (293, '/api/v1/plugins/kubernetes/resources/persistentvolumeclaims/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (294, '/api/v1/plugins/kubernetes/resources/persistentvolumeclaims/:ns/:name', 'DELETE', NOW(), NOW()),
  (295, '/api/v1/plugins/kubernetes/resources/persistentvolumes/yaml', 'POST', NOW(), NOW()),
  (296, '/api/v1/plugins/kubernetes/resources/persistentvolumes/:name/yaml', 'PUT', NOW(), NOW()),
  (297, '/api/v1/plugins/kubernetes/resources/persistentvolumes/:name', 'DELETE', NOW(), NOW()),
  (298, '/api/v1/plugins/kubernetes/resources/storageclasses/yaml', 'POST', NOW(), NOW()),
  (299, '/api/v1/plugins/kubernetes/resources/storageclasses/:name/yaml', 'PUT', NOW(), NOW()),
  (300, '/api/v1/plugins/kubernetes/resources/storageclasses/:name', 'DELETE', NOW(), NOW()),
  (301, '/api/v1/plugins/kubernetes/resources/serviceaccounts/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (302, '/api/v1/plugins/kubernetes/resources/serviceaccounts/:ns/:name', 'DELETE', NOW(), NOW()),
  (303, '/api/v1/plugins/kubernetes/resources/roles/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (304, '/api/v1/plugins/kubernetes/resources/roles/:ns/:name', 'DELETE', NOW(), NOW()),
  (305, '/api/v1/plugins/kubernetes/resources/clusterroles/:name/yaml', 'PUT', NOW(), NOW()),
  (306, '/api/v1/plugins/kubernetes/resources/clusterroles/:name', 'DELETE', NOW(), NOW()),
  (307, '/api/v1/plugins/kubernetes/resources/rolebindings/:ns/:name/yaml', 'PUT', NOW(), NOW()),
  (308, '/api/v1/plugins/kubernetes/resources/rolebindings/:ns/:name', 'DELETE', NOW(), NOW()),
  (309, '/api/v1/plugins/kubernetes/resources/clusterrolebindings/:name/yaml', 'PUT', NOW(), NOW()),
  (310, '/api/v1/plugins/kubernetes/resources/clusterrolebindings/:name', 'DELETE', NOW(), NOW()),
  (311, '/api/v1/plugins/kubernetes/inspection/start', 'POST', NOW(), NOW()),
  (312, '/api/v1/plugins/task/templates', 'POST', NOW(), NOW()),
  (313, '/api/v1/plugins/task/templates/:id', 'PUT', NOW(), NOW()),
  (314, '/api/v1/plugins/task/templates/:id', 'DELETE', NOW(), NOW()),
  (315, '/api/v1/plugins/task/execute', 'POST', NOW(), NOW()),
  (316, '/api/v1/plugins/task/distribute', 'POST', NOW(), NOW()),
  (317, '/api/v1/identity/sources', 'POST', NOW(), NOW()),
  (318, '/api/v1/identity/sources/:id', 'PUT', NOW(), NOW()),
  (319, '/api/v1/identity/sources/:id', 'DELETE', NOW(), NOW()),
  (320, '/api/v1/identity/apps', 'POST', NOW(), NOW()),
  (321, '/api/v1/identity/apps/:id', 'PUT', NOW(), NOW()),
  (322, '/api/v1/identity/apps/:id', 'DELETE', NOW(), NOW()),
  (323, '/api/v1/identity/credentials', 'POST', NOW(), NOW()),
  (324, '/api/v1/identity/credentials/:id', 'PUT', NOW(), NOW()),
  (325, '/api/v1/identity/credentials/:id', 'DELETE', NOW(), NOW()),
  (326, '/api/v1/identity/permissions', 'POST', NOW(), NOW()),
  (327, '/api/v1/identity/permissions/:id', 'DELETE', NOW(), NOW()),
  (328, '/api/v1/audit/data-logs/batch-delete', 'POST', NOW(), NOW()),
  (329, '/api/v1/audit/data-logs/:id', 'DELETE', NOW(), NOW()),
  (330, '/api/v1/audit/operation-logs', 'GET', NOW(), NOW()),
  (331, '/api/v1/audit/login-logs', 'GET', NOW(), NOW()),
  (332, '/api/v1/plugins/task/execution-history/batch-delete', 'POST', NOW(), NOW()),
  (333, '/api/v1/plugins/task/execution-history/export', 'POST', NOW(), NOW()),
  (334, '/api/v1/plugins/task/execution-history/:id', 'DELETE', NOW(), NOW());

-- 为管理员角色分配所有按钮权限
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (1, 200), (1, 201), (1, 202), (1, 203), (1, 204), (1, 205), (1, 206), (1, 207), (1, 208), (1, 209),
  (1, 210), (1, 211), (1, 212), (1, 213), (1, 214), (1, 215), (1, 216), (1, 217), (1, 218), (1, 219),
  (1, 220), (1, 221), (1, 222), (1, 223), (1, 224), (1, 225), (1, 226), (1, 227), (1, 228), (1, 229),
  (1, 230), (1, 231), (1, 232), (1, 233), (1, 234), (1, 235), (1, 236), (1, 237), (1, 238), (1, 239),
  (1, 240), (1, 241), (1, 242), (1, 243), (1, 244), (1, 245), (1, 246), (1, 247), (1, 248), (1, 249),
  (1, 250), (1, 251), (1, 252), (1, 253), (1, 254), (1, 255), (1, 256), (1, 257), (1, 258), (1, 259),
  (1, 260), (1, 261), (1, 262), (1, 263), (1, 264), (1, 265), (1, 266), (1, 267), (1, 268), (1, 269),
  (1, 270), (1, 271), (1, 272), (1, 273), (1, 274), (1, 275), (1, 276), (1, 277), (1, 278), (1, 279),
  (1, 280), (1, 281), (1, 282), (1, 283), (1, 284), (1, 285), (1, 286), (1, 287), (1, 288), (1, 289),
  (1, 290), (1, 291), (1, 292), (1, 293), (1, 294), (1, 295), (1, 296), (1, 297), (1, 298), (1, 299),
  (1, 300), (1, 301), (1, 302), (1, 303), (1, 304), (1, 305), (1, 306), (1, 307), (1, 308), (1, 309),
  (1, 310), (1, 311), (1, 312), (1, 313), (1, 314), (1, 315), (1, 316), (1, 317), (1, 318), (1, 319),
  (1, 320), (1, 321), (1, 322), (1, 323), (1, 324), (1, 325), (1, 326), (1, 327),
  (1, 328), (1, 329), (1, 330), (1, 331), (1, 332), (1, 333), (1, 334);

-- Total: 128 button permissions, 132 API bindings

-- ============================================================
-- 11. 插件状态表
-- ============================================================

-- 插件状态表（用于记录插件启用/禁用状态）
CREATE TABLE IF NOT EXISTS `plugin_states` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT '插件名称',
  `enabled` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否启用 1:启用 0:禁用',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 默认启用所有内置插件
INSERT INTO `plugin_states` (`name`, `enabled`, `created_at`, `updated_at`)
VALUES
  ('kubernetes', 1, NOW(), NOW()),
  ('monitor', 1, NOW(), NOW()),
  ('task', 1, NOW(), NOW()),
  ('ssl-cert', 1, NOW(), NOW()),
  ('nginx', 1, NOW(), NOW());

-- 插入默认系统配置
INSERT INTO `sys_config` (`key`, `value`, `type`, `group`, `remark`, `created_at`, `updated_at`)
VALUES
  -- 基础配置
  ('system_name', 'OpsHub', 'string', 'basic', '系统名称', NOW(), NOW()),
  ('system_logo', '', 'string', 'basic', '系统Logo路径', NOW(), NOW()),
  ('system_description', '运维管理平台', 'string', 'basic', '系统描述', NOW(), NOW()),
  -- 安全配置
  ('password_min_length', '8', 'int', 'security', '密码最小长度', NOW(), NOW()),
  ('session_timeout', '3600', 'int', 'security', 'Session超时时间(秒)', NOW(), NOW()),
  ('enable_captcha', 'true', 'bool', 'security', '是否开启验证码', NOW(), NOW()),
  ('max_login_attempts', '5', 'int', 'security', '最大登录失败次数', NOW(), NOW()),
  ('lockout_duration', '300', 'int', 'security', '账户锁定时间(秒)', NOW(), NOW());

SET FOREIGN_KEY_CHECKS = 1;

-- 创建默认的admin用户
-- 密码: 123456
-- 警告: 生产环境请立即修改默认密码!
INSERT INTO `sys_user` (`id`, `username`, `password`, `real_name`, `email`, `status`, `department_id`, `created_at`, `updated_at`)
VALUES (1, 'admin', '$2a$10$RLkgoedTSa0dYj3ujbXMcunSED3c6GLvfdKYsmpz0l0YFZbVrSBqW', '系统管理员', 'admin@opshub.io', 1, 1, NOW(), NOW());

-- 关联admin用户到admin角色
INSERT INTO `sys_user_role` (`user_id`, `role_id`) VALUES (1, 1);

-- ============================================================
-- 12. 中间件管理 — 菜单与按钮权限（增量）
-- ============================================================

-- 12.1 页面菜单：资产管理(parent_id=15)下新增两个子菜单
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (88, '中间件管理', 'middleware-management', 2, 15, '/asset/middlewares', 'asset/Middlewares', 'Coin', 7, 1, 1, '', '', NOW(), NOW()),
  (89, '中间件权限', 'middleware-permission', 2, 15, '/asset/middleware-permissions', 'asset/MiddlewarePermission', 'Lock', 8, 1, 1, '', '', NOW(), NOW());

-- 12.2 按钮权限(type=3)
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  -- 中间件管理按钮 (parent_id=88)
  (335, '新增中间件', 'middlewares:create', 3, 88, '', '', '', 1, 1, 1, '/api/v1/middlewares', 'POST', NOW(), NOW()),
  (336, '编辑中间件', 'middlewares:update', 3, 88, '', '', '', 2, 1, 1, '/api/v1/middlewares/:id', 'PUT', NOW(), NOW()),
  (337, '删除中间件', 'middlewares:delete', 3, 88, '', '', '', 3, 1, 1, '/api/v1/middlewares/:id', 'DELETE', NOW(), NOW()),
  (338, '批量删除',   'middlewares:batch-delete', 3, 88, '', '', '', 4, 1, 1, '/api/v1/middlewares/batch-delete', 'POST', NOW(), NOW()),
  (339, '测试连接',   'middlewares:connect', 3, 88, '', '', '', 5, 1, 1, '/api/v1/middlewares/:id/test', 'POST', NOW(), NOW()),
  (340, '数据操作',   'middlewares:execute', 3, 88, '', '', '', 6, 1, 1, '/api/v1/middlewares/:id/execute', 'POST', NOW(), NOW()),
  -- 中间件权限按钮 (parent_id=89)
  (341, '添加权限', 'middleware-perms:create', 3, 89, '', '', '', 1, 1, 1, '/api/v1/middleware-permissions', 'POST', NOW(), NOW()),
  (342, '编辑权限', 'middleware-perms:update', 3, 89, '', '', '', 2, 1, 1, '/api/v1/middleware-permissions/:id', 'PUT', NOW(), NOW()),
  (343, '删除权限', 'middleware-perms:delete', 3, 89, '', '', '', 3, 1, 1, '/api/v1/middleware-permissions/:id', 'DELETE', NOW(), NOW());

-- 12.3 菜单API关联
INSERT INTO `sys_menu_api` (`menu_id`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (335, '/api/v1/middlewares', 'POST', NOW(), NOW()),
  (336, '/api/v1/middlewares/:id', 'PUT', NOW(), NOW()),
  (337, '/api/v1/middlewares/:id', 'DELETE', NOW(), NOW()),
  (338, '/api/v1/middlewares/batch-delete', 'POST', NOW(), NOW()),
  (339, '/api/v1/middlewares/:id/test', 'POST', NOW(), NOW()),
  (340, '/api/v1/middlewares/:id/execute', 'POST', NOW(), NOW()),
  (341, '/api/v1/middleware-permissions', 'POST', NOW(), NOW()),
  (342, '/api/v1/middleware-permissions/:id', 'PUT', NOW(), NOW()),
  (343, '/api/v1/middleware-permissions/:id', 'DELETE', NOW(), NOW());

-- 12.4 为管理员角色(role_id=1)分配中间件菜单和按钮权限
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (1, 88), (1, 89),
  (1, 335), (1, 336), (1, 337), (1, 338), (1, 339), (1, 340),
  (1, 341), (1, 342), (1, 343);

-- ============================================================
-- 13. 中间件审计日志菜单与按钮权限
-- ============================================================

-- 13.1 页面菜单：操作审计(parent_id=23)下新增中间件审计
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (344, '中间件审计', 'middleware-audit-logs', 2, 23, '/audit/middleware-audit-logs', 'audit/MiddlewareAuditLogs', 'DataLine', 4, 1, 1, '', '', NOW(), NOW());

-- 13.2 按钮权限(type=3, parent_id=344)
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (345, '查询日志', 'mw-audit:search', 3, 344, '', '', '', 0, 1, 1, '/api/v1/audit/middleware-audit-logs', 'GET', NOW(), NOW()),
  (346, '批量删除', 'mw-audit:batch-delete', 3, 344, '', '', '', 1, 1, 1, '/api/v1/audit/middleware-audit-logs/batch-delete', 'POST', NOW(), NOW()),
  (347, '删除日志', 'mw-audit:delete', 3, 344, '', '', '', 2, 1, 1, '/api/v1/audit/middleware-audit-logs/:id', 'DELETE', NOW(), NOW());

-- 13.3 菜单API关联
INSERT INTO `sys_menu_api` (`menu_id`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (345, '/api/v1/audit/middleware-audit-logs', 'GET', NOW(), NOW()),
  (346, '/api/v1/audit/middleware-audit-logs/batch-delete', 'POST', NOW(), NOW()),
  (347, '/api/v1/audit/middleware-audit-logs/:id', 'DELETE', NOW(), NOW());

-- 13.4 为管理员角色(role_id=1)分配中间件审计菜单和按钮权限
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (1, 344), (1, 345), (1, 346), (1, 347);
