# Agent 状态与主机信息缓存方案

## 概述

为了提高大批量主机纳管时的性能，我们设计了一套基于 Redis 的缓存方案，用于缓存 Agent 状态和主机监控信息。

## 架构设计

### 缓存层级

```
┌─────────────┐
│   应用层    │
└──────┬──────┘
       │
┌──────▼──────────────────────┐
│   CacheManager (一致性保证)  │
└──────┬──────────────────────┘
       │
┌──────▼──────┐    ┌──────────┐
│ Redis (L1)  │    │  MySQL   │
│  热数据缓存  │    │ 持久化存储│
└─────────────┘    └──────────┘
```

### 数据结构

#### Agent 状态缓存
- **Key**: `agent:status:{agent_id}`
- **Type**: String (JSON)
- **TTL**: 3 分钟
- **内容**: 状态、最后心跳时间、版本、主机名、OS、架构

#### Agent 在线列表
- **Key**: `agent:online`
- **Type**: Set
- **TTL**: 无（成员自动维护）
- **内容**: 所有在线 Agent ID

#### 主机基本信息
- **Key**: `host:info:{host_id}`
- **Type**: String (JSON)
- **TTL**: 10 分钟
- **内容**: 主机 ID、名称、IP、端口、Agent 状态等

#### 主机监控指标
- **Key**: `host:metrics:{host_id}`
- **Type**: String (JSON)
- **TTL**: 1 分钟
- **内容**: CPU、内存、磁盘使用率、网络流量等

## 一致性保证策略

### 1. Write-Through（写穿透）

**适用场景**: Agent 状态更新（心跳）

**流程**:
```
更新请求 → 更新 MySQL → 更新 Redis → 返回成功
```

**优点**:
- 数据强一致性
- 缓存始终是最新的

**实现**:
```go
func (m *CacheManager) UpdateAgentStatus(ctx context.Context, agentID string, updates map[string]any) error {
    // 1. 更新 MySQL
    if err := m.agentRepo.UpdateInfo(ctx, agentID, updates); err != nil {
        return err
    }

    // 2. 查询最新数据
    agentInfo, _ := m.agentRepo.GetByAgentID(ctx, agentID)

    // 3. 更新 Redis
    cacheData := ConvertAgentInfoToCache(agentInfo)
    m.cache.SetAgentStatus(ctx, agentID, cacheData)

    return nil
}
```

### 2. Cache-Aside（旁路缓存）

**适用场景**: 主机信息查询

**流程**:
```
查询请求 → 查 Redis → 命中？
                      ├─ 是 → 返回缓存数据
                      └─ 否 → 查 MySQL → 回写 Redis → 返回数据
```

**优点**:
- 灵活性高
- 缓存失败不影响业务

**实现**:
```go
func (m *CacheManager) GetHostInfoWithFallback(ctx context.Context, hostID uint) (*HostInfoCache, error) {
    // 1. 查 Redis
    cached, _ := m.cache.GetHostInfo(ctx, hostID)
    if cached != nil {
        return cached, nil
    }

    // 2. 查 MySQL
    var host asset.Host
    if err := m.db.Where("id = ?", hostID).First(&host).Error; err != nil {
        return nil, err
    }

    // 3. 回写 Redis
    cacheData := ConvertHostToCache(&host)
    m.cache.SetHostInfo(ctx, hostID, cacheData)

    return cacheData, nil
}
```

### 3. 延迟双删

**适用场景**: 主机信息更新

**流程**:
```
更新请求 → 删除 Redis → 更新 MySQL → 延迟 500ms → 再次删除 Redis
```

**优点**:
- 防止脏读
- 解决并发更新问题

**实现**:
```go
func (m *CacheManager) UpdateHostInfo(ctx context.Context, hostID uint, updates map[string]any) error {
    // 1. 删除缓存
    m.cache.InvalidateHostCache(ctx, hostID)

    // 2. 更新 MySQL
    m.db.Model(&asset.Host{}).Where("id = ?", hostID).Updates(updates)

    // 3. 延迟再次删除
    go func() {
        time.Sleep(500 * time.Millisecond)
        m.cache.InvalidateHostCache(context.Background(), hostID)
    }()

    return nil
}
```

## 使用方法

### 1. 初始化缓存管理器

```go
// 在 internal/server/http.go 中初始化
func NewHTTPServer(cfg *conf.Config, db *gorm.DB) *HTTPServer {
    // ... 其他初始化代码

    // 创建 Redis 客户端
    redisClient, err := data.NewRedis(cfg)
    if err != nil {
        appLogger.Fatal("Redis 初始化失败", zap.Error(err))
    }

    // 创建缓存管理器
    agentRepo := agentrepo.NewRepository(db)
    cacheManager := cache.NewCacheManager(redisClient.Get(), agentRepo, db)

    // 创建调度器并启动
    scheduler := cache.NewCacheSyncScheduler(cacheManager)
    scheduler.WarmupCache() // 预热缓存
    scheduler.Start()       // 启动定期任务

    // 注入到 AgentService
    agentService.SetCacheManager(cacheManager)

    return &HTTPServer{
        cacheManager: cacheManager,
        scheduler:    scheduler,
        // ...
    }
}
```

### 2. Agent 心跳处理（集成缓存）

```go
// 在 internal/server/agent/agent_service.go 中
type AgentService struct {
    // ... 其他字段
    cacheManager *cache.CacheManager
}

func (s *AgentService) SetCacheManager(manager *cache.CacheManager) {
    s.cacheManager = manager
}

func (s *AgentService) handleHeartbeat(as *AgentStream, req *pb.HeartbeatRequest) {
    now := time.Now()

    // 使用缓存管理器更新（异步，不阻塞心跳响应）
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
        defer cancel()

        updates := map[string]any{
            "status":    "online",
            "last_seen": &now,
        }

        s.cacheManager.UpdateAgentStatus(ctx, req.AgentId, updates)
    }()

    // 立即响应心跳
    as.Send(&pb.ServerMessage{
        Payload: &pb.ServerMessage_HeartbeatAck{
            HeartbeatAck: &pb.HeartbeatResponse{Success: true},
        },
    })
}
```

### 3. 主机列表查询（使用缓存）

```go
// 在 internal/biz/asset/host_usecase.go 中
func (uc *HostUseCase) ListHosts(ctx context.Context, req *ListHostsRequest) ([]*HostVO, int64, error) {
    // 1. 查询主机 ID 列表（从 MySQL）
    var hostIDs []uint
    query := uc.hostRepo.GetDB().Model(&Host{})
    // ... 应用过滤条件
    query.Pluck("id", &hostIDs)

    // 2. 批量从缓存获取主机信息
    if uc.cacheManager != nil {
        cachedHosts, err := uc.cacheManager.BatchGetHostInfoWithFallback(ctx, hostIDs)
        if err == nil {
            // 转换为 VO
            vos := make([]*HostVO, 0, len(cachedHosts))
            for _, cached := range cachedHosts {
                vos = append(vos, convertCacheToVO(cached))
            }
            return vos, int64(len(vos)), nil
        }
    }

    // 3. 降级：直接查询 MySQL
    var hosts []Host
    query.Find(&hosts)
    // ... 转换为 VO
}
```

### 4. Agent 状态批量查询

```go
// 在 internal/server/agent/http.go 中
func (s *HTTPServer) GetAgentStatuses(c *gin.Context) {
    var req struct {
        HostIDs []uint `json:"hostIds"`
    }
    c.ShouldBindJSON(&req)

    // 1. 查询主机的 agent_id
    var hosts []struct {
        ID      uint
        AgentID string
    }
    s.db.Model(&asset.Host{}).
        Where("id IN ?", req.HostIDs).
        Select("id, agent_id").
        Find(&hosts)

    // 2. 提取 agent_id 列表
    agentIDs := make([]string, 0)
    hostAgentMap := make(map[string]uint)
    for _, h := range hosts {
        if h.AgentID != "" {
            agentIDs = append(agentIDs, h.AgentID)
            hostAgentMap[h.AgentID] = h.ID
        }
    }

    // 3. 批量从缓存获取 Agent 状态
    statuses, err := s.cacheManager.BatchGetAgentStatusWithFallback(c.Request.Context(), agentIDs)
    if err != nil {
        response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
        return
    }

    // 4. 组装响应
    result := make(map[uint]string)
    for agentID, status := range statuses {
        hostID := hostAgentMap[agentID]
        result[hostID] = status.Status
    }

    response.Success(c, result)
}
```

## 性能优化

### 1. 批量操作

使用 Redis Pipeline 批量操作，减少网络往返：

```go
func (c *AgentCache) BatchGetAgentStatus(ctx context.Context, agentIDs []string) (map[string]*AgentStatusCache, error) {
    pipe := c.rdb.Pipeline()
    cmds := make(map[string]*redis.StringCmd)

    for _, agentID := range agentIDs {
        key := AgentStatusPrefix + agentID
        cmds[agentID] = pipe.Get(ctx, key)
    }

    pipe.Exec(ctx) // 一次性执行所有命令

    // 处理结果...
}
```

### 2. 异步回写

缓存未命中时，异步回写 Redis，不阻塞主流程：

```go
go func(agentID string, data *AgentStatusCache) {
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()
    c.cache.SetAgentStatus(ctx, agentID, data)
}(info.AgentID, cacheData)
```

### 3. 降级策略

Redis 故障时自动降级到 MySQL：

```go
cached, err := m.cache.GetAgentStatus(ctx, agentID)
if err != nil {
    appLogger.Warn("Redis 查询失败，降级到 MySQL")
    // 直接查询 MySQL
}
```

## 监控指标

### 1. 缓存命中率

```go
// 在 CacheManager 中添加统计
type CacheStats struct {
    Hits   int64
    Misses int64
}

func (m *CacheManager) GetHitRate() float64 {
    total := m.stats.Hits + m.stats.Misses
    if total == 0 {
        return 0
    }
    return float64(m.stats.Hits) / float64(total)
}
```

### 2. 缓存健康度

- 在线 Agent 数量
- 缓存 key 数量
- 内存使用情况

### 3. 同步延迟

- MySQL 到 Redis 的同步延迟
- 缓存更新失败率

## 注意事项

### 1. TTL 设置

- Agent 状态：3 分钟（心跳间隔 60 秒 × 3）
- 主机信息：10 分钟（变更频率低）
- 监控指标：1 分钟（实时性要求高）

### 2. 缓存穿透防护

对于不存在的数据，缓存空值（TTL 较短）：

```go
if agentInfo == nil {
    // 缓存空值，防止穿透
    m.cache.SetAgentStatus(ctx, agentID, &AgentStatusCache{Status: "not_found"})
}
```

### 3. 缓存雪崩防护

- 使用随机 TTL，避免同时过期
- 使用互斥锁，防止并发查询 MySQL

### 4. 数据一致性

- 写操作优先更新 MySQL，再更新 Redis
- 使用延迟双删防止脏读
- 定期同步修复不一致数据

## 测试建议

### 1. 单元测试

测试缓存的基本操作：

```bash
go test -v ./internal/cache/...
```

### 2. 压力测试

模拟大批量主机查询：

```bash
# 1000 个主机，并发 100
ab -n 10000 -c 100 http://localhost:9876/api/v1/asset/hosts
```

### 3. 故障测试

- Redis 宕机时的降级
- MySQL 慢查询时的超时处理
- 网络抖动时的重试机制
