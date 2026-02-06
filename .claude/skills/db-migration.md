# Skill: 数据库表结构变更

当新增功能涉及数据库表结构变更（新增表、新增字段、修改字段）时，必须将 AutoMigrate 调用添加到应用启动初始化流程中，确保应用启动时自动同步表结构。

## 自动迁移的整体流程

应用启动时，数据库迁移按以下顺序执行：

```
runServer() (cmd/server/server.go)
  ├── 1. autoMigrate(db)                    → 核心系统表
  ├── 2. NewHTTPServer() → enablePlugins()  → 各插件表（通过 plugin.Enable()）
  └── 3. registerRoutes() → NewIdentityServices() → 身份认证表
```

## 变更位置说明

### 1. 核心系统表 — `cmd/server/server.go` 的 `autoMigrate()` 函数

适用于：RBAC、审计、系统配置等核心模块的 model。

在 `autoMigrate()` 函数的 `db.AutoMigrate(...)` 调用中追加新的 model 指针：

```go
func autoMigrate(db *gorm.DB) error {
    if err := db.AutoMigrate(
        // ... 已有的 model ...
        &yourpackage.YourNewModel{},  // ← 在这里追加
    ); err != nil {
        return err
    }
    // ...
}
```

对应的 model 定义放在 `internal/biz/<模块>/` 目录下。

### 2. 插件表 — 各插件的 `Enable()` 方法

适用于：插件自有的 model。

每个插件在 `plugins/<插件名>/plugin.go` 的 `Enable(db *gorm.DB) error` 方法中管理自己的表迁移：

```go
func (p *Plugin) Enable(db *gorm.DB) error {
    p.db = db
    models := []interface{}{
        // ... 已有的 model ...
        &model.YourNewModel{},  // ← 在这里追加
    }
    for _, m := range models {
        if err := db.AutoMigrate(m); err != nil {
            return err
        }
    }
    return nil
}
```

对应的 model 定义放在 `plugins/<插件名>/model/` 目录下。

现有插件的 Enable 位置：
- Kubernetes: `plugins/kubernetes/plugin.go`
- Task: `plugins/task/plugin.go`
- Monitor: `plugins/monitor/plugin.go`
- Nginx: `plugins/nginx/plugin.go`
- SSL-Cert: `plugins/ssl-cert/plugin.go`

### 3. 身份认证表 — `internal/server/identity/http.go` 的 `NewIdentityServices()`

适用于：身份认证模块（Identity）的 model。

在 `NewIdentityServices()` 函数的 `db.AutoMigrate(...)` 调用中追加：

```go
func NewIdentityServices(db *gorm.DB) (*HTTPServer, error) {
    if err := db.AutoMigrate(
        // ... 已有的 model ...
        &bizIdentity.YourNewModel{},  // ← 在这里追加
    ); err != nil {
        return nil, err
    }
    // ...
}
```

## 操作步骤清单

当涉及数据库表结构变更时，按以下步骤操作：

1. **定义 model struct**：在对应目录下创建或修改 GORM model（含 `gorm` tag）
2. **注册 AutoMigrate**：根据 model 所属模块，在上述 3 个位置之一追加 AutoMigrate 注册
3. **如果需要 import 新包**：在对应文件的 import 块中添加引用
4. **如果需要自定义索引或虚拟列**：参考 `cmd/server/server.go` 中 `autoMigrate()` 函数末尾的 `sys_user` 索引创建方式，在 AutoMigrate 之后用 `db.Exec()` 执行 DDL

## 注意事项

- GORM AutoMigrate 只会新增表/字段，不会删除已有字段或修改字段类型
- 如需删除字段或修改字段类型，需要手动编写 `db.Exec()` 执行 ALTER TABLE
- 数据库配置中已设置 `DisableForeignKeyConstraintWhenMigrating: true`，迁移时不会创建外键约束
- 新增插件时需同时实现 `Plugin` 接口并在 `internal/server/http.go` 的 `NewHTTPServer()` 中注册
