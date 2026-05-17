# 断言跳过状态修复说明

## 修复概述

修复了巡检项执行时，无断言规则的情况下，断言结果显示为 `"pass"`（通过）的问题，改为正确显示 `"skip"`（跳过）。

## 问题背景

### 修复前的逻辑

当巡检项未配置断言规则时：
- 断言验证器返回 `Pass: true`
- 执行结果中 `AssertionResult = "pass"` ❌
- 用户误以为配置了断言并且通过了

**问题示例**：
```json
{
  "status": "success",
  "assertionResult": "pass",  ❌ 应该是 "skip"
  "assertionDetails": {
    "pass": true,
    "message": "无断言规则，跳过校验"
  }
}
```

### 修复后的逻辑

当巡检项未配置断言规则时：
- 断言验证器返回 `Pass: true, Skip: true`
- 执行结果中 `AssertionResult = "skip"` ✅
- 明确表示跳过了断言校验

**修复后示例**：
```json
{
  "status": "success",
  "assertionResult": "skip",  ✅ 正确
  "assertionDetails": {
    "pass": true,
    "message": "无断言规则，跳过校验"
  }
}
```

## 技术实现

### 方案选择

采用**方案一**：修改断言验证器，增加 `Skip` 字段

**优点**：
- ✅ 逻辑清晰，明确区分"跳过"和"通过"
- ✅ 扩展性好，未来可能有其他跳过场景
- ✅ 语义准确，`Skip: true` 明确表示跳过

### 改动文件清单

#### 1. 核心断言验证器

**文件**：`internal/biz/inspection_mgmt/assertion_validator.go`

**改动 1**：修改 `AssertionResult` 结构体
```go
// 修复前
type AssertionResult struct {
	Pass    bool   `json:"pass"`
	Message string `json:"message"`
}

// 修复后
type AssertionResult struct {
	Pass    bool   `json:"pass"`
	Message string `json:"message"`
	Skip    bool   `json:"skip"` // 标识是否跳过断言（无断言规则时为 true）
}
```

**改动 2**：修改 `Validate` 方法
```go
// 修复前
if assertionType == "" {
	return &AssertionResult{Pass: true, Message: "无断言规则，跳过校验"}
}

// 修复后
if assertionType == "" {
	return &AssertionResult{Pass: true, Message: "无断言规则，跳过校验", Skip: true}
}
```

#### 2. 巡检项服务

**文件**：`internal/service/inspection_mgmt/item_service.go`

**改动**：修改 `executeItem` 函数中的断言结果处理逻辑
```go
// 修复前
assertionResult := s.validator.Validate(effectiveAssertionType, effectiveAssertionValue, execResult.Output)
if assertionResult.Pass {
	result.AssertionResult = "pass"
} else {
	result.AssertionResult = "fail"
	result.Status = "failed"
	result.ErrorMessage = fmt.Sprintf("断言失败: %s", assertionResult.Message)
}

// 修复后
assertionResult := s.validator.Validate(effectiveAssertionType, effectiveAssertionValue, execResult.Output)
if assertionResult.Skip {
	// 无断言规则，跳过校验
	result.AssertionResult = "skip"
} else if assertionResult.Pass {
	// 断言通过
	result.AssertionResult = "pass"
} else {
	// 断言失败
	result.AssertionResult = "fail"
	result.Status = "failed"
	result.ErrorMessage = fmt.Sprintf("断言失败: %s", assertionResult.Message)
}
```

#### 3. 插件断言验证器

**文件**：`plugins/inspection/executor/assertion_validator.go`

**改动 1**：修改 `AssertionResult` 结构体（同核心验证器）

**改动 2**：修改 `Validate` 方法（同核心验证器）

#### 4. 插件巡检项服务

**文件**：`plugins/inspection/service/item_service.go`

**改动位置 1**：命令/脚本执行的断言处理（第 224-238 行）
```go
// 修复后
assertionResult := s.validator.Validate(item.AssertionType, item.AssertionValue, execResult.Output)
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

**改动位置 2**：PromQL 执行的断言处理（第 278-297 行）
```go
// 修复后
if item.AssertionType != "" {
	assertionResult := s.validator.Validate(item.AssertionType, item.AssertionValue, promqlResult.PromQLResult)
	if assertionResult.Skip {
		result.AssertionResult = "skip"
	} else if assertionResult.Pass {
		result.AssertionResult = "pass"
	} else {
		result.AssertionResult = "fail"
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("断言失败: %s", assertionResult.Message)
	}
	// ...
} else {
	result.AssertionResult = "skip"
}
```

## 断言结果状态说明

### 三种状态

| 状态 | 说明 | 场景 |
|-----|------|------|
| `skip` | 跳过 | 未配置断言规则 |
| `pass` | 通过 | 配置了断言规则且断言通过 |
| `fail` | 失败 | 配置了断言规则但断言失败 |

### 状态优先级

1. **执行失败** → `status = "failed"`, `assertionResult = "skip"`
2. **执行成功 + 无断言** → `status = "success"`, `assertionResult = "skip"`
3. **执行成功 + 断言通过** → `status = "success"`, `assertionResult = "pass"`
4. **执行成功 + 断言失败** → `status = "failed"`, `assertionResult = "fail"`

## 测试场景

### 场景 1：无断言规则（本次修复重点）

**配置**：
- 执行类型：命令执行
- 命令：`echo "Hello"`
- 断言类型：无（空）

**修复前**：
```json
{
  "status": "success",
  "output": "Hello",
  "assertionResult": "pass",  ❌
  "assertionDetails": {
    "pass": true,
    "message": "无断言规则，跳过校验"
  }
}
```

**修复后**：
```json
{
  "status": "success",
  "output": "Hello",
  "assertionResult": "skip",  ✅
  "assertionDetails": {
    "pass": true,
    "message": "无断言规则，跳过校验"
  }
}
```

### 场景 2：有断言规则且通过

**配置**：
- 执行类型：PromQL
- 断言类型：小于（lt）
- 断言阈值：80

**实际输出**：`45.23`

**结果**（不变）：
```json
{
  "status": "success",
  "output": "45.23",
  "assertionResult": "pass",  ✅
  "assertionDetails": {
    "pass": true,
    "message": "实际值 45.23 < 期望值 80.00"
  }
}
```

### 场景 3：有断言规则但失败

**配置**：
- 执行类型：PromQL
- 断言类型：小于（lt）
- 断言阈值：80

**实际输出**：`85.50`

**结果**（不变）：
```json
{
  "status": "failed",
  "output": "85.50",
  "errorMessage": "断言失败: 实际值 85.50 >= 期望值 80.00",
  "assertionResult": "fail",  ✅
  "assertionDetails": {
    "pass": false,
    "message": "实际值 85.50 >= 期望值 80.00"
  }
}
```

### 场景 4：执行失败

**配置**：
- 执行类型：命令执行
- 命令：`invalid_command`

**结果**（不变）：
```json
{
  "status": "failed",
  "output": "",
  "errorMessage": "command not found: invalid_command",
  "assertionResult": "skip",  ✅
  "assertionDetails": null
}
```

## 影响范围

### 对前端的影响

**巡检执行记录列表**：
- ✅ 无断言规则的记录，断言结果显示为"跳过"
- ✅ 有断言规则且通过的记录，断言结果显示为"通过"
- ✅ 有断言规则但失败的记录，断言结果显示为"失败"

**测试执行日志**：
- ✅ 无断言规则的日志，断言标签显示为"跳过"
- ✅ 更清晰地区分"跳过"和"通过"

**统计数据**：
- ✅ 断言通过率计算更准确（跳过不计入通过）

### 兼容性

- ✅ **向后兼容**：历史数据不受影响
- ✅ **API 兼容**：响应格式不变，仅增加 `skip` 字段
- ✅ **前端兼容**：前端已支持 `skip` 状态
- ✅ **数据库兼容**：无需修改表结构

## 前端显示建议

### 断言结果标签颜色

```typescript
const getAssertionResultColor = (result: string) => {
  const map: Record<string, string> = {
    pass: 'green',   // 通过 - 绿色
    fail: 'red',     // 失败 - 红色
    skip: 'gray'     // 跳过 - 灰色
  }
  return map[result] || 'gray'
}

const getAssertionResultText = (result: string) => {
  const map: Record<string, string> = {
    pass: '通过',
    fail: '失败',
    skip: '跳过'
  }
  return map[result] || result
}
```

### 显示示例

```vue
<a-tag :color="getAssertionResultColor(record.assertionResult)">
  断言: {{ getAssertionResultText(record.assertionResult) }}
</a-tag>
```

## 总结

本次修复通过在 `AssertionResult` 结构体中增加 `Skip` 字段，明确区分了"跳过"和"通过"两种状态，使得：

1. ✅ 无断言规则时，断言结果正确显示为 `"skip"`
2. ✅ 有断言规则且通过时，断言结果显示为 `"pass"`
3. ✅ 有断言规则但失败时，断言结果显示为 `"fail"`，状态为 `"failed"`
4. ✅ 语义更清晰，避免用户误解

修复涉及 4 个文件，共增加约 30 行代码，逻辑清晰，扩展性好。
