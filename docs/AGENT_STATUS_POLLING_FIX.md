# Agent 状态前端定时刷新修复

## 问题描述

**现象**: 主机管理页面中，Agent 断开连接后，"连接方式"列的 Agent 状态颜色不会自动更新，仍然显示绿色（在线状态），需要手动刷新页面才能看到灰色（离线状态）。

**原因**: 前端只在页面加载时调用一次 `loadAgentStatuses()` 获取 Agent 状态，之后没有定时刷新机制。当 Agent 断开连接后，前端不会自动获取最新状态。

## 问题分析

### 数据流

1. 页面加载时，调用 `loadAgentStatuses()` 获取所有 Agent 状态
2. 将状态合并到主机列表中（`mergeAgentStatuses()`）
3. 界面根据 `agentStatus` 字段显示颜色
4. **问题**: 之后 Agent 状态变化时，前端不会自动刷新

### 原有代码

```typescript
onMounted(async () => {
  // ...
  loadAgentStatuses()  // 只调用一次
})
```

### 后端 API

后端的 `/api/v1/agents/statuses` API 会返回实时的 Agent 状态：

```go
func (s *HTTPServer) GetAllStatuses(c *gin.Context) {
    agents, err := s.grpcServer.AgentRepo().List(c.Request.Context())
    // ...
    for _, a := range agents {
        online := s.hub.IsOnline(a.HostID)  // 检查 Hub 中的实时状态
        status := a.Status
        if online {
            status = "online"
        } else if status == "online" {
            status = "offline"
        }
        // ...
    }
}
```

这个 API 是正确的，会返回实时状态。问题是前端没有定期调用它。

## 解决方案

### 修复代码

在 `web/src/views/asset/Hosts.vue` 中添加定时刷新逻辑：

```typescript
// 加载Agent状态并合并到主机列表
const agentStatusMap = ref<Record<number, any>>({})
let agentStatusTimer: ReturnType<typeof setInterval> | null = null

const loadAgentStatuses = async () => {
  try {
    const data = await getAgentStatuses()
    const map: Record<number, any> = {}
    const list = Array.isArray(data) ? data : []
    for (const item of list) {
      map[item.hostId] = item
    }
    agentStatusMap.value = map
    // 合并Agent状态到主机列表
    mergeAgentStatuses()
  } catch {
    // 静默失败
  }
}

// 启动Agent状态定时刷新
const startAgentStatusPolling = () => {
  // 立即加载一次
  loadAgentStatuses()
  // 每30秒刷新一次Agent状态
  agentStatusTimer = setInterval(() => {
    loadAgentStatuses()
  }, 30000)
}

// 停止Agent状态定时刷新
const stopAgentStatusPolling = () => {
  if (agentStatusTimer) {
    clearInterval(agentStatusTimer)
    agentStatusTimer = null
  }
}
```

### 修改生命周期钩子

```typescript
onMounted(async () => {
  // ...
  startAgentStatusPolling()  // 启动定时刷新
})

onBeforeUnmount(() => {
  closeTerminal()
  stopAgentStatusPolling()  // 停止定时刷新
})
```

## 修复效果

### 修复前

1. 页面加载时显示 Agent 在线（绿色）
2. Agent 断开连接
3. 前端界面仍然显示绿色（在线）❌
4. 需要手动刷新页面才能看到灰色（离线）

### 修复后

1. 页面加载时显示 Agent 在线（绿色）
2. Agent 断开连接
3. 最多 30 秒后，前端自动刷新状态
4. 界面自动更新为灰色（离线）✅

## 技术细节

### 刷新频率

- **间隔**: 30 秒
- **原因**:
  - 太频繁会增加服务器负载
  - 太慢会导致状态更新不及时
  - 30 秒是一个平衡点

### 性能影响

- **网络请求**: 每 30 秒一次 GET 请求
- **数据量**: 取决于 Agent 数量，通常很小
- **CPU 占用**: 极低，只是简单的数据合并

### 内存管理

- 定时器在组件销毁时会被清理（`onBeforeUnmount`）
- 避免内存泄漏

## 相关代码

### 前端状态显示逻辑

```vue
<a-table-column title="连接方式" :width="100" align="center">
  <template #cell="{ record }">
    <a-tag v-if="record.connectionMode === 'agent' && record.agentStatus === 'online'" color="green" size="small">
      <icon-cloud /> Agent
    </a-tag>
    <a-tag v-else-if="record.connectionMode === 'agent'" color="orange" size="small">
      <icon-cloud /> Agent离线
    </a-tag>
    <a-tag v-else color="gray" size="small">
      SSH
    </a-tag>
  </template>
</a-table-column>
```

### 状态合并逻辑

```typescript
const mergeAgentStatuses = () => {
  if (!hostList.value || hostList.value.length === 0) return
  for (const host of hostList.value) {
    const agentInfo = agentStatusMap.value[(host as any).id]
    if (agentInfo) {
      ;(host as any).agentStatus = agentInfo.status
      ;(host as any).connectionMode = 'agent'
    }
  }
}
```

## 测试建议

### 功能测试

1. **正常刷新**:
   - 打开主机管理页面
   - 观察 Agent 状态正常显示
   - 等待 30 秒，确认状态自动刷新

2. **Agent 断开**:
   - 停止一个 Agent
   - 等待最多 30 秒
   - 确认界面自动变为灰色（离线）

3. **Agent 重连**:
   - 重启 Agent
   - 等待最多 30 秒
   - 确认界面自动变为绿色（在线）

4. **页面切换**:
   - 切换到其他页面
   - 切换回主机管理页面
   - 确认定时器正常工作

5. **组件销毁**:
   - 打开浏览器开发者工具
   - 切换到其他页面
   - 确认定时器被清理（无内存泄漏）

### 性能测试

1. **大量主机**:
   - 测试 100+ 台主机时的刷新性能
   - 确认不会卡顿

2. **网络延迟**:
   - 模拟慢网络
   - 确认不会阻塞界面

## 后续优化建议

### 1. WebSocket 实时推送

当前使用轮询（polling）方式，可以优化为 WebSocket 实时推送：

```typescript
// 伪代码
const ws = new WebSocket('/api/v1/agents/status-stream')
ws.onmessage = (event) => {
  const { hostId, status } = JSON.parse(event.data)
  updateAgentStatus(hostId, status)
}
```

**优点**:
- 实时性更好（秒级响应）
- 减少不必要的请求
- 降低服务器负载

**缺点**:
- 需要后端支持 WebSocket
- 实现复杂度更高

### 2. 智能刷新频率

根据 Agent 状态变化频率动态调整刷新间隔：

```typescript
// 伪代码
let refreshInterval = 30000  // 初始 30 秒

// 如果检测到状态变化频繁，缩短间隔
if (statusChangeCount > 5) {
  refreshInterval = 10000  // 10 秒
}

// 如果长时间无变化，延长间隔
if (noChangeTime > 300000) {
  refreshInterval = 60000  // 60 秒
}
```

### 3. 仅刷新可见区域

如果主机列表很长，只刷新当前可见区域的 Agent 状态：

```typescript
// 伪代码
const visibleHostIds = getVisibleHostIds()
const data = await getAgentStatuses({ hostIds: visibleHostIds })
```

## 修改的文件

- `web/src/views/asset/Hosts.vue` - 添加定时刷新逻辑

## 影响范围

- **正面影响**:
  - Agent 状态自动更新，用户体验提升
  - 无需手动刷新页面

- **性能影响**:
  - 每 30 秒一次 HTTP 请求
  - 影响极小

## 总结

本次修复通过添加定时刷新机制，解决了 Agent 状态在前端界面不自动更新的问题。修复采用了简单可靠的轮询方式，刷新间隔为 30 秒，在实时性和性能之间取得了平衡。

---

**修复时间**: 2026-03-03
**修复人员**: Claude (AI Assistant)
**影响版本**: 所有版本
