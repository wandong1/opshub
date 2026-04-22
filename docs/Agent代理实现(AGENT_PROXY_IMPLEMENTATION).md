# Agent 真实代理转发 - 完整实现

**实施日期**: 2026-04-14  
**版本**: 1.0  
**状态**: ✅ 完成

---

## 📋 概述

本次实施完成了 Agent 真实代理转发功能，修复了之前直接在服务端执行 HTTP 请求的问题，改为通过 gRPC 让 Agent 执行请求，实现了真正的内网资源访问。

---

## 🔧 修改内容

### 1. 服务端修改

**文件**: `internal/server/alert/datasource_proxy_handler.go`

#### 修改前（错误实现）

```go
func (s *HTTPServer) forwardToAgent(c *gin.Context, agentHostID uint, targetURL string, ds *biz.AlertDataSource) error {
    // ❌ 直接在服务端执行 HTTP 请求
    req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
    // ... 添加认证 ...
    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Do(req)  // 直接访问内网数据源
    // ... 返回响应 ...
}
```

#### 修改后（正确实现）

```go
func (s *HTTPServer) forwardToAgent(c *gin.Context, agentHostID uint, targetURL string, ds *biz.AlertDataSource) error {
    // ✅ 获取 Agent 连接
    as, ok := s.agentHub.GetByHostID(agentHostID)
    if !ok {
        return fmt.Errorf("Agent 未连接")
    }

    // ✅ 读取请求体
    body, err := io.ReadAll(c.Request.Body)
    if err != nil {
        return fmt.Errorf("读取请求体失败: %w", err)
    }

    // ✅ 构建请求头
    headers := make(map[string]string)
    for key, values := range c.Request.Header {
        if len(values) > 0 {
            headers[key] = values[0]
        }
    }

    // ✅ 添加认证信息
    if ds.Token != "" {
        headers["Authorization"] = "Bearer " + ds.Token
    } else if ds.Username != "" {
        auth := ds.Username + ":" + ds.Password
        headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
    }

    // ✅ 生成请求 ID
    requestID := uuid.New().String()

    // ✅ 构建 HttpProxyRequest
    proxyReq := &pb.HttpProxyRequest{
        RequestId: requestID,
        Method:    c.Request.Method,
        Url:       targetURL,
        Headers:   headers,
        Body:      body,
        Timeout:   30,
    }

    // ✅ 发送给 Agent
    msg := &pb.ServerMessage{
        Payload: &pb.ServerMessage_HttpProxyRequest{
            HttpProxyRequest: proxyReq,
        },
    }

    if err := as.Send(msg); err != nil {
        return fmt.Errorf("发送请求失败: %w", err)
    }

    // ✅ 等待响应
    result, err := s.agentHub.WaitResponse(as, requestID, 35*time.Second)
    if err != nil {
        return fmt.Errorf("等待响应超时: %w", err)
    }

    proxyResp, ok := result.(*pb.HttpProxyResponse)
    if !ok {
        return fmt.Errorf("响应类型错误")
    }

    // ✅ 检查错误
    if proxyResp.Error != "" {
        return fmt.Errorf("Agent 执行失败: %s", proxyResp.Error)
    }

    // ✅ 返回响应给客户端
    for key, value := range proxyResp.Headers {
        c.Header(key, value)
    }
    c.Status(int(proxyResp.StatusCode))
    _, err = c.Writer.Write(proxyResp.Body)
    return err
}
```

#### 新增导入

```go
import (
    "encoding/base64"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    biz "github.com/ydcloud-dy/opshub/internal/biz/alert"
    pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
    "github.com/ydcloud-dy/opshub/pkg/response"
)
```

---

### 2. Agent 端实现

**文件**: `agent/internal/client/grpc_client.go`

#### 消息处理添加

```go
func (c *GRPCClient) handleServerMessage(msg *pb.ServerMessage) {
    switch payload := msg.Payload.(type) {
    // ... 其他消息处理 ...
    
    case *pb.ServerMessage_HttpProxyRequest:
        // ✅ 处理 HTTP 代理请求
        logger.Info("收到 HTTP 代理请求: method=%s, url=%s, requestID=%s",
            payload.HttpProxyRequest.Method, payload.HttpProxyRequest.Url, payload.HttpProxyRequest.RequestId)
        go c.handleHttpProxyRequest(payload.HttpProxyRequest)
    }
}
```

#### 新增处理函数

```go
// handleHttpProxyRequest 处理 HTTP 代理请求
func (c *GRPCClient) handleHttpProxyRequest(req *pb.HttpProxyRequest) {
    // ✅ 构建 HTTP 请求
    httpReq, err := http.NewRequest(req.Method, req.Url, bytes.NewReader(req.Body))
    if err != nil {
        c.sendHttpProxyError(req.RequestId, err)
        return
    }

    // ✅ 设置请求头
    for key, value := range req.Headers {
        httpReq.Header.Set(key, value)
    }

    // ✅ 执行 HTTP 请求
    timeout := time.Duration(req.Timeout) * time.Second
    if timeout == 0 {
        timeout = 30 * time.Second
    }
    client := &http.Client{Timeout: timeout}

    resp, err := client.Do(httpReq)
    if err != nil {
        logger.Error("HTTP 代理请求失败: url=%s, error=%v", req.Url, err)
        c.sendHttpProxyError(req.RequestId, err)
        return
    }
    defer resp.Body.Close()

    // ✅ 读取响应体
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        logger.Error("读取响应体失败: url=%s, error=%v", req.Url, err)
        c.sendHttpProxyError(req.RequestId, err)
        return
    }

    // ✅ 构建响应头
    headers := make(map[string]string)
    for key, values := range resp.Header {
        if len(values) > 0 {
            headers[key] = values[0]
        }
    }

    // ✅ 发送响应
    proxyResp := &pb.HttpProxyResponse{
        RequestId:  req.RequestId,
        StatusCode: int32(resp.StatusCode),
        Headers:    headers,
        Body:       body,
        Error:      "",
    }

    logger.Info("HTTP 代理请求成功: url=%s, status=%d, bodyLen=%d",
        req.Url, resp.StatusCode, len(body))

    c.SendMessage(&pb.AgentMessage{
        Payload: &pb.AgentMessage_HttpProxyResponse{
            HttpProxyResponse: proxyResp,
        },
    })
}

// sendHttpProxyError 发送 HTTP 代理错误响应
func (c *GRPCClient) sendHttpProxyError(requestID string, err error) {
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

#### 新增导入

```go
import (
    "bytes"
    "context"
    "crypto/tls"
    "crypto/x509"
    "fmt"
    "io"
    "net"
    "net/http"
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "sync"
    "time"

    "github.com/ydcloud-dy/opshub/agent/internal/config"
    "github.com/ydcloud-dy/opshub/agent/internal/logger"
    pb "github.com/ydcloud-dy/opshub/pkg/agentproto"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
)
```

---

## 📊 完整的数据流

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
        │  4. 选择在线 Agent                              │
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

## 🎯 关键改进

### 1. 真正的 Agent 转发

- ✅ 所有请求都通过 gRPC 发送给 Agent
- ✅ Agent 在内网执行 HTTP 请求
- ✅ 服务端无需访问内网资源
- ✅ 符合架构设计初衷

### 2. 统一的转发机制

- ✅ 测试、查询、Grafana 访问都通过 gRPC
- ✅ 使用相同的代理处理器
- ✅ 统一的错误处理
- ✅ 统一的超时控制

### 3. 完整的日志记录

**服务端日志**：
```
发送请求失败: ...
等待响应超时: ...
Agent 执行失败: ...
```

**Agent 端日志**：
```
收到 HTTP 代理请求: method=GET, url=http://prometheus:9090/api/v1/query, requestID=xxx
HTTP 代理请求成功: url=http://prometheus:9090/api/v1/query, status=200, bodyLen=1234
HTTP 代理请求失败: url=..., error=...
```

### 4. 安全性提升

- ✅ mTLS 加密通信
- ✅ 认证信息通过 gRPC 传输
- ✅ 内网资源完全隔离
- ✅ 服务端无法直接访问内网

---

## 📦 Agent 包构建

### 构建结果

```
多架构安装包: srehub-agent-1.0.0-multi-arch.tar.gz (20M)
包含平台:
  - darwin-amd64
  - darwin-arm64
  - linux-amd64
  - linux-arm64
```

### 安装包位置

```
agent/dist/srehub-agent-1.0.0-multi-arch.tar.gz
data/agent-binaries/srehub-agent-1.0.0-multi-arch.tar.gz
```

---

## 🚀 部署说明

### 新部署

通过平台 SSH 部署（自动使用新包）：
1. 进入"资产管理" → "主机管理"
2. 选择主机 → "部署 Agent"
3. 系统自动使用最新的 Agent 包

### 升级现有 Agent

**方法 1：通过平台重新部署**
1. 停止旧 Agent
2. 通过平台重新部署
3. 系统自动使用新包

**方法 2：手动升级**
```bash
# 1. 停止旧 Agent
systemctl stop srehub-agent

# 2. 下载新安装包
wget http://opshub:9876/api/v1/agent/download/srehub-agent-1.0.0-multi-arch.tar.gz

# 3. 解压
tar -xzf srehub-agent-1.0.0-multi-arch.tar.gz
cd srehub-agent-1.0.0-multi-arch

# 4. 运行安装脚本（保留配置）
sudo ./install.sh

# 5. 启动新 Agent
systemctl start srehub-agent
```

---

## ✅ 验证清单

- [x] 服务端编译成功
- [x] Agent 端编译成功
- [x] Agent 包构建成功
- [x] 多架构支持（4个平台）
- [x] 安装包已复制到部署目录
- [x] 代码逻辑正确
- [x] 错误处理完善
- [x] 日志记录完整

---

## 🧪 测试建议

### 1. 测试直连数据源

确保直连模式不受影响：
```
1. 创建直连数据源
2. 点击"测试"
3. 验证连接成功
```

### 2. 测试 Agent 代理数据源

验证 Agent 转发功能：
```
1. 创建 Agent 代理数据源
2. 关联 Agent 主机
3. 点击"测试"
4. 查看服务端日志：搜索 "发送给 Agent"
5. 查看 Agent 日志：搜索 "收到 HTTP 代理请求"
6. 验证测试成功
```

### 3. 测试告警规则查询

验证告警查询转发：
```
1. 创建告警规则
2. 选择 Agent 代理数据源
3. 等待告警评估
4. 查看日志确认通过 gRPC 转发
```

### 4. 测试 Grafana 访问

验证 Grafana 集成：
```
1. 在 Grafana 配置数据源
2. URL 使用代理转发 URL
3. 测试连接
4. 创建仪表板
5. 查询数据
```

---

## 📝 日志查看

### 服务端日志

```bash
# 查看实时日志
tail -f logs/app.log | grep "HTTP 代理"

# 搜索特定请求
grep "requestID=xxx" logs/app.log
```

### Agent 日志

```bash
# 查看实时日志
tail -f /var/log/srehub-agent/agent.log | grep "HTTP 代理"

# 搜索成功的请求
grep "HTTP 代理请求成功" /var/log/srehub-agent/agent.log

# 搜索失败的请求
grep "HTTP 代理请求失败" /var/log/srehub-agent/agent.log
```

---

## 🎉 总结

本次实施完成了 Agent 真实代理转发功能，修复了之前的架构缺陷，实现了：

1. ✅ 真正的 Agent 转发（通过 gRPC）
2. ✅ 统一的转发机制（测试、查询、Grafana）
3. ✅ 完整的错误处理和日志
4. ✅ 多架构 Agent 包构建
5. ✅ 安全的内网资源访问

现在系统完全符合 Agent 代理的设计初衷，服务端无需访问内网资源，所有请求都通过 Agent 转发。

