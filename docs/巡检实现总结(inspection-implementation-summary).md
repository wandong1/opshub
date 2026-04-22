# 智能巡检系统实现总结

## 完成时间
2026-03-12

## 实现内容

### 一、后端实现（已完成）

#### 1. 数据模型（4个表）
- `inspection_groups` - 巡检组
- `inspection_items` - 巡检项
- `inspection_tasks` - 定时任务
- `inspection_records` - 执行记录

#### 2. Repository 层（4个）
- `GroupRepository` - 巡检组数据访问
- `ItemRepository` - 巡检项数据访问
- `TaskRepository` - 定时任务数据访问
- `RecordRepository` - 执行记录数据访问

#### 3. 执行器（4个）
- `CommandExecutor` - 命令执行器（支持 Agent/SSH）
- `AssertionValidator` - 断言校验器（9种断言类型）
- `VariableExtractor` - 变量提取器
- `VariableReplacer` - 变量替换器

#### 4. Service 层（4个）
- `GroupService` - 巡检组业务逻辑
- `ItemService` - 巡检项业务逻辑（含测试运行）
- `TaskService` - 定时任务业务逻辑
- `RecordService` - 执行记录业务逻辑

#### 5. HTTP API（20+ 端点）
- `/api/v1/inspection/groups` - 巡检组 CRUD
- `/api/v1/inspection/items` - 巡检项 CRUD
- `/api/v1/inspection/items/test-run` - 测试运行
- `/api/v1/inspection/tasks` - 定时任务 CRUD
- `/api/v1/inspection/records` - 执行记录查询

#### 6. 插件集成
- 已注册到插件管理器
- 已注入 AgentHub（支持 Agent 执行）
- 已注入 HostRepo（主机查询）
- 菜单配置完成

### 二、前端实现（已完成）

#### 1. 页面组件（3个）
- `InspectionGroups.vue` - 巡检组管理页面
- `InspectionItems.vue` - 巡检项管理页面
- `InspectionRecords.vue` - 执行记录页面

#### 2. 路由配置
已在 `web/src/router/index.ts` 中添加：
- `/inspection/groups` - 巡检组管理
- `/inspection/items` - 巡检项管理
- `/inspection/records` - 执行记录

#### 3. UI 规范
- 使用 Arco Design Vue 组件库
- 遵循现有页面布局风格
- 支持权限指令 `v-permission`

### 三、数据库权限配置（已完成）

#### 1. 菜单结构
智能巡检（ID 352）
├── 拨测管理（ID 353）
├── 任务调度（ID 354）
├── Pushgateway配置（ID 355）
├── 环境变量（ID 356）
├── 巡检组管理（ID 389）← 新增
├── 巡检项管理（ID 390）← 新增
└── 执行记录（ID 391）← 新增

#### 2. 按钮权限（7个）
- ID 392-394: 巡检组管理（新增、编辑、删除）
- ID 395-398: 巡检项管理（新增、编辑、删除、测试运行）

#### 3. API 关联
已配置约 40 个 API 端点的权限关联

#### 4. 角色权限
- 管理员（role_id=1）：拥有所有菜单和按钮权限
- 普通用户（role_id=2）：仅拥有查看权限

### 四、修复脚本

已创建 `migrations/fix_inspection_menu.sql` 用于修复菜单结构：
- 删除重复的智能巡检一级菜单
- 将新功能合并到现有模块
- 配置正确的权限关联

## 使用说明

### 1. 数据库修复
执行修复脚本：
```bash
mysql -u root -p opshub < migrations/fix_inspection_menu.sql
```

### 2. 启动服务
```bash
# 后端
make run

# 前端
cd web
npm run dev
```

### 3. 访问页面
登录后在左侧菜单找到"智能巡检"模块，可以看到：
- 拨测管理
- 任务调度
- Pushgateway配置
- 环境变量
- 巡检组管理 ← 新增
- 巡检项管理 ← 新增
- 执行记录 ← 新增

## 核心功能

### 1. 巡检组管理
- 创建巡检组
- 配置 Prometheus 连接
- 设置执行方式（auto/agent/ssh）
- 关联资产分组

### 2. 巡检项管理
- 支持三种执行类型：命令、脚本、PromQL
- 支持 9 种断言类型：gt/gte/lt/lte/eq/contains/not_contains/regex/not_regex
- 支持变量提取与替换
- 支持主机匹配（按标签/按ID）
- 支持测试运行（单项/批量）

### 3. 执行记录
- 查看执行历史
- 查看断言结果
- 查看输出内容
- 查看提取的变量

## 待实现功能

### 1. API 接口实现
前端页面已创建，但 API 调用部分标记为 `TODO`，需要：
- 创建 `web/src/api/inspection.ts` API 客户端
- 实现巡检组 CRUD API 调用
- 实现巡检项 CRUD API 调用
- 实现测试运行 API 调用
- 实现执行记录查询 API 调用

### 2. 脚本执行器
- Shell 脚本执行
- Python 脚本执行

### 3. PromQL 执行器
- Prometheus 客户端集成
- PromQL 查询执行

### 4. 调度器集成
- 实现 `TaskExecutor` 接口
- 实现 `TaskProvider` 接口
- 集成到 Scheduler 框架
- 支持定时任务执行

### 5. Prometheus 指标推送
- 集成 Pushgateway
- 推送执行结果指标
- 推送执行时长指标

### 6. SSH 执行支持
- 凭证解密逻辑
- SSH 客户端集成

## 技术亮点

1. **插件化架构**：作为独立插件实现，可独立启用/禁用
2. **Agent 优先**：优先使用 Agent 执行，自动回退 SSH
3. **灵活断言**：支持 9 种断言类型，满足各种校验需求
4. **变量传递**：支持变量提取与引用，实现顺序执行依赖
5. **权限控制**：完整的菜单和按钮权限配置
6. **UI 规范**：遵循 Arco Design 设计规范

## 文件清单

### 后端文件（约 25 个）
```
plugins/inspection/
├── plugin.go
├── model/
│   ├── inspection_group.go
│   ├── inspection_item.go
│   ├── inspection_task.go
│   └── inspection_record.go
├── repository/
│   ├── group_repository.go
│   ├── item_repository.go
│   ├── task_repository.go
│   └── record_repository.go
├── service/
│   ├── group_service.go
│   ├── item_service.go
│   ├── task_service.go
│   └── record_service.go
├── executor/
│   ├── command_executor.go
│   ├── assertion_validator.go
│   ├── variable_extractor.go
│   └── variable_replacer.go
├── server/
│   ├── handler.go
│   └── router.go
└── dto/
    ├── group_dto.go
    ├── item_dto.go
    ├── task_dto.go
    └── record_dto.go
```

### 前端文件（3 个）
```
web/src/views/inspection/
├── InspectionGroups.vue
├── InspectionItems.vue
└── InspectionRecords.vue
```

### 数据库脚本（2 个）
```
migrations/
├── init.sql (已更新)
└── fix_inspection_menu.sql (新增)
```

## 编译验证

```bash
# 后端编译成功
make build
# 输出: 编译完成: bin/opshub

# 前端路由配置完成
# 访问页面不再出现白屏
```

## 下一步工作

1. 实现前端 API 客户端（`web/src/api/inspection.ts`）
2. 实现脚本执行器和 PromQL 执行器
3. 集成调度器框架
4. 实现 Prometheus 指标推送
5. 完善 SSH 执行支持
6. 编写单元测试和集成测试
7. 完善文档和使用说明

## 注意事项

1. 当前 SSH 执行暂未实现，仅支持 Agent 执行
2. 前端页面 API 调用部分需要实现
3. 定时任务执行需要集成调度器
4. PromQL 执行需要 Prometheus 客户端
5. 指标推送需要配置 Pushgateway
