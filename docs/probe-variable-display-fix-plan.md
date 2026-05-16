# 拨测执行结果中变量未渲染问题修复方案

## 一、问题描述

### 问题现象
在拨测管理的"执行"功能和任务调度执行中，基础网络类型（Ping）和四层协议类型（TCP/UDP）的执行结果显示的是**未解析的变量**，而不是解析后的实际值。

**示例**：
- 配置：`Target: {{db_host}}`，`Port: {{mysql_port}}`
- 变量定义：`db_host=192.168.1.100`，`mysql_port=3306`
- **期望显示**：`目标地址: 192.168.1.100`，`端口: 3306`
- **实际显示**：`目标地址: {{db_host}}`，`端口: {{mysql_port}}`

### 影响范围
1. **拨测管理 - 执行功能**（`/api/v1/inspection/probe-configs/:id/run-once`）
2. **拨测管理 - 测试功能**（`/api/v1/inspection/probe-configs/test`）
3. **任务调度 - 执行结果**（巡检结果记录）

### 根本原因
后端在返回执行结果时，只返回了拨测结果（Success、Latency 等），**没有返回解析后的 Target 和 Port 值**。前端显示时使用的是原始配置中的值（`currentRecord.target` 和 `currentRecord.port`），导致变量未渲染。

---

## 二、现状分析

### 2.1 后端现状

#### 文件：`internal/service/inspection/probe_config_service.go`

**RunOnce 方法**（第370-386行）：
```go
response.Success(c, gin.H{
    "Success":         result.Success,
    "Latency":         result.Latency,
    "PacketLoss":      result.PacketLoss,
    // ... 其他拨测结果字段
    "Error":           result.Error,
    "agentHostId":     agentHostID,
    "retryAttempt":    retryAttempt,
    // ❌ 缺少：解析后的 Target 和 Port
})
```

**问题**：
- 只返回了拨测结果，没有返回解析后的配置信息
- 前端无法获取解析后的 Target 和 Port 值

---

### 2.2 前端现状

#### 文件：`web/src/views/inspection/ProbeManagement.vue`

**执行结果显示**（第578-581行）：
```vue
<template v-if="currentRecord.category !== 'application'">
  <a-descriptions-item label="目标地址">{{ currentRecord.target || '-' }}</a-descriptions-item>
  <a-descriptions-item v-if="currentRecord.type !== 'ping'" label="端口">{{ currentRecord.port || '-' }}</a-descriptions-item>
  <a-descriptions-item label="拨测类型">{{ (currentRecord.type || '').toUpperCase() }}</a-descriptions-item>
</template>
```

**问题**：
- `currentRecord` 是原始配置，包含未解析的变量
- 应该显示解析后的值，而不是原始配置

---

## 三、修复方案

### 方案 A：后端返回解析后的配置（推荐）

**原理**：在后端返回执行结果时，同时返回解析后的 Target 和 Port 值。

**优点**：
- 前端无需修改逻辑，直接使用后端返回的值
- 数据来源统一，避免前端重复解析
- 适用于所有场景（拨测管理、任务调度）

**缺点**：
- 需要修改后端返回结构

---

### 方案 B：前端解析变量

**原理**：前端在显示时，调用变量解析 API 解析 Target 和 Port。

**优点**：
- 后端无需修改

**缺点**：
- 前端需要额外调用 API
- 增加网络请求
- 逻辑复杂，需要处理异步

---

## 四、方案 A 详细实现（推荐）

### 4.1 后端修改

#### 修改1：RunOnce 方法返回解析后的配置

**文件**：`internal/service/inspection/probe_config_service.go`

**位置**：第370-386行

```go
// 修改前
response.Success(c, gin.H{
    "Success":         result.Success,
    "Latency":         result.Latency,
    // ...
    "Error":           result.Error,
    "agentHostId":     agentHostID,
    "retryAttempt":    retryAttempt,
})

// 修改后
response.Success(c, gin.H{
    "Success":         result.Success,
    "Latency":         result.Latency,
    // ...
    "Error":           result.Error,
    "agentHostId":     agentHostID,
    "retryAttempt":    retryAttempt,
    // 新增：解析后的配置信息
    "ResolvedTarget":  resolvedConfig.Target,
    "ResolvedPort":    resolvedConfig.Port,
})
```

**说明**：
- `resolvedConfig` 是变量解析后的配置
- `ResolvedTarget` 和 `ResolvedPort` 是解析后的值

---

#### 修改2：TestProbe 方法返回解析后的配置

**文件**：`internal/service/inspection/probe_config_service.go`

**位置**：约第667-680行

```go
// 修改前
response.Success(c, gin.H{
    "Success":         result.Success,
    "Latency":         result.Latency,
    // ...
})

// 修改后
response.Success(c, gin.H{
    "Success":         result.Success,
    "Latency":         result.Latency,
    // ...
    // 新增：解析后的配置信息
    "ResolvedTarget":  resolvedConfig.Target,
    "ResolvedPort":    resolvedConfig.Port,
})
```

---

#### 修改3：runOnceApp 方法返回解析后的配置

**文件**：`internal/service/inspection/probe_config_service.go`

**位置**：约第401-414行 和 第435-450行

```go
// 修改前（Agent 模式）
response.Success(c, gin.H{
    "Success":           appResult.Success,
    // ...
    "agentHostId":       agentHostID,
    "retryAttempt":      0,
})

// 修改后（Agent 模式）
response.Success(c, gin.H{
    "Success":           appResult.Success,
    // ...
    "agentHostId":       agentHostID,
    "retryAttempt":      0,
    // 新增：解析后的配置信息
    "ResolvedURL":       resolvedConfig.URL,
})

// 修改前（本地模式）
response.Success(c, gin.H{
    "Success":           appResult.Success,
    // ...
    "retryAttempt":      retryAttempt,
})

// 修改后（本地模式）
response.Success(c, gin.H{
    "Success":           appResult.Success,
    // ...
    "retryAttempt":      retryAttempt,
    // 新增：解析后的配置信息
    "ResolvedURL":       resolvedConfig.URL,
})
```

**说明**：
- 应用服务类型返回 `ResolvedURL`（已解析变量的完整 URL）

---

### 4.2 前端修改

#### 修改1：执行结果显示使用解析后的值

**文件**：`web/src/views/inspection/ProbeManagement.vue`

**位置**：第578-581行

```vue
<!-- 修改前 -->
<template v-if="currentRecord.category !== 'application'">
  <a-descriptions-item label="目标地址">{{ currentRecord.target || '-' }}</a-descriptions-item>
  <a-descriptions-item v-if="currentRecord.type !== 'ping'" label="端口">{{ currentRecord.port || '-' }}</a-descriptions-item>
  <a-descriptions-item label="拨测类型">{{ (currentRecord.type || '').toUpperCase() }}</a-descriptions-item>
</template>

<!-- 修改后 -->
<template v-if="currentRecord.category !== 'application'">
  <a-descriptions-item label="目标地址">{{ runResult.ResolvedTarget || currentRecord.target || '-' }}</a-descriptions-item>
  <a-descriptions-item v-if="currentRecord.type !== 'ping'" label="端口">{{ runResult.ResolvedPort || currentRecord.port || '-' }}</a-descriptions-item>
  <a-descriptions-item label="拨测类型">{{ (currentRecord.type || '').toUpperCase() }}</a-descriptions-item>
</template>
```

**说明**：
- 优先使用 `runResult.ResolvedTarget` 和 `runResult.ResolvedPort`（解析后的值）
- 如果不存在，回退到 `currentRecord.target` 和 `currentRecord.port`（原始值）

---

#### 修改2：应用服务类型显示解析后的 URL

**文件**：`web/src/views/inspection/ProbeManagement.vue`

**位置**：第585行

```vue
<!-- 修改前 -->
<a-descriptions-item label="请求URL">{{ buildDisplayUrl(currentRecord) }}</a-descriptions-item>

<!-- 修改后 -->
<a-descriptions-item label="请求URL">{{ runResult.ResolvedURL || buildDisplayUrl(currentRecord) }}</a-descriptions-item>
```

**说明**：
- 优先使用 `runResult.ResolvedURL`（解析后的 URL）
- 如果不存在，回退到 `buildDisplayUrl(currentRecord)`（原始 URL）

---

### 4.3 任务调度执行结果修改

#### 问题分析

任务调度执行后，巡检结果记录中也需要显示解析后的值。

**涉及文件**：
1. `internal/biz/inspection/executor.go` - 保存巡检结果
2. `web/src/views/inspection/InspectionRecords.vue` - 显示巡检结果

#### 修改方案

**方案1**：在 ProbeResult 模型中添加解析后的字段

**文件**：`internal/biz/inspection/models.go`

```go
// 修改前
type ProbeResult struct {
    ID            uint           `gorm:"primaryKey" json:"id"`
    ProbeTaskID   uint           `gorm:"index" json:"probeTaskId"`
    ProbeConfigID uint           `gorm:"index" json:"probeConfigId"`
    Success       bool           `json:"success"`
    Latency       float64        `json:"latency"`
    // ...
}

// 修改后
type ProbeResult struct {
    ID            uint           `gorm:"primaryKey" json:"id"`
    ProbeTaskID   uint           `gorm:"index" json:"probeTaskId"`
    ProbeConfigID uint           `gorm:"index" json:"probeConfigId"`
    Success       bool           `json:"success"`
    Latency       float64        `json:"latency"`
    // ...
    // 新增：解析后的配置信息
    ResolvedTarget string        `gorm:"type:varchar(255)" json:"resolvedTarget"`
    ResolvedPort   string        `gorm:"type:varchar(50)" json:"resolvedPort"`
    ResolvedURL    string        `gorm:"type:varchar(2000)" json:"resolvedUrl"`
}
```

**说明**：
- 在数据库中保存解析后的值
- 前端查询巡检结果时可以直接使用

---

**方案2**：前端显示时动态解析（不推荐）

前端在显示巡检结果时，调用变量解析 API 解析变量。

**缺点**：
- 增加网络请求
- 逻辑复杂
- 性能差

---

## 五、修改文件清单

### 方案 A（推荐）

#### 后端
1. `internal/service/inspection/probe_config_service.go`
   - 第370-386行：RunOnce 方法添加 ResolvedTarget 和 ResolvedPort
   - 约第401-414行：runOnceApp 方法（Agent 模式）添加 ResolvedURL
   - 约第435-450行：runOnceApp 方法（本地模式）添加 ResolvedURL
   - 约第667-680行：TestProbe 方法添加 ResolvedTarget 和 ResolvedPort

2. `internal/biz/inspection/models.go`（可选，用于任务调度）
   - ProbeResult 结构体添加 ResolvedTarget、ResolvedPort、ResolvedURL 字段

3. `internal/biz/inspection/executor.go`（可选，用于任务调度）
   - 保存巡检结果时填充 ResolvedTarget、ResolvedPort、ResolvedURL

#### 前端
1. `web/src/views/inspection/ProbeManagement.vue`
   - 第578-581行：执行结果显示使用 runResult.ResolvedTarget 和 runResult.ResolvedPort
   - 第585行：应用服务类型显示使用 runResult.ResolvedURL

2. `web/src/views/inspection/InspectionRecords.vue`（可选，用于任务调度）
   - 巡检结果显示使用 resolvedTarget、resolvedPort、resolvedUrl

#### 修改统计
- **后端文件**：2-3个
- **前端文件**：1-2个
- **修改行数**：约30-50行
- **数据库迁移**：可选（如果修改 ProbeResult 模型）

---

## 六、修复效果

### 效果1：拨测管理 - 执行功能

**修复前**：
```
目标地址: {{db_host}}
端口: {{mysql_port}}
```

**修复后**：
```
目标地址: 192.168.1.100
端口: 3306
```

---

### 效果2：应用服务类型

**修复前**：
```
请求URL: {{base_url}}/api/health
```

**修复后**：
```
请求URL: https://api.example.com/api/health
```

---

### 效果3：任务调度 - 巡检结果

**修复前**：
```
目标: {{db_host}}:{{mysql_port}}
```

**修复后**：
```
目标: 192.168.1.100:3306
```

---

## 七、注意事项

### 7.1 向后兼容

修改后，旧的执行结果（没有 ResolvedTarget 等字段）仍然可以正常显示：
- 前端使用 `runResult.ResolvedTarget || currentRecord.target`
- 如果 `runResult.ResolvedTarget` 不存在，回退到 `currentRecord.target`

### 7.2 任务调度修改（可选）

如果需要在任务调度的巡检结果中也显示解析后的值，需要：
1. 修改 ProbeResult 模型（添加字段）
2. 修改 executor.go（保存时填充字段）
3. 修改前端巡检结果显示页面
4. 执行数据库迁移（AutoMigrate）

**建议**：先修复拨测管理的执行功能，任务调度可以后续优化。

### 7.3 测试建议

1. **基础网络类型**：
   - 创建 Ping 拨测，Target 使用变量
   - 执行拨测，验证显示解析后的值

2. **四层协议类型**：
   - 创建 TCP 拨测，Target 和 Port 使用变量
   - 执行拨测，验证显示解析后的值

3. **应用服务类型**：
   - 创建 HTTP 拨测，URL 使用变量
   - 执行拨测，验证显示解析后的 URL

4. **任务调度**（可选）：
   - 创建任务调度，执行拨测
   - 查看巡检结果，验证显示解析后的值

---

## 八、实施步骤

### 阶段1：拨测管理执行功能修复（1-2小时）
1. 修改后端 RunOnce 方法
2. 修改后端 TestProbe 方法
3. 修改后端 runOnceApp 方法
4. 修改前端执行结果显示
5. 测试验证

### 阶段2：任务调度修复（可选，2-3小时）
1. 修改 ProbeResult 模型
2. 修改 executor.go 保存逻辑
3. 修改前端巡检结果显示
4. 数据库迁移
5. 测试验证

### 总计：1-5小时

---

## 九、推荐方案

**推荐：方案 A（后端返回解析后的配置）**

**理由**：
1. 前端逻辑简单，无需额外 API 调用
2. 数据来源统一，避免前端重复解析
3. 适用于所有场景（拨测管理、任务调度）
4. 向后兼容，不影响旧数据

**实施优先级**：
1. **高优先级**：拨测管理执行功能（RunOnce、TestProbe、runOnceApp）
2. **中优先级**：任务调度巡检结果（ProbeResult 模型）

---

**方案整理完成日期**：2026-05-16  
**整理人员**：Claude (Opus 4.6)  
**状态**：待用户确认
