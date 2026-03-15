# 智能巡检模块 - 主机标签和主机名下拉菜单修复

## 修复日期
2026-03-13

## 问题描述

在巡检项配置中：
- 选择"按标签匹配"时，主机标签下拉菜单为空
- 选择"按主机名匹配"时，主机名下拉菜单为空
- 用户无法看到可选的标签和主机名列表

## 修复方案

### 1. 添加状态变量

在 `InspectionManagement.vue` 中添加两个新的状态变量：

```typescript
const availableTags = ref<string[]>([])        // 存储所有可用的主机标签
const availableHostNames = ref<string[]>([])   // 存储所有可用的主机名
```

### 2. 修改数据加载逻辑

修改 `loadHostsForItem()` 方法，从业务分组的主机列表中提取标签和主机名：

```typescript
const loadHostsForItem = async (item: any) => {
  if (!formData.groupIds || formData.groupIds.length === 0) {
    Message.warning('请先选择业务分组')
    return
  }

  loadingHosts.value = true
  try {
    const hosts: any[] = []
    const tagsSet = new Set<string>()
    const hostNamesSet = new Set<string>()

    // 遍历所有业务分组
    for (const groupId of formData.groupIds) {
      const res = await getHostList({ groupId, page: 1, pageSize: 1000 })
      if (res.list) {
        hosts.push(...res.list)

        // 收集所有主机标签
        res.list.forEach((host: any) => {
          if (host.tags) {
            const tags = host.tags.split(',').map((t: string) => t.trim()).filter((t: string) => t)
            tags.forEach((tag: string) => tagsSet.add(tag))
          }
          // 收集所有主机名
          if (host.name) {
            hostNamesSet.add(host.name)
          }
        })
      }
    }

    // 去重并排序
    availableHosts.value = hosts.filter((host, index, self) =>
      index === self.findIndex(h => h.id === host.id)
    )
    availableTags.value = Array.from(tagsSet).sort()
    availableHostNames.value = Array.from(hostNamesSet).sort()
  } catch (error: any) {
    Message.error(error.message || '加载主机列表失败')
  } finally {
    loadingHosts.value = false
  }
}
```

### 3. 更新下拉菜单组件

**按标签匹配：**
```vue
<a-form-item v-if="item.hostMatchType === 'tag'" label="主机标签" :label-col-flex="'100px'">
  <a-select
    v-model="item.hostTags"
    placeholder="请选择主机标签（可多选）"
    multiple
    allow-create
    allow-search
    :loading="loadingHosts"
    @focus="loadHostsForItem(item)"
  >
    <a-option v-for="tag in availableTags" :key="tag" :value="tag">
      {{ tag }}
    </a-option>
  </a-select>
</a-form-item>
```

**按主机名匹配：**
```vue
<a-form-item v-if="item.hostMatchType === 'name'" label="主机名匹配" :label-col-flex="'100px'">
  <a-select
    v-model="item.hostTags"
    placeholder="请选择主机名（可多选）"
    multiple
    allow-create
    allow-search
    :loading="loadingHosts"
    @focus="loadHostsForItem(item)"
  >
    <a-option v-for="hostName in availableHostNames" :key="hostName" :value="hostName">
      {{ hostName }}
    </a-option>
  </a-select>
</a-form-item>
```

### 4. 添加事件监听

在主机匹配方式的单选框上添加 `@change` 事件：

```vue
<a-radio-group v-model="item.hostMatchType" @change="loadHostsForItem(item)">
  <a-radio value="tag">按标签匹配</a-radio>
  <a-radio value="name">按主机名匹配</a-radio>
  <a-radio value="id">按主机ID匹配</a-radio>
</a-radio-group>
```

### 5. 优化编辑时的数据加载

修改 `handleEdit()` 方法，在编辑巡检组时自动加载主机数据：

```typescript
const handleEdit = async (record: any) => {
  // ... 其他代码 ...

  // 加载主机列表、标签和主机名（用于后续巡检项配置）
  if (groupIds.length > 0) {
    await loadHostsForItem(null)
  }

  // ... 其他代码 ...
}
```

## 功能说明

### 按标签匹配
- 下拉菜单显示该巡检组关联的所有业务分组下的主机标签
- 标签从主机的 `tags` 字段提取（逗号分隔）
- 自动去重和排序
- 支持多选
- 支持手动输入新标签（allow-create）

### 按主机名匹配
- 下拉菜单显示该巡检组关联的所有业务分组下的主机名
- 主机名从主机的 `name` 字段提取
- 自动去重和排序
- 支持多选
- 支持手动输入新主机名关键词（allow-create）

### 按主机ID匹配
- 下拉菜单显示完整的主机列表（主机名 + IP）
- 支持多选
- 不支持手动输入

## 数据流程

1. 用户选择业务分组
2. 用户展开巡检项配置
3. 用户选择主机匹配方式
4. 点击下拉菜单时触发 `@focus` 事件
5. 调用 `loadHostsForItem()` 方法
6. 从所有业务分组加载主机列表
7. 提取标签和主机名，去重排序
8. 更新 `availableTags` 和 `availableHostNames`
9. 下拉菜单显示可选项

## 用户体验优化

1. **加载状态**：显示 loading 状态，用户知道数据正在加载
2. **友好提示**：未选择业务分组时提示用户
3. **自动排序**：标签和主机名按字母顺序排序，方便查找
4. **懒加载**：只在用户点击下拉菜单时才加载数据，提高性能
5. **支持搜索**：allow-search 属性支持在下拉菜单中搜索
6. **支持自定义**：allow-create 属性支持手动输入新值

## 测试建议

1. **基本功能测试**：
   - 创建巡检组，选择业务分组
   - 添加巡检项，选择"按标签匹配"
   - 点击主机标签下拉菜单，验证显示所有标签
   - 选择"按主机名匹配"
   - 点击主机名下拉菜单，验证显示所有主机名

2. **数据准确性测试**：
   - 验证标签列表包含所有业务分组下的主机标签
   - 验证主机名列表包含所有业务分组下的主机名
   - 验证数据去重正确
   - 验证数据排序正确

3. **边界情况测试**：
   - 未选择业务分组时，提示用户
   - 业务分组下没有主机时，下拉菜单为空
   - 主机没有标签时，标签列表为空
   - 切换业务分组后，标签和主机名列表更新

4. **性能测试**：
   - 业务分组包含大量主机（1000+）时，加载速度
   - 主机包含大量标签时，提取速度
   - 多次切换匹配方式时，响应速度

## 注意事项

1. 主机标签从 `tags` 字段提取，格式为逗号分隔的字符串
2. 标签和主机名会自动去重，避免重复显示
3. 支持手动输入新值，但建议优先从下拉列表选择
4. 数据加载采用懒加载策略，只在需要时加载
5. 编辑巡检组时会自动加载数据，方便用户操作

## 修改的文件

- `web/src/views/inspection/InspectionManagement.vue`
  - 添加 `availableTags` 和 `availableHostNames` 状态变量
  - 修改 `loadHostsForItem()` 方法，提取标签和主机名
  - 更新主机标签和主机名的下拉菜单组件
  - 添加 `@focus` 和 `@change` 事件监听
  - 修改 `handleEdit()` 方法，编辑时自动加载数据
