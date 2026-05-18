# 巡检项多条断言支持方案

## 需求分析

### 当前问题
- 每个巡检项只能配置**一条断言规则**
- 无法实现多条件组合判断（如：拨测成功 AND 响应时间<500ms AND 状态码=200）
- 限制了复杂场景的断言能力

### 目标
- 支持配置**多条断言规则**
- 所有断言**全部通过**才算成功（AND 逻辑）
- 保持**向后兼容**，不影响现有单条断言的巡检项
- 尽可能**复用现有代码**，减少改动范围

---

## 设计方案

### 方案选择：渐进式兼容方案

**核心思路**：
1. 保留现有的 `assertionType` 和 `assertionValue` 字段（向后兼容）
2. 新增 `assertions` 字段（JSON 数组），存储多条断言
3. 优先使用 `assertions` 字段，如果为空则回退到单条断言字段
4. 前端提供动态添加/删除断言的 UI

**优势**：
- ✅ 完全向后兼容，现有巡检项无需修改
- ✅ 数据库迁移简单，只需新增一个字段
- ✅ 代码改动集中，风险可控
- ✅ 前端 UI 渐进式升级

---

## 一、数据模型改动

### 1.1 后端数据模型

**文件**：`internal/data/inspection_mgmt/inspection_item.go`

**新增字段**：
```go
type InspectionItem struct {
    ID        uint           `gorm:"primarykey" json:"id"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

    // ... 现有字段 ...

    // 单条断言（向后兼容，保留）
    AssertionType  string `gorm:"size:30" json:"assertion_type"`
    AssertionValue string `gorm:"size:500" json:"assertion_value"`

    // 多条断言（新增）
    Assertions string `gorm:"type:json" json:"assertions"` // JSON 数组，存储多条断言
}
```

**Assertions 字段格式**：
```json
[
  {
    "type": "probe_success",
    "value": "",
    "description": "拨测是否成功"
  },
  {
    "type": "probe_latency_lt",
    "value": "500",
    "description": "响应时间小于500ms"
  },
  {
    "type": "probe_status_code",
    "value": "200",
    "description": "HTTP状态码等于200"
  }
]
```

**数据结构定义**：
```go
// AssertionRule 断言规则
type AssertionRule struct {
    Type        string `json:"type"`        // 断言类型
    Value       string `json:"value"`       // 断言值
    Description string `json:"description"` // 断言描述（可选，用于前端显示）
}
```

### 1.2 前端数据模型

**文件**：`web/src/api/inspectionManagement.ts`

**更新接口定义**：
```typescript
export interface InspectionItem {
  id?: number
  groupId: number
  name: string
  executionType: string
  
  // 单条断言（向后兼容，保留）
  assertionType?: string
  assertionValue?: string
  
  // 多条断言（新增）
  assertions?: AssertionRule[]
  
  // ... 其他字段
}

// 断言规则
export interface AssertionRule {
  type: string        // 断言类型
  value: string       // 断言值
  description?: string // 断言描述
}
```

---

## 二、后端逻辑改动

### 2.1 断言验证器增强

**文件**：`internal/biz/inspection_mgmt/assertion_validator.go`

**新增方法**：
```go
// ValidateMultiple 执行多条断言校验（所有断言必须全部通过）
func (v *AssertionValidator) ValidateMultiple(assertions []AssertionRule, output string) *AssertionResult {
    if len(assertions) == 0 {
        return &AssertionResult{Pass: true, Message: "无断言规则，跳过校验", Skip: true}
    }

    var failedAssertions []string
    var passedCount int

    for i, rule := range assertions {
        result := v.Validate(rule.Type, rule.Value, output)
        
        if result.Skip {
            continue // 跳过的断言不计入失败
        }
        
        if result.Pass {
            passedCount++
        } else {
            // 记录失败的断言
            desc := rule.Description
            if desc == "" {
                desc = fmt.Sprintf("断言%d", i+1)
            }
            failedAssertions = append(failedAssertions, fmt.Sprintf("%s: %s", desc, result.Message))
        }
    }

    // 所有断言都通过才算成功
    if len(failedAssertions) == 0 {
        return &AssertionResult{
            Pass:    true,
            Message: fmt.Sprintf("所有断言通过 (%d/%d)", passedCount, len(assertions)),
        }
    }

    // 有断言失败
    return &AssertionResult{
        Pass:    false,
        Message: fmt.Sprintf("断言失败 (%d/%d):\n%s", len(failedAssertions), len(assertions), strings.Join(failedAssertions, "\n")),
    }
}

// AssertionRule 断言规则
type AssertionRule struct {
    Type        string `json:"type"`
    Value       string `json:"value"`
    Description string `json:"description"`
}
```

### 2.2 巡检项执行逻辑修改

**文件**：`internal/service/inspection_mgmt/item_service.go`

**修改 executeItem 方法**：
```go
// 应用断言覆盖（如果提供）
effectiveAssertionType := item.AssertionType
effectiveAssertionValue := item.AssertionValue
effectiveAssertions := item.Assertions // 新增

if assertionOverride != nil {
    effectiveAssertionType = assertionOverride.AssertionType
    effectiveAssertionValue = assertionOverride.AssertionValue
}

// 执行断言校验
var assertionResult *inspectionmgmtbiz.AssertionResult

// 优先使用多条断言
if effectiveAssertions != "" && effectiveAssertions != "[]" {
    var assertions []inspectionmgmtbiz.AssertionRule
    if err := json.Unmarshal([]byte(effectiveAssertions), &assertions); err == nil && len(assertions) > 0 {
        // 使用多条断言验证
        assertionResult = s.validator.ValidateMultiple(assertions, execResult.Output)
    } else {
        // JSON 解析失败，回退到单条断言
        assertionResult = s.validator.Validate(effectiveAssertionType, effectiveAssertionValue, execResult.Output)
    }
} else {
    // 使用单条断言（向后兼容）
    assertionResult = s.validator.Validate(effectiveAssertionType, effectiveAssertionValue, execResult.Output)
}

// 处理断言结果
if assertionResult.Skip {
    result.AssertionResult = "skip"
} else if assertionResult.Pass {
    result.AssertionResult = "pass"
} else {
    result.AssertionResult = "fail"
    result.Status = "failed"
    result.ErrorMessage = fmt.Sprintf("断言失败: %s", assertionResult.Message)
}
```

### 2.3 数据模型更新

**文件**：`internal/data/inspection_mgmt/inspection_item.go`

**新增字段**：
```go
type InspectionItem struct {
    // ... 现有字段 ...
    
    // 多条断言（新增）
    Assertions string `gorm:"type:json" json:"assertions"`
}
```

**数据库迁移**：
```go
// 在 AutoMigrate 中添加
db.AutoMigrate(&InspectionItem{})

// 或者手动执行 SQL
ALTER TABLE inspection_items ADD COLUMN assertions JSON COMMENT '多条断言配置';
```

---

## 三、前端 UI 改动

### 3.1 断言配置 UI 重构

**文件**：`web/src/views/inspection/InspectionManagement.vue`

**UI 设计**：

```
断言规则:
┌─────────────────────────────────────────────────────────────┐
│ 断言 1:                                                      │
│   [拨测是否成功 ▼]  [无需填写]  [删除]                      │
│                                                              │
│ 断言 2:                                                      │
│   [响应时间小于（毫秒） ▼]  [500]  [删除]                   │
│                                                              │
│ 断言 3:                                                      │
│   [HTTP状态码等于 ▼]  [200]  [删除]                         │
│                                                              │
│ [+ 添加断言]                                                 │
└─────────────────────────────────────────────────────────────┘
提示: 所有断言必须全部通过才算成功
```

**实现代码**：
```vue
<template>
  <!-- 断言配置 -->
  <a-form-item label="断言规则" :label-col-flex="'100px'">
    <div class="assertion-list">
      <!-- 多条断言列表 -->
      <div
        v-for="(assertion, index) in item.assertionList"
        :key="index"
        class="assertion-item"
      >
        <div class="assertion-label">断言 {{ index + 1 }}:</div>
        <a-row :gutter="8">
          <a-col :span="10">
            <a-select
              v-model="assertion.type"
              placeholder="选择断言类型"
              allow-clear
              @change="onAssertionTypeChange(item, index)"
            >
              <!-- 拨测类型显示拨测专用断言 -->
              <template v-if="item.executionType === 'probe'">
                <a-optgroup label="拨测专用">
                  <a-option value="probe_success">拨测是否成功</a-option>
                  <a-option value="probe_latency_lt">响应时间小于（毫秒）</a-option>
                  <a-option value="probe_assertion_all">原始断言全部通过</a-option>
                  <a-option value="probe_status_code">HTTP状态码等于</a-option>
                </a-optgroup>
              </template>
              <!-- PromQL 类型只显示数值比较 -->
              <template v-else-if="item.executionType === 'promql'">
                <a-option value="gt">大于 (&gt;)</a-option>
                <a-option value="gte">大于等于 (&gt;=)</a-option>
                <a-option value="lt">小于 (&lt;)</a-option>
                <a-option value="lte">小于等于 (&lt;=)</a-option>
                <a-option value="eq">等于 (==)</a-option>
                <a-option value="neq">不等于 (!=)</a-option>
              </template>
              <!-- 命令和脚本类型显示所有断言 -->
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
          </a-col>
          <a-col :span="11">
            <a-input
              v-model="assertion.value"
              :placeholder="getAssertionPlaceholder(assertion.type, item.executionType)"
              :disabled="!assertion.type || isAssertionValueDisabled(assertion.type)"
            />
          </a-col>
          <a-col :span="3">
            <a-button
              type="text"
              status="danger"
              @click="removeAssertion(item, index)"
            >
              删除
            </a-button>
          </a-col>
        </a-row>
      </div>

      <!-- 添加断言按钮 -->
      <a-button type="dashed" @click="addAssertion(item)" style="width: 100%; margin-top: 8px;">
        <template #icon><icon-plus /></template>
        添加断言
      </a-button>
    </div>
    <template #extra>
      <span style="color: var(--ops-text-tertiary); font-size: 12px;">
        {{ item.assertionList && item.assertionList.length > 0 ? '所有断言必须全部通过才算成功' : '对执行结果进行断言验证' }}
      </span>
    </template>
  </a-form-item>
</template>

<script setup lang="ts">
// 断言列表数据结构
interface AssertionItem {
  type: string
  value: string
  description?: string
}

// 添加断言
const addAssertion = (item: any) => {
  if (!item.assertionList) {
    item.assertionList = []
  }
  item.assertionList.push({
    type: '',
    value: '',
    description: ''
  })
}

// 删除断言
const removeAssertion = (item: any, index: number) => {
  if (item.assertionList && item.assertionList.length > index) {
    item.assertionList.splice(index, 1)
  }
}

// 断言类型变化时自动生成描述
const onAssertionTypeChange = (item: any, index: number) => {
  const assertion = item.assertionList[index]
  if (assertion && assertion.type) {
    assertion.description = getAssertionTypeText(assertion.type)
  }
}

// 保存时序列化断言列表
const serializeAssertions = (item: any) => {
  if (item.assertionList && item.assertionList.length > 0) {
    // 过滤掉空的断言
    const validAssertions = item.assertionList.filter((a: AssertionItem) => a.type)
    if (validAssertions.length > 0) {
      return JSON.stringify(validAssertions)
    }
  }
  return ''
}

// 加载时反序列化断言列表
const deserializeAssertions = (item: any) => {
  if (item.assertions && item.assertions !== '[]') {
    try {
      item.assertionList = JSON.parse(item.assertions)
    } catch (e) {
      console.error('Failed to parse assertions:', e)
      item.assertionList = []
    }
  } else {
    // 向后兼容：如果有单条断言，转换为断言列表
    if (item.assertionType) {
      item.assertionList = [{
        type: item.assertionType,
        value: item.assertionValue || '',
        description: getAssertionTypeText(item.assertionType)
      }]
    } else {
      item.assertionList = []
    }
  }
}
</script>

<style scoped>
.assertion-list {
  width: 100%;
}

.assertion-item {
  margin-bottom: 12px;
  padding: 12px;
  background: var(--color-fill-2);
  border-radius: 4px;
}

.assertion-label {
  font-size: 12px;
  color: var(--ops-text-secondary);
  margin-bottom: 8px;
}
</style>
```

### 3.2 数据处理逻辑

**保存巡检项时**：
```typescript
const handleSaveItems = async () => {
  const items = groupItems.value.map(item => {
    // 序列化断言列表
    const assertions = serializeAssertions(item)
    
    return {
      ...item,
      assertions: assertions,
      // 保留单条断言字段（向后兼容）
      assertionType: item.assertionType || '',
      assertionValue: item.assertionValue || ''
    }
  })
  
  await batchSaveInspectionItems(currentGroupId.value, items)
}
```

**加载巡检项时**：
```typescript
const loadGroupItems = async (groupId: number) => {
  const res = await getInspectionItems(groupId)
  
  // 反序列化断言列表
  res.forEach(item => {
    deserializeAssertions(item)
  })
  
  groupItems.value = res
}
```

---

## 四、向后兼容策略

### 4.1 数据兼容

**场景 1：现有巡检项（只有单条断言）**
```json
{
  "assertionType": "probe_latency_lt",
  "assertionValue": "500",
  "assertions": ""
}
```

**处理逻辑**：
- 后端：`assertions` 为空，使用 `assertionType` 和 `assertionValue`
- 前端：加载时自动转换为 `assertionList`，显示为一条断言

**场景 2：新巡检项（多条断言）**
```json
{
  "assertionType": "",
  "assertionValue": "",
  "assertions": "[{\"type\":\"probe_success\",\"value\":\"\"},{\"type\":\"probe_latency_lt\",\"value\":\"500\"}]"
}
```

**处理逻辑**：
- 后端：优先使用 `assertions` 字段
- 前端：显示为多条断言列表

### 4.2 API 兼容

**保持现有 API 不变**：
- `createInspectionItem(data)` - 接受 `assertions` 字段
- `updateInspectionItem(id, data)` - 接受 `assertions` 字段
- `batchSaveInspectionItems(groupId, items)` - 接受 `assertions` 字段

**前端自动处理**：
- 保存时：序列化 `assertionList` → `assertions`
- 加载时：反序列化 `assertions` → `assertionList`

### 4.3 UI 兼容

**渐进式升级**：
1. 第一阶段：保留单条断言 UI，新增"切换到多条断言"按钮
2. 第二阶段：默认使用多条断言 UI，自动迁移单条断言
3. 第三阶段：完全移除单条断言 UI（可选）

**推荐方案**：直接使用多条断言 UI，自动兼容单条断言

---

## 五、实施步骤

### 阶段 1：后端改动（2-3 小时）

1. ✅ 修改数据模型，新增 `assertions` 字段
2. ✅ 执行数据库迁移
3. ✅ 新增 `ValidateMultiple` 方法
4. ✅ 修改 `executeItem` 方法，支持多条断言
5. ✅ 编写单元测试

### 阶段 2：前端改动（3-4 小时）

1. ✅ 更新 TypeScript 接口定义
2. ✅ 重构断言配置 UI
3. ✅ 实现添加/删除断言功能
4. ✅ 实现序列化/反序列化逻辑
5. ✅ 测试向后兼容性

### 阶段 3：测试与文档（1-2 小时）

1. ✅ 端到端测试
2. ✅ 向后兼容性测试
3. ✅ 编写使用文档
4. ✅ 更新 API 文档

**总计工作量**：6-9 小时

---

## 六、测试方案

### 测试场景 1：单条断言（向后兼容）

**配置**：
- 断言 1：`probe_latency_lt = 500`

**预期结果**：
- 响应时间 < 500ms：断言通过 ✅
- 响应时间 >= 500ms：断言失败 ❌

### 测试场景 2：多条断言（全部通过）

**配置**：
- 断言 1：`probe_success`
- 断言 2：`probe_latency_lt = 500`
- 断言 3：`probe_status_code = 200`

**预期结果**：
- 拨测成功 AND 响应时间 < 500ms AND 状态码 = 200：断言通过 ✅
- 任一条件不满足：断言失败 ❌

### 测试场景 3：多条断言（部分失败）

**配置**：
- 断言 1：`probe_success`（通过）
- 断言 2：`probe_latency_lt = 100`（失败，实际 150ms）
- 断言 3：`probe_status_code = 200`（通过）

**预期结果**：
- 断言失败：`断言失败 (1/3): 响应时间小于（毫秒）: 响应时间 150.00ms >= 阈值 100.00ms`
- 巡检项状态：`failed`

### 测试场景 4：现有巡检项迁移

**步骤**：
1. 加载现有巡检项（单条断言）
2. 前端自动转换为断言列表
3. 添加新的断言
4. 保存

**预期结果**：
- 加载时：单条断言显示为一条断言列表
- 保存后：`assertions` 字段包含多条断言
- 执行时：所有断言都生效

---

## 七、风险与注意事项

### 风险 1：数据库迁移

**风险**：新增 `assertions` 字段可能影响现有数据

**缓解措施**：
- 字段设置为可空（NULL）
- 保留 `assertionType` 和 `assertionValue` 字段
- 后端优先使用 `assertions`，为空时回退到单条断言

### 风险 2：性能影响

**风险**：多条断言可能增加执行时间

**缓解措施**：
- 断言验证是串行的，性能影响可控
- 限制断言数量（建议最多 10 条）
- 断言验证逻辑已优化，单条断言耗时 < 1ms

### 风险 3：UI 复杂度

**风险**：多条断言 UI 可能增加配置复杂度

**缓解措施**：
- 提供清晰的 UI 指引
- 支持断言模板（预定义常用组合）
- 提供"添加断言"按钮，默认为空列表

### 风险 4：向后兼容性

**风险**：现有巡检项可能无法正常工作

**缓解措施**：
- 保留单条断言字段
- 后端自动回退逻辑
- 前端自动迁移逻辑
- 充分测试现有巡检项

---

## 八、替代方案

### 方案 A：仅前端组合（不推荐）

**思路**：前端配置多条断言，但后端仍然单条验证

**优势**：
- 后端改动最小

**劣势**：
- 前端需要多次调用后端验证
- 无法保存多条断言配置
- 无法在任务调度中复用

### 方案 B：断言表达式（复杂）

**思路**：使用表达式语言（如 JSONPath + 逻辑运算符）

**示例**：
```
$.probe_details.success == true && $.probe_details.latency_ms < 500 && $.probe_details.status_code == 200
```

**优势**：
- 灵活性最高
- 支持复杂逻辑（AND/OR/NOT）

**劣势**：
- 实现复杂度高
- 用户学习成本高
- 调试困难

### 方案 C：断言组（推荐）

**思路**：当前方案，使用 JSON 数组存储多条断言

**优势**：
- 实现简单，改动可控
- 用户友好，易于配置
- 向后兼容性好

**劣势**：
- 仅支持 AND 逻辑（所有断言必须通过）
- 不支持 OR 逻辑（满足任一断言即可）

**推荐理由**：平衡了实现复杂度和功能需求

---

## 九、总结

### 核心设计

1. **数据模型**：新增 `assertions` 字段（JSON 数组）
2. **后端逻辑**：新增 `ValidateMultiple` 方法，支持多条断言验证
3. **前端 UI**：重构为动态断言列表，支持添加/删除
4. **向后兼容**：保留单条断言字段，自动回退和迁移

### 实施优先级

**P0（必须）**：
- ✅ 后端数据模型和验证逻辑
- ✅ 前端 UI 重构
- ✅ 向后兼容处理

**P1（重要）**：
- ✅ 单元测试和集成测试
- ✅ 使用文档

**P2（可选）**：
- 断言模板功能
- 断言数量限制
- 断言顺序调整

### 预期效果

**用户价值**：
- 🎯 支持多条件组合断言
- 🚀 提高断言灵活性
- 📊 更精准的巡检结果判断

**技术价值**：
- ✅ 向后兼容，风险可控
- ✅ 代码改动集中，易于维护
- ✅ 扩展性强，支持未来需求

**工作量**：6-9 小时

**风险等级**：低（充分的向后兼容措施）
