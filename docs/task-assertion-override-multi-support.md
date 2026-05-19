# 任务调度断言覆盖功能 - 多断言支持改造方案

## 背景

当前任务调度中的断言覆盖功能使用的是旧的单断言模式（`assertion_type` + `assertion_value`），需要改造为新的多断言模式（`assertions` + `assertion_logic`），以支持：
- 多条断言规则（最多 10 条）
- AND/OR 逻辑组合
- 拨测专用断言类型（probe_success、probe_latency_lt 等）

## 现状分析

### 后端现状

**数据模型**（`internal/data/inspection_mgmt/inspection_task.go`）：
```go
// 巡检项断言覆盖（任务级断言覆盖功能）
// JSON 数组：[{"item_id": 123, "assertion_type": "lt", "assertion_value": "80"}]
ItemAssertionOverrides string `gorm:"type:text" json:"item_assertion_overrides"`
```

**DTO 结构**（`internal/service/inspection_mgmt/task_dto.go`）：
```go
ItemAssertionOverrides string `json:"item_assertion_overrides"` // 巡检项断言覆盖（JSON 数组）
```

**断言覆盖结构**（`internal/service/inspection_mgmt/item_service.go`）：
```go
// ItemAssertionOverride 巡检项断言覆盖结构
type ItemAssertionOverride struct {
    ItemID          uint   `json:"item_id"`
    Assertions      string `json:"assertions"`       // 断言规则列表（JSON）
    AssertionLogic  string `json:"assertion_logic"`  // 断言逻辑：and/or
}
```

**✅ 后端已支持多断言**：`ItemAssertionOverride` 结构体已经包含 `Assertions` 和 `AssertionLogic` 字段。

### 前端现状

**数据结构**（`web/src/views/inspection/TaskSchedule.vue`）：
```typescript
// 断言覆盖状态（旧的单断言模式）
const assertionOverrides = ref<Record<number, { type: string; value: string }>>({})
```

**序列化逻辑**：
```typescript
// 构建断言覆盖数组（旧格式）
const assertionOverridesArray: ItemAssertionOverride[] = []
for (const [itemIdStr, override] of Object.entries(assertionOverrides.value)) {
  if (override.type) {
    assertionOverridesArray.push({
      item_id: Number(itemIdStr),
      assertion_type: override.type,    // ❌ 旧字段
      assertion_value: override.value   // ❌ 旧字段
    })
  }
}
```

**UI 界面**（断言覆盖配置弹窗）：
- 每个巡检项只能配置一条断言
- 只支持基础断言类型（gt/lt/eq/contains/regex）
- 不支持 AND/OR 逻辑

**❌ 前端使用旧的单断言模式**，需要改造。

---

## 改造方案

### 方案概述

**核心思路**：前端改造为多断言模式，与巡检管理页面保持一致。

**改造范围**：
1. 前端数据结构：`assertionOverrides` 改为支持多断言列表
2. 前端 UI：断言配置弹窗改为支持多条断言 + AND/OR 逻辑
3. 前端序列化：生成新格式的 JSON（`assertions` + `assertion_logic`）
4. 前端反序列化：解析新格式的 JSON
5. 后端无需改动（已支持）

---

## 详细实施步骤

### 阶段 1：前端数据结构改造

**文件**：`web/src/views/inspection/TaskSchedule.vue`

**改动 1：修改数据结构**

```typescript
// 旧的单断言结构
// const assertionOverrides = ref<Record<number, { type: string; value: string }>>({})

// 新的多断言结构
interface AssertionRule {
  type: string
  value: string
  description?: string
}

interface ItemAssertionOverride {
  assertionList: AssertionRule[]
  assertionLogic: 'and' | 'or'
}

const assertionOverrides = ref<Record<number, ItemAssertionOverride>>({})
```

**改动 2：初始化逻辑**

```typescript
// 加载指定巡检组的巡检项（用于断言覆盖配置）
const loadItemsForGroup = async (groupId: number) => {
  try {
    const res = await getInspectionItems({ groupId, page: 1, pageSize: 1000 })
    itemsForOverride.value = res.list
    currentGroupName.value = inspectionGroups.value.find(g => g.id === groupId)?.name || ''
    
    // 初始化断言覆盖对象（新格式）
    for (const item of res.list) {
      if (!assertionOverrides.value[item.id]) {
        assertionOverrides.value[item.id] = {
          assertionList: [],
          assertionLogic: 'and'
        }
      }
    }
    
    assertionOverrideModalVisible.value = true
  } catch (error: any) {
    Message.error(error.message || '加载巡检项失败')
  }
}
```

---

### 阶段 2：前端 UI 改造

**文件**：`web/src/views/inspection/TaskSchedule.vue`

**改动 1：断言覆盖配置弹窗 UI**

```vue
<!-- 断言覆盖详细配置弹窗（嵌套弹窗） -->
<a-modal 
  v-model:visible="assertionOverrideModalVisible" 
  :title="`断言覆盖配置 - ${currentGroupName}`" 
  :width="1200" 
  @ok="handleSaveAssertionOverrides" 
  @cancel="assertionOverrideModalVisible = false"
>
  <a-table :data="itemsForOverride" :pagination="false" :bordered="{ cell: true }">
    <template #columns>
      <a-table-column title="巡检项" data-index="name" :width="180" />

      <a-table-column title="原始断言" :width="200">
        <template #cell="{ record }">
          <span v-if="record.assertions && record.assertions !== '[]'" style="color: var(--ops-text-secondary);">
            已配置 {{ parseAssertionCount(record.assertions) }} 条断言
          </span>
          <span v-else style="color: var(--ops-text-tertiary);">无断言</span>
        </template>
      </a-table-column>

      <a-table-column title="覆盖配置" :width="700">
        <template #cell="{ record }">
          <div class="assertion-override-config">
            <!-- 断言逻辑选择 -->
            <div v-if="assertionOverrides[record.id]?.assertionList?.length > 1" class="assertion-logic-row">
              <span class="logic-label">断言逻辑:</span>
              <a-radio-group v-model="assertionOverrides[record.id].assertionLogic" type="button" size="small">
                <a-radio value="and">AND（所有通过）</a-radio>
                <a-radio value="or">OR（任一通过）</a-radio>
              </a-radio-group>
            </div>

            <!-- 断言列表 -->
            <div class="assertion-list">
              <div 
                v-for="(assertion, index) in assertionOverrides[record.id]?.assertionList || []" 
                :key="index" 
                class="assertion-item"
              >
                <span class="assertion-index">{{ index + 1 }}.</span>
                <a-select 
                  v-model="assertion.type" 
                  placeholder="选择断言类型" 
                  size="small" 
                  style="width: 200px;"
                  @change="onAssertionTypeChange(assertion)"
                >
                  <!-- 根据巡检项执行类型显示不同的断言选项 -->
                  <template v-if="record.executionType === 'probe'">
                    <a-optgroup label="拨测专用">
                      <a-option value="probe_success">拨测是否成功</a-option>
                      <a-option value="probe_latency_lt">响应时间小于（毫秒）</a-option>
                      <a-option value="probe_assertion_all">原始断言全部通过</a-option>
                      <a-option value="probe_status_code">HTTP状态码等于</a-option>
                    </a-optgroup>
                  </template>
                  <template v-else-if="record.executionType === 'promql'">
                    <a-option value="gt">大于 (&gt;)</a-option>
                    <a-option value="gte">大于等于 (&gt;=)</a-option>
                    <a-option value="lt">小于 (&lt;)</a-option>
                    <a-option value="lte">小于等于 (&lt;=)</a-option>
                    <a-option value="eq">等于 (==)</a-option>
                    <a-option value="neq">不等于 (!=)</a-option>
                  </template>
                  <template v-else>
                    <a-option value="gt">大于 (&gt;)</a-option>
                    <a-option value="gte">大于等于 (&gt;=)</a-option>
                    <a-option value="lt">小于 (&lt;)</a-option>
                    <a-option value="lte">小于等于 (&lt;=)</a-option>
                    <a-option value="eq">等于 (==)</a-option>
                    <a-option value="contains">包含</a-option>
                    <a-option value="not_contains">不包含</a-option>
                    <a-option value="regex">正则匹配</a-option>
                    <a-option value="not_regex">反正则匹配</a-option>
                  </template>
                </a-select>
                <a-input 
                  v-model="assertion.value" 
                  :placeholder="getAssertionPlaceholder(assertion.type, record.executionType)" 
                  size="small" 
                  style="flex: 1;"
                  :disabled="isAssertionValueDisabled(assertion.type)"
                />
                <a-button 
                  type="text" 
                  status="danger" 
                  size="small" 
                  @click="removeAssertion(record.id, index)"
                >
                  <template #icon><icon-delete /></template>
                </a-button>
              </div>

              <!-- 添加断言按钮 -->
              <a-button 
                type="dashed" 
                size="small" 
                @click="addAssertion(record.id)"
                :disabled="(assertionOverrides[record.id]?.assertionList?.length || 0) >= 10"
                style="width: 100%; margin-top: 8px;"
              >
                <template #icon><icon-plus /></template>
                添加断言 ({{ assertionOverrides[record.id]?.assertionList?.length || 0 }}/10)
              </a-button>
            </div>
          </div>
        </template>
      </a-table-column>
    </template>
  </a-table>
</a-modal>
```

**改动 2：辅助方法**

```typescript
// 添加断言
const addAssertion = (itemId: number) => {
  if (!assertionOverrides.value[itemId]) {
    assertionOverrides.value[itemId] = {
      assertionList: [],
      assertionLogic: 'and'
    }
  }
  
  if (assertionOverrides.value[itemId].assertionList.length >= 10) {
    Message.warning('最多只能添加10条断言')
    return
  }
  
  assertionOverrides.value[itemId].assertionList.push({
    type: '',
    value: '',
    description: ''
  })
}

// 删除断言
const removeAssertion = (itemId: number, index: number) => {
  if (assertionOverrides.value[itemId]?.assertionList) {
    assertionOverrides.value[itemId].assertionList.splice(index, 1)
  }
}

// 断言类型变化时自动生成描述
const onAssertionTypeChange = (assertion: AssertionRule) => {
  if (assertion.type) {
    assertion.description = getAssertionTypeText(assertion.type)
  }
}

// 获取断言值输入框的占位符
const getAssertionPlaceholder = (assertionType: string, executionType: string) => {
  if (!assertionType) return '请先选择断言类型'
  
  const probeAssertionPlaceholders: Record<string, string> = {
    probe_success: '无需填写',
    probe_latency_lt: '输入毫秒数，如: 500',
    probe_assertion_all: '无需填写',
    probe_status_code: '输入状态码，如: 200'
  }
  
  if (probeAssertionPlaceholders[assertionType]) {
    return probeAssertionPlaceholders[assertionType]
  }
  
  if (executionType === 'promql') {
    return '断言值（数值）'
  }
  
  return '断言值'
}

// 判断断言值输入框是否禁用
const isAssertionValueDisabled = (assertionType: string) => {
  return assertionType === 'probe_success' || assertionType === 'probe_assertion_all'
}

// 解析断言数量（用于显示原始断言）
const parseAssertionCount = (assertions: string) => {
  try {
    const list = JSON.parse(assertions)
    return Array.isArray(list) ? list.length : 0
  } catch {
    return 0
  }
}

// 获取指定巡检组的断言覆盖数量
const getAssertionOverrideCountForGroup = (groupId: number) => {
  const items = itemsForOverride.value.filter(item => item.groupId === groupId)
  let count = 0
  for (const item of items) {
    const override = assertionOverrides.value[item.id]
    if (override?.assertionList && override.assertionList.length > 0) {
      const validAssertions = override.assertionList.filter(a => a.type)
      if (validAssertions.length > 0) {
        count++
      }
    }
  }
  return count
}
```

---

### 阶段 3：前端序列化/反序列化改造

**文件**：`web/src/views/inspection/TaskSchedule.vue`

**改动 1：序列化逻辑（保存时）**

```typescript
// 构建断言覆盖数组（新格式）
const assertionOverridesArray: any[] = []
for (const [itemIdStr, override] of Object.entries(assertionOverrides.value)) {
  if (override.assertionList && override.assertionList.length > 0) {
    // 过滤掉空的断言
    const validAssertions = override.assertionList.filter(a => a.type)
    if (validAssertions.length > 0) {
      assertionOverridesArray.push({
        item_id: Number(itemIdStr),
        assertions: JSON.stringify(validAssertions),
        assertion_logic: override.assertionLogic || 'and'
      })
    }
  }
}

requestData.item_assertion_overrides = assertionOverridesArray.length > 0 
  ? JSON.stringify(assertionOverridesArray) 
  : ''
```

**改动 2：反序列化逻辑（加载时）**

```typescript
// 解析断言覆盖（编辑任务时）
assertionOverrides.value = {}
if (task.item_assertion_overrides) {
  try {
    const overrides = JSON.parse(task.item_assertion_overrides)
    for (const o of overrides) {
      // 解析 assertions 字段（JSON 字符串）
      let assertionList: AssertionRule[] = []
      if (o.assertions) {
        try {
          assertionList = JSON.parse(o.assertions)
        } catch (e) {
          console.error('解析断言列表失败:', e)
        }
      }
      
      assertionOverrides.value[o.item_id] = {
        assertionList: assertionList,
        assertionLogic: o.assertion_logic || 'and'
      }
    }
  } catch (e) {
    console.error('解析断言覆盖失败:', e)
  }
}
```

**改动 3：重置逻辑**

```typescript
// 重置表单时清空断言覆盖
const resetForm = () => {
  // ... 其他重置逻辑
  assertionOverrides.value = {}
  itemsForOverride.value = []
}
```

---

### 阶段 4：样式调整

**文件**：`web/src/views/inspection/TaskSchedule.vue`

**新增样式**：

```vue
<style scoped>
/* 断言覆盖配置样式 */
.assertion-override-config {
  width: 100%;
}

.assertion-logic-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  padding: 8px 12px;
  background: #f7f8fa;
  border-radius: 4px;
}

.logic-label {
  font-weight: 500;
  color: var(--ops-text-primary);
}

.assertion-list {
  width: 100%;
}

.assertion-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  padding: 8px;
  background: #fafafa;
  border: 1px solid var(--ops-border-color);
  border-radius: 4px;
}

.assertion-index {
  font-weight: 500;
  color: var(--ops-text-primary);
  min-width: 24px;
}
</style>
```

---

## 数据格式对比

### 旧格式（单断言）

**前端数据结构**：
```typescript
assertionOverrides = {
  123: { type: 'lt', value: '80' },
  456: { type: 'contains', value: 'success' }
}
```

**序列化后的 JSON**：
```json
[
  {
    "item_id": 123,
    "assertion_type": "lt",
    "assertion_value": "80"
  },
  {
    "item_id": 456,
    "assertion_type": "contains",
    "assertion_value": "success"
  }
]
```

### 新格式（多断言）

**前端数据结构**：
```typescript
assertionOverrides = {
  123: {
    assertionList: [
      { type: 'lt', value: '80', description: '小于' },
      { type: 'gt', value: '50', description: '大于' }
    ],
    assertionLogic: 'and'
  },
  456: {
    assertionList: [
      { type: 'probe_success', value: '', description: '拨测是否成功' },
      { type: 'probe_latency_lt', value: '500', description: '响应时间小于' }
    ],
    assertionLogic: 'or'
  }
}
```

**序列化后的 JSON**：
```json
[
  {
    "item_id": 123,
    "assertions": "[{\"type\":\"lt\",\"value\":\"80\"},{\"type\":\"gt\",\"value\":\"50\"}]",
    "assertion_logic": "and"
  },
  {
    "item_id": 456,
    "assertions": "[{\"type\":\"probe_success\",\"value\":\"\"},{\"type\":\"probe_latency_lt\",\"value\":\"500\"}]",
    "assertion_logic": "or"
  }
]
```

---

## 后端兼容性说明

**后端已完全支持新格式**，无需改动：

1. **数据模型**：`ItemAssertionOverride` 结构体已包含 `Assertions` 和 `AssertionLogic` 字段
2. **执行逻辑**：`inspection_executor.go` 中已正确解析和传递断言覆盖
3. **断言验证**：`item_service.go` 中的 `executeItem` 方法已支持多断言验证

**向后兼容性**：
- 如果前端发送旧格式（`assertion_type` + `assertion_value`），后端会忽略（因为字段不匹配）
- 建议前端改造完成后，清理旧数据或提供迁移脚本

---

## 测试验证

### 测试场景 1：单条断言覆盖

**配置**：
- 巡检项 ID: 123
- 断言类型: `lt`（小于）
- 断言值: `80`

**预期结果**：
- 前端序列化为：`[{"item_id":123,"assertions":"[{\"type\":\"lt\",\"value\":\"80\"}]","assertion_logic":"and"}]`
- 后端正确解析并应用断言覆盖
- 执行结果中断言验证生效

### 测试场景 2：多条断言覆盖（AND 逻辑）

**配置**：
- 巡检项 ID: 456
- 断言 1: `gt` > `50`
- 断言 2: `lt` < `80`
- 断言逻辑: `and`

**预期结果**：
- 前端序列化为：`[{"item_id":456,"assertions":"[{\"type\":\"gt\",\"value\":\"50\"},{\"type\":\"lt\",\"value\":\"80\"}]","assertion_logic":"and"}]`
- 后端验证：只有当结果 > 50 且 < 80 时才通过

### 测试场景 3：拨测专用断言

**配置**：
- 巡检项 ID: 789（拨测类型）
- 断言 1: `probe_success`（拨测是否成功）
- 断言 2: `probe_latency_lt` < `500`
- 断言逻辑: `or`

**预期结果**：
- 前端序列化为：`[{"item_id":789,"assertions":"[{\"type\":\"probe_success\",\"value\":\"\"},{\"type\":\"probe_latency_lt\",\"value\":\"500\"}]","assertion_logic":"or"}]`
- 后端验证：拨测成功 或 响应时间 < 500ms 即通过

### 测试场景 4：编辑已有任务

**操作**：
1. 创建任务并配置断言覆盖
2. 保存任务
3. 重新编辑任务

**预期结果**：
- 断言覆盖配置正确回显
- 断言列表、断言逻辑正确显示
- 修改后保存成功

---

## 实施计划

### 阶段 1：前端数据结构改造（1-2 小时）
- 修改 `assertionOverrides` 数据结构
- 修改初始化逻辑
- 修改重置逻辑

### 阶段 2：前端 UI 改造（2-3 小时）
- 重构断言覆盖配置弹窗
- 实现添加/删除断言功能
- 实现断言逻辑选择器
- 添加样式

### 阶段 3：前端序列化/反序列化改造（1-2 小时）
- 修改保存时的序列化逻辑
- 修改加载时的反序列化逻辑
- 修改复制任务时的逻辑

### 阶段 4：测试与验证（1-2 小时）
- 单元测试
- 集成测试
- 端到端测试

**总计工作量**：5-9 小时

---

## 风险与注意事项

### 风险 1：数据兼容性

**风险**：已有任务使用旧格式的断言覆盖，改造后无法正确解析

**缓解措施**：
- 前端加载时兼容旧格式（检测 `assertion_type` 字段）
- 提供数据迁移脚本（可选）
- 或者在编辑旧任务时提示用户重新配置断言覆盖

### 风险 2：UI 复杂度

**风险**：多断言配置界面复杂，用户体验下降

**缓解措施**：
- 提供清晰的操作提示
- 限制断言数量（最多 10 条）
- 提供默认值和占位符
- 参考巡检管理页面的成熟 UI

### 风险 3：性能影响

**风险**：多条断言验证可能影响执行性能

**缓解措施**：
- 后端已优化（并发执行、超时控制）
- 前端限制断言数量
- 合理使用 AND/OR 逻辑

---

## 总结

**核心改动**：
- ✅ 后端已支持多断言，无需改动
- ❌ 前端使用旧的单断言模式，需要改造

**改造重点**：
1. 前端数据结构：`{ type, value }` → `{ assertionList, assertionLogic }`
2. 前端 UI：单行配置 → 多行列表 + 逻辑选择器
3. 前端序列化：`assertion_type/value` → `assertions/assertion_logic`

**预期收益**：
- 支持多条断言规则，提升灵活性
- 支持 AND/OR 逻辑，满足复杂场景
- 支持拨测专用断言，功能更完善
- 与巡检管理页面保持一致，降低学习成本

**建议**：
- 优先实施，解决功能不一致问题
- 充分测试，确保向后兼容
- 提供用户文档，说明新功能
