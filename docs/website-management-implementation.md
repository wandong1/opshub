# Web站点管理功能实现完成

## 功能概述

已成功实现 Web 站点管理功能，支持统一管理内外部 Web 站点，并通过 Agent 代理解决内网站点访问问题。

## 实现内容

### 后端实现

1. **数据模型层** (`internal/biz/asset/website.go`)
   - `Website` - 站点模型
   - `WebsiteGroup` - 站点与业务分组关联表
   - `WebsiteAgent` - 站点与Agent主机关联表
   - 支持字段：站点名称、URL、图标、类型（外部/内部）、加密凭据、访问用户名/密码等

2. **Repository层** (`internal/data/asset/website.go`)
   - 实现 `WebsiteRepo` 接口
   - 支持站点的增删改查
   - 支持分组关联和Agent关联管理

3. **UseCase层** (`internal/biz/asset/website_usecase.go`)
   - 业务逻辑处理
   - 敏感信息加密（AES-256-GCM）
   - 内部站点必须绑定至少1台Agent主机的验证
   - 数据转换为VO

4. **Service层** (`internal/service/asset/website_service.go`)
   - HTTP请求处理
   - 参数验证和错误处理

5. **代理处理器** (`internal/server/asset/website_proxy.go`)
   - 外部站点：直接返回URL
   - 内部站点：通过Agent代理访问
   - 自动选择在线的Agent主机
   - HTTP请求代理转发

6. **路由注册** (`internal/server/asset/http.go`)
   - GET `/api/v1/websites` - 获取站点列表
   - GET `/api/v1/websites/:id` - 获取站点详情
   - POST `/api/v1/websites` - 创建站点
   - PUT `/api/v1/websites/:id` - 更新站点
   - DELETE `/api/v1/websites/:id` - 删除站点
   - GET `/api/v1/websites/:id/access` - 获取访问信息
   - ANY `/api/v1/websites/:id/proxy/*path` - 代理请求

7. **数据库迁移** (`cmd/server/server.go`)
   - 自动创建 `websites`、`website_groups`、`website_agents` 表

### 前端实现

1. **API接口** (`web/src/api/website.ts`)
   - TypeScript类型定义
   - 完整的CRUD接口封装

2. **页面视图** (`web/src/views/asset/Websites.vue`)
   - 左侧业务分组树
   - 右侧站点列表（支持搜索、筛选、分页）
   - 新增/编辑站点弹窗
   - 站点访问功能
   - 遵循 Arco Design 设计规范

3. **路由配置** (`web/src/router/index.ts`)
   - 路由路径：`/asset/websites`
   - 路由名称：`AssetWebsites`

## 核心特性

### 1. 站点类型
- **外部站点**：直接在新标签页打开URL
- **内部站点**：通过Agent代理访问，解决内网隔离问题

### 2. Agent代理机制
- 内部站点必须绑定至少1台Agent主机
- 自动选择在线的Agent进行代理
- Agent离线时提示用户无法访问
- 支持HTTP请求完整代理（请求头、响应头、状态码）

### 3. 数据安全
- 访问密码使用AES-256-GCM加密存储
- 加密凭据使用AES-256-GCM加密存储
- 前端仅显示脱敏信息

### 4. 业务分组
- 支持多业务分组关联
- 通过分组筛选站点
- 分组树形展示

### 5. Agent状态监控
- 实时显示Agent在线/离线状态
- 列表中显示Agent状态标签
- 访问前检查Agent可用性

## 技术亮点

1. **避免循环导入**：使用接口 `AgentHubInterface` 解耦 asset 和 agent 包
2. **加密实现**：内置AES-256-GCM加密，无需外部依赖
3. **三层架构**：严格遵循 Repository → UseCase → Service 模式
4. **类型安全**：前端使用TypeScript完整类型定义
5. **UI一致性**：完全遵循Arco Design设计规范

## 使用方式

### 创建外部站点
1. 点击"新增站点"
2. 填写站点名称和URL
3. 选择类型为"外部站点"
4. 可选填写访问用户名/密码、加密凭据
5. 选择业务分组
6. 保存

### 创建内部站点
1. 点击"新增站点"
2. 填写站点名称和内网URL
3. 选择类型为"内部站点"
4. **必须**绑定至少1台Agent主机
5. 可选填写访问用户名/密码、加密凭据
6. 选择业务分组
7. 保存

### 访问站点
1. 在站点列表中点击"访问"按钮
2. 外部站点：直接在新标签页打开
3. 内部站点：通过Agent代理访问（需要Agent在线）

## 数据库表结构

### websites 表
- id, name, url, icon, type, credential, secure_copy_url
- access_user, access_password, description, status
- created_at, updated_at, deleted_at

### website_groups 表
- id, website_id, group_id, created_at

### website_agents 表
- id, website_id, host_id, created_at

## 编译验证

✅ 后端编译成功
✅ 无循环依赖
✅ 类型检查通过

## 下一步

1. 在数据库中添加菜单项（通过管理后台）
2. 启动服务测试功能
3. 根据实际使用情况优化体验
