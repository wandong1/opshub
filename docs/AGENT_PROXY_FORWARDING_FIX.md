# Agent 代理转发修复 - 真正通过 Agent 转发所有请求

**修复日期**: 2026-04-13  
**问题类型**: 架构设计缺陷  
**状态**: ✅ 完成

---

## 🔴 问题发现

用户在测试 Agent 代理数据源时发现：**测试和查询并不是真正通过 Agent 转发**。

### 错误的实现

```go
// 测试功能（错误）
testURL := ds.URL + testPath  // 直接使用数据源 URL
req, _ := http.NewRequest("GET", testURL, nil)
client.Do(req)  // 直接从服务端访问数据源

// 告警查询（错误）
baseURL := strings.TrimRight(ds.URL, "/")
reqURL := fmt.Sprintf("%s/api/v1/query?%s", baseURL, params.Encode())
// 直接访问数据源 URL
```

### 问题分析

1. **测试功能**：直接在服务端发起 HTTP 请求到数据源 URL
2. **告警查询**：告警规则查询也是直接访问数据源 URL
3. **违背设计**：Agent 代理模式的目的是通过 Agent 访问内网资源，但实际上绕过了 Agent

---

## ✅ 正确的设计

### Agent 代理模式应该

```
所有请求 → 代理转发 URL → 代理处理器 → Agent → 内网数据源
```

### 关键点

1. **统一入口**：所有请求都通过代理转发 URL
2. **代理处理器**：统一处理认证、Agent 选择、请求转发
3. **真正转发**：通过 Agent 访问内网资源

---

## 🛠️ 修复方案

### 1. 测试功能修复

**文件**: `internal/server/alert/datasource_handler.go`

```go
// ❌ 之前（错误）
testURL := ds.URL + testPath
req, _ := http.NewRequest("GET", testURL, nil)
// 添加认证
if ds.Token != "" {
    req.Header.Set("Authorization", "Bearer "+ds.Token)
}
client.Do(req)  // 直接访问数据源

// ✅ 现在（正确）
// 使用代理转发 URL
proxyURL := ds.ProxyURL + testPath
testURL := "http://localhost:9876" + proxyURL
req, _ := http.NewRequest("GET", testURL, nil)
// 不添加认证（由代理处理器添加）
client.Do(req)  // 经过代理处理器转发
```

**关键改进**：
- 使用 `ds.ProxyURL`（格式：`/api/v1/alert/proxy/datasource/{token}`）
- 请求本地服务器的代理端点
- 不添加认证信息（由代理处理器统一处理）

---

### 2. 告警规则查询修复

**文件**: `internal/service/alert/datasource_query.go`

#### Prometheus/VictoriaMetrics 查询

```go
func queryPrometheus(ds *biz.AlertDataSource, expr string) ([]QueryResult, error) {
    // ✅ 根据接入方式构建 URL
    var baseURL string
    if ds.AccessMode == "agent" {
        // Agent 代理模式：使用代理转发 URL
        baseURL = "http://localhost:9876" + ds.ProxyURL
    } else {
        // 直连模式：直接使用数据源 URL
        baseURL = strings.TrimRight(ds.URL, "/")
    }

    params := url.Values{}
    params.Set("query", expr)
    params.Set("time", fmt.Sprintf("%d", time.Now().Unix()))

    reqURL := fmt.Sprintf("%s/api/v1/query?%s", baseURL, params.Encode())

    req, err := http.NewRequest("GET", reqURL, nil)
    if err != nil {
        return nil, err
    }

    // ✅ 直连模式才需要添加认证（Agent 模式由代理处理器添加）
    if ds.AccessMode == "direct" {
        if ds.Token != "" {
            req.Header.Set("Authorization", "Bearer "+ds.Token)
        } else if ds.Username != "" {
            req.SetBasicAuth(ds.Username, ds.Password)
        }
    }

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    // ...
}
```

#### InfluxDB 查询

```go
func queryInfluxDB(ds *biz.AlertDataSource, expr string) ([]QueryResult, error) {
    // ✅ 根据接入方式构建 URL
    var baseURL string
    if ds.AccessMode == "agent" {
        // Agent 代理模式：使用代理转发 URL
        baseURL = "http://localhost:9876" + ds.ProxyURL
    } else {
        // 直连模式：直接使用数据源 URL
        baseURL = strings.TrimRight(ds.URL, "/")
    }

    reqURL := fmt.Sprintf("%s/query", baseURL)

    params := url.Values{}
    params.Set("q", expr)

    req, err := http.NewRequest("POST", reqURL, strings.NewReader(params.Encode()))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    // ✅ 直连模式才需要添加认证
    if ds.AccessMode == "direct" {
        if ds.Token != "" {
            req.Header.Set("Authorization", "Token "+ds.Token)
        } else if ds.Username != "" {
            req.SetBasicAuth(ds.Username, ds.Password)
        }
    }

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    // ...
}
```

---

## 📊 完整的数据流

### 测试数据源（Agent 代理模式）

```
1. 用户点击"测试"按钮
   ↓
2. 后端构建代理 URL
   proxyURL = "/api/v1/alert/proxy/datasource/{token}/api/v1/query?query=up"
   ↓
3. 发起请求到本地服务器
   testURL = "http://localhost:9876" + proxyURL
   ↓
4. 请求到达代理处理器
   proxyDataSourceRequest(c *gin.Context)
   ↓
5. 代理处理器查询 ProxyToken
   ds, _ := s.dsRepo.GetByProxyToken(ctx, proxyToken)
   ↓
6. 获取在线 Agent
   rels, _ := s.dsAgentRelationRepo.ListByDataSourceID(ctx, ds.ID)
   selectedRel := 选择第一个在线的 Agent
   ↓
7. 构建目标 URL
   targetURL = ds.URL + "/api/v1/query?query=up"
   ↓
8. 添加认证信息（从数据源配置）
   if ds.Token != "" {
       req.Header.Set("Authorization", "Bearer "+ds.Token)
   }
   ↓
9. 通过 Agent 转发到内网数据源
   forwardToAgent(c, selectedRel.AgentHostID, targetURL, ds)
   ↓
10. 返回响应给客户端
```

### 告警规则查询（Agent 代理模式）

```
1. 告警评估触发
   evalRule(rule)
   ↓
2. 调用数据源查询
   QueryDataSource(ds, expr)
   ↓
3. 检查接入方式
   if ds.AccessMode == "agent"
   ↓
4. 使用代理 URL
   baseURL = "http://localhost:9876" + ds.ProxyURL
   ↓
5. 构建完整请求 URL
   reqURL = baseURL + "/api/v1/query?query=" + expr
   ↓
6. 发起请求（不添加认证）
   req, _ := http.NewRequest("GET", reqURL, nil)
   ↓
7. 请求到达代理处理器
   （同上面的流程 4-10）
   ↓
8. 返回查询结果
   parsePrometheusResponse(body)
```

### Grafana 访问（Agent 代理模式）

```
1. Grafana 配置数据源
   URL = http://opshub:9876/api/v1/alert/proxy/datasource/{token}
   ↓
2. Grafana 发起查询
   GET /api/v1/alert/proxy/datasource/{token}/api/v1/query?query=up
   ↓
3. 请求到达代理处理器
   （同上面的流程 4-10）
   ↓
4. 返回数据给 Grafana
```

---

## 🎯 关键改进

### 1. 统一的转发机制

**之前**：
- 测试：直接访问数据源
- 查询：直接访问数据源
- Grafana：通过代理转发

**现在**：
- 测试：通过代理转发 ✅
- 查询：通过代理转发 ✅
- Grafana：通过代理转发 ✅

### 2. 真正的 Agent 转发

所有 Agent 代理模式的请求都经过：
```
请求 → 代理处理器 → Agent → 内网数据源
```

### 3. 认证处理正确

- **直连模式**：查询时添加认证
- **Agent 模式**：查询时不添加认证，由代理处理器统一添加

### 4. 符合设计初衷

Agent 代理模式真正实现了：
- 通过 Agent 访问内网资源
- 服务端无法直接访问内网
- 统一的代理转发机制

---

## ✅ 验证清单

- [x] Go 编译成功
- [x] 测试功能使用代理转发
- [x] 告警查询使用代理转发
- [x] 认证处理正确
- [x] 直连模式不受影响
- [x] Agent 模式真正通过 Agent
- [x] 代码逻辑清晰

---

## 📝 修改文件

| 文件 | 修改内容 |
|------|---------|
| `internal/server/alert/datasource_handler.go` | 测试功能使用代理 URL |
| `internal/service/alert/datasource_query.go` | 查询功能根据接入方式选择 URL |

---

## 🚀 现在可以

✅ **测试 Agent 代理数据源**
- 真正通过 Agent 转发
- 经过代理处理器
- 认证由代理处理器添加

✅ **告警规则查询 Agent 代理数据源**
- 真正通过 Agent 转发
- 统一的转发机制
- 与测试、Grafana 访问一致

✅ **Grafana 访问 Agent 代理数据源**
- 通过代理 URL
- 与测试、查询使用相同的转发机制

✅ **所有请求统一处理**
- 测试、查询、Grafana 访问都通过代理处理器
- Agent 真正发挥作用
- 符合架构设计

