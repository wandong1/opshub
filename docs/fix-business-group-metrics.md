# 业务分组（business_group）指标标签修复报告

## 一、问题概述

修复了 43 个 `srehub_inspect_*` 指标中 `business_group` 标签的值错误和缺失问题：

### 1.1 巡检类指标（12个）
**问题**：`business_group` 值错误，使用了巡检组名称而非业务分组名称
- 涉及指标：`srehub_inspect_task_*`（6个）和 `srehub_inspect_check_*`（6个）
- 现象：`business_group` 与 `check_group` 值完全一致
- 根本原因：代码中使用了 `group.Name`（巡检组名称），而非 `record.BusinessGroup`（业务分组名称）

### 1.2 拨测类指标（31个）
**问题**：完全缺失 `business_group` 标签
- 涉及指标：HTTP/Ping/TCP/UDP/WebSocket/Flow 所有拨测类指标
- 根本原因：代码中未传入业务分组信息到指标推送方法

## 二、修复方案

### 2.1 巡检类指标修复（简单）

**原理**：直接使用执行记录中的 `record.BusinessGroup` 字段

**修改文件**：`internal/service/inspection_mgmt/inspection_executor.go`

**修改位置**：第557行和第566行

```go
// 修复前
baseLabels := map[string]string{
    "business_group": group.Name,  // ❌ 错误：巡检组名称
}

// 修复后
baseLabels := map[string]string{
    "business_group": record.BusinessGroup,  // ✅ 正确：业务分组名称
}
```

**说明**：
- `record.BusinessGroup` 字段已经包含了经过优先级计算后的正确业务分组名称
- 优先级规则：任务级覆盖 > 组级覆盖 > 组原始配置
- 无需修改任何业务逻辑，只需修改标签填充代码

---

### 2.2 拨测类指标修复（复杂）

**原理**：
1. 优先使用调度任务中的业务分组（`InspectionTask.BusinessGroupIDs`）
2. 如果任务中没有设置，使用拨测配置中的分组（`ProbeConfig.GroupIDs`）
3. 支持多业务分组，为每个业务分组生成一份指标

**修改文件**：`internal/biz/inspection/executor.go`

#### 步骤1：修改指标推送方法签名

添加 `businessGroupNames []string` 参数：

```go
// 修改前
func (e *NetworkProbeExecutor) pushMetrics(ctx context.Context, task *ProbeTask, config *ProbeConfig, result *probers.Result)

// 修改后
func (e *NetworkProbeExecutor) pushMetrics(ctx context.Context, task *ProbeTask, config *ProbeConfig, result *probers.Result, businessGroupNames []string)
```

同样修改：
- `pushAppMetrics` 方法
- `pushWorkflowMetrics` 方法

#### 步骤2：修改指标推送方法内部逻辑

为每个业务分组生成一份指标：

```go
// 修复前
groupName := ""
if e.groupLookup != nil && task.GroupID > 0 {
    groupName = e.groupLookup(ctx, task.GroupID)  // ❌ 资产分组
}

baseLabels := map[string]string{
    "business_group": groupName,  // ❌ 错误
}

// 修复后
// 如果没有传入业务分组，使用空字符串（向后兼容）
if len(businessGroupNames) == 0 {
    businessGroupNames = []string{""}
}

// 为每个业务分组生成一份指标
for _, businessGroupName := range businessGroupNames {
    baseLabels := map[string]string{
        "business_group": businessGroupName,  // ✅ 正确
    }
    
    // ... 生成并推送指标 ...
}
```

#### 步骤3：修改执行方法签名

添加 `businessGroupNames []string` 参数：

```go
func (e *NetworkProbeExecutor) executeAndSaveNetworkProbe(..., businessGroupNames []string)
func (e *NetworkProbeExecutor) executeAndSaveAppProbe(..., businessGroupNames []string)
func (e *NetworkProbeExecutor) executeAndSaveWorkflowProbe(..., businessGroupNames []string)
```

#### 步骤4：在调用推送方法时传入业务分组

```go
if probeTask.PushgatewayID > 0 {
    e.pushMetrics(ctx, probeTask, origCfg, result, businessGroupNames)
}
```

#### 步骤5：在任务执行时获取业务分组名称

**新表路径**（`ExecuteTaskV2` 方法）：

```go
// 获取业务分组名称列表
var businessGroupNames []string

// 优先级1：使用任务中配置的业务分组
if inspectionTask.BusinessGroupIDs != "" {
    var businessGroupIDs []uint
    if err := json.Unmarshal([]byte(inspectionTask.BusinessGroupIDs), &businessGroupIDs); err == nil && len(businessGroupIDs) > 0 {
        for _, bgID := range businessGroupIDs {
            if e.groupLookup != nil {
                if bgName := e.groupLookup(ctx, bgID); bgName != "" {
                    businessGroupNames = append(businessGroupNames, bgName)
                }
            }
        }
    }
}

// 优先级2：如果任务中没有配置，使用拨测配置中的分组
if len(businessGroupNames) == 0 && cfg.GroupIDs != "" {
    groupIDStrs := strings.Split(cfg.GroupIDs, ",")
    for _, idStr := range groupIDStrs {
        if id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 64); err == nil && id > 0 {
            if e.groupLookup != nil {
                if bgName := e.groupLookup(ctx, uint(id)); bgName != "" {
                    businessGroupNames = append(businessGroupNames, bgName)
                }
            }
        }
    }
}

// 调用执行方法时传入
executor.executeAndSaveAppProbe(ctx, probeTask, cfgPtr, resolvedCfg, &failCount, businessGroupNames)
```

**旧表路径**（`Execute` 方法）：

```go
// 旧表路径：从拨测配置的 GroupIDs 获取业务分组
var businessGroupNames []string
if cfg.GroupIDs != "" {
    groupIDStrs := strings.Split(cfg.GroupIDs, ",")
    for _, idStr := range groupIDStrs {
        if id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 64); err == nil && id > 0 {
            if e.groupLookup != nil {
                if bgName := e.groupLookup(ctx, uint(id)); bgName != "" {
                    businessGroupNames = append(businessGroupNames, bgName)
                }
            }
        }
    }
}
```

## 三、修改文件清单

### 3.1 巡检类指标
- `internal/service/inspection_mgmt/inspection_executor.go`
  - 第557行：修改 `baseLabels` 中的 `business_group`
  - 第566行：修改 `allLabels` 中的 `business_group`

### 3.2 拨测类指标
- `internal/biz/inspection/executor.go`
  - 第198行：修改 `executeAndSaveNetworkProbe` 方法签名
  - 第234行：修改调用 `pushMetrics` 的地方
  - 第239行：修改 `executeAndSaveAppProbe` 方法签名
  - 第299行：修改调用 `pushAppMetrics` 的地方
  - 第534行：修改 `executeAndSaveWorkflowProbe` 方法签名
  - 第561行：修改调用 `pushWorkflowMetrics` 的地方
  - 第1378行：修改 `pushMetrics` 方法签名和内部逻辑
  - 第1579行：修改 `pushAppMetrics` 方法签名和内部逻辑
  - 第1785行：修改 `pushWorkflowMetrics` 方法签名和内部逻辑
  - 第113-145行：旧表路径，添加业务分组获取逻辑
  - 第2223-2260行：新表路径，添加业务分组获取逻辑

## 四、修复效果

### 4.1 巡检类指标

**修复前**：
```prometheus
srehub_inspect_task_exec_total{
    task_name="主机巡检",
    business_group="主机基础指标检查",  # ❌ 错误：巡检组名称
    check_group="主机基础指标检查"
} 100
```

**修复后**：
```prometheus
srehub_inspect_task_exec_total{
    task_name="主机巡检",
    business_group="生产环境",  # ✅ 正确：业务分组名称
    check_group="主机基础指标检查"
} 100
```

### 4.2 拨测类指标

**修复前**（缺失 business_group）：
```prometheus
srehub_inspect_http_response_duration_seconds{
    task_name="API监控",
    target="https://api.example.com"
} 0.123
```

**修复后**：
```prometheus
srehub_inspect_http_response_duration_seconds{
    task_name="API监控",
    business_group="生产环境",  # ✅ 新增：业务分组标签
    target="https://api.example.com"
} 0.123
```

### 4.3 多业务分组支持

**场景**：一个任务关联多个业务分组

**效果**：为每个业务分组生成一份指标

```prometheus
srehub_inspect_http_response_duration_seconds{
    task_name="API监控",
    business_group="生产环境",
    target="https://api.example.com"
} 0.123

srehub_inspect_http_response_duration_seconds{
    task_name="API监控",
    business_group="测试环境",
    target="https://api.example.com"
} 0.145
```

## 五、业务分组优先级规则

### 5.1 巡检类指标
业务分组来源于执行记录（`record.BusinessGroup`），该字段已经应用了以下优先级规则：
1. **优先级1**：任务级覆盖（`InspectionTask.GroupBusinessGroupOverrides`）
2. **优先级2**：任务级配置（`InspectionTask.BusinessGroupIDs`）
3. **优先级3**：巡检组原始配置（`InspectionGroup.BusinessGroupIDs`）

### 5.2 拨测类指标
业务分组获取优先级：
1. **优先级1**：调度任务中的业务分组（`InspectionTask.BusinessGroupIDs`）
2. **优先级2**：拨测配置中的分组（`ProbeConfig.GroupIDs`）

**注意**：`ProbeConfig.GroupIDs` 是逗号分隔的分组ID，通过 `groupLookup` 函数查询分组名称。

## 六、向后兼容性

### 6.1 空业务分组处理
如果业务分组为空，`business_group` 标签值为空字符串：

```go
if len(businessGroupNames) == 0 {
    businessGroupNames = []string{""}
}
```

### 6.2 旧表路径兼容
旧的 `ProbeTask` 表调度路径（`Execute` 方法）也已修复，从拨测配置的 `GroupIDs` 获取业务分组。

### 6.3 Grafana 查询兼容
建议 Grafana 查询中使用正则匹配兼容空值：

```promql
business_group=~"$business_group|"
```

## 七、验证结果

### 7.1 编译验证
```bash
go build -o /dev/null ./cmd/... ./internal/... ./pkg/... ./plugins/...
```
✅ 编译通过，无错误

### 7.2 影响范围
- **修复的指标数量**：43个
  - 巡检类：12个（task 6个 + check 6个）
  - 拨测类：31个（HTTP 7个 + Ping 7个 + TCP 4个 + UDP 4个 + WebSocket 4个 + Flow 5个）

### 7.3 代码修改统计
- **修改文件数**：2个
  - `internal/service/inspection_mgmt/inspection_executor.go`（巡检）
  - `internal/biz/inspection/executor.go`（拨测）
- **修改行数**：约150行
- **新增代码**：约80行（业务分组获取逻辑）

## 八、注意事项

### 8.1 历史数据
- 修复后历史数据中的 `business_group` 值不会自动更新
- 旧数据仍然是错误值（巡检组名称或资产组名称）
- 如需修正历史数据，需通过 Prometheus remote_write 或重新采集

### 8.2 指标数量增加
- 多业务分组会导致指标数量成倍增加
- 例如：1个任务关联3个业务分组，指标数量 × 3
- 建议限制单个任务关联的业务分组数量（如最多5个）

### 8.3 Grafana 看板调整
修复后需要调整 Grafana 看板：
1. 确保 `business_group` 变量查询正确
2. 使用正则匹配兼容空值：`business_group=~"$business_group|"`
3. 验证所有使用 `business_group` 的面板是否正常显示

### 8.4 告警规则调整
如果告警规则中使用了 `business_group` 标签，需要验证：
1. 告警分组是否正确
2. 告警路由是否生效
3. 告警消息中的业务分组信息是否准确

## 九、测试建议

### 9.1 功能测试
1. **巡检任务测试**：
   - 创建巡检任务，配置业务分组
   - 执行任务后查看 Prometheus 指标
   - 验证 `business_group` 标签值是否正确

2. **拨测任务测试**：
   - 创建拨测任务，配置业务分组
   - 执行任务后查看 Prometheus 指标
   - 验证 `business_group` 标签是否存在且值正确

3. **多业务分组测试**：
   - 创建任务，关联多个业务分组
   - 验证是否为每个业务分组生成了独立的指标

### 9.2 性能测试
1. 测试多业务分组对指标推送性能的影响
2. 监控 Pushgateway 的负载变化
3. 验证 Prometheus 的存储和查询性能

### 9.3 兼容性测试
1. 测试旧的 `ProbeTask` 表调度路径是否正常
2. 测试业务分组为空的情况
3. 测试 Grafana 看板是否正常显示

## 十、后续优化建议

### 10.1 性能优化
- 考虑在 Redis 中缓存业务分组名称，减少数据库查询
- 批量查询业务分组名称，减少查询次数

### 10.2 功能增强
- 支持业务分组的层级结构（如：生产环境 > 核心业务）
- 支持业务分组的动态更新（修改分组名称后自动更新指标）

### 10.3 监控告警
- 添加业务分组缺失的告警
- 监控业务分组标签值的分布情况

## 十一、相关文档

- 原始问题报告：`docs/alert-patrol-guide.md`（如果存在）
- 指标规范文档：Prometheus 指标命名规范
- 业务分组设计文档：资产管理业务分组功能设计

## 十二、修复时间线

- **问题发现**：2026-05-14
- **方案设计**：2026-05-14
- **代码修复**：2026-05-14
- **编译验证**：2026-05-14 ✅
- **状态**：已完成，待测试验证

---

**修复完成日期**：2026-05-14  
**修复人员**：Claude (Opus 4.6)  
**审核状态**：待用户验证
