# 代理路由认证问题修复

**修复日期**: 2026-04-15  
**问题**: Grafana 无法通过代理 URL 访问数据源，返回 401 未登录  
**状态**: ✅ 已修复

---

## 🔴 问题描述

### 现象

在 Grafana 中配置数据源代理 URL 后，测试连接失败：

```bash
curl "http://localhost:9876/api/v1/alert/proxy/datasource/{token}/api/v1/query?query=up"

# 返回
{"code":401,"message":"未登录","timestamp":1776229083}
```

### 根本原因

代理路由 `/api/v1/alert/proxy/datasource/:token/*path` 被注册在 `/api/v1` 路由组下，该路由组使用了认证中间件：

```go
// internal/server/http.go
v1 := router.Group("/api/v1")
v1.Use(authMiddleware.AuthRequired())      // ❌ 需要 JWT 认证
v1.Use(authMiddleware.RequirePermission()) // ❌ 需要权限验证
```

**问题**：
- 代理 URL 是给 Grafana 等外部系统使用的
- 这些系统无法提供 JWT Token
- 代理 URL 本身通过 ProxyToken 来验证安全性
- 不应该再要求 JWT 认证

---

## ✅ 修复方案

### 方案概述

将代理路由从需要认证的路由组中移出，注册为公开路由（无需认证）。

### 实施步骤

#### 1. 修改告警模块路由注册

**文件**: `internal/server/alert/http.go`

**删除**：从 `RegisterRoutes` 中删除代理路由
```go
// 删除这部分
// 数据源代理
proxy := alert.Group("/proxy/datasource")
{
    proxy.Any("/:token/*path", s.proxyDataSourceRequest)
}
```

**新增**：添加公开路由注册方法
```go
// RegisterPublicRoutes 注册公开路由（无需认证）
func (s *HTTPServer) RegisterPublicRoutes(router *gin.Engine) {
    // 数据源代理路由（无需认证，通过 ProxyToken 验证）
    proxy := router.Group("/api/v1/alert/proxy/datasource")
    {
        proxy.Any("/:token/*path", s.proxyDataSourceRequest)
    }
}
```

#### 2. 在主路由中注册公开路由

**文件**: `internal/server/http.go`

在告警路由注册后添加：
```go
// 注册 Alert（告警管理）路由
s.alertServer = alertserver.NewAlertServices(s.db, s.redisClient)
if s.grpcServer != nil {
    s.alertServer.SetAgentHub(s.grpcServer.Hub())
}
s.alertServer.RegisterRoutes(v1)

// ✅ 新增：注册告警模块的公开路由（数据源代理，无需认证）
s.alertServer.RegisterPublicRoutes(router)
```

---

## 📊 修复前后对比

### 修复前

```
请求流程：
Grafana → /api/v1/alert/proxy/datasource/{token}/api/v1/query
         ↓
    认证中间件检查 JWT Token
         ↓
    ❌ 没有 Token，返回 401
```

### 修复后

```
请求流程：
Grafana → /api/v1/alert/proxy/datasource/{token}/api/v1/query
         ↓
    直接到达 proxyDataSourceRequest 处理器
         ↓
    通过 ProxyToken 查询数据源
         ↓
    ✅ 验证通过，转发请求
```

---

## 🔒 安全性说明

### ProxyToken 验证机制

虽然代理路由不需要 JWT 认证，但它有自己的安全验证机制：

1. **ProxyToken 唯一性**
   - 每个数据源有唯一的 UUID 作为 ProxyToken
   - ProxyToken 存储在数据库中
   - 无法猜测或伪造

2. **数据源验证**
   ```go
   // 查询数据源
   ds, err := s.dsRepo.GetByProxyToken(c.Request.Context(), proxyToken)
   if err != nil || ds == nil {
       response.ErrorCode(c, http.StatusNotFound, "数据源不存在")
       return
   }
   ```

3. **访问模式验证**
   ```go
   if ds.AccessMode != "agent" {
       response.ErrorCode(c, http.StatusBadRequest, "数据源不是Agent代理模式")
       return
   }
   ```

4. **Agent 在线验证**
   ```go
   if !s.agentHub.IsOnline(rel.AgentHostID) {
       response.ErrorCode(c, http.StatusServiceUnavailable, "没有可用的Agent")
       return
   }
   ```

### 安全建议

1. **定期轮换 ProxyToken**（可选）
   - 在数据源管理界面添加"重新生成 Token"功能
   - 旧 Token 立即失效

2. **访问日志记录**
   - 记录所有代理请求
   - 包含 IP、时间、数据源、查询内容

3. **速率限制**（可选）
   - 对单个 ProxyToken 的请求频率进行限制
   - 防止滥用

---

## 🧪 测试验证

### 1. 直接测试代理 URL

```bash
# 获取 ProxyToken
TOKEN="0727f1ab-b73c-4a43-9323-b71c640cac4e"

# 测试简单查询
curl "http://localhost:9876/api/v1/alert/proxy/datasource/$TOKEN/api/v1/query?query=up"

# 预期：返回 Prometheus 数据（不再是 401）
```

### 2. 测试标签查询

```bash
# Grafana 首次连接会调用这个接口
curl "http://localhost:9876/api/v1/alert/proxy/datasource/$TOKEN/api/v1/label/__name__/values"

# 预期：返回所有指标名称列表
```

### 3. 测试范围查询

```bash
START=$(date -u -d '1 hour ago' +%s)
END=$(date -u +%s)
curl "http://localhost:9876/api/v1/alert/proxy/datasource/$TOKEN/api/v1/query_range?query=up&start=$START&end=$END&step=60"

# 预期：返回时间范围内的数据
```

### 4. Grafana 集成测试

1. 在 Grafana 中添加数据源
2. 类型：Prometheus
3. URL：`http://localhost:9876/api/v1/alert/proxy/datasource/{token}`
4. 点击 "Save & Test"
5. 预期：✅ "Data source is working"

---

## 📝 路由结构

### 需要认证的路由

```
/api/v1/alert/datasources          (数据源管理)
/api/v1/alert/datasources/:id      (数据源详情)
/api/v1/alert/rules                (告警规则)
/api/v1/alert/events               (告警事件)
/api/v1/alert/channels             (通知渠道)
...
```

### 公开路由（无需认证）

```
/api/v1/alert/proxy/datasource/:token/*path  (数据源代理)
```

---

## 🎯 影响范围

### 受益功能

1. **Grafana 集成**
   - 可以正常配置数据源
   - 可以查询和展示数据
   - 可以创建仪表板

2. **告警规则查询**
   - 通过代理 URL 查询数据源
   - 评估告警条件

3. **外部系统集成**
   - 任何需要访问 Agent 代理数据源的外部系统
   - 通过代理 URL 访问内网 Prometheus

### 不受影响的功能

1. **数据源管理**
   - 仍然需要登录才能管理数据源
   - 创建、编辑、删除数据源需要认证

2. **其他告警功能**
   - 告警规则管理
   - 告警事件查看
   - 通知渠道配置

---

## ✅ 验证清单

- [x] 代理路由从认证路由组中移除
- [x] 添加 RegisterPublicRoutes 方法
- [x] 在主路由中注册公开路由
- [x] 编译成功
- [x] ProxyToken 验证逻辑保留
- [x] 安全性不受影响

---

## 🚀 部署说明

1. **重启服务端**
   ```bash
   systemctl restart opshub
   # 或
   ./bin/opshub server -c config/config.yaml
   ```

2. **测试代理 URL**
   ```bash
   curl "http://localhost:9876/api/v1/alert/proxy/datasource/{token}/api/v1/query?query=up"
   ```

3. **在 Grafana 中配置**
   - 添加 Prometheus 数据源
   - URL 使用代理 URL
   - 测试连接

---

## 📚 相关文档

- Agent 代理转发机制：`AGENT_PROXY_MECHANISM.md`
- Agent 代理实现：`AGENT_PROXY_IMPLEMENTATION.md`
- Grafana 调试指南：`GRAFANA_PROXY_DEBUG_GUIDE.md`

---

## 🎉 总结

本次修复解决了代理 URL 被认证中间件拦截的问题，使得 Grafana 和其他外部系统可以正常通过代理 URL 访问 Agent 代理的数据源。

**关键改进**：
- ✅ 代理路由不再需要 JWT 认证
- ✅ 通过 ProxyToken 验证安全性
- ✅ Grafana 可以正常集成
- ✅ 外部系统可以访问内网数据源

