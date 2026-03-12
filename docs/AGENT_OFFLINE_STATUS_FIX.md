# Agent 离线状态缓存同步修复

## 问题描述

**现象**: 主机管理页面中，某些主机的 `agentStatus` 在数据库中已经是 `"offline"`，但前端界面仍然显示绿色的在线状态。

**原因**: 当 Agent 断开连接时，代码只更新了 MySQL 数据库中的状态为 `offline`，但没有同步更新 Redis 缓存。前端调用 `getAgentStatuses()` API 时，获取到的是 Redis 缓存中的旧数据（仍然是 `online`），导致界面显示不一致。

## 问题分析

### 原有代码逻辑

在 `internal/server/agent/agent_service.go` 的 `Connect()` 方法中：

```go
defer func() {
    if agentID != "" {
        s.hub.Unregister(agentID)
        // 更新状态为offline
        s.agentRepo.UpdateStatus(context.Background(), agentID, "offline")
        s.db.Model(&struct{ AgentStatus string }{}).
            Exec("UPDATE hosts SET agent_status = 'offline' WHERE agent_id = ?", agentID)
    }
}()
```

**问题**: 只更新了 MySQL，没有更新 Redis 缓存。

### 数据流

1. Agent 断开连接
2. `defer` 函数执行，更新 MySQL 状态为 `offline`
3. Redis 缓存中的状态仍然是 `online`（TTL 3分钟）
4. 前端查询 Agent 状态时，从 Redis 获取到 `online`
5. 界面显示绿色（在线状态）

## 解决方案

### 修复代码

在 Agent 断开连接时，同步更新 Redis 缓存：

```go
defer func() {
    if agentID != "" {
        s.hub.Unregister(agentID)

        // 更新状态为offline
        s.agentRepo.UpdateStatus(context.Background(), agentID, "offline")
        s.db.Model(&struct{ AgentStatus string }{}).
            Exec("UPDATE hosts SET agent_status = 'offline' WHERE agent_id = ?", agentID)

        // 更新缓存中的状态为offline
        if s.cacheManager != nil {
            ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
            defer cancel()

            updates := map[string]any{
                "status": "offline",
            }

            if err := s.cacheManager.UpdateAgentStatus(ctx, agentID, updates); err != nil {
                appLogger.Warn("更新 Agent 离线状态到缓存失败",
                    zap.String("agentID", agentID),
                    zap.Error(err))
            }
        }
    }
}()
```

### 修复后的数据流

1. Agent 断开连接
2. `defer` 函数执行
3. 更新 MySQL 状态为 `offline`
4. 更新 Redis 缓存状态为 `offline`
5. 前端查询 Agent 状态时，从 Redis 获取到 `offline`
6. 界面显示灰色（离线状态）✅

## 修改的文件

- `internal/server/agent/agent_service.go` - Connect() 方法的 defer 函数

## 验证方法

### 1. 重启服务

```bash
make run
```

### 2. 测试 Agent 断开

1. 启动一个 Agent
2. 观察主机管理页面，Agent 状态显示为绿色（在线）
3. 停止 Agent
4. 刷新主机管理页面，Agent 状态应该立即变为灰色（离线）

### 3. 验证 Redis 缓存

```bash
redis-cli

# 查看 Agent 状态
> GET agent:status:3785c438-e777-468c-8b41-14c7c54b0958

# 应该看到 status 为 "offline"
```

## 相关代码

### Agent 状态更新的三个场景

1. **Agent 心跳** - 更新为 `online`
   - 位置: `handleHeartbeat()` 方法
   - 策略: 异步更新 Redis + MySQL

2. **Agent 断开** - 更新为 `offline`
   - 位置: `Connect()` 方法的 `defer` 函数
   - 策略: 同步更新 MySQL + Redis（本次修复）

3. **Agent 注册** - 更新为 `online`
   - 位置: `handleRegister()` 方法
   - 策略: 同步更新 MySQL

## 注意事项

1. **超时控制**: 缓存更新使用 3 秒超时，避免阻塞 Agent 断开流程
2. **错误处理**: 如果缓存更新失败，只记录警告日志，不影响主流程
3. **降级策略**: 如果 `cacheManager` 为 nil，跳过缓存更新，保持原有逻辑

## 影响范围

- **正面影响**:
  - 前端界面显示的 Agent 状态与实际状态一致
  - 用户体验提升，不会看到已断开的 Agent 仍显示在线

- **性能影响**:
  - Agent 断开时增加一次 Redis 更新操作
  - 影响极小（断开是低频操作）

## 测试建议

1. **功能测试**:
   - Agent 正常连接和断开
   - 多个 Agent 同时断开
   - Agent 频繁重连

2. **边界测试**:
   - Redis 不可用时 Agent 断开
   - 网络延迟时 Agent 断开
   - 大量 Agent 同时断开

3. **回归测试**:
   - 验证 Agent 心跳仍然正常
   - 验证主机列表查询正常
   - 验证缓存预热正常

## 总结

本次修复解决了 Agent 离线状态在缓存中不同步的问题，确保了前端界面显示的 Agent 状态与实际状态一致。修复采用了与心跳更新相同的缓存同步机制，保持了代码的一致性。

---

**修复时间**: 2026-03-03
**修复人员**: Claude (AI Assistant)
**影响版本**: 所有使用缓存系统的版本
