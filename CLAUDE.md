# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

OpsHub 是一个插件化的云原生运维管理平台。项目采用单仓库结构，后端使用 Go，前端使用 Vue 3，通过插件架构组织功能模块（Kubernetes 管理、任务执行、监控告警、Nginx 日志分析、SSL 证书管理），各插件可独立启用/禁用。

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
- `task/` - 脚本执行、模板管理、文件分发
- `monitor/` - 域名监控、SSL 证书到期告警
- `nginx/` - Nginx 日志分析与 IP 地理定位
- `ssl-cert/` - ACME 自动续期、DNS 验证、部署到 Nginx/K8s
- `test/` - 示例插件

### 权限系统

**后端**：`pkg/middleware/` 提供 JWT 认证和审计中间件。`internal/biz/rbac/` 实现基于角色的权限控制，资产权限使用位掩码（View=1, Edit=2, Delete=4, Connect=8, Execute=16）。路由级权限通过 `authMiddleware.RequireHostPermission()` 等方法控制。

**前端**：
- `web/src/stores/permission.ts` — Pinia store，从菜单树中提取按钮权限码（type=3 的菜单项）
- `web/src/directives/permission.ts` — `v-permission` 指令，用法：`v-permission="'middlewares:create'"`，无权限时移除 DOM 元素
- 管理员用户自动拥有所有权限

### 中间件连接器模式

`internal/service/asset/middleware_connector.go` 定义了 `MiddlewareConnector` 接口（`TestConnection` + `Execute`），通过工厂函数 `GetConnector(mwType)` 获取对应实现。当前支持：MySQLConnector、RedisConnector、ClickHouseConnector、MongoDBConnector、KafkaConnector（空壳）、MilvusConnector（空壳）。

新增中间件类型时需：实现 `MiddlewareConnector` 接口 → 在 `GetConnector` 中注册 → 在 `internal/biz/asset/middleware.go` 中添加类型常量和默认端口。

### 前端（Vue 3 + TypeScript + Element Plus）

- 构建工具：Vite（`web/vite.config.ts`）
- 状态管理：Pinia（`web/src/stores/`）
- UI 组件库：Element Plus
- API 客户端：基于 Axios，按业务域组织在 `web/src/api/`
- 页面视图按业务域组织在 `web/src/views/`
- 路由：`web/src/router/index.ts` 定义核心路由，插件路由动态注入

### 公共工具包

`pkg/` 包含可导出的公共包：`middleware/`（CORS、认证、审计）、`logger/`（zap 日志）、`response/`（标准 JSON 响应格式）、`ssh/`（SSH 客户端）、`utils/`、`error/`。

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

1. **核心系统表** - `cmd/server/server.go` 的 `autoMigrate()` 函数（RBAC、审计、系统配置等）
2. **插件表** - 各插件 `Enable()` 方法中（通过 `internal/server/http.go` 的 `enablePlugins()` 触发）
3. **身份认证表** - `internal/server/identity/http.go` 的 `NewIdentityServices()` 中

新增功能涉及表结构变更时，需在对应位置追加 AutoMigrate 注册。详见 skill: `.claude/skills/db-migration.md`。

## 主要依赖

- **后端**：Go 1.25、Gin、GORM、client-go（K8s）、Cobra/Viper（CLI/配置）、zap（日志）、gorilla/websocket、lego（ACME）、clickhouse-go、mongo-driver、多云 SDK（阿里云、AWS、华为云、腾讯云）
- **前端**：Vue 3.5+、TypeScript 5.9+、Element Plus、Vite 5、Pinia、xterm.js、ECharts、Axios、CodeMirror（SQL 编辑器）、sql-formatter