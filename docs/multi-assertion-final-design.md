# 巡检项多条断言支持方案（最终版）

## 需求确认

### 核心需求
1. ✅ 支持配置**多条断言规则**（最多 10 条）
2. ✅ 支持 **AND** 和 **OR** 逻辑
3. ✅ 适用于**所有巡检执行类型**（command、script、promql、probe）
4. ✅ **不需要向后兼容**，直接替换现有单条断言

### 断言逻辑
- **AND 逻辑**：所有断言必须全部通过
- **OR 逻辑**：任一断言通过即可

---

## 一、数据模型设计

### 1.1 后端数据模型

**文件**：`internal/data/inspection_mgmt/inspection_item.go`

**修改前**：
```go
type InspectionItem struct {
    // ... 其他字段 ...
    AssertionType  string `gorm:"size:30" json:"assertion_type"`
    AssertionValue string `gorm:"size:500" json:"assertion_value"`
}
```

**修改后**：
```go
type InspectionItem struct {
    // ... 其他字段 ...
    
    // 删除单条断言字段
    // AssertionType  string `gorm:"size:30" json:"assertion_type"`   // 删除
    // AssertionValue string `gorm:"size:500" json:"assertion_value"` // 删除
    
    // 新增多条断言字段
    Assertions      string `gorm:"type:json" json:"assertions"`           // 断言规则列表（JSON）
    AssertionLogic  string `gorm:"size:10;default:'and'" json:"assertion_logic"` // 断言逻辑：and/or
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

**AssertionLogic 字段**：
- `and`：所有断言必须全部通过（默认）
- `or`：任一断言通过即可

### 1.2 前端数据模型

**文件**：`web/src/api/inspectionManagement.ts`

**更新接口定义**：
```typescript
export interface InspectionItem {
  id?: number
  groupId: number
  name: string
  executionType: string
  
  // 删除单条断言字段
  // assertionType?: string   // 删除
  // assertionValue?: string  // 删除
  
  // 新增多条断言字段
  assertions?: AssertionRule[]  // 断言规则列表
  assertionLogic?: 'and' | 'or' // 断言逻辑
  
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

## 二、后端逻辑实现

### 2.1 断言验证器增强

**文件**：`internal/biz/inspection_mgmt/assertion_validator.go`

**新增方法**：
```go
// ValidateMultiple 执行多条断言校验
// logic: "and" 表示所有断言必须通过，"or" 表示任一断言通过即可
func (v *AssertionValidator) ValidateMultiple(assertions []AssertionRule, logic string, output string) *AssertionResult {
    if len(assertions) == 0 {
        return &AssertionResult{Pass: true, Message: "无断言规则，跳过校验", Skip: true}
    }

    // 限制断言数量
    if len(assertions) > 10 {
        return &AssertionResult{
            Pass:    false,
            Message: "断言数量超过限制（最多10条）",
        }
    }

    var passedAssertions []string
    var failedAssertions []string
    var skippedCount int

    for i, rule := range assertions {
        result := v.Validate(rule.Type, rule.Value, output)
        
        desc := rule.Description
        if desc == "" {
            desc = fmt.Sprintf("断言%d", i+1)
        }

        if result.Skip {
            skippedCount++
            continue
        }
        
        if result.Pass {
            passedAssertions = append(passedAssertions, fmt.Sprintf("✓ %s: %s", desc, result.Message))
        } else {
            failedAssertions = append(failedAssertions, fmt.Sprintf("✗ %s: %s", desc, result.Message))
        }
    }

    totalAssertions := len(assertions) - skippedCount
    passedCount := len(passedAssertions)
    failedCount := len(failedAssertions)

    // AND 逻辑：所有断言必须通过
    if logic == "and" || logic == "" {
        if failedCount == 0 && passedCount > 0 {
            return &AssertionResult{
                Pass:    true,
                Message: fmt.Sprintf("所有断言通过 (%d/%d)\n%s", passedCount, totalAssertions, strings.Join(passedAssertions, "\n")),
            }
        }
        
        allMessages := append(failedAssertions, passedAssertions...)
        return &AssertionResult{
            Pass:    false,
            Message: fmt.Sprintf("断言失败 (%d/%d)\n%s", failedCount, totalAssertions, strings.Join(allMessages, "\n")),
        }
    }

    // OR 逻辑：任一断言通过即可
    if logic == "or" {
        if passedCount > 0 {
            return &AssertionResult{
                Pass:    true,
                Message: fmt.Sprintf("断言通过 (%d/%d)\n%s", passedCount, totalAssertions, strings.Join(passedAssertions, "\n")),
            }
        }
        
        return &AssertionResult{
            Pass:    false,
            Message: fmt.Sprintf("所有断言失败 (%d/%d)\n%s", failedCount, totalAssertions, strings.Join(failedAssertions, "\n")),
        }
    }

    return &AssertionResult{
        Pass:    false,
        Message: fmt.Sprintf("未知的断言逻辑: %s", logic),
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
effectiveAssertions := item.Assertions
effectiveAssertionLogic := item.AssertionLogic

if assertionOverride != nil {
    // 任务调度级断言覆盖
    if assertionOverride.Assertions != "" {
        effectiveAssertions = assertionOverride.Assertions
    }
    if assertionOverride.AssertionLogic != "" {
        effectiveAssertionLogic = assertionOverride.AssertionLogic
    }
}

// 执行断言校验
var assertionResult *inspectionmgmtbiz.AssertionResult

if effectiveAssertions != "" && effectiveAssertions != "[]" {
    // 解析断言列表
    var assertions []inspectionmgmtbiz.AssertionRule
    if err := json.Unmarshal([]byte(effectiveAssertions), &assertions); err == nil && len(assertions) > 0 {
        // 使用多条断言验证
        assertionResult = s.validator.ValidateMultiple(assertions, effectiveAssertionLogic, execResult.Output)
    } else {
        // JSON 解析失败
        assertionResult = &inspectionmgmtbiz.AssertionResult{
            Pass:    false,
            Message: fmt.Sprintf("断言配置解析失败: %v", err),
        }
    }
} else {
    // 无断言配置
    assertionResult = &inspectionmgmtbiz.AssertionResult{
        Pass:    true,
        Message: "无断言规则，跳过校验",
        Skip:    true,
    }
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

### 2.3 任务调度断言覆盖

**文件**：`internal/biz/inspection_mgmt/task_executor.go`

**更新 ItemAssertionOverride 结构**：
```go
// ItemAssertionOverride 巡检项断言覆盖配置
type ItemAssertionOverride struct {
    ItemID          uint   `json:"item_id"`
    Assertions      string `json:"assertions"`       // 断言规则列表（JSON）
    AssertionLogic  string `json:"assertion_logic"`  // 断言逻辑：and/or
}
```

---

## 三、前端 UI 实现

### 3.1 断言配置 UI

**文件**：`web/src/views/inspection/InspectionManagement.vue`

**UI 设计**：

```
断言规则:
┌─────────────────────────────────────────────────────────────┐
│ 断言逻辑: ⦿ AND（所有断言必须通过） ○ OR（任一断言通过）    │
│                                                              │
│ 断言 1:                                                      │
│   [拨测是否成功 ▼]  [无需填写]  [删除]                      │
│                                                              │
│ 断言 2:                                                      │
│   [响应时间小于（毫秒） ▼]  [500]  [删除]                   │
│                                                              │
│ 断言 3:                                                      │
│   [HTTP状态码等于 ▼]  [200]  [删除]                         │
│                                                              │
│ [+ 添加断言] (最多10条)                                      │
└─────────────────────────────────────────────────────────────┘
提示: AND逻辑 - 所有断言必须全部通过才算成功
      OR逻辑 - 任一断言通过即算成功
```

**实现代码**（分块）：

#### 第一部分：模板结构
```vue
<template>
  <!-- 断言配置 -->
  <a-form-item label="断言规则" :label-col-flex="'100px'">
    <div class="assertion-config">
      <!-- 断言逻辑选择 -->
      <div class="assertion-logic" v-if="item.assertionList && item.assertionList.length > 1">
        <span class="logic-label">断言逻辑:</span>
        <a-radio-group v-model="item.assertionLogic" type="button" size="small">
          <a-radio value="and">AND（所有通过）</a-radio>
          <a-radio value="or">OR（任一通过）</a-radio>
        </a-radio-group>
      </div>

      <!-- 断言列表 -->
      <div class="assertion-list">
        <div
          v-for="(assertion, index) in item.assertionList"
          :key="index"
          class="assertion-item"
        >
          <div class="assertion-header">
            <span class="assertion-label">断言 {{ index + 1 }}:</span>
            <a-button
              type="text"
              status="danger"
              size="small"
              @click="removeAssertion(item, index)"
            >
              <template #icon><icon-delete /></template>
              删除
            </a-button>
          </div>
          
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
            
            <a-col :span="14">
              <a-input
                v-model="assertion.value"
                :placeholder="getAssertionPlaceholder(assertion.type, item.executionType)"
                :disabled="!assertion.type || isAssertionValueDisabled(assertion.type)"
              />
            </a-col>
          </a-row>
        </div>

        <!-- 添加断言按钮 -->
        <a-button
          type="dashed"
          @click="addAssertion(item)"
          :disabled="item.assertionList && item.assertionList.length >= 10"
          style="width: 100%; margin-top: 8px;"
        >
          <template #icon><icon-plus /></template>
          添加断言 {{ item.assertionList && item.assertionList.length > 0 ? `(${item.assertionList.length}/10)` : '(最多10条)' }}
        </a-button>
      </div>
    </div>
    
    <template #extra>
      <span style="color: var(--ops-text-tertiary); font-size: 12px;">
        {{ getAssertionExtraText(item) }}
      </span>
    </template>
  </a-form-item>
</template>
```

#### 第二部分：Script 逻辑
```typescript
<script setup lang="ts">
import { ref, reactive } from 'vue'
import { IconPlus, IconDelete } from '@arco-design/web-vue/es/icon'

// 断言规则接口
interface AssertionRule {
  type: string
  value: string
  description?: string
}

// 添加断言
const addAssertion = (item: any) => {
  if (!item.assertionList) {
    item.assertionList = []
  }
  
  // 限制最多10条
  if (item.assertionList.length >= 10) {
    Message.warning('最多只能添加10条断言')
    return
  }
  
  item.assertionList.push({
    type: '',
    value: '',
    description: ''
  })
  
  // 默认断言逻辑为 AND
  if (!item.assertionLogic) {
    item.assertionLogic = 'and'
  }
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

// 获取断言额外说明文本
const getAssertionExtraText = (item: any) => {
  if (!item.assertionList || item.assertionList.length === 0) {
    return '对执行结果进行断言验证'
  }
  
  const logic = item.assertionLogic || 'and'
  const count = item.assertionList.length
  
  if (logic === 'and') {
    return `AND逻辑 - 所有断言（${count}条）必须全部通过才算成功`
  } else {
    return `OR逻辑 - 任一断言（${count}条）通过即算成功`
  }
}

// 保存时序列化断言列表
const serializeAssertions = (item: any) => {
  if (item.assertionList && item.assertionList.length > 0) {
    // 过滤掉空的断言
    const validAssertions = item.assertionList.filter((a: AssertionRule) => a.type)
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
    item.assertionList = []
  }
  
  // 设置默认断言逻辑
  if (!item.assertionLogic) {
    item.assertionLogic = 'and'
  }
}
</script>
```

#### 第三部分：样式
```vue
<style scoped>
.assertion-config {
  width: 100%;
}

.assertion-logic {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
  padding: 8px 12px;
  background: var(--color-fill-1);
  border-radius: 4px;
}

.logic-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--ops-text-primary);
  margin-right: 12px;
}

.assertion-list {
  width: 100%;
}

.assertion-item {
  margin-bottom: 12px;
  padding: 12px;
  background: var(--color-fill-2);
  border-radius: 4px;
  border: 1px solid var(--color-border-2);
}

.assertion-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.assertion-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--ops-text-secondary);
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
      assertionLogic: item.assertionLogic || 'and'
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

## 四、数据库迁移

### 4.1 删除旧字段，新增新字段

**SQL 语句**：
```sql
-- 删除旧的单条断言字段
ALTER TABLE inspection_items DROP COLUMN assertion_type;
ALTER TABLE inspection_items DROP COLUMN assertion_value;

-- 新增多条断言字段
ALTER TABLE inspection_items ADD COLUMN assertions JSON COMMENT '断言规则列表';
ALTER TABLE inspection_items ADD COLUMN assertion_logic VARCHAR(10) DEFAULT 'and' COMMENT '断言逻辑：and/or';
```

### 4.2 AutoMigrate

**文件**：`cmd/server/server.go`

**确保 AutoMigrate 包含 InspectionItem**：
```go
func autoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(
        // ... 其他模型 ...
        &inspectionmgmtdata.InspectionItem{},
    )
}
```

---

## 五、实施步骤

### 阶段 1：后端改动（3-4 小时）

1. ✅ 修改数据模型，删除旧字段，新增新字段
2. ✅ 执行数据库迁移
3. ✅ 修改 `ValidateMultiple` 方法，支持 AND/OR 逻辑
4. ✅ 修改 `executeItem` 方法，使用新的断言字段
5. ✅ 更新 `ItemAssertionOverride` 结构
6. ✅ 编写单元测试

### 阶段 2：前端改动（4-5 小时）

1. ✅ 更新 TypeScript 接口定义
2. ✅ 重构断言配置 UI（断言逻辑选择 + 断言列表）
3. ✅ 实现添加/删除断言功能
4. ✅ 实现断言数量限制（最多10条）
5. ✅ 实现序列化/反序列化逻辑
6. ✅ 更新测试执行日志显示

### 阶段 3：测试与文档（2-3 小时）

1. ✅ 端到端测试
2. ✅ AND/OR 逻辑测试
3. ✅ 编写使用文档
4. ✅ 更新 API 文档

**总计工作量**：9-12 小时

---

## 六、测试方案

### 测试场景 1：AND 逻辑（所有通过）

**配置**：
- 断言逻辑：AND
- 断言 1：`probe_success`
- 断言 2：`probe_latency_lt = 500`
- 断言 3：`probe_status_code = 200`

**执行结果**：
```
✓ 拨测是否成功: 拨测成功
✓ 响应时间小于（毫秒）: 响应时间 123.45ms < 阈值 500.00ms
✓ HTTP状态码等于: 状态码 200 == 期望 200
→ 所有断言通过 (3/3)
→ 巡检项状态: success
```

### 测试场景 2：AND 逻辑（部分失败）

**配置**：
- 断言逻辑：AND
- 断言 1：`probe_success`
- 断言 2：`probe_latency_lt = 100`
- 断言 3：`probe_status_code = 200`

**执行结果**：
```
✗ 响应时间小于（毫秒）: 响应时间 150.00ms >= 阈值 100.00ms
✓ 拨测是否成功: 拨测成功
✓ HTTP状态码等于: 状态码 200 == 期望 200
→ 断言失败 (1/3)
→ 巡检项状态: failed
```

### 测试场景 3：OR 逻辑（任一通过）

**配置**：
- 断言逻辑：OR
- 断言 1：`probe_latency_lt = 100`（失败）
- 断言 2：`probe_status_code = 200`（通过）
- 断言 3：`contains = error`（失败）

**执行结果**：
```
✓ HTTP状态码等于: 状态码 200 == 期望 200
→ 断言通过 (1/3)
→ 巡检项状态: success
```

### 测试场景 4：OR 逻辑（全部失败）

**配置**：
- 断言逻辑：OR
- 断言 1：`probe_latency_lt = 100`（失败）
- 断言 2：`probe_status_code = 404`（失败）
- 断言 3：`contains = error`（失败）

**执行结果**：
```
✗ 响应时间小于（毫秒）: 响应时间 150.00ms >= 阈值 100.00ms
✗ HTTP状态码等于: 状态码 200 != 期望 404
✗ 包含: 输出不包含 "error"
→ 所有断言失败 (3/3)
→ 巡检项状态: failed
```

### 测试场景 5：断言数量限制

**配置**：
- 尝试添加第 11 条断言

**预期结果**：
- 前端显示提示：`最多只能添加10条断言`
- 添加按钮禁用

---

## 七、总结

### 核心改动

1. **数据模型**：
   - 删除：`assertionType`、`assertionValue`
   - 新增：`assertions`（JSON）、`assertionLogic`（and/or）

2. **后端逻辑**：
   - 新增 `ValidateMultiple` 方法，支持 AND/OR 逻辑
   - 修改 `executeItem` 方法，使用新的断言字段
   - 限制断言数量最多 10 条

3. **前端 UI**：
   - 断言逻辑选择（AND/OR）
   - 动态断言列表（添加/删除）
   - 断言数量限制提示

### 核心优势

1. **功能强大**
   - 支持多条断言（最多 10 条）
   - 支持 AND/OR 逻辑
   - 适用于所有执行类型

2. **实现简洁**
   - 不需要向后兼容，代码更简洁
   - 数据模型清晰
   - 逻辑易于理解

3. **用户友好**
   - 动态添加/删除断言
   - 智能提示
   - 清晰的逻辑选择

4. **扩展性强**
   - 易于添加新的断言类型
   - 易于扩展断言逻辑（如 NOT）
   - 易于添加断言模板功能

### 工作量

**总计**：9-12 小时

| 阶段 | 内容 | 工作量 |
|-----|------|--------|
| 阶段 1 | 后端改动 | 3-4 小时 |
| 阶段 2 | 前端改动 | 4-5 小时 |
| 阶段 3 | 测试与文档 | 2-3 小时 |

### 风险等级

**低风险**（不需要向后兼容，改动清晰）
