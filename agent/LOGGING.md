# Agent 日志系统

## 概述

Agent 现已集成完整的日志系统，支持日志轮转、多级别日志、自动清理等功能。

## 功能特性

### 1. 日志轮转
- 使用 `lumberjack` 库实现自动日志轮转
- 当日志文件达到最大大小时自动创建新文件
- 旧日志文件自动压缩（gzip）
- 自动清理超过保留数量的旧日志

### 2. 日志级别
支持 4 个日志级别（从低到高）：
- `debug` - 调试信息（心跳发送、详细流程）
- `info` - 一般信息（启动、连接、注册成功）
- `warn` - 警告信息（连接断开、重连）
- `error` - 错误信息（连接失败、命令执行失败）

### 3. 双输出
- 同时输出到日志文件和标准输出（stdout）
- 便于容器化部署和传统部署

## 配置说明

在 `agent.yaml` 中配置日志参数：

```yaml
log_file: "/var/log/srehub-agent/agent.log"  # 日志文件路径
log_max_size: 100                             # 单个日志文件最大大小（MB）
log_max_backups: 3                            # 保留的旧日志文件数量
log_level: "info"                             # 日志级别：debug/info/warn/error
```

### 配置项说明

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `log_file` | string | `/var/log/srehub-agent/agent.log` | 日志文件路径 |
| `log_max_size` | int | 100 | 单个日志文件最大大小（MB） |
| `log_max_backups` | int | 3 | 保留的旧日志文件数量 |
| `log_level` | string | info | 日志级别 |

## 日志内容

### Agent 端日志

#### 启动日志
```
[2026-03-04 10:00:00] [INFO] SREHub Agent 启动中...
[2026-03-04 10:00:00] [INFO] AgentID: xxx-xxx-xxx
[2026-03-04 10:00:00] [INFO] Server: 192.168.1.100:9090
[2026-03-04 10:00:00] [INFO] 日志文件: /var/log/srehub-agent/agent.log (最大: 100MB, 保留: 3个备份)
```

#### 连接日志
```
[2026-03-04 10:00:01] [INFO] 正在连接到服务器: 192.168.1.100:9090
[2026-03-04 10:00:01] [INFO] 发送注册请求 - AgentID: xxx, Hostname: server01, OS: linux, Arch: amd64
[2026-03-04 10:00:01] [INFO] 启动心跳循环，间隔: 30秒
```

#### 心跳日志（Debug 级别）
```
[2026-03-04 10:00:31] [DEBUG] 发送心跳 - AgentID: xxx-xxx-xxx
[2026-03-04 10:01:01] [DEBUG] 发送心跳 - AgentID: xxx-xxx-xxx
```

#### 错误日志
```
[2026-03-04 10:05:00] [ERROR] 发送心跳失败: connection refused
[2026-03-04 10:05:00] [WARN] 连接断开: EOF, 5秒后重连...
```

#### 关闭日志
```
[2026-03-04 10:10:00] [INFO] 收到退出信号，正在关闭Agent...
[2026-03-04 10:10:00] [DEBUG] 心跳循环退出
[2026-03-04 10:10:00] [INFO] Agent已关闭
```

### 服务端日志

服务端在接收到 Agent 心跳时会打印 Debug 日志：

```
[2026-03-04 10:00:31] [DEBUG] 收到 Agent 心跳 agentID=xxx-xxx-xxx time=2026-03-04T10:00:31+08:00
```

## 日志文件管理

### 文件命名规则
- 当前日志：`agent.log`
- 轮转日志：`agent-2026-03-04T10-00-00.000.log.gz`（压缩）

### 自动清理策略
1. 当 `agent.log` 达到 `log_max_size` 时，自动重命名并压缩
2. 保留最近 `log_max_backups` 个备份文件
3. 超过 30 天的日志自动删除

### 磁盘空间估算
- 默认配置（100MB × 4 = 400MB）：
  - 当前日志：最大 100MB
  - 备份日志：3 个 × 100MB = 300MB（压缩后约 30-50MB）
  - 总计：约 130-150MB

## 调试建议

### 1. 查看实时日志
```bash
tail -f /var/log/srehub-agent/agent.log
```

### 2. 查看最近错误
```bash
grep ERROR /var/log/srehub-agent/agent.log | tail -20
```

### 3. 启用 Debug 级别
修改 `agent.yaml`：
```yaml
log_level: "debug"
```
重启 Agent 后可以看到心跳等详细日志。

### 4. 查看历史日志
```bash
ls -lh /var/log/srehub-agent/
zcat /var/log/srehub-agent/agent-*.log.gz | grep "关键词"
```

## 常见问题

### Q: 日志文件过大怎么办？
A: 调整 `log_max_size` 为更小的值（如 50MB），或减少 `log_max_backups` 数量。

### Q: 如何完全禁用日志？
A: 不建议禁用日志。如需减少日志量，可设置 `log_level: "error"` 只记录错误。

### Q: 日志目录权限问题？
A: 确保 Agent 运行用户对日志目录有写权限：
```bash
sudo mkdir -p /var/log/srehub-agent
sudo chown srehub-agent:srehub-agent /var/log/srehub-agent
```

### Q: 容器化部署如何查看日志？
A: 日志同时输出到 stdout，可使用 `docker logs` 或 `kubectl logs` 查看。

## 技术实现

### 依赖库
- `gopkg.in/natefinch/lumberjack.v2` - 日志轮转

### 核心代码
- `agent/internal/logger/logger.go` - 日志实现
- `agent/internal/config/config.go` - 配置定义
- `agent/cmd/main.go` - 日志初始化
- `agent/internal/client/grpc_client.go` - 日志调用

### 日志格式
```
[时间戳] [级别] 消息内容
```

示例：
```
[2026-03-04 10:00:00] [INFO] SREHub Agent 启动中...
```
