# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

OpsHub 是一个插件化的云原生运维管理平台。项目采用单仓库结构，后端使用 Go，前端使用 Vue 3，通过插件架构组织功能模块（Kubernetes 管理、任务执行、监控告警、Nginx 日志分析、SSL 证书管理），各插件可独立启用/禁用。平台支持 Agent 和 SSH 双通道管理主机，Agent 在线时优先使用 gRPC 通道。

## 常用命令

### 后端

```bash
make build          # 编译 Go 二进制文件到 bin/opshub
make run            # 启动后端服务（使用 config/config.yaml）
make test           # 运行所有 Go 测试（go test -v ./...）
make swagger        # 生成 Swagger 文档（swag init）
make fmt            # 格式化 Go 代码
make lint           # 运行 golangci-lint 代码检查
make deps           # go mod tidy && go mod verify
make clean          # 清理构建产物和日志
go run main.go server -c config/config.yaml   # 直接运行
go test -v ./plugins/kubernetes/...            # 运行单个包的测试
go build -o /dev/null ./...                    # 快速验证编译
```

### 前端

```bash
cd web
npm install         # 安装依赖
npm run dev         # 启动 Vite 开发服务器（localhost:5173）
npm run build       # 类型检查（vue-tsc）后构建生产版本
npx vue-tsc --noEmit  # 仅类型检查，不输出文件
```

### Docker

```bash
docker-compose up -d    # 启动完整服务栈（MySQL、Redis、后端、前端/Nginx）
```

## 架构

### 后端分层结构（Go + Gin + GORM）

后端在 `internal/` 目录下遵循三层架构模式：

- **`server/`** - HTTP 处理器、路由定义、中间件集成（Gin）
- **`biz/`** - 业务逻辑层，UseCase 模式（`*_usecase.go` 文件）
- **`data/`** - 数据访问层，Repository 模式（`*_repository.go` 文件），GORM 模型和查询

入口：`main.go` -> `cmd/server/` 启动 Gin HTTP 服务，监听端口 9876。

### 依赖注入与服务组装

各业务域通过 `New*Services()` 工厂函数组装依赖链：`Repository → UseCase → Service → HTTPServer`。例如 `internal/server/asset/http.go` 的 `NewAssetServices(db)` 创建整个资产域的 Repo → UseCase → Service 链。顶层组装在 `internal/server/http.go` 的 `NewHTTPServer()` 中完成。

新增业务域时需遵循此模式：在对应 `server/` 子包中创建 `New*Services()` 工厂函数，在 `http.go` 中注册路由。

### 插件系统

前后端实现了对称的插件架构，每个插件自包含路由、菜单和数据库迁移。

**后端**（`internal/plugin/plugin.go`）：插件实现 `Plugin` 接口（`Name`、`Description`、`Version`、`Author`、`Enable`、`Disable`、`RegisterRoutes`、`GetMenus`）。`Manager` 负责插件注册、生命周期管理，并将启用状态持久化到 `plugin_states` 表。仅已启用的插件会注册路由。插件在 `internal/server/http.go` 中通过 `pluginMgr.Register()` 注册。

**前端**（`web/src/plugins/types.ts`）：插件实现对应的 `Plugin` 接口，包含 `install`、`uninstall`、`getMenus`、`getRoutes`。插件管理器位于 `web/src/plugins/manager.ts`，通过 `router.addRoute('Layout', route)` 动态注册路由，启用状态存储在 localStorage（`opshub_installed_plugins`）。

**插件实现**位于 `plugins/`（后端）和 `web/src/plugins/`（前端）：
- `kubernetes/` - 多集群 K8s 管理、Web 终端、集群巡检
- `task/` - 脚本执行（SSH/Agent）、模板管理、文件分发（SSH/SFTP/Agent），通过 `SetAgentHub()` 注入 Agent 能力
- `monitor/` - 域名监控、SSL 证书到期告警
- `nginx/` - Nginx 日志分析与 IP 地理定位
- `ssl-cert/` - ACME 自动续期、DNS 验证、部署到 Nginx/K8s
- `test/` - 示例插件

### 权限系统

**后端**：`pkg/middleware/` 提供 JWT 认证和审计中间件。`internal/biz/rbac/` 实现基于角色的权限控制，资产权限使用位掩码（View=1, Edit=2, Delete=4, Connect=8, Execute=16, Collect=32, File=64, Terminal=128）。路由级权限通过 `authMiddleware.RequireHostPermission()` 等方法控制。

**前端**：
- `web/src/stores/permission.ts` — Pinia store，从菜单树中提取按钮权限码（type=3 的菜单项）
- `web/src/directives/permission.ts` — `v-permission` 指令，用法：`v-permission="'middlewares:create'"`，无权限时移除 DOM 元素
- 管理员用户自动拥有所有权限

### 中间件连接器模式

`internal/service/asset/middleware_connector.go` 定义了 `MiddlewareConnector` 接口（`TestConnection` + `Execute`），通过工厂函数 `GetConnector(mwType)` 获取对应实现。当前支持：MySQLConnector、RedisConnector、ClickHouseConnector、MongoDBConnector、KafkaConnector（空壳）、MilvusConnector（空壳）。

新增中间件类型时需：实现 `MiddlewareConnector` 接口 → 在 `GetConnector` 中注册 → 在 `internal/biz/asset/middleware.go` 中添加类型常量和默认端口。

### 终端与审计

**SSH 终端**（`internal/server/asset/terminal.go`）：通过 WebSocket 代理 SSH 会话，路径 `/api/v1/asset/terminal/:id`。

**Agent 终端**（`internal/server/agent/terminal_handler.go`）：通过 WebSocket 代理 Agent PTY 会话，路径 `/api/v1/agent/terminal/:hostId`。Agent 不在线时前端自动回退到 SSH 终端。

**终端审计**（`internal/server/asset/terminal_audit.go`）：
- `TerminalSession` 模型记录每次终端会话，包含 `ConnectionType` 字段（`ssh`/`agent`）
- SSH 终端保存会话时设置 `ConnectionType: "ssh"`，Agent 终端设置 `ConnectionType: "agent"`
- 审计列表 VO 包含 `ConnectionType` 和 `ConnectionTypeText` 用于前端展示

**前端终端**（`web/src/views/asset/Terminal.vue`）：
- 加载主机列表后调用 `getAgentStatuses()` 合并 Agent 状态
- Agent 已安装的主机优先使用 Agent 终端端点
- WebSocket 连接失败时（`onerror`/`onclose` 且未曾成功连接）自动回退 SSH 端点重连

### 前端（Vue 3 + TypeScript + Arco Design Vue）

- 构建工具：Vite（`web/vite.config.ts`）
- 状态管理：Pinia（`web/src/stores/`）
- UI 组件库：Arco Design Vue（主库），Element Plus（仅旧页面兼容）
- API 客户端：基于 Axios，按业务域组织在 `web/src/api/`
- 页面视图按业务域组织在 `web/src/views/`
- 路由：`web/src/router/index.ts` 定义核心路由，插件路由动态注入
- Agent 状态：`web/src/api/agent.ts` 的 `getAgentStatuses()` 获取实时 Agent 在线状态，各页面加载主机列表后合并 Agent 状态信息

### 公共工具包

`pkg/` 包含可导出的公共包：`middleware/`（CORS、认证、审计）、`logger/`（zap 日志）、`response/`（标准 JSON 响应格式）、`ssh/`（SSH 客户端）、`collector/`（主机信息采集）、`agentproto/`（Agent gRPC protobuf 定义）、`utils/`、`error/`。

### Agent 系统

OpsHub 支持在主机上部署轻量级 Agent，通过 gRPC 双向流与服务端通信。Agent 在线时，终端连接、任务执行、文件分发、主机信息采集均优先使用 Agent 通道，离线时自动回退到 SSH。

**Agent 端**（`agent/`）：独立的 Go 二进制，包含：
- `agent/internal/filemanager/` — 本地文件操作（list/upload/download/delete）
- `agent/internal/terminal/` — 终端 PTY 管理
- Agent 启动后通过 gRPC 连接服务端，注册自身 AgentID 和 HostID

**服务端 Agent 管理**（`internal/server/agent/`）：
- `hub.go` — `AgentHub` 管理所有 Agent 连接，提供 `IsOnline(hostID)`、`GetByHostID(hostID)`、`WaitResponse(as, requestID, timeout)` 等方法
- `grpc_server.go` — gRPC 服务端，处理 Agent 的 `Connect` 双向流
- `terminal_handler.go` — Agent 终端 WebSocket 处理
- `agent_executor.go` — `AgentExecutor` 实现 `collector.CommandExecutor` 接口，通过 gRPC 执行命令；`AgentCommandFactory` 提供工厂方法

**Agent 通信协议**（`pkg/agentproto/`）：
- `ServerMessage` → Agent：`CmdRequest`（命令执行）、`FileRequest`（文件操作）、`TerminalInput`（终端输入）
- `AgentMessage` → Server：`CommandResult`（命令结果）、`FileChunk`/`FileListResult`（文件结果）、`TerminalOutput`（终端输出）
- 请求-响应通过 `requestID` 关联，使用 `AgentStream.RegisterPending/ResolvePending` 机制

**Agent 优先策略**：所有涉及主机操作的功能（终端、任务执行、文件分发、信息采集）统一遵循以下策略：
- 执行模式 `auto`（默认）：Agent 在线则使用 Agent，否则回退 SSH
- 执行模式 `agent`：仅使用 Agent，不在线则报错
- 执行模式 `ssh`：仅使用 SSH

**注入方式**：Agent 能力通过 setter 方法注入，避免修改接口签名：
- `HostUseCase.SetAgentCommandFactory(factory)` — 主机信息采集
- `taskPlugin.SetAgentHub(hub)` — 任务执行和文件分发
- 注入在 `internal/server/http.go` 的 `registerRoutes()` 中完成

### 采集器（Collector）

`pkg/collector/collector.go` 定义了 `CommandExecutor` 接口和 `Collector` 采集器：

```go
type CommandExecutor interface {
    Execute(cmd string) (string, error)
}
```

- `NewCollector(sshClient)` — 使用 SSH 客户端创建（`sshclient.Client` 已实现 `CommandExecutor`）
- `NewCollectorWithExecutor(executor)` — 使用任意 `CommandExecutor` 创建（如 `AgentExecutor`）
- `CollectAll()` 并发采集 CPU、内存、磁盘、系统信息、运行时间

新增采集方式时只需实现 `CommandExecutor` 接口。

## 配置

- 运行时配置：`config/config.yaml`（从 `config/config.yaml.example` 复制）
- 关键配置项：服务端口（9876）、MySQL 连接、Redis 连接、JWT 密钥
- 数据库：MySQL 8.0+，使用 GORM 自动迁移
- CLI 框架：Cobra（`cmd/` 目录）
- 默认账号：admin / 123456

## API 规范

- 基础路径：`/api/v1/*`
- 认证方式：JWT Bearer Token
- 标准响应格式：`{"code": 0, "message": "success", "data": {}}`
- 标准错误响应：使用 `pkg/response` 包的 `response.ErrorCode(c, httpStatus, message)` 和 `response.Success(c, data)`
- Swagger 文档：`http://localhost:9876/swagger/index.html`

## 数据库自动迁移

应用启动时通过 GORM AutoMigrate 自动同步表结构，分布在 3 个位置按顺序执行：

1. **核心系统表** - `cmd/server/server.go` 的 `autoMigrate()` 函数（RBAC、审计、系统配置、终端会话等）
2. **插件表** - 各插件 `Enable()` 方法中（通过 `internal/server/http.go` 的 `enablePlugins()` 触发）
3. **身份认证表** - `internal/server/identity/http.go` 的 `NewIdentityServices()` 中

新增功能涉及表结构变更时，需在对应位置追加 AutoMigrate 注册。详见 skill: `.claude/skills/db-migration.md`。

## 主要依赖

- **后端**：Go 1.25、Gin、GORM、client-go（K8s）、Cobra/Viper（CLI/配置）、zap（日志）、gorilla/websocket、google/uuid、pkg/sftp、lego（ACME）、clickhouse-go、mongo-driver、多云 SDK（阿里云、AWS、华为云、腾讯云）、gRPC + protobuf（Agent 通信）
- **前端**：Vue 3.5+、TypeScript 5.9+、Arco Design Vue（主 UI 库）、Element Plus（兼容未迁移页面）、Vite 5、Pinia、xterm.js、ECharts、Axios、CodeMirror（SQL 编辑器）、sql-formatter

## 前端 UI 规范（Arco Design 风格）

所有新增和修改的前端页面必须遵循以下规范，确保全局风格统一。

### 组件库优先级

1. 优先使用 Arco Design Vue（`@arco-design/web-vue`）组件
2. Element Plus 仅用于尚未迁移的旧页面，新页面禁止使用 Element Plus 组件
3. 两库共存期间，CSS 加载顺序：Arco CSS → Element Plus CSS → 自定义主题 CSS

### 色彩体系（CSS 变量定义在 `web/src/styles/arco-theme.css`）

- 品牌主色：`--ops-primary: #165dff`（Arco Blue）
- 侧边栏背景：`--ops-sidebar-bg: #232324`
- 顶栏背景：`--ops-header-bg: #ffffff`
- 内容区背景：`--ops-content-bg: #f7f8fa`
- 文字主色：`--ops-text-primary: #1d2129`
- 文字次色：`--ops-text-secondary: #4e5969`
- 文字辅助色：`--ops-text-tertiary: #86909c`
- 边框色：`--ops-border-color: #e5e6eb`
- 功能色：success `#00b42a`、warning `#ff7d00`、danger `#f53f3f`

### 布局规范

- 侧边栏宽度 220px，折叠后 48px，深色背景 + Arco dark menu
- 顶栏高度 60px，白色背景，底部 1px 边框 + 轻微阴影
- 内容区 padding 20px，浅灰背景
- 卡片圆角 8px（`--ops-border-radius-md`），表格圆角 4px（`--ops-border-radius-sm`）

### 图标映射

后端菜单数据中的图标名为 Element Plus 格式（如 `HomeFilled`、`Setting`），前端通过 `arcoIconMap` 映射到 Arco 图标组件。新增菜单图标时需在 `Layout.vue` 的映射表中添加对应条目。

### 页面编写规范

- 统计卡片使用 `a-card` + `hoverable`，内部用 flex 布局（图标 + 数值 + 标签）
- 表格使用 `a-table`，搜索栏使用 `a-form` inline 模式
- 弹窗使用 `a-modal`，表单使用 `a-form`
- 消息提示使用 `Message`（从 `@arco-design/web-vue` 导入），不使用 `ElMessage`
- 按钮使用 `a-button`，链接按钮使用 `a-link`
- 下拉选择使用 `a-select`，输入框使用 `a-input`
- 栅格布局使用 `a-row` + `a-col`（24 栅格）