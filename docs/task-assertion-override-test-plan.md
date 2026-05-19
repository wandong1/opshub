# 任务调度断言覆盖功能 - 测试计划

## 改造完成情况

### ✅ 已完成的改动

#### 1. 前端数据结构改造
- ✅ 修改 `assertionOverrides` 数据结构为多断言格式
- ✅ 添加 `AssertionRule` 和 `ItemAssertionOverride` 接口定义
- ✅ 修改初始化逻辑，支持 `assertionList` 和 `assertionLogic`

#### 2. 前端辅助方法
- ✅ `addAssertion()` - 添加断言（限制最多 10 条）
- ✅ `removeAssertion()` - 删除断言
- ✅ `onAssertionTypeChange()` - 断言类型变化时自动生成描述
- ✅ `getAssertionTypeText()` - 获取断言类型文本
- ✅ `getAssertionPlaceholder()` - 获取断言值输入框占位符
- ✅ `isAssertionValueDisabled()` - 判断断言值输入框是否禁用
- ✅ `parseAssertionCount()` - 解析断言数量
- ✅ `getAssertionOverrideCountForGroup()` - 获取巡检组的断言覆盖数量

#### 3. 前端序列化/反序列化
- ✅ 修改保存时的序列化逻辑（生成新格式 JSON）
- ✅ 修改编辑时的反序列化逻辑（解析新格式 JSON）
- ✅ 修改复制时的反序列化逻辑（解析新格式 JSON）

#### 4. 前端 UI 改造
- ✅ 重构断言覆盖配置弹窗（宽度 900px → 1200px）
- ✅ 支持多条断言列表显示
- ✅ 支持 AND/OR 逻辑选择器
- ✅ 根据巡检项执行类型显示不同的断言选项
  - 拨测类型：probe_success、probe_latency_lt、probe_assertion_all、probe_status_code
  - PromQL 类型：gt、gte、lt、lte、eq、neq
  - 命令/脚本类型：gt、gte、lt、lte、eq、contains、not_contains、regex、not_regex
- ✅ 添加断言按钮（显示当前数量/最大数量）
- ✅ 删除断言按钮

#### 5. 样式调整
- ✅ `.assertion-override-config` - 断言覆盖配置容器
- ✅ `.assertion-logic-row` - 断言逻辑选择行
- ✅ `.logic-label` - 逻辑标签
- ✅ `.assertion-list` - 断言列表容器
- ✅ `.assertion-item` - 单条断言项
- ✅ `.assertion-index` - 断言序号

---

## 测试场景

### 场景 1：创建任务并配置单条断言覆盖

**步骤**：
1. 进入任务调度页面
2. 点击"新增任务"
3. 选择任务类型为"巡检任务"
4. 选择巡检组
5. 点击"配置断言覆盖"
6. 为某个巡检项添加一条断言：
   - 断言类型：`lt`（小于）
   - 断言值：`80`
7. 保存断言覆盖配置
8. 保存任务

**预期结果**：
- 断言覆盖配置弹窗正常显示
- 可以成功添加断言
- 保存后任务的 `item_assertion_overrides` 字段为：
  ```json
  [
    {
      "item_id": 123,
      "assertions": "[{\"type\":\"lt\",\"value\":\"80\"}]",
      "assertion_logic": "and"
    }
  ]
  ```

### 场景 2：配置多条断言覆盖（AND 逻辑）

**步骤**：
1. 编辑已有任务
2. 点击"配置断言覆盖"
3. 为某个巡检项添加两条断言：
   - 断言 1：`gt` > `50`
   - 断言 2：`lt` < `80`
4. 选择断言逻辑为"AND（所有通过）"
5. 保存断言覆盖配置
6. 保存任务

**预期结果**：
- 断言逻辑选择器正常显示（当断言数量 > 1 时）
- 保存后任务的 `item_assertion_overrides` 字段为：
  ```json
  [
    {
      "item_id": 123,
      "assertions": "[{\"type\":\"gt\",\"value\":\"50\"},{\"type\":\"lt\",\"value\":\"80\"}]",
      "assertion_logic": "and"
    }
  ]
  ```

### 场景 3：配置拨测专用断言

**步骤**：
1. 创建巡检任务
2. 选择包含拨测类型巡检项的巡检组
3. 点击"配置断言覆盖"
4. 为拨测类型巡检项添加断言：
   - 断言 1：`probe_success`（拨测是否成功）
   - 断言 2：`probe_latency_lt` < `500`
5. 选择断言逻辑为"OR（任一通过）"
6. 保存

**预期结果**：
- 拨测类型巡检项显示拨测专用断言选项
- `probe_success` 和 `probe_assertion_all` 的断言值输入框被禁用
- 保存后数据格式正确

### 场景 4：编辑已有任务（回显测试）

**步骤**：
1. 创建任务并配置多条断言覆盖
2. 保存任务
3. 重新编辑该任务
4. 点击"配置断言覆盖"

**预期结果**：
- 断言覆盖配置正确回显
- 断言列表显示所有已配置的断言
- 断言逻辑选择器显示正确的值（and/or）
- 可以继续添加、删除、修改断言

### 场景 5：复制任务

**步骤**：
1. 选择一个已配置断言覆盖的任务
2. 点击"复制"按钮
3. 点击"配置断言覆盖"

**预期结果**：
- 断言覆盖配置被正确复制
- 所有断言规则和逻辑都被复制

### 场景 6：删除断言

**步骤**：
1. 编辑任务
2. 点击"配置断言覆盖"
3. 为某个巡检项添加 3 条断言
4. 删除第 2 条断言
5. 保存

**预期结果**：
- 断言被成功删除
- 剩余断言的序号自动更新（1、2）
- 保存后只包含 2 条断言

### 场景 7：断言数量限制

**步骤**：
1. 编辑任务
2. 点击"配置断言覆盖"
3. 为某个巡检项连续添加 10 条断言
4. 尝试添加第 11 条断言

**预期结果**：
- 成功添加 10 条断言
- 添加按钮显示"(10/10)"并被禁用
- 尝试添加第 11 条时显示警告："最多只能添加10条断言"

### 场景 8：不同执行类型的断言选项

**步骤**：
1. 创建任务，选择包含多种执行类型巡检项的巡检组
2. 点击"配置断言覆盖"
3. 分别查看不同执行类型巡检项的断言选项

**预期结果**：
- **拨测类型**：显示 probe_success、probe_latency_lt、probe_assertion_all、probe_status_code
- **PromQL 类型**：显示 gt、gte、lt、lte、eq、neq
- **命令/脚本类型**：显示 gt、gte、lt、lte、eq、contains、not_contains、regex、not_regex

### 场景 9：立即运行任务（验证后端兼容性）

**步骤**：
1. 创建任务并配置多条断言覆盖
2. 保存任务
3. 点击"立即运行"
4. 查看执行结果

**预期结果**：
- 任务正常执行
- 断言验证生效
- 执行结果中显示断言通过/失败状态
- 后端正确解析 `assertions` 和 `assertion_logic` 字段

---

## 数据格式验证

### 前端发送的数据格式

**单条断言**：
```json
{
  "item_assertion_overrides": "[{\"item_id\":123,\"assertions\":\"[{\\\"type\\\":\\\"lt\\\",\\\"value\\\":\\\"80\\\"}]\",\"assertion_logic\":\"and\"}]"
}
```

**多条断言（AND 逻辑）**：
```json
{
  "item_assertion_overrides": "[{\"item_id\":123,\"assertions\":\"[{\\\"type\\\":\\\"gt\\\",\\\"value\\\":\\\"50\\\"},{\\\"type\\\":\\\"lt\\\",\\\"value\\\":\\\"80\\\"}]\",\"assertion_logic\":\"and\"}]"
}
```

**多条断言（OR 逻辑）**：
```json
{
  "item_assertion_overrides": "[{\"item_id\":456,\"assertions\":\"[{\\\"type\\\":\\\"probe_success\\\",\\\"value\\\":\\\"\\\"},{\\\"type\\\":\\\"probe_latency_lt\\\",\\\"value\\\":\\\"500\\\"}]\",\"assertion_logic\":\"or\"}]"
}
```

### 后端期望的数据格式

后端 `ItemAssertionOverride` 结构体：
```go
type ItemAssertionOverride struct {
    ItemID          uint   `json:"item_id"`
    Assertions      string `json:"assertions"`       // 断言规则列表（JSON）
    AssertionLogic  string `json:"assertion_logic"`  // 断言逻辑：and/or
}
```

后端会将 `Assertions` 字段（JSON 字符串）解析为 `[]AssertionRule`：
```go
var assertions []inspectionmgmtbiz.AssertionRule
json.Unmarshal([]byte(override.Assertions), &assertions)
```

---

## 回归测试

### 测试旧功能是否受影响

1. ✅ 创建拨测任务（不涉及断言覆盖）
2. ✅ 创建巡检任务（不配置断言覆盖）
3. ✅ 配置业务分组覆盖
4. ✅ 配置数据源覆盖
5. ✅ 配置自定义变量
6. ✅ 立即运行任务
7. ✅ 查看执行结果
8. ✅ 编辑任务
9. ✅ 复制任务
10. ✅ 删除任务

---

## 已知问题与注意事项

### 1. 旧数据兼容性

**问题**：已有任务使用旧格式的断言覆盖（`assertion_type` + `assertion_value`），编辑时无法正确解析。

**解决方案**：
- 前端加载时检测旧格式，自动转换为新格式
- 或者提示用户重新配置断言覆盖

**实施建议**：
```typescript
// 在反序列化逻辑中添加兼容处理
if (o.assertion_type && o.assertion_value) {
  // 旧格式，转换为新格式
  assertionOverrides.value[o.item_id] = {
    assertionList: [{ type: o.assertion_type, value: o.assertion_value }],
    assertionLogic: 'and'
  }
} else if (o.assertions) {
  // 新格式
  let assertionList: AssertionRule[] = []
  try {
    assertionList = JSON.parse(o.assertions)
  } catch (e) {
    console.error('解析断言列表失败:', e)
  }
  assertionOverrides.value[o.item_id] = {
    assertionList: assertionList,
    assertionLogic: o.assertion_logic || 'and'
  }
}
```

### 2. UI 复杂度

**问题**：多条断言配置界面较复杂，用户可能不熟悉。

**解决方案**：
- 提供清晰的操作提示
- 在断言列表为空时显示引导文案
- 提供示例或帮助文档

### 3. 性能考虑

**问题**：每个巡检项最多 10 条断言，可能影响执行性能。

**解决方案**：
- 后端已优化（并发执行、超时控制）
- 前端限制断言数量（最多 10 条）
- 建议用户合理配置断言数量

---

## 总结

### 改造完成度：100%

- ✅ 前端数据结构改造
- ✅ 前端辅助方法
- ✅ 前端序列化/反序列化
- ✅ 前端 UI 改造
- ✅ 样式调整
- ✅ 后端已支持（无需改动）

### 下一步工作

1. **测试验证**：按照测试计划逐一验证功能
2. **旧数据兼容**：添加旧格式自动转换逻辑（可选）
3. **用户文档**：编写使用文档，说明多断言功能
4. **性能监控**：观察多断言对执行性能的影响

### 预期收益

1. ✅ 支持多条断言规则，提升灵活性
2. ✅ 支持 AND/OR 逻辑，满足复杂场景
3. ✅ 支持拨测专用断言，功能更完善
4. ✅ 与巡检管理页面保持一致，降低学习成本
