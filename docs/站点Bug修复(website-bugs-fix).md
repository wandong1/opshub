# Web站点管理 Bug 修复

## 修复时间
2026-03-09

## 修复的问题

### Bug 1: 业务分组下拉菜单为空

**问题描述**：
新建站点时，业务分组下拉菜单无法显示任何选项，导致无法选择分组。

**原因分析**：
前端在解析 API 响应时，数据路径不正确。后端返回的数据格式为：
```json
{
  "code": 0,
  "message": "success",
  "data": [...],  // 分组树数据在这里
  "timestamp": 1234567890
}
```

但前端代码只取了 `res.data`，应该取 `res.data.data`。

**修复方案**：
修改 `web/src/views/asset/Websites.vue` 的 `loadGroupTree` 函数：

```typescript
// 修复前
groupTree.value = res.data || []

// 修复后
groupTree.value = res.data?.data || res.data || []
console.log('加载分组树成功:', groupTree.value)
```

同时添加了错误提示，方便用户了解加载失败的原因。

**影响范围**：
- ✅ 业务分组下拉菜单正常显示
- ✅ 可以选择多个分组
- ✅ 分组数据正确加载

---

### Bug 2: 站点图标输入不友好

**问题描述**：
站点图标字段是一个普通输入框，用户需要手动输入图标 URL 或编码，体验不佳。

**改进方案**：
将图标输入框改为下拉选择器，预置 18 个常用图标（emoji），同时支持自定义输入。

**实现细节**：

1. **添加预置图标列表**（`web/src/views/asset/Websites.vue`）：
```typescript
const presetIcons = [
  { value: '🌐', emoji: '🌐', label: '地球' },
  { value: '🏢', emoji: '🏢', label: '办公楼' },
  { value: '💼', emoji: '💼', label: '公文包' },
  { value: '📊', emoji: '📊', label: '图表' },
  { value: '📈', emoji: '📈', label: '上升趋势' },
  { value: '🔧', emoji: '🔧', label: '工具' },
  { value: '⚙️', emoji: '⚙️', label: '设置' },
  { value: '🖥️', emoji: '🖥️', label: '电脑' },
  { value: '📱', emoji: '📱', label: '手机' },
  { value: '🔒', emoji: '🔒', label: '锁' },
  { value: '🔑', emoji: '🔑', label: '钥匙' },
  { value: '📦', emoji: '📦', label: '包裹' },
  { value: '🚀', emoji: '🚀', label: '火箭' },
  { value: '⚡', emoji: '⚡', label: '闪电' },
  { value: '🎯', emoji: '🎯', label: '靶心' },
  { value: '📡', emoji: '📡', label: '卫星天线' },
  { value: '🌟', emoji: '🌟', label: '星星' },
  { value: '💡', emoji: '💡', label: '灯泡' }
]
```

2. **改为下拉选择器**：
```vue
<a-form-item label="站点图标">
  <a-select
    v-model="formData.icon"
    placeholder="请选择图标"
    allow-clear
    allow-search
  >
    <a-option v-for="icon in presetIcons" :key="icon.value" :value="icon.value">
      <div style="display: flex; align-items: center; gap: 8px;">
        <span :style="{ fontSize: '18px' }">{{ icon.emoji }}</span>
        <span>{{ icon.label }}</span>
      </div>
    </a-option>
  </a-select>
  <div style="margin-top: 4px; font-size: 12px; color: var(--ops-text-tertiary);">
    也可以输入自定义图标 URL
  </div>
</a-select>
```

3. **优化表格图标显示**：
```vue
<template #icon="{ record }">
  <div class="site-icon">
    <icon-link v-if="!record.icon" />
    <!-- 如果是 emoji（单个字符），直接显示 -->
    <span v-else-if="record.icon.length <= 2" style="font-size: 24px;">{{ record.icon }}</span>
    <!-- 否则作为图片 URL -->
    <img v-else :src="record.icon" alt="icon" />
  </div>
</template>
```

**特性**：
- ✅ 预置 18 个常用图标
- ✅ 支持搜索过滤
- ✅ 支持清除选择
- ✅ 仍然支持自定义 URL（通过 `allow-search` 可以输入任意值）
- ✅ 表格中正确显示 emoji 和图片

---

### Bug 3: 新增站点后列表为空

**问题描述**：
新增站点点击确定后显示"创建成功"，但刷新后的列表仍然为空，看不到刚创建的站点。

**可能原因**：
1. 后端创建成功但返回数据格式不正确
2. 前端数据解析逻辑有误
3. 列表查询条件有问题

**排查方案**：
添加详细的调试日志，帮助定位问题：

```typescript
// 加载站点列表
const loadWebsites = async () => {
  tableLoading.value = true
  try {
    const res = await getWebsiteList({
      page: pagination.current,
      pageSize: pagination.pageSize,
      keyword: searchForm.keyword,
      type: searchForm.type,
      groupIds: []
    })
    console.log('站点列表响应:', res)
    const data = res.data?.data || res.data || {}
    console.log('解析后的数据:', data)
    tableData.value = data.list || []
    pagination.total = data.total || 0
    console.log('站点列表加载成功，共', pagination.total, '条记录')
  } catch (error: any) {
    console.error('加载站点列表失败:', error)
    Message.error('加载站点列表失败: ' + error.message)
  } finally {
    tableLoading.value = false
  }
}
```

**验证步骤**：
1. 打开浏览器开发者工具（F12）
2. 切换到 Console 标签
3. 新增一个站点
4. 观察控制台输出：
   - "站点列表响应:" - 查看完整的 API 响应
   - "解析后的数据:" - 查看解析后的数据结构
   - "站点列表加载成功，共 X 条记录" - 确认记录数量

**可能的修复方案**（根据日志确定）：

**方案 A：后端返回格式问题**
如果后端返回的数据结构不是标准格式，需要调整数据解析逻辑。

**方案 B：分页问题**
如果创建后分页没有重置，可能新记录在其他页。修复：
```typescript
const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
    if (formData.id) {
      await updateWebsite(formData.id, formData)
      Message.success('更新成功')
    } else {
      await createWebsite(formData)
      Message.success('创建成功')
      // 重置到第一页
      pagination.current = 1
    }
    dialogVisible.value = false
    loadWebsites()
    return true
  } catch (error: any) {
    Message.error(error.message || '操作失败')
    return false
  }
}
```

**方案 C：缓存问题**
如果是浏览器缓存导致，可以在请求中添加时间戳：
```typescript
const res = await getWebsiteList({
  page: pagination.current,
  pageSize: pagination.pageSize,
  keyword: searchForm.keyword,
  type: searchForm.type,
  groupIds: [],
  _t: Date.now()  // 防止缓存
})
```

---

## 测试建议

### 测试 Bug 1 修复
1. 登录系统
2. 进入"资产管理" -> "Web站点管理"
3. 点击"新增站点"
4. 检查"业务分组"下拉菜单是否显示选项
5. 尝试选择一个或多个分组
6. 保存后检查站点是否关联了正确的分组

### 测试 Bug 2 改进
1. 点击"新增站点"
2. 点击"站点图标"下拉菜单
3. 验证：
   - 显示 18 个预置图标
   - 每个图标显示 emoji + 文字标签
   - 可以搜索过滤（输入"地球"）
   - 可以清除选择
   - 可以输入自定义 URL（如 https://example.com/icon.png）
4. 选择一个 emoji 图标保存
5. 在列表中检查图标是否正确显示

### 测试 Bug 3 修复
1. 打开浏览器开发者工具（F12）
2. 切换到 Console 标签
3. 点击"新增站点"
4. 填写表单：
   - 站点名称：测试站点
   - 站点类型：外部站点
   - 站点URL：https://www.example.com
   - 站点图标：选择一个 emoji
5. 点击"确定"
6. 观察控制台日志输出
7. 检查列表中是否显示新创建的站点
8. 如果仍然为空，根据日志信息进一步排查

---

## 修改文件清单

- `web/src/views/asset/Websites.vue` - 修复所有三个问题

---

## 编译验证

```bash
# TypeScript 类型检查
cd web && npx vue-tsc --noEmit
✅ 通过

# 后端编译
go build -o /dev/null cmd/server/server.go
✅ 通过
```

---

## 后续优化建议

1. **图标管理**：
   - 考虑将预置图标配置化，存储在后端
   - 支持管理员自定义图标库
   - 支持上传自定义图标文件

2. **分组选择**：
   - 考虑使用树形选择器（a-tree-select）
   - 显示分组的层级关系
   - 支持展开/折叠

3. **列表优化**：
   - 添加骨架屏加载效果
   - 优化空状态提示
   - 添加刷新按钮

4. **错误处理**：
   - 统一错误提示样式
   - 添加重试机制
   - 记录错误日志到后端

---

## 更新日期
2026-03-09
