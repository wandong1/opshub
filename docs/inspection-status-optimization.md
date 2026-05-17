# 巡检项执行状态优化说明

## 优化概述

优化了巡检项执行结果的状态逻辑，当断言失败时，将状态标记为 `failed`，方便通过状态字段直接判断巡检是否通过。

## 问题背景

### 优化前的逻辑

1. 执行成功（无错误）→ `status = "success"`
2. 执行失败（有错误）→ `status = "failed"`，`assertionResult = "skip"`
3. 断言通过 → `status = "success"`，`assertionResult = "pass"`
4. 断言失败 → `status = "success"`，`assertionResult = "fail"` ❌

**问题**：
- 断言失败时，`status` 字段仍然是 `"success"`
- 无法通过 `status` 字段直接判断巡检是否真正通过
- 前端需要同时检查 `status` 和 `assertionResult` 两个字段
- 不符合直觉：断言失败应该视为巡检失败

### 优化后的逻辑

1. 执行失败（有错误）→ `status = "failed"`，`errorMessage = "执行错误信息"`
2. 执行成功 + 无断言 → `status = "success"`
3. 执行成功 + 断言通过 → `status = "success"`，`assertionResult = "pass"`
4. 执行成功 + 断言失败 → `status = "failed"`，`assertionResult = "fail"`，`errorMessage = "断言失败: {断言详情}"` ✅

**优势**：
- ✅ 通过 `status` 字段即可判断巡检是否通过
- ✅ 符合直觉：断言失败 = 巡检失败
- ✅ 前端无需修改，现有的状态判断逻辑自动生效
- ✅ 错误信息清晰：区分执行错误和断言错误

## 技术实现

### 改动文件

**后端（1 个文件）**：
- `internal/service/inspection_mgmt/item_service.go`

### 改动内容

**位置**：`executeItem` 函数（第 630-645 行）

**改动前**：
```go
// 断言校验
assertionResult := s.validator.Validate(effectiveAssertionType, effectiveAssertionValue, execResult.Output)
if assertionResult.Pass {
    result.AssertionResult = "pass"
} else {
    result.AssertionResult = "fail"
}
result.AssertionDetails = map[string]interface{}{
    "pass":    assertionResult.Pass,
    "message": assertionResult.Message,
}

return result
```

**改动后**：
```go
// 断言校验
assertionResult := s.validator.Validate(effectiveAssertionType, effectiveAssertionValue, execResult.Output)
if assertionResult.Pass {
    result.AssertionResult = "pass"
} else {
    result.AssertionResult = "fail"
    // 断言失败时，将状态改为 failed，并设置错误信息
    result.Status = "failed"
    result.ErrorMessage = fmt.Sprintf("断言失败: %s", assertionResult.Message)
}
result.AssertionDetails = map[string]interface{}{
    "pass":    assertionResult.Pass,
    "message": assertionResult.Message,
}

return result
```

**改动说明**：
- 新增 3 行代码
- 当断言失败时，设置 `result.Status = "failed"`
- 当断言失败时，设置 `result.ErrorMessage = "断言失败: {断言详情}"`

## 影响范围

### 对现有功能的影响

#### 1. 巡检执行记录列表

**页面**：`web/src/views/inspection/InspectionRecords.vue`

**影响**：
- ✅ 断言失败的记录将显示为"失败"（红色标签）
- ✅ 错误信息显示"断言失败: xxx"
- ✅ 可以通过状态筛选断言失败的记录

**示例**：
```
状态: [失败]
错误信息: 断言失败: 实际值 85.50 >= 期望值 80.00
```

#### 2. 任务调度同步执行

**页面**：`web/src/views/inspection/TaskSchedule.vue`

**影响**：
- ✅ 断言失败的巡检项将计入失败统计
- ✅ 整体状态计算更准确（成功/部分成功/失败）
- ✅ 同步执行结果中，断言失败显示为失败状态

**示例**：
```
执行结果: 部分成功
成功: 45 项
失败: 5 项（包含 3 项断言失败）
```

#### 3. 测试执行日志

**页面**：`web/src/views/inspection/InspectionManagement.vue`

**影响**：
- ✅ 断言失败的日志将显示为"failed"状态（红色标签）
- ✅ 错误信息显示"断言失败: xxx"
- ✅ 更直观地看到哪些巡检项断言失败

**示例**：
```
[failed] CPU总使用率（%） | PromQL
主机: web-server-01 (192.168.1.10) | 100ms
输出: 85.50
错误信息: 断言失败: 实际值 85.50 >= 期望值 80.00
断言: 失败
```

#### 4. 统计数据

**影响**：
- ✅ 成功率计算更准确（断言失败不计入成功）
- ✅ 失败统计包含断言失败
- ✅ 巡检报告更真实反映系统健康状态

### 兼容性

- ✅ **向后兼容**：现有数据不受影响
- ✅ **API 兼容**：响应格式不变，仅状态值变化
- ✅ **前端兼容**：无需修改前端代码，现有逻辑自动适配
- ✅ **数据库兼容**：无需修改表结构

## 测试场景

### 场景 1：执行成功 + 无断言

**配置**：
- 执行类型：命令执行
- 命令：`echo "Hello"`
- 断言类型：无

**预期结果**：
```json
{
  "status": "success",
  "output": "Hello",
  "errorMessage": "",
  "assertionResult": "pass",
  "assertionDetails": {
    "pass": true,
    "message": "无断言规则，跳过校验"
  }
}
```

### 场景 2：执行成功 + 断言通过

**配置**：
- 执行类型：PromQL
- 查询：`node_cpu_usage`
- 断言类型：小于（lt）
- 断言阈值：80

**实际输出**：`45.23`

**预期结果**：
```json
{
  "status": "success",
  "output": "45.23",
  "errorMessage": "",
  "assertionResult": "pass",
  "assertionDetails": {
    "pass": true,
    "message": "实际值 45.23 < 期望值 80.00"
  }
}
```

### 场景 3：执行成功 + 断言失败（本次优化重点）

**配置**：
- 执行类型：PromQL
- 查询：`node_cpu_usage`
- 断言类型：小于（lt）
- 断言阈值：80

**实际输出**：`85.50`

**预期结果**：
```json
{
  "status": "failed",
  "output": "85.50",
  "errorMessage": "断言失败: 实际值 85.50 >= 期望值 80.00",
  "assertionResult": "fail",
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

**预期结果**：
```json
{
  "status": "failed",
  "output": "",
  "errorMessage": "command not found: invalid_command",
  "assertionResult": "skip",
  "assertionDetails": null
}
```

## 断言类型说明

| 断言类型 | 说明 | 失败示例 |
|---------|------|---------|
| lt | 小于 | 断言失败: 实际值 85.50 >= 期望值 80.00 |
| lte | 小于等于 | 断言失败: 实际值 85.50 > 期望值 80.00 |
| gt | 大于 | 断言失败: 实际值 45.23 <= 期望值 50.00 |
| gte | 大于等于 | 断言失败: 实际值 45.23 < 期望值 50.00 |
| eq | 等于 | 断言失败: 实际值 '200' != 期望值 '404' |
| contains | 包含 | 断言失败: 输出 ✗ 包含 'success' |
| not_contains | 不包含 | 断言失败: 输出 ✓ 包含 'error' |
| regex | 正则匹配 | 断言失败: 输出 ✗ 匹配正则 '^\d+$' |

## 使用建议

### 1. 合理设置断言阈值

- CPU 使用率：建议 < 80%
- 内存使用率：建议 < 85%
- 磁盘使用率：建议 < 90%
- 响应时间：根据业务需求设置

### 2. 区分告警级别

- **巡检级别**：high（高）、medium（中）、low（低）
- **风险等级**：high（高）、medium（中）、low（低）
- 高风险巡检项断言失败应及时处理

### 3. 监控断言失败趋势

- 定期查看巡检执行记录
- 关注断言失败率变化
- 及时调整阈值或优化系统

## 注意事项

1. **断言失败不影响后续巡检项执行**
   - 单个巡检项断言失败不会中断整个巡检任务
   - 所有巡检项都会执行完毕

2. **错误信息格式统一**
   - 执行错误：直接显示错误信息
   - 断言失败：前缀 "断言失败: " + 断言详情

3. **状态字段优先级**
   - 执行错误 > 断言失败 > 成功
   - 即使断言失败，如果执行本身有错误，状态仍为执行错误

4. **历史数据不受影响**
   - 优化仅影响新执行的巡检记录
   - 历史记录保持原有状态

## 回滚方案

如需回滚，只需删除新增的 3 行代码：

```go
// 删除这 3 行
result.Status = "failed"
result.ErrorMessage = fmt.Sprintf("断言失败: %s", assertionResult.Message)
```

恢复为原逻辑即可。

## 总结

本次优化通过最小改动（3 行代码），实现了断言失败时状态自动标记为失败的功能，使得：

1. ✅ 状态字段更准确反映巡检结果
2. ✅ 前端无需修改，自动适配
3. ✅ 错误信息更清晰
4. ✅ 统计数据更真实

符合"断言失败 = 巡检失败"的直觉逻辑，提升了系统的易用性和准确性。
