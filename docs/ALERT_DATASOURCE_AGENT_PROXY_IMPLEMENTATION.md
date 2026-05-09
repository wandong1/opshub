# 告警管理数据源Agent代理转发功能实现总结

## 📋 需求概述

在SREHub-Agent上新增代理转发功能，实现以下业务能力：
1. 告警规则可关联边缘机房的Prometheus/VictoriaMetrics/InfluxDB等数据源
2. 平台内Grafana可使用代理转发URL访问边缘数据源
3. 支持多Agent关联和故障转移

## 🏗️ 架构设计

### 数据模型

**AlertDataSource（告警数据源）**
```go
type AlertDataSource struct {
    ID            uint      // 主键
    Name          string    // 数据源名称
    Type          string    // prometheus | victoriametrics | influxdb
    URL           string    // 直连模式：完整URL
    AccessMode    string    // direct | agent
    Host          string    // Agent代理模式：数据源地址
    Port          int       // Agent代理模式：数据源端口
    Username      string    // 认证用户名
    Password      string    // 认证密码
    Token         string    // 认证Token
    ProxyToken    string    // UUID，唯一标识（Agent模式自动生成）
    ProxyURL      string    // 代理转发URL（Agent模式自动生成）
    Status        int       // 1=启用 0=禁用
}
```

**DataSourceAgentRelation（数据源-Agent关联）**
```go
type DataSourceAgentRelation struct {
    ID            uint      // 主键
    DataSourceID  uint      // 数据源ID
    AgentHostID   uint      // Agent主机ID
    Priority      int       // 优先级 0-10，越小优先级越高
    CreatedAt     time.Time
}
```

### API端点

#### 数据源管理
- `POST /api/v1/alert/datasources` - 创建数据源
  - 请求体：AlertDataSource
  - Agent模式时自动生成ProxyToken和ProxyURL

- `GET /api/v1/alert/datasources` - 获取数据源列表
- `GET /api/v1/alert/datasources/:id` - 获取数据源详情
- `PUT /api/v1/alert/datasources/:id` - 更新数据源
- `DELETE /api/v1/alert/datasources/:id` - 删除数据源
- `POST /api/v1/alert/datasources/:id/test` - 测试连接

#### Agent关联管理
- `GET /api/v1/alert/datasources/:id/agent-relations` - 查询关联的Agent列表
- `POST /api/v1/alert/datasources/:id/agent-relations` - 创建Agent关联
- `DELETE /api/v1/alert/datasources/agent-relations/:id` - 删除Agent关联

#### 代理转发
- `ANY /api/v1/alert/proxy/datasource/:token/*path` - 转发请求
  - 查询ProxyToken对应的数据源
  - 选择在线的Agent（按优先级）
  - 转发请求到Agent访问的数据源
  - 返回结果给客户端

## 🔄 业务流程

### 1. 创建Agent代理数据源
```
前端选择"新增数据源" 
  ↓
选择"Agent代理"接入方式
  ↓
配置数据源信息（Host、Port、认证）
  ↓
保存数据源 → 后端自动生成ProxyToken和ProxyURL
  ↓
在"关联主机"部分添加Agent主机
  ↓
选择在线的Agent、设置优先级
  ↓
完成配置，生成代理转发URL供Grafana使用
```

### 2. 告警规则执行时查询数据源
```
告警规则触发 → 执行PromQL表达式
  ↓
调用QueryDataSourceWithAgent()
  ↓
if accessMode == "direct":
    直接HTTP请求数据源URL
else (accessMode == "agent"):
    获取该数据源关联的所有Agent（按优先级排序）
    ↓
    遍历Agent列表
    ↓
    if Agent在线:
        通过Agent代理转发请求
        ↓
        return 查询结果
    else:
        继续下一个Agent
    ↓
    if 所有Agent都不在线:
        return 错误
```

### 3. Grafana访问边缘数据源
```
Grafana配置数据源
  ↓
使用代理转发URL: http://opshub:8080/api/v1/alert/proxy/datasource/{proxyToken}
  ↓
平台后端处理请求
  ↓
选择在线的Agent（按优先级）
  ↓
通过Agent转发请求到边缘数据源
  ↓
Grafana接收结果、渲染大屏
```

## 💾 数据库表

### alert_datasources（告警数据源表）
```sql
CREATE TABLE alert_datasources (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(30) NOT NULL,
    url VARCHAR(500),
    username VARCHAR(100),
    password VARCHAR(200),
    token VARCHAR(500),
    description VARCHAR(500),
    status INT DEFAULT 1,
    access_mode VARCHAR(20) DEFAULT 'direct',
    host VARCHAR(255),
    port INT DEFAULT 0,
    proxy_token VARCHAR(100) UNIQUE,
    proxy_url VARCHAR(500),
    proxy_enabled BOOLEAN DEFAULT false,
    INDEX idx_access_mode (access_mode),
    INDEX idx_proxy_token (proxy_token)
);
```

### alert_datasource_agent_relations（数据源-Agent关联表）
```sql
CREATE TABLE alert_datasource_agent_relations (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP,
    data_source_id BIGINT NOT NULL,
    agent_host_id BIGINT NOT NULL,
    priority INT DEFAULT 0,
    INDEX idx_datasource_id (data_source_id),
    INDEX idx_agent_host_id (agent_host_id),
    UNIQUE KEY uk_datasource_agent (data_source_id, agent_host_id)
);
```

## 📦 已实现的文件

### 后端
1. **internal/biz/alert/datasource.go** - 数据源模型（已有）
2. **internal/biz/alert/datasource_agent_relation.go** - Agent关联模型 ✅ 新增
3. **internal/data/alert/datasource_repo.go** - 数据源仓储 ✅ 新增GetByProxyToken()
4. **internal/data/alert/datasource_agent_relation_repo.go** - Agent关联仓储 ✅ 新增
5. **internal/service/alert/datasource_query.go** - 查询服务 ✅ 新增Agent代理支持
6. **internal/server/alert/datasource_handler.go** - HTTP处理器 ✅ 新增Agent关联API
7. **internal/server/alert/datasource_proxy_handler.go** - 代理转发处理 ✅ 新增
8. **internal/server/alert/http.go** - 路由注册 ✅ 更新
9. **internal/server/http.go** - 主服务器 ✅ 注入AgentHub
10. **cmd/server/server.go** - 启动脚本 ✅ 自动迁移新表

### 前端
1. **web/src/views/alert/DataSources.vue** - 数据源管理页面 ✅ 更新
2. **web/src/views/alert/AgentRelationModal.vue** - Agent关联模态框 ✅ 新增
3. **web/src/api/alert.ts** - API调用 ✅ 新增Agent相关API

## 🔑 核心功能点

### 1. ProxyToken生成
```go
if ds.AccessMode == "agent" {
    ds.ProxyToken = uuid.New().String()
    ds.ProxyURL = "/api/v1/alert/proxy/datasource/" + ds.ProxyToken
}
```

### 2. Agent故障转移
```go
rels, _ := agentRelationRepo.ListByDataSourceID(ctx, ds.ID)
sort.Slice(rels, func(i, j int) bool {
    return rels[i].Priority < rels[j].Priority
})

for _, rel := range rels {
    if agentHub.IsOnline(rel.AgentHostID) {
        // 使用这个Agent转发请求
        break
    }
}
```

### 3. 请求转发
```go
// 代理将请求转发到Agent可访问的数据源
targetURL := fmt.Sprintf("http://%s:%d%s", ds.Host, ds.Port, path)
req, _ := http.NewRequest("GET", targetURL, nil)
// 添加认证信息
if ds.Token != "" {
    req.Header.Set("Authorization", "Bearer "+ds.Token)
}
// 转发请求并返回结果
```

## 🧪 测试场景

### 场景1：直连模式
- 创建数据源，选择"直连"模式
- 配置完整URL（如 http://prometheus:9090）
- 测试连接应成功
- 告警规则可正常执行

### 场景2：Agent代理单个主机
- 创建数据源，选择"Agent代理"模式
- 配置Host和Port（如 prometheus:9090）
- 关联一个在线的Agent主机
- 系统生成ProxyURL
- 告警规则执行时通过Agent转发
- Grafana使用ProxyURL访问数据源

### 场景3：Agent代理多主机（高可用）
- 同场景2，关联多个Agent主机
- 设置不同的优先级
- 第一个Agent离线，系统自动切换到第二个
- 确保服务连续性

### 场景4：Agent离线恢复
- Agent离线时，自动转移到备用Agent
- Agent恢复在线后，仍优先使用（按优先级）

## 📊 高可用特性

1. **多Agent关联**：一个数据源可关联多个Agent
2. **优先级控制**：按优先级0-10选择Agent（越小优先级越高）
3. **自动转移**：Agent离线自动转移到下一个在线Agent
4. **独立隔离**：不同数据源的ProxyURL相互隔离
5. **认证透明**：代理透明传递数据源的认证信息

## 🔐 安全考虑

1. **ProxyToken隔离**：每个数据源独立ProxyToken，防止混淆
2. **认证信息加密**：数据库存储密码和Token时应加密
3. **访问控制**：代理endpoint应受权限控制（可选）
4. **请求验证**：验证ProxyToken有效性
5. **响应过滤**：过滤敏感头信息

## 📈 性能考虑

1. **Agent在线状态缓存**：通过AgentHub.IsOnline()缓存
2. **连接复用**：HTTP Client复用
3. **超时控制**：设置合理的超时时间（10-30s）
4. **并发支持**：支持多个并发查询

## 🚀 部署清单

- [x] 后端代码编译通过
- [x] 前端代码完成
- [x] 数据库表自动迁移
- [ ] 前端npm build验证
- [ ] 集成测试
- [ ] 性能测试
- [ ] 安全审计

## 📝 使用指南

### 为Grafana配置代理数据源

1. 在OpsHub中创建Agent代理数据源
2. 完成Agent主机关联
3. 复制生成的ProxyURL
4. 在Grafana中创建新数据源
5. 数据源类型选择Prometheus
6. URL填入ProxyURL
7. 保存并测试

### 监控Agent状态

- 查看Agent在线状态
- 在关联主机列表中看到优先级
- Agent离线时自动转移（无感知）
- 日志中记录转发情况

## 🔄 后续优化方向

1. 支持Agent健康检查
2. 支持负载均衡策略
3. 请求缓存优化
4. 性能指标收集
5. 审计日志完善
