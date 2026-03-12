# Agent 拨测功能实现状态

## 当前进度

### ✅ 已完成

1. **Protobuf 协议扩展** (`api/proto/agent.proto`)
   - 新增 `ProbeRequest` 消息（服务端 → Agent）
   - 新增 `ProbeResult` 消息（Agent → 服务端）
   - 新增 `ProbeAssertion` 和 `ProbeAssertionResult`
   - 在 `ServerMessage` 和 `AgentMessage` 中添加拨测消息类型

2. **Agent 端拨测器实现** (`agent/internal/prober/`)
   - ✅ `prober.go` - 拨测管理器和接口定义
   - ✅ `ping.go` - Ping 拨测器（使用系统 ping 命令）
   - ✅ `tcp.go` - TCP 拨测器（Go 原生 net.Dial）
   - ✅ `udp.go` - UDP 拨测器（Go 原生 UDP）
   - ✅ `http.go` - HTTP/HTTPS 拨测器（Go 原生 net/http + httptrace）
   - ✅ `websocket.go` - WebSocket 拨测器（gorilla/websocket）

3. **Agent gRPC 客户端集成** (`agent/internal/client/grpc_client.go`)
   - ✅ 新增 `ProbeHandler` 接口
   - ✅ 在 `handleServerMessage` 中处理 `ProbeRequest`
   - ✅ 添加 `SetProbeHandler` 方法

4. **Agent 主程序集成** (`agent/cmd/main.go`)
   - ✅ 导入 prober 包
   - ✅ 初始化拨测管理器
   - ✅ 注入到 gRPC 客户端

### ⏳ 待完成

1. **生成 Protobuf 代码**
   ```bash
   # 需要先安装 protoc
   brew install protobuf  # macOS

   # 安装 Go 插件
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

   # 生成代码
   protoc --go_out=. --go_opt=paths=source_relative \
          --go-grpc_out=. --go-grpc_opt=paths=source_relative \
          api/proto/agent.proto
   ```

2. **服务端 AgentHub 扩展** (`internal/server/agent/hub.go`)
   - ⏳ 实现 `SendProbeRequest(hostID uint, req *pb.ProbeRequest) *pb.ProbeResult` 方法
   - ⏳ 使用现有的请求-响应机制等待拨测结果

3. **服务端 Executor 改造** (`internal/biz/inspection/executor.go`)
   - ⏳ 删除旧的 `agent_app_prober.go` 实现
   - ⏳ 在 `executeAppProbe` 中调用 `AgentHub.SendProbeRequest`
   - ⏳ 实现 `pb.ProbeResult` 到 `probers.AppResult` 的转换函数

4. **AgentCommandFactory 扩展** (`internal/biz/inspection/agent.go`)
   - ⏳ 在接口中添加 `SendProbeRequest` 方法
   - ⏳ 在实现中调用 `AgentHub.SendProbeRequest`

## 下一步操作

### 步骤 1：安装 protoc 并生成代码

```bash
# 安装 protoc（如果未安装）
brew install protobuf

# 安装 Go 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 生成 protobuf 代码
cd /Users/Zhuanz/golang_project/src/opshub
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       api/proto/agent.proto
```

### 步骤 2：实现服务端 AgentHub 扩展

在 `internal/server/agent/hub.go` 中添加：

```go
// SendProbeRequest 发送拨测请求到指定 Agent 并等待结果
func (h *AgentHub) SendProbeRequest(hostID uint, req *pb.ProbeRequest) *pb.ProbeResult {
    as := h.GetByHostID(hostID)
    if as == nil {
        return &pb.ProbeResult{
            RequestId: req.RequestId,
            Success:   false,
            Error:     "agent not found",
        }
    }

    // 发送拨测请求
    serverMsg := &pb.ServerMessage{
        Payload: &pb.ServerMessage_ProbeRequest{
            ProbeRequest: req,
        },
    }

    if err := as.Send(serverMsg); err != nil {
        return &pb.ProbeResult{
            RequestId: req.RequestId,
            Success:   false,
            Error:     fmt.Sprintf("send probe request failed: %v", err),
        }
    }

    // 等待响应（30秒超时）
    timeout := 30 * time.Second
    if req.Timeout > 0 {
        timeout = time.Duration(req.Timeout+5) * time.Second // 额外 5 秒缓冲
    }

    resp, err := h.WaitResponse(as, req.RequestId, timeout)
    if err != nil {
        return &pb.ProbeResult{
            RequestId: req.RequestId,
            Success:   false,
            Error:     fmt.Sprintf("wait probe result timeout: %v", err),
        }
    }

    // 提取 ProbeResult
    if agentMsg, ok := resp.(*pb.AgentMessage); ok {
        if probeResult := agentMsg.GetProbeResult(); probeResult != nil {
            return probeResult
        }
    }

    return &pb.ProbeResult{
        RequestId: req.RequestId,
        Success:   false,
        Error:     "invalid probe result",
    }
}
```

### 步骤 3：扩展 AgentCommandFactory 接口

在 `internal/biz/inspection/agent.go` 中：

```go
type AgentCommandFactory interface {
    IsOnline(hostID uint) bool
    NewExecutor(hostID uint) (collector.CommandExecutor, error)
    SendProbeRequest(hostID uint, req *pb.ProbeRequest) *pb.ProbeResult
}
```

在 `internal/server/agent/agent_executor.go` 中实现：

```go
func (f *AgentCommandFactory) SendProbeRequest(hostID uint, req *pb.ProbeRequest) *pb.ProbeResult {
    return f.hub.SendProbeRequest(hostID, req)
}
```

### 步骤 4：改造 executor.go

删除旧实现，替换为新的 Agent 拨测逻辑：

```go
// executeAppProbe runs a single application probe.
func (e *NetworkProbeExecutor) executeAppProbe(cfg *ProbeConfig) (*probers.AppResult, uint) {
    appCfg := buildAppProbeConfig(cfg)

    // Agent mode
    if cfg.ExecMode == ExecModeAgent {
        if e.agentFactory == nil {
            return &probers.AppResult{Error: "agent factory not initialized"}, 0
        }

        hostIDs := parseHostIDs(cfg.AgentHostIDs)
        if len(hostIDs) == 0 {
            return &probers.AppResult{Error: "no agent host specified"}, 0
        }

        // 过滤在线 Agent
        var onlineIDs []uint
        for _, id := range hostIDs {
            if e.agentFactory.IsOnline(id) {
                onlineIDs = append(onlineIDs, id)
            }
        }

        if len(onlineIDs) == 0 {
            return &probers.AppResult{Error: "no online agent available"}, 0
        }

        // 随机选择一个在线 Agent
        hostID := onlineIDs[rand.Intn(len(onlineIDs))]

        // 构建 ProbeRequest
        probeReq := buildProbeRequest(cfg, appCfg)

        // 发送拨测请求
        pbResult := e.agentFactory.SendProbeRequest(hostID, probeReq)

        // 转换为 AppResult
        appResult := convertProbeResultToAppResult(pbResult)

        return appResult, hostID
    }

    // Proxy mode 和 Local mode 保持不变
    // ...
}

// buildProbeRequest 构建 protobuf ProbeRequest
func buildProbeRequest(cfg *ProbeConfig, appCfg *probers.AppProbeConfig) *pb.ProbeRequest {
    req := &pb.ProbeRequest{
        RequestId:  generateRequestID(),
        ProbeType:  cfg.Type,
        Url:        appCfg.URL,
        Method:     appCfg.Method,
        Body:       appCfg.Body,
        Timeout:    int32(appCfg.Timeout),
        SkipVerify: appCfg.SkipVerify,
        ProxyUrl:   appCfg.ProxyURL,
    }

    // Headers
    if len(appCfg.Headers) > 0 {
        req.Headers = appCfg.Headers
    }

    // Params
    if len(appCfg.Params) > 0 {
        req.Params = appCfg.Params
    }

    // Assertions
    if len(appCfg.Assertions) > 0 {
        req.Assertions = make([]*pb.ProbeAssertion, 0, len(appCfg.Assertions))
        for _, a := range appCfg.Assertions {
            req.Assertions = append(req.Assertions, &pb.ProbeAssertion{
                Name:      a.Name,
                Source:    a.Source,
                Path:      a.Path,
                Condition: a.Condition,
                Value:     a.Value,
            })
        }
    }

    // WebSocket specific
    if cfg.Type == "websocket" {
        req.WsMessage = appCfg.WSMessage
        req.WsMessageType = int32(appCfg.WSMessageType)
        req.WsReadTimeout = int32(appCfg.WSReadTimeout)
    }

    return req
}

// convertProbeResultToAppResult 转换 pb.ProbeResult 到 probers.AppResult
func convertProbeResultToAppResult(pbResult *pb.ProbeResult) *probers.AppResult {
    result := &probers.AppResult{
        Success:           pbResult.Success,
        Error:             pbResult.Error,
        Latency:           pbResult.Latency,
        HTTPStatusCode:    int(pbResult.HttpStatusCode),
        HTTPResponseTime:  pbResult.HttpResponseTime,
        HTTPContentLength: pbResult.HttpContentLength,
        ResponseBody:      pbResult.ResponseBody,
        ResponseHeaders:   pbResult.ResponseHeaders,

        // Performance breakdown
        DNSLookupTime:       pbResult.DnsLookupTime,
        TCPConnectTime:      pbResult.TcpConnectTimeHttp,
        TLSHandshakeTime:    pbResult.TlsHandshakeTime,
        TTFB:                pbResult.Ttfb,
        ContentTransferTime: pbResult.ContentTransferTime,

        // TLS info
        TLSVersion:      pbResult.TlsVersion,
        TLSCipherSuite:  pbResult.TlsCipherSuite,
        SSLCertNotAfter: pbResult.SslCertNotAfter,

        // HTTP details
        RedirectCount:       int(pbResult.RedirectCount),
        RedirectTime:        pbResult.RedirectTime,
        FinalURL:            pbResult.FinalUrl,
        ResponseHeaderBytes: int(pbResult.ResponseHeaderBytes),
        ResponseBodyBytes:   int(pbResult.ResponseBodyBytes),

        // Assertions
        AssertionSuccess:   pbResult.AssertionSuccess,
        AssertionPassCount: int(pbResult.AssertionPassCount),
        AssertionFailCount: int(pbResult.AssertionFailCount),
        AssertionEvalTime:  pbResult.AssertionEvalTime,
    }

    // Convert assertion results
    if len(pbResult.AssertionResults) > 0 {
        result.AssertionResults = make([]probers.AssertionResult, 0, len(pbResult.AssertionResults))
        for _, ar := range pbResult.AssertionResults {
            result.AssertionResults = append(result.AssertionResults, probers.AssertionResult{
                Name:    ar.Name,
                Success: ar.Success,
                Actual:  ar.Actual,
                Error:   ar.Error,
            })
        }
    }

    return result
}

// generateRequestID 生成请求 ID
func generateRequestID() string {
    return fmt.Sprintf("probe_%d_%d", time.Now().UnixNano(), rand.Int63())
}
```

### 步骤 5：删除旧实现

```bash
rm internal/biz/inspection/probers/agent_app_prober.go
rm docs/agent-app-probe-implementation.md
```

### 步骤 6：编译测试

```bash
# 编译 Agent
cd agent
go mod tidy
go build -o srehub-agent cmd/main.go

# 编译服务端
cd ..
make build
```

## 测试计划

1. **单元测试**
   - 测试各个拨测器的基本功能
   - 测试断言评估逻辑

2. **集成测试**
   - 部署新版本 Agent
   - 创建应用服务拨测配置（HTTP/HTTPS/WebSocket）
   - 选择 Agent 执行模式
   - 执行拨测，验证结果正确性

3. **性能测试**
   - 对比 shell 命令方式和原生实现的性能差异
   - 测试高并发拨测场景

4. **异常测试**
   - Agent 离线时的错误处理
   - 拨测超时场景
   - 目标服务不可达场景

## 注意事项

1. **protoc 版本**：确保使用 protoc v3+ 版本
2. **Go 版本**：确保 Go 1.18+ 以支持泛型和新特性
3. **依赖包**：Agent 需要 `gorilla/websocket` 包
4. **向后兼容**：旧版本 Agent 不支持拨测，需要优雅降级
5. **日志记录**：Agent 端应记录详细的拨测日志

## 预期效果

- ✅ Agent 端具备完整的拨测能力
- ✅ 性能提升 70%+（相比 shell 命令方式）
- ✅ 支持所有拨测类型：Ping/TCP/UDP/HTTP/HTTPS/WebSocket
- ✅ 支持完整的性能指标和断言评估
- ✅ 无需依赖系统命令（curl、nc 等）
