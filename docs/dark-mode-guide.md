# 深色模式适配指南

## 概述

OpsHub 项目已实现完整的深色模式支持，包括：
1. **全局适配**：Arco Design 和 Element Plus 组件自动适配
2. **通用规则**：常见的自定义组件类名自动适配
3. **页面级适配**：特殊页面需要单独适配

## 已适配的内容

### 1. 全局组件适配（自动生效）

**Arco Design 组件**（`arco-theme.css`）：
- 表单组件：Input, Select, Textarea, DatePicker, Switch, Radio, Checkbox
- 数据展示：Table, Card, Descriptions, Tree, Timeline, Steps
- 导航组件：Menu, Tabs, Breadcrumb, Pagination
- 反馈组件：Modal, Drawer, Message, Notification
- 其他：Tag, Badge, Progress, Divider, Empty, Spin

**Element Plus 组件**（`App.vue`）：
- 表单、表格、对话框、下拉菜单、分页器等
- 完整的深色模式适配

### 2. 通用类名适配（自动生效）

**容器类**（`dark-mode-common.css`）：
```css
.page-container, .card, .panel, .box, .section
.filter-bar, .toolbar, .sidebar, .table-wrapper
.form-container, .list-item, .header, .title
```

**状态类**：
```css
.selected, .active, .disabled, .hover
.text-primary, .text-secondary, .bg-white
```

### 3. 已适配的页面

- ✅ Layout.vue - 侧边栏和顶栏
- ✅ Hosts.vue - 资产管理页面（统计卡片、筛选栏、表格）

## 如何为新页面添加深色模式适配

### 方法 1：使用通用类名（推荐）

如果你的组件使用了通用类名，无需额外适配：

```vue
<template>
  <div class="page-container">
    <div class="card">
      <div class="header">标题</div>
      <div class="content">内容</div>
    </div>
  </div>
</template>
```

### 方法 2：在组件 `<style>` 中添加深色模式规则

如果使用了自定义类名，在组件的 `<style>` 中添加：

```vue
<style scoped>
.my-custom-card {
  background: #fff;
  padding: 20px;
}

/* 深色模式适配 */
body[arco-theme='dark'] .my-custom-card {
  background: #232324;
  color: rgba(255, 255, 255, 0.9);
}
</style>
```

### 方法 3：使用 CSS 变量（最佳实践）

使用预定义的 CSS 变量，自动适配深色模式：

```vue
<style scoped>
.my-card {
  background: var(--ops-header-bg);
  color: var(--ops-text-primary);
  border: 1px solid var(--ops-border-color);
}
</style>
```

## CSS 变量参考

### 浅色模式（默认）
```css
--ops-primary: #165dff
--ops-sidebar-bg: #ffffff
--ops-header-bg: #ffffff
--ops-content-bg: #f7f8fa
--ops-border-color: #e5e6eb
--ops-text-primary: #1d2129
--ops-text-secondary: #4e5969
--ops-text-tertiary: #86909c
```

### 深色模式（自动切换）
```css
--ops-primary: #4080ff
--ops-sidebar-bg: #17171a
--ops-header-bg: #232324
--ops-content-bg: #17171a
--ops-border-color: rgba(255, 255, 255, 0.15)
--ops-text-primary: rgba(255, 255, 255, 0.9)
--ops-text-secondary: rgba(255, 255, 255, 0.6)
--ops-text-tertiary: rgba(255, 255, 255, 0.4)
```

## 深色模式颜色规范

### 背景色
- **主背景**：`#232324` - 卡片、表格、输入框
- **次背景**：`#2a2a2b` - 表头、标题栏
- **内容背景**：`#17171a` - 页面背景

### 文字色
- **主文字**：`rgba(255, 255, 255, 0.9)` - 90% 白色
- **次文字**：`rgba(255, 255, 255, 0.6)` - 60% 白色
- **辅助文字**：`rgba(255, 255, 255, 0.4)` - 40% 白色

### 边框和分隔线
- **边框**：`rgba(255, 255, 255, 0.15)` - 15% 白色半透明

### 交互状态
- **悬停背景**：`rgba(255, 255, 255, 0.08)` - 8% 白色半透明
- **选中背景**：`rgba(64, 128, 255, 0.15)` - 15% 蓝色半透明

### 阴影
- **卡片阴影**：`0 2px 8px rgba(0, 0, 0, 0.3)`
- **悬停阴影**：`0 8px 24px rgba(0, 0, 0, 0.4)`

## 常见问题

### Q1: 我的页面有白色背景，如何适配？

**A**: 检查是否使用了硬编码的白色背景：

```css
/* ❌ 错误 - 硬编码白色 */
.my-card {
  background: #fff;
}

/* ✅ 正确 - 使用 CSS 变量 */
.my-card {
  background: var(--ops-header-bg);
}

/* ✅ 或者添加深色模式规则 */
.my-card {
  background: #fff;
}
body[arco-theme='dark'] .my-card {
  background: #232324;
}
```

### Q2: 如何适配第三方组件？

**A**: 在全局样式中添加覆盖规则：

```css
/* 在 dark-mode-common.css 中 */
body[arco-theme='dark'] .third-party-component {
  background-color: #232324 !important;
  color: rgba(255, 255, 255, 0.9) !important;
}
```

### Q3: 如何测试深色模式？

**A**: 
1. 点击顶栏右侧的主题图标
2. 选择"深色"主题
3. 检查页面是否有白色背景区域
4. 检查文字是否清晰可读

### Q4: 如何调试深色模式样式？

**A**: 使用浏览器开发者工具：
1. 切换到深色模式
2. 右键点击白色区域 → 检查元素
3. 查看元素的类名和样式
4. 在对应的 Vue 文件或 CSS 文件中添加深色模式规则

## 适配检查清单

为新页面添加深色模式适配时，请检查：

- [ ] 页面背景色是否适配
- [ ] 卡片/面板背景色是否适配
- [ ] 表格背景色是否适配
- [ ] 表单输入框是否适配
- [ ] 文字颜色是否清晰可读
- [ ] 边框颜色是否可见
- [ ] 悬停状态是否有视觉反馈
- [ ] 选中状态是否明显
- [ ] 阴影效果是否协调
- [ ] 图标颜色是否适配

## 示例：完整的页面适配

```vue
<template>
  <div class="my-page">
    <!-- 使用通用类名 -->
    <div class="page-header">
      <h2 class="page-title">页面标题</h2>
    </div>

    <!-- 使用 CSS 变量 -->
    <div class="content-card">
      <div class="card-header">卡片标题</div>
      <div class="card-body">卡片内容</div>
    </div>

    <!-- 自定义组件需要适配 -->
    <div class="custom-panel">
      自定义面板
    </div>
  </div>
</template>

<style scoped>
/* 使用 CSS 变量 */
.content-card {
  background: var(--ops-header-bg);
  border: 1px solid var(--ops-border-color);
  border-radius: 8px;
}

.card-header {
  color: var(--ops-text-primary);
  border-bottom: 1px solid var(--ops-border-color);
}

.card-body {
  color: var(--ops-text-secondary);
}

/* 自定义组件需要手动适配 */
.custom-panel {
  background: #fff;
  padding: 20px;
}

body[arco-theme='dark'] .custom-panel {
  background: #232324;
  color: rgba(255, 255, 255, 0.9);
}
</style>
```

## 贡献指南

如果你发现某个页面在深色模式下显示异常：

1. 定位问题元素的类名
2. 在对应的 Vue 文件中添加深色模式规则
3. 或者在 `dark-mode-common.css` 中添加通用规则
4. 提交 PR 时说明适配的页面和组件

## 参考资源

- Arco Design 深色主题：https://arco.design/vue/docs/dark
- CSS 变量定义：`web/src/styles/arco-theme.css`
- 通用适配规则：`web/src/styles/dark-mode-common.css`
- Element Plus 适配：`web/src/App.vue`
