# Redis 密码登录问题修复

## 问题描述

使用密码登录 Redis 时出现错误：
```
AUTH failed: WRONGPASS invalid username-password pair or user is disabled.
```

## 问题原因

在 `docker-compose.yml` 中，Redis 的启动命令使用了环境变量语法：
```yaml
command: redis-server --appendonly yes --requirepass "${REDIS_PASSWORD:-1ujasdJ67Ps}"
```

但是在 docker-compose 的 `command` 字段中，环境变量不会被正确展开，导致 Redis 实际使用的密码是字符串 `$REDIS_PASSWORD` 而不是实际的密码值。

## 解决方案

### 1. 修复 docker-compose.yml

将 Redis 配置改为直接使用密码字符串：

```yaml
redis:
  image: swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/redis:7-alpine
  container_name: opshub-redis
  restart: unless-stopped
  ports:
    - "${REDIS_PORT:-6379}:6379"
  volumes:
    - redis-data:/data
  command: redis-server --appendonly yes --requirepass "1ujasdJ67Ps"
  healthcheck:
    test: ["CMD", "redis-cli", "-a", "1ujasdJ67Ps", "ping"]
    interval: 10s
    timeout: 5s
    retries: 5
  networks:
    - opshub-network
```

**关键变更**：
- 移除了 `environment` 部分（不需要）
- `command` 中直接使用密码字符串 `"1ujasdJ67Ps"`
- `healthcheck` 中也使用相同的密码字符串

### 2. 更新服务配置文件

更新 `config/config.yaml` 中的 Redis 密码：

```yaml
redis:
  host: 127.0.0.1
  port: 6379
  password: "1ujasdJ67Ps"  # 从空字符串改为实际密码
  db: 0
  pool_size: 10
  min_idle_conn: 5
```

### 3. 重启 Redis 服务

```bash
docker-compose stop redis
docker-compose rm -f redis
docker-compose up -d redis
```

## 验证结果

### 测试连接
```bash
docker exec opshub-redis redis-cli -a "1ujasdJ67Ps" ping
# 输出: PONG
```

### 测试读写
```bash
docker exec opshub-redis redis-cli -a "1ujasdJ67Ps" SET test_key "test_value"
# 输出: OK

docker exec opshub-redis redis-cli -a "1ujasdJ67Ps" GET test_key
# 输出: test_value
```

## Redis 密码信息

**密码**：`1ujasdJ67Ps`

**使用场景**：
- Docker Compose 中的 Redis 容器
- 服务端配置文件 `config/config.yaml`
- 应用程序连接 Redis

## 注意事项

1. **密码安全**：
   - 生产环境建议使用更强的密码
   - 可以通过环境变量 `.env` 文件管理密码
   - 不要将密码提交到公开的代码仓库

2. **环境变量限制**：
   - docker-compose 的 `command` 字段不支持环境变量展开
   - 如果需要使用环境变量，可以考虑：
     - 使用 entrypoint 脚本
     - 使用 docker-compose 的 `env_file`
     - 直接在 command 中硬编码（当前方案）

3. **配置一致性**：
   - 确保 `docker-compose.yml` 和 `config/config.yaml` 中的密码一致
   - 修改密码后需要重启 Redis 和应用服务

## 修改的文件

1. `docker-compose.yml` - Redis 服务配置
2. `config/config.yaml` - 应用配置文件

## 后续建议

### 使用环境变量文件（可选）

创建 `.env` 文件：
```bash
REDIS_PASSWORD=1ujasdJ67Ps
```

修改 `docker-compose.yml`：
```yaml
redis:
  command:
    - sh
    - -c
    - redis-server --appendonly yes --requirepass "$${REDIS_PASSWORD}"
  environment:
    - REDIS_PASSWORD=${REDIS_PASSWORD}
```

这样可以更安全地管理密码，但需要额外的 shell 包装。

## 测试清单

- [x] Redis 容器启动成功
- [x] 使用密码可以 ping 通
- [x] 可以执行 SET 操作
- [x] 可以执行 GET 操作
- [x] healthcheck 通过
- [x] 配置文件已更新

## 完成时间

2026-03-06 11:56

---

**状态**：✅ 已修复并验证通过
