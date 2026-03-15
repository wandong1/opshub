-- ============================================================
-- 智能巡检菜单结构修复脚本
-- 用途：删除重复的智能巡检一级菜单，将新功能合并到现有模块
-- ============================================================

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- 1. 删除错误创建的重复智能巡检一级菜单及其子菜单和按钮
DELETE FROM `sys_menu` WHERE `id` IN (91, 92, 93, 94, 95);
DELETE FROM `sys_menu` WHERE `id` BETWEEN 378 AND 388;
DELETE FROM `sys_menu_api` WHERE `menu_id` IN (91, 92, 93, 94, 95);
DELETE FROM `sys_menu_api` WHERE `menu_id` BETWEEN 378 AND 388;
DELETE FROM `sys_role_menu` WHERE `menu_id` IN (91, 92, 93, 94, 95);
DELETE FROM `sys_role_menu` WHERE `menu_id` BETWEEN 378 AND 388;

-- 2. 在现有智能巡检模块(parent_id=352)下新增子菜单
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (389, '巡检组管理', 'inspection_groups', 2, 352, '/inspection/groups', 'inspection/InspectionGroups', 'Folder', 5, 1, 1, '', '', NOW(), NOW()),
  (390, '巡检项管理', 'inspection_items', 2, 352, '/inspection/items', 'inspection/InspectionItems', 'List', 6, 1, 1, '', '', NOW(), NOW()),
  (391, '执行记录', 'inspection_records', 2, 352, '/inspection/records', 'inspection/InspectionRecords', 'History', 7, 1, 1, '', '', NOW(), NOW());

-- 3. 新增按钮权限
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  -- 巡检组管理按钮 (parent_id=389)
  (392, '新增巡检组', 'inspection_groups:create', 3, 389, '', '', '', 1, 1, 1, '/api/v1/inspection/groups', 'POST', NOW(), NOW()),
  (393, '编辑巡检组', 'inspection_groups:update', 3, 389, '', '', '', 2, 1, 1, '/api/v1/inspection/groups/:id', 'PUT', NOW(), NOW()),
  (394, '删除巡检组', 'inspection_groups:delete', 3, 389, '', '', '', 3, 1, 1, '/api/v1/inspection/groups/:id', 'DELETE', NOW(), NOW()),
  -- 巡检项管理按钮 (parent_id=390)
  (395, '新增巡检项', 'inspection_items:create', 3, 390, '', '', '', 1, 1, 1, '/api/v1/inspection/items', 'POST', NOW(), NOW()),
  (396, '编辑巡检项', 'inspection_items:update', 3, 390, '', '', '', 2, 1, 1, '/api/v1/inspection/items/:id', 'PUT', NOW(), NOW()),
  (397, '删除巡检项', 'inspection_items:delete', 3, 390, '', '', '', 3, 1, 1, '/api/v1/inspection/items/:id', 'DELETE', NOW(), NOW()),
  (398, '测试运行', 'inspection_items:test', 3, 390, '', '', '', 4, 1, 1, '/api/v1/inspection/items/test-run', 'POST', NOW(), NOW());

-- 4. 菜单API关联
INSERT INTO `sys_menu_api` (`menu_id`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  -- 巡检组管理页面需要的API
  (389, '/api/v1/inspection/groups', 'GET', NOW(), NOW()),
  (389, '/api/v1/inspection/groups/:id', 'GET', NOW(), NOW()),
  (389, '/api/v1/inspection/groups/all', 'GET', NOW(), NOW()),
  (389, '/api/v1/asset-groups/tree', 'GET', NOW(), NOW()),
  -- 新增巡检组按钮关联的API
  (392, '/api/v1/inspection/groups', 'POST', NOW(), NOW()),
  (392, '/api/v1/asset-groups/tree', 'GET', NOW(), NOW()),
  -- 编辑巡检组按钮关联的API
  (393, '/api/v1/inspection/groups/:id', 'PUT', NOW(), NOW()),
  (393, '/api/v1/inspection/groups/:id', 'GET', NOW(), NOW()),
  (393, '/api/v1/asset-groups/tree', 'GET', NOW(), NOW()),
  -- 删除巡检组按钮关联的API
  (394, '/api/v1/inspection/groups/:id', 'DELETE', NOW(), NOW()),

  -- 巡检项管理页面需要的API
  (390, '/api/v1/inspection/items', 'GET', NOW(), NOW()),
  (390, '/api/v1/inspection/items/:id', 'GET', NOW(), NOW()),
  (390, '/api/v1/inspection/groups/all', 'GET', NOW(), NOW()),
  (390, '/api/v1/hosts', 'GET', NOW(), NOW()),
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
  (398, '/api/v1/inspection/items/test-run', 'POST', NOW(), NOW()),

  -- 执行记录页面需要的API
  (391, '/api/v1/inspection/records', 'GET', NOW(), NOW()),
  (391, '/api/v1/inspection/records/:id', 'GET', NOW(), NOW()),
  (391, '/api/v1/inspection/groups/all', 'GET', NOW(), NOW()),
  (391, '/api/v1/inspection/items', 'GET', NOW(), NOW()),
  (391, '/api/v1/hosts', 'GET', NOW(), NOW());

-- 5. 为管理员角色(role_id=1)分配新增的巡检功能菜单和按钮权限
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (1, 389),  -- 巡检组管理
  (1, 390),  -- 巡检项管理
  (1, 391),  -- 执行记录
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
  (2, 389),  -- 巡检组管理（仅查看）
  (2, 390),  -- 巡检项管理（仅查看）
  (2, 391);  -- 执行记录（仅查看）

SET FOREIGN_KEY_CHECKS = 1;

-- 修复完成！
-- 现在智能巡检模块(ID=352)包含以下子菜单：
-- 1. 拨测管理 (ID=353)
-- 2. 任务调度 (ID=354)
-- 3. Pushgateway配置 (ID=355)
-- 4. 环境变量 (ID=356)
-- 5. 巡检组管理 (ID=389) [新增]
-- 6. 巡检项管理 (ID=390) [新增]
-- 7. 执行记录 (ID=391) [新增]
