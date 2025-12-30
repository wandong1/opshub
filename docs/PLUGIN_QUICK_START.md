# OpsHub 插件系统快速参考

## 一、插件系统架构

```
┌─────────────────────────────────────────────────────────┐
│                     OpsHub 核心系统                       │
├─────────────────────────────────────────────────────────┤
│  后端插件管理器 (internal/plugin/plugin.go)              │
│  ├─ Register()      注册插件                             │
│  ├─ Enable()        启用插件                             │
│  ├─ Disable()       禁用插件                             │
│  └─ GetMenus()      获取插件菜单                         │
├─────────────────────────────────────────────────────────┤
│  前端插件管理器 (web/src/plugins/manager.ts)             │
│  ├─ register()      注册插件                             │
│  ├─ install()       安装插件                             │
│  ├─ uninstall()     卸载插件                             │
│  └─ getRoutes()     获取插件路由                         │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────────┐
        │         插件生态系统                  │
        ├──────────────────────────────────────┤
        │  Kubernetes 容器管理插件              │
        │  监控告警插件 (示例)                  │
        │  日志分析插件 (示例)                  │
        │  ... 你的插件                        │
        └──────────────────────────────────────┘
```

## 二、后端插件开发清单

### 1. 创建插件结构

```bash
mkdir -p internal/plugins/myplugin
cd internal/plugins/myplugin
```

### 2. 实现插件接口 (必需)

```go
package myplugin

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
    "github.com/ydcloud-dy/opshub/internal/plugin"
)

type MyPlugin struct{ db *gorm.DB }

// 必需实现的接口方法
func (p *MyPlugin) Name() string        { return "myplugin" }
func (p *MyPlugin) Description() string { return "插件描述" }
func (p *MyPlugin) Version() string     { return "1.0.0" }
func (p *MyPlugin) Author() string      { return "作者" }
func (p *MyPlugin) Enable(db *gorm.DB) error {
    p.db = db
    // db.AutoMigrate(&MyModel{})  // 创建数据库表
    return nil
}
func (p *MyPlugin) Disable(db *gorm.DB) error {
    p.db = nil  // 清理资源
    return nil
}
func (p *MyPlugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
    router.GET("", p.listHandler)
    router.POST("", p.createHandler)
}
func (p *MyPlugin) GetMenus() []plugin.MenuConfig {
    return []plugin.MenuConfig{
        {Name: "插件名", Path: "/myplugin", Icon: "Star", Sort: 100, Hidden: false, ParentPath: ""},
    }
}

// Handler 实现
func (p *MyPlugin) listHandler(c *gin.Context) {
    c.JSON(200, gin.H{"code": 0, "data": []interface{}{}})
}
func (p *MyPlugin) createHandler(c *gin.Context) {
    c.JSON(200, gin.H{"code": 0, "message": "success"})
}
```

### 3. 注册插件

在 `internal/server/http.go` 中添加：

```go
import myplugin "github.com/ydcloud-dy/opshub/internal/plugins/myplugin"

// 在 NewHTTPServer 函数中
pluginMgr.Register(myplugin.New())
```

### 4. 访问插件 API

```
GET    /api/v1/plugins/myplugin
POST   /api/v1/plugins/myplugin
GET    /api/v1/plugins/myplugin/:id
PUT    /api/v1/plugins/myplugin/:id
DELETE /api/v1/plugins/myplugin/:id
```

## 三、前端插件开发清单

### 1. 创建插件结构

```bash
mkdir -p web/src/plugins/myplugin
mkdir -p web/src/views/myplugin
```

### 2. 实现插件类 (必需)

```typescript
// web/src/plugins/myplugin/index.ts
import { Plugin } from '../types'
import { pluginManager } from '../manager'

class MyPlugin implements Plugin {
  name = 'myplugin'
  description = '插件描述'
  version = '1.0.0'
  author = '作者'

  async install() {
    console.log('插件安装')
  }

  async uninstall() {
    console.log('插件卸载')
  }

  getMenus() {
    return [
      {name: '插件名', path: '/myplugin', icon: 'Star', sort: 100, hidden: false, parentPath: ''}
    ]
  }

  getRoutes() {
    return [
      {
        path: '/myplugin',
        name: 'MyPlugin',
        component: () => import('@/views/myplugin/Index.vue'),
        children: [
          {
            path: 'items',
            name: 'MyPluginItems',
            component: () => import('@/views/myplugin/Items.vue'),
            meta: { title: '列表' }
          }
        ]
      }
    ]
  }
}

export const myPlugin = new MyPlugin()
pluginManager.register(myPlugin)
```

### 3. 导入插件

在 `web/src/main.ts` 中添加：

```typescript
import '@/plugins/myplugin'
```

### 4. 创建页面组件

```vue
<!-- web/src/views/myplugin/Index.vue -->
<template>
  <router-view />
</template>

<!-- web/src/views/myplugin/Items.vue -->
<template>
  <el-card>
    <template #header><h2>插件页面</h2></template>
    <div>页面内容</div>
  </el-card>
</template>
```

## 四、如何禁用/删除插件

### 后端插件

**禁用插件**（注释掉注册代码）:
```go
// internal/server/http.go
// pluginMgr.Register(myplugin.New())  // 注释这一行
```

**删除插件**:
```bash
rm -rf internal/plugins/myplugin
```

### 前端插件

**禁用插件**（注释掉导入代码）:
```typescript
// web/src/main.ts
// import '@/plugins/myplugin'  // 注释这一行
```

**删除插件**:
```bash
rm -rf web/src/plugins/myplugin
rm -rf web/src/views/myplugin
```

## 五、Kubernetes 插件目录结构

### 已实现的 Kubernetes 插件结构

```
internal/plugins/kubernetes/
└── plugin.go                 # 插件实现，包含10个子功能的路由

web/src/plugins/kubernetes/
└── index.ts                  # 插件定义

web/src/views/kubernetes/
├── Index.vue                 # 布局容器
├── Clusters.vue              # 集群管理
├── Nodes.vue                 # 节点管理
├── Workloads.vue             # 工作负载
├── Namespaces.vue            # 命名空间
├── Network.vue               # 网络管理
├── Config.vue                # 配置管理
├── Storage.vue               # 存储管理
├── Access.vue                # 访问控制
├── Audit.vue                 # 终端审计
└── Diagnostic.vue            # 应用诊断
```

### Kubernetes 插件菜单结构

```
Kubernetes管理 (/kubernetes)
├── 集群管理      (/kubernetes/clusters)
├── 节点管理      (/kubernetes/nodes)
├── 工作负载      (/kubernetes/workloads)
├── 命名空间      (/kubernetes/namespaces)
├── 网络管理      (/kubernetes/network)
├── 配置管理      (/kubernetes/config)
├── 存储管理      (/kubernetes/storage)
├── 访问控制      (/kubernetes/access)
├── 终端审计      (/kubernetes/audit)
└── 应用诊断      (/kubernetes/diagnostic)
```

## 六、插件开发规范

### 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| 插件名称 | 小写，单词连写 | `myplugin`, `kubernetes` |
| 数据库表 | `plugin_{插件名}_{表名}` | `plugin_myplugin_items` |
| API路由 | `/api/v1/plugins/{插件名}/...` | `/api/v1/plugins/myplugin` |
| 前端路由 | `/{插件名}/{页面}` | `/myplugin/items` |

### 目录规范

```
internal/plugins/{plugin_name}/    # 后端插件
├── plugin.go                       # 插件主文件
├── handler.go                      # HTTP处理器
├── service.go                      # 业务逻辑
├── model.go                        # 数据模型
└── README.md                       # 说明文档

web/src/plugins/{plugin_name}/      # 前端插件
├── index.ts                        # 插件入口
└── README.md                       # 说明文档

web/src/views/{plugin_name}/        # 页面组件
├── Index.vue                       # 布局
├── List.vue                        # 列表
└── Detail.vue                      # 详情

web/src/api/                       # API定义
└── {plugin_name}.ts               # 插件API
```

## 七、快速测试

### 测试后端插件

```bash
# 1. 启动后端服务
go run cmd/server/main.go

# 2. 测试插件API
curl http://localhost:9876/api/v1/plugins
curl http://localhost:9876/api/v1/plugins/kubernetes
curl http://localhost:9876/api/v1/plugins/kubernetes/menus
```

### 测试前端插件

```bash
# 1. 启动前端服务
cd web && npm run dev

# 2. 访问页面
# 打开浏览器访问 http://localhost:5173
# 查看侧边栏菜单，应该能看到 "Kubernetes管理" 菜单项
```

## 八、常见问题

**Q: 如何查看已安装的插件?**
```bash
curl http://localhost:9876/api/v1/plugins
```

**Q: 插件菜单不显示?**
- 检查插件是否正确注册
- 检查 `GetMenus()` 是否返回了菜单配置
- 前端检查路由是否正确注册

**Q: 插件路由404?**
- 后端:检查 `RegisterRoutes()` 是否正确实现
- 前端:检查 `getRoutes()` 和路由注册

**Q: 如何调试插件?**
- 后端:使用 `zap.L().Debug()` 打日志
- 前端:使用 `console.log()` 和 Vue DevTools

---

详细文档请参考: [PLUGIN_DEVELOPMENT.md](./PLUGIN_DEVELOPMENT.md)
