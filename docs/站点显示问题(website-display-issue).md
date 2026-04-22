# Web站点列表显示问题排查

## 问题描述
后端返回了正确的数据，但前端表格没有显示。

## 后端返回数据（正常）
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": 2,
        "name": "baidu",
        "url": "https://www.baidu.com",
        "icon": "📡",
        ...
      }
    ],
    "total": 2
  }
}
```

## 已修复的问题

### 1. TypeScript 接口类型不匹配
**问题**：后端返回的 `groupNames`、`groupIds` 等字段为 `null`，但接口定义为数组类型。

**修复**：
```typescript
// 修复前
groupNames: string[]
groupIds: number[]

// 修复后
groupNames: string[] | null
groupIds: number[] | null
```

### 2. 缺少 row-key 配置
**问题**：Arco Design 表格需要唯一的 row-key。

**修复**：
```vue
<a-table
  :columns="columns"
  :data="tableData"
  row-key="id"
  ...
>
```

### 3. 添加详细调试日志
```typescript
console.log('站点列表响应:', res)
console.log('解析后的数据:', data)
console.log('data.list:', data.list)
console.log('tableData.value:', tableData.value)
console.log('tableData.value.length:', tableData.value.length)
```

## 排查步骤

### 步骤 1: 检查浏览器控制台
1. 打开浏览器开发者工具（F12）
2. 切换到 Console 标签
3. 刷新页面
4. 查看以下日志：
   - "站点列表响应:" - 应该看到完整的 API 响应
   - "data.list:" - 应该看到包含 2 条记录的数组
   - "tableData.value:" - 应该看到赋值后的数据
   - "tableData.value.length:" - 应该显示 2

### 步骤 2: 检查 Network 请求
1. 切换到 Network 标签
2. 刷新页面
3. 找到 `/api/v1/websites` 请求
4. 检查：
   - Status: 应该是 200
   - Response: 应该包含 2 条记录
   - Headers: Content-Type 应该是 application/json

### 步骤 3: 检查 Vue DevTools
1. 安装 Vue DevTools 浏览器扩展
2. 打开 Vue DevTools
3. 找到 Websites 组件
4. 检查 data：
   - tableData: 应该包含 2 条记录
   - tableLoading: 应该是 false
   - pagination.total: 应该是 2

### 步骤 4: 检查表格渲染
1. 在 Elements 标签中检查 DOM
2. 找到 `.websites-table` 元素
3. 检查是否有 `<tbody>` 和 `<tr>` 元素
4. 如果没有，可能是表格组件的问题

## 可能的原因

### 原因 A: 数据格式问题
如果控制台显示 `data.list` 是 undefined 或 null：
- 检查 API 响应格式
- 检查数据解析逻辑

### 原因 B: 响应式失效
如果 `tableData.value` 有数据但表格不显示：
- 可能是 Vue 响应式系统的问题
- 尝试使用 `nextTick` 或强制刷新

### 原因 C: 表格组件配置问题
如果数据正确但表格不渲染：
- 检查 Arco Design 版本
- 检查表格列配置
- 检查是否有 CSS 隐藏了表格

### 原因 D: 路由或权限问题
如果整个页面不显示：
- 检查路由配置
- 检查权限控制
- 检查是否有错误拦截

## 临时调试方案

在表格上方添加调试信息：

```vue
<template>
  <div>
    <!-- 调试信息 -->
    <div style="padding: 10px; background: #f0f0f0; margin-bottom: 10px;">
      <p>tableData.length: {{ tableData.length }}</p>
      <p>pagination.total: {{ pagination.total }}</p>
      <p>tableLoading: {{ tableLoading }}</p>
      <pre>{{ JSON.stringify(tableData, null, 2) }}</pre>
    </div>

    <!-- 原有表格 -->
    <a-table ...>
  </div>
</template>
```

## 下一步操作

1. **立即检查**：打开浏览器控制台，查看日志输出
2. **如果有日志**：根据日志内容判断问题所在
3. **如果没有日志**：说明 `loadWebsites()` 没有执行，检查 `onMounted` 钩子
4. **如果数据正确但不显示**：添加临时调试信息到模板中

## 修改文件
- `web/src/api/website.ts` - 修复接口类型定义
- `web/src/views/asset/Websites.vue` - 添加 row-key 和调试日志

## 更新时间
2026-03-09 22:00
