# OpsHub 插件化架构说明

## 🎯 插件化架构概述

OpsHub 采用了**前后端分离的插件化架构**,实现了真正的功能可插拔。每个功能模块都可以作为独立插件开发、部署和管理。

### 核心特性

✅ **完全解耦** - 插件之间相互独立,互不影响
✅ **一键启用/禁用** - 通过代码配置即可控制插件
✅ **统一规范** - 前后端遵循统一的插件接口规范
✅ **热插拔** - 前端插件支持动态路由注册
✅ **版本管理** - 每个插件独立版本号和作者信息

## 📁 项目结构

```
opshub/
├── internal/
│   ├── plugin/                    # 插件核心框架
│   │   └── plugin.go              # 插件接口定义和管理器
│   └── plugins/                   # 插件实现目录
│       └── kubernetes/            # Kubernetes管理插件
│           └── plugin.go
│
├── web/src/
│   ├── plugins/                   # 前端插件核心框架
│   │   ├── types.ts               # 插件类型定义
│   │   ├── manager.ts             # 插件管理器
│   │   └── kubernetes/            # Kubernetes前端插件
│   │       └── index.ts
│   │
│   └── views/                     # 插件页面组件
│       └── kubernetes/            # Kubernetes插件页面
│           ├── Index.vue
│           ├── Clusters.vue
│           ├── Nodes.vue
│           └── ... (共10个子功能)
│
└── docs/                          # 文档
    ├── PLUGIN_DEVELOPMENT.md      # 详细开发指南
    └── PLUGIN_QUICK_START.md      # 快速参考
```

## 🚀 Kubernetes管理插件

系统已内置一个完整的 Kubernetes 容器管理插件示例,包含以下10个子功能:

### 菜单结构

```
📦 Kubernetes管理
├── 🏢 集群管理         (Clusters)
├── 🖥️  节点管理          (Nodes)
├── ⚙️  工作负载         (Workloads)
├── 📁 命名空间        (Namespaces)
├── 🌐 网络管理         (Network)
├── 📄 配置管理         (Config)
├── 💾 存储管理         (Storage)
├── 🔒 访问控制         (Access)
├── 👁️  终端审计        (Audit)
└── 🔍 应用诊断        (Diagnostic)
```

### API 路由

所有插件API遵循统一规范: `/api/v1/plugins/{plugin_name}/{resources}`

示例:
```
GET    /api/v1/plugins/kubernetes/clusters        # 获取集群列表
POST   /api/v1/plugins/kubernetes/clusters        # 创建集群
GET    /api/v1/plugins/kubernetes/clusters/:id    # 获取集群详情
PUT    /api/v1/plugins/kubernetes/clusters/:id    # 更新集群
DELETE /api/v1/plugins/kubernetes/clusters/:id    # 删除集群
```

## 🔧 如何开发自己的插件

### 快速开始

1. **后端插件** (5分钟)
```bash
# 1. 创建插件目录
mkdir -p internal/plugins/myplugin

# 2. 实现 Plugin 接口 (参考 docs/PLUGIN_DEVELOPMENT.md)
# 3. 在 internal/server/http.go 中注册插件

# 完成! 重启后端服务即可使用
```

2. **前端插件** (5分钟)
```bash
# 1. 创建插件目录
mkdir -p web/src/plugins/myplugin
mkdir -p web/src/views/myplugin

# 2. 实现插件类 (参考 docs/PLUGIN_DEVELOPMENT.md)
# 3. 在 web/src/main.ts 中导入插件

# 完成! 重启前端服务即可使用
```

### 详细文档

- **开发指南**: [docs/PLUGIN_DEVELOPMENT.md](./docs/PLUGIN_DEVELOPMENT.md)
- **快速参考**: [docs/PLUGIN_QUICK_START.md](./docs/PLUGIN_QUICK_START.md)

## 📚 插件接口规范

### 后端插件接口

```go
type Plugin interface {
    Name() string              // 插件唯一标识
    Description() string       // 插件描述
    Version() string          // 插件版本
    Author() string           // 插件作者
    Enable(db *gorm.DB) error // 启用插件
    Disable(db *gorm.DB) error // 禁用插件
    RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) // 注册路由
    GetMenus() []MenuConfig   // 获取菜单配置
}
```

### 前端插件接口

```typescript
interface Plugin {
    name: string               // 插件唯一标识
    description: string        // 插件描述
    version: string           // 插件版本
    author: string            // 插件作者
    install(): void           // 安装插件
    uninstall(): void         // 卸载插件
    getMenus?(): MenuConfig[] // 获取菜单配置
    getRoutes?(): RouteConfig[] // 获取路由配置
}
```

## 🎨 插件系统优势

### 1. 模块化开发
每个插件都是独立的模块,可以单独开发、测试和维护

### 2. 按需启用
不需要的功能可以随时禁用,减少系统复杂度和资源占用

### 3. 易于扩展
新功能可以作为插件开发,不需要修改核心代码

### 4. 团队协作
不同团队可以并行开发不同的插件,互不干扰

### 5. 版本管理
每个插件独立版本号,便于升级和维护

## 🔍 查看已安装插件

### 通过 API 查看

```bash
# 获取所有插件
curl http://localhost:9876/api/v1/plugins

# 获取特定插件详情
curl http://localhost:9876/api/v1/plugins/kubernetes

# 获取插件菜单配置
curl http://localhost:9876/api/v1/plugins/kubernetes/menus
```

### 通过前端查看

登录系统后,侧边栏会显示所有已启用插件的菜单

## ⚙️ 禁用/删除插件

### 禁用插件(保留代码)

**后端**:
```go
// internal/server/http.go
// pluginMgr.Register(kubernetes.New())  // 注释掉即可
```

**前端**:
```typescript
// web/src/main.ts
// import '@/plugins/kubernetes'  // 注释掉即可
```

### 完全删除插件

```bash
# 后端
rm -rf internal/plugins/kubernetes

# 前端
rm -rf web/src/plugins/kubernetes
rm -rf web/src/views/kubernetes
```

## 📖 示例插件

系统已包含一个完整的 **Kubernetes 容器管理插件** 作为示例:

- **后端实现**: `internal/plugins/kubernetes/plugin.go`
- **前端实现**: `web/src/plugins/kubernetes/index.ts`
- **页面组件**: `web/src/views/kubernetes/`

你可以参考这个插件的实现来开发自己的插件。

## 🤝 贡献指南

欢迎为 OpsHub 开发新的插件!

1. Fork 本项目
2. 创建插件分支: `git checkout -b feature/my-plugin`
3. 按照插件开发规范实现你的插件
4. 提交代码: `git commit -am 'Add new plugin: xxx'`
5. 推送分支: `git push origin feature/my-plugin`
6. 创建 Pull Request

## 📞 技术支持

- 📖 文档: [docs/PLUGIN_DEVELOPMENT.md](./docs/PLUGIN_DEVELOPMENT.md)
- 🐛 Issue: https://github.com/ydcloud-dy/opshub/issues
- 💬 讨论: https://github.com/ydcloud-dy/opshub/discussions

## 📝 更新日志

### v1.0.0 (2025-12-30)
- ✨ 实现前后端插件化架构
- ✨ 创建 Kubernetes 管理插件示例
- 📚 编写完整的插件开发文档
- 🎨 支持动态菜单和路由注册

---

**OpsHub - 让运维更简单!** 🚀
