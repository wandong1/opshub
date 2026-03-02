# Skill: 新增页面时注册菜单与按钮权限

当新增一个前端页面（路由）时，必须同时创建对应的菜单记录、按钮权限记录，并为按钮绑定 API 接口。否则非管理员用户将无法看到菜单，按钮权限也无法生效。

## 菜单体系结构

OpsHub 使用三级菜单体系，存储在 `sys_menu` 表中：

```
目录 (Type=1)  — 侧边栏一级分组，如"智能巡检"
├── 菜单 (Type=2)  — 侧边栏二级页面入口，如"拨测管理"
│   ├── 按钮 (Type=3)  — 页面内操作按钮，如"新增拨测"
│   │   └── APIs (sys_menu_api)  — 按钮绑定的后端 API
│   └── 按钮 (Type=3)
└── 菜单 (Type=2)
    └── 按钮 (Type=3)
```

## SysMenu 关键字段

| 字段 | 说明 | 示例 |
|------|------|------|
| id | 主键，手动指定以避免冲突 | 400 |
| name | 菜单名称 | "拨测管理" |
| code | 权限编码（唯一），按钮用于 `v-permission` 指令 | "inspection:probes:create" |
| type | 1=目录 2=菜单 3=按钮 | 2 |
| parent_id | 父菜单 ID | 目录的 ID |
| path | 前端路由路径（仅 Type=2） | "/inspection/probes" |
| component | 前端组件路径（仅 Type=2） | "inspection/ProbeManagement" |
| icon | 图标名称（仅 Type=1/2） | "Connection" |
| sort | 排序，数字越小越靠前 | 0 |
| visible | 1=显示 0=隐藏 | 1 |
| status | 1=启用 0=禁用 | 1 |
| api_path | 按钮主 API 路径（Type=3） | "/api/v1/inspection/probes" |
| api_method | HTTP 方法 | "POST" |

## SysMenuAPI 关键字段

一个按钮可绑定多个 API（通过 `sys_menu_api` 表）：

| 字段 | 说明 |
|------|------|
| menu_id | 关联的按钮菜单 ID |
| api_path | API 路径，支持 `:id` 占位符 |
| api_method | GET / POST / PUT / DELETE |

## 操作步骤

### 1. 在 `migrations/` 目录下创建 SQL 迁移文件

文件命名规范：`migrations/YYYYMMDD_模块名.sql`

SQL 文件需包含以下 4 部分，参考 `migrations/20260214_inspection_module.sql`：

```sql
-- ============================================================
-- XXX模块 — 菜单与按钮权限
-- ============================================================

-- 1. 目录 (type=1)
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (500, 'XXX模块', 'xxx-dir', 1, 0, '', '', 'Monitor', 90, 1, 1, '', '', NOW(), NOW());

-- 2. 页面菜单 (type=2, parent_id=目录ID)
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (501, 'XXX管理', 'xxx:page', 2, 500, '/xxx/page', 'xxx/Page', 'List', 0, 1, 1, '', '', NOW(), NOW());

-- 3. 按钮权限 (type=3, parent_id=菜单ID)
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (510, '查询', 'xxx:page:list',   3, 501, '', '', '', 0, 1, 1, '/api/v1/xxx/page',     'GET',    NOW(), NOW()),
  (511, '新增', 'xxx:page:create', 3, 501, '', '', '', 1, 1, 1, '/api/v1/xxx/page',     'POST',   NOW(), NOW()),
  (512, '编辑', 'xxx:page:update', 3, 501, '', '', '', 2, 1, 1, '/api/v1/xxx/page/:id', 'PUT',    NOW(), NOW()),
  (513, '删除', 'xxx:page:delete', 3, 501, '', '', '', 3, 1, 1, '/api/v1/xxx/page/:id', 'DELETE', NOW(), NOW());

-- 4. 按钮 API 关联 (sys_menu_api)
INSERT INTO `sys_menu_api` (`menu_id`, `api_path`, `api_method`, `created_at`, `updated_at`)
VALUES
  (510, '/api/v1/xxx/page',     'GET',    NOW(), NOW()),
  (511, '/api/v1/xxx/page',     'POST',   NOW(), NOW()),
  (512, '/api/v1/xxx/page/:id', 'PUT',    NOW(), NOW()),
  (513, '/api/v1/xxx/page/:id', 'DELETE', NOW(), NOW());

-- 5. 为管理员角色 (role_id=1) 分配权限
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES
  (1, 500), (1, 501),
  (1, 510), (1, 511), (1, 512), (1, 513);
```

**ID 分配规则：**
- 查看 `migrations/init.sql` 和已有迁移文件中使用的最大 ID
- 当前已使用的 ID 段：1-89（核心系统）、100-106（身份认证目录/菜单）、317-347（按钮）、400-434（智能巡检）
- 新模块从下一个百位整数开始（如 500、600...）
- 目录和菜单用连续 ID（500, 501, 502...），按钮从 +10 的整十位开始（510, 520...），留出扩展空间

### 2. 前端按钮添加 `v-permission` 指令

在 Vue 页面的操作按钮上使用 `v-permission` 指令，值为按钮的 `code`：

```vue
<el-button v-permission="'xxx:page:create'" @click="handleCreate">新增</el-button>
<el-button v-permission="'xxx:page:update'" link @click="handleEdit(row)">编辑</el-button>
<el-button v-permission="'xxx:page:delete'" link @click="handleDelete(row)">删除</el-button>
```

### 3. 前端路由中添加路由条目

在 `web/src/router/index.ts` 的 Layout children 中添加：

```typescript
{
  path: 'xxx/page',
  name: 'XxxPage',
  component: () => import('@/views/xxx/Page.vue'),
  meta: { title: 'XXX管理' }
},
```

## 权限编码命名规范

- 目录：`{module}-dir`（如 `inspection-dir`）
- 菜单：`{module}:{resource}`（如 `inspection:probes`）
- 按钮：`{module}:{resource}:{action}`（如 `inspection:probes:create`）
- 常用 action：`list` / `create` / `update` / `delete` / `import` / `export` / `execute`

## 权限生效流程

```
用户登录 → 获取角色 → 角色关联菜单 → 菜单树返回前端
  → permissionStore 提取 type=3 的 code → v-permission 指令控制按钮显隐
  → RequirePermission 中间件校验 API 访问权限（通过 sys_menu_api 匹配）
```

## 注意事项

- `code` 字段有唯一索引，不能重复
- 菜单 ID 手动指定，避免不同环境自增 ID 不一致
- 新菜单必须通过 `sys_role_menu` 分配给管理员角色（role_id=1），否则管理员也看不到
- 按钮的 API 绑定决定了后端权限校验，漏绑会导致非管理员用户 403
- 管理员角色（code=admin）在前端自动拥有所有权限，但后端仍需通过 API 绑定校验
- SQL 文件需要手动执行（`mysql -u root -p opshub < migrations/xxx.sql`），不会自动运行

## 参考文件

- 完整示例：`migrations/20260214_inspection_module.sql`
- 核心初始化：`migrations/init.sql`（第 12、13 节为增量菜单示例）
- 身份认证模块：`migrations/20260130_identity_module.sql`
