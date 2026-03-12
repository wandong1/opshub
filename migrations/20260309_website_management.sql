-- ============================================================
-- Web站点管理功能菜单数据
-- 创建时间: 2026-03-09
-- 说明: 在资产管理模块下新增Web站点管理功能
-- ============================================================

-- 1. 插入Web站点管理菜单 (parent_id=15 资产管理)
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (90, 'Web站点管理', 'asset_websites', 2, 15, '/asset/websites', 'asset/Websites', 'Link', 9, 1, 1, '', '', NOW(), NOW());

-- 2. 插入Web站点管理按钮权限 (type=3, parent_id=90)
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (374, '新增站点', 'websites:create', 3, 90, '', '', '', 1, 1, 1, '/api/v1/websites', 'POST', NOW(), NOW()),
  (375, '编辑站点', 'websites:update', 3, 90, '', '', '', 2, 1, 1, '/api/v1/websites/:id', 'PUT', NOW(), NOW()),
  (376, '删除站点', 'websites:delete', 3, 90, '', '', '', 3, 1, 1, '/api/v1/websites/:id', 'DELETE', NOW(), NOW()),
  (377, '访问站点', 'websites:access', 3, 90, '', '', '', 4, 1, 1, '/api/v1/websites/:id/access', 'GET', NOW(), NOW());

-- 3. 插入菜单API关联 (sys_menu_api表)
INSERT INTO `sys_menu_api` (`menu_id`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  -- 站点列表页面需要的API
  (90, '/api/v1/websites', 'GET', NOW(), NOW()),
  (90, '/api/v1/websites/:id', 'GET', NOW(), NOW()),
  (90, '/api/v1/asset-groups/tree', 'GET', NOW(), NOW()),
  -- 新增站点按钮关联的API
  (374, '/api/v1/websites', 'POST', NOW(), NOW()),
  (374, '/api/v1/asset-groups/tree', 'GET', NOW(), NOW()),
  (374, '/api/v1/hosts', 'GET', NOW(), NOW()),
  -- 编辑站点按钮关联的API
  (375, '/api/v1/websites/:id', 'PUT', NOW(), NOW()),
  (375, '/api/v1/websites/:id', 'GET', NOW(), NOW()),
  (375, '/api/v1/asset-groups/tree', 'GET', NOW(), NOW()),
  (375, '/api/v1/hosts', 'GET', NOW(), NOW()),
  -- 删除站点按钮关联的API
  (376, '/api/v1/websites/:id', 'DELETE', NOW(), NOW()),
  -- 访问站点按钮关联的API
  (377, '/api/v1/websites/:id/access', 'GET', NOW(), NOW()),
  (377, '/api/v1/websites/:id/proxy/*path', 'ANY', NOW(), NOW());

-- 4. 为管理员角色分配Web站点管理菜单权限
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (1, 90),   -- Web站点管理菜单
  (1, 374),  -- 新增站点按钮
  (1, 375),  -- 编辑站点按钮
  (1, 376),  -- 删除站点按钮
  (1, 377);  -- 访问站点按钮

-- 5. 为普通用户角色分配基础查看权限
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (2, 90),   -- Web站点管理菜单（仅查看）
  (2, 377);  -- 访问站点按钮

-- ============================================================
-- 说明:
-- 1. 菜单ID 90: Web站点管理主菜单，位于资产管理(15)下，排序为9
-- 2. 按钮权限ID 374-377: 新增、编辑、删除、访问四个操作按钮
-- 3. sys_menu_api 关联了所有相关的后端API接口
-- 4. 管理员(role_id=1)拥有所有权限
-- 5. 普通用户(role_id=2)仅有查看和访问权限
-- ============================================================
