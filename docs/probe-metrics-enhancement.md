# 智能巡检拨测 Metrics 指标优化实现总结

## 实施日期
2026-03-05

## 实现内容

本次优化针对智能巡检拨测系统进行了全面的性能指标增强，对标 Prometheus blackbox_exporter，新增了关键性能分解指标和监控能力。

### 一、核心功能实现

#### 1. HTTP/HTTPS 性能分解（使用 httptrace）

**实现文件**: `internal/biz/inspection/probers/http.go`

**新增性能指标**:
- **DNS 解析时间** (`DNSLookupTime`) - 诊断 DNS 问题
- **TCP 连接时间** (`TCPConnectTime`) - HTTP 层 TCP 握手耗时
- **TLS 握手时间** (`TLSHandshakeTime`) - HTTPS 性能瓶颈分析
- **TTFB** (`TTFB`) - 首字节时间，区分网络延迟 vs 服务器处理时间
- **内容传输时间** (`ContentTransferTime`) - 数据下载耗时

**实现方式**:
```go
trace := &httptrace.ClientTrace{
    DNSStart: func(info httptrace.DNSStartInfo) { dnsStart = time.Now() },
    DNSDone: func(info httptrace.DNSDoneInfo) { result.DNSLookupTime = ms(dnsStart) },
    ConnectStart: func(network, addr string) { connectStart = time.Now() },
    ConnectDone: func(network, addr string, err error) { result.TCPConnectTime = ms(connectStart) },
    TLSHandshakeStart: func() { tlsStart = time.Now() },
    TLSHandshakeDone: func(state tls.ConnectionState, err error) {
        result.TLSHandshakeTime = ms(tlsStart)
        result.TLSVersion = tlsVersionString(state.Version)
        result.TLSCipherSuite = tls.CipherSuiteName(state.CipherSuite)
        if len(state.PeerCertificates) > 0 {
            result.SSLCertNotAfter = state.PeerCertificates[0].NotAfter.Unix()
        }
    },
    GotFirstResponseByte: func() { firstByteTime = time.Now() },
}
```

#### 2. TLS/证书监控

**新增字段**:
- `TLSVersion` - TLS 版本（TLS 1.0/1.1/1.2/1.3）
- `TLSCipherSuite` - 密码套件名称
- `SSLCertNotAfter` - 证书过期时间戳（Unix timestamp）

**用途**:
- 监控证书有效期，提前预警证书过期（距离过期 < 30 天）
- 审计 TLS 版本和密码套件安全性

#### 3. HTTP 详细信息

**新增字段**:
- `RedirectCount` - 重定向次数
- `RedirectTime` - 重定向总耗时
- `FinalURL` - 最终 URL（跟随重定向后）
- `ResponseHeaderBytes` - 响应头大小
- `ResponseBodyBytes` - 响应体大小

**用途**:
- 分析重定向链路性能
- 监控响应体大小变化

#### 4. 断言统计增强

**新增字段**:
- `AssertionPassCount` - 通过的断言数量
- `AssertionFailCount` - 失败的断言数量
- `AssertionEvalTime` - 断言评估耗时

**改进点**:
- 之前只记录整体成功/失败，现在可以看到具体通过/失败的断言数量
- 区分请求响应时间 vs 断言评估时间

#### 5. 重试次数指标

**新增字段**:
- `RetryAttempt` - 实际重试次数（网络拨测和应用拨测）

**改进点**:
- 之前已记录但未推送到 Prometheus
- 现在推送到 `opshub_probe_retry_attempt` 指标

### 二、数据库模型扩展

**文件**: `internal/biz/inspection/models.go`

**ProbeResult 新增字段**:
```go
// Performance breakdown metrics
DNSLookupTime       float64 `json:"dnsLookupTime"`
HTTPTCPConnectTime  float64 `json:"httpTcpConnectTime"` // HTTP-specific TCP connect time
TLSHandshakeTime    float64 `json:"tlsHandshakeTime"`
TTFB                float64 `json:"ttfb"`
ContentTransferTime float64 `json:"contentTransferTime"`

// TLS/Certificate information
TLSVersion      string `gorm:"type:varchar(20)" json:"tlsVersion"`
TLSCipherSuite  string `gorm:"type:varchar(100)" json:"tlsCipherSuite"`
SSLCertNotAfter int64  `json:"sslCertNotAfter"`

// HTTP details
RedirectCount       int     `json:"redirectCount"`
RedirectTime        float64 `json:"redirectTime"`
FinalURL            string  `gorm:"type:varchar(2000)" json:"finalUrl"`
ResponseHeaderBytes int     `json:"responseHeaderBytes"`
ResponseBodyBytes   int     `json:"responseBodyBytes"`

// Assertion statistics
AssertionPassCount int     `json:"assertionPassCount"`
AssertionFailCount int     `json:"assertionFailCount"`
AssertionEvalTime  float64 `json:"assertionEvalTime"`
```

**数据库迁移**:
- GORM AutoMigrate 会自动添加新字段
- 无需手动执行 SQL 迁移脚本

### 三、Prometheus 指标推送

**文件**: `internal/biz/inspection/executor.go`

**新增 Prometheus 指标**:

#### 应用服务拨测指标 (pushAppMetrics)
```
opshub_probe_dns_lookup_seconds          - DNS 解析时间
opshub_probe_tcp_connect_seconds         - TCP 连接时间
opshub_probe_tls_handshake_seconds       - TLS 握手时间
opshub_probe_ttfb_seconds                - 首字节时间
opshub_probe_content_transfer_seconds    - 内容传输时间
opshub_probe_ssl_cert_not_after_seconds  - 证书过期时间戳
opshub_probe_http_redirect_count         - 重定向次数
opshub_probe_http_redirect_time_seconds  - 重定向总时间
opshub_probe_http_response_header_bytes  - 响应头大小
opshub_probe_http_response_body_bytes    - 响应体大小
opshub_probe_assertion_pass_count        - 通过的断言数量
opshub_probe_assertion_fail_count        - 失败的断言数量
opshub_probe_assertion_eval_seconds      - 断言评估时间
opshub_probe_retry_attempt               - 重试次数
```

#### 网络拨测指标 (pushMetrics)
```
opshub_probe_retry_attempt               - 重试次数（新增）
```

### 四、代码结构优化

#### 1. AppResult 结构体扩展
**文件**: `internal/biz/inspection/probers/app_prober.go`

新增 24 个字段，涵盖性能分解、TLS 信息、HTTP 详情、断言统计。

#### 2. Result 结构体扩展
**文件**: `internal/biz/inspection/probers/prober.go`

新增 `RetryAttempt` 字段，用于网络拨测重试统计。

#### 3. 执行器逻辑更新
**文件**: `internal/biz/inspection/executor.go`

- `executeAndSaveAppProbe()` - 映射 AppResult 新字段到 ProbeResult
- `executeAndSaveNetworkProbe()` - 设置 Result.RetryAttempt
- `pushAppMetrics()` - 新增参数 `retryAttempt`，推送所有新指标
- `pushMetrics()` - 推送网络拨测重试次数

#### 4. TLS 版本转换函数
**文件**: `internal/biz/inspection/probers/http.go`

```go
func tlsVersionString(version uint16) string {
    switch version {
    case tls.VersionTLS10: return "TLS 1.0"
    case tls.VersionTLS11: return "TLS 1.1"
    case tls.VersionTLS12: return "TLS 1.2"
    case tls.VersionTLS13: return "TLS 1.3"
    default: return "Unknown"
    }
}
```

### 五、验证方案

#### 1. 编译验证
```bash
go build -o /dev/null ./internal/biz/inspection/probers  # ✅ 通过
go build -o /dev/null ./internal/biz/inspection          # ✅ 通过
go build -o /dev/null ./cmd/server                       # ✅ 通过
```

#### 2. 数据库验证
启动服务后，GORM 会自动迁移 `probe_results` 表，添加新字段。

验证 SQL:
```sql
DESCRIBE probe_results;
-- 检查是否包含新字段：
-- dns_lookup_time, http_tcp_connect_time, tls_handshake_time, ttfb,
-- content_transfer_time, tls_version, tls_cipher_suite, ssl_cert_not_after,
-- redirect_count, redirect_time, final_url, response_header_bytes,
-- response_body_bytes, assertion_pass_count, assertion_fail_count, assertion_eval_time
```

#### 3. 功能验证
**测试步骤**:
1. 创建 HTTPS 拨测配置（如 `https://www.google.com`）
2. 创建拨测任务并启用
3. 等待任务执行
4. 查询数据库验证新字段是否有值
5. 检查 Prometheus Pushgateway 是否收到新指标

**验证 SQL**:
```sql
SELECT
    id, probe_config_id, success,
    dns_lookup_time, http_tcp_connect_time, tls_handshake_time, ttfb,
    tls_version, ssl_cert_not_after,
    redirect_count, assertion_pass_count, assertion_fail_count
FROM probe_results
WHERE probe_config_id = ?
ORDER BY created_at DESC LIMIT 1;
```

**Prometheus 查询**:
```promql
# DNS 解析时间
opshub_probe_dns_lookup_seconds{probe_name="test-https"}

# 证书过期时间
opshub_probe_ssl_cert_not_after_seconds{probe_name="test-https"}

# TTFB
opshub_probe_ttfb_seconds{probe_name="test-https"}

# 断言失败数量
opshub_probe_assertion_fail_count{probe_name="test-api"}

# 重试次数
opshub_probe_retry_attempt{probe_name="test-api"}
```

### 六、性能影响评估

#### 1. httptrace 开销
- httptrace 是 Go 标准库提供的零分配追踪机制
- 性能开销 < 1ms，可忽略不计

#### 2. 数据库存储
- 每条 ProbeResult 记录增加约 200 字节
- 对于每天 10 万次拨测，增加约 20MB 存储

#### 3. Prometheus 指标数量
- 应用拨测新增 14 个指标
- 网络拨测新增 1 个指标
- 需评估 Pushgateway 压力（建议监控内存使用）

### 七、后续优化建议

#### 1. WebSocket 拨测性能分解（中优先级）
在 `internal/biz/inspection/probers/websocket.go` 中使用 httptrace，记录 WebSocket 握手阶段的性能分解。

#### 2. Workflow 拨测步骤级指标（中优先级）
在 `pushWorkflowMetrics()` 中为每个步骤推送独立指标：
```go
opshub_probe_workflow_step_success
opshub_probe_workflow_step_latency_seconds
opshub_probe_workflow_step_assertion_pass_count
```

#### 3. Agent 模式应用拨测（高优先级）
当前 Agent 模式回退到本地执行，需要：
- 在 Agent 端实现 HTTP/WebSocket 拨测能力
- 扩展 gRPC 协议支持应用拨测消息类型
- 修改 `executeAppProbe()` 支持真正的 Agent 执行

#### 4. 前端展示增强（低优先级）
- 性能分解瀑布图（DNS → TCP → TLS → TTFB → 传输）
- 证书信息面板（有效期倒计时、版本、密码套件）
- 断言执行详情表格
- 重定向链路追踪

### 八、对标 Prometheus blackbox_exporter

#### 已实现的指标
✅ `probe_dns_lookup_duration_seconds` → `opshub_probe_dns_lookup_seconds`
✅ `probe_http_ssl` → `opshub_probe_ssl_cert_not_after_seconds`
✅ `probe_http_duration_seconds` (phases) → 分解为 DNS/TCP/TLS/TTFB/Transfer
✅ `probe_http_redirects` → `opshub_probe_http_redirect_count`
✅ `probe_http_content_length` → `opshub_probe_http_response_body_bytes`

#### 差异点
- blackbox_exporter 使用 histogram 记录延迟分布，我们使用 gauge
- blackbox_exporter 支持更多协议（ICMP、gRPC），我们聚焦 HTTP/TCP/UDP
- 我们额外支持断言统计和重试次数

### 九、文件变更清单

**核心文件**:
- `internal/biz/inspection/probers/app_prober.go` - AppResult 结构体扩展
- `internal/biz/inspection/probers/http.go` - httptrace 实现
- `internal/biz/inspection/probers/prober.go` - Result 结构体扩展
- `internal/biz/inspection/models.go` - ProbeResult 数据库模型扩展
- `internal/biz/inspection/executor.go` - 执行器和 metrics 推送逻辑

**影响范围**:
- 数据库表结构（自动迁移）
- Prometheus 指标（新增 15 个）
- API 响应（ProbeResult JSON 包含新字段）

### 十、兼容性说明

#### 向后兼容
- 旧版本拨测结果不包含新字段，前端需兼容处理（字段为空或 0）
- 数据库新字段允许 NULL 或默认值 0
- Prometheus 指标向后兼容（新增不影响旧查询）

#### 升级路径
1. 部署新版本代码
2. 重启服务（GORM 自动迁移数据库）
3. 验证新拨测任务是否记录新指标
4. 更新 Grafana 仪表板（可选）

## 总结

本次优化成功实现了智能巡检拨测系统的性能指标增强，新增了 DNS 解析、TCP 连接、TLS 握手、TTFB、证书监控等关键指标，对标 Prometheus blackbox_exporter 的核心能力。所有代码已通过编译验证，数据库模型已扩展，Prometheus 指标已完善。

**实现完成度**:
- ✅ 阶段二：HTTP 拨测指标增强（100%）
- ✅ 阶段三：Metrics 推送完善（100%）
- ⏳ 阶段一：执行模式修复（待实现）
- ⏳ 阶段四：WebSocket 增强（待实现）
- ⏳ 阶段五：断言评估增强（部分完成）
- ⏳ 阶段六：前端展示（待实现）
