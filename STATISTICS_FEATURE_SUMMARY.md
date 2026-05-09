# 主机统计功能实现总结

## 功能概述

实现了主机管理页面的资产统计模块，支持根据筛选条件动态更新统计数据，并在主机表格中新增GPU列展示。

## 实现内容

### 1. 后端实现

#### 1.1 数据模型扩展
**文件**: `internal/biz/asset/host.go`

- 在 `HostStatistics` 结构体中添加了SSH和Agent统计字段：
  ```go
  SSHCount         int64 `json:"sshCount"`
  AgentCount       int64 `json:"agentCount"`
  AgentOnlineCount int64 `json:"agentOnlineCount"`
  AgentOfflineCount int64 `json:"agentOfflineCount"`
  ```

#### 1.2 Repository层
**文件**: `internal/data/asset/host.go`

- 修改 `GetStatistics` 方法，支持筛选参数：
  - `keyword`: 关键词搜索
  - `groupIDs`: 分组ID列表
  - `status`: 主机状态
  - `tags`: 标签列表

- 使用 `buildBaseQuery` 函数为每个统计查询创建独立的查询对象，避免WHERE条件累积

- 新增统计项：
  - SSH主机数量
  - Agent主机总数
  - Agent在线数量
  - Agent离线数量

#### 1.3 UseCase层
**文件**: `internal/biz/asset/host_usecase.go`

- 更新 `GetStatistics` 方法签名，接收筛选参数

#### 1.4 Service层
**文件**: `internal/service/asset/host.go`

- 修改 `GetHostStatistics` API接口，从查询参数中获取筛选条件：
  - `keyword`: 关键词
  - `groupId`: 分组ID
  - `status`: 状态
  - `tags`: 标签（逗号分隔）

### 2. 前端实现

#### 2.1 API接口
**文件**: `web/src/api/host.ts`

- 修改 `getHostStatistics` 函数，支持传递筛选参数

#### 2.2 主机管理页面
**文件**: `web/src/views/asset/Hosts.vue`

##### 统计数据动态更新
- 修改 `loadStatistics` 方法，根据当前筛选条件构建请求参数
- 在以下操作时触发统计数据刷新：
  - 搜索 (`handleSearch`)
  - 重置 (`handleReset`)
  - 选择分组 (`handleGroupSelect`)
  - 清除分组 (`clearGroupSelection`)

##### 统计卡片优化
- 主机总数卡片新增显示：
  - SSH主机数量
  - Agent主机数量
  - Agent在线/离线数量（当有Agent主机时显示）

##### GPU列展示
- 在主机表格中添加GPU列（位于磁盘列之后）
- 显示内容：
  - GPU卡数（带闪电图标）
  - GPU型号
  - GPU显存总量
- 样式特点：
  - 紫色主题色（#722ed1）
  - 垂直布局，信息紧凑
  - 无GPU时显示"-"

#### 2.3 样式优化
- 新增 `.gpu-cell`、`.gpu-info`、`.gpu-count`、`.gpu-text`、`.gpu-model`、`.gpu-memory` 样式类
- 统计卡片中的badge样式支持多行显示

## 技术要点

### 1. GORM查询优化
使用闭包函数 `buildBaseQuery` 为每个统计查询创建独立的查询对象，避免WHERE条件在多次查询中累积导致结果错误。

```go
buildBaseQuery := func() *gorm.DB {
    query := r.db.WithContext(ctx).Model(&asset.Host{})
    // 应用筛选条件
    return query
}
```

### 2. 前端响应式更新
通过在筛选操作的回调函数中调用 `loadStatistics()`，实现统计数据与列表数据的同步更新。

### 3. 参数传递
- 后端：使用查询参数接收筛选条件
- 前端：根据当前状态动态构建请求参数对象

## 测试验证

### 后端验证
1. 启动服务后访问 `/api/v1/hosts/statistics` 接口
2. 测试不同筛选参数组合：
   - `?keyword=test`
   - `?groupId=1`
   - `?status=1`
   - `?tags=mysql,redis`

### 前端验证
1. 访问主机管理页面，检查统计模块显示
2. 测试筛选条件变化时统计数据是否同步更新：
   - 输入关键词搜索
   - 选择不同分组
   - 筛选主机状态
   - 选择服务标签
3. 检查GPU列是否正确显示GPU信息

## 相关文件清单

### 后端
- `internal/biz/asset/host.go` - 数据模型
- `internal/data/asset/host.go` - 数据访问层
- `internal/biz/asset/host_usecase.go` - 业务逻辑层
- `internal/service/asset/host.go` - API服务层

### 前端
- `web/src/api/host.ts` - API接口定义
- `web/src/views/asset/Hosts.vue` - 主机管理页面

## 后续优化建议

1. **性能优化**：对于大量主机的场景，可以考虑使用缓存机制
2. **实时更新**：可以通过WebSocket实现统计数据的实时推送
3. **更多维度**：可以添加更多统计维度，如按云厂商、按操作系统等
4. **图表展示**：可以使用ECharts等图表库将统计数据可视化
