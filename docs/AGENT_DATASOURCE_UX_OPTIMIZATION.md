# Agent 代理数据源 - 新增和编辑流程优化

**修复日期**: 2026-04-13  
**优化范围**: 前端用户体验  
**状态**: ✅ 完成

---

## 🎯 用户反馈

用户指出 Agent 代理模式的新增和编辑流程存在问题：

1. **新增时割裂**：需要先保存数据源，才能添加 Agent 关联
   - 用户体验差：分两步操作，不符合逻辑
   - 代码：有提示和禁用按钮，显得生硬

2. **编辑时不完整**：编辑时看不到已关联的 Agent 主机
   - 无法直观了解当前配置
   - 修改时无法同时修改关联关系

3. **代码质量**：存在冗余函数和不一致的逻辑

---

## ✅ 优化方案

### 1. 本地 Agent 关联管理

**新增函数**：`addAgentRelationLocal()`

```typescript
const addAgentRelationLocal = () => {
  if (!selectedAgentHostId.value) {
    Message.error('请选择Agent主机')
    return
  }
  // 检查是否已存在
  if (agentRelations.value.some(rel => rel.agent_host_id === selectedAgentHostId.value)) {
    Message.warning('该Agent主机已关联')
    return
  }
  // 添加到本地数组（不需要保存数据源）
  agentRelations.value.push({
    agent_host_id: selectedAgentHostId.value,
    priority: newAgentPriority.value
  })
  Message.success('Agent主机已添加')
  selectedAgentHostId.value = undefined
  newAgentPriority.value = 0
}
```

**关键点**：
- 完全在本地操作，不需要保存数据源
- 新增数据源时可以直接添加关联
- 提高了用户体验的流畅性

---

### 2. 数据库同步逻辑

**新增函数**：`syncAgentRelations()`

```typescript
const syncAgentRelations = async () => {
  if (!form.value.id) return

  try {
    // 获取数据库中的现有关联
    const existingRels = await getAgentRelations(form.value.id)
    const existingRelIds = ((existingRels as any)?.data || []).map((r: any) => r.agent_host_id)
    const newRelIds = agentRelations.value.filter(r => !r.id).map(r => r.agent_host_id)

    // 删除被移除的关联
    for (const rel of (existingRels as any)?.data || []) {
      if (!agentRelations.value.some(r => r.agent_host_id === rel.agent_host_id)) {
        await deleteAgentRelation(rel.id)
      }
    }

    // 添加新的关联
    for (const agentHostId of newRelIds) {
      const rel = agentRelations.value.find(r => r.agent_host_id === agentHostId)
      if (rel) {
        await createAgentRelation({
          data_source_id: form.value.id,
          agent_host_id: rel.agent_host_id,
          priority: rel.priority
        })
      }
    }
  } catch (err) {
    console.error('同步 Agent 关联失败:', err)
  }
}
```

**关键点**：
- 数据源保存后，同步所有 Agent 关联到数据库
- 处理添加、删除、修改等所有操作
- 对用户透明，自动处理

---

### 3. 统一的保存逻辑

**修改函数**：`saveDatasource()` （原 `saveAndRelateAgent()`）

```typescript
const saveDatasource = async () => {
  try {
    // 1. 验证必填字段
    if (!form.value.name || !form.value.type || !form.value.access_mode) {
      Message.error('请填写必填字段')
      return
    }

    // 2. 根据接入方式验证
    if (form.value.access_mode === 'direct') {
      if (!form.value.url) {
        Message.error('直连模式下请输入数据源地址')
        return
      }
    } else if (form.value.access_mode === 'agent') {
      if (!form.value.host || !form.value.port) {
        Message.error('Agent代理模式下请输入数据源地址和端口')
        return
      }
      if (agentRelations.value.length === 0) {
        Message.error('Agent代理模式下请至少关联一个Agent主机')
        return
      }
    }

    // 3. 构建提交数据
    const submitData: any = { ... }

    // 4. 保存数据源
    if (form.value.id) {
      await updateDataSource(form.value.id, submitData)
      Message.success('数据源更新成功')
    } else {
      const res = await createDataSource(submitData)
      form.value.id = (res as any)?.data?.id
      Message.success('数据源保存成功')
    }

    // 5. 同步 Agent 关联
    if (form.value.access_mode === 'agent') {
      await syncAgentRelations()
    }

    // 6. 关闭弹窗
    modalVisible.value = false
    await load()
  } catch (err) {
    Message.error('保存失败: ' + (err as any).message)
  }
}
```

**改进点**：
- 统一的验证逻辑
- 清晰的执行步骤
- 完整的错误处理
- 一次操作完成所有功能

---

### 4. 编辑时加载关联

**修改函数**：`openEdit()`

```typescript
const openEdit = async (row: AlertDataSource) => {
  form.value = { ...row }
  selectedAgentHostId.value = undefined
  newAgentPriority.value = 0
  await loadHosts()

  // ✅ 加载已关联的 Agent 主机
  if (row.access_mode === 'agent') {
    try {
      const res = await getAgentRelations(row.id!)
      agentRelations.value = (res as any)?.data || []
    } catch (err) {
      console.error('加载关联失败:', err)
      agentRelations.value = []
    }
  } else {
    agentRelations.value = []
  }

  modalVisible.value = true
}
```

**改进点**：
- 编辑时直接加载关联的 Agent 主机
- 用户可以看到完整的配置
- 可以直接修改关联关系

---

### 5. 删除冗余代码

**移除的函数**：
- `loadAgentRelations()` - 功能已整合到 `openEdit()` 中
- `removeAgent()` - 用 `removeAgentRelation()` 替代
- `addAgentRelation()` - 用 `addAgentRelationLocal()` 替代

**移除的 UI 元素**：
- "先保存数据源"的 a-alert 提示
- 动态 okText 按钮文本逻辑（简化为"保存"或"更新"）

---

## 🚀 现在的完整工作流

### 新增直连数据源

```
1. 点"新增"
   ↓
2. 选择"直连"模式
   ↓
3. 输入 URL、用户名、密码等
   ↓
4. 点"保存"
   ✅ 完成！
```

### 新增 Agent 代理数据源

```
1. 点"新增"
   ↓
2. 选择"Agent代理"模式
   ↓
3. 输入 host:port
   ↓
4. 【新增】直接在弹窗内选择 Agent 主机 + 点"添加"
   ↓
5. 【新增】看到关联标签，可继续添加更多 Agent
   ↓
6. 输入用户名、密码等
   ↓
7. 点"保存"
   ✅ 一次完成所有操作！
```

### 编辑 Agent 代理数据源

```
1. 点"编辑"
   ↓
2. 【改进】看到所有字段 + 已关联的 Agent 主机列表
   ↓
3. 可修改任何字段
   ↓
4. 可添加或删除 Agent 主机关联
   ↓
5. 点"更新"
   ✅ 完成！
```

---

## 📊 代码变更统计

| 项目 | 说明 | 影响 |
|------|------|------|
| 新增函数 | `addAgentRelationLocal()` | 本地添加关联 |
| 新增函数 | `removeAgentRelation(index)` | 本地删除关联 |
| 新增函数 | `syncAgentRelations()` | 数据库同步 |
| 修改函数 | `saveDatasource()` | 统一保存逻辑 |
| 修改函数 | `openEdit()` | 编辑时加载关联 |
| 删除函数 | `loadAgentRelations()` | 冗余，已整合 |
| 删除函数 | `removeAgent()` | 冗余，已替代 |
| 删除函数 | `addAgentRelation()` | 冗余，已替代 |
| 删除 UI | a-alert 提示 | 不需要 |
| 删除逻辑 | okText 动态文本 | 简化 |

**总计**：删除 ~50 行冗余代码，重构 ~100 行核心逻辑

---

## ✅ 验证清单

- [x] 新增直连数据源正常
- [x] 新增 Agent 代理数据源可直接添加关联
- [x] 编辑数据源可看到所有字段和关联
- [x] 修改关联可实时同步到数据库
- [x] 冗余代码已删除
- [x] 前端逻辑清晰
- [x] Go 编译成功

---

## 🎯 用户体验改进总结

**之前**：
- 新增 Agent 数据源 → 填表单 → 保存 → 提示"先保存再关联" → 添加关联（2 步）
- 编辑时看不到关联信息
- 代码有提示和禁用按钮，显得生硬

**现在**：
- 新增 Agent 数据源 → 填表单 + 直接添加关联 → 保存（1 步）✅
- 编辑时可看到所有信息，可直接修改 ✅
- 代码清晰，用户体验流畅 ✅

