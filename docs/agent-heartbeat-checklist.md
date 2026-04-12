# Agent 心跳优化 - 实施检查清单

## 代码实现检查清单 ✅

### 核心组件

- [x] **Lua 脚本管理器** (`internal/cache/lua_scripts.go`)
  - [x] UpdateAndDetectChange 脚本
  - [x] EnqueueWithDedup 脚本
  - [x] DequeueBatch 脚本
  - [x] ReleaseLock 脚本

- [x] **批量同步 Worker** (`internal/cache/batch_worker.go`)
  - [x] Start/Stop 方法
  - [x] 分布式锁获取/释放
  - [x] 批量出队逻辑
  - [x] 批量读取 Redis
  - [x] 批量更新 MySQL
  - [x] 失败重试机制

- [x] **监控指标** (`internal/cache/metrics.go`)
  - [x] BatchQueueSize
  - [x] BatchFlushDuration
  - [x] BatchFlushCount
  - [x] ImmediateWriteCount
  - [x] RedisFallbackCount
  - [x] LockContentionCount

- [x] **CacheManager 重构** (`internal/cache/cache_manager.go`)
  - [x] UpdateAgentStatus 方法重构
  - [x] updateRedisAndDetectChange 方法
  - [x] enqueueToRedis 方法
  - [x] StartBatchWorker/StopBatchWorker 方法

- [x] **配置转换器** (`internal/cache/config_converter.go`)
  - [x] ConvertConfigToCacheConfig 方法

- [x] **单元测试** (`internal/cache/cache_manager_test.go`)
  - [x] 状态变化测试
  - [x] 状态未变化测试
  - [x] 批量同步测试
  - [x] 入队去重测试
  - [x] Redis 故障降级测试

### 配置文件

- [x] **配置结构体** (`internal/conf/conf.go`)
  - [x] CacheConfig 结构体定义
  - [x] Config 中添加 Cache 字段

- [x] **配置文件** (`config/config.yaml`)
  - [x] cache 配置段
  - [x] batch_flush_interval
  - [x] batch_size
  - [x] batch_queue_max_size
  - [x] redis_ttl
  - [x] lock_timeout
  - [x] offline_threshold

### 服务集成

- [x] **AgentService** (`internal/server/agent/agent_service.go`)
  - [x] handleHeartbeat 方法修改
  - [x] 移除异步 goroutine
  - [x] 修改 last_seen 类型

- [x] **HTTPServer** (`internal/server/http.go`)
  - [x] CacheManager 初始化传入配置
  - [x] 启动 BatchWorker
  - [x] Stop 方法中停止 BatchWorker

### 文档

- [x] **详细设计文档** (`docs/agent-heartbeat-optimization.md`)
- [x] **快速启动指南** (`docs/agent-heartbeat-quickstart.md`)
- [x] **实施总结** (`docs/agent-heartbeat-summary.md`)
- [x] **检查清单** (`docs/agent-heartbeat-checklist.md`)

---

## 测试验证清单 ⏳

### 编译测试

- [ ] **编译项目**
  ```bash
  make clean
  make build
  ```
  - [ ] 无编译错误
  - [ ] 无类型错误
  - [ ] 无导入错误

### 启动测试

- [ ] **启动服务**
  ```bash
  ./bin/opshub server -c config/config.yaml
  ```
  - [ ] 服务正常启动
  - [ ] 日志显示 "BatchWorker 已启动"
  - [ ] 日志显示 "缓存调度器已启动"
  - [ ] HTTP 服务器启动成功

### 功能测试

#### 测试 1：状态变化（立即写入）

- [ ] **部署 Agent**
  - [ ] Agent 成功连接
  - [ ] 日志显示 "检测到状态变化，立即同步 MySQL"
  - [ ] MySQL 中 agent_info 表已更新
  - [ ] Redis 中 agent:status:{agentID} 已更新

- [ ] **停止 Agent**
  - [ ] 日志显示状态变为 offline
  - [ ] MySQL 立即更新为 offline
  - [ ] Redis 立即更新为 offline

#### 测试 2：普通心跳（批量队列）

- [ ] **Agent 保持在线**
  - [ ] 心跳正常发送（每 60 秒）
  - [ ] Redis 队列长度增长
  - [ ] 去重集合正常工作

- [ ] **等待批量同步**
  - [ ] 5 分钟后触发批量同步
  - [ ] 日志显示 "开始批量同步"
  - [ ] 日志显示 "批量同步完成"
  - [ ] Redis 队列清空
  - [ ] MySQL last_seen 已更新

#### 测试 3：Redis 故障降级

- [ ] **停止 Redis**
  ```bash
  docker stop opshub-redis
  ```
  - [ ] 日志显示 "Redis 故障，降级到直接写 MySQL"
  - [ ] 心跳仍然正常处理
  - [ ] MySQL 正常更新
  - [ ] 指标 redis_fallback_total 增加

- [ ] **恢复 Redis**
  ```bash
  docker start opshub-redis
  ```
  - [ ] 服务恢复正常
  - [ ] 不再降级

#### 测试 4：多副本部署

- [ ] **启动 3 个副本**
  - [ ] 所有副本正常启动
  - [ ] 只有一个副本获得锁
  - [ ] 其他副本日志显示 "其他副本正在处理"
  - [ ] 指标 lock_contention_total 正常

- [ ] **停止持锁副本**
  - [ ] 锁自动过期（60 秒）
  - [ ] 其他副本接管任务

### 性能测试

#### 测试 5：MySQL QPS

- [ ] **观察 MySQL QPS**
  ```sql
  SHOW GLOBAL STATUS LIKE 'Questions';
  ```
  - [ ] 优化前：100+ QPS
  - [ ] 优化后：3-5 QPS
  - [ ] 降低 95%+

#### 测试 6：心跳响应延迟

- [ ] **观察日志中的延迟**
  - [ ] 优化前：10-50ms
  - [ ] 优化后：1-2ms
  - [ ] 降低 80%+

#### 测试 7：批量同步耗时

- [ ] **观察批量同步日志**
  - [ ] 100 条数据：10-50ms
  - [ ] 500 条数据：50-200ms
  - [ ] 性能可接受

### 监控测试

#### 测试 8：Prometheus 指标

- [ ] **访问指标端点**
  ```bash
  curl http://localhost:9876/metrics | grep agent_batch
  ```
  - [ ] agent_batch_queue_size 正常
  - [ ] agent_batch_flush_duration_seconds 正常
  - [ ] agent_batch_flush_total 正常
  - [ ] agent_immediate_write_total 正常
  - [ ] agent_redis_fallback_total 正常
  - [ ] agent_lock_contention_total 正常

### 压力测试

#### 测试 9：gRPC 压测

- [ ] **使用 ghz 压测**
  ```bash
  ghz --insecure \
      --proto api/proto/agent.proto \
      --call AgentHub.Connect \
      --duration 60s \
      --connections 1000 \
      --rps 100 \
      localhost:9090
  ```
  - [ ] 无错误
  - [ ] 延迟可接受
  - [ ] QPS 达标

---

## 问题排查清单

### 常见问题

#### 问题 1：编译错误

- [ ] **undefined: CacheConfig**
  - [ ] 检查 `internal/conf/conf.go` 是否定义了 CacheConfig
  - [ ] 检查导入路径是否正确

- [ ] **undefined: NewLuaScripts**
  - [ ] 检查 `internal/cache/lua_scripts.go` 是否存在
  - [ ] 检查文件是否在正确的包中

#### 问题 2：启动失败

- [ ] **BatchWorker 未启动**
  - [ ] 检查 `internal/server/http.go` 是否调用 StartBatchWorker
  - [ ] 检查配置是否正确加载

- [ ] **Redis 连接失败**
  - [ ] 检查 Redis 是否运行
  - [ ] 检查 Redis 配置是否正确

#### 问题 3：功能异常

- [ ] **队列持续增长**
  - [ ] 检查 BatchWorker 是否正常运行
  - [ ] 检查 MySQL 是否正常
  - [ ] 检查批量同步日志

- [ ] **数据不一致**
  - [ ] 检查批量同步是否成功
  - [ ] 检查 Redis TTL 是否合理
  - [ ] 手动触发数据同步

---

## 回滚清单

### 回滚触发条件

- [ ] 编译失败
- [ ] 启动失败
- [ ] 功能异常
- [ ] 性能下降
- [ ] 数据不一致

### 回滚步骤

#### 方法 1：配置回滚

- [ ] **修改配置文件**
  ```yaml
  cache:
    batch_flush_interval: 0  # 禁用批量模式
  ```

- [ ] **重启服务**
  ```bash
  pkill opshub
  ./bin/opshub server -c config/config.yaml
  ```

- [ ] **验证回滚**
  - [ ] 服务正常启动
  - [ ] 心跳正常处理
  - [ ] MySQL QPS 恢复到 100

#### 方法 2：代码回滚

- [ ] **回滚代码**
  ```bash
  git log --oneline
  git checkout <previous-commit>
  ```

- [ ] **重新编译**
  ```bash
  make build
  ```

- [ ] **重启服务**
  ```bash
  ./bin/opshub server -c config/config.yaml
  ```

- [ ] **验证回滚**
  - [ ] 服务正常启动
  - [ ] 功能正常

---

## 上线清单

### 上线前检查

- [ ] **代码审查**
  - [ ] 代码逻辑正确
  - [ ] 无明显 bug
  - [ ] 代码风格统一

- [ ] **测试完成**
  - [ ] 单元测试通过
  - [ ] 集成测试通过
  - [ ] 性能测试通过

- [ ] **文档完善**
  - [ ] 设计文档完整
  - [ ] 操作手册完整
  - [ ] 回滚方案明确

- [ ] **监控配置**
  - [ ] Prometheus 指标正常
  - [ ] Grafana Dashboard 配置
  - [ ] 告警规则配置

### 上线步骤

- [ ] **灰度发布**
  - [ ] 10% 流量（观察 1 小时）
  - [ ] 30% 流量（观察 2 小时）
  - [ ] 100% 流量（全量发布）

- [ ] **监控观察**
  - [ ] 观察 MySQL QPS
  - [ ] 观察心跳延迟
  - [ ] 观察错误日志
  - [ ] 观察队列大小

- [ ] **验收标准**
  - [ ] MySQL QPS < 5
  - [ ] 心跳延迟 < 5ms
  - [ ] 无错误日志
  - [ ] 队列大小 < 1000

### 上线后观察

- [ ] **第 1 天**
  - [ ] 每小时检查一次
  - [ ] 观察关键指标
  - [ ] 处理告警

- [ ] **第 2-7 天**
  - [ ] 每天检查一次
  - [ ] 观察趋势变化
  - [ ] 优化参数

- [ ] **第 8-30 天**
  - [ ] 每周检查一次
  - [ ] 总结经验
  - [ ] 持续优化

---

## 总结

### 完成情况

✅ **代码实现**：100% 完成
✅ **配置更新**：100% 完成
✅ **文档编写**：100% 完成
⏳ **测试验证**：待进行
⏳ **性能测试**：待进行
⏳ **上线发布**：待进行

### 下一步

1. **立即执行**：编译测试
2. **今天完成**：功能测试
3. **明天完成**：性能测试
4. **本周完成**：灰度发布

---

**检查清单版本**：v1.0
**最后更新**：2026-04-06
**负责人**：开发团队
