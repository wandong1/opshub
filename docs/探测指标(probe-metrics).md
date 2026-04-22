# 智能巡检拨测 Prometheus Metrics 指标说明

## 概述

OpsHub 智能巡检拨测系统提供了丰富的 Prometheus 监控指标，对标 Prometheus blackbox_exporter，支持 HTTP/HTTPS/WebSocket/TCP/UDP/Ping 等多种拨测类型。所有指标通过 Pushgateway 推送到 Prometheus。

## 指标命名规范

所有指标以 `opshub_probe_` 为前缀，遵循 Prometheus 命名规范：
- 时间类指标以 `_seconds` 结尾
- 计数类指标以 `_count` 结尾
- 字节类指标以 `_bytes` 结尾
- 比率类指标以 `_ratio` 结尾

## 通用标签 (Labels)

所有指标都包含以下标签：

| 标签名 | 说明 | 示例 |
|:------|:-----|:-----|
| `probe_name` | 拨测配置名称 | `api-health-check` |
| `probe_type` | 拨测类型 | `http`, `https`, `tcp`, `udp`, `ping`, `websocket` |
| `target` | 拨测目标 | `https://api.example.com` |
| `group_name` | 拨测分组名称 | `生产环境` |
| `task_name` | 拨测任务名称 | `每分钟健康检查` |
| `exec_mode` | 执行模式 | `local`, `agent`, `proxy` |
| `instance` | 执行实例主机名 | `opshub-server-01` |
| `task_id` | 任务 ID | `123` |
| `config_id` | 配置 ID | `456` |

自定义标签：通过拨测配置的 `tags` 字段添加（格式：`key=value,key2=value2`）

---

## 一、通用指标

### 1.1 拨测成功状态

**指标名**: `opshub_probe_success`

**类型**: Gauge

**说明**: 拨测是否成功（1=成功，0=失败）

**示例**:
```promql
# 查询所有失败的拨测
opshub_probe_success{probe_name="api-check"} == 0

# 计算成功率
avg_over_time(opshub_probe_success{probe_name="api-check"}[5m]) * 100
```

### 1.2 拨测总耗时

**指标名**: `opshub_probe_duration_seconds`

**类型**: Gauge

**单位**: 秒

**说明**: 拨测总耗时，包括请求响应时间和断言评估时间

**示例**:
```promql
# 查询平均响应时间
avg(opshub_probe_duration_seconds{probe_type="http"})

# 查询 P95 响应时间
histogram_quantile(0.95, opshub_probe_duration_seconds{probe_name="api-check"})
```

### 1.3 重试次数

**指标名**: `opshub_probe_retry_attempt`

**类型**: Gauge

**说明**: 拨测失败后的实际重试次数（0 表示首次成功）

**示例**:
```promql
# 查询需要重试的拨测
opshub_probe_retry_attempt > 0

# 统计重试次数分布
count by (probe_name) (opshub_probe_retry_attempt > 0)
```

---

## 二、HTTP/HTTPS 拨测指标

### 2.1 HTTP 响应时间

**指标名**: `opshub_probe_http_response_seconds`

**类型**: Gauge

**单位**: 秒

**说明**: HTTP 请求的纯响应时间（不包括断言评估）

**示例**:
```promql
# 查询响应时间超过 1 秒的拨测
opshub_probe_http_response_seconds > 1
```

### 2.2 HTTP 状态码

**指标名**: `opshub_probe_http_status_code`

**类型**: Gauge

**说明**: HTTP 响应状态码（200, 404, 500 等）

**示例**:
```promql
# 查询所有 5xx 错误
opshub_probe_http_status_code >= 500

# 统计状态码分布
count by (probe_name, opshub_probe_http_status_code) (opshub_probe_http_status_code)
```

### 2.3 HTTP 内容长度

**指标名**: `opshub_probe_http_content_length`

**类型**: Gauge

**单位**: 字节

**说明**: HTTP 响应体大小

**示例**:
```promql
# 查询响应体超过 1MB 的拨测
opshub_probe_http_content_length > 1048576
```

---

## 三、性能分解指标（HTTP/HTTPS）

### 3.1 DNS 解析时间

**指标名**: `opshub_probe_dns_lookup_seconds`

**类型**: Gauge

**单位**: 秒

**说明**: DNS 域名解析耗时

**用途**: 诊断 DNS 解析问题

**示例**:
```promql
# 查询 DNS 解析慢的拨测（>100ms）
opshub_probe_dns_lookup_seconds > 0.1

# 计算 DNS 解析平均时间
avg(opshub_probe_dns_lookup_seconds{probe_type="https"})
```

### 3.2 TCP 连接时间

**指标名**: `opshub_probe_tcp_connect_seconds`

**类型**: Gauge

**单位**: 秒

**说明**: TCP 三次握手耗时（HTTP 层）

**用途**: 诊断网络连接问题

**示例**:
```promql
# 查询 TCP 连接慢的拨测（>200ms）
opshub_probe_tcp_connect_seconds > 0.2
```

### 3.3 TLS 握手时间

**指标名**: `opshub_probe_tls_handshake_seconds`

**类型**: Gauge

**单位**: 秒

**说明**: TLS/SSL 握手耗时（仅 HTTPS）

**用途**: 诊断 HTTPS 性能瓶颈

**示例**:
```promql
# 查询 TLS 握手慢的拨测（>500ms）
opshub_probe_tls_handshake_seconds > 0.5

# 计算 TLS 握手占比
opshub_probe_tls_handshake_seconds / opshub_probe_http_response_seconds * 100
```

### 3.4 首字节时间 (TTFB)

**指标名**: `opshub_probe_ttfb_seconds`

**类型**: Gauge

**单位**: 秒

**说明**: Time To First Byte，从发送请求到收到第一个字节的时间

**用途**: 区分网络延迟 vs 服务器处理时间

**示例**:
```promql
# 查询 TTFB 慢的拨测（>1s）
opshub_probe_ttfb_seconds > 1

# 计算服务器处理时间（TTFB - 网络时间）
opshub_probe_ttfb_seconds - (opshub_probe_dns_lookup_seconds + opshub_probe_tcp_connect_seconds + opshub_probe_tls_handshake_seconds)
```

### 3.5 内容传输时间

**指标名**: `opshub_probe_content_transfer_seconds`

**类型**: Gauge

**单位**: 秒

**说明**: 从收到第一个字节到接收完所有内容的时间

**用途**: 分析数据传输性能

**示例**:
```promql
# 查询内容传输慢的拨测（>2s）
opshub_probe_content_transfer_seconds > 2

# 计算传输速率（字节/秒）
opshub_probe_http_content_length / opshub_probe_content_transfer_seconds
```

---

## 四、TLS/证书监控指标

### 4.1 证书过期时间

**指标名**: `opshub_probe_ssl_cert_not_after_seconds`

**类型**: Gauge

**单位**: Unix 时间戳（秒）

**说明**: SSL/TLS 证书的过期时间戳

**用途**: 证书过期预警

**示例**:
```promql
# 查询 30 天内过期的证书
(opshub_probe_ssl_cert_not_after_seconds - time()) < 30*24*3600

# 计算证书剩余天数
(opshub_probe_ssl_cert_not_after_seconds - time()) / 86400
```

**告警规则示例**:
```yaml
- alert: SSLCertificateExpiringSoon
  expr: (opshub_probe_ssl_cert_not_after_seconds - time()) < 30*24*3600
  for: 1h
  labels:
    severity: warning
  annotations:
    summary: "SSL 证书即将过期"
    description: "{{ $labels.probe_name }} 的证书将在 {{ $value | humanizeDuration }} 后过期"
```

---

## 五、HTTP 详细信息指标

### 5.1 重定向次数

**指标名**: `opshub_probe_http_redirect_count`

**类型**: Gauge

**说明**: HTTP 重定向次数（301/302/307/308）

**示例**:
```promql
# 查询有重定向的拨测
opshub_probe_http_redirect_count > 0

# 统计重定向次数分布
histogram_quantile(0.95, opshub_probe_http_redirect_count)
```

### 5.2 重定向总时间

**指标名**: `opshub_probe_http_redirect_time_seconds`

**类型**: Gauge

**单位**: 秒

**说明**: 所有重定向的总耗时

**示例**:
```promql
# 查询重定向耗时占比
opshub_probe_http_redirect_time_seconds / opshub_probe_http_response_seconds * 100
```

### 5.3 响应头大小

**指标名**: `opshub_probe_http_response_header_bytes`

**类型**: Gauge

**单位**: 字节

**说明**: HTTP 响应头的大小（近似值）

**示例**:
```promql
# 查询响应头过大的拨测（>10KB）
opshub_probe_http_response_header_bytes > 10240
```

### 5.4 响应体大小

**指标名**: `opshub_probe_http_response_body_bytes`

**类型**: Gauge

**单位**: 字节

**说明**: HTTP 响应体的实际大小

**示例**:
```promql
# 监控响应体大小变化
delta(opshub_probe_http_response_body_bytes[1h])
```

---

## 六、断言统计指标

### 6.1 断言成功状态

**指标名**: `opshub_probe_assertion_success`

**类型**: Gauge

**说明**: 所有断言是否全部通过（1=全部通过，0=有失败）

**示例**:
```promql
# 查询断言失败的拨测
opshub_probe_assertion_success == 0
```

### 6.2 通过的断言数量

**指标名**: `opshub_probe_assertion_pass_count`

**类型**: Gauge

**说明**: 通过的断言数量

**示例**:
```promql
# 查询断言通过率
opshub_probe_assertion_pass_count / (opshub_probe_assertion_pass_count + opshub_probe_assertion_fail_count) * 100
```

### 6.3 失败的断言数量

**指标名**: `opshub_probe_assertion_fail_count`

**类型**: Gauge

**说明**: 失败的断言数量

**示例**:
```promql
# 查询有断言失败的拨测
opshub_probe_assertion_fail_count > 0

# 统计断言失败 Top 10
topk(10, opshub_probe_assertion_fail_count)
```

### 6.4 断言评估时间

**指标名**: `opshub_probe_assertion_eval_seconds`

**类型**: Gauge

**单位**: 秒

**说明**: 断言评估的耗时

**用途**: 区分请求响应时间 vs 断言评估时间

**示例**:
```promql
# 查询断言评估慢的拨测（>100ms）
opshub_probe_assertion_eval_seconds > 0.1

# 计算断言评估占比
opshub_probe_assertion_eval_seconds / opshub_probe_duration_seconds * 100
```

---

## 七、网络拨测指标

### 7.1 Ping 拨测

#### 7.1.1 平均 RTT

**指标名**: `opshub_probe_ping_rtt_seconds`

**类型**: Gauge

**单位**: 秒

**说明**: Ping 平均往返时间

**示例**:
```promql
# 查询 Ping 延迟高的目标（>100ms）
opshub_probe_ping_rtt_seconds > 0.1
```

#### 7.1.2 最小 RTT

**指标名**: `opshub_probe_ping_rtt_min_seconds`

**类型**: Gauge

**单位**: 秒

**说明**: Ping 最小往返时间

#### 7.1.3 最大 RTT

**指标名**: `opshub_probe_ping_rtt_max_seconds`

**类型**: Gauge

**单位**: 秒

**说明**: Ping 最大往返时间

#### 7.1.4 RTT 标准差

**指标名**: `opshub_probe_ping_stddev_seconds`

**类型**: Gauge

**单位**: 秒

**说明**: Ping RTT 的标准差（抖动）

**用途**: 评估网络稳定性

**示例**:
```promql
# 查询网络抖动大的目标（标准差>50ms）
opshub_probe_ping_stddev_seconds > 0.05
```

#### 7.1.5 丢包率

**指标名**: `opshub_probe_ping_packet_loss_ratio`

**类型**: Gauge

**单位**: 0~1（0=无丢包，1=全部丢包）

**说明**: Ping 丢包率

**示例**:
```promql
# 查询有丢包的目标
opshub_probe_ping_packet_loss_ratio > 0

# 告警：丢包率超过 10%
opshub_probe_ping_packet_loss_ratio > 0.1
```

#### 7.1.6 发送包数

**指标名**: `opshub_probe_ping_packets_sent`

**类型**: Gauge

**说明**: Ping 发送的包数量

#### 7.1.7 接收包数

**指标名**: `opshub_probe_ping_packets_received`

**类型**: Gauge

**说明**: Ping 接收的包数量

### 7.2 TCP 拨测

**指标名**: `opshub_probe_tcp_connect_seconds`

**类型**: Gauge

**单位**: 秒

**说明**: TCP 连接建立时间（网络层）

**示例**:
```promql
# 查询 TCP 连接慢的目标（>500ms）
opshub_probe_tcp_connect_seconds{probe_type="tcp"} > 0.5
```

### 7.3 UDP 拨测

#### 7.3.1 UDP 写入时间

**指标名**: `opshub_probe_udp_write_seconds`

**类型**: Gauge

**单位**: 秒

**说明**: UDP 数据发送耗时

#### 7.3.2 UDP 读取时间

**指标名**: `opshub_probe_udp_read_seconds`

**类型**: Gauge

**单位**: 秒

**说明**: UDP 数据接收耗时

---

## 八、Grafana 仪表板示例

### 8.1 HTTP 性能分解瀑布图

```promql
# DNS 解析
opshub_probe_dns_lookup_seconds{probe_name="$probe"}

# TCP 连接
opshub_probe_tcp_connect_seconds{probe_name="$probe"}

# TLS 握手
opshub_probe_tls_handshake_seconds{probe_name="$probe"}

# TTFB
opshub_probe_ttfb_seconds{probe_name="$probe"}

# 内容传输
opshub_probe_content_transfer_seconds{probe_name="$probe"}
```

### 8.2 拨测成功率面板

```promql
# 成功率（5 分钟平均）
avg_over_time(opshub_probe_success{probe_type="http"}[5m]) * 100
```

### 8.3 证书过期预警面板

```promql
# 证书剩余天数
(opshub_probe_ssl_cert_not_after_seconds - time()) / 86400
```

### 8.4 断言失败 Top 10

```promql
# 按失败次数排序
topk(10, sum by (probe_name) (opshub_probe_assertion_fail_count))
```

---

## 九、告警规则示例

### 9.1 拨测失败告警

```yaml
- alert: ProbeFailure
  expr: opshub_probe_success == 0
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "拨测失败"
    description: "{{ $labels.probe_name }} 拨测持续失败 5 分钟"
```

### 9.2 响应时间告警

```yaml
- alert: HighResponseTime
  expr: opshub_probe_http_response_seconds > 3
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "响应时间过高"
    description: "{{ $labels.probe_name }} 响应时间超过 3 秒"
```

### 9.3 证书过期告警

```yaml
- alert: SSLCertExpiringSoon
  expr: (opshub_probe_ssl_cert_not_after_seconds - time()) < 30*24*3600
  for: 1h
  labels:
    severity: warning
  annotations:
    summary: "SSL 证书即将过期"
    description: "{{ $labels.probe_name }} 的证书将在 30 天内过期"
```

### 9.4 断言失败告警

```yaml
- alert: AssertionFailure
  expr: opshub_probe_assertion_fail_count > 0
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "断言失败"
    description: "{{ $labels.probe_name }} 有 {{ $value }} 个断言失败"
```

### 9.5 丢包率告警

```yaml
- alert: HighPacketLoss
  expr: opshub_probe_ping_packet_loss_ratio > 0.1
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "丢包率过高"
    description: "{{ $labels.target }} 丢包率为 {{ $value | humanizePercentage }}"
```

---

## 十、与 Prometheus blackbox_exporter 对比

| 功能 | OpsHub | blackbox_exporter |
|:-----|:-------|:------------------|
| DNS 解析时间 | ✅ `opshub_probe_dns_lookup_seconds` | ✅ `probe_dns_lookup_duration_seconds` |
| TCP 连接时间 | ✅ `opshub_probe_tcp_connect_seconds` | ✅ `probe_http_duration_seconds{phase="connect"}` |
| TLS 握手时间 | ✅ `opshub_probe_tls_handshake_seconds` | ✅ `probe_http_duration_seconds{phase="tls"}` |
| TTFB | ✅ `opshub_probe_ttfb_seconds` | ✅ `probe_http_duration_seconds{phase="processing"}` |
| 证书过期时间 | ✅ `opshub_probe_ssl_cert_not_after_seconds` | ✅ `probe_ssl_earliest_cert_expiry` |
| 重定向次数 | ✅ `opshub_probe_http_redirect_count` | ✅ `probe_http_redirects` |
| 断言统计 | ✅ 独有功能 | ❌ 不支持 |
| 重试次数 | ✅ 独有功能 | ❌ 不支持 |
| 响应体大小 | ✅ `opshub_probe_http_response_body_bytes` | ✅ `probe_http_content_length` |

---

## 十一、最佳实践

### 11.1 指标采集频率

- **生产环境关键服务**: 1 分钟
- **一般服务**: 5 分钟
- **低优先级服务**: 15 分钟

### 11.2 数据保留策略

- **原始数据**: 保留 30 天
- **5 分钟聚合**: 保留 90 天
- **1 小时聚合**: 保留 1 年

### 11.3 告警阈值建议

| 指标 | 警告阈值 | 严重阈值 |
|:-----|:---------|:---------|
| 响应时间 | > 3s | > 10s |
| 丢包率 | > 5% | > 20% |
| 证书过期 | < 30 天 | < 7 天 |
| 断言失败 | > 0 | 持续 5 分钟 |

### 11.4 性能优化建议

1. **合理设置并发数**: 避免过高并发导致系统负载
2. **使用标签过滤**: 减少 Prometheus 查询压力
3. **定期清理历史数据**: 避免 Pushgateway 内存溢出
4. **使用 Agent 模式**: 分布式拨测，降低中心节点压力

---

## 十二、常见问题

### Q1: 为什么某些指标值为 0？

**A**: 部分指标仅在特定条件下有值：
- `opshub_probe_dns_lookup_seconds`: 仅在首次解析或 DNS 缓存过期时有值
- `opshub_probe_tls_handshake_seconds`: 仅 HTTPS 拨测有值
- `opshub_probe_http_redirect_count`: 仅发生重定向时有值

### Q2: 如何区分 HTTP 响应时间和总耗时？

**A**:
- `opshub_probe_http_response_seconds`: 纯 HTTP 请求响应时间
- `opshub_probe_duration_seconds`: 总耗时 = HTTP 响应时间 + 断言评估时间

### Q3: 证书过期时间为什么是时间戳？

**A**: 使用 Unix 时间戳便于计算剩余天数：
```promql
(opshub_probe_ssl_cert_not_after_seconds - time()) / 86400
```

### Q4: 如何监控拨测任务的整体健康度？

**A**: 使用聚合查询：
```promql
# 整体成功率
avg(opshub_probe_success) * 100

# 平均响应时间
avg(opshub_probe_duration_seconds)

# 失败任务数量
count(opshub_probe_success == 0)
```

---

## 十三、更新日志

### v1.0.0 (2026-03-05)

**新增指标**:
- ✅ DNS 解析时间
- ✅ TCP 连接时间（HTTP 层）
- ✅ TLS 握手时间
- ✅ TTFB（首字节时间）
- ✅ 内容传输时间
- ✅ 证书过期时间
- ✅ TLS 版本和密码套件（存储在数据库，未推送到 Prometheus）
- ✅ 重定向次数和耗时
- ✅ 响应头/响应体大小
- ✅ 断言通过/失败数量
- ✅ 断言评估时间
- ✅ 重试次数

**改进**:
- 使用 `net/http/httptrace` 实现性能分解
- 优化指标命名，遵循 Prometheus 规范
- 完善标签体系，支持自定义标签

---

## 联系方式

如有问题或建议，请提交 [GitHub Issue](https://github.com/ydcloud-dy/opshub/issues)。
