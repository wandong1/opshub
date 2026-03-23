# SREHub 巡检&拨测平台监控指标文档

> 版本：v2.0 | 前缀统一为 `srehub_inspect_*` | 支持 Prometheus + PushGateway

---

## 目录

1. [指标设计原则](#一指标设计原则)
2. [通用核心指标](#二通用核心指标所有任务类型)
3. [Ping 拨测专属指标](#三ping-拨测专属指标)
4. [TCP 拨测专属指标](#四tcp-拨测专属指标)
5. [UDP 拨测专属指标](#五udp-拨测专属指标)
6. [HTTP/HTTPS 拨测专属指标](#六httphttps-拨测专属指标)
7. [WebSocket/WSS 拨测专属指标](#七websocketwss-拨测专属指标)
8. [业务编排（Workflow）专属指标](#八业务编排workflow专属指标)
9. [智能巡检专属指标](#九智能巡检专属指标)
10. [统一标签体系](#十统一标签体系)
11. [Counter 持久化机制（Redis）](#十一counter-持久化机制redis)
12. [PushGateway 推送说明](#十二pushgateway-推送说明)
13. [PromQL 示例查询](#十三promql-示例查询)
14. [Grafana 面板建议](#十四grafana-面板建议)

---

## 一、指标设计原则

| 原则 | 说明 |
|------|------|
| **命名规范** | 统一前缀 `srehub_inspect_`，格式 `srehub_inspect_{类型}_{维度}_{单位}` |
| **指标类型** | Counter（累计计数）、Gauge（瞬时值）；Counter 通过 Redis 持久化，以 Gauge 形式推送至 PushGateway |
| **低基数标签** | 不使用随机 ID、全量日志字段；优先业务/环境/目标维度 |
| **触发条件** | 仅当调度任务配置了 `pushgateway_id` 时才推送指标 |
| **重启不丢失** | 所有 Counter 类指标持久化到 Redis，应用重启后自动恢复累计值 |

---

## 二、通用核心指标（所有任务类型）

适用于：巡检、ping、tcp、udp、http/https、websocket、业务编排 全类型。

### 2.1 调度执行计数（Counter，Redis 持久化）

| 指标名 | 类型 | 含义 |
|--------|------|------|
| `srehub_inspect_task_exec_total` | Counter | 调度任务总执行次数 |
| `srehub_inspect_task_success_total` | Counter | 调度任务执行成功次数 |
| `srehub_inspect_task_fail_total` | Counter | 调度任务执行失败次数 |
| `srehub_inspect_task_retry_total` | Counter | 调度任务重试总次数 |
| `srehub_inspect_task_abort_total` | Counter | 调度任务主动终止/取消次数（手动 stop） |

> **注意**：Counter 类型以 Gauge 形式推送至 PushGateway，Gauge 值为 Redis 中的累计计数。

### 2.2 调度性能（Gauge）

| 指标名 | 类型 | 含义 | 单位 |
|--------|------|------|------|
| `srehub_inspect_task_exec_duration_seconds` | Gauge | 本次任务执行耗时 | 秒 |
| `srehub_inspect_task_availability` | Gauge | 任务可用性比率（成功次数 / 总次数） | 0~1 |
| `srehub_inspect_task_availability_gauge` | Gauge | 本次执行结果（1=成功，0=失败） | 0 或 1 |

**示例标签：**
```
srehub_inspect_task_exec_total{
  task_id="42",
  task_name="生产环境HTTP拨测",
  task_type="http",
  business_group="运维组",
  owner="alice",
  schedule_mode="scheduled"
} 1024
```

---

## 三、Ping 拨测专属指标

> `task_type="ping"`

| 指标名 | 类型 | 含义 | 单位 |
|--------|------|------|------|
| `srehub_inspect_ping_avg_rtt_seconds` | Gauge | 平均往返时延（RTT） | 秒 |
| `srehub_inspect_ping_min_rtt_seconds` | Gauge | 最小时延 | 秒 |
| `srehub_inspect_ping_max_rtt_seconds` | Gauge | 最大时延 | 秒 |
| `srehub_inspect_ping_jitter_seconds` | Gauge | 时延抖动（RTT 标准差） | 秒 |
| `srehub_inspect_ping_loss_ratio` | Gauge | 丢包率 | 0~1 |
| `srehub_inspect_ping_packet_send_total` | Counter | 发送数据包总数（累计） | 个 |
| `srehub_inspect_ping_packet_recv_total` | Counter | 接收数据包总数（累计） | 个 |

**示例标签：**
```
srehub_inspect_ping_avg_rtt_seconds{
  task_id="10",
  task_name="核心服务器Ping",
  task_type="ping",
  business_group="基础设施",
  owner="ops",
  schedule_mode="scheduled",
  target="192.168.1.1",
  probe_name="IDC主机-Ping"
} 0.003
```

**告警建议：**
```promql
# RTT 超过 100ms 告警
srehub_inspect_ping_avg_rtt_seconds > 0.1

# 丢包率超过 5% 告警
srehub_inspect_ping_loss_ratio > 0.05
```

---

## 四、TCP 拨测专属指标

> `task_type="tcp"`

| 指标名 | 类型 | 含义 | 单位 |
|--------|------|------|------|
| `srehub_inspect_tcp_connect_success_total` | Counter | TCP 建连成功次数（累计） | 次 |
| `srehub_inspect_tcp_connect_fail_total` | Counter | TCP 建连失败次数（累计） | 次 |
| `srehub_inspect_tcp_connect_duration_seconds` | Gauge | TCP 建连耗时 | 秒 |
| `srehub_inspect_tcp_port_reachable` | Gauge | 端口是否可达（1=可达，0=不可达） | 0 或 1 |

**示例标签：**
```
srehub_inspect_tcp_port_reachable{
  task_id="15",
  task_name="MySQL端口探测",
  task_type="tcp",
  business_group="数据库组",
  owner="dba",
  schedule_mode="scheduled",
  target="db.prod.internal",
  probe_name="MySQL-3306"
} 1
```

**告警建议：**
```promql
# 端口不可达告警
srehub_inspect_tcp_port_reachable == 0

# 建连成功率低于 95%
rate(srehub_inspect_tcp_connect_success_total[5m]) /
  (rate(srehub_inspect_tcp_connect_success_total[5m]) + rate(srehub_inspect_tcp_connect_fail_total[5m])) < 0.95
```

---

## 五、UDP 拨测专属指标

> `task_type="udp"`

| 指标名 | 类型 | 含义 | 单位 |
|--------|------|------|------|
| `srehub_inspect_udp_send_total` | Counter | UDP 报文发送总数（累计） | 个 |
| `srehub_inspect_udp_recv_total` | Counter | UDP 报文接收总数（累计） | 个 |
| `srehub_inspect_udp_loss_total` | Counter | UDP 报文丢失总数（累计） | 个 |
| `srehub_inspect_udp_transfer_delay_seconds` | Gauge | 报文传输时延（写+读时延之和） | 秒 |

**示例标签：**
```
srehub_inspect_udp_transfer_delay_seconds{
  task_id="20",
  task_name="DNS服务UDP探测",
  task_type="udp",
  business_group="网络组",
  owner="netops",
  schedule_mode="scheduled",
  target="8.8.8.8",
  probe_name="DNS-UDP-53"
} 0.005
```

---

## 六、HTTP/HTTPS 拨测专属指标

> `task_type="http"` 或 `task_type="https"`

| 指标名 | 类型 | 含义 | 单位 |
|--------|------|------|------|
| `srehub_inspect_http_response_duration_seconds` | Gauge | HTTP 请求总响应耗时（建连→完整接收） | 秒 |
| `srehub_inspect_http_dns_duration_seconds` | Gauge | DNS 解析耗时 | 秒 |
| `srehub_inspect_http_tls_duration_seconds` | Gauge | TLS 握手耗时 | 秒 |
| `srehub_inspect_http_first_byte_seconds` | Gauge | 首字节响应时间（TTFB） | 秒 |
| `srehub_inspect_http_assertion_result` | Gauge | HTTP 断言结果（1=通过，0=失败） | 0 或 1 |
| `srehub_inspect_http_status_code_total` | Counter | 响应码分布计数（累计，含 status_code 标签） | 次 |
| `srehub_inspect_https_cert_valid_days` | Gauge | HTTPS 证书剩余有效天数 | 天 |

**扩展标签（HTTP/HTTPS 额外标签）：**

| 标签名 | 含义 | 示例 |
|--------|------|------|
| `http_method` | 请求方法 | `GET`、`POST` |
| `http_path` | 请求路径 | `/api/health` |
| `status_code` | 响应状态码（`http_status_code_total` 专属） | `200`、`500` |

**示例：**
```
srehub_inspect_http_response_duration_seconds{
  task_id="30",
  task_name="官网HTTP探测",
  task_type="http",
  business_group="前端组",
  owner="frontend",
  schedule_mode="scheduled",
  target="https://example.com",
  probe_name="官网-HTTP",
  http_method="GET",
  http_path="/",
  status_code="200"
} 0.245

srehub_inspect_https_cert_valid_days{
  task_id="30",
  ...
} 87

srehub_inspect_http_status_code_total{
  task_id="30",
  task_name="官网HTTP探测",
  task_type="http",
  business_group="前端组",
  owner="frontend",
  schedule_mode="scheduled",
  status_code="200"
} 2048
```

**告警建议：**
```promql
# 响应时间超过 2 秒
srehub_inspect_http_response_duration_seconds > 2

# 证书 30 天内到期
srehub_inspect_https_cert_valid_days < 30

# 断言失败
srehub_inspect_http_assertion_result == 0

# 5xx 错误率
rate(srehub_inspect_http_status_code_total{status_code=~"5.."}[5m]) > 0
```

---

## 七、WebSocket/WSS 拨测专属指标

> `task_type="websocket"`

| 指标名 | 类型 | 含义 | 单位 |
|--------|------|------|------|
| `srehub_inspect_ws_connection_established` | Gauge | WebSocket 连接是否成功建立（1=成功，0=失败） | 0 或 1 |
| `srehub_inspect_ws_handshake_duration_seconds` | Gauge | WebSocket 握手建连耗时 | 秒 |
| `srehub_inspect_ws_handshake_success_total` | Counter | 握手成功次数（累计） | 次 |
| `srehub_inspect_ws_disconnect_total` | Counter | 异常断开连接次数（累计） | 次 |

**示例：**
```
srehub_inspect_ws_connection_established{
  task_id="40",
  task_name="实时推送WebSocket探测",
  task_type="websocket",
  business_group="消息组",
  owner="backend",
  schedule_mode="scheduled",
  target="wss://ws.example.com",
  probe_name="WS-Push"
} 1
```

**告警建议：**
```promql
# WebSocket 连接失败
srehub_inspect_ws_connection_established == 0

# 断连次数激增（5分钟内超过10次）
increase(srehub_inspect_ws_disconnect_total[5m]) > 10
```

---

## 八、业务编排（Workflow）专属指标

> `task_type="probe_flow"`；每个步骤单独推送，含步骤级标签

| 指标名 | 类型 | 含义 | 单位/取值 |
|--------|------|------|----------|
| `srehub_inspect_flow_step_exec_total` | Counter | 编排步骤执行次数（累计） | 次 |
| `srehub_inspect_flow_step_fail_total` | Counter | 编排步骤失败次数（累计） | 次 |
| `srehub_inspect_flow_step_status` | Gauge | 步骤执行状态 | 1=成功，0=失败，2=跳过 |
| `srehub_inspect_flow_step_exec_duration` | Gauge | 步骤执行耗时 | 秒 |
| `srehub_inspect_flow_step_assert_result` | Gauge | 步骤断言结果 | 1=通过，0=失败，-1=无断言 |

**步骤级扩展标签：**

| 标签名 | 含义 | 示例 |
|--------|------|------|
| `flow_id` | 编排配置 ID | `12` |
| `step_id` | 步骤序号（从 0 开始） | `0`、`1`、`2` |
| `step_name` | 步骤名称 | `登录接口`、`查询订单` |

**示例（一个 3 步骤的编排任务）：**
```
# 步骤 0：登录
srehub_inspect_flow_step_status{
  task_id="50", task_name="下单流程编排",
  task_type="probe_flow", flow_id="12",
  step_id="0", step_name="登录接口"
} 1

# 步骤 1：查询商品
srehub_inspect_flow_step_exec_duration{
  task_id="50", task_name="下单流程编排",
  task_type="probe_flow", flow_id="12",
  step_id="1", step_name="查询商品"
} 0.032

# 步骤 2：提交订单（断言失败）
srehub_inspect_flow_step_assert_result{
  task_id="50", task_name="下单流程编排",
  task_type="probe_flow", flow_id="12",
  step_id="2", step_name="提交订单"
} 0
```

**告警建议：**
```promql
# 任意步骤失败
srehub_inspect_flow_step_status == 0

# 步骤断言不通过
srehub_inspect_flow_step_assert_result == 0

# 步骤失败率超过 10%
rate(srehub_inspect_flow_step_fail_total[10m]) /
  rate(srehub_inspect_flow_step_exec_total[10m]) > 0.1
```

---

## 九、智能巡检专属指标

> `task_type="inspect"`；每个巡检项 × 每台主机单独推送

| 指标名 | 类型 | 含义 | 单位/取值 |
|--------|------|------|----------|
| `srehub_inspect_check_pass_total` | Counter | 巡检项合规（通过）次数（累计） | 次 |
| `srehub_inspect_check_fail_total` | Counter | 巡检项不合规（不通过）次数（累计） | 次 |
| `srehub_inspect_check_abnormal_total` | Counter | 巡检异常数（累计） | 次 |
| `srehub_inspect_check_status` | Gauge | 巡检项是否通过（1=通过，0=不通过） | 0 或 1 |
| `srehub_inspect_check_duration_seconds` | Gauge | 巡检项执行耗时 | 秒 |
| `srehub_inspect_check_assertion_result` | Gauge | 巡检断言结果（1=通过，0=失败） | 0 或 1 |

**巡检专属扩展标签：**

| 标签名 | 含义 | 示例 |
|--------|------|------|
| `check_group` | 巡检组名称 | `Linux基线巡检` |
| `check_item` | 巡检项名称 | `检查SSH端口` |
| `check_level` | 巡检级别 | `high`、`medium`、`low` |
| `host_id` | 被巡检主机 ID | `101` |

**示例：**
```
srehub_inspect_check_status{
  task_id="60",
  task_name="Linux日常基线巡检",
  task_type="inspect",
  business_group="运维组",
  owner="sre-team",
  schedule_mode="scheduled",
  check_group="Linux基线巡检",
  check_item="检查SSH配置",
  check_level="high",
  host_id="101"
} 0
```

**告警建议：**
```promql
# 高级别巡检不通过
srehub_inspect_check_status{check_level="high"} == 0

# 巡检通过率低于 90%
sum(srehub_inspect_check_pass_total) /
  (sum(srehub_inspect_check_pass_total) + sum(srehub_inspect_check_fail_total)) < 0.9

# 某巡检组异常数激增
increase(srehub_inspect_check_abnormal_total{check_group="Linux基线巡检"}[1h]) > 5
```

---

## 十、统一标签体系

### 10.1 通用必选标签（所有指标强制携带）

| 标签名 | 含义 | 来源 | 示例 |
|--------|------|------|------|
| `task_id` | 任务唯一标识 | InspectionTask.ID | `42` |
| `task_name` | 任务名称 | InspectionTask.Name | `生产环境HTTP探测` |
| `task_type` | 任务类型 | ProbeConfig.Type 或 "inspect" | `ping`、`http`、`inspect` |
| `business_group` | 业务分组名称 | groupLookup(GroupID) | `运维组` |
| `owner` | 负责人 | InspectionTask.Owner | `alice` |
| `schedule_mode` | 触发方式 | TriggerType | `scheduled`、`manual` |

### 10.2 通用可选标签（拨测配置附带）

| 标签名 | 含义 | 适用类型 |
|--------|------|----------|
| `target` | 探测目标地址 | 所有拨测类型 |
| `probe_name` | 拨测配置名称 | 所有拨测类型 |

### 10.3 类型扩展标签

| 标签名 | 含义 | 适用类型 |
|--------|------|----------|
| `http_method` | HTTP 请求方法 | http、https |
| `http_path` | HTTP 请求路径 | http、https |
| `status_code` | HTTP 响应状态码 | http、https（仅 status_code_total） |
| `tls_version` | TLS 版本 | https |
| `ws_frame_type` | WebSocket 帧类型 | websocket |
| `handshake_code` | WebSocket 握手状态码 | websocket |
| `flow_id` | 编排配置 ID | probe_flow |
| `step_id` | 步骤序号 | probe_flow |
| `step_name` | 步骤名称 | probe_flow |
| `check_group` | 巡检组名称 | inspect |
| `check_item` | 巡检项名称 | inspect |
| `check_level` | 巡检级别 | inspect |
| `host_id` | 被巡检主机 ID | inspect |

---

## 十一、Counter 持久化机制（Redis）

### 11.1 设计原理

所有 `*_total` 后缀的 Counter 类指标通过 Redis 原子操作 `INCR` 持久化，确保应用重启后计数不丢失。

```
应用启动 → 执行任务 → INCR Redis key → 返回最新累计值 → 包装为 Gauge → 推送 PushGateway
                ↓
           应用重启
                ↓
       执行新任务 → INCR Redis key（从历史值继续累加，不归零）
```

### 11.2 Redis Key 格式

```
{prefix}:{metricName}:{sorted_label_k=v,...}
```

**示例：**
```
srehub:counter:srehub_inspect_task_exec_total:business_group=运维组,schedule_mode=scheduled,task_id=42,task_name=官网HTTP探测
srehub:counter:srehub_inspect_task_success_total:business_group=运维组,schedule_mode=scheduled,task_id=42,task_name=官网HTTP探测
srehub:counter:srehub_inspect_ping_packet_send_total:business_group=基础设施,schedule_mode=scheduled,task_id=10,task_name=核心服务器Ping
```

### 11.3 查看 Redis 中的 Counter

```bash
# 查看所有 Counter key
redis-cli KEYS "srehub:counter:*"

# 查看某个任务的执行总次数
redis-cli GET "srehub:counter:srehub_inspect_task_exec_total:business_group=运维组,schedule_mode=scheduled,task_id=42,task_name=官网HTTP探测"

# 查看某任务类型的所有 Counter
redis-cli KEYS "srehub:counter:*task_id=42*"
```

### 11.4 并发安全说明

- Redis `INCR` 命令是原子操作，天然支持多实例并发递增，无需额外分布式锁
- 不同标签组合的 Counter 独立存储，互不干扰

### 11.5 存储容量估算

| 场景 | 预估 Key 数量 |
|------|---------------|
| 100 个拨测任务 × 6 个 Counter 指标 | ~600 个 key |
| 50 个巡检任务 × 20 台主机 × 4 个 Counter | ~4000 个 key |
| 单个 key 大小 | < 200 bytes |
| 总存储（极端场景） | < 10 MB |

---

## 十二、PushGateway 推送说明

### 12.1 推送触发条件

- 任务调度中配置了 `pushgateway_id`（不为 0）
- Pushgateway 配置状态为启用（`status = 1`）
- 拨测/巡检任务执行完成后立即推送

### 12.2 推送 Job 名称

所有指标统一使用 `job="srehub"` 推送（替换原 `opshub_probe` / `opshub_inspection`）。

### 12.3 推送 Grouping 标签

PushGateway 的 grouping 标签用于唯一标识监控对象（新数据覆盖旧数据，不累积历史）：

```
instance = {hostname}
task_id  = {task.ID}
config_id = {config.ID}   # 拨测类
group_id  = {group.ID}    # 巡检类
item_id   = {item.ID}     # 巡检类
host_id   = {host.ID}     # 巡检类
```

### 12.4 Prometheus scrape 配置示例

```yaml
scrape_configs:
  - job_name: 'pushgateway'
    honor_labels: true
    static_configs:
      - targets: ['pushgateway:9091']
    metric_relabel_configs:
      # 只保留 srehub_inspect_* 指标
      - source_labels: [__name__]
        regex: 'srehub_inspect_.*'
        action: keep
```

---

## 十三、PromQL 示例查询

### 13.1 全局任务可用性

```promql
# 所有任务当前可用性
srehub_inspect_task_availability

# 按业务分组聚合可用性
avg(srehub_inspect_task_availability) by (business_group)

# 可用性低于 99% 的任务
srehub_inspect_task_availability < 0.99
```

### 13.2 拨测性能分析

```promql
# HTTP 响应时间 P95（近似，基于 Gauge 历史值）
quantile(0.95, srehub_inspect_http_response_duration_seconds)

# 按业务分组统计平均 HTTP 响应时间
avg(srehub_inspect_http_response_duration_seconds) by (business_group)

# Ping 丢包率最高的 5 个目标
topk(5, srehub_inspect_ping_loss_ratio)

# DNS 解析耗时超过 1 秒的探测
srehub_inspect_http_dns_duration_seconds > 1
```

### 13.3 证书监控

```promql
# 证书即将在 7 天内过期
srehub_inspect_https_cert_valid_days < 7

# 按任务名排序证书剩余天数
sort(srehub_inspect_https_cert_valid_days)
```

### 13.4 执行统计

```promql
# 各任务执行总次数
srehub_inspect_task_exec_total

# 失败率最高的 3 个任务
topk(3,
  srehub_inspect_task_fail_total /
  (srehub_inspect_task_success_total + srehub_inspect_task_fail_total)
)

# 重试次数趋势
srehub_inspect_task_retry_total
```

### 13.5 智能巡检

```promql
# 高级别巡检通过率
sum(srehub_inspect_check_pass_total{check_level="high"}) /
  (
    sum(srehub_inspect_check_pass_total{check_level="high"}) +
    sum(srehub_inspect_check_fail_total{check_level="high"})
  )

# 各巡检组异常数
sum(srehub_inspect_check_abnormal_total) by (check_group)

# 巡检执行耗时最长的巡检项
topk(10, srehub_inspect_check_duration_seconds)
```

### 13.6 业务编排

```promql
# 编排步骤失败率
sum(srehub_inspect_flow_step_fail_total) by (task_name, step_name) /
  sum(srehub_inspect_flow_step_exec_total) by (task_name, step_name)

# 步骤断言失败的编排任务
srehub_inspect_flow_step_assert_result == 0

# 步骤耗时超过 5 秒
srehub_inspect_flow_step_exec_duration > 5
```

---

## 十四、Grafana 面板建议

### 面板 1：任务总览（所有类型）

| 面板 | 类型 | PromQL |
|------|------|--------|
| 任务可用性热力图 | Heatmap | `srehub_inspect_task_availability` |
| 执行成功/失败比例 | Pie | `sum(srehub_inspect_task_success_total)` vs `sum(srehub_inspect_task_fail_total)` |
| 当前任务执行耗时 Top10 | Bar | `topk(10, srehub_inspect_task_exec_duration_seconds)` |

### 面板 2：拨测网络质量

| 面板 | 类型 | PromQL |
|------|------|--------|
| Ping RTT 趋势 | Time series | `srehub_inspect_ping_avg_rtt_seconds` |
| 丢包率分布 | Gauge | `srehub_inspect_ping_loss_ratio` |
| HTTP 响应时间 | Time series | `srehub_inspect_http_response_duration_seconds` |
| DNS/TLS/TTFB 瀑布图 | Bar | 各耗时指标组合 |
| 证书到期预警 | Table | `sort_desc(srehub_inspect_https_cert_valid_days)` |

### 面板 3：智能巡检大盘

| 面板 | 类型 | PromQL |
|------|------|--------|
| 巡检通过率（按分组） | Stat | 通过率计算 |
| 不通过巡检项列表 | Table | `srehub_inspect_check_status == 0` |
| 巡检异常趋势 | Time series | `srehub_inspect_check_abnormal_total` |

### 面板 4：业务编排监控

| 面板 | 类型 | PromQL |
|------|------|--------|
| 步骤成功率（按编排任务） | Table | 步骤成功率计算 |
| 失败步骤列表 | Table | `srehub_inspect_flow_step_status == 0` |
| 步骤耗时瀑布图 | Bar | `srehub_inspect_flow_step_exec_duration` |

---

## 附录：指标变更记录

| 版本 | 变更内容 |
|------|----------|
| v1.0 | 原始版本，使用 `opshub_probe_*` / `opshub_inspection_*` 命名 |
| v2.0 | 全面重构为 `srehub_inspect_*`，新增 Redis Counter 持久化，扩充 WS/UDP/业务编排/智能巡检专属指标，统一标签体系 |
