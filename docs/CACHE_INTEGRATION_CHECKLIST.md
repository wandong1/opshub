# 缓存系统集成检查清单

## 📋 集成前检查

- [ ] Redis 服务已安装并运行
  ```bash
  redis-cli ping
  # 应返回: PONG
  ```

- [ ] Redis 配置正确（`config/config.yaml`）
  ```yaml
  redis:
    addr: "localhost:6379"
    password: ""
    db: 0
  ```

- [ ] Go 依赖已更新
  ```bash
  go mod tidy
  ```

- [ ] 单元测试通过
  ```bash
  go test -v ./internal/cache/
  # 应显示: PASS (10/10 tests)
  ```

- [ ] 已阅读快速集成指南
  ```bash
  cat docs/cache-quick-integration.md
  ```

---

## 🔧 集成步骤检查清单

### 步骤 1: 修改 `internal/server/http.go` - NewHTTPServer()

- [ ] 在 gRPC 服务器初始化后添加缓存初始化代码
- [ ] 初始化 Redis 客户端
  ```go
  redisClient, err := data.NewRedis(cfg)
  ```
- [ ] 创建 CacheManager
  ```go
  agentRepo := agentrepo.NewRepository(db)
  cacheManager := cache.NewCacheManager(redisClient.Get(), agentRepo, db)
  ```
- [ ] 创建 CacheSyncScheduler
  ```go
  scheduler := cache.NewCacheSyncScheduler(cacheManager)
  ```
- [ ] 执行缓存预热
  ```go
  if err := scheduler.WarmupCache(); err != nil {
      appLogger.Warn("缓存预热失败", zap.Error(err))
  }
  ```
- [ ] 启动调度器
  ```go
  scheduler.Start()
  ```
- [ ] 注入到 AgentService
  ```go
  grpcServer.GetAgentService().SetCacheManager(cacheManager)
  ```

### 步骤 2: 修改 `internal/server/http.go` - HTTPServer 结构体

- [ ] 添加 cacheManager 字段
  ```go
  cacheManager *cache.CacheManager
  ```
- [ ] 添加 scheduler 字段
  ```go
  scheduler *cache.CacheSyncScheduler
  ```
- [ ] 在 return 语句中包含这两个字段
  ```go
  return &HTTPServer{
      // ... 现有字段
      cacheManager: cacheManager,
      scheduler:    scheduler,
  }
  ```

### 步骤 3: 修改 `internal/server/http.go` - Shutdown()

- [ ] 在 Shutdown 方法开始处添加调度器停止逻辑
  ```go
  if s.scheduler != nil {
      s.scheduler.Stop()
      appLogger.Info("缓存调度器已停止")
  }
  ```

### 步骤 4: 修改 `internal/server/agent/grpc_server.go`

- [ ] 添加 GetAgentService() 方法
  ```go
  func (s *GRPCServer) GetAgentService() *AgentService {
      return s.service
  }
  ```

### 步骤 5: 添加导入语句

- [ ] 在 `internal/server/http.go` 添加导入
  ```go
  "github.com/ydcloud-dy/opshub/internal/cache"
  agentrepo "github.com/ydcloud-dy/opshub/internal/data/agent"
  ```

---

## ✅ 集成后验证

### 编译检查

- [ ] 代码编译通过
  ```bash
  go build -o /dev/null ./...
  ```

### 启动验证

- [ ] 服务启动成功
  ```bash
  make run
  ```

- [ ] 日志显示缓存预热成功
  ```
  [INFO] 缓存预热开始...
  [INFO] 从 MySQL 加载了 X 个 Agent 状态到 Redis
  [INFO] 缓存预热完成
  [INFO] 缓存调度器已启动
  ```

- [ ] 没有错误日志

### Redis 验证

- [ ] Redis 中存在缓存数据
  ```bash
  redis-cli
  > KEYS agent:*
  > KEYS host:*
  > SMEMBERS agent:online
  > GET agent:status:agent-001
  ```

- [ ] 缓存 key 数量合理（与 Agent 数量匹配）

### 功能验证

- [ ] Agent 心跳正常工作
  ```bash
  # 观察日志，确认心跳更新频率正常（60秒一次）
  tail -f logs/opshub.log | grep "心跳"
  ```

- [ ] 主机列表查询正常
  ```bash
  # 通过前端或 API 测试主机列表查询
  curl -H "Authorization: Bearer <token>" \
       http://localhost:9876/api/v1/asset/hosts
  ```

- [ ] Agent 状态显示正常

### 性能验证

- [ ] 主机列表查询响应时间明显降低
- [ ] Agent 心跳响应时间 <5ms
- [ ] 数据库查询日志减少

---

## 🔍 故障排查检查清单

### 如果 Redis 连接失败

- [ ] 检查 Redis 服务状态
  ```bash
  redis-cli ping
  systemctl status redis  # Linux
  brew services list | grep redis  # macOS
  ```

- [ ] 检查配置文件中的 Redis 地址
  ```bash
  cat config/config.yaml | grep -A 3 redis
  ```

- [ ] 检查防火墙设置

### 如果缓存预热失败

- [ ] 检查 MySQL 连接是否正常
- [ ] 检查 agent_info 表是否存在
- [ ] 查看详细错误日志
  ```bash
  tail -100 logs/opshub.log | grep -i error
  ```

### 如果 Agent 状态未缓存

- [ ] 确认 Agent 已连接并发送心跳
- [ ] 检查 SetCacheManager() 是否被调用
- [ ] 查看 Agent 心跳处理日志
- [ ] 使用 Redis CLI 检查缓存
  ```bash
  redis-cli
  > KEYS agent:status:*
  ```

### 如果性能没有提升

- [ ] 确认缓存命中率（应 >90%）
- [ ] 检查 Redis 内存使用
- [ ] 查看缓存相关日志
- [ ] 使用 Redis MONITOR 命令观察缓存访问

---

## 🔄 回滚检查清单

### 如果需要回滚

- [ ] 注释掉步骤 1 中的缓存初始化代码
- [ ] 移除步骤 2 中添加的字段
- [ ] 注释掉步骤 3 中的调度器停止代码
- [ ] 移除步骤 4 中添加的方法
- [ ] 重新编译
  ```bash
  go build -o bin/opshub ./cmd/main.go
  ```
- [ ] 重启服务
  ```bash
  make run
  ```
- [ ] 验证服务正常运行

---

## 📊 性能基准记录

### 集成前基准（记录当前性能）

- [ ] 主机列表查询响应时间: _______ ms
- [ ] Agent 心跳响应时间: _______ ms
- [ ] 批量 Agent 状态查询（100个）: _______ ms
- [ ] 数据库 QPS: _______
- [ ] 数据库 CPU 使用率: _______ %

### 集成后基准（记录优化后性能）

- [ ] 主机列表查询响应时间: _______ ms
- [ ] Agent 心跳响应时间: _______ ms
- [ ] 批量 Agent 状态查询（100个）: _______ ms
- [ ] 数据库 QPS: _______
- [ ] 数据库 CPU 使用率: _______ %
- [ ] 缓存命中率: _______ %
- [ ] Redis 内存使用: _______ MB

### 性能提升计算

- [ ] 主机查询提升倍数: _______ x
- [ ] 心跳响应提升倍数: _______ x
- [ ] 批量查询提升倍数: _______ x

---

## 📝 集成完成确认

### 最终检查

- [ ] 所有集成步骤已完成
- [ ] 所有验证项通过
- [ ] 性能提升符合预期
- [ ] 没有错误日志
- [ ] 服务稳定运行 >1 小时
- [ ] 已记录性能基准数据

### 文档更新

- [ ] 更新 CLAUDE.md（如需要）
- [ ] 记录集成日期和性能数据
- [ ] 更新运维文档（如需要）

### 团队通知

- [ ] 通知团队成员缓存系统已上线
- [ ] 分享性能提升数据
- [ ] 说明监控指标位置

---

## 🎉 集成完成！

恭喜！缓存系统已成功集成到生产环境。

### 后续监控

定期检查以下指标：
- 缓存命中率（应保持 >90%）
- Redis 内存使用（避免 OOM）
- 缓存响应时间
- 数据库负载降低情况

### 持续优化

参考 `docs/cache-summary.md` 中的"后续优化方向"章节：
- 本地缓存（L0）
- 缓存预加载
- 智能淘汰策略
- 分布式锁
- 缓存分片

---

**集成日期**: _______________
**集成人员**: _______________
**验证人员**: _______________
**备注**: _______________
