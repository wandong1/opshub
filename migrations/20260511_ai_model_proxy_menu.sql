-- AI模型代理菜单配置
-- 执行时间：2026-05-11

-- 1. 插入AI模型代理菜单项（假设资产管理的parent_id为某个值，需要根据实际情况调整）
-- 先查询Web站点管理的parent_id，AI模型代理应该和它在同一级别

-- 插入菜单项（id=92，可根据实际情况调整）
INSERT INTO `sys_menu` (`id`, `parent_id`, `name`, `path`, `component`, `icon`, `sort`, `type`, `status`, `created_at`, `updated_at`)
SELECT
  92,
  parent_id,
  'AI模型代理',
  '/asset/ai-model-proxies',
  'asset/AIModelProxies',
  'icon-robot',
  (SELECT MAX(sort) + 1 FROM sys_menu WHERE parent_id = (SELECT parent_id FROM sys_menu WHERE path = '/asset/websites')),
  1,
  1,
  NOW(),
  NOW()
FROM sys_menu
WHERE path = '/asset/websites'
LIMIT 1;

-- 2. 插入按钮权限
-- 查看
INSERT INTO `sys_menu` (`id`, `parent_id`, `name`, `path`, `component`, `icon`, `sort`, `type`, `status`, `created_at`, `updated_at`)
VALUES (385, 92, '查看', '', '', '', 1, 2, 1, NOW(), NOW());

-- 新增
INSERT INTO `sys_menu` (`id`, `parent_id`, `name`, `path`, `component`, `icon`, `sort`, `type`, `status`, `created_at`, `updated_at`)
VALUES (386, 92, '新增', '', '', '', 2, 2, 1, NOW(), NOW());

-- 编辑
INSERT INTO `sys_menu` (`id`, `parent_id`, `name`, `path`, `component`, `icon`, `sort`, `type`, `status`, `created_at`, `updated_at`)
VALUES (387, 92, '编辑', '', '', '', 3, 2, 1, NOW(), NOW());

-- 删除
INSERT INTO `sys_menu` (`id`, `parent_id`, `name`, `path`, `component`, `icon`, `sort`, `type`, `status`, `created_at`, `updated_at`)
VALUES (388, 92, '删除', '', '', '', 4, 2, 1, NOW(), NOW());

-- 测试连接
INSERT INTO `sys_menu` (`id`, `parent_id`, `name`, `path`, `component`, `icon`, `sort`, `type`, `status`, `created_at`, `updated_at`)
VALUES (389, 92, '测试连接', '', '', '', 5, 2, 1, NOW(), NOW());

-- 重新生成Token
INSERT INTO `sys_menu` (`id`, `parent_id`, `name`, `path`, `component`, `icon`, `sort`, `type`, `status`, `created_at`, `updated_at`)
VALUES (390, 92, '重新生成Token', '', '', '', 6, 2, 1, NOW(), NOW());

-- 3. 插入菜单API关联（假设API表为sys_menu_api）
-- 列表查询
INSERT INTO `sys_menu_api` (`menu_id`, `api_path`, `api_method`)
VALUES (385, '/api/v1/ai-model-proxies', 'GET');

-- 详情查询
INSERT INTO `sys_menu_api` (`menu_id`, `api_path`, `api_method`)
VALUES (385, '/api/v1/ai-model-proxies/:id', 'GET');

-- 创建
INSERT INTO `sys_menu_api` (`menu_id`, `api_path`, `api_method`)
VALUES (386, '/api/v1/ai-model-proxies', 'POST');

-- 更新
INSERT INTO `sys_menu_api` (`menu_id`, `api_path`, `api_method`)
VALUES (387, '/api/v1/ai-model-proxies/:id', 'PUT');

-- 删除
INSERT INTO `sys_menu_api` (`menu_id`, `api_path`, `api_method`)
VALUES (388, '/api/v1/ai-model-proxies/:id', 'DELETE');

-- 测试连接
INSERT INTO `sys_menu_api` (`menu_id`, `api_path`, `api_method`)
VALUES (389, '/api/v1/ai-model-proxies/:id/test', 'GET');

-- 重新生成Token
INSERT INTO `sys_menu_api` (`menu_id`, `api_path`, `api_method`)
VALUES (390, '/api/v1/ai-model-proxies/:id/regenerate-token', 'POST');

-- 4. 为管理员角色分配权限（假设管理员角色ID为1）
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (1, 92),   -- AI模型代理菜单
  (1, 385),  -- 查看
  (1, 386),  -- 新增
  (1, 387),  -- 编辑
  (1, 388),  -- 删除
  (1, 389),  -- 测试连接
  (1, 390);  -- 重新生成Token

-- 注意：
-- 1. 菜单ID（92, 385-390）需要根据实际情况调整，确保不与现有ID冲突
-- 2. parent_id需要根据实际的资产管理菜单结构调整
-- 3. 如果使用了不同的表名或字段名，需要相应修改
-- 4. 建议先在测试环境执行，确认无误后再在生产环境执行
