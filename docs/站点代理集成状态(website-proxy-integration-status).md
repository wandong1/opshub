# Web站点代理功能集成状态

## 功能概述
通过 Agent 的拨测能力实现内部站点的 HTTP 代理访问，复用现有的 Agent gRPC 通信和拨测基础设施。

## 集成完成情况

### ✅ 后端集成

#### 1. Agent 通信层 (`internal/server/agent/`)
- **agent_service.go:108-111** - 添加 ProbeResult 消息处理
  ```go
  case *pb.AgentMessage_ProbeResult:
      if as != nil {
          as.ResolvePending(payload.ProbeResult.RequestId, payload.ProbeResult)
      }
  ```

- **hub.go:210-263** - SendProbeRequest 方法（返回 error）
  - 发送拨测请求到 Agent
  - 等待响应（30秒超时）
  - 正确的错误处理

- **agent_executor.go:82-84** - AgentCommandFactory 实现
  ```go
  func (f *AgentCommandFactory) SendProbeRequest(hostID uint, req *pb.ProbeRequest) (*pb.ProbeResult, error) {
      return f.hub.SendProbeRequest(hostID, req)
  }
  ```

#### 2. 业务接口层 (`internal/biz/inspection/`)
- **agent.go:15** - 接口定义更新
  ```go
  SendProbeRequest(hostID uint, req *pb.ProbeRequest) (*pb.ProbeResult, error)
  ```

- **executor.go:366-377** - 拨测功能使用（兼容性验证）
  - 正确调用 SendProbeRequest
  - 完整的错误处理
  - 日志记录

#### 3. Web站点代理 (`internal/server/asset/`)
- **website_proxy.go** - 代理处理器
  - `AccessWebsite()` - 返回代理访问信息
  - `ProxyRequest()` - 通过 Agent 代理 HTTP 请求
  - `proxyViaAgent()` - 构建拨测请求并转发

- **http.go:208-211** - 依赖注入
  ```go
  var websiteProxyHandler *assetserver.WebsiteProxyHandler
  if s.grpcServer != nil {
      websiteProxyHandler = assetserver.NewWebsiteProxyHandler(websiteUseCase, s.grpcServer.Hub())
  }
  ```

- **http.go:412-413** - 路由注册
  ```go
  websites.GET("/:id/access", s.websiteProxyHandler.AccessWebsite)
  websites.Any("/:id/proxy/*path", s.websiteProxyHandler.ProxyRequest)
  ```

### ✅ 前端集成

#### 1. API 层 (`web/src/api/website.ts`)
- `accessWebsite(id)` - 获取访问信息
- 返回类型：`{ type: string; url?: string; proxyUrl?: string; hostId?: number }`

#### 2. 页面层 (`web/src/views/asset/Websites.vue`)
- **handleAccess()** - 访问站点逻辑
  - 外部站点：直接打开 URL
  - 内部站点：打开代理 URL (`/api/v1/websites/:id/proxy`)

- **表单验证** - 内部站点必须绑定至少1台 Agent 主机
- **Agent 状态显示** - 表格中显示 Agent 在线/离线状态

## 兼容性验证

### ✅ 与现有拨测功能兼容
1. **接口统一** - `SendProbeRequest` 方法同时服务于：
   - 拨测功能 (`internal/biz/inspection/executor.go:366`)
   - 站点代理 (`internal/server/asset/website_proxy.go:185`)

2. **消息处理** - ProbeResult 消息在 `agent_service.go` 中统一处理

3. **错误处理** - 所有调用点都正确处理返回的 error

4. **超时机制** - 统一使用 30 秒超时

## 工作流程

### 内部站点访问流程
1. 用户点击"访问"按钮
2. 前端调用 `/api/v1/websites/:id/access`
3. 后端检查 Agent 在线状态，返回 `proxyUrl`
4. 前端打开 `proxyUrl` (`/api/v1/websites/:id/proxy`)
5. 后端构建 ProbeRequest（HTTP 拨测请求）
6. 通过 gRPC 发送到 Agent
7. Agent 执行 HTTP 请求并返回 ProbeResult
8. 后端解析结果，返回给前端

### Agent 选择策略
- 内部站点可绑定多台 Agent 主机
- 优先选择第一台在线的 Agent
- 所有 Agent 离线时返回 503 错误

## 编译状态
✅ 主服务编译成功：`go build -o /dev/null cmd/server/server.go`
✅ Agent 编译成功：`cd agent && go build -o /dev/null cmd/main.go`

## 响应体大小限制优化

### 问题
原始实现中，Agent 端的 HTTP 拨测器将响应体限制为 4KB，这对于拨测功能是合理的，但对于 Web 站点代理来说太小（会截断网页内容）。

### 解决方案
在 `ProbeRequest` 中添加 `max_response_body_size` 字段（int32），允许调用方配置响应体大小限制：
- **默认值**：4KB（拨测场景）
- **代理场景**：设置为 0 表示不限制（实际上限 10MB，防止内存溢出）
- **自定义**：可设置任意大小

### 修改文件
1. **api/proto/agent.proto:159** - 添加 `max_response_body_size` 字段
2. **agent/internal/prober/http.go:143-157** - 实现可配置的响应体大小限制
3. **internal/server/asset/website_proxy.go:171** - 代理场景设置为 0（不限制）

### 兼容性
- ✅ 拨测功能：不设置该字段时默认 4KB，保持原有行为
- ✅ 代理功能：明确设置为 0，支持完整网页内容
- ✅ 向后兼容：旧版本 Agent 忽略该字段，使用默认 4KB

## 待测试项
1. 外部站点直接访问
2. 内部站点通过 Agent 代理访问
3. Agent 离线时的错误提示
4. 多 Agent 主机的故障转移
5. 拨测功能不受影响

## 技术亮点
1. **复用基础设施** - 利用现有 Agent 拨测能力，无需新增通信协议
2. **接口解耦** - 通过 `AgentHubInterface` 避免循环依赖
3. **统一错误处理** - 所有 Agent 通信使用一致的错误处理模式
4. **优雅降级** - Agent 离线时提供明确的错误信息

## 相关文件
- `internal/server/agent/agent_service.go` - Agent 消息处理
- `internal/server/agent/hub.go` - Agent 连接管理
- `internal/server/agent/agent_executor.go` - Agent 命令执行
- `internal/biz/inspection/agent.go` - Agent 能力接口
- `internal/biz/inspection/executor.go` - 拨测执行器
- `internal/server/asset/website_proxy.go` - 站点代理处理
- `internal/server/asset/http.go` - 路由注册
- `web/src/views/asset/Websites.vue` - 前端页面
- `web/src/api/website.ts` - 前端 API

## 更新日期
2026-03-09
