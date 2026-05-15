# 拨测指标 target 标签值修复报告

## 一、问题概述

修复了 31 个拨测类指标中 `target` 标签的两个问题：

### 问题1：变量未解析
**现象**：指标中的 `target` 值包含未解析的变量，如 `{{base_url}}/get_output`

**根本原因**：
- 调用 `pushMetrics`、`pushAppMetrics`、`pushWorkflowMetrics` 时传入的是 `origCfg`（原始配置）
- 而实际执行拨测时使用的是 `resolvedCfg`（变量解析后的配置）
- 导致指标中的 `target` 值是未解析的原始值

**影响指标**：所有拨测类指标（31个）

### 问题2：TCP/UDP 端口未拼接
**现象**：TCP 和 UDP 指标中的 `target` 只有 IP，没有端口

**根本原因**：
- `config.Target` 只包含 IP 地址
- 端口信息存储在 `config.Port` 字段中
- 指标推送时未将 IP 和端口拼接

**影响指标**：TCP 和 UDP 类指标（8个）

---

## 二、修复方案

### 方案概述
1. 修改调用时传入 `resolvedCfg`（解决变量未解析问题）
2. 在指标推送方法内部根据拨测类型构建正确的 `target` 值（解决端口拼接问题）

---

## 三、详细修改内容

### 修改1：调用时传入 `resolvedCfg`

**文件**：`internal/biz/inspection/executor.go`

#### 位置1：第249行（`executeAndSaveNetworkProbe`）
```go
// 修改前
e.pushMetrics(ctx, probeTask, origCfg, result, businessGroupNames)

// 修改后
e.pushMetrics(ctx, probeTask, resolvedCfg, result, businessGroupNames)
```

#### 位置2：第314行（`executeAndSaveAppProbe`）
```go
// 修改前
e.pushAppMetrics(ctx, probeTask, origCfg, appResult, retryAttempt, businessGroupNames)

// 修改后
e.pushAppMetrics(ctx, probeTask, resolvedCfg, appResult, retryAttempt, businessGroupNames)
```

#### 位置3：第576行（`executeAndSaveWorkflowProbe`）
```go
// 修改前
e.pushWorkflowMetrics(ctx, probeTask, origCfg, wfResult, businessGroupNames)

// 修改后
e.pushWorkflowMetrics(ctx, probeTask, resolvedCfg, wfResult, businessGroupNames)
```

---

### 修改2：`pushMetrics` 方法内部构建正确的 target 值

**文件**：`internal/biz/inspection/executor.go`

**位置**：约第1420-1450行

```go
// 修改前
allLabels := prometheus.Labels{
    "task_name":      task.Name,
    "task_type":      config.Type,
    "business_group": businessGroupName,
    "schedule_mode":  scheduleMode,
    "target":         config.Target,  // ❌ 问题1：未解析变量；问题2：TCP/UDP 缺少端口
    "probe_name":     config.Name,
}

// 修改后
// 构建正确的 target 值
targetValue := config.Target
switch config.Type {
case "tcp", "udp":
    // TCP/UDP：拼接 IP:Port
    if config.Port > 0 {
        targetValue = fmt.Sprintf("%s:%d", config.Target, config.Port)
    }
case "ping":
    // Ping：直接使用 Target（已解析变量）
    targetValue = config.Target
}

allLabels := prometheus.Labels{
    "task_name":      task.Name,
    "task_type":      config.Type,
    "business_group": businessGroupName,
    "schedule_mode":  scheduleMode,
    "target":         targetValue,  // ✅ 修复：使用构建后的值
    "probe_name":     config.Name,
}
```

**影响指标**（21个）：
1. `srehub_inspect_task_exec_total` - 通用指标
2. `srehub_inspect_task_success_total` - 通用指标
3. `srehub_inspect_task_fail_total` - 通用指标
4. `srehub_inspect_task_exec_duration_seconds` - 通用指标
5. `srehub_inspect_task_availability` - 通用指标
6. `srehub_inspect_task_availability_gauge` - 通用指标
7. `srehub_inspect_ping_avg_rtt_seconds` - Ping 专属
8. `srehub_inspect_ping_min_rtt_seconds` - Ping 专属
9. `srehub_inspect_ping_max_rtt_seconds` - Ping 专属
10. `srehub_inspect_ping_jitter_seconds` - Ping 专属
11. `srehub_inspect_ping_loss_ratio` - Ping 专属
12. `srehub_inspect_ping_packet_send_total` - Ping 专属
13. `srehub_inspect_ping_packet_recv_total` - Ping 专属
14. `srehub_inspect_tcp_connect_duration_seconds` - TCP 专属
15. `srehub_inspect_tcp_port_reachable` - TCP 专属
16. `srehub_inspect_tcp_connect_success_total` - TCP 专属
17. `srehub_inspect_tcp_connect_fail_total` - TCP 专属
18. `srehub_inspect_udp_transfer_delay_seconds` - UDP 专属
19. `srehub_inspect_udp_send_total` - UDP 专属
20. `srehub_inspect_udp_recv_total` - UDP 专属
21. `srehub_inspect_udp_loss_total` - UDP 专属

---

### 修改3：`pushAppMetrics` 方法内部构建正确的 target 值

**文件**：`internal/biz/inspection/executor.go`

**位置**：约第1642-1670行

```go
// 修改前
allLabels := prometheus.Labels{
    "task_name":      task.Name,
    "task_type":      config.Type,
    "business_group": businessGroupName,
    "schedule_mode":  scheduleMode,
    "target":         config.Target,  // ❌ 问题：未解析变量
    "probe_name":     config.Name,
    "http_method":    httpMethod,
    "http_path":      httpPath,
    "status_code":    statusCodeStr,
}

// 修改后
// 构建正确的 target 值
targetValue := config.Target
if config.URL != "" {
    // HTTP/HTTPS/WebSocket：使用完整的 URL（已解析变量）
    targetValue = config.URL
}

allLabels := prometheus.Labels{
    "task_name":      task.Name,
    "task_type":      config.Type,
    "business_group": businessGroupName,
    "schedule_mode":  scheduleMode,
    "target":         targetValue,  // ✅ 修复：使用 URL（已解析变量）
    "probe_name":     config.Name,
    "http_method":    httpMethod,
    "http_path":      httpPath,
    "status_code":    statusCodeStr,
}
```

**影响指标**（10个）：
1. `srehub_inspect_task_exec_total` - 通用指标
2. `srehub_inspect_task_success_total` - 通用指标
3. `srehub_inspect_task_fail_total` - 通用指标
4. `srehub_inspect_task_availability_gauge` - 通用指标
5. `srehub_inspect_task_exec_duration_seconds` - 通用指标
6. `srehub_inspect_task_availability` - 通用指标
7. `srehub_inspect_http_response_duration_seconds` - HTTP 专属
8. `srehub_inspect_http_dns_duration_seconds` - HTTP 专属
9. `srehub_inspect_http_tls_duration_seconds` - HTTP 专属
10. `srehub_inspect_http_first_byte_seconds` - HTTP 专属
11. `srehub_inspect_http_assertion_result` - HTTP 专属
12. `srehub_inspect_https_cert_valid_days` - HTTPS 专属
13. `srehub_inspect_http_status_code_total` - HTTP 专属
14. `srehub_inspect_ws_connection_established` - WebSocket 专属
15. `srehub_inspect_ws_handshake_duration_seconds` - WebSocket 专属
16. `srehub_inspect_ws_handshake_success_total` - WebSocket 专属
17. `srehub_inspect_ws_disconnect_total` - WebSocket 专属

---

## 四、修复效果

### 效果1：变量解析

#### Ping 类指标
**修复前**：
```prometheus
srehub_inspect_ping_avg_rtt_seconds{
    target="{{ping_host}}"  # ❌ 未解析变量
} 0.015
```

**修复后**：
```prometheus
srehub_inspect_ping_avg_rtt_seconds{
    target="192.168.1.1"  # ✅ 已解析变量
} 0.015
```

#### HTTP 类指标
**修复前**：
```prometheus
srehub_inspect_http_response_duration_seconds{
    target="{{base_url}}/get_output"  # ❌ 未解析变量
} 0.123
```

**修复后**：
```prometheus
srehub_inspect_http_response_duration_seconds{
    target="https://api.example.com/get_output"  # ✅ 已解析变量
} 0.123
```

---

### 效果2：TCP/UDP 端口拼接

#### TCP 类指标
**修复前**：
```prometheus
srehub_inspect_tcp_port_reachable{
    target="192.168.1.100"  # ❌ 缺少端口
} 1
```

**修复后**：
```prometheus
srehub_inspect_tcp_port_reachable{
    target="192.168.1.100:3306"  # ✅ 包含端口
} 1
```

#### UDP 类指标
**修复前**：
```prometheus
srehub_inspect_udp_transfer_delay_seconds{
    target="192.168.1.200"  # ❌ 缺少端口
} 0.005
```

**修复后**：
```prometheus
srehub_inspect_udp_transfer_delay_seconds{
    target="192.168.1.200:53"  # ✅ 包含端口
} 0.005
```

---

## 五、修改文件清单

**文件**：`internal/biz/inspection/executor.go`

**修改位置**：
1. 第249行：修改 `pushMetrics` 调用参数（`origCfg` → `resolvedCfg`）
2. 第314行：修改 `pushAppMetrics` 调用参数（`origCfg` → `resolvedCfg`）
3. 第576行：修改 `pushWorkflowMetrics` 调用参数（`origCfg` → `resolvedCfg`）
4. 约第1420-1450行：修改 `pushMetrics` 方法内部，构建正确的 `target` 值
5. 约第1642-1670行：修改 `pushAppMetrics` 方法内部，构建正确的 `target` 值

**修改行数**：约30行

---

## 六、影响指标汇总

**总计**：31个拨测类指标

### 通用指标（6个）- 所有拨测类型共享
1. `srehub_inspect_task_exec_total`
2. `srehub_inspect_task_success_total`
3. `srehub_inspect_task_fail_total`
4. `srehub_inspect_task_exec_duration_seconds`
5. `srehub_inspect_task_availability`
6. `srehub_inspect_task_availability_gauge`

### Ping 类（7个）
7. `srehub_inspect_ping_avg_rtt_seconds`
8. `srehub_inspect_ping_min_rtt_seconds`
9. `srehub_inspect_ping_max_rtt_seconds`
10. `srehub_inspect_ping_jitter_seconds`
11. `srehub_inspect_ping_loss_ratio`
12. `srehub_inspect_ping_packet_send_total`
13. `srehub_inspect_ping_packet_recv_total`

### TCP 类（4个）
14. `srehub_inspect_tcp_connect_duration_seconds`
15. `srehub_inspect_tcp_port_reachable`
16. `srehub_inspect_tcp_connect_success_total`
17. `srehub_inspect_tcp_connect_fail_total`

### UDP 类（4个）
18. `srehub_inspect_udp_transfer_delay_seconds`
19. `srehub_inspect_udp_send_total`
20. `srehub_inspect_udp_recv_total`
21. `srehub_inspect_udp_loss_total`

### HTTP/HTTPS 类（7个）
22. `srehub_inspect_http_response_duration_seconds`
23. `srehub_inspect_http_dns_duration_seconds`
24. `srehub_inspect_http_tls_duration_seconds`
25. `srehub_inspect_http_first_byte_seconds`
26. `srehub_inspect_http_assertion_result`
27. `srehub_inspect_https_cert_valid_days`
28. `srehub_inspect_http_status_code_total`

### WebSocket 类（4个）
29. `srehub_inspect_ws_connection_established`
30. `srehub_inspect_ws_handshake_duration_seconds`
31. `srehub_inspect_ws_handshake_success_total`
32. `srehub_inspect_ws_disconnect_total`

### Flow 类（5个）- 无 target 标签，不受影响
33. `srehub_inspect_flow_step_exec_total`
34. `srehub_inspect_flow_step_fail_total`
35. `srehub_inspect_flow_step_status`
36. `srehub_inspect_flow_step_exec_duration`
37. `srehub_inspect_flow_step_assert_result`

---

## 七、验证结果

### 7.1 编译验证
```bash
go build -o /dev/null ./cmd/... ./internal/... ./pkg/... ./plugins/...
```
✅ 编译通过，无错误

### 7.2 修改统计
- **修改文件数**：1个
- **修改位置数**：5处
- **修改行数**：约30行
- **影响指标数**：31个

---

## 八、注意事项

### 8.1 变量解析依赖
- 修复后，`target` 值依赖于变量解析功能正常工作
- 如果变量解析失败，`target` 值可能仍包含未解析的变量
- 建议在拨测配置中验证变量是否正确定义

### 8.2 历史数据
- 修复后历史数据中的 `target` 值不会自动更新
- 旧数据仍然是未解析的变量或缺少端口的值
- 如需修正历史数据，需通过 Prometheus remote_write 或重新采集

### 8.3 Grafana 查询调整
修复后 `target` 值格式变化，可能影响 Grafana 查询：

**变化1：变量解析**
- 修复前：`target="{{base_url}}/api"`
- 修复后：`target="https://api.example.com/api"`
- 影响：如果 Grafana 查询中使用了精确匹配，需要调整为模糊匹配或正则匹配

**变化2：TCP/UDP 端口拼接**
- 修复前：`target="192.168.1.100"`
- 修复后：`target="192.168.1.100:3306"`
- 影响：如果 Grafana 查询中使用了 IP 精确匹配，需要调整为正则匹配

**建议查询方式**：
```promql
# 使用正则匹配
srehub_inspect_tcp_port_reachable{target=~"192.168.1.100.*"}

# 或使用标签过滤
srehub_inspect_tcp_port_reachable{probe_name="MySQL健康检查"}
```

### 8.4 指标基数变化
修复后 `target` 值更精确，可能导致指标基数增加：

**场景1：同一个 IP 的不同端口**
- 修复前：`target="192.168.1.100"` → 1个时间序列
- 修复后：`target="192.168.1.100:3306"` 和 `target="192.168.1.100:6379"` → 2个时间序列

**场景2：同一个域名的不同路径**
- 修复前：`target="{{base_url}}/api"` → 可能多个配置共享同一个未解析的值
- 修复后：`target="https://api.example.com/api"` → 每个配置独立的解析值

**影响**：
- Prometheus 存储空间增加
- 查询性能可能下降
- 建议监控指标基数变化

### 8.5 告警规则调整
如果告警规则中使用了 `target` 标签，需要验证：

**示例告警规则**：
```yaml
# 修复前
- alert: TCPPortDown
  expr: srehub_inspect_tcp_port_reachable{target="192.168.1.100"} == 0
  
# 修复后（需要调整）
- alert: TCPPortDown
  expr: srehub_inspect_tcp_port_reachable{target="192.168.1.100:3306"} == 0
  
# 或使用正则匹配
- alert: TCPPortDown
  expr: srehub_inspect_tcp_port_reachable{target=~"192.168.1.100:.*"} == 0
```

---

## 九、测试建议

### 9.1 功能测试

#### 测试1：Ping 拨测变量解析
1. 创建 Ping 拨测配置，`Target` 使用变量：`{{ping_host}}`
2. 定义变量：`ping_host=192.168.1.1`
3. 执行拨测任务
4. 查看 Prometheus 指标：`srehub_inspect_ping_avg_rtt_seconds`
5. 验证 `target` 标签值是否为 `192.168.1.1`（已解析）

#### 测试2：TCP 拨测端口拼接
1. 创建 TCP 拨测配置，`Target=192.168.1.100`，`Port=3306`
2. 执行拨测任务
3. 查看 Prometheus 指标：`srehub_inspect_tcp_port_reachable`
4. 验证 `target` 标签值是否为 `192.168.1.100:3306`

#### 测试3：HTTP 拨测变量解析
1. 创建 HTTP 拨测配置，`URL` 使用变量：`{{base_url}}/api/health`
2. 定义变量：`base_url=https://api.example.com`
3. 执行拨测任务
4. 查看 Prometheus 指标：`srehub_inspect_http_response_duration_seconds`
5. 验证 `target` 标签值是否为 `https://api.example.com/api/health`（已解析）

#### 测试4：UDP 拨测端口拼接
1. 创建 UDP 拨测配置，`Target=192.168.1.200`，`Port=53`
2. 执行拨测任务
3. 查看 Prometheus 指标：`srehub_inspect_udp_transfer_delay_seconds`
4. 验证 `target` 标签值是否为 `192.168.1.200:53`

### 9.2 兼容性测试

#### 测试1：无变量的拨测配置
1. 创建拨测配置，不使用变量
2. 验证指标是否正常生成

#### 测试2：变量解析失败
1. 创建拨测配置，使用未定义的变量
2. 验证指标中的 `target` 值（可能仍包含未解析的变量）

#### 测试3：端口为0的TCP/UDP
1. 创建 TCP/UDP 拨测配置，`Port=0`
2. 验证 `target` 值是否只包含 IP（不拼接端口）

### 9.3 Grafana 看板测试
1. 打开现有的拨测监控看板
2. 验证所有面板是否正常显示数据
3. 检查 `target` 变量下拉框是否正常
4. 验证告警规则是否正常触发

---

## 十、后续优化建议

### 10.1 变量解析增强
- 添加变量解析失败的日志记录
- 在指标中添加 `variable_resolved` 标签，标识变量是否成功解析
- 支持变量解析失败时的降级策略（使用原始值或默认值）

### 10.2 指标优化
- 考虑将 `target` 拆分为多个标签：`host`、`port`、`path` 等
- 减少指标基数，提高查询性能
- 添加 `target_type` 标签，标识 target 的类型（IP、域名、URL）

### 10.3 监控告警
- 添加指标基数监控，及时发现异常增长
- 添加变量解析失败的告警
- 监控 `target` 标签值的分布情况

---

## 十一、相关文档

- 原始问题报告：用户反馈
- 变量解析功能文档：`internal/biz/inspection/variable_resolver.go`
- 指标规范文档：Prometheus 指标命名规范
- 业务分组修复报告：`docs/fix-business-group-metrics.md`

---

## 十二、修复时间线

- **问题发现**：2026-05-15
- **方案设计**：2026-05-15
- **代码修复**：2026-05-15
- **编译验证**：2026-05-15 ✅
- **状态**：已完成，待测试验证

---

**修复完成日期**：2026-05-15  
**修复人员**：Claude (Opus 4.6)  
**审核状态**：待用户验证
