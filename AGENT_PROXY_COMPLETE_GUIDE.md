# Agent 代理完整逻辑总结

**日期**: 2026-04-15  
**版本**: 最终完整版  
**状态**: ✅ 完全可用

---

## 📋 概述

Agent 代理是一个完全透明的 HTTP 代理系统，通过 gRPC 让 Agent 转发 HTTP 请求，实现服务端访问内网资源。

**核心特点**：
- ✅ 完全透明（所有 HTTP 状态码、响应头、响应体原样传递）
- ✅ 无需认证（通过 ProxyToken 验证）
- ✅ 支持所有 HTTP 方法（GET、POST、PUT、DELETE 等）
- ✅ 支持静态文件（CSS、JS、图片等）
- ✅ 支持 gzip 压缩
- ✅ 支持认证（Bearer Token、Basic Auth）

---

## 🔄 完整的请求流程

### 1. 客户端发起请求

```
客户端（浏览器/Grafana/curl）
  ↓
GET http://opshub:9876/api/v1/alert/proxy/datasource/{token}/api/v1/query?query=up
```

### 2. 服务端接收请求

**文件**: `internal/server/alert/datasource_proxy_handler.go`

```go
func (s *HTTPServer) proxyDataSourceRequest(c *gin.Context) {
    // 1. 提取 ProxyToken
    proxyToken := c.Param("token")
    
    // 2. 查询数据源配置
    ds, err := s.dsRepo.GetByProxyToken(ctx, proxyToken)
    
    // 3. 验证访问模式
    if ds.AccessMode != "agent" {
        return error
    }
    
    // 4. 获取在线 Agent
    rels := s.dsAgentRelationRepo.ListByDataSourceID(ds.ID)
    selectedRel := 选择第一个在线的 Agent
    
    // 5. 构建目标 URL
    proxyPath := c.Request.URL.Path
    prefix := "/api/v1/alert/proxy/datasource/{token}"
    targetPath := strings.TrimPrefix(proxyPath, prefix)
    baseURL := strings.TrimRight(ds.URL, "/")
    targetURL := baseURL + targetPath + "?" + c.Request.URL.RawQuery
    
    // 6. 转发请求
    s.forwardToAgent(c, selectedRel.AgentHostID, targetURL, ds)
}
```

### 3. 构建 gRPC 请求

**文件**: `internal/server/alert/datasource_proxy_handler.go`

```go
func (s *HTTPServer) forwardToAgent(c *gin.Context, agentHostID uint, targetURL string, ds *biz.AlertDataSource) error {
    // 1. 获取 Agent 连接
    as, ok := s.agentHub.GetByHostID(agentHostID)
    
    // 2. 读取请求体
    body, err := io.ReadAll(c.Request.Body)
    
    // 3. 构建请求头
    headers := make(map[string]string)
    for key, values := range c.Request.Header {
        headers[key] = values[0]
    }
    
    // 4. 添加认证信息
    if ds.Token != "" {
        headers["Authorization"] = "Bearer " + ds.Token
    } else if ds.Username != "" {
        headers["Authorization"] = "Basic " + base64(username:password)
    }
    
    // 5. 生成请求 ID
    requestID := uuid.New().String()
    
    // 6. 构建 HttpProxyRequest
    proxyReq := &pb.HttpProxyRequest{
        RequestId: requestID,
        Method:    c.Request.Method,
        Url:       targetURL,
        Headers:   headers,
        Body:      body,
        Timeout:   30,
    }
    
    // 7. 发送给 Agent
    msg := &pb.ServerMessage{
        Payload: &pb.ServerMessage_HttpProxyRequest{
            HttpProxyRequest: proxyReq,
        },
    }
    as.Send(msg)
    
    // 8. 等待响应
    result, err := s.agentHub.WaitResponse(as, requestID, 35*time.Second)
    
    // 9. 返回响应
    proxyResp := result.(*pb.HttpProxyResponse)
    for key, value := range proxyResp.Headers {
        c.Header(key, value)
    }
    c.Status(int(proxyResp.StatusCode))
    c.Writer.Write(proxyResp.Body)
    
    return nil
}
```

### 4. Agent 接收并处理请求

**文件**: `agent/internal/client/grpc_client.go`

```go
func (c *GRPCClient) handleServerMessage(msg *pb.ServerMessage) {
    switch payload := msg.Payload.(type) {
    case *pb.ServerMessage_HttpProxyRequest:
        go c.handleHttpProxyRequest(payload.HttpProxyRequest)
    }
}

func (c *GRPCClient) handleHttpProxyRequest(req *pb.HttpProxyRequest) {
    // 1. 构建 HTTP 请求
    httpReq, err := http.NewRequest(req.Method, req.Url, bytes.NewReader(req.Body))
    
    // 2. 设置请求头
    for key, value := range req.Headers {
        httpReq.Header.Set(key, value)
    }
    
    // 3. 执行 HTTP 请求（访问内网资源）
    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Do(httpReq)
    
    // 4. 读取响应
    body, err := io.ReadAll(resp.Body)
    
    // 5. 构建响应头
    headers := make(map[string]string)
    for key, values := range resp.Header {
        headers[key] = values[0]
    }
    
    // 6. 发送响应
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
```

### 5. 服务端接收 Agent 响应

**文件**: `internal/server/agent/agent_service.go`

```go
func (s *AgentService) Connect(stream pb.AgentHub_ConnectServer) error {
    for {
        msg, err := stream.Recv()
        
        switch payload := msg.Payload.(type) {
        case *pb.AgentMessage_HttpProxyResponse:
            if as != nil {
                // 解析响应，唤醒等待的 WaitResponse
                as.ResolvePending(payload.HttpProxyResponse.RequestId, payload.HttpProxyResponse)
            }
        }
    }
}
```

### 6. 返回给客户端

响应已在步骤 3 中返回给客户端。

---

## 🔑 关键组件

### 1. ProxyToken

**生成**：
```go
proxyToken := uuid.New().String()
proxyURL := fmt.Sprintf("/api/v1/alert/proxy/datasource/%s", proxyToken)
```

**验证**：
```go
ds, err := s.dsRepo.GetByProxyToken(ctx, proxyToken)
```

### 2. AgentHub

**职责**：管理所有 Agent 连接

**关键方法**：
- `GetByHostID(hostID uint)` - 获取 Agent 连接
- `IsOnline(hostID uint)` - 检查 Agent 是否在线
- `WaitResponse(as, requestID, timeout)` - 等待响应

### 3. AgentStream

**职责**：封装单个 Agent 的 gRPC 双向流

**关键方法**：
- `Send(msg)` - 发送消息给 Agent
- `RegisterPending(requestID)` - 注册等待响应
- `ResolvePending(requestID, result)` - 解析响应

### 4. HttpProxyRequest/Response

**Proto 定义**：
```protobuf
message HttpProxyRequest {
  string request_id = 1;
  string method = 2;
  string url = 3;
  map<string, string> headers = 4;
  bytes body = 5;
  int32 timeout = 6;
}

message HttpProxyResponse {
  string request_id = 1;
  int32 status_code = 2;
  map<string, string> headers = 3;
  bytes body = 4;
  string error = 5;
}
```

---

## 🎯 关键设计原则

### 1. 完全透明

**原则**：代理应该像不存在一样

**实现**：
- 所有 HTTP 状态码原样返回（200、301、404、500 等）
- 所有响应头原样返回（Content-Type、Content-Encoding 等）
- 所有响应体原样返回（包括二进制数据）

### 2. 路径处理

**原则**：路径拼接必须正确

**实现**：
```go
// 避免双斜杠
baseURL := strings.TrimRight(ds.URL, "/")
targetURL := baseURL + targetPath
```

### 3. 错误区分

**原则**：区分 gRPC 错误和 HTTP 错误

**实现**：
```go
// gRPC 错误：Agent 未返回 HTTP 响应
if proxyResp.Error != "" && proxyResp.StatusCode == 0 {
    return fmt.Errorf("Agent 执行失败: %s", proxyResp.Error)
}

// HTTP 错误：原样返回给客户端
c.Status(int(proxyResp.StatusCode))
c.Writer.Write(proxyResp.Body)
return nil
```

### 4. 无需认证

**原则**：代理 URL 是公开的，通过 ProxyToken 验证

**实现**：
```go
// 注册为公开路由
func (s *HTTPServer) RegisterPublicRoutes(router *gin.Engine) {
    proxy := router.Group("/api/v1/alert/proxy/datasource")
    {
        proxy.Any("/:token/*path", s.proxyDataSourceRequest)
    }
}
```

---

## 🐛 已修复的问题

### 1. 路径拼接双斜杠

**问题**：`http://prometheus:9090//api/v1/query`

**修复**：`strings.TrimRight(ds.URL, "/")`

### 2. 代理路由认证

**问题**：返回 401 未登录

**修复**：将代理路由移到公开路由

### 3. 响应处理逻辑

**问题**：HTTP 错误响应被拦截

**修复**：区分 gRPC 错误和 HTTP 错误

### 4. 服务端消息处理

**问题**：服务端未处理 HttpProxyResponse

**修复**：在 switch 中添加 HttpProxyResponse 处理

### 5. Agent 真实转发

**问题**：服务端直接执行 HTTP 请求

**修复**：通过 gRPC 让 Agent 执行请求

---

## 📊 数据流图

```
┌─────────────────────────────────────────────────────────────────┐
│  客户端（浏览器/Grafana/curl）                                    │
│  GET /api/v1/alert/proxy/datasource/{token}/path?query          │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
        ┌────────────────────────────────────────────────┐
        │  代理处理器 (proxyDataSourceRequest)            │
        │  1. 提取 ProxyToken                             │
        │  2. 查询数据源配置                              │
        │  3. 验证访问模式                                │
        │  4. 选择在线 Agent                              │
        │  5. 构建目标 URL                                │
        └────────────────────┬──────────────────────────┘
                             │
                             ▼
        ┌────────────────────────────────────────────────┐
        │  forwardToAgent()                               │
        │  1. 获取 AgentStream                            │
        │  2. 构建 HttpProxyRequest                      │
        │  3. 发送给 Agent (as.Send)                     │
        │  4. 等待响应 (WaitResponse)                    │
        └────────────────────┬──────────────────────────┘
                             │
                             ▼ gRPC 双向流
        ┌────────────────────────────────────────────────┐
        │  Agent 端 (handleHttpProxyRequest)             │
        │  1. 接收 HttpProxyRequest                      │
        │  2. 构建 HTTP 请求                             │
        │  3. 执行请求（访问内网资源）                    │
        │  4. 读取响应                                    │
        │  5. 发送 HttpProxyResponse                     │
        └────────────────────┬──────────────────────────┘
                             │
                             ▼
        ┌────────────────────────────────────────────────┐
        │  内网资源                                        │
        │  (Prometheus/Web站点/API)                       │
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
        │  服务端 (agent_service.go)                     │
        │  1. 接收 HttpProxyResponse                     │
        │  2. ResolvePending(requestID, response)        │
        │  3. WaitResponse 返回                          │
        └────────────────────┬──────────────────────────┘
                             │
                             ▼
        ┌────────────────────────────────────────────────┐
        │  forwardToAgent()                               │
        │  1. 设置响应头                                  │
        │  2. 设置状态码                                  │
        │  3. 写入响应体                                  │
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

## 🎯 应用场景

### 1. Grafana 数据源

**配置**：
```
数据源类型：Prometheus
URL：http://opshub:9876/api/v1/alert/proxy/datasource/{token}
```

**效果**：
- ✅ 可以查询指标
- ✅ 可以创建仪表板
- ✅ 可以设置告警

### 2. 告警规则查询

**配置**：
```go
datasource := &AlertDataSource{
    AccessMode: "agent",
    URL: "http://prometheus:9090",
}
```

**效果**：
- ✅ 告警规则可以查询数据源
- ✅ 评估告警条件
- ✅ 触发告警事件

### 3. Web 站点代理

**配置**：
```go
site := &WebSite{
    AccessMode: "agent",
    URL: "http://internal-site:8080",
}
```

**效果**：
- ✅ 可以访问内网 Web 站点
- ✅ 静态文件正常加载
- ✅ 完全透明的代理

---

## 📝 开发指南

### 如何添加新的代理功能

1. **创建代理处理器**
   ```go
   func (s *HTTPServer) proxyRequest(c *gin.Context) {
       // 1. 提取 ProxyToken
       // 2. 查询配置
       // 3. 选择 Agent
       // 4. 构建目标 URL
       // 5. 调用 forwardToAgent
   }
   ```

2. **注册公开路由**
   ```go
   func (s *HTTPServer) RegisterPublicRoutes(router *gin.Engine) {
       proxy := router.Group("/api/v1/xxx/proxy")
       {
           proxy.Any("/:token/*path", s.proxyRequest)
       }
   }
   ```

3. **调用公开路由注册**
   ```go
   // internal/server/http.go
   s.xxxServer.RegisterPublicRoutes(router)
   ```

### 关键注意事项

1. **路径处理**
   - 使用 `strings.TrimRight(baseURL, "/")`
   - 使用 `strings.TrimPrefix(path, prefix)`

2. **响应处理**
   - 区分 gRPC 错误和 HTTP 错误
   - 所有 HTTP 响应原样返回
   - 成功写入响应后返回 nil

3. **消息处理**
   - 在 `agent_service.go` 的 switch 中添加处理
   - 调用 `as.ResolvePending(requestID, response)`

4. **无需认证**
   - 代理路由注册为公开路由
   - 通过 ProxyToken 验证安全性

---

## 🎉 总结

Agent 代理是一个完全透明的 HTTP 代理系统，核心特点：

1. ✅ 完全透明（所有响应原样传递）
2. ✅ 真实转发（通过 gRPC 让 Agent 执行）
3. ✅ 无需认证（通过 ProxyToken 验证）
4. ✅ 支持所有 HTTP 特性（方法、状态码、响应头、响应体）
5. ✅ 支持静态文件（CSS、JS、图片等）

**关键文件**：
- `internal/server/alert/datasource_proxy_handler.go` - 代理处理器
- `internal/server/agent/agent_service.go` - 消息处理
- `agent/internal/client/grpc_client.go` - Agent 端处理
- `pkg/agentproto/agent.proto` - Proto 定义

