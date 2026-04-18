# Agent 代理转发机制详解

**文档日期**: 2026-04-14  
**版本**: 1.0  

---

## 📋 概述

本文档详细说明数据源通过 Agent 进行代理转发的完整机制，包括当前实现和应该的实现方式。

---

## 🔍 当前实现分析

### 1. 请求入口

**文件**: `internal/server/alert/datasource_proxy_handler.go`

```go
// 路由注册
proxy := alert.Group("/proxy/datasource")
{
    proxy.Any("/:token/*path", s.proxyDataSourceRequest)
}

// 请求格式
GET /api/v1/alert/proxy/datasource/{token}/api/v1/query?query=up
```

### 2. 代理处理器流程

```go
func (s *HTTPServer) proxyDataSourceRequest(c *gin.Context) {
    // 步骤 1: 提取 ProxyToken
    proxyToken := c.Param("token")
    
    // 步骤 2: 查询数据源配置
    ds, err := s.dsRepo.GetByProxyToken(ctx, proxyToken)
    // 验证: ds.AccessMode == "agent"
    
    // 步骤 3: 获取关联的 Agent 列表
    rels, err := s.dsAgentRelationRepo.ListByDataSourceID(ctx, ds.ID)
    
    // 步骤 4: 选择在线 Agent
    var selectedRel *biz.DataSourceAgentRelation
    for _, rel := range rels {
        if s.agentHub.IsOnline(rel.AgentHostID) {
            selectedRel = rel
            break
        }
    }
    
    // 步骤 5: 构建目标 URL
    proxyPath := c.Request.URL.Path
    prefix := fmt.Sprintf("/api/v1/alert/proxy/datasource/%s", proxyToken)
    targetPath := strings.TrimPrefix(proxyPath, prefix)
    targetURL := ds.URL + targetPath
    if c.Request.URL.RawQuery != "" {
        targetURL += "?" + c.Request.URL.RawQuery
    }
    
    // 步骤 6: 转发请求
    if err := s.forwardToAgent(c, selectedRel.AgentHostID, targetURL, ds); err != nil {
        response.ErrorCode(c, http.StatusBadGateway, "转发请求失败: "+err.Error())
        return
    }
}
```

### 3. 当前的 forwardToAgent 实现

```go
func (s *HTTPServer) forwardToAgent(c *gin.Context, agentHostID uint, targetURL string, ds *biz.AlertDataSource) error {
    // ❌ 问题：直接在服务端执行 HTTP 请求
    
    // 构建请求
    req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
    
    // 复制请求头
    for key, values := range c.Request.Header {
        for _, value := range values {
            req.Header.Add(key, value)
        }
    }
    
    // 添加认证信息（从数据源配置）
    if ds.Token != "" {
        req.Header.Set("Authorization", "Bearer "+ds.Token)
    } else if ds.Username != "" {
        req.SetBasicAuth(ds.Username, ds.Password)
    }
    
    // 执行请求（直接访问内网数据源）
    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // 复制响应
    for key, values := range resp.Header {
        for _, value := range values {
            c.Header(key, value)
        }
    }
    c.Status(resp.StatusCode)
    io.Copy(c.Writer, resp.Body)
    
    return nil
}
```

**问题**：
- ❌ 在服务端直接执行 HTTP 请求
- ❌ 服务端需要能访问内网数据源
- ❌ 没有真正通过 Agent 转发
- ❌ 违反了 Agent 代理的设计初衷

---

## ✅ 应该的实现方式

### 1. AgentHub 架构

**文件**: `internal/server/agent/hub.go`

```go
type AgentHub struct {
    mu            sync.RWMutex
    byAgentID     map[string]*AgentStream  // AgentID → Stream
    byHostID      map[uint]*AgentStream    // HostID → Stream
    termCallbacks map[string]func(data []byte)
    termMu        sync.RWMutex
}

type AgentStream struct {
    AgentID  string
    HostID   uint
    Stream   pb.AgentHub_ConnectServer  // gRPC 双向流
    SendCh   chan *pb.ServerMessage     // 发送消息通道
    DoneCh   chan struct{}              // 关闭信号
    pending  map[string]chan any        // 请求ID → 响应通道
    pendMu   sync.Mutex
}
```

**关键方法**：
- `Register(as *AgentStream)`: 注册 Agent 连接
- `GetByHostID(hostID uint)`: 根据主机 ID 获取 Agent 流
- `IsOnline(hostID uint)`: 检查 Agent 是否在线
- `WaitResponse(as *AgentStream, requestID string, timeout time.Duration)`: 等待响应

### 2. gRPC Proto 定义

**文件**: `api/proto/agent.proto`

```protobuf
// Server → Agent 消息
message ServerMessage {
  oneof payload {
    RegisterAck register_ack = 1;
    TermOpenRequest term_open = 2;
    TermInput term_input = 3;
    TermResize term_resize = 4;
    TermClose term_close = 5;
    FileRequest file_request = 6;
    CmdRequest cmd_request = 7;
    ProbeRequest probe_request = 9;
    HttpProxyRequest http_proxy_request = 10;  // ✅ HTTP 代理请求
  }
}

// Agent → Server 消息
message AgentMessage {
  oneof payload {
    RegisterRequest register = 1;
    HeartbeatRequest heartbeat = 2;
    TermOutput term_output = 3;
    FileChunk file_chunk = 4;
    FileListResult file_list = 5;
    CommandResult cmd_result = 6;
    ProbeResult probe_result = 7;
    HttpProxyResponse http_proxy_response = 8;  // ✅ HTTP 代理响应
  }
}

// HTTP 代理请求
message HttpProxyRequest {
  string request_id = 1;      // 请求 ID（用于匹配响应）
  string method = 2;          // HTTP 方法（GET/POST/PUT/DELETE）
  string url = 3;             // 目标 URL
  map<string, string> headers = 4;  // 请求头
  bytes body = 5;             // 请求体
  int32 timeout = 6;          // 超时时间（秒）
}

// HTTP 代理响应
message HttpProxyResponse {
  string request_id = 1;      // 请求 ID（匹配请求）
  int32 status_code = 2;      // HTTP 状态码
  map<string, string> headers = 3;  // 响应头
  bytes body = 4;             // 响应体
  string error = 5;           // 错误信息（如果有）
}
```

### 3. 正确的 forwardToAgent 实现

```go
func (s *HTTPServer) forwardToAgent(c *gin.Context, agentHostID uint, targetURL string, ds *biz.AlertDataSource) error {
    // ✅ 步骤 1: 获取 Agent 连接
    as, ok := s.agentHub.GetByHostID(agentHostID)
    if !ok {
        return fmt.Errorf("Agent 未连接")
    }
    
    // ✅ 步骤 2: 读取请求体
    body, err := io.ReadAll(c.Request.Body)
    if err != nil {
        return err
    }
    
    // ✅ 步骤 3: 构建 HttpProxyRequest
    requestID := uuid.New().String()
    headers := make(map[string]string)
    for key, values := range c.Request.Header {
        if len(values) > 0 {
            headers[key] = values[0]
        }
    }
    
    // 添加认证信息
    if ds.Token != "" {
        headers["Authorization"] = "Bearer " + ds.Token
    } else if ds.Username != "" {
        auth := base64.StdEncoding.EncodeToString([]byte(ds.Username + ":" + ds.Password))
        headers["Authorization"] = "Basic " + auth
    }
    
    proxyReq := &pb.HttpProxyRequest{
        RequestId: requestID,
        Method:    c.Request.Method,
        Url:       targetURL,
        Headers:   headers,
        Body:      body,
        Timeout:   30,
    }
    
    // ✅ 步骤 4: 发送给 Agent
    msg := &pb.ServerMessage{
        Payload: &pb.ServerMessage_HttpProxyRequest{
            HttpProxyRequest: proxyReq,
        },
    }
    
    if err := as.Send(msg); err != nil {
        return fmt.Errorf("发送请求失败: %w", err)
    }
    
    // ✅ 步骤 5: 等待响应
    result, err := s.agentHub.WaitResponse(as, requestID, 35*time.Second)
    if err != nil {
        return fmt.Errorf("等待响应超时: %w", err)
    }
    
    proxyResp, ok := result.(*pb.HttpProxyResponse)
    if !ok {
        return fmt.Errorf("响应类型错误")
    }
    
    // ✅ 步骤 6: 检查错误
    if proxyResp.Error != "" {
        return fmt.Errorf("Agent 执行失败: %s", proxyResp.Error)
    }
    
    // ✅ 步骤 7: 返回响应给客户端
    for key, value := range proxyResp.Headers {
        c.Header(key, value)
    }
    c.Status(int(proxyResp.StatusCode))
    c.Writer.Write(proxyResp.Body)
    
    return nil
}
```

### 4. Agent 端实现

**文件**: `agent/internal/client/grpc_client.go`

```go
func (c *Client) handleServerMessage(msg *pb.ServerMessage) {
    switch payload := msg.Payload.(type) {
    
    // ... 其他消息处理 ...
    
    case *pb.ServerMessage_HttpProxyRequest:
        // ✅ 处理 HTTP 代理请求
        go c.handleHttpProxyRequest(payload.HttpProxyRequest)
    }
}

func (c *Client) handleHttpProxyRequest(req *pb.HttpProxyRequest) {
    // ✅ 步骤 1: 构建 HTTP 请求
    httpReq, err := http.NewRequest(req.Method, req.Url, bytes.NewReader(req.Body))
    if err != nil {
        c.sendHttpProxyError(req.RequestId, err)
        return
    }
    
    // ✅ 步骤 2: 设置请求头
    for key, value := range req.Headers {
        httpReq.Header.Set(key, value)
    }
    
    // ✅ 步骤 3: 执行 HTTP 请求（访问内网数据源）
    timeout := time.Duration(req.Timeout) * time.Second
    if timeout == 0 {
        timeout = 30 * time.Second
    }
    client := &http.Client{Timeout: timeout}
    
    resp, err := client.Do(httpReq)
    if err != nil {
        c.sendHttpProxyError(req.RequestId, err)
        return
    }
    defer resp.Body.Close()
    
    // ✅ 步骤 4: 读取响应
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        c.sendHttpProxyError(req.RequestId, err)
        return
    }
    
    // ✅ 步骤 5: 构建响应头
    headers := make(map[string]string)
    for key, values := range resp.Header {
        if len(values) > 0 {
            headers[key] = values[0]
        }
    }
    
    // ✅ 步骤 6: 发送响应给服务端
    proxyResp := &pb.HttpProxyResponse{
        RequestId:  req.RequestId,
        StatusCode: int32(resp.StatusCode),
        Headers:    headers,
        Body:       body,
        Error:      "",
    }
    
    c.SendMessage(&pb.AgentMessage{
        Payload: &pb.AgentMessage_HttpProxyResponse{
            HttpProxyResponse: proxyResp,
        },
    })
}

func (c *Client) sendHttpProxyError(requestID string, err error) {
    proxyResp := &pb.HttpProxyResponse{
        RequestId:  requestID,
        StatusCode: 500,
        Error:      err.Error(),
    }
    
    c.SendMessage(&pb.AgentMessage{
        Payload: &pb.AgentMessage_HttpProxyResponse{
            HttpProxyResponse: proxyResp,
        },
    })
}
```

---

## 📊 完整的数据流

### 正确的 Agent 代理转发流程

```
┌─────────────────────────────────────────────────────────────────┐
│  客户端（Grafana/测试/告警查询）                                   │
│  GET /api/v1/alert/proxy/datasource/{token}/api/v1/query        │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
        ┌────────────────────────────────────────────────┐
        │  代理处理器 (proxyDataSourceRequest)            │
        │  1. 查询数据源 (by ProxyToken)                  │
        │  2. 验证 AccessMode == "agent"                 │
        │  3. 获取关联的 Agent 列表                       │
        │  4. 选择在线 Agent (agentHub.IsOnline)        │
        └────────────────────┬──────────────────────────┘
                             │
                             ▼
        ┌────────────────────────────────────────────────┐
        │  forwardToAgent()                               │
        │  1. 获取 AgentStream (agentHub.GetByHostID)    │
        │  2. 构建 HttpProxyRequest                      │
        │  3. 发送给 Agent (as.Send)                     │
        │  4. 等待响应 (agentHub.WaitResponse)           │
        └────────────────────┬──────────────────────────┘
                             │
                             ▼ gRPC 双向流
        ┌────────────────────────────────────────────────┐
        │  Agent 端 (handleHttpProxyRequest)             │
        │  1. 接收 HttpProxyRequest                      │
        │  2. 构建 HTTP 请求                             │
        │  3. 执行请求（访问内网数据源）                  │
        │  4. 读取响应                                    │
        │  5. 发送 HttpProxyResponse                     │
        └────────────────────┬──────────────────────────┘
                             │
                             ▼
        ┌────────────────────────────────────────────────┐
        │  内网数据源                                      │
        │  (Prometheus/VictoriaMetrics/InfluxDB)         │
        │  - Agent 可以访问                               │
        │  - 服务端无法直接访问                           │
        └────────────────────┬──────────────────────────┘
                             │
                             ▼ HTTP 响应
        ┌────────────────────────────────────────────────┐
        │  Agent 端                                       │
        │  构建 HttpProxyResponse                         │
        └────────────────────┬──────────────────────────┘
                             │
                             ▼ gRPC 双向流
        ┌────────────────────────────────────────────────┐
        │  服务端                                          │
        │  1. 接收 HttpProxyResponse                     │
        │  2. 解析响应                                    │
        │  3. 返回给客户端                                │
        └────────────────────┬──────────────────────────┘
                             │
                             ▼
        ┌────────────────────────────────────────────────┐
        │  客户端收到响应                                  │
        │  - 状态码                                       │
        │  - 响应头                                       │
        │  - 响应体                                       │
        └────────────────────────────────────────────────┘
```

---

## 🔑 关键组件

### 1. AgentHub（服务端）

**职责**：
- 管理所有 Agent 连接
- 提供 Agent 查询接口
- 处理请求-响应匹配

**关键方法**：
```go
Register(as *AgentStream)                                    // 注册 Agent
Unregister(agentID string)                                   // 注销 Agent
GetByHostID(hostID uint) (*AgentStream, bool)               // 获取 Agent
IsOnline(hostID uint) bool                                   // 检查在线
WaitResponse(as *AgentStream, requestID string, timeout)    // 等待响应
```

### 2. AgentStream（服务端）

**职责**：
- 封装 gRPC 双向流
- 管理消息发送
- 处理请求-响应匹配

**关键方法**：
```go
Send(msg *pb.ServerMessage) error                           // 发送消息
RegisterPending(requestID string) chan any                  // 注册等待
ResolvePending(requestID string, result any)                // 完成请求
```

### 3. gRPC Client（Agent 端）

**职责**：
- 连接服务端
- 接收服务端消息
- 执行 HTTP 代理请求
- 返回响应

**关键方法**：
```go
Connect() error                                             // 连接服务端
handleServerMessage(msg *pb.ServerMessage)                  // 处理消息
handleHttpProxyRequest(req *pb.HttpProxyRequest)           // 处理代理请求
SendMessage(msg *pb.AgentMessage) error                    // 发送消息
```

---

## 📝 实现步骤

### 第一步：服务端实现

1. ✅ 修改 `forwardToAgent()` 使用 gRPC
2. ✅ 实现请求-响应匹配机制
3. ✅ 添加超时处理
4. ✅ 添加错误处理

### 第二步：Agent 端实现

1. ✅ 添加 `handleHttpProxyRequest()` 处理器
2. ✅ 实现 HTTP 请求执行
3. ✅ 实现响应返回
4. ✅ 添加错误处理

### 第三步：测试验证

1. ✅ 测试直连模式（不受影响）
2. ✅ 测试 Agent 代理模式
3. ✅ 测试超时处理
4. ✅ 测试错误处理
5. ✅ 测试并发请求

---

## 🎯 优势

### 真正的 Agent 代理

- ✅ 服务端无需访问内网
- ✅ 通过 Agent 访问内网资源
- ✅ 符合架构设计

### 统一的转发机制

- ✅ 测试、查询、Grafana 访问都通过 gRPC
- ✅ 统一的错误处理
- ✅ 统一的超时控制

### 安全性

- ✅ mTLS 加密通信
- ✅ 认证信息不暴露
- ✅ 内网资源隔离

---

## 📚 相关文件

| 组件 | 文件路径 |
|------|---------|
| 代理处理器 | `internal/server/alert/datasource_proxy_handler.go` |
| AgentHub | `internal/server/agent/hub.go` |
| gRPC 服务端 | `internal/server/agent/grpc_server.go` |
| Agent 服务 | `internal/server/agent/agent_service.go` |
| gRPC 客户端 | `agent/internal/client/grpc_client.go` |
| Proto 定义 | `api/proto/agent.proto` |
| 数据源关联 | `internal/biz/alert/datasource_agent_relation.go` |

