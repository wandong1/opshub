# 前端 UI 风格迁移：Element Plus → Arco Design Vue

## Context

当前项目前端使用 Element Plus 组件库，配色为金色+黑色。用户希望将整体 UI 风格迁移为 Arco Design Vue，使界面更加简洁大气。迁移采用渐进式策略：两个库共存，逐步替换。本次仅改前端 UI，不动后端代码和数据结构。

配色方案：Arco 默认蓝色(#165DFF)为基础，后续支持用户自定义主题色。

## Phase 1 范围（本次实施）

先改主体框架 + 登录页 + Dashboard，验证效果后再批量迁移。

## 1. 安装依赖

```bash
cd web && npm install @arco-design/web-vue
```

保留 `element-plus` 和 `@element-plus/icons-vue`，未迁移页面继续使用。

## 2. 修改文件清单

### `web/src/main.ts`
- 引入 Arco Design Vue + CSS + Icon
- 同时保留 Element Plus 注册
- 导入顺序：Arco CSS → 自定义主题 CSS → Element Plus CSS

### `web/src/styles/arco-theme.css`（新建）
- 定义 OpsHub 主题 CSS 变量（侧边栏、头部、内容区、卡片等）
- 覆盖 Arco 默认 CSS 变量实现品牌定制
- 使用 CSS 变量架构，为后续用户自定义主题色做准备

### `web/src/App.vue`
- 保留现有 Element Plus 全局样式覆盖（兼容未迁移页面）
- 新增 Arco 全局样式微调
- 调整 html font-size 从 20px 回归 14px（Arco 标准）

### `web/src/style.css`
- 清理 Vite 默认样式（dark mode 背景色、max-width 限制等）
- 保留弹窗抖动修复

### `web/src/views/Layout.vue`（重写）
组件替换：
- `el-container/el-aside/el-header/el-main` → `a-layout / a-layout-sider / a-layout-header / a-layout-content`
- `el-menu / el-sub-menu / el-menu-item` → `a-menu / a-sub-menu / a-menu-item`
- `el-breadcrumb` → `a-breadcrumb`
- `el-dropdown` → `a-dropdown`
- `el-avatar` → `a-avatar`
- Element Plus icons → Arco icons（建立映射表）

新增功能：
- 侧边栏折叠/展开（Arco `a-layout-sider` 内置 `collapsible`）
- 菜单点击路由跳转（`@menu-item-click` 替代 `router` prop）

### `web/src/views/Login.vue`（重写）
组件替换：
- `el-form / el-form-item` → `a-form / a-form-item`（`prop` → `field`）
- `el-input` → `a-input` / `a-input-password`
- `el-checkbox` → `a-checkbox`
- `el-button` → `a-button`
- `ElMessage` → `Message`（from `@arco-design/web-vue`）

表单验证：Arco 使用 `@submit-success` 事件或 `formRef.validate()` 返回 Promise。

### `web/src/views/Dashboard.vue`（重写）
组件替换：
- `el-row / el-col` → `a-row / a-col`（API 基本一致）
- `el-card` → `a-card`（`shadow="hover"` → `hoverable`，`#header` → `#title` + `#extra`）
- `el-button link` → `a-link`
- `el-icon` → Arco icon 组件
- ECharts 部分不变

### `web/src/utils/request.ts`
- `ElMessage` → `Message`（from `@arco-design/web-vue`）
- `ElMessage.error({ message, duration, showClose })` → `Message.error({ content, duration, closable })`

### `web/src/plugins/manager.ts`
- `ElMessage` → `Message`

### `CLAUDE.md`
- 新增 UI 风格规范章节，包含：
  - 组件映射表（Element Plus → Arco Design）
  - 图标映射规则
  - 主题变量说明
  - 表格迁移模式（`:columns` prop + `slotName` 插槽）
  - 新页面必须使用 Arco Design 的规则

## 3. 关键设计决策

### 图标映射
创建 `iconMap` 对象，将 Element Plus 图标名映射到 Arco 图标组件：
```
HomeFilled → IconHome, User → IconUser, Setting → IconSettings,
Document → IconFile, Monitor → IconDesktop, Lock → IconLock, ...
```

### 表格迁移模式（Phase 2 用）
Arco `a-table` 使用 `:columns` prop + `slotName` 定义列：
```typescript
const columns = [
  { title: '用户名', dataIndex: 'username' },
  { title: '操作', slotName: 'operations' }
]
```
复杂单元格通过 `slotName` 在 template 中定义，避免 render 函数。

### CSS 变量架构
所有自定义颜色通过 CSS 变量定义，为后续主题切换做准备：
```css
:root {
  --ops-sidebar-bg: #232324;
  --ops-primary: var(--color-primary-6, #165dff);
  --ops-header-bg: #ffffff;
  --ops-content-bg: #f7f8fa;
}
```

## 4. 验证方式

1. `cd web && npx vue-tsc --noEmit` — 类型检查通过
2. `cd web && npm run build` — 构建成功
3. 手动验证：
   - 登录页显示正常，表单验证工作
   - Layout 侧边栏菜单正常展开/折叠/路由跳转
   - Dashboard 卡片和图表正常渲染
   - 未迁移页面（如 Users、Roles 等）仍使用 Element Plus 正常工作

## 5. Phase 2+（后续批量迁移，本次不做）

按模块逐步迁移：
1. system/ — Users, Roles, Menus, DeptInfo, PositionInfo, SystemConfig
2. asset/ — Hosts, Middlewares, CloudAccounts, Credentials, Groups, Terminal 等
3. audit/ — OperationLogs, LoginLogs, DataLogs, MiddlewareAuditLogs
4. inspection/ — ProbeManagement, TaskSchedule, PushgatewayConfig
5. identity/ — IdentitySources, SSOApplications, Permissions 等
6. kubernetes/ — 最大模块，54 个文件，最后迁移
7. task/ — Templates, Execute, FileDistribution 等
8. plugin/ — PluginList, PluginInstall
9. 清理：移除 Element Plus 依赖
