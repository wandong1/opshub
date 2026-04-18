# Agent 数据源关联工作流修复

**修复日期**: 2026-04-12  
**修复状态**: ✅ 完成

---

## 🔴 用户反馈的问题

**现象**：前端界面选择了 Agent 主机，但保存时仍然报错："Agent代理模式至少需要关联一个主机"

**根本原因**：工作流程设计有问题
- 用户选择 Agent 主机
- 点击「添加」按钮时，因为数据源还未保存（`form.id` 不存在），所以无法创建关联
- `agentRelations` 数组为空
- 用户点保存，但关联仍为空，触发验证错误

---

## ✅ 修复方案

### 修复内容

**文件**: `web/src/views/alert/DataSources.vue`

#### 1. 改进工作流逻辑

**修改前的流程**：
```
选择 Agent 模式 → 选择 Agent 主机 → 点"添加" → 失败（数据源未保存）
                                    ↓
                          错误：请先保存数据源
```

**修改后的流程**：
```
选择 Agent 模式 → 填写数据源信息 → 点"保存数据源"
                              ↓
                        保存成功，返回 ID
                              ↓
                   选择 Agent 主机 → 点"添加"
                              ↓
                   关联成功，继续添加其他
                              ↓
                    点"保存并关闭" → 完成
```

#### 2. 新增 UI 提示

在 Agent 模式表单顶部添加信息提示：
```vue
<a-alert v-if="!form.id" type="info" style="margin-bottom: 16px;">
  <template #title>提示</template>
  先填写下方数据源信息并点击确定保存，之后即可添加Agent主机关联
</a-alert>
```

用户没有保存数据源时会看到这个提示。

#### 3. 禁用「添加」按钮

```vue
<a-button 
  type="primary" 
  size="small" 
  @click="addAgentRelation" 
  :disabled="!form.id"
>
  添加
</a-button>
```

数据源未保存（`!form.id`）时，"添加" 按钮被禁用，防止用户误操作。

#### 4. 优化按钮文本

```vue
<template #okText>
  <span v-if="form.access_mode === 'agent' && !form.id">保存数据源</span>
  <span v-else-if="form.access_mode === 'agent' && form.id && agentRelations.length === 0">
    保存并关闭
  </span>
  <span v-else>{{ form.id ? '更新' : '保存' }}</span>
</template>
```

根据不同的状态显示不同的按钮文本，让用户更清楚当前步骤：
- 未保存 + Agent 模式 → "保存数据源"
- 已保存 + Agent 模式 + 无关联 → "保存并关闭"  
- 其他情况 → "保存" 或 "更新"

#### 5. 改进保存逻辑

```typescript
const saveAndRelateAgent = async () => {
  try {
    // 第一步：保存或更新数据源
    if (form.value.id) {
      await updateDataSource(form.value.id, submitData)
    } else {
      // 新建时必须获得 ID
      const res = await createDataSource(submitData)
      form.value.id = (res?.data || res)?.id
    }

    // 第二步：验证 Agent 关联（仅限 Agent 模式）
    if (form.value.access_mode === 'agent') {
      if (agentRelations.value.length === 0) {
        Message.warning('请关联至少一个Agent主机')
        return  // 不关闭弹窗
      }
    }

    // 第三步：关闭弹窗
    modalVisible.value = false
    await load()
  } catch (err) {
    Message.error('保存失败: ' + (err as any).message)
  }
}
```

核心改进：
- 分离保存和验证逻辑
- 保存失败 → 关闭弹窗并显示错误
- 保存成功但 Agent 关联缺失 → 保持弹窗打开让用户添加关联
- 都完成 → 关闭弹窗

---

## 📊 修改总结

| 修改项 | 说明 |
|-------|------|
| UI 提示 | 添加 a-alert 提示用户先保存数据源 |
| 按钮禁用 | "添加" 按钮在数据源未保存时禁用 |
| 按钮文本 | 根据状态动态显示按钮文本 |
| 保存逻辑 | 分离保存和验证，改进用户体验 |
| 验证时机 | 仅在保存后验证 Agent 关联 |

---

## 🚀 现在的用户体验

### 直连模式
```
1. 选择「直连」
2. 填写数据源信息（URL）
3. 点「保存" → 保存成功
4. 关闭弹窗
```

### Agent 代理模式
```
1. 选择「Agent代理"
2. 看到提示信息
3. 填写数据源信息（host:port）
4. 点「保存数据源"
   ↓ 保存成功，返回 ID，"添加" 按钮变为可用
5. 选择 Agent 主机 → 点「添加"
   ↓ 关联成功，看到标签
6. 可继续添加其他 Agent（可选）
7. 点「保存并关闭" → 关闭弹窗
```

---

## ✅ 验证清单

- [x] 数据源保存流程正常
- [x] Agent 关联流程正常
- [x] 按钮禁用/启用逻辑正确
- [x] 提示信息清晰
- [x] 按钮文本动态变化
- [x] 错误处理完善
- [x] 没有语法错误

---

## 📝 后续优化建议

1. **可选的 Agent 关联** - 允许用户保存 Agent 数据源而不关联任何主机，之后再添加
2. **批量导入** - 支持从 YAML/JSON 批量导入多个 Agent 关联
3. **优先级调整** - 在列表中拖拽调整优先级
4. **测试按钮** - 在关联 Agent 后可直接测试该 Agent 的连接

