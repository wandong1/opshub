# OpsHub RBAC权限管理系统

这是一个完整的基于RBAC（Role-Based Access Control）的权限管理系统，支持用户、角色、部门、菜单的完整管理功能。

## 技术栈

### 后端
- Go 1.25
- Gin Web框架
- GORM ORM框架
- MySQL数据库
- JWT身份认证
- Cobra CLI框架
- Viper配置管理

### 前端
- Vue 3
- TypeScript
- Vite
- Element Plus UI组件库
- Vue Router路由管理
- Pinia状态管理
- Axios HTTP客户端

## 功能特性

### 用户管理
- 用户增删改查
- 用户注册/登录
- 密码加密存储
- 用户角色分配
- 用户部门关联

### 角色管理
- 角色增删改查
- 角色权限分配
- 角色菜单关联

### 部门管理
- 部门树形结构
- 部门增删改查
- 部门层级管理

### 菜单管理
- 菜单树形结构
- 菜单类型：目录/菜单/按钮
- 路由配置
- 图标配置
- 排序管理

## 项目结构

```
opshub/
├── bin/                    # 编译后的二进制文件
├── cmd/                    # 命令行工具
│   ├── config/            # 配置命令
│   ├── root/              # 根命令
│   ├── server/            # 服务器命令
│   └── version/           # 版本命令
├── config/                # 配置文件
│   └── config.yaml        # 主配置文件
├── internal/              # 内部代码
│   ├── biz/              # 业务层
│   │   └── rbac/         # RBAC业务逻辑
│   ├── conf/             # 配置管理
│   ├── data/             # 数据层
│   │   └── rbac/         # RBAC数据访问
│   ├── server/           # 服务器
│   │   └── rbac/         # RBAC HTTP服务
│   └── service/          # 服务层
│       └── rbac/         # RBAC服务
├── pkg/                   # 公共包
│   ├── error/            # 错误处理
│   ├── logger/           # 日志
│   ├── middleware/       # 中间件
│   └── response/         # 响应处理
├── web/                   # 前端项目
│   ├── src/
│   │   ├── api/         # API接口
│   │   ├── router/      # 路由
│   │   ├── stores/      # 状态管理
│   │   ├── utils/       # 工具类
│   │   └── views/       # 页面
│   └── ...
├── docs/                  # Swagger文档
├── main.go               # 程序入口
├── go.mod
└── go.sum
```

## 快速开始

### 1. 数据库配置

创建MySQL数据库：
```sql
CREATE DATABASE opshub CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

修改 `config/config.yaml` 中的数据库配置：
```yaml
database:
  driver: mysql
  host: 127.0.0.1
  port: 3306
  database: opshub
  username: root
  password: "your_password"
```

### 2. 启动后端服务

```bash
# 安装依赖
go mod tidy

# 编译
go build -o bin/opshub main.go

# 运行
./bin/opshub server
```

后端服务将在 `http://localhost:9876` 启动。

### 3. 启动前端服务

```bash
# 进入前端目录
cd web

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

前端服务将在 `http://localhost:5173` 启动。

### 4. 访问系统

打开浏览器访问 `http://localhost:5173`，默认无用户，需要先注册。

## API接口

### 公开接口
- `POST /api/v1/public/login` - 用户登录
- `POST /api/v1/public/register` - 用户注册

### 需要认证的接口

#### 用户管理
- `GET /api/v1/users` - 用户列表
- `GET /api/v1/users/:id` - 获取用户详情
- `POST /api/v1/users` - 创建用户
- `PUT /api/v1/users/:id` - 更新用户
- `DELETE /api/v1/users/:id` - 删除用户
- `POST /api/v1/users/:id/roles` - 分配用户角色

#### 角色管理
- `GET /api/v1/roles` - 角色列表
- `GET /api/v1/roles/all` - 所有角色
- `GET /api/v1/roles/:id` - 获取角色详情
- `POST /api/v1/roles` - 创建角色
- `PUT /api/v1/roles/:id` - 更新角色
- `DELETE /api/v1/roles/:id` - 删除角色
- `POST /api/v1/roles/:id/menus` - 分配角色菜单

#### 部门管理
- `GET /api/v1/departments/tree` - 部门树
- `GET /api/v1/departments/:id` - 获取部门详情
- `POST /api/v1/departments` - 创建部门
- `PUT /api/v1/departments/:id` - 更新部门
- `DELETE /api/v1/departments/:id` - 删除部门

#### 菜单管理
- `GET /api/v1/menus/tree` - 菜单树
- `GET /api/v1/menus/user` - 当前用户菜单
- `GET /api/v1/menus/:id` - 获取菜单详情
- `POST /api/v1/menus` - 创建菜单
- `PUT /api/v1/menus/:id` - 更新菜单
- `DELETE /api/v1/menus/:id` - 删除菜单

## 数据库表结构

系统启动时会自动创建以下表：

- `sys_user` - 用户表
- `sys_role` - 角色表
- `sys_department` - 部门表
- `sys_menu` - 菜单表
- `sys_user_role` - 用户角色关联表
- `sys_role_menu` - 角色菜单关联表

## 配置说明

配置文件位于 `config/config.yaml`：

```yaml
server:
  mode: debug              # 运行模式: debug/release/test
  http_port: 9876          # HTTP端口
  rpc_port: 9090           # RPC端口
  jwt_secret: "your-secret-key-change-in-production"  # JWT密钥

database:
  driver: mysql
  host: 127.0.0.1
  port: 3306
  database: opshub
  username: root
  password: "123456"
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 3600

redis:
  host: 127.0.0.1
  port: 6379
  password: ""
  db: 0

log:
  level: info              # 日志级别: debug/info/warn/error
  filename: logs/app.log
  max_size: 100            # MB
  max_backups: 10
  max_age: 30              # days
  compress: true
  console: true
```

## 开发说明

### 后端开发

```bash
# 运行开发服务器
go run main.go server

# 生成Swagger文档
swag init -g main.go -o docs

# 运行测试
go test ./...
```

### 前端开发

```bash
cd web

# 开发模式
npm run dev

# 构建生产版本
npm run build

# 预览生产构建
npm run preview
```

## 许可证

MIT License
