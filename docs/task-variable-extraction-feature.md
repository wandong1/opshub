# 任务调度变量提取功能 - 使用文档

## 功能概述

在任务调度页面新增**"从配置中提取变量"**功能，用户选择巡检组/拨测项后，可一键提取所有可配置变量并自动填充到自定义变量列表，无需手动查看原配置即可快速完成变量赋值。

## 功能特性

### 1. 智能提取
- **自动识别变量模式**：识别 `{{variable_name}}` 格式的变量引用
- **多源提取**：
  - 拨测任务：从 target、URL、headers、body、workflow steps 中提取
  - 巡检任务：从 command、script、PromQL 查询中提取
  - **拨测类型巡检项**：自动提取关联的拨测配置中的变量（递归提取）
- **智能过滤**：自动过滤系统预置变量（11 种）

### 2. 智能合并
- **保留已有变量**：不会覆盖用户已填写的变量值
- **只添加新变量**：只添加尚未存在的变量
- **自动去重**：同名变量只保留一个

### 3. 视觉反馈
- **高亮显示**：新提取的变量高亮显示 3 秒
- **提示消息**：显示提取结果（如"成功提取 5 个新变量"）
- **友好提示**：无新变量时显示"所有变量已存在，无需提取"

## 使用流程

### 步骤 1：创建任务并选择配置

1. 进入任务调度页面
2. 点击"新增任务"
3. 选择任务类型（拨测任务 或 巡检任务）
4. 选择拨测配置 或 巡检组/巡检项

### 步骤 2：提取变量

1. 滚动到"调度变量"区域
2. 点击**"从配置中提取变量"**按钮
3. 系统自动提取所有变量并填充到列表
4. 新提取的变量会高亮显示 3 秒

### 步骤 3：填写变量值

1. 在高亮的变量行中填写变量值
2. 变量名已自动填充，只需填写值即可
3. 可继续手动添加其他变量

### 步骤 4：保存任务

1. 填写其他必填项（任务名称、Cron 表达式等）
2. 点击"确定"保存任务

## 使用示例

### 示例 1：拨测任务变量提取

**场景**：创建 HTTP 拨测任务，目标 URL 为 `https://{{api_host}}/api/v1/users?token={{api_token}}`

**操作**：
1. 选择任务类型：拨测任务
2. 选择拨测配置：包含上述 URL 的拨测配置
3. 点击"从配置中提取变量"

**结果**：
- 自动提取 2 个变量：`api_host`、`api_token`
- 变量名已填充，用户只需填写值：
  - `api_host` → `api.example.com`
  - `api_token` → `your_token_here`

### 示例 2：巡检任务变量提取

**场景**：创建巡检任务，巡检项包含命令 `curl -H "Authorization: Bearer {{token}}" {{api_url}}/health`

**操作**：
1. 选择任务类型：巡检任务
2. 选择巡检组（包含上述巡检项）
3. 点击"从配置中提取变量"

**结果**：
- 自动提取 2 个变量：`token`、`api_url`
- 变量名已填充，用户只需填写值：
  - `token` → `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...`
  - `api_url` → `https://api.example.com`

### 示例 3：拨测类型巡检项变量提取（递归提取）

**场景**：创建巡检任务，巡检组包含拨测类型的巡检项，拨测配置中使用了变量

**巡检项配置**：
- 巡检项类型：拨测
- 关联拨测配置：HTTP 拨测，URL 为 `https://{{api_host}}/api/v1/users?token={{api_token}}`

**操作**：
1. 选择任务类型：巡检任务
2. 选择巡检组（包含上述拨测类型巡检项）
3. 点击"从配置中提取变量"

**结果**：
- 自动提取 2 个变量：`api_host`、`api_token`
- 系统自动识别拨测类型巡检项，并递归提取关联拨测配置中的变量
- 用户只需填写值：
  - `api_host` → `api.example.com`
  - `api_token` → `your_token_here`

### 示例 4：业务流程（Workflow）变量提取

**场景**：创建拨测任务，选择 Workflow 类型的拨测配置

**Workflow 配置**：
```json
{
  "variables": {
    "base_url": "https://{{api_host}}",
    "auth_token": "{{token}}"
  },
  "steps": [
    {
      "name": "登录",
      "url": "{{base_url}}/api/v1/login",
      "body": "{\"username\":\"{{username}}\",\"password\":\"{{password}}\"}"
    },
    {
      "name": "获取用户信息",
      "url": "{{base_url}}/api/v1/user/profile",
      "headers": {
        "Authorization": "Bearer {{auth_token}}"
      }
    }
  ]
}
```

**操作**：
1. 选择任务类型：拨测任务
2. 选择 Workflow 类型的拨测配置
3. 点击"从配置中提取变量"

**结果**：
- 自动提取 4 个变量：
  - `api_host`（来自 workflow variables）
  - `token`（来自 workflow variables）
  - `username`（来自 step 1 body）
  - `password`（来自 step 1 body）
- 用户只需填写值：
  - `api_host` → `api.example.com`
  - `token` → `your_token_here`
  - `username` → `admin`
  - `password` → `admin123`

### 示例 5：多配置变量合并

**场景**：选择多个拨测配置，每个配置包含不同的变量

**拨测配置 1**：`https://{{api_host}}/api/v1/users`
**拨测配置 2**：`https://{{api_host}}/api/v2/orders?key={{api_key}}`

**操作**：
1. 选择 2 个拨测配置
2. 点击"从配置中提取变量"

**结果**：
- 自动提取 2 个变量（去重后）：`api_host`、`api_key`
- 用户只需填写 2 个变量值

## 系统预置变量（自动过滤）

以下变量为系统预置变量，提取时会自动过滤：

### 时间相关（5 个）
- `timestamp` - Unix 时间戳（秒）
- `timestamp_ms` - Unix 时间戳（毫秒）
- `current_time` - 当前时间（HH:mm:ss）
- `current_date` - 当前日期（YYYY-MM-DD）
- `current_datetime` - 当前日期时间（YYYY-MM-DD HH:mm:ss）

### 随机值相关（3 个）
- `random_number` - 随机数字
- `random_string` - 随机字符串
- `random_uuid` - 随机 UUID

### 巡检相关（3 个）
- `exec_node_ip` - 执行节点 IP
- `instance` - 实例标识
- `{label}_instance` - 动态实例标识（如 `host_instance`、`pod_instance`）

## 技术实现

### 核心文件

1. **变量提取工具**：`web/src/utils/variableExtractor.ts`
   - 提供变量提取、过滤、合并等核心功能
   - 约 230 行代码

2. **任务调度页面**：`web/src/views/inspection/TaskSchedule.vue`
   - 添加提取按钮和逻辑
   - 添加高亮动画样式

### 提取逻辑

```typescript
// 1. 根据任务类型提取变量
if (taskType === 'probe') {
  // 从拨测配置提取
  extractedVars = extractFromProbeConfigs(selectedConfigs)
} else {
  // 从巡检项提取
  extractedVars = extractFromInspectionItems(selectedItems)
}

// 2. 过滤预置变量
extractedVars = extractedVars.filter(v => !isPresetVariable(v))

// 3. 合并到现有变量列表（保留已有值）
customVariablesList = mergeVariables(existing, extractedVars)

// 4. 高亮显示新变量
newlyExtractedKeys = new Set(newVariables)
```

### 变量提取规则

**正则表达式**：`/\{\{([^}]+)\}\}/g`

**提取范围**：
- **拨测配置**（全面覆盖所有类型）：
  - **基础网络（Ping）**：target、port
  - **四层协议（TCP/UDP）**：target、port
  - **应用服务（HTTP/HTTPS/WebSocket）**：target、url、body、headers、params、proxyUrl、wsMessage、assertions
  - **业务流程（Workflow）**：
    - Workflow 级别：variables（流程变量）
    - Step 级别：url、body、headers、params、proxyUrl、wsMessage、assertions
- **巡检项**：
  - command 类型：command 字段
  - script 类型：scriptContent、scriptArgs 字段
  - promql 类型：promqlQuery 字段
  - probe 类型：递归提取关联的拨测配置（自动识别 probeConfigId 并提取对应拨测配置中的变量）

## 注意事项

### 1. 提取时机
- 必须先选择拨测配置或巡检组/巡检项
- 未选择配置时点击提取按钮会提示"未找到可提取的变量"

### 2. 变量合并规则
- **已存在的变量**：保留原值，不会被覆盖
- **新变量**：添加到列表末尾，值为空
- **重复变量**：自动去重，只保留一个

### 3. 高亮显示
- 新提取的变量会高亮显示 3 秒
- 高亮颜色：浅蓝色（`rgba(22, 93, 255, 0.15)`）
- 3 秒后自动恢复正常样式

### 4. 变量优先级
提取的变量属于"调度变量"，优先级最高：
- **拨测任务**：调度变量 > 巡检组环境变量 > 拨测流程变量
- **巡检任务**：调度变量 > 巡检组自定义变量 > 全局变量

## 常见问题

### Q1：为什么提取不到变量？
**A**：可能的原因：
1. 未选择拨测配置或巡检组/巡检项
2. 配置中没有使用 `{{variable}}` 格式的变量
3. 配置中只包含系统预置变量（会被自动过滤）

### Q2：为什么提示"所有变量已存在"？
**A**：提取的变量已经在自定义变量列表中，无需重复添加。

### Q3：提取后可以修改变量名吗？
**A**：可以。提取后的变量名可以手动修改，但建议保持与配置中一致。

### Q4：提取后可以删除不需要的变量吗？
**A**：可以。点击变量行右侧的删除按钮即可删除。

### Q5：提取的变量值为什么是空的？
**A**：这是设计行为。提取功能只提取变量名，变量值需要用户根据实际情况填写。

## 更新日志

### v1.0.0 (2026-05-19)
- ✅ 新增变量提取工具（`variableExtractor.ts`）
- ✅ 任务调度页面新增"从配置中提取变量"按钮
- ✅ 支持从拨测配置和巡检项中提取变量
- ✅ 自动过滤 11 种系统预置变量
- ✅ 智能合并变量（保留已有值）
- ✅ 新变量高亮显示 3 秒
- ✅ 友好的提示消息

## 反馈与建议

如有问题或建议，请联系开发团队或在项目 Issue 中反馈。
