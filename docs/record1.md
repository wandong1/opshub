# OpsHub 开发记录

本文件记录 OpsHub 项目的开发历程，包括需求、实现和里程碑。

---

## ✅ 实现记录

### [2026-03-04] Agent 日志系统实现
- **开发者**: Claude
- **关联需求**: Agent 日志打印与自动清理功能
- **描述**: 为 Agent 实现完整的日志系统，支持日志轮转、多级别日志、自动清理等功能，便于问题排查和运维监控
- **修改文件**:
  - `agent/internal/logger/logger.go` - 新增日志模块，实现 Debug/Info/Warn/Error/Fatal 五级日志
  - `agent/internal/config/config.go` - 添加日志配置字段（log_file、log_max_size、log_max_backups、log_level）
  - `agent/cmd/main.go` - 集成日志初始化，记录启动和关闭事件
  - `agent/internal/client/grpc_client.go` - 添加连接、注册、心跳、错误等关键日志点
  - `internal/server/agent/agent_service.go` - 添加服务端心跳接收 Debug 日志
  - `agent/config/agent.yaml` - 更新配置模板，添加日志配置项
  - `agent/dist/srehub-agent-1.0.0-multi-arch/agent.yaml` - 更新部署包配置模板
  - `agent/LOGGING.md` - 新增日志系统完整文档
- **技术要点**:
  - 使用 `lumberjack` 库实现日志轮转（达到 100MB 自动创建新文件）
  - 支持四级日志（debug/info/warn/error），可通过配置文件调整
  - 双输出机制（文件 + stdout），适配容器和传统部署
  - 自动压缩旧日志（gzip）
  - 自动清理策略（保留 3 个备份，超过 30 天自动删除）
  - 线程安全的日志写入（使用 sync.Mutex）
  - 记录关键事件：启动、连接、注册、心跳、错误、关闭
- **测试验证**:
  - ✅ Agent 编译通过
  - ✅ 多架构构建成功（linux-amd64/arm64, darwin-amd64/arm64）
  - ✅ 配置文件模板更新
  - ✅ 部署包已复制到 `data/agent-binaries/`

### [2026-03-04] Agent 离线状态同步优化
- **开发者**: Claude
- **关联需求**: 修复前端 Agent 状态显示不一致问题
- **描述**: 解决 Agent 断开连接后，后端数据库状态已更新为 offline，但前端界面仍显示 online 的问题
- **修改文件**:
  - `internal/server/agent/agent_service.go` - 在 Connect() defer 中添加 Redis 缓存更新逻辑
  - `web/src/views/asset/Hosts.vue` - 添加 30 秒轮询机制自动刷新 Agent 状态
- **技术要点**:
  - 后端：Agent 断开时同步更新 MySQL 和 Redis 缓存
  - 前端：使用 setInterval 实现 30 秒自动轮询
  - 前端：在组件卸载时清理定时器，避免内存泄漏
  - 确保数据库、缓存、前端三层状态一致性

### [2026-03-04] Redis 缓存系统集成
- **开发者**: Claude
- **关联需求**: 提升 Agent 状态查询性能
- **描述**: 集成 Redis 缓存管理器，优化 Agent 心跳和状态查询性能
- **修改文件**:
  - `internal/server/agent/grpc_server.go` - 添加 GetAgentService() 方法
  - `internal/server/http.go` - 集成缓存管理器初始化和优雅关闭
- **技术要点**:
  - 实现三种缓存一致性策略（Write-Through、Cache-Aside、Delayed Double Delete）
  - Agent 心跳异步更新缓存，降低延迟
  - 服务关闭时优雅清理缓存连接

---

## 📋 需求记录

### [2026-03-04] Agent 日志打印与自动清理
- **提出人**: 用户
- **描述**: 主机上部署的 Agent 日志打印缺失，对问题排查不友好。需要将关键日志、错误信息、上报心跳的信息打印出来，并实现自动清理。自动清理的大小可配置，默认为 100MB。同时，服务端接收到 Agent 上报的心跳时，也应打印 debug 日志
- **状态**: ✅ 已完成

### [2026-03-04] Agent 离线状态显示不一致
- **提出人**: 用户
- **描述**: 前端连接方式列显示的 Agent 状态与后端接口返回的 agent_status 字段不一致。后端 agent_status 与 Agent 是否断开状态一致，但前端界面连接方式列的 Agent 状态颜色显示不对，好像没有更新
- **状态**: ✅ 已完成

---

## 🎯 里程碑

### [2026-03-04] v1.1.0 - Agent 可观测性增强
- **参与人员**: Claude
- **主要功能**:
  - Agent 完整日志系统（轮转、多级别、自动清理）
  - Agent 状态实时同步（后端缓存 + 前端轮询）
  - Redis 缓存系统集成
- **技术亮点**:
  - 日志轮转机制，支持自动压缩和清理
  - 多级别日志（debug/info/warn/error），可配置
  - 双输出（文件 + stdout），适配多种部署场景
  - 缓存与数据库双写，确保状态一致性
  - 前端自动轮询，实时反馈 Agent 状态变化
- **影响范围**:
  - Agent 端：日志系统、配置管理
  - 服务端：缓存集成、状态同步
  - 前端：状态轮询、用户体验优化

---

*最后更新: 2026-03-04*
