# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

OpsHub 是一个插件化的云原生运维管理平台。项目采用单仓库结构，后端使用 Go，前端使用 Vue 3，通过插件架构组织功能模块（Kubernetes 管理、任务执行、监控告警、Nginx 日志分析、SSL 证书管理），各插件可独立启用/禁用。平台支持 Agent 和 SSH 双通道管理主机，Agent 在线时优先使用 gRPC 通道。

## 项目目录结构示例

```
opshub/
├── agent/                          # Agent 端独立项目
│   ├── cmd/                        # Agent 命令行入口
│   ├── internal/                   # Agent 内部实现
│   │   ├── client/                 # gRPC 客户端，连接服务端
│   │   ├── config/                 # Agent 配置管理
│   │   ├── executor/               # 命令执行器
│   │   ├── filemanager/            # 文件管理（上传/下载/列表/删除）
│   │   ├── logger/                 # 日志管理
│   │   ├── prober/                 # 探测器（健康检查）
│   │   └── terminal/               # PTY 终端管理
│   ├── config/                     # Agent 配置文件
│   ├── dist/                       # 构建产物目录
│   ├── build.sh                    # 多架构构建脚本
│   ├── go.mod                      # Agent Go 依赖
│   └── README.md                   # Agent 文档
│
├── cmd/                            # 服务端命令行
│   ├── root/                       # 根命令
│   ├── server/                     # 服务启动命令
│   ├── version/                    # 版本命令
│   └── config/                     # 配置命令
│
├── internal/                       # 服务端内部实现（不可导出）
│   ├── biz/                        # 业务逻辑层（UseCase）
│   │   ├── asset/                  # 资产管理业务逻辑
│   │   ├── audit/                  # 审计业务逻辑
│   │   ├── identity/               # 身份认证业务逻辑
│   │   ├── inspection/             # 拨测业务逻辑
│   │   ├── inspection_mgmt/        # 巡检管理业务逻辑
│   │   ├── rbac/                   # 权限控制业务逻辑
│   │   └── system/                 # 系统配置业务逻辑
│   │
│   ├── data/                       # 数据访问层（Repository + Model）
│   │   ├── agent/                  # Agent 数据模型和仓储
│   │   ├── asset/                  # 资产数据模型和仓储
│   │   ├── audit/                  # 审计数据模型和仓储
│   │   ├── identity/               # 身份认证数据模型和仓储
│   │   ├── inspection/             # 拨测数据模型和仓储
│   │   ├── inspection_mgmt/        # 巡检管理数据模型和仓储
│   │   ├── rbac/                   # RBAC 数据模型和仓储
│   │   └── system/                 # 系统配置数据模型和仓储
│   │
│   ├── service/                    # 服务层（Service）
│   │   ├── asset/                  # 资产服务
│   │   ├── audit/                  # 审计服务
│   │   ├── identity/               # 身份认证服务
│   │   ├── inspection/             # 拨测服务
│   │   ├── inspection_mgmt/        # 巡检管理服务
│   │   ├── rbac/                   # RBAC 服务
│   │   └── system/                 # 系统配置服务
│   │
│   ├── server/                     # HTTP 服务层（Handler + Router）
│   │   ├── agent/                  # Agent 管理 HTTP 处理器
│   │   ├── asset/                  # 资产管理 HTTP 处理器
│   │   ├── audit/                  # 审计 HTTP 处理器
│   │   ├── identity/               # 身份认证 HTTP 处理器
│   │   ├── inspection/             # 巡检/拨测 HTTP 处理器
│   │   ├── rbac/                   # RBAC HTTP 处理器
│   │   ├── system/                 # 系统配置 HTTP 处理器
│   │   └── http.go                 # 主 HTTP 服务器，路由注册
│   │
│   ├── agent/                      # Agent Hub（管理所有 Agent 连接）
│   ├── cache/                      # 缓存层
│   └── plugin/                     # 插件管理器
│
├── plugins/                        # 后端插件目录
│   ├── kubernetes/                 # K8s 管理插件
│   │   ├── biz/                    # K8s 业务逻辑
│   │   ├── data/                   # K8s 数据层
│   │   ├── model/                  # K8s 数据模型
│   │   ├── server/                 # K8s HTTP 处理器
│   │   ├── service/                # K8s 服务层
│   │   └── resources/              # K8s 资源定义
│   │
│   ├── task/                       # 任务执行插件
│   │   ├── model/                  # 任务数据模型
│   │   ├── repository/             # 任务仓储
│   │   ├── server/                 # 任务 HTTP 处理器
│   │   └── service/                # 任务服务（脚本执行、文件分发）
│   │
│   ├── monitor/                    # 监控告警插件
│   │   ├── model/                  # 监控数据模型
│   │   ├── repository/             # 监控仓储
│   │   ├── server/                 # 监控 HTTP 处理器
│   │   └── service/                # 监控服务
│   │
│   ├── nginx/                      # Nginx 日志分析插件
│   │   ├── data/                   # Nginx 数据处理
│   │   ├── model/                  # Nginx 数据模型
│   │   ├── repository/             # Nginx 仓储
│   │   ├── server/                 # Nginx HTTP 处理器
│   │   └── service/                # Nginx 服务
│   │
│   ├── ssl-cert/                   # SSL 证书管理插件
│   │   ├── deployer/               # 证书部署器
│   │   ├── model/                  # 证书数据模型
│   │   ├── provider/               # ACME 提供商
│   │   ├── repository/             # 证书仓储
│   │   ├── server/                 # 证书 HTTP 处理器
│   │   └── service/                # 证书服务
│   │
│   ├── inspection/                 # 拨测插件
│   │   ├── dto/                    # 数据传输对象
│   │   ├── executor/               # 拨测执行器
│   │   ├── model/                  # 拨测数据模型
│   │   ├── repository/             # 拨测仓储
│   │   ├── server/                 # 拨测 HTTP 处理器
│   │   └── service/                # 拨测服务
│   │
│   └── test/                       # 测试插件示例
│
├── pkg/                            # 公共工具包（可导出）
│   ├── agentproto/                 # Agent gRPC protobuf 定义
│   ├── collector/                  # 主机信息采集器
│   ├── error/                      # 错误定义
│   ├── logger/                     # 日志工具（zap）
│   ├── metrics/                    # 指标采集
│   ├── middleware/                 # HTTP 中间件（认证、审计、CORS）
│   ├── response/                   # 标准响应格式
│   ├── scheduler/                  # 任务调度器
│   ├── ssh/                        # SSH 客户端
│   ├── utils/                      # 通用工具函数
│   └── version/                    # 版本信息
│
├── web/                            # 前端项目（Vue 3 + TypeScript）
│   ├── public/                     # 静态资源
│   │   └── uploads/                # 上传文件目录
│   │
│   ├── src/
│   │   ├── api/                    # API 客户端（按业务域组织）
│   │   │   ├── agent.ts            # Agent API
│   │   │   ├── asset.ts            # 资产 API
│   │   │   ├── audit.ts            # 审计 API
│   │   │   ├── inspection.ts       # 巡检/拨测 API
│   │   │   ├── rbac.ts             # RBAC API
│   │   │   └── system.ts           # 系统 API
│   │   │
│   │   ├── assets/                 # 静态资源（图片、字体等）
│   │   │
│   │   ├── components/             # 公共组件
│   │   │
│   │   ├── directives/             # 自定义指令
│   │   │   └── permission.ts       # 权限指令 v-permission
│   │   │
│   │   ├── plugins/                # 前端插件
│   │   │   ├── kubernetes/         # K8s 插件
│   │   │   ├── monitor/            # 监控插件
│   │   │   ├── nginx/              # Nginx 插件
│   │   │   ├── ssl-cert/           # SSL 证书插件
│   │   │   ├── task/               # 任务插件
│   │   │   ├── test/               # 测试插件
│   │   │   ├── manager.ts          # 插件管理器
│   │   │   └── types.ts            # 插件类型定义
│   │   │
│   │   ├── router/                 # 路由配置
│   │   │   └── index.ts            # 路由定义
│   │   │
│   │   ├── stores/                 # Pinia 状态管理
│   │   │   ├── permission.ts       # 权限 store
│   │   │   ├── user.ts             # 用户 store
│   │   │   └── app.ts              # 应用 store
│   │   │
│   │   ├── styles/                 # 全局样式
│   │   │   ├── arco-theme.css      # Arco Design 主题
│   │   │   └── global.css          # 全局样式
│   │   │
│   │   ├── utils/                  # 工具函数
│   │   │
│   │   ├── views/                  # 页面视图（按业务域组织）
│   │   │   ├── asset/              # 资产管理页面
│   │   │   ├── audit/              # 审计页面
│   │   │   ├── identity/           # 身份认证页面
│   │   │   ├── inspection/         # 巡检/拨测页面
│   │   │   ├── kubernetes/         # K8s 管理页面
│   │   │   ├── plugin/             # 插件管理页面
│   │   │   ├── system/             # 系统配置页面
│   │   │   └── task/               # 任务执行页面
│   │   │
│   │   ├── App.vue                 # 根组件
│   │   └── main.ts                 # 入口文件
│   │
│   ├── index.html                  # HTML 模板
│   ├── vite.config.ts              # Vite 配置
│   ├── tsconfig.json               # TypeScript 配置
│   └── package.json                # 前端依赖
│
├── config/                         # 配置文件
│   ├── config.yaml                 # 运行时配置
│   └── config.yaml.example         # 配置示例
│
├── data/                           # 运行时数据目录
│   ├── agent-binaries/             # Agent 安装包
│   ├── agent-certs/                # Agent TLS 证书
│   └── terminal-recordings/        # 终端录像
│
├── logs/                           # 日志目录
│   └── app.log                     # 应用日志
│
├── api/                            # API 定义
│   └── proto/                      # Protobuf 定义
│
├── docs/                           # 文档
│   └── plugin-development/         # 插件开发文档
│
├── migrations/                     # 数据库迁移脚本
├── scripts/                        # 工具脚本
├── bin/                            # 编译产物
│   └── opshub                      # 服务端二进制
│
├── docker-compose.yml              # Docker Compose 配置
├── Dockerfile                      # Docker 镜像构建
├── Makefile                        # 构建脚本
├── go.mod                          # Go 依赖
├── go.sum                          # Go 依赖校验
├── main.go                         # 服务端入口
├── CLAUDE.md                       # 本文件
└── README.md                       # 项目说明
```

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
- 支持多架构：linux-amd64, linux-arm64, darwin-amd64, darwin-arm64

**Agent 构建与部署**：
- `agent/build.sh` — 多架构构建脚本，生成统一的多架构安装包（`srehub-agent-{version}-multi-arch.tar.gz`）
- 安装包包含所有平台二进制文件（位于 `bin/` 目录）
- 智能安装脚本自动检测操作系统和架构，选择对应二进制
- 构建产物自动复制到 `data/agent-binaries/` 供服务端部署使用
- 部署方式：通过平台 SSH 部署 或 手动安装

**服务端 Agent 管理**（`internal/server/agent/`）：
- `hub.go` — `AgentHub` 管理所有 Agent 连接，提供 `IsOnline(hostID)`、`GetByHostID(hostID)`、`WaitResponse(as, requestID, timeout)`、`CloseAll()` 等方法
- `grpc_server.go` — gRPC 服务端，处理 Agent 的 `Connect` 双向流，支持优雅关闭（5秒超时）
- `terminal_handler.go` — Agent 终端 WebSocket 处理
- `agent_executor.go` — `AgentExecutor` 实现 `collector.CommandExecutor` 接口，通过 gRPC 执行命令；`AgentCommandFactory` 提供工厂方法
- `deploy_handler.go` — Agent 部署处理，支持单机和批量部署，自动查找多架构安装包
- `agent_service.go` — Agent 注册逻辑，区分 SSH 部署和手动安装场景

**Agent 通信协议**（`pkg/agentproto/`）：
- `ServerMessage` → Agent：`CmdRequest`（命令执行）、`FileRequest`（文件操作）、`TerminalInput`（终端输入）
- `AgentMessage` → Server：`CommandResult`（命令结果）、`FileChunk`/`FileListResult`（文件结果）、`TerminalOutput`（终端输出）
- 请求-响应通过 `requestID` 关联，使用 `AgentStream.RegisterPending/ResolvePending` 机制

**Agent 注册与去重逻辑**：
- **SSH 部署场景**：平台通过 SSH 部署 Agent 时，先在 hosts 表创建记录，Agent 连接时通过 agent_id 或 IP 匹配现有主机，更新 agent_id 和主机名
- **手动安装场景**：用户手动安装 Agent 时，Agent 连接时自动注册新主机到"Agent自动注册"分组
- **IP 去重**：注册时优先检查 agent_id，其次检查 IP，防止创建重复主机记录
- **主机名更新**：允许 Agent 上报的真实主机名覆盖用户手动设置的主机名

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
## 开发规范

### 目录结构规范

#### 后端目录规范

**核心业务模块**（位于 `internal/`）：
- `biz/` - 业务逻辑层，包含 UseCase 和业务规则
- `data/` - 数据访问层，包含 Repository 接口、实现和 GORM 模型
- `service/` - 服务层，协调业务逻辑和数据访问
- `server/` - HTTP 处理层，包含路由注册和请求处理

**插件模块**（位于 `plugins/`）：
每个插件遵循相同的目录结构：
```
plugins/plugin-name/
├── model/          # 数据模型（GORM）
├── repository/     # 数据仓储接口和实现
├── service/        # 业务服务
├── server/         # HTTP 处理器和路由
├── biz/            # 业务逻辑（可选）
└── plugin.go       # 插件注册入口
```

**Agent 模块**（位于 `agent/`）：
```
agent/
├── cmd/            # 命令行入口
├── internal/       # 内部实现
│   ├── client/     # gRPC 客户端
│   ├── config/     # 配置管理
│   ├── executor/   # 命令执行
│   ├── filemanager/# 文件管理
│   ├── logger/     # 日志
│   ├── prober/     # 健康检查
│   └── terminal/   # 终端管理
├── config/         # 配置文件
└── build.sh        # 构建脚本
```

**公共包**（位于 `pkg/`）：
- 所有 `pkg/` 下的包必须是可导出的，不依赖 `internal/`
- 按功能划分子包：`logger/`、`middleware/`、`response/`、`ssh/`、`collector/` 等

#### 前端目录规范

**核心目录**（位于 `web/src/`）：
```
web/src/
├── api/            # API 客户端，按业务域组织（asset.ts, rbac.ts 等）
├── assets/         # 静态资源（图片、字体）
├── components/     # 公共组件
├── directives/     # 自定义指令（permission.ts）
├── plugins/        # 前端插件，每个插件一个子目录
├── router/         # 路由配置
├── stores/         # Pinia 状态管理
├── styles/         # 全局样式
├── utils/          # 工具函数
├── views/          # 页面视图，按业务域组织
├── App.vue         # 根组件
└── main.ts         # 入口文件
```

**插件目录**（位于 `web/src/plugins/`）：
每个插件包含：
- `index.ts` - 插件入口，导出 Plugin 对象
- `routes.ts` - 插件路由定义
- `views/` - 插件页面组件（可选）

#### 文件命名规范

**后端**：
- Go 文件使用 snake_case：`user_service.go`、`host_repository.go`
- 测试文件：`*_test.go`
- 接口定义文件：通常在 `biz/` 或 `data/` 中，如 `user_repo.go`
- HTTP 处理器：`http.go` 或 `*_handler.go`

**前端**：
- Vue 组件使用 PascalCase：`UserList.vue`、`HostDetail.vue`
- TypeScript 文件使用 camelCase：`userApi.ts`、`permission.ts`
- 类型定义文件：`types.ts` 或 `*.d.ts`
- 工具函数：`utils.ts` 或按功能命名如 `dateUtils.ts`

### 代码组织规范

#### 后端三层架构

**严格遵循依赖方向**：`Server → Service → Biz → Data`

1. **Data 层**（`internal/data/`）：
   - 定义 GORM 模型（struct）
   - 定义 Repository 接口
   - 实现 Repository 接口
   - 只依赖数据库和缓存，不依赖业务逻辑

2. **Biz 层**（`internal/biz/`）：
   - 定义业务接口（UseCase）
   - 实现业务逻辑
   - 依赖 Repository 接口，不依赖具体实现
   - 处理业务规则和数据转换

3. **Service 层**（`internal/service/`）：
   - 协调多个 UseCase
   - 处理事务边界
   - 数据格式转换（DTO ↔ Model）
   - 依赖 Biz 层接口

4. **Server 层**（`internal/server/`）：
   - HTTP 请求处理
   - 路由注册
   - 参数验证
   - 响应格式化
   - 依赖 Service 层

**示例**：
```go
// data/user_repository.go
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id uint) (*User, error)
}

// biz/user_usecase.go
type UserUseCase interface {
    CreateUser(ctx context.Context, req *CreateUserRequest) error
}

// service/user_service.go
type UserService struct {
    userUC UserUseCase
}

// server/user_handler.go
func (s *HTTPServer) createUser(c *gin.Context) {
    // 处理 HTTP 请求
}
```

#### 依赖注入模式

**使用工厂函数组装依赖链**：

```go
// internal/server/asset/http.go
func NewAssetServices(db *gorm.DB) (*HTTPServer, error) {
    // 1. 创建 Repository
    hostRepo := assetdata.NewHostRepository(db)

    // 2. 创建 UseCase
    hostUC := assetbiz.NewHostUseCase(hostRepo)

    // 3. 创建 Service
    hostService := assetservice.NewHostService(hostUC)

    // 4. 创建 HTTPServer
    return &HTTPServer{
        hostService: hostService,
    }, nil
}
```

**顶层组装**（`internal/server/http.go`）：
```go
func NewHTTPServer(db *gorm.DB, rdb *redis.Client) *HTTPServer {
    // 初始化各业务域
    assetServer, _ := asset.NewAssetServices(db)
    rbacServer, _ := rbac.NewRBACServices(db, rdb)

    return &HTTPServer{
        assetServer: assetServer,
        rbacServer:  rbacServer,
    }
}
```

#### 插件开发规范

**后端插件**必须实现 `Plugin` 接口：
```go
type Plugin interface {
    Name() string
    Description() string
    Version() string
    Author() string
    Enable(db *gorm.DB) error
    Disable() error
    RegisterRoutes(r *gin.RouterGroup)
    GetMenus() []Menu
}
```

**前端插件**必须导出 `Plugin` 对象：
```typescript
export default {
    name: 'plugin-name',
    version: '1.0.0',
    install: () => { /* 安装逻辑 */ },
    uninstall: () => { /* 卸载逻辑 */ },
    getMenus: () => [ /* 菜单定义 */ ],
    getRoutes: () => [ /* 路由定义 */ ]
}
```

### 项目记录

使用 `/record-milestone` skill 记录项目开发过程：
- 📋 需求记录：包含时间、提出人、描述、状态
- ✅ 实现记录：包含时间、开发者、关联需求、修改文件
- 🎯 里程碑记录：包含时间、版本、参与人员、主要功能、技术亮点

所有记录自动追加到项目根目录的 `record.md` 文件中，保持开发历史的完整性和可追溯性。详见 `.claude/skills/record-milestone.md`。

### 代码提交规范

- 提交信息使用中文，格式：`类型(模块): 简短描述`
- 类型：feat（新功能）、fix（修复）、refactor（重构）、docs（文档）、style（格式）、test（测试）
- 示例：`feat(Agent): 支持多架构部署`、`fix(Agent): 修复 IP 重复检查逻辑`

### Agent 开发注意事项

1. **部署场景区分**：
   - SSH 部署：平台通过 SSH 部署 Agent，主机记录已存在
   - 手动安装：用户手动安装 Agent，需要自动注册新主机

2. **主机去重逻辑**：
   - 优先通过 `agent_id` 查找主机
   - 其次通过 `IP` 查找主机（防止 SSH 部署后创建重复记录）
   - 只有 IP 不存在时才创建新主机

3. **主机名更新策略**：
   - 允许 Agent 上报的真实主机名覆盖用户手动设置的主机名
   - 确保主机信息的准确性

4. **多架构支持**：
   - 使用 `agent/build.sh` 构建多架构安装包
   - 安装脚本自动检测平台并选择对应二进制
   - 部署时优先使用 `multi-arch` 包

5. **优雅关闭**：
   - 服务关闭时主动关闭所有 Agent 连接
   - 使用 5 秒超时机制，超时后强制停止

### API 开发规范

#### 路由命名规范

**RESTful 风格**：
- 资源使用复数名词：`/api/v1/hosts`、`/api/v1/users`
- 使用 HTTP 方法表示操作：
  - `GET /api/v1/hosts` - 列表查询
  - `GET /api/v1/hosts/:id` - 详情查询
  - `POST /api/v1/hosts` - 创建
  - `PUT /api/v1/hosts/:id` - 更新
  - `DELETE /api/v1/hosts/:id` - 删除
- 子资源嵌套：`/api/v1/hosts/:id/terminals`
- 操作动词：`/api/v1/hosts/:id/connect`、`/api/v1/tasks/:id/execute`

#### 请求参数规范

**查询参数**（GET 请求）：
```go
type ListRequest struct {
    Page     int    `form:"page" binding:"required,min=1"`
    PageSize int    `form:"page_size" binding:"required,min=1,max=100"`
    Keyword  string `form:"keyword"`
    Status   string `form:"status"`
}
```

**请求体**（POST/PUT 请求）：
```go
type CreateHostRequest struct {
    Name     string `json:"name" binding:"required"`
    IP       string `json:"ip" binding:"required,ip"`
    Port     int    `json:"port" binding:"required,min=1,max=65535"`
    Username string `json:"username" binding:"required"`
}
```

#### 响应格式规范

**成功响应**：
```go
// 单条数据
response.Success(c, data)

// 分页数据
response.Pagination(c, total, page, pageSize, data)

// 带消息
response.SuccessWithMessage(c, "操作成功", data)
```

**错误响应**：
```go
// 使用标准错误码
response.Error(c, appError.New(appError.ErrNotFound, "主机不存在"))

// 简化版本
response.ErrorCode(c, http.StatusBadRequest, "参数错误")
```

**标准响应结构**：
```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "timestamp": 1234567890
}
```

**分页响应结构**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 100,
    "page": 1,
    "page_size": 20,
    "data": []
  },
  "timestamp": 1234567890
}
```

#### 错误处理规范

**使用统一的错误码**（`pkg/error/error.go`）：
```go
const (
    Success            ErrorCode = 0
    ErrBadRequest      ErrorCode = 400
    ErrUnauthorized    ErrorCode = 401
    ErrForbidden       ErrorCode = 403
    ErrNotFound        ErrorCode = 404
    ErrInternalServer  ErrorCode = 500
    // 业务错误码 2000-2999
    ErrHostNotFound    ErrorCode = 2001
    ErrAgentOffline    ErrorCode = 2002
)
```

**错误包装**：
```go
if err != nil {
    return appError.Wrap(err, appError.ErrDatabase, "查询主机失败")
}
```

### 数据库开发规范

#### 模型定义规范

**GORM 模型**：
```go
type Host struct {
    ID        uint           `gorm:"primarykey" json:"id"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

    Name     string `gorm:"size:100;not null" json:"name"`
    IP       string `gorm:"size:50;not null;index" json:"ip"`
    Port     int    `gorm:"not null;default:22" json:"port"`
    Status   string `gorm:"size:20;default:'offline'" json:"status"`
}

func (Host) TableName() string {
    return "asset_hosts"
}
```

**字段规范**：
- 主键使用 `uint` 类型的 `ID`
- 必须包含 `CreatedAt`、`UpdatedAt`、`DeletedAt`（软删除）
- 字符串字段指定 `size`
- 必填字段添加 `not null`
- 枚举字段使用 `default` 指定默认值
- 外键字段添加 `index`

#### Repository 规范

**接口定义**：
```go
type HostRepository interface {
    Create(ctx context.Context, host *Host) error
    Update(ctx context.Context, host *Host) error
    Delete(ctx context.Context, id uint) error
    GetByID(ctx context.Context, id uint) (*Host, error)
    List(ctx context.Context, page, pageSize int, filters map[string]interface{}) ([]*Host, int64, error)
}
```

**实现规范**：
```go
type hostRepository struct {
    db *gorm.DB
}

func NewHostRepository(db *gorm.DB) HostRepository {
    return &hostRepository{db: db}
}

func (r *hostRepository) GetByID(ctx context.Context, id uint) (*Host, error) {
    var host Host
    err := r.db.WithContext(ctx).First(&host, id).Error
    if err != nil {
        return nil, err
    }
    return &host, nil
}
```

**查询规范**：
- 所有查询必须使用 `WithContext(ctx)`
- 列表查询必须支持分页
- 使用 `Preload` 预加载关联数据
- 复杂查询使用链式调用构建

#### 数据库迁移规范

**自动迁移位置**：
1. 核心系统表 → `cmd/server/server.go` 的 `autoMigrate()`
2. 插件表 → 各插件的 `Enable()` 方法
3. 身份认证表 → `internal/server/identity/http.go` 的 `NewIdentityServices()`

**迁移示例**：
```go
func autoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(
        &assetdata.Host{},
        &assetdata.HostGroup{},
        &rbacdata.User{},
        &rbacdata.Role{},
    )
}
```

### 前端开发规范

#### 组件开发规范

**Vue 3 Composition API**：
```vue
<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import { getHostList } from '@/api/asset'

// 响应式数据
const loading = ref(false)
const dataList = ref([])
const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

// 方法
const loadData = async () => {
  loading.value = true
  try {
    const res = await getHostList(pagination)
    dataList.value = res.data
    pagination.total = res.total
  } catch (error) {
    Message.error('加载失败')
  } finally {
    loading.value = false
  }
}

// 生命周期
onMounted(() => {
  loadData()
})
</script>
```

**组件命名**：
- 使用 PascalCase：`<HostList />`、`<UserForm />`
- 多单词组件名：避免与 HTML 元素冲突

#### API 调用规范

**API 文件组织**（`web/src/api/`）：
```typescript
// asset.ts
import request from '@/utils/request'

export interface Host {
  id: number
  name: string
  ip: string
  port: number
}

export interface ListParams {
  page: number
  page_size: number
  keyword?: string
}

export const getHostList = (params: ListParams) => {
  return request.get('/api/v1/asset/hosts', { params })
}

export const getHostDetail = (id: number) => {
  return request.get(`/api/v1/asset/hosts/${id}`)
}

export const createHost = (data: Partial<Host>) => {
  return request.post('/api/v1/asset/hosts', data)
}
```

**请求拦截器**（`web/src/utils/request.ts`）：
- 自动添加 Authorization header
- 统一错误处理
- 响应数据解包

#### 状态管理规范

**Pinia Store**（`web/src/stores/`）：
```typescript
// user.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useUserStore = defineStore('user', () => {
  const userInfo = ref(null)
  const token = ref(localStorage.getItem('token') || '')

  const setToken = (newToken: string) => {
    token.value = newToken
    localStorage.setItem('token', newToken)
  }

  const logout = () => {
    token.value = ''
    userInfo.value = null
    localStorage.removeItem('token')
  }

  return {
    userInfo,
    token,
    setToken,
    logout
  }
})
```

#### 权限控制规范

**使用 v-permission 指令**：
```vue
<template>
  <a-button v-permission="'hosts:create'" @click="handleCreate">
    新增主机
  </a-button>

  <a-button v-permission="'hosts:delete'" status="danger" @click="handleDelete">
    删除
  </a-button>
</template>
```

**权限码命名**：
- 格式：`资源:操作`
- 示例：`hosts:create`、`hosts:edit`、`hosts:delete`、`users:view`

### 测试规范

#### 单元测试

**后端测试**（使用 Go testing）：
```go
func TestHostRepository_Create(t *testing.T) {
    db := setupTestDB(t)
    repo := NewHostRepository(db)

    host := &Host{
        Name: "test-host",
        IP:   "192.168.1.1",
        Port: 22,
    }

    err := repo.Create(context.Background(), host)
    assert.NoError(t, err)
    assert.NotZero(t, host.ID)
}
```

**前端测试**（使用 Vitest）：
```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import HostList from '@/views/asset/HostList.vue'

describe('HostList', () => {
  it('renders properly', () => {
    const wrapper = mount(HostList)
    expect(wrapper.find('.host-list').exists()).toBe(true)
  })
})
```

### 安全规范

1. **SQL 注入防护**：使用 GORM 参数化查询，禁止字符串拼接
2. **XSS 防护**：前端使用 Vue 自动转义，后端返回数据不包含 HTML
3. **CSRF 防护**：使用 JWT Token，不依赖 Cookie
4. **权限校验**：所有 API 必须进行权限验证
5. **敏感信息**：密码、密钥等敏感信息不记录日志，不返回前端
6. **文件上传**：限制文件类型、大小，使用随机文件名

### 性能优化规范

1. **数据库查询**：
   - 使用索引优化查询
   - 避免 N+1 查询，使用 Preload
   - 大数据量使用分页
   - 使用 Redis 缓存热点数据

2. **API 响应**：
   - 使用 gzip 压缩
   - 合理使用 HTTP 缓存
   - 避免返回过大的数据

3. **前端优化**：
   - 路由懒加载
   - 组件按需加载
   - 图片懒加载
   - 使用虚拟滚动处理大列表

### Agent 代理开发规范

#### HTTP 代理实现规范

**核心原则**：透明代理，保持原始 HTTP 语义

1. **协议定义**（`pkg/agentproto/agent.proto`）：
   ```protobuf
   message HttpProxyRequest {
       string request_id = 1;
       string method = 2;
       string url = 3;
       map<string, string> headers = 4;
       bytes body = 5;
       int32 timeout = 6;
   }
   
   message HttpProxyResponse {
       string request_id = 1;
       int32 status_code = 2;
       map<string, string> headers = 3;
       bytes body = 4;
       string error = 5;
   }
   ```

2. **请求-响应关联**：
   - 使用 UUID 作为 `request_id`
   - 通过 `AgentStream.RegisterPending(requestID, chan)` 注册等待通道
   - 通过 `AgentStream.ResolvePending(requestID, response)` 解析响应
   - 超时机制：默认 35 秒（Agent 执行超时 30 秒 + 网络延迟 5 秒）

3. **错误处理**：
   - 区分 gRPC 错误和 HTTP 错误
   - gRPC 错误：`error != "" && status_code == 0`（Agent 执行失败）
   - HTTP 错误：`status_code >= 400`（目标服务返回错误，需透传）
   ```go
   if proxyResp.Error != "" && proxyResp.StatusCode == 0 {
       return fmt.Errorf("Agent 执行失败: %s", proxyResp.Error)
   }
   // 透传 HTTP 错误响应（包括 4xx、5xx）
   c.Status(int(proxyResp.StatusCode))
   c.Writer.Write(proxyResp.Body)
   ```

4. **消息处理**：
   - 服务端必须在 `Connect()` 方法的 switch 语句中处理 `HttpProxyResponse`
   - 调用 `as.ResolvePending(requestID, response)` 唤醒等待的请求
   ```go
   case *pb.AgentMessage_HttpProxyResponse:
       if as != nil {
           as.ResolvePending(payload.HttpProxyResponse.RequestId, payload.HttpProxyResponse)
       }
   ```

#### 数据源 Agent 代理规范

**场景**：平台无法直接访问的边缘数据源（如内网 Prometheus、VictoriaMetrics）

1. **数据源配置**：
   - `AccessMode` 字段：`direct`（平台直接访问）或 `agent`（通过 Agent 代理）
   - `AgentHostIDs` 字段：绑定的 Agent 主机 ID 列表（支持多个，实现高可用）
   - `ProxyToken` 字段：UUID，用于生成无需认证的代理 URL

2. **代理 URL 生成**：
   ```go
   proxyURL := fmt.Sprintf("%s/api/v1/alert/proxy/datasource/%s", platformURL, ds.ProxyToken)
   ```
   - 公开路由，无需 JWT 认证（通过 `ProxyToken` 验证）
   - Grafana 等外部系统可直接使用此 URL

3. **路由注册**：
   ```go
   // 在 RegisterPublicRoutes() 中注册（不在认证中间件下）
   func (s *HTTPServer) RegisterPublicRoutes(router *gin.Engine) {
       proxy := router.Group("/api/v1/alert/proxy/datasource")
       {
           proxy.Any("/:token/*path", s.proxyDataSourceRequest)
       }
   }
   ```

4. **Agent 选择策略**：
   - 遍历 `AgentHostIDs`，选择第一个在线的 Agent
   - 如果所有 Agent 都离线，返回 503 Service Unavailable

5. **路径处理**：
   ```go
   // 移除代理前缀，构建目标 URL
   proxyPath := c.Request.URL.Path  // /api/v1/alert/proxy/datasource/{token}/api/v1/query
   targetPath := strings.TrimPrefix(proxyPath, "/api/v1/alert/proxy/datasource/"+token)
   targetURL := strings.TrimRight(ds.URL, "/") + targetPath
   ```

#### 站点 Agent 代理规范

**场景**：代理访问内网 Web 站点（支持 Vue/React/Angular 等现代 SPA 应用）

1. **站点配置**：
   - `Type` 字段：`external`（外部站点）或 `internal`（内部站点，需要代理）
   - `AgentHostIDs` 字段：绑定的 Agent 主机 ID 列表
   - `URL` 字段：目标站点的内网地址（如 `http://192.168.1.100:8080`）

2. **代理路由**：
   ```go
   // 公开路由，无需认证
   router.Any("/api/v1/websites/:id/proxy/*path", h.ProxyWebsiteRequest)
   ```

3. **路径重写策略**：
   
   **核心原则**：将绝对路径转换为带代理前缀的路径，相对路径由 `<base>` 标签处理
   
   - **HTML 处理**：
     - 注入 `<base href="/api/v1/websites/{id}/proxy/">` 标签
     - 重写所有 `src`/`href` 属性中的绝对路径（`/xxx` → `/api/v1/websites/{id}/proxy/xxx`）
     - 不限制扩展名，支持 SPA 路由（`/dashboard`、`/users/123`）
   
   - **CSS 处理**：
     - 重写 `url()` 中的绝对路径
     - 保持引号风格（`"`、`'`、无引号）
   
   - **JavaScript 处理**（保守策略）：
     - 只重写明确的资源路径（带扩展名：`.js`、`.css`、`.png`、`.vue` 等）
     - 不重写无扩展名的路径（避免误伤正则表达式、注释、日期格式等）
     - 支持的扩展名：`js|css|json|png|jpg|jpeg|gif|svg|woff|woff2|ttf|eot|ico|webp|mp4|webm|xml|txt|pdf|vue|jsx|tsx`

4. **内容重写实现**：
   ```go
   // 判断是否需要重写
   needRewrite := strings.Contains(contentType, "text/html") ||
                  strings.Contains(contentType, "text/css") ||
                  strings.Contains(contentType, "javascript")
   
   if needRewrite && len(responseBody) > 0 {
       // 处理 gzip 压缩
       if strings.Contains(contentEncoding, "gzip") {
           responseBody = h.decompressGzip(responseBody)
           delete(headers, "Content-Encoding")
       }
       
       // 重写路径
       responseBody = h.rewriteResourcePaths(responseBody, proxyPrefix, contentType)
       
       // 更新 Content-Length
       headers["Content-Length"] = fmt.Sprintf("%d", len(responseBody))
   }
   ```

5. **正则表达式规范**：
   ```go
   // HTML：匹配所有 src/href 属性
   reHtmlAttr = regexp.MustCompile(`(src|href)\s*=\s*["']([^"']+)["']`)
   
   // CSS：匹配 url()
   reCssUrl = regexp.MustCompile(`url\(\s*["']?([^"')]+)["']?\s*\)`)
   
   // JS：只匹配带资源扩展名的路径（保守策略）
   jsResourceExts = `\.(js|css|json|png|jpg|...)`
   reJsResourceDouble = regexp.MustCompile(`"(/[^"]*` + jsResourceExts + `[^"]*)"`)
   reJsResourceSingle = regexp.MustCompile(`'(/[^']*` + jsResourceExts + `[^']*)'`)
   ```

6. **过滤规则**：
   - 跳过已包含代理前缀的路径
   - 跳过外部链接：`http://`、`https://`、`//`
   - 跳过特殊协议：`data:`、`javascript:`、`mailto:`、`tel:`、`blob:`、`#`
   - 跳过相对路径：`./`、`../`（由 `<base>` 标签处理）

7. **gzip 处理**：
   ```go
   func (h *WebsiteProxyHandlerV2) decompressGzip(body []byte) ([]byte, error) {
       reader, err := gzip.NewReader(bytes.NewReader(body))
       if err != nil {
           return nil, err
       }
       defer reader.Close()
       return io.ReadAll(reader)
   }
   ```

8. **适配器模式**（避免循环导入）：
   ```go
   // internal/server/asset/agent_hub_adapter.go
   type AgentHubInterfaceV2 interface {
       IsOnline(hostID uint) bool
       GetByHostID(hostID uint) (AgentStreamInterface, bool)
       WaitResponse(as AgentStreamInterface, requestID string, timeout time.Duration) (interface{}, error)
   }
   
   // 使用反射调用真实的 AgentHub 方法
   func NewAgentHubAdapter(realHub interface{}) AgentHubInterfaceV2 {
       return &AgentHubAdapter{realHub: realHub}
   }
   ```

#### Agent 代理最佳实践

1. **超时设置**：
   - Agent 执行超时：30 秒
   - 服务端等待超时：35 秒（留 5 秒网络延迟）
   - 前端请求超时：40 秒

2. **错误日志**：
   - 记录 `request_id`、`target_url`、`status_code`、`error`
   - 区分 Agent 错误和 HTTP 错误
   - 使用结构化日志（zap）

3. **性能优化**：
   - 使用 gRPC 双向流，避免频繁建立连接
   - 响应体大小限制：建议 < 10MB
   - 对于大文件下载，考虑使用分块传输

4. **安全考虑**：
   - 数据源代理：使用 `ProxyToken` 验证，避免暴露内部 URL
   - 站点代理：验证站点 ID 和 Agent 绑定关系
   - 不在日志中记录敏感信息（密码、Token）

5. **测试建议**：
   - 单元测试：测试路径重写逻辑
   - 集成测试：测试 Agent 通信和超时处理
   - 端到端测试：测试完整的代理流程（HTML + CSS + JS + 图片）

## 命令操作规范

### 数据库操作

如果需要操作数据库数据，使用 Docker 命令进入容器：

```bash
# 进入 MySQL 容器
docker exec -it opshub-mysql mysql -uroot -p'OpsHub@2026' opshub

# 执行 SQL 查询
SELECT * FROM asset_hosts LIMIT 10;

# 进入 Redis 容器
docker exec -it opshub-redis redis-cli -a '1ujasdJ67Ps'

# 执行 Redis 命令
KEYS *
GET key_name
```

### 服务启动

```bash
# 后端服务
./bin/opshub server --config config/config.yaml

# 前端开发服务
cd web && npm run dev

# Docker Compose 启动全栈
docker-compose up -d
```

### 日志查看

```bash
# 查看应用日志
tail -f logs/app.log

# 查看 Docker 容器日志
docker logs -f opshub-server
docker logs -f opshub-mysql
```