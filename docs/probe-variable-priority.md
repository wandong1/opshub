# 拨测类型巡检项变量优先级说明

## 变量优先级体系

拨测类型巡检项实现了**四级变量优先级体系**，确保用户可以在不同层级灵活覆盖变量值。

### 优先级顺序（从高到低）

```
1. 任务调度变量（最高优先级）
   ↓ 覆盖
2. 巡检组变量
   ↓ 覆盖
3. 拨测配置变量
   ↓ 覆盖
4. 全局环境变量
   ↓ 覆盖
5. 系统预置变量（最低优先级）
```

---

## 各级变量详解

### 1. 任务调度变量（最高优先级）

**定义位置**：任务调度页面 - 任务配置

**作用域**：仅该任务执行期间

**使用场景**：
- 临时覆盖变量值
- 针对特定任务的个性化配置
- 测试不同环境的配置

**示例**：
```json
{
  "api_host": "dev.example.com",
  "api_port": "8080",
  "timeout": "5000"
}
```

**前端配置**：
- 页面：`TaskSchedule.vue`
- 字段：`InspectionTask.customVariables`（JSON 字符串）

---

### 2. 巡检组变量

**定义位置**：巡检管理页面 - 巡检组配置

**作用域**：该巡检组下所有巡检项

**使用场景**：
- 环境级配置（开发/测试/生产）
- 业务分组级配置
- 共享配置管理

**示例**：
```json
{
  "api_host": "staging.example.com",
  "db_host": "db-staging.internal",
  "redis_host": "redis-staging.internal"
}
```

**前端配置**：
- 页面：`InspectionManagement.vue`
- 字段：`InspectionGroup.customVariables`（JSON 字符串）

---

### 3. 拨测配置变量

**定义位置**：拨测管理页面 - 变量管理

**作用域**：按业务分组过滤（GroupIDs 字段）

**使用场景**：
- 拨测模板通用配置
- 跨巡检组共享的拨测变量
- 拨测专用变量

**示例**：
```
变量名: mysql_port
变量值: 3306
作用域: 数据库分组

变量名: redis_port
变量值: 6379
作用域: 缓存分组
```

**前端配置**：
- 页面：`ProbeManagement.vue` - 变量管理
- 存储：`ProbeVariable` 表

**注意**：拨测配置变量和全局环境变量都存储在 `ProbeVariable` 表中，通过 `GroupIDs` 字段区分作用域。

---

### 4. 全局环境变量

**定义位置**：拨测管理页面 - 变量管理

**作用域**：全局（GroupIDs 为空）

**使用场景**：
- 系统级全局配置
- 所有巡检项共享的变量
- 默认配置值

**示例**：
```
变量名: default_timeout
变量值: 30
作用域: 全局

变量名: company_domain
变量值: example.com
作用域: 全局
```

**前端配置**：
- 页面：`ProbeManagement.vue` - 变量管理
- 存储：`ProbeVariable` 表（GroupIDs 为空）

---

### 5. 系统预置变量（最低优先级）

**定义位置**：系统自动生成

**作用域**：全局

**变量列表**：

#### 时间相关
- `{{timestamp}}` - Unix 时间戳（秒）
- `{{timestamp_ms}}` - Unix 时间戳（毫秒）
- `{{current_time}}` - 当前时间（HHmmss）
- `{{current_date}}` - 当前日期（yyyyMMdd）
- `{{current_datetime}}` - 当前日期时间（yyyyMMddHHmmss）

#### 随机数相关
- `{{random_number}}` - 10 位随机数
- `{{random_string}}` - 10 位随机字符串
- `{{random_uuid}}` - UUID

#### 巡检专属
- `{{exec_node_ip}}` - 执行主机 IP
- `{{instance}}` - 主机实例地址（IP:端口）

---

## 变量传递流程

### 完整链路

```
任务调度（customVariables）
  ↓
ItemService.executeItem(runtimeVariables)
  ↓
VariableResolver.ResolveVariables(groupID, runtimeVariables, hostIP)
  ├─ 1. 生成预置变量
  ├─ 2. 加载全局环境变量
  ├─ 3. 加载巡检组变量（覆盖全局）
  └─ 4. 合并运行时变量（覆盖所有）
  ↓
variables（合并后的变量 Map）
  ↓
ProbeExecutor.Execute(probeConfigID, timeout, variables)
  ↓
拨测管理 VariableResolver.ResolveConfigWithExtra(cfg, variables)
  ├─ 1. 生成预置变量
  ├─ 2. 加载系统变量（拨测配置变量 + 全局环境变量）
  └─ 3. 合并 extraVars（覆盖所有）
  ↓
解析后的拨测配置（变量已替换）
  ↓
执行拨测
```

### 关键代码位置

**巡检管理侧**：
- `internal/service/inspection_mgmt/variable_resolver.go`
  - `ResolveVariables()` - 合并任务调度变量、巡检组变量、全局变量

**拨测管理侧**：
- `internal/biz/inspection/variable_resolver.go`
  - `ResolveConfigWithExtra()` - 合并 extraVars、拨测配置变量、全局变量

**执行器**：
- `internal/service/inspection_mgmt/probe_executor.go`
  - `Execute()` - 调用拨测管理的变量解析器

---

## 使用示例

### 场景 1：开发环境测试

**配置**：
- 拨测配置：`target = "{{api_host}}"`, `port = "{{api_port}}"`
- 拨测变量：`api_host = "prod.example.com"`, `api_port = "443"`
- 巡检组变量：`api_host = "staging.example.com"`, `api_port = "8443"`
- 任务调度变量：`api_host = "dev.example.com"`, `api_port = "8080"`

**实际请求**：
- 目标：`dev.example.com:8080`
- 原因：任务调度变量优先级最高

---

### 场景 2：生产环境巡检

**配置**：
- 拨测配置：`target = "{{api_host}}"`, `port = "{{api_port}}"`
- 拨测变量：`api_host = "prod.example.com"`, `api_port = "443"`
- 巡检组变量：无
- 任务调度变量：无

**实际请求**：
- 目标：`prod.example.com:443`
- 原因：使用拨测配置变量

---

### 场景 3：多环境共享拨测模板

**拨测配置**（通用模板）：
```
URL: https://{{api_host}}/api/health
Headers: {"Authorization": "Bearer {{api_token}}"}
```

**开发环境巡检组**：
```json
{
  "api_host": "dev.example.com",
  "api_token": "dev-token-123"
}
```

**生产环境巡检组**：
```json
{
  "api_host": "prod.example.com",
  "api_token": "prod-token-456"
}
```

**效果**：
- 同一个拨测配置，在不同巡检组中自动使用对应环境的变量
- 无需为每个环境创建单独的拨测配置

---

## 最佳实践

### 1. 变量命名规范

**推荐格式**：`<资源类型>_<属性>`

**示例**：
- `api_host` - API 主机地址
- `db_port` - 数据库端口
- `redis_password` - Redis 密码
- `timeout_ms` - 超时时间（毫秒）

### 2. 变量分层策略

**全局环境变量**：
- 系统级默认值
- 公司域名、默认端口等

**拨测配置变量**：
- 拨测模板专用变量
- 跨巡检组共享的配置

**巡检组变量**：
- 环境级配置（开发/测试/生产）
- 业务分组级配置

**任务调度变量**：
- 临时测试配置
- 一次性覆盖

### 3. 安全建议

**敏感信息**：
- 使用 `secret` 类型存储密码、Token
- 前端不显示 secret 类型变量的值
- 定期轮换敏感变量

**权限控制**：
- 拨测配置变量通过 GroupIDs 限制作用域
- 任务调度变量仅任务执行期间有效

---

## 故障排查

### 问题 1：变量未生效

**症状**：拨测配置中的变量占位符未被替换

**排查步骤**：
1. 检查变量名是否正确（区分大小写）
2. 检查变量作用域（GroupIDs）是否包含当前巡检组
3. 查看日志：`[ProbeExecutor] Config after variable resolution`
4. 确认变量优先级是否被更高级别覆盖

**解决方法**：
- 使用正确的变量名
- 调整变量作用域
- 在更高优先级层级定义变量

---

### 问题 2：变量被意外覆盖

**症状**：变量值不是预期的值

**排查步骤**：
1. 检查是否有更高优先级的变量定义
2. 查看任务调度变量配置
3. 查看巡检组变量配置
4. 查看日志：`[ItemService] Variables after resolve`

**解决方法**：
- 删除不需要的高优先级变量
- 调整变量优先级策略

---

### 问题 3：变量解析失败

**症状**：日志显示 `Failed to resolve variables`

**排查步骤**：
1. 检查变量 JSON 格式是否正确
2. 检查数据库连接是否正常
3. 查看详细错误日志

**解决方法**：
- 修正 JSON 格式错误
- 检查数据库服务状态

---

## 技术实现细节

### 数据结构

**任务调度变量**：
```go
type InspectionTask struct {
    CustomVariables string `json:"custom_variables"` // JSON 字符串
}
```

**巡检组变量**：
```go
type InspectionGroup struct {
    CustomVariables string `json:"custom_variables"` // JSON 字符串
}
```

**拨测配置变量 / 全局环境变量**：
```go
type ProbeVariable struct {
    Name     string `json:"name"`
    Value    string `json:"value"`
    VarType  string `json:"var_type"`  // plain / secret
    GroupIDs string `json:"group_ids"` // 作用域（逗号分隔），空表示全局
}
```

### 变量替换语法

**格式**：`{{variable_name}}`

**正则**：`\{\{(\w+)\}\}`

**示例**：
```
原始配置: https://{{api_host}}:{{api_port}}/api/health
变量值: api_host=example.com, api_port=8080
替换后: https://example.com:8080/api/health
```

---

## 总结

拨测类型巡检项的四级变量优先级体系提供了灵活的配置管理能力：

1. ✅ **任务调度变量**：临时覆盖，最高优先级
2. ✅ **巡检组变量**：环境级配置，覆盖拨测配置
3. ✅ **拨测配置变量**：模板级配置，覆盖全局变量
4. ✅ **全局环境变量**：系统级默认值
5. ✅ **系统预置变量**：自动生成，最低优先级

**核心优势**：
- 🎯 优先级清晰，符合直觉
- 🔄 灵活覆盖，适应多种场景
- 🛡️ 作用域隔离，安全可控
- 📦 模板复用，提高效率

**使用建议**：
- 全局变量定义默认值
- 拨测配置变量定义模板专用配置
- 巡检组变量定义环境级配置
- 任务调度变量用于临时测试
