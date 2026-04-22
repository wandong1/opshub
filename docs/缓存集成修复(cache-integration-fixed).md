# 缓存系统集成说明（已修正）

## ⚠️ 重要更新

之前创建的独立集成文件（`heartbeat_cache.go` 和 `agent_service_cache.go`）已删除，因为它们会导致重复定义错误。

缓存功能已直接集成到 `internal/server/agent/agent_service.go` 中。

## ✅ 已完成的修改

### 1. `internal/server/agent/agent_service.go`

已添加以下内容：

#### 导入缓存包
```go
import (
    // ... 其他导入
    "github.com/ydcloud-dy/opshub/internal/cache"
)
```

#### AgentService 结构体添加字段
```go
type AgentService struct {
    // ... 现有字段
    cacheManager *cache.CacheManager // 缓存管理器
}
```

#### 修改 handleHeartbeat 方法
```go
func (s *AgentService) handleHeartbeat(as *AgentStream, req *pb.HeartbeatRequest) {
    now := time.Now()

    // 使用缓存管理器更新状态（异步）
    if s.cacheManager != nil {
        go func() {
            ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
            defer cancel()

            updates := map[string]any{
                "status":    "online",
                "last_seen": &now,
            }

            if err := s.cacheManager.UpdateAgentStatus(ctx, req.AgentId, updates); err != nil {
                appLogger.Warn("更新 Agent 状态失败",
                    zap.String("agentID", req.AgentId),
                    zap.Error(err))
            }
        }()
    } else {
        // 降级：直接更新数据库
        s.agentRepo.UpdateInfo(context.Background(), req.AgentId, map[string]any{
            "status":    "online",
            "last_seen": &now,
        })
    }

    // 立即响应心跳
    as.Send(&pb.ServerMessage{
        Payload: &pb.ServerMessage_HeartbeatAck{
            HeartbeatAck: &pb.HeartbeatResponse{Success: true},
        },
    })
}
```

#### 添加 SetCacheManager 方法
```go
func (s *AgentService) SetCacheManager(manager *cache.CacheManager) {
    s.cacheManager = manager
}
```

## ⏳ 待完成的集成步骤

现在只需要完成以下步骤即可启用缓存系统：

### 步骤 1: 修改 `internal/server/agent/grpc_server.go`

添加 `GetAgentService()` 方法：

```go
// GetAgentService 获取 AgentService 实例（用于注入依赖）
func (s *GRPCServer) GetAgentService() *AgentService {
    return s.service
}
```

### 步骤 2: 修改 `internal/server/http.go` - NewHTTPServer()

在 gRPC 服务器初始化后添加缓存初始化代码：

```go
func NewHTTPServer(cfg *conf.Config, db *gorm.DB) *HTTPServer {
    // ... 现有代码（路由、中间件等）

    // 初始化 gRPC 服务器
    grpcServer := agent.NewGRPCServer(cfg, db)

    // ========== 新增：初始化缓存系统 ==========
    // 1. 获取 Redis 客户端
    redisClient, err := data.NewRedis(cfg)
    if err != nil {
        appLogger.Fatal("Redis 初始化失败", zap.Error(err))
    }

    // 2. 创建缓存管理器
    agentRepo := agentrepo.NewRepository(db)
    cacheManager := cache.NewCacheManager(redisClient.Get(), agentRepo, db)

    // 3. 创建调度器
    scheduler := cache.NewCacheSyncScheduler(cacheManager)

    // 4. 预热缓存
    if err := scheduler.WarmupCache(); err != nil {
        appLogger.Warn("缓存预热失败", zap.Error(err))
    }

    // 5. 启动定期任务
    scheduler.Start()

    // 6. 注入到 AgentService
    grpcServer.GetAgentService().SetCacheManager(cacheManager)
    // ========== 缓存系统初始化完成 ==========

    return &HTTPServer{
        // ... 现有字段
        cacheManager: cacheManager,  // 新增
        scheduler:    scheduler,     // 新增
    }
}
```

### 步骤 3: 修改 `internal/server/http.go` - HTTPServer 结构体

添加缓存相关字段：

```go
type HTTPServer struct {
    engine       *gin.Engine
    cfg          *conf.Config
    db           *gorm.DB
    grpcServer   *agent.GRPCServer
    pluginMgr    *plugin.Manager
    cacheManager *cache.CacheManager      // 新增
    scheduler    *cache.CacheSyncScheduler // 新增
}
```

### 步骤 4: 修改 `internal/server/http.go` - Shutdown()

添加调度器停止逻辑：

```go
func (s *HTTPServer) Shutdown(ctx context.Context) error {
    appLogger.Info("正在关闭 HTTP 服务器...")

    // 停止缓存调度器
    if s.scheduler != nil {
        s.scheduler.Stop()
        appLogger.Info("缓存调度器已停止")
    }

    // ... 其他关闭逻辑

    return nil
}
```

### 步骤 5: 添加导入语句

在 `internal/server/http.go` 文件顶部添加：

```go
import (
    // ... 现有导入
    "github.com/ydcloud-dy/opshub/internal/cache"
    agentrepo "github.com/ydcloud-dy/opshub/internal/data/agent"
)
```

## ✅ 验证

完成上述步骤后：

1. 编译检查
```bash
go build -o /dev/null ./main.go
```

2. 启动服务
```bash
make run
```

3. 检查日志，应该看到：
```
[INFO] 缓存预热开始...
[INFO] 从 MySQL 加载了 X 个 Agent 状态到 Redis
[INFO] 缓存预热完成
[INFO] 缓存调度器已启动
```

4. 验证 Redis 缓存
```bash
redis-cli
> KEYS agent:status:*
> SMEMBERS agent:online
```

## 📝 总结

- ✅ `agent_service.go` 已修改完成（添加 cacheManager 字段和 SetCacheManager 方法）
- ✅ `handleHeartbeat` 方法已更新为使用缓存
- ✅ 编译测试通过
- ⏳ 还需完成 4 个步骤来启用缓存系统

完成剩余步骤后，缓存系统将完全集成并开始工作。
