# Agent 应用服务拨测实现

## 问题描述

在应用服务拨测中选择 Agent 主机进行拨测时，Agent 端没有收到任何请求，也没有打印日志。经排查发现，应用服务的 Agent 拨测功能尚未实现。

## 问题根源

在 `internal/biz/inspection/executor.go:355-362` 行，应用服务（HTTP/HTTPS/WebSocket）的 Agent 拨测直接返回错误：

```go
// For now, agent app probing is not implemented, return error
appLogger.Warn("Agent mode for application probing not yet implemented",
    zap.String("probe_name", cfg.Name),
    zap.Uint("host_id", hostID),
)
return &probers.AppResult{
    Error: fmt.Sprintf("agent mode for application probing not yet implemented (agent host: %d)", hostID),
}, hostID
```

## 解决方案

### 1. 新增 Agent 应用服务 Prober

创建 `internal/biz/inspection/probers/agent_app_prober.go`，实现：

- **AgentHTTPProber** - 通过 Agent 执行 curl 命令进行 HTTP/HTTPS 拨测
- **AgentWebSocketProber** - 通过 Agent 执行 curl 升级请求进行 WebSocket 拨测
- **GetAgentAppProber()** - 工厂函数，根据类型返回对应的 Agent Prober

### 2. 实现原理

#### HTTP/HTTPS 拨测

使用 curl 命令通过 Agent 执行，关键特性：

- 支持自定义 Method、Headers、Body、Params
- 支持代理（ProxyURL）
- 支持跳过 TLS 验证（-k）
- 使用 curl 的 `-w` 参数获取详细性能指标：
  - DNS 解析时间
  - TCP 连接时间
  - TLS 握手时间
  - TTFB（首字节时间）
  - 内容传输时间
  - 重定向次数和最终 URL
  - 响应大小

#### 断言评估

- 复用现有的 `EvaluateAssertions()` 函数
- 支持对响应体和响应头进行断言
- 统计断言通过/失败数量

### 3. 修改 executor.go

在 `executeAppProbe()` 函数中实现 Agent 模式逻辑：

1. 解析 Agent 主机 ID 列表
2. 过滤在线的 Agent
3. 随机选择一个在线 Agent
4. 创建 Agent Executor
5. 获取 Agent App Prober
6. 执行拨测并返回结果

## 技术细节

### curl 命令构建

```bash
curl -s -S --max-time 30 \
  -X POST \
  -H 'Content-Type: application/json' \
  -d '{"key":"value"}' \
  -k \
  -w '\n__CURL_METRICS__\n{"http_code":%{http_code},"time_total":%{time_total},...}' \
  'https://example.com/api' 2>&1 | head -c 4096
```

### 输出解析

curl 输出分为两部分，通过 `__CURL_METRICS__` 分隔：

1. **响应体** - 前半部分，截取前 4KB
2. **性能指标** - 后半部分，JSON 格式，包含所有时间和状态信息

### Agent 依赖

Agent 主机需要安装 `curl` 命令，这是大多数 Linux 系统的标准工具。

## 测试建议

1. **基础 HTTP 拨测**
   - 创建一个 HTTP 类型的拨测配置
   - 执行模式选择 "Agent"
   - 选择一个在线的 Agent 主机
   - 执行拨测，验证结果正确

2. **HTTPS 拨测**
   - 测试跳过 TLS 验证选项
   - 验证 TLS 握手时间等指标

3. **带断言的拨测**
   - 配置响应体断言（如 JSON 路径）
   - 配置响应头断言
   - 验证断言评估正确

4. **WebSocket 拨测**
   - 测试 WebSocket 升级请求
   - 验证 101 状态码识别

5. **错误场景**
   - Agent 离线时的错误提示
   - 目标服务不可达时的错误信息
   - 超时场景

## 日志输出

实现中添加了详细的日志，便于排查问题：

- Agent 模式检测
- Agent 主机 ID 解析
- 在线 Agent 筛选
- Agent 选择
- Executor 创建
- Prober 获取
- 拨测执行

## 兼容性

- 与现有的本地拨测和代理拨测模式完全兼容
- 复用现有的断言评估逻辑
- 复用现有的结果存储和指标推送逻辑

## 后续优化

1. **WebSocket 完整支持** - 当前使用 curl 升级请求，可考虑使用 websocat 工具实现完整的 WebSocket 交互
2. **响应头采集** - curl 可以通过 `-i` 参数获取响应头，可增强响应头的采集和断言
3. **证书信息** - 可通过 curl 的 `--cert-status` 等参数获取更详细的证书信息
4. **性能优化** - 对于高频拨测，可考虑复用 Agent 连接

## 修改文件

- `internal/biz/inspection/probers/agent_app_prober.go` - 新增
- `internal/biz/inspection/executor.go` - 修改 executeAppProbe() 函数

## 提交信息

```
feat(拨测): 实现 Agent 模式下的应用服务拨测

- 新增 AgentHTTPProber 和 AgentWebSocketProber
- 通过 curl 命令在 Agent 端执行 HTTP/HTTPS/WebSocket 拨测
- 支持完整的性能指标采集（DNS、TCP、TLS、TTFB 等）
- 支持断言评估
- 修复 Agent 拨测时无日志输出的问题
```
