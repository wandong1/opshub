# OpsHub 插件开发指南

## 概述

OpsHub 采用**前后端分离的插件化架构**，允许开发者以插件的形式扩展系统功能。每个插件都是独立的、可插拔的模块，可以一键启用或禁用，不影响系统核心功能。

## 插件化架构优势

1. **模块化**: 每个插件独立开发、测试和维护
2. **可插拔**: 可以随时启用或禁用插件，无需重启系统
3. **低耦合**: 插件之间相互独立，互不影响
4. **易扩展**: 新功能可以作为插件开发，不需要修改核心代码
5. **版本管理**: 每个插件独立版本号，便于维护

---

## 后端插件开发

### 1. 插件接口定义

所有后端插件必须实现 `Plugin` 接口：

```go
package plugin

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

type Plugin interface {
    // Name 插件名称(唯一标识)
    Name() string

    // Description 插件描述
    Description() string

    // Version 插件版本
    Version() string

    // Author 插件作者
    Author() string

    // Enable 启用插件
    // 在这里可以初始化插件需要的资源、数据库表等
    Enable(db *gorm.DB) error

    // Disable 禁用插件
    // 在这里可以清理插件资源,注意:默认只清理缓存,不会删除数据库表
    Disable(db *gorm.DB) error

    // RegisterRoutes 注册路由
    // 插件可以在这里注册自己的API路由
    RegisterRoutes(router *gin.RouterGroup, db *gorm.DB)

    // GetMenus 获取插件菜单配置
    // 返回插件需要添加到系统的菜单项
    GetMenus() []MenuConfig
}
```

### 2. 创建后端插件

#### 步骤 1: 创建插件目录

在 `internal/plugins/` 下创建插件目录，例如 `internal/plugins/myplugin/`

#### 步骤 2: 实现插件接口

```go
package myplugin

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
    "github.com/ydcloud-dy/opshub/internal/plugin"
)

// MyPlugin 我的插件
type MyPlugin struct {
    db *gorm.DB
}

// New 创建插件实例
func New() *MyPlugin {
    return &MyPlugin{}
}

// Name 插件名称
func (p *MyPlugin) Name() string {
    return "myplugin"
}

// Description 插件描述
func (p *MyPlugin) Description() string {
    return "我的自定义插件"
}

// Version 插件版本
func (p *MyPlugin) Version() string {
    return "1.0.0"
}

// Author 插件作者
func (p *MyPlugin) Author() string {
    return "Your Name"
}

// Enable 启用插件
func (p *MyPlugin) Enable(db *gorm.DB) error {
    p.db = db

    // 在这里初始化数据库表
    // err := db.AutoMigrate(&MyModel{})
    // if err != nil {
    //     return err
    // }

    return nil
}

// Disable 禁用插件
func (p *MyPlugin) Disable(db *gorm.DB) error {
    // 清理插件资源
    p.db = nil
    return nil
}

// RegisterRoutes 注册路由
func (p *MyPlugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
    // 注册插件的 API 路由
    router.GET("", p.listItems)
    router.POST("", p.createItem)
    router.GET("/:id", p.getItem)
    router.PUT("/:id", p.updateItem)
    router.DELETE("/:id", p.deleteItem)
}

// GetMenus 获取菜单配置
func (p *MyPlugin) GetMenus() []plugin.MenuConfig {
    return []plugin.MenuConfig{
        {
            Name:       "我的插件",
            Path:       "/myplugin",
            Icon:       "Star",
            Sort:       200,
            Hidden:     false,
            ParentPath: "",
        },
        {
            Name:       "功能列表",
            Path:       "/myplugin/items",
            Icon:       "List",
            Sort:       1,
            Hidden:     false,
            ParentPath: "/myplugin",
        },
    }
}

// Handler 方法实现
func (p *MyPlugin) listItems(c *gin.Context) {
    c.JSON(200, gin.H{
        "code":    0,
        "message": "success",
        "data":    []interface{}{},
    })
}

func (p *MyPlugin) createItem(c *gin.Context) {
    // 实现创建逻辑
    c.JSON(200, gin.H{"code": 0, "message": "success"})
}

func (p *MyPlugin) getItem(c *gin.Context) {
    // 实现获取逻辑
    c.JSON(200, gin.H{"code": 0, "message": "success"})
}

func (p *MyPlugin) updateItem(c *gin.Context) {
    // 实现更新逻辑
    c.JSON(200, gin.H{"code": 0, "message": "success"})
}

func (p *MyPlugin) deleteItem(c *gin.Context) {
    // 实现删除逻辑
    c.JSON(200, gin.H{"code": 0, "message": "success"})
}
```

#### 步骤 3: 注册插件

在 `internal/server/http.go` 中注册插件：

```go
import (
    myplugin "github.com/ydcloud-dy/opshub/internal/plugins/myplugin"
)

func NewHTTPServer(conf *conf.Config, svc *service.Service, db *gorm.DB) *HTTPServer {
    // ... 其他代码 ...

    // 创建插件管理器
    pluginMgr := plugin.NewManager(db)

    // 注册插件
    if err := pluginMgr.Register(myplugin.New()); err != nil {
        appLogger.Error("注册MyPlugin插件失败", zap.Error(err))
    }

    // 注册其他插件...

    // ... 其他代码 ...
}
```

### 3. 访问插件 API

插件 API 路由格式：`/api/v1/plugins/{插件名称}/{路由}`

例如：
```
GET  /api/v1/plugins/myplugin
POST /api/v1/plugins/myplugin
GET  /api/v1/plugins/myplugin/123
```

### 4. 管理插件接口

系统提供了插件管理接口：

```
GET /api/v1/plugins                    # 获取所有插件列表
GET /api/v1/plugins/:name              # 获取插件详情
GET /api/v1/plugins/:name/menus        # 获取插件菜单配置
```

---

## 前端插件开发

### 1. 插件类型定义

前端插件必须实现 `Plugin` 接口：

```typescript
export interface Plugin {
  /** 插件名称(唯一标识) */
  name: string

  /** 插件描述 */
  description: string

  /** 插件版本 */
  version: string

  /** 插件作者 */
  author: string

  /** 安装插件 */
  install: () => void | Promise<void>

  /** 卸载插件 */
  uninstall: () => void | Promise<void>

  /** 获取菜单配置 */
  getMenus?: () => PluginMenuConfig[]

  /** 获取路由配置 */
  getRoutes?: () => PluginRouteConfig[]
}
```

### 2. 创建前端插件

#### 步骤 1: 创建插件目录

在 `web/src/plugins/` 下创建插件目录，例如 `web/src/plugins/myplugin/`

#### 步骤 2: 创建插件入口文件

```typescript
// web/src/plugins/myplugin/index.ts
import { Plugin, PluginMenuConfig, PluginRouteConfig } from '../types'
import { pluginManager } from '../manager'

// 懒加载组件
const ItemsList = () => import('@/views/myplugin/ItemsList.vue')
const ItemDetail = () => import('@/views/myplugin/ItemDetail.vue')

/**
 * 我的插件
 */
class MyPlugin implements Plugin {
  name = 'myplugin'
  description = '我的自定义插件'
  version = '1.0.0'
  author = 'Your Name'

  /**
   * 安装插件
   */
  async install() {
    console.log('MyPlugin 插件安装中...')
    // 在这里可以进行一些初始化操作
    // 例如:注册全局指令、混入、全局组件等
  }

  /**
   * 卸载插件
   */
  async uninstall() {
    console.log('MyPlugin 插件卸载中...')
    // 清理插件创建的资源
  }

  /**
   * 获取菜单配置
   */
  getMenus(): PluginMenuConfig[] {
    return [
      {
        name: '我的插件',
        path: '/myplugin',
        icon: 'Star',
        sort: 200,
        hidden: false,
        parentPath: '',
      },
      {
        name: '功能列表',
        path: '/myplugin/items',
        icon: 'List',
        sort: 1,
        hidden: false,
        parentPath: '/myplugin',
      },
    ]
  }

  /**
   * 获取路由配置
   */
  getRoutes(): PluginRouteConfig[] {
    return [
      {
        path: '/myplugin',
        name: 'MyPlugin',
        component: () => import('@/views/myplugin/Index.vue'),
        meta: { title: '我的插件' },
        children: [
          {
            path: 'items',
            name: 'MyPluginItems',
            component: ItemsList,
            meta: { title: '功能列表' },
          },
          {
            path: 'items/:id',
            name: 'MyPluginItemDetail',
            component: ItemDetail,
            meta: { title: '详情' },
          },
        ],
      },
    ]
  }
}

// 创建并注册插件实例
export const myPlugin = new MyPlugin()

// 自动注册到插件管理器
pluginManager.register(myPlugin)

export default myPlugin
```

#### 步骤 3: 创建页面组件

创建对应的 Vue 组件文件：

```vue
<!-- web/src/views/myplugin/Index.vue -->
<template>
  <div class="myplugin-container">
    <router-view />
  </div>
</template>

<script setup lang="ts">
// 我的插件布局容器
</script>
```

```vue
<!-- web/src/views/myplugin/ItemsList.vue -->
<template>
  <div class="items-list-page">
    <el-card>
      <template #header>
        <h2>功能列表</h2>
      </template>
      <div class="content">
        <!-- 页面内容 -->
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

// 组件逻辑
</script>

<style scoped>
.items-list-page {
  padding: 20px;
}
</style>
```

#### 步骤 4: 导入插件

在 `web/src/main.ts` 中导入插件：

```typescript
// 导入插件
import '@/plugins/myplugin'

// ... 其他代码 ...
```

### 3. 创建 API 请求函数

创建 `web/src/api/myplugin.ts`：

```typescript
import request from '@/utils/request'

export interface Item {
  id: number
  name: string
  // ... 其他字段
}

// 获取列表
export const listItems = (params?: any) => {
  return request.get<any, Item[]>('/api/v1/plugins/myplugin', { params })
}

// 创建
export const createItem = (data: any) => {
  return request.post('/api/v1/plugins/myplugin', data)
}

// 获取详情
export const getItem = (id: number) => {
  return request.get<any, Item>(`/api/v1/plugins/myplugin/${id}`)
}

// 更新
export const updateItem = (id: number, data: any) => {
  return request.put(`/api/v1/plugins/myplugin/${id}`, data)
}

// 删除
export const deleteItem = (id: number) => {
  return request.delete(`/api/v1/plugins/myplugin/${id}`)
}
```

---

## 插件开发最佳实践

### 1. 目录结构规范

```
internal/plugins/myplugin/
├── plugin.go           # 插件主文件
├── handler.go          # HTTP 处理器
├── model.go            # 数据模型
├── service.go          # 业务逻辑
└── README.md           # 插件说明文档

web/src/plugins/myplugin/
├── index.ts            # 插件入口
└── README.md           # 插件说明文档

web/src/views/myplugin/
├── Index.vue           # 布局组件
├── ItemsList.vue       # 列表页面
└── ItemDetail.vue      # 详情页面

web/src/api/
└── myplugin.ts         # API 请求
```

### 2. 命名规范

- **插件名称**: 小写字母，例如 `myplugin`, `kubernetes`
- **数据库表**: `plugin_{插件名}_{表名}`, 例如 `plugin_myplugin_items`
- **API 路由**: `/api/v1/plugins/{插件名}/{资源}`
- **前端路由**: `/{插件名}/{页面}`

### 3. 数据库表设计

建议为插件的数据库表添加统一前缀，便于管理和识别：

```go
// 表名示例
type Cluster struct {
    gorm.Model
    Name string `gorm:"type:varchar(100);not null"`
    // ... 其他字段
}

// 指定表名
func (Cluster) TableName() string {
    return "plugin_kubernetes_clusters"
}
```

### 4. 错误处理

插件中应该统一使用系统的错误处理：

```go
import "github.com/ydcloud-dy/opshub/pkg/response"

func (p *MyPlugin) getItem(c *gin.Context) {
    id := c.Param("id")

    item, err := p.service.GetItem(id)
    if err != nil {
        response.Error(c, err.Error())
        return
    }

    response.Success(c, item)
}
```

### 5. 权限控制

插件可以集成 RBAC 权限系统：

```go
// 在路由中添加权限中间件
func (p *MyPlugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
    // 需要认证的路由
    auth := router.Group("")
    auth.Use(middleware.Auth())
    {
        auth.GET("", p.listItems)
        auth.POST("", p.createItem)
    }

    // 需要特定权限的路由
    admin := router.Group("")
    admin.Use(middleware.Auth(), middleware.RequiredPermission("myplugin.admin"))
    {
        admin.DELETE("/:id", p.deleteItem)
    }
}
```

### 6. 日志记录

使用系统的日志记录器：

```go
import appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
import "go.uber.org/zap"

func (p *MyPlugin) Enable(db *gorm.DB) error {
    appLogger.Info("插件启用中", zap.String("plugin", p.Name()))
    // ... 初始化逻辑
    return nil
}
```

---

## 禁用/删除插件

### 后端

1. **禁用插件**: 注释掉 `internal/server/http.go` 中的插件注册代码
2. **删除插件**: 删除插件目录

### 前端

1. **禁用插件**: 注释掉 `web/src/main.ts` 中的插件导入
2. **删除插件**: 删除插件目录

---

## 示例插件

系统已包含一个完整的 Kubernetes 管理插件示例：

- **后端**: `internal/plugins/kubernetes/`
- **前端**: `web/src/plugins/kubernetes/`
- **页面**: `web/src/views/kubernetes/`

你可以参考这个插件的实现来开发自己的插件。

---

## 常见问题

### Q1: 插件之间如何通信?

A: 插件之间应该保持独立，如需通信建议通过：
- 数据库共享表
- HTTP API 调用
- 事件系统（待实现）

### Q2: 插件如何使用系统的公共功能?

A: 可以直接导入系统的包：
```go
import "github.com/ydcloud-dy/opshub/pkg/response"
import appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
```

### Q3: 插件的数据库表会自动删除吗?

A: 不会。插件的 `Disable` 方法默认不删除数据库表，以保护数据。如需删除，可以手动执行 SQL。

---

## 技术支持

如有问题，请联系：
- Issue: https://github.com/ydcloud-dy/opshub/issues
- 文档: https://github.com/ydcloud-dy/opshub/wiki
