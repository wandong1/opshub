-- ============================================================
-- 智能巡检菜单结构优化脚本
-- 用途：将巡检组管理和巡检项管理合并为一个"巡检管理"页面
-- ============================================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- 1. 删除原有的巡检组管理和巡检项管理菜单
DELETE FROM `sys_menu` WHERE `id` IN (389, 390);
DELETE FROM `sys_menu` WHERE `id` BETWEEN 392 AND 398;
DELETE FROM `sys_menu_api` WHERE `menu_id` IN (389, 390);
DELETE FROM `sys_menu_api` WHERE `menu_id` BETWEEN 392 AND 398;
DELETE FROM `sys_role_menu` WHERE `menu_id` IN (389, 390);
DELETE FROM `sys_role_menu` WHERE `menu_id` BETWEEN 392 AND 398;

-- 2. 新增合并后的"巡检管理"菜单
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (389, '巡检管理', 'inspection_management', 2, 352, '/inspection/management', 'inspection/InspectionManagement', 'FolderAdd', 5, 1, 1, '', '', NOW(), NOW());

-- 3. 新增按钮权限（合并巡检组和巡检项的操作）
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  -- 巡检管理按钮 (parent_id=389)
  (392, '新增巡检组', 'inspection_management:create_group', 3, 389, '', '', '', 1, 1, 1, '/api/v1/inspection/groups', 'POST', NOW(), NOW()),
  (393, '编辑巡检组', 'inspection_management:update_group', 3, 389, '', '', '', 2, 1, 1, '/api/v1/inspection/groups/:id', 'PUT', NOW(), NOW()),
  (394, '删除巡检组', 'inspection_management:delete_group', 3, 389, '', '', '', 3, 1, 1, '/api/v1/inspection/groups/:id', 'DELETE', NOW(), NOW()),
  (395, '新增巡检项', 'inspection_management:create_item', 3, 389, '', '', '', 4, 1, 1, '/api/v1/inspection/items', 'POST', NOW(), NOW()),
  (396, '编辑巡检项', 'inspection_management:update_item', 3, 389, '', '', '', 5, 1, 1, '/api/v1/inspection/items/:id', 'PUT', NOW(), NOW()),
  (397, '删除巡检项', 'inspection_management:delete_item', 3, 389, '', '', '', 6, 1, 1, '/api/v1/inspection/items/:id', 'DELETE', NOW(), NOW()),
  (398, '测试运行', 'inspection_management:test', 3, 389, '', '', '', 7, 1, 1, '/api/v1/inspection/items/test-run', 'POST', NOW(), NOW());

-- 4. 菜单API关联
INSERT INTO `sys_menu_api` (`menu_id`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  -- 巡检管理页面需要的API
  (389, '/api/v1/inspection/groups', 'GET', NOW(), NOW()),
  (389, '/api/v1/inspection/groups/:id', 'GET', NOW(), NOW()),
  (389, '/api/v1/inspection/groups/all', 'GET', NOW(), NOW()),
  (389, '/api/v1/inspection/items', 'GET', NOW(), NOW()),
  (389, '/api/v1/inspection/items/:id', 'GET', NOW(), NOW()),
  (389, '/api/v1/asset-groups/tree', 'GET', NOW(), NOW()),
  (389, '/api/v1/hosts', 'GET', NOW(), NOW()),

  -- 新增巡检组按钮关联的API
  (392, '/api/v1/inspection/groups', 'POST', NOW(), NOW()),
  (392, '/api/v1/asset-groups/tree', 'GET', NOW(), NOW()),

  -- 编辑巡检组按钮关联的API
  (393, '/api/v1/inspection/groups/:id', 'PUT', NOW(), NOW()),
  (393, '/api/v1/inspection/groups/:id', 'GET', NOW(), NOW()),
  (393, '/api/v1/inspection/items', 'GET', NOW(), NOW()),
  (393, '/api/v1/asset-groups/tree', 'GET', NOW(), NOW()),

  -- 删除巡检组按钮关联的API
  (394, '/api/v1/inspection/groups/:id', 'DELETE', NOW(), NOW()),

  -- 新增巡检项按钮关联的API
  (395, '/api/v1/inspection/items', 'POST', NOW(), NOW()),
  (395, '/api/v1/inspection/groups/all', 'GET', NOW(), NOW()),
  (395, '/api/v1/hosts', 'GET', NOW(), NOW()),

  -- 编辑巡检项按钮关联的API
  (396, '/api/v1/inspection/items/:id', 'PUT', NOW(), NOW()),
  (396, '/api/v1/inspection/items/:id', 'GET', NOW(), NOW()),
  (396, '/api/v1/inspection/groups/all', 'GET', NOW(), NOW()),
  (396, '/api/v1/hosts', 'GET', NOW(), NOW()),

  -- 删除巡检项按钮关联的API
  (397, '/api/v1/inspection/items/:id', 'DELETE', NOW(), NOW()),

  -- 测试运行按钮关联的API
  (398, '/api/v1/inspection/items/test-run', 'POST', NOW(), NOW());

-- 5. 为管理员角色(role_id=1)分配巡检管理菜单和按钮权限
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (1, 389),  -- 巡检管理
  (1, 392),  -- 新增巡检组
  (1, 393),  -- 编辑巡检组
  (1, 394),  -- 删除巡检组
  (1, 395),  -- 新增巡检项
  (1, 396),  -- 编辑巡检项
  (1, 397),  -- 删除巡检项
  (1, 398);  -- 测试运行

-- 6. 为普通用户角色(role_id=2)分配基础查看权限
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (2, 389);  -- 巡检管理（仅查看）

SET FOREIGN_KEY_CHECKS = 1;

-- 优化完成！
-- 现在智能巡检模块(ID=352)包含以下子菜单：
-- 1. 拨测管理 (ID=353)
-- 2. 任务调度 (ID=354)
-- 3. Pushgateway配置 (ID=355)
-- 4. 环境变量 (ID=356)
-- 5. 巡检管理 (ID=389) [合并了巡检组和巡检项]
-- 6. 执行记录 (ID=391)
