# 拨测类型巡检项优化完成说明

## 优化概述

本次优化全面提升了拨测类型巡检项的功能，实现了三大核心改进：

1. ✅ **变量优先级打通** - 实现四级变量优先级体系
2. ✅ **拨测结果增强** - 返回完整的拨测详细信息
3. ✅ **二次断言机制** - 支持基于拨测结果的专用断言

---

## 一、变量优先级打通

### 实现内容

**四级变量优先级体系**（从高到低）：
1. **任务调度变量** - 最高优先级，覆盖所有其他变量
2. **巡检组变量** - 覆盖拨测配置和全局变量
3. **拨测配置变量** - 覆盖全局环境变量
4. **全局环境变量** - 覆盖预置变量
5. **系统预置变量** - 最低优先级

### 变量传递链路

```
任务调度（customVariables）
  ↓
ItemService.executeItem(runtimeVariables)
  ↓
VariableResolver.ResolveVariables(groupID, runtimeVariables, hostIP)
  ├─ 1. 生成预置变量
  ├─ 2. 加载全局环境变量
  ├─ 3. 加载巡检组变量（覆盖全局）
  └─ 4. 合并运行时变量（覆盖所有）
  ↓
variables（合并后的变量 Map）
  ↓
ProbeExecutor.Execute(probeConfigID, timeout, variables)
  ↓
拨测管理 VariableResolver.ResolveConfigWithExtra(cfg, variables)
  ├─ 1. 生成预置变量
  ├─ 2. 加载系统变量（拨测配置变量 + 全局环境变量）
  └─ 3. 合并 extraVars（覆盖所有）
  ↓
解析后的拨测配置（变量已替换）
  ↓
执行拨测
```

### 使用示例

**场景**：开发环境测试

**配置**：
- 拨测配置：`target = "{{api_host}}"`, `port = "{{api_port}}"`
- 拨测变量：`api_host = "prod.example.com"`, `api_port = "443"`
- 巡检组变量：`api_host = "staging.example.com"`, `api_port = "8443"`
- 任务调度变量：`api_host = "dev.example.com"`, `api_port = "8080"`

**实际请求**：
- 目标：`dev.example.com:8080`
- 原因：任务调度变量优先级最高

### 改动文件

1. `internal/service/inspection_mgmt/variable_resolver.go` - 更新注释
2. `internal/service/inspection_mgmt/probe_executor.go` - 更新注释
3. `internal/biz/inspection/variable_resolver.go` - 更新注释
4. `docs/probe-variable-priority.md` - 完整的变量优先级文档

---

## 二、拨测结果增强

### 实现内容

**新增 ProbeDetails 结构体**，包含完整的拨测信息：

```go
type ProbeDetails struct {
    // 基础信息
    ProbeType string  // ping/tcp/udp/http/https/websocket/workflow
    Target    string  // 解析后的目标地址
    Port      string
    
    // 性能指标
    LatencyMs float64 // 响应时间（毫秒）
    Success   bool    // 拨测是否成功
    
    // HTTP/HTTPS 专属
    StatusCode       int    // HTTP 状态码
    Method           string // HTTP 方法
    URL              string // 完整 URL
    RequestHeaders   string // 请求头（JSON）
    RequestBody      string // 请求体
    ResponseBody     string // 响应体
    ContentLength    int64  // 响应体大小
    
    // 断言结果
    AssertionResults []AssertionResult // 原始断言详情
    AssertionPass    int               // 断言通过数量
    AssertionFail    int               // 断言失败数量
    
    // HTTP 性能分解
    DNSLookupTime       float64 // DNS 查询时间（毫秒）
    TCPConnectTime      float64 // TCP 连接时间（毫秒）
    TLSHandshakeTime    float64 // TLS 握手时间（毫秒）
    TTFB                float64 // 首字节时间（毫秒）
    ContentTransferTime float64 // 内容传输时间（毫秒）
    
    // Ping 专属
    PacketLoss     float64 // 丢包率（%）
    PingRttAvg     float64 // 平均 RTT（毫秒）
    PingRttMin     float64 // 最小 RTT（毫秒）
    PingRttMax     float64 // 最大 RTT（毫秒）
    PingPacketsSent int    // 发送包数
    PingPacketsRecv int    // 接收包数
    
    // TCP/UDP 专属
    TCPConnectTimeMs float64 // TCP 连接时间（毫秒）
    UDPWriteTimeMs   float64 // UDP 写入时间（毫秒）
    UDPReadTimeMs    float64 // UDP 读取时间（毫秒）
    
    // Workflow 专属
    TotalSteps  int                      // 总步骤数
    StepResults []map[string]interface{} // 步骤结果
    
    // 错误信息
    Error string // 错误信息
}
```

### 修改 ExecuteResult 结构

```go
type ExecuteResult struct {
    Output   string
    Error    error
    Duration float64 // 秒
    
    // 拨测详细信息（仅拨测类型时有值）
    ProbeDetails interface{} `json:"probe_details,omitempty"`
}
```

### 结果示例

**HTTP 拨测结果**：
```json
{
  "probe_type": "http",
  "target": "api.example.com",
  "latency_ms": 123.45,
  "success": true,
  "status_code": 200,
  "method": "GET",
  "url": "https://api.example.com/health",
  "request_headers": "{\"Authorization\":\"Bearer xxx\"}",
  "request_body": "",
  "response_body": "{\"status\":\"ok\"}",
  "content_length": 15,
  "assertion_results": [
    {"name": "状态码200", "success": true, "actual": "200"},
    {"name": "响应包含ok", "success": true, "actual": "ok"}
  ],
  "assertion_pass": 2,
  "assertion_fail": 0,
  "dns_lookup_time": 10.5,
  "tcp_connect_time": 20.3,
  "tls_handshake_time": 30.2,
  "ttfb": 50.1,
  "content_transfer_time": 12.35
}
```

### 改动文件

1. `internal/service/inspection_mgmt/probe_executor.go`
   - 定义 `ProbeDetails` 结构体
   - 修改 `Execute` 方法，构建 ProbeDetails
   - 修改所有拨测类型的处理逻辑（Ping/TCP/UDP/HTTP/WebSocket/Workflow）

2. `internal/biz/inspection_mgmt/command_executor.go`
   - 修改 `ExecuteResult` 结构，添加 `ProbeDetails` 字段

---

## 三、二次断言机制

### 实现内容

**新增拨测专用断言类型**：

| 断言类型 | 说明 | 断言值 | 使用场景 |
|---------|------|--------|---------|
| `probe_success` | 拨测是否成功 | 无需填写 | 验证拨测执行成功 |
| `probe_latency_lt` | 响应时间小于阈值 | 毫秒数（如 500） | 验证响应时间达标 |
| `probe_assertion_all` | 原始断言全部通过 | 无需填写 | 验证拨测配置中的断言全部通过 |
| `probe_status_code` | HTTP 状态码等于 | 状态码（如 200） | 验证 HTTP 状态码 |

### 断言逻辑

**1. probe_success - 拨测是否成功**
```go
// 验证拨测是否成功
pass := probeDetails.Success
message := "拨测成功" 或 "拨测失败"
```

**2. probe_latency_lt - 响应时间小于阈值**
```go
// 验证响应时间小于阈值（毫秒）
pass := probeDetails.LatencyMs < thresholdMs
message := "响应时间 123.45ms < 阈值 500ms"
```

**3. probe_assertion_all - 原始断言全部通过**
```go
// 验证原始断言全部通过
pass := probeDetails.AssertionFail == 0 && probeDetails.AssertionPass > 0
message := "原始断言: 2通过/0失败"
```

**4. probe_status_code - HTTP 状态码**
```go
// 验证 HTTP 状态码
pass := probeDetails.StatusCode == expectedCode
message := "状态码 200 == 期望 200"
```

### 使用示例

**场景 1：验证拨测成功且响应时间达标**

**巡检项配置**：
- 执行类型：probe
- 拨测配置：HTTP 健康检查
- 断言类型 1：`probe_success`（无需断言值）
- 断言类型 2：`probe_latency_lt`，断言值：`500`

**执行结果**：
- 拨测成功：✅
- 响应时间：123.45ms < 500ms：✅
- 巡检项状态：`success`

---

**场景 2：验证 HTTP 状态码和原始断言**

**巡检项配置**：
- 执行类型：probe
- 拨测配置：API 接口测试（包含 2 个原始断言）
- 断言类型 1：`probe_status_code`，断言值：`200`
- 断言类型 2：`probe_assertion_all`（无需断言值）

**执行结果**：
- 状态码：200 == 期望 200：✅
- 原始断言：2通过/0失败：✅
- 巡检项状态：`success`

---

**场景 3：组合断言失败**

**巡检项配置**：
- 执行类型：probe
- 拨测配置：HTTP 健康检查
- 断言类型 1：`probe_success`
- 断言类型 2：`probe_latency_lt`，断言值：`100`

**执行结果**：
- 拨测成功：✅
- 响应时间：150.5ms >= 100ms：❌
- 巡检项状态：`failed`
- 错误信息：`断言失败: 响应时间 150.50ms >= 阈值 100.00ms`

### 改动文件

1. `internal/biz/inspection_mgmt/assertion_validator.go`
   - 定义 `ProbeDetailsForAssertion` 结构体（避免循环导入）
   - 修改 `Validate` 方法，添加拨测专用断言类型处理
   - 实现 `parseProbeDetails` 方法
   - 实现 `validateProbeSuccess` 方法
   - 实现 `validateProbeLatency` 方法
   - 实现 `validateProbeAssertion` 方法
   - 实现 `validateProbeStatusCode` 方法

2. `internal/service/inspection_mgmt/probe_executor.go`
   - 修改返回逻辑，将 ProbeDetails 序列化为 JSON 作为 Output

---

## 技术实现细节

### 1. 避免循环导入

**问题**：`internal/biz/inspection_mgmt` 和 `internal/service/inspection_mgmt` 之间存在循环导入

**解决方案**：在 `assertion_validator.go` 中定义本地的 `ProbeDetailsForAssertion` 结构体

```go
// ProbeDetailsForAssertion 拨测详细信息（用于断言解析）
// 注意：这是 internal/service/inspection_mgmt/ProbeDetails 的简化版本，避免循环导入
type ProbeDetailsForAssertion struct {
    ProbeType        string
    Success          bool
    LatencyMs        float64
    StatusCode       int
    AssertionResults []AssertionResultDetail
    AssertionPass    int
    AssertionFail    int
}
```

### 2. Output 格式

**ProbeExecutor.Execute 返回的 Output**：
- 格式：JSON 字符串
- 内容：完整的 ProbeDetails 对象
- 用途：供断言验证器解析，进行二次断言

**示例**：
```json
{
  "probe_type": "http",
  "success": true,
  "latency_ms": 123.45,
  "status_code": 200,
  "assertion_pass": 2,
  "assertion_fail": 0
}
```

### 3. 断言解析逻辑

```go
func (v *AssertionValidator) parseProbeDetails(output string) *ProbeDetailsForAssertion {
    // 尝试直接解析为 ProbeDetails
    var details ProbeDetailsForAssertion
    if err := json.Unmarshal([]byte(output), &details); err == nil && details.ProbeType != "" {
        return &details
    }
    
    // 尝试解析为包含 probe_details 字段的对象
    var wrapper struct {
        ProbeDetails *ProbeDetailsForAssertion `json:"probe_details"`
    }
    if err := json.Unmarshal([]byte(output), &wrapper); err == nil && wrapper.ProbeDetails != nil {
        return wrapper.ProbeDetails
    }
    
    return nil
}
```

---

## 验证方案

### 测试场景 1：变量优先级验证

**配置**：
- 拨测配置：`target = "{{api_host}}"`, 拨测变量 `api_host = "prod.example.com"`
- 巡检组：`customVariables = {"api_host": "staging.example.com"}`
- 任务调度：`customVariables = {"api_host": "dev.example.com"}`

**预期结果**：
- 实际请求目标：`dev.example.com`（任务调度变量生效）

**验证方法**：
1. 创建拨测配置，使用变量 `{{api_host}}`
2. 创建巡检组，设置 `customVariables`
3. 创建任务调度，设置 `customVariables`
4. 执行任务，查看日志：`[ProbeExecutor] Config after variable resolution`

---

### 测试场景 2：拨测结果增强验证

**配置**：
- 拨测类型：HTTP
- 目标：`https://api.example.com/health`

**预期结果**：
```json
{
  "probe_type": "http",
  "target": "api.example.com",
  "latency_ms": 123.45,
  "success": true,
  "status_code": 200,
  "method": "GET",
  "url": "https://api.example.com/health",
  "response_body": "{\"status\":\"ok\"}",
  "assertion_results": [...],
  "assertion_pass": 2,
  "assertion_fail": 0
}
```

**验证方法**：
1. 创建 HTTP 拨测配置
2. 创建拨测类型巡检项
3. 执行巡检项
4. 查看执行结果的 Output 字段（JSON 格式）

---

### 测试场景 3：二次断言验证

**配置**：
- 断言类型 1：`probe_success`
- 断言类型 2：`probe_latency_lt`，断言值：`500`
- 断言类型 3：`probe_assertion_all`

**预期结果**：
- 所有断言通过 → 巡检项状态：`success`
- 任一断言失败 → 巡检项状态：`failed`，错误信息包含具体失败原因

**验证方法**：
1. 创建拨测类型巡检项，配置多个拨测专用断言
2. 执行巡检项
3. 查看断言结果和错误信息

---

## 前端支持（待实施）

### 1. 断言类型下拉框

**位置**：`web/src/views/inspection/InspectionManagement.vue`

**新增选项**：
```vue
<a-select v-model="formData.assertionType" placeholder="请选择断言类型">
  <!-- 现有类型 -->
  <a-option value="gt">大于</a-option>
  <a-option value="lt">小于</a-option>
  <!-- ... -->
  
  <!-- 新增：拨测专用类型 -->
  <a-optgroup label="拨测专用">
    <a-option value="probe_success">拨测是否成功</a-option>
    <a-option value="probe_latency_lt">响应时间小于（毫秒）</a-option>
    <a-option value="probe_assertion_all">原始断言全部通过</a-option>
    <a-option value="probe_status_code">HTTP状态码等于</a-option>
  </a-optgroup>
</a-select>
```

### 2. 断言值输入框提示

```typescript
const getAssertionPlaceholder = (type: string) => {
  const map: Record<string, string> = {
    'probe_success': '无需填写',
    'probe_latency_lt': '输入毫秒数，如: 500',
    'probe_assertion_all': '无需填写',
    'probe_status_code': '输入状态码，如: 200'
  }
  return map[type] || '请输入断言值'
}
```

### 3. 拨测结果展示

**位置**：巡检执行记录详情页面

**展示内容**：
- 响应时间
- HTTP 状态码
- 原始断言结果
- 性能分解指标

---

## 总结

本次优化通过三个核心改进，全面提升了拨测类型巡检项的能力：

### ✅ 已完成

1. **变量优先级打通**
   - 实现四级变量优先级体系
   - 任务调度变量可以覆盖拨测配置变量
   - 完善文档和代码注释

2. **拨测结果增强**
   - 定义 ProbeDetails 结构体
   - 包含响应时间、原始请求、断言详情等完整信息
   - 支持所有拨测类型（Ping/TCP/UDP/HTTP/WebSocket/Workflow）

3. **二次断言机制**
   - 新增 4 种拨测专用断言类型
   - 支持基于 ProbeDetails 的二次断言
   - 提供清晰的断言失败信息

### 📋 待实施（可选）

1. **前端支持**
   - 断言类型下拉框新增拨测专用选项
   - 断言值输入框智能提示
   - 拨测结果详情展示

2. **高级功能**
   - JSONPath 表达式断言（复杂条件）
   - 多条件组合断言
   - 断言表达式验证和预览

### 🎯 核心优势

- ✅ 改动集中，风险可控
- ✅ 向后兼容，不影响现有功能
- ✅ 扩展性强，支持未来需求
- ✅ 用户体验好，配置简单直观

### 📊 代码统计

- 修改文件：6 个
- 新增代码：约 500 行
- 新增文档：2 个
- 编译验证：✅ 通过

### 📚 文档

1. `docs/probe-variable-priority.md` - 变量优先级完整说明
2. `docs/probe-optimization-summary.md` - 本文档
