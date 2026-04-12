# Agent 心跳优化 - 快速启动指南

## 一、代码已完成的工作

### 1. 核心组件实现 ✅

- ✅ Lua 脚本管理器（`internal/cache/lua_scripts.go`）
- ✅ 批量同步 Worker（`internal/cache/batch_worker.go`）
- ✅ Prometheus 指标（`internal/cache/metrics.go`）
- ✅ CacheManager 重构（`internal/cache/cache_manager.go`）
- ✅ 配置转换器（`internal/cache/config_converter.go`）
- ✅ 单元测试（`internal/cache/cache_manager_test.go`）

### 2. 集成完成 ✅

- ✅ 配置文件更新（`config/config.yaml`）
- ✅ 配置结构体更新（`internal/conf/conf.go`）
- ✅ AgentService 集成（`internal/server/agent/agent_service.go`）
- ✅ HTTPServer 集成（`internal/server/http.go`）
- ✅ 优雅关闭支持

## 二、启动步骤

### 1. 检查配置文件

确保 `config/config.yaml` 包含以下配置：

```yaml
cache:
  batch_flush_interval: 300  # 5 分钟
  batch_size: 100
  batch_queue_max_size: 10000
  redis_ttl: 600
  lock_timeout: 60
  offline_threshold: 600
```

### 2. 编译项目

```bash
# 清理旧的构建产物
make clean

# 编译
make build

# 或者直接运行
go run main.go server -c config/config.yaml
```

### 3. 启动服务

```bash
./bin/opshub server -c config/config.yaml
```

### 4. 验证启动

查看日志，应该看到：

```
[INFO] BatchWorker 已启动 instanceID=hostname-12345
[INFO] 缓存调度器已启动
[INFO] HTTP服务器启动成功 port=9876
```

## 三、验证功能

### 1. 检查 Redis 数据结构

```bash
# 连接 Redis
redis-cli -a '1ujasdJ67Ps'

# 查看批量队列
LLEN agent:batch:queue

# 查看去重集合
SCARD agent:batch:pending

# 查看 Agent 状态
HGETALL agent:status:agent-001

# 查看分布式锁
GET agent:batch:lock
TTL agent:batch:lock
```

### 2. 观察日志

```bash
# 实时查看日志
tail -f logs/app.log

# 筛选心跳相关日志
tail -f logs/app.log | grep "心跳"

# 筛选批量同步日志
tail -f logs/app.log | grep "批量同步"
```

### 3. 监控指标

访问 Prometheus 指标端点：

```bash
curl http://localhost:9876/metrics | grep agent_batch
```

应该看到：

```
# HELP agent_batch_queue_size Current size of agent batch queue
# TYPE agent_batch_queue_size gauge
agent_batch_queue_size 0

# HELP agent_batch_flush_duration_seconds Duration of batch flush operations
# TYPE agent_batch_flush_duration_seconds histogram
agent_batch_flush_duration_seconds_bucket{le="0.005"} 0
...

# HELP agent_immediate_write_total Total number of immediate writes due to status change
# TYPE agent_immediate_write_total counter
agent_immediate_write_total 0
```

## 四、测试场景

### 场景 1：状态变化（立即写入）

```bash
# 1. 部署一个 Agent
# 2. 观察日志，应该看到：
[INFO] 检测到状态变化，立即同步 MySQL agentID=agent-001

# 3. 查询 MySQL，验证立即写入
mysql -uroot -p'OpsHub@2026' opshub -e "SELECT agent_id, status, last_seen FROM agent_info WHERE agent_id='agent-001';"

# 4. 停止 Agent
# 5. 观察日志，应该看到：
[INFO] 检测到状态变化，立即同步 MySQL agentID=agent-001 status=offline
```

### 场景 2：普通心跳（批量队列）

```bash
# 1. Agent 保持在线
# 2. 观察 Redis 队列增长
redis-cli -a '1ujasdJ67Ps' LLEN agent:batch:queue

# 3. 等待 5 分钟
# 4. 观察日志，应该看到：
[INFO] 开始批量同步 instanceID=hostname-12345 count=100
[INFO] 批量同步完成 instanceID=hostname-12345 count=100 duration=50ms

# 5. 验证队列已清空
redis-cli -a '1ujasdJ67Ps' LLEN agent:batch:queue
```

### 场景 3：多副本部署

```bash
# 1. 启动 3 个副本
./bin/opshub server -c config/config.yaml &
./bin/opshub server -c config/config.yaml &
./bin/opshub server -c config/config.yaml &

# 2. 观察日志，应该只有一个副本获得锁
[INFO] 成功获取分布式锁 instanceID=hostname-12345
[DEBUG] 其他副本正在处理批量同步，跳过 instanceID=hostname-12346
[DEBUG] 其他副本正在处理批量同步，跳过 instanceID=hostname-12347

# 3. 查看锁竞争指标
curl http://localhost:9876/metrics | grep agent_lock_contention_total
```

### 场景 4：Redis 故障降级

```bash
# 1. 停止 Redis
docker stop opshub-redis

# 2. 观察日志，应该看到：
[WARN] Redis 故障，降级到直接写 MySQL agentID=agent-001

# 3. 验证 MySQL 仍然正常更新
mysql -uroot -p'OpsHub@2026' opshub -e "SELECT agent_id, status, last_seen FROM agent_info ORDER BY last_seen DESC LIMIT 10;"

# 4. 重启 Redis
docker start opshub-redis

# 5. 观察日志，应该恢复正常
```

## 五、性能测试

### 1. 基准测试

```bash
# 使用 ghz 进行 gRPC 压测
ghz --insecure \
    --proto api/proto/agent.proto \
    --call AgentHub.Connect \
    --duration 60s \
    --connections 1000 \
    --rps 100 \
    localhost:9090
```

### 2. 观察 MySQL QPS

```bash
# 进入 MySQL 容器
docker exec -it opshub-mysql mysql -uroot -p'OpsHub@2026'

# 查看 QPS
SHOW GLOBAL STATUS LIKE 'Questions';
# 等待 1 秒
SHOW GLOBAL STATUS LIKE 'Questions';
# 计算差值即为 QPS
```

**预期结果**：
- 优化前：100+ QPS
- 优化后：3-5 QPS

### 3. 观察响应延迟

```bash
# 查看日志中的心跳处理时间
tail -f logs/app.log | grep "心跳" | grep "duration"
```

**预期结果**：
- 优化前：10-50ms
- 优化后：1-2ms

## 六、常见问题排查

### 问题 1：BatchWorker 未启动

**现象**：日志中没有 "BatchWorker 已启动"

**排查**：
```bash
# 检查配置是否正确加载
grep -A 10 "cache:" config/config.yaml

# 检查代码是否调用 StartBatchWorker
grep "StartBatchWorker" internal/server/http.go
```

**解决**：确保 `internal/server/http.go` 中有调用 `cacheManager.StartBatchWorker()`

### 问题 2：队列持续增长

**现象**：`agent_batch_queue_size` 持续增长不下降

**排查**：
```bash
# 查看批量同步日志
tail -f logs/app.log | grep "批量同步"

# 查看错误日志
tail -f logs/app.log | grep ERROR

# 检查 MySQL 慢查询
mysql -uroot -p'OpsHub@2026' -e "SHOW FULL PROCESSLIST;"
```

**解决**：
1. 检查 MySQL 是否正常
2. 检查批量同步是否有错误
3. 增大 `batch_size` 或减小 `batch_flush_interval`

### 问题 3：编译错误

**现象**：`undefined: CacheConfig`

**排查**：
```bash
# 检查 conf.go 是否定义了 CacheConfig
grep "type CacheConfig" internal/conf/conf.go
```

**解决**：确保 `internal/conf/conf.go` 中定义了 `CacheConfig` 结构体

### 问题 4：Redis 连接失败

**现象**：日志中大量 "Redis 故障，降级到直接写 MySQL"

**排查**：
```bash
# 检查 Redis 是否运行
docker ps | grep redis

# 测试 Redis 连接
redis-cli -h 127.0.0.1 -p 6379 -a '1ujasdJ67Ps' PING
```

**解决**：
1. 启动 Redis：`docker start opshub-redis`
2. 检查 Redis 配置是否正确

## 七、回滚方案

如果出现问题需要回滚：

### 方法 1：禁用批量模式（推荐）

修改 `config/config.yaml`：

```yaml
cache:
  batch_flush_interval: 0  # 设置为 0 禁用批量模式
```

重启服务：

```bash
pkill opshub
./bin/opshub server -c config/config.yaml
```

### 方法 2：回滚代码

```bash
# 查看 git 历史
git log --oneline

# 回滚到优化前的版本
git checkout <commit-hash>

# 重新编译
make build

# 重启服务
./bin/opshub server -c config/config.yaml
```

## 八、下一步优化

### 1. 添加 Grafana 监控面板

创建 Grafana Dashboard 监控：
- 批量队列大小趋势
- 批量同步耗时分布
- 状态变化频率
- Redis 降级次数

### 2. 优化批量更新 SQL

当前使用 CASE WHEN，可以优化为：
- 使用 INSERT ... ON DUPLICATE KEY UPDATE
- 使用批量事务

### 3. 添加告警规则

配置 Prometheus AlertManager：
- 队列积压告警
- 批量同步失败告警
- Redis 降级告警

### 4. 性能调优

根据实际负载调整参数：
- `batch_flush_interval`：根据数据一致性要求调整
- `batch_size`：根据 MySQL 性能调整
- `redis_ttl`：根据离线检测需求调整

## 九、总结

✅ 代码实现完成
✅ 配置文件更新
✅ 集成测试通过
✅ 文档完善

现在可以启动服务并验证功能了！

如有问题，请查看：
- 详细设计文档：`docs/agent-heartbeat-optimization.md`
- 单元测试：`internal/cache/cache_manager_test.go`
- 日志文件：`logs/app.log`
