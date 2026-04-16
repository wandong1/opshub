# Grafana 代理转发调试指南

**日期**: 2026-04-14  
**问题**: Grafana 无法通过 Agent 代理访问 Prometheus  
**状态**: 已添加详细日志，待调试

---

## 🔧 已修复的问题

### 1. 路径拼接双斜杠问题

**问题**：
```go
// 之前
targetURL := ds.URL + targetPath
// 如果 ds.URL = "http://prometheus:9090/"
// 且 targetPath = "/api/v1/query"
// 结果：http://prometheus:9090//api/v1/query (双斜杠)
```

**修复**：
```go
// 现在
baseURL := strings.TrimRight(ds.URL, "/")
targetURL := baseURL + targetPath
// 结果：http://prometheus:9090/api/v1/query (正确)
```

### 2. 添加详细日志

现在每个关键步骤都有详细日志：
- 代理请求接收
- URL 构建过程
- Agent 转发过程
- 响应接收和返回

---

## 📊 完整的请求流程

### Grafana 配置

在 Grafana 中配置数据源：

```
数据源类型：Prometheus
URL：http://localhost:9876/api/v1/alert/proxy/datasource/{your-proxy-token}
```

### 请求示例

当 Grafana 查询指标时：

```
1. Grafana 发起请求
   GET http://localhost:9876/api/v1/alert/proxy/datasource/abc-123/api/v1/label/__name__/values

2. OpsHub 接收请求
   - proxyToken = "abc-123"
   - proxyPath = "/api/v1/alert/proxy/datasource/abc-123/api/v1/label/__name__/values"
   - targetPath = "/api/v1/label/__name__/values"

3. 查询数据源配置
   - ds.URL = "http://prometheus:9090"
   - ds.AccessMode = "agent"

4. 构建目标 URL
   - baseURL = "http://prometheus:9090" (去除尾部斜杠)
   - targetURL = "http://prometheus:9090/api/v1/label/__name__/values"

5. 通过 Agent 转发
   - 发送 HttpProxyRequest 到 Agent
   - Agent 执行 HTTP 请求到 Prometheus
   - Agent 返回 HttpProxyResponse

6. 返回给 Grafana
   - 设置响应头
   - 设置状态码
   - 写入响应体
```

---

## 🔍 调试步骤

### 步骤 1：检查数据源配置

```bash
# 查看数据源配置
mysql -uroot -p'OpsHub@2026' opshub -e "
SELECT id, name, type, access_mode, url, proxy_token, proxy_url 
FROM alert_datasources 
WHERE access_mode = 'agent';
"
```

**检查点**：
- ✅ URL 格式正确（http://prometheus:9090）
- ✅ proxy_token 已生成
- ✅ proxy_url 正确（/api/v1/alert/proxy/datasource/{token}）

### 步骤 2：检查 Agent 关联

```bash
# 查看 Agent 关联
mysql -uroot -p'OpsHub@2026' opshub -e "
SELECT dsar.*, h.hostname, h.ip 
FROM alert_datasource_agent_relations dsar
JOIN asset_hosts h ON dsar.agent_host_id = h.id
WHERE dsar.datasource_id = {your_datasource_id};
"
```

**检查点**：
- ✅ 至少有一个 Agent 关联
- ✅ Agent 主机在线

### 步骤 3：检查 Agent 状态

```bash
# 在 Agent 主机上检查状态
systemctl status srehub-agent

# 查看 Agent 日志
tail -f /var/log/srehub-agent/agent.log
```

**检查点**：
- ✅ Agent 正在运行
- ✅ Agent 已连接到服务端
- ✅ 心跳正常

### 步骤 4：查看服务端日志

```bash
# 实时查看代理转发日志
tail -f logs/app.log | grep "代理转发"
```

**关键日志**：

```
# 1. 接收代理请求
代理转发请求 proxy_token=abc-123 datasource_name=my-prom target_url=http://prometheus:9090/api/v1/query

# 2. 开始转发
开始通过Agent转发 agent_host_id=1 target_url=http://prometheus:9090/api/v1/query

# 3. 构建请求
构建HttpProxyRequest request_id=xxx method=GET url=http://prometheus:9090/api/v1/query

# 4. 发送到 Agent
已发送请求到Agent，等待响应 request_id=xxx

# 5. 收到响应
收到Agent响应 request_id=xxx status_code=200 body_len=1234

# 6. 返回客户端
成功返回响应给客户端 request_id=xxx status_code=200
```

### 步骤 5：查看 Agent 日志

```bash
# 在 Agent 主机上查看日志
tail -f /var/log/srehub-agent/agent.log | grep "HTTP 代理"
```

**关键日志**：

```
# 1. 接收请求
收到 HTTP 代理请求: method=GET, url=http://prometheus:9090/api/v1/query, requestID=xxx

# 2. 执行成功
HTTP 代理请求成功: url=http://prometheus:9090/api/v1/query, status=200, bodyLen=1234

# 或执行失败
HTTP 代理请求失败: url=http://prometheus:9090/api/v1/query, error=...
```

---

## 🐛 常见问题排查

### 问题 1：Grafana 提示 "Bad Gateway"

**可能原因**：
1. Agent 未连接
2. Agent 无法访问 Prometheus
3. URL 配置错误

**排查**：
```bash
# 检查 Agent 是否在线
curl http://localhost:9876/api/v1/agent/status/{host_id}

# 在 Agent 主机上测试 Prometheus 连接
curl http://prometheus:9090/api/v1/query?query=up
```

### 问题 2：Grafana 提示 "Timeout"

**可能原因**：
1. Agent 响应超时（>30秒）
2. Prometheus 响应慢
3. 网络问题

**排查**：
```bash
# 查看服务端日志
grep "等待Agent响应超时" logs/app.log

# 在 Agent 主机上测试响应时间
time curl http://prometheus:9090/api/v1/query?query=up
```

### 问题 3：Grafana 提示 "Unauthorized"

**可能原因**：
1. Prometheus 需要认证但数据源未配置
2. 认证信息错误

**排查**：
```bash
# 检查数据源认证配置
mysql -uroot -p'OpsHub@2026' opshub -e "
SELECT id, name, username, token 
FROM alert_datasources 
WHERE id = {your_datasource_id};
"

# 在 Agent 主机上测试认证
curl -H "Authorization: Bearer {token}" http://prometheus:9090/api/v1/query?query=up
```

### 问题 4：Grafana 提示 "Not Found"

**可能原因**：
1. ProxyToken 错误
2. 数据源不存在
3. 路径错误

**排查**：
```bash
# 检查 ProxyToken
mysql -uroot -p'OpsHub@2026' opshub -e "
SELECT id, name, proxy_token, proxy_url 
FROM alert_datasources 
WHERE proxy_token = '{your_token}';
"

# 查看服务端日志
grep "数据源不存在" logs/app.log
```

### 问题 5：返回空数据

**可能原因**：
1. Prometheus 中没有数据
2. 查询表达式错误
3. 时间范围问题

**排查**：
```bash
# 在 Agent 主机上直接查询
curl "http://prometheus:9090/api/v1/query?query=up&time=$(date +%s)"

# 查看 Agent 日志中的响应体长度
grep "HTTP 代理请求成功" /var/log/srehub-agent/agent.log
```

---

## 📝 调试命令速查

### 服务端

```bash
# 查看所有代理请求
tail -f logs/app.log | grep "代理转发请求"

# 查看特定 token 的请求
tail -f logs/app.log | grep "proxy_token=abc-123"

# 查看失败的请求
tail -f logs/app.log | grep "转发请求失败\|Agent执行失败"

# 查看超时的请求
tail -f logs/app.log | grep "等待Agent响应超时"
```

### Agent 端

```bash
# 查看所有 HTTP 代理请求
tail -f /var/log/srehub-agent/agent.log | grep "HTTP 代理请求"

# 查看成功的请求
tail -f /var/log/srehub-agent/agent.log | grep "HTTP 代理请求成功"

# 查看失败的请求
tail -f /var/log/srehub-agent/agent.log | grep "HTTP 代理请求失败"

# 查看特定 URL 的请求
tail -f /var/log/srehub-agent/agent.log | grep "url=http://prometheus:9090"
```

### 数据库

```bash
# 查看所有 Agent 代理数据源
mysql -uroot -p'OpsHub@2026' opshub -e "
SELECT 
    ds.id,
    ds.name,
    ds.type,
    ds.url,
    ds.proxy_token,
    COUNT(dsar.id) as agent_count
FROM alert_datasources ds
LEFT JOIN alert_datasource_agent_relations dsar ON ds.id = dsar.datasource_id
WHERE ds.access_mode = 'agent'
GROUP BY ds.id;
"

# 查看数据源的 Agent 关联
mysql -uroot -p'OpsHub@2026' opshub -e "
SELECT 
    dsar.datasource_id,
    dsar.agent_host_id,
    dsar.priority,
    h.hostname,
    h.ip,
    h.agent_id
FROM alert_datasource_agent_relations dsar
JOIN asset_hosts h ON dsar.agent_host_id = h.id
WHERE dsar.datasource_id = {your_datasource_id}
ORDER BY dsar.priority;
"
```

---

## 🧪 测试步骤

### 1. 测试 Agent 连接

```bash
# 在服务端测试
curl http://localhost:9876/api/v1/agent/list
```

### 2. 测试代理转发（不通过 Grafana）

```bash
# 获取 ProxyToken
TOKEN=$(mysql -uroot -p'OpsHub@2026' opshub -sN -e "
SELECT proxy_token FROM alert_datasources WHERE access_mode='agent' LIMIT 1;
")

# 测试查询
curl "http://localhost:9876/api/v1/alert/proxy/datasource/${TOKEN}/api/v1/query?query=up"

# 测试标签查询（Grafana 首次连接会调用）
curl "http://localhost:9876/api/v1/alert/proxy/datasource/${TOKEN}/api/v1/label/__name__/values"
```

### 3. 在 Grafana 中测试

1. 添加数据源
2. URL 填写：`http://localhost:9876/api/v1/alert/proxy/datasource/{token}`
3. 点击 "Save & Test"
4. 查看服务端和 Agent 日志

---

## 📋 检查清单

在 Grafana 测试前，确保：

- [ ] 数据源已创建（access_mode = 'agent'）
- [ ] 数据源 URL 格式正确（http://prometheus:9090）
- [ ] 数据源已关联至少一个 Agent
- [ ] Agent 正在运行且已连接
- [ ] Agent 可以访问 Prometheus（在 Agent 主机上 curl 测试）
- [ ] 服务端日志正常（无错误）
- [ ] Agent 日志正常（无错误）
- [ ] ProxyToken 正确配置在 Grafana 中

---

## 🎯 下一步

1. 重启服务端（加载新代码）
2. 在 Grafana 中测试连接
3. 查看服务端日志（`tail -f logs/app.log | grep "代理转发"`）
4. 查看 Agent 日志（`tail -f /var/log/srehub-agent/agent.log | grep "HTTP 代理"`）
5. 根据日志定位问题

