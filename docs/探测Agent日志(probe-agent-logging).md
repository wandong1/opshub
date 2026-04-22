# 拨测 Agent 模式日志追踪指南

## 概述

为了帮助诊断和验证 Agent 模式拨测是否正确执行，我们在关键位置添加了详细的日志。本文档说明如何使用这些日志来追踪 Agent 模式的执行流程。

## 日志位置

### 1. Agent 连接管理日志

**文件**: `internal/server/agent/hub.go`

#### Agent 注册日志
```
[INFO] Agent已注册 {"agent_id": "xxx", "host_id": 123, "total_agents": 5}
```

**说明**：
- 当 Agent 连接到服务端时记录
- `agent_id`: Agent 的唯一标识符
- `host_id`: 关联的主机 ID
- `total_agents`: 当前在线的 Agent 总数

#### Agent 注销日志
```
[INFO] Agent已注销 {"agent_id": "xxx", "host_id": 123, "total_agents": 4}
```

**说明**：
- 当 Agent 断开连接时记录
- 显示断开后剩余的 Agent 数量

#### Agent 在线状态检查日志
```
[INFO] Checking agent online status {"host_id": 123, "is_online": true, "total_agents": 5}
```

**说明**：
- 每次检查 Agent 是否在线时记录
- `is_online`: true=在线，false=离线
- 这是诊断 Agent 模式问题的关键日志

### 2. 应用服务拨测日志

**文件**: `internal/biz/inspection/executor.go`

#### 拨测执行入口日志
```
[INFO] executeAppProbe called {"probe_name": "api-health-check", "exec_mode": "agent", "agent_host_ids": "123,456", "type": "http"}
```

**说明**：
- 每次执行应用服务拨测时记录
- `exec_mode`: 执行模式（local/agent/proxy）
- `agent_host_ids`: 配置的 Agent 主机 ID 列表

#### Agent 模式检测日志
```
[INFO] Agent mode detected for application probe {"probe_name": "api-health-check", "agent_host_ids": "123,456"}
```

**说明**：
- 确认进入 Agent 模式分支

#### Agent 主机 ID 解析日志
```
[INFO] Parsed agent host IDs {"probe_name": "api-health-check", "host_ids": [123, 456]}
```

**说明**：
- 显示解析后的 Agent 主机 ID 列表
- 如果为空，说明配置有问题

#### Agent 状态检查日志
```
[INFO] Checking agent status {"probe_name": "api-health-check", "host_id": 123}
[INFO] Agent online status {"probe_name": "api-health-check", "host_id": 123, "is_online": false}
[WARN] Agent is offline, trying next {"probe_name": "api-health-check", "host_id": 123}
```

**说明**：
- 逐个检查配置的 Agent 是否在线
- 如果离线，尝试下一个 Agent

#### Agent 模式未实现警告
```
[WARN] Agent mode for application probing not yet implemented {"probe_name": "api-health-check", "host_id": 123}
```

**说明**：
- 当 Agent 在线但功能未实现时记录
- 这是当前的预期行为

#### 所有 Agent 不可用错误
```
[ERROR] All agents unavailable {"probe_name": "api-health-check", "error": "agent on host 123 is offline"}
```

**说明**：
- 所有配置的 Agent 都离线时记录

#### Local/Proxy 模式日志
```
[INFO] Local mode detected, executing locally {"probe_name": "api-health-check"}
[INFO] Proxy mode detected, executing locally with proxy {"probe_name": "api-health-check", "proxy_url": "http://proxy:8080"}
```

**说明**：
- 确认使用本地或代理模式执行

### 3. Workflow 拨测日志

**文件**: `internal/biz/inspection/executor.go`

#### Workflow 步骤执行日志
```
[INFO] Workflow step execution {"workflow_name": "user-login-flow", "step_name": "login", "step_index": 0, "step_exec_mode": "agent", "probe_type": "http", "url": "https://api.example.com/login"}
```

**说明**：
- 每个 Workflow 步骤执行时记录
- `step_exec_mode`: 步骤的执行模式

#### Workflow Agent 模式日志
```
[INFO] Workflow step using agent mode {"workflow_name": "user-login-flow", "step_name": "login", "agent_host_ids": "123"}
[INFO] Parsed agent host IDs for workflow step {"workflow_name": "user-login-flow", "step_name": "login", "host_ids": [123]}
[WARN] Agent mode for workflow step not yet implemented {"workflow_name": "user-login-flow", "step_name": "login", "host_ids": [123]}
```

**说明**：
- Workflow 步骤使用 Agent 模式时的日志流程

#### Workflow Local/Proxy 模式日志
```
[INFO] Workflow step using local/proxy mode {"workflow_name": "user-login-flow", "step_name": "login", "exec_mode": "local"}
```

## 日志追踪场景

### 场景 1：验证 Agent 是否在线

**步骤**：
1. 查看 Agent 注册日志，确认 Agent 已连接
2. 记录 `host_id`
3. 执行拨测时，查找 "Checking agent online status" 日志
4. 验证 `is_online` 字段

**示例日志流程**：
```
[INFO] Agent已注册 {"agent_id": "abc-123", "host_id": 456, "total_agents": 1}
...
[INFO] executeAppProbe called {"probe_name": "test", "exec_mode": "agent", "agent_host_ids": "456"}
[INFO] Agent mode detected for application probe {"probe_name": "test", "agent_host_ids": "456"}
[INFO] Parsed agent host IDs {"probe_name": "test", "host_ids": [456]}
[INFO] Checking agent status {"probe_name": "test", "host_id": 456}
[INFO] Checking agent online status {"host_id": 456, "is_online": true, "total_agents": 1}
[INFO] Agent online status {"probe_name": "test", "host_id": 456, "is_online": true}
[WARN] Agent mode for application probing not yet implemented {"probe_name": "test", "host_id": 456}
```

### 场景 2：诊断 Agent 离线问题

**步骤**：
1. 执行拨测
2. 查找 "Agent is offline" 日志
3. 检查 Agent 注册/注销日志，确认 Agent 状态

**示例日志流程**：
```
[INFO] executeAppProbe called {"probe_name": "test", "exec_mode": "agent", "agent_host_ids": "999"}
[INFO] Agent mode detected for application probe {"probe_name": "test", "agent_host_ids": "999"}
[INFO] Parsed agent host IDs {"probe_name": "test", "host_ids": [999]}
[INFO] Checking agent status {"probe_name": "test", "host_id": 999}
[INFO] Checking agent online status {"host_id": 999, "is_online": false, "total_agents": 1}
[INFO] Agent online status {"probe_name": "test", "host_id": 999, "is_online": false}
[WARN] Agent is offline, trying next {"probe_name": "test", "host_id": 999}
[ERROR] All agents unavailable {"probe_name": "test", "error": "agent on host 999 is offline"}
```

### 场景 3：验证 Agent 模式配置

**步骤**：
1. 检查 "executeAppProbe called" 日志中的 `exec_mode`
2. 验证 `agent_host_ids` 是否正确
3. 检查 "Parsed agent host IDs" 日志，确认解析结果

**示例日志流程（配置错误）**：
```
[INFO] executeAppProbe called {"probe_name": "test", "exec_mode": "agent", "agent_host_ids": ""}
[INFO] Agent mode detected for application probe {"probe_name": "test", "agent_host_ids": ""}
[INFO] Parsed agent host IDs {"probe_name": "test", "host_ids": []}
[ERROR] No agent host specified for agent mode {"probe_name": "test"}
```

### 场景 4：验证 Workflow 步骤 Agent 模式

**步骤**：
1. 查找 "Workflow step execution" 日志
2. 检查 `step_exec_mode` 字段
3. 验证是否进入 Agent 模式分支

**示例日志流程**：
```
[INFO] Workflow step execution {"workflow_name": "test-flow", "step_name": "step1", "step_index": 0, "step_exec_mode": "agent", "probe_type": "http", "url": "https://api.example.com"}
[INFO] Workflow step using agent mode {"workflow_name": "test-flow", "step_name": "step1", "agent_host_ids": "123"}
[INFO] Parsed agent host IDs for workflow step {"workflow_name": "test-flow", "step_name": "step1", "host_ids": [123]}
[WARN] Agent mode for workflow step not yet implemented {"workflow_name": "test-flow", "step_name": "step1", "host_ids": [123]}
```

## 日志查询命令

### 查看 Agent 注册/注销日志
```bash
tail -f logs/opshub.log | grep -E "Agent已注册|Agent已注销"
```

### 查看 Agent 在线状态检查
```bash
tail -f logs/opshub.log | grep "Checking agent online status"
```

### 查看应用服务拨测执行日志
```bash
tail -f logs/opshub.log | grep "executeAppProbe"
```

### 查看 Agent 模式相关日志
```bash
tail -f logs/opshub.log | grep -E "Agent mode|agent mode"
```

### 查看特定拨测的完整日志
```bash
tail -f logs/opshub.log | grep "probe_name.*test-probe"
```

### 查看 Workflow 步骤执行日志
```bash
tail -f logs/opshub.log | grep "Workflow step"
```

## 常见问题诊断

### 问题 1：拨测成功但怀疑没有通过 Agent 执行

**诊断步骤**：
1. 查找 "executeAppProbe called" 日志，检查 `exec_mode`
2. 如果 `exec_mode` 不是 "agent"，说明配置错误
3. 如果是 "agent"，查找 "Agent mode detected" 日志
4. 如果没有找到，说明代码逻辑有问题

### 问题 2：Agent 明明在线，但拨测报告离线

**诊断步骤**：
1. 查找最近的 "Agent已注册" 日志，记录 `host_id`
2. 查找拨测的 "Parsed agent host IDs" 日志
3. 对比两个 `host_id` 是否一致
4. 如果不一致，说明配置的 Agent 主机 ID 错误

### 问题 3：配置了多个 Agent，不知道使用了哪个

**诊断步骤**：
1. 查找 "Parsed agent host IDs" 日志，查看配置的 Agent 列表
2. 查找 "Checking agent status" 日志，按顺序检查每个 Agent
3. 第一个 `is_online: true` 的 Agent 会被使用

### 问题 4：Workflow 步骤没有使用 Agent 模式

**诊断步骤**：
1. 查找 "Workflow step execution" 日志
2. 检查 `step_exec_mode` 字段
3. 如果是 "local" 或空，说明步骤配置错误
4. 如果是 "agent"，查找 "Workflow step using agent mode" 日志

## 日志级别说明

- **INFO**: 正常流程日志，用于追踪执行路径
- **WARN**: 警告日志，表示非预期但可处理的情况（如 Agent 离线、功能未实现）
- **ERROR**: 错误日志，表示执行失败（如所有 Agent 不可用、配置错误）

## 性能考虑

这些日志在生产环境中会产生一定的性能开销。如果需要减少日志量，可以：

1. 调整日志级别，只记录 WARN 和 ERROR
2. 在 `config/config.yaml` 中配置日志级别：
```yaml
logger:
  level: warn  # debug, info, warn, error
```

3. 使用日志采样（如果日志量过大）

## 未来改进

当 Agent 模式应用拨测功能实现后，日志会包含：

```
[INFO] Executing application probe via agent {"probe_name": "test", "host_id": 123, "agent_id": "abc-123"}
[INFO] Agent probe request sent {"probe_name": "test", "request_id": "req-456"}
[INFO] Agent probe response received {"probe_name": "test", "request_id": "req-456", "success": true, "latency": 123.45}
```

## 总结

通过这些详细的日志，您可以：

1. ✅ 验证 Agent 是否在线
2. ✅ 确认拨测是否使用了 Agent 模式
3. ✅ 诊断 Agent 离线问题
4. ✅ 验证配置是否正确
5. ✅ 追踪 Workflow 步骤的执行模式
6. ✅ 了解当前功能的实现状态

如果您发现拨测仍然能够成功但日志显示 Agent 离线，请检查：
- 拨测配置中的 `execMode` 字段是否为 "agent"
- 拨测配置中的 `agentHostIds` 字段是否正确
- Agent 是否真的在线（查看 Agent 注册日志）
