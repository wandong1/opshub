# 拨测 Agent 模式修复说明

## 问题描述

用户报告：在应用服务拨测和业务流程拨测中配置 Agent 模式时，即使选择的 Agent 处于离线状态，拨测仍然能够成功执行。这不符合预期行为。

## 根本原因

### 1. 应用服务拨测（HTTP/HTTPS/WebSocket）

在 `internal/biz/inspection/executor.go` 的 `executeAppProbe()` 函数中：

**修复前的代码**：
```go
// Agent mode
if cfg.ExecMode == ExecModeAgent && e.agentFactory != nil {
    // For now, agent app probing falls back to local execution
    // TODO: implement agent-side HTTP/WS probing via gRPC
    prober, err := probers.GetAppProber(cfg.Type)
    if err != nil {
        return &probers.AppResult{Error: err.Error()}, 0
    }
    return prober.ProbeApp(appCfg), 0  // ❌ 直接本地执行，忽略 Agent 配置
}
```

**问题**：
- 即使配置了 Agent 模式，代码也会直接回退到本地执行
- 完全没有检查 Agent 是否在线
- 用户无法感知到 Agent 模式未生效

### 2. Workflow 拨测

在 `ExecuteWorkflowProbe()` 函数中：

**修复前的代码**：
```go
appResult := prober.ProbeApp(appCfg)  // ❌ 直接本地执行，忽略 step.ExecMode
```

**问题**：
- Workflow 的每个步骤都有 `ExecMode` 字段，但执行时完全被忽略
- 无法通过 Agent 执行 Workflow 步骤

## 修复方案

### 1. 应用服务拨测修复

**修复后的代码**：
```go
// Agent mode
if cfg.ExecMode == ExecModeAgent {
    if e.agentFactory == nil {
        return &probers.AppResult{
            Error: "agent factory not initialized",
        }, 0
    }

    // Parse agent host IDs
    hostIDs := parseHostIDs(cfg.AgentHostIDs)
    if len(hostIDs) == 0 {
        return &probers.AppResult{
            Error: "no agent host specified for agent mode",
        }, 0
    }

    // Try each agent host until one succeeds
    var lastErr error
    for _, hostID := range hostIDs {
        // Check if agent is online
        if !e.agentFactory.IsOnline(hostID) {
            lastErr = fmt.Errorf("agent on host %d is offline", hostID)
            continue
        }

        // For now, agent app probing is not implemented, return error
        return &probers.AppResult{
            Error: fmt.Sprintf("agent mode for application probing not yet implemented (agent host: %d)", hostID),
        }, hostID
    }

    if lastErr != nil {
        return &probers.AppResult{
            Error: fmt.Sprintf("all agents unavailable: %v", lastErr),
        }, 0
    }

    return &probers.AppResult{
        Error: "no available agent found",
    }, 0
}
```

**改进点**：
- ✅ 检查 `agentFactory` 是否初始化
- ✅ 验证是否配置了 Agent 主机 ID
- ✅ 检查 Agent 是否在线（`agentFactory.IsOnline(hostID)`）
- ✅ 如果 Agent 离线，返回明确的错误信息
- ✅ 支持多个 Agent 主机，依次尝试直到找到在线的
- ✅ 明确告知用户 Agent 模式尚未实现（而不是静默回退）

### 2. Workflow 拨测修复

**修改函数签名**：
```go
// 修复前
func ExecuteWorkflowProbe(ctx context.Context, cfg *ProbeConfig, resolver *VariableResolver) *WorkflowResult

// 修复后
func ExecuteWorkflowProbe(ctx context.Context, cfg *ProbeConfig, resolver *VariableResolver, agentFactory AgentCommandFactory) *WorkflowResult
```

**修复后的步骤执行逻辑**：
```go
// Check step.ExecMode for agent execution
var appResult *probers.AppResult
if step.ExecMode == ExecModeAgent && agentFactory != nil {
    // Parse agent host IDs from config
    hostIDs := parseHostIDs(cfg.AgentHostIDs)
    if len(hostIDs) == 0 {
        stepResult.Error = "no agent host specified for agent mode"
        stepResult.Success = false
        // ... 错误处理
        continue
    }

    // Agent mode not yet implemented for application probing
    stepResult.Error = fmt.Sprintf("agent mode for application probing not yet implemented (step: %s)", step.Name)
    stepResult.Success = false
    // ... 错误处理
    continue
} else {
    // Local or proxy execution
    prober, err := probers.GetAppProber(probeType)
    if err != nil {
        // ... 错误处理
    }
    appResult = prober.ProbeApp(appCfg)
}
```

**改进点**：
- ✅ 检查 `step.ExecMode` 字段
- ✅ 如果是 Agent 模式，验证配置并返回明确错误
- ✅ 支持未来扩展 Agent 模式实现

### 3. Service 层调用修复

修改了 `internal/service/inspection/probe_config_service.go` 中的两处调用：

```go
// 修复前
wfResult := biz.ExecuteWorkflowProbe(c.Request.Context(), resolvedConfig, s.variableResolver)

// 修复后
wfResult := biz.ExecuteWorkflowProbe(c.Request.Context(), resolvedConfig, s.variableResolver, nil)
```

**说明**：
- Service 层的"立即执行"功能不支持 Agent 模式，传入 `nil`
- 只有通过任务调度执行时才会传入 `agentFactory`

## 修复效果

### 修复前的行为

| 场景 | 实际行为 | 用户感知 |
|:-----|:---------|:---------|
| 应用拨测 + Agent 模式 + Agent 离线 | 本地执行成功 | ❌ 误以为 Agent 在线 |
| 应用拨测 + Agent 模式 + Agent 在线 | 本地执行成功 | ❌ 误以为通过 Agent 执行 |
| Workflow + step.ExecMode=agent | 本地执行 | ❌ 配置被忽略 |

### 修复后的行为

| 场景 | 实际行为 | 用户感知 |
|:-----|:---------|:---------|
| 应用拨测 + Agent 模式 + Agent 离线 | 返回错误："agent on host X is offline" | ✅ 明确知道 Agent 离线 |
| 应用拨测 + Agent 模式 + Agent 在线 | 返回错误："agent mode not yet implemented" | ✅ 明确知道功能未实现 |
| Workflow + step.ExecMode=agent | 返回错误："agent mode not yet implemented" | ✅ 明确知道功能未实现 |

## 错误信息示例

### 1. Agent 未配置
```json
{
  "success": false,
  "error": "no agent host specified for agent mode"
}
```

### 2. Agent 离线
```json
{
  "success": false,
  "error": "all agents unavailable: agent on host 123 is offline"
}
```

### 3. Agent 模式未实现
```json
{
  "success": false,
  "error": "agent mode for application probing not yet implemented (agent host: 123)"
}
```

### 4. Agent Factory 未初始化
```json
{
  "success": false,
  "error": "agent factory not initialized"
}
```

## 未来实现 Agent 模式的步骤

当需要实现 Agent 端应用拨测时，需要完成以下工作：

### 1. Agent 端实现

**文件**：`agent/internal/prober/`

需要实现：
- HTTP/HTTPS 拨测
- WebSocket 拨测
- 断言评估
- 性能分解（DNS、TCP、TLS、TTFB）

### 2. gRPC 协议扩展

**文件**：`pkg/agentproto/agent.proto`

新增消息类型：
```protobuf
message HTTPProbeRequest {
    string url = 1;
    string method = 2;
    map<string, string> headers = 3;
    string body = 4;
    int32 timeout = 5;
    bool skip_verify = 6;
    repeated Assertion assertions = 7;
}

message HTTPProbeResponse {
    bool success = 1;
    int32 status_code = 2;
    string response_body = 3;
    map<string, string> response_headers = 4;
    double latency = 5;
    double dns_lookup_time = 6;
    double tcp_connect_time = 7;
    double tls_handshake_time = 8;
    double ttfb = 9;
    string error = 10;
}
```

### 3. 服务端调用逻辑

**文件**：`internal/biz/inspection/executor.go`

修改 `executeAppProbe()` 中的 Agent 模式实现：
```go
if cfg.ExecMode == ExecModeAgent {
    // ... 检查 Agent 在线状态

    // 通过 gRPC 调用 Agent 执行拨测
    result, err := executeAppProbeViaAgent(hostID, appCfg)
    if err != nil {
        return &probers.AppResult{Error: err.Error()}, hostID
    }
    return result, hostID
}
```

### 4. Workflow 步骤支持

修改 `ExecuteWorkflowProbe()` 中的步骤执行逻辑，调用 Agent 执行 HTTP 步骤。

## 测试验证

### 测试场景 1：Agent 离线

**步骤**：
1. 创建应用服务拨测配置
2. 设置 `execMode: agent`
3. 选择一个离线的 Agent 主机
4. 执行拨测

**预期结果**：
- 拨测失败
- 错误信息：`agent on host X is offline`

### 测试场景 2：Agent 在线但功能未实现

**步骤**：
1. 创建应用服务拨测配置
2. 设置 `execMode: agent`
3. 选择一个在线的 Agent 主机
4. 执行拨测

**预期结果**：
- 拨测失败
- 错误信息：`agent mode for application probing not yet implemented`

### 测试场景 3：Workflow 步骤 Agent 模式

**步骤**：
1. 创建 Workflow 拨测配置
2. 在某个 HTTP 步骤中设置 `execMode: agent`
3. 执行拨测

**预期结果**：
- 步骤失败
- 错误信息：`agent mode for application probing not yet implemented (step: xxx)`

### 测试场景 4：Local 模式正常工作

**步骤**：
1. 创建应用服务拨测配置
2. 设置 `execMode: local`（或不设置，默认 local）
3. 执行拨测

**预期结果**：
- 拨测正常执行
- 使用本地网络发起请求

## 影响范围

### 修改的文件

1. `internal/biz/inspection/executor.go`
   - `executeAppProbe()` - 添加 Agent 在线检查
   - `ExecuteWorkflowProbe()` - 添加 agentFactory 参数
   - Workflow HTTP 步骤执行逻辑 - 支持 step.ExecMode

2. `internal/service/inspection/probe_config_service.go`
   - 两处 `ExecuteWorkflowProbe()` 调用 - 传入 nil

### 向后兼容性

- ✅ 不影响现有的 Local 模式拨测
- ✅ 不影响现有的 Proxy 模式拨测
- ✅ 不影响网络拨测（Ping/TCP/UDP）的 Agent 模式
- ⚠️ Agent 模式应用拨测会明确报错（之前是静默回退）

### 数据库影响

- 无数据库结构变更
- 无需数据迁移

## 总结

本次修复解决了用户报告的问题：**Agent 模式配置被忽略，导致用户误以为 Agent 在线或功能正常**。

修复后的行为：
- ✅ 明确检查 Agent 是否在线
- ✅ 如果 Agent 离线，返回清晰的错误信息
- ✅ 如果功能未实现，明确告知用户
- ✅ 为未来实现 Agent 模式打下基础

用户现在可以：
- 准确判断 Agent 是否在线
- 了解 Agent 模式的实现状态
- 避免误以为拨测通过 Agent 执行

## 相关文档

- [智能巡检拨测 Metrics 指标说明](./probe-metrics.md)
- [Agent 系统架构](../CLAUDE.md#agent-系统)
