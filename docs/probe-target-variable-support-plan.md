# 基础网络拨测目标地址和端口支持变量引用方案

## 一、需求概述

在拨测管理中，为基础网络类型（Ping/TCP/UDP）的目标地址和端口字段增加变量引用功能，与应用服务类型保持一致的用户体验。

### 当前状态

#### 已支持变量引用的字段
- **应用服务类型**（HTTP/HTTPS/WebSocket）：
  - URL 字段 ✅
  - Headers 字段 ✅
  - Params 字段 ✅
  - Body 字段 ✅
  - ProxyURL 字段 ✅

#### 不支持变量引用的字段
- **基础网络类型**（Ping/TCP/UDP）：
  - Target 字段（目标地址）❌
  - Port 字段（端口）❌

### 需求目标

1. **前端**：Target 和 Port 字段使用 `VariableInput` 组件，支持 `/` 快捷键插入变量
2. **后端**：变量解析器支持解析 Target 和 Port 字段中的变量
3. **指标**：Prometheus 指标中的 `target` 标签显示解析后的值
4. **巡检结果**：巡检结果记录中显示解析后的值

---

## 二、现状分析

### 2.1 前端现状

**文件**：`web/src/views/inspection/ProbeManagement.vue`

**当前代码**（第128-129行）：
```vue
<a-form-item v-if="formData.category !== 'application' && formData.category !== 'workflow'" 
             label="目标地址" field="target">
  <a-input v-model="formData.target" placeholder="IP 或域名" />
</a-form-item>

<a-form-item v-if="formData.type !== 'ping' && formData.category !== 'application' && formData.category !== 'workflow'" 
             label="端口" field="port">
  <a-input-number v-model="formData.port" :min="1" :max="65535" style="width: 100%;" />
</a-form-item>
```

**问题**：
- Target 使用普通的 `a-input`，不支持变量引用
- Port 使用 `a-input-number`，不支持变量引用

---

### 2.2 后端现状

**文件**：`internal/biz/inspection/variable_resolver.go`

**当前代码**（第124行）：
```go
func (r *VariableResolver) ResolveConfig(ctx context.Context, cfg *ProbeConfig) (*ProbeConfig, error) {
    texts := []string{cfg.Target, cfg.URL, cfg.Headers, cfg.Params, cfg.Body, cfg.ProxyURL}
    names := ExtractVariableNames(texts...)
    // ...
    resolved.Target = replacer(cfg.Target)
    // ...
}
```

**现状**：
- Target 字段已经在变量解析逻辑中 ✅
- Port 字段未包含在变量解析逻辑中 ❌

---

### 2.3 指标现状

**文件**：`internal/biz/inspection/executor.go`

**当前代码**（第249行）：
```go
e.pushMetrics(ctx, probeTask, resolvedCfg, result, businessGroupNames)
```

**现状**：
- 已经传入 `resolvedCfg`（变量解析后的配置）✅
- Target 字段的变量会被正确解析并显示在指标中 ✅
- Port 字段未支持变量，无法解析 ❌

---

## 三、详细方案

### 方案 A：完整支持（推荐）

支持 Target 和 Port 字段的变量引用，包括前端 UI 和后端解析。

---

### 方案 B：仅支持 Target

只支持 Target 字段的变量引用，Port 保持数字输入框。

**理由**：
- Port 通常是固定值（如 3306、6379、22）
- 变量引用的场景较少
- 保持 Port 为数字类型，避免类型转换问题

---

## 四、方案 A 详细实现（推荐）

### 4.1 前端修改

#### 修改1：Target 字段改用 VariableInput

**文件**：`web/src/views/inspection/ProbeManagement.vue`

**位置**：第128行

```vue
<!-- 修改前 -->
<a-form-item v-if="formData.category !== 'application' && formData.category !== 'workflow'" 
             label="目标地址" field="target">
  <a-input v-model="formData.target" placeholder="IP 或域名" />
</a-form-item>

<!-- 修改后 -->
<a-form-item v-if="formData.category !== 'application' && formData.category !== 'workflow'" 
             label="目标地址" field="target">
  <VariableInput 
    v-model="formData.target" 
    placeholder="IP 或域名（输入 / 插入变量）" 
    :variables="variableOptions" 
  />
</a-form-item>
```

**效果**：
- 用户可以输入 `{{ping_host}}` 或 `{{db_host}}`
- 按 `/` 键弹出变量选择下拉框
- 变量以蓝色高亮显示

---

#### 修改2：Port 字段改用 VariableInput

**文件**：`web/src/views/inspection/ProbeManagement.vue`

**位置**：第129行

```vue
<!-- 修改前 -->
<a-form-item v-if="formData.type !== 'ping' && formData.category !== 'application' && formData.category !== 'workflow'" 
             label="端口" field="port">
  <a-input-number v-model="formData.port" :min="1" :max="65535" style="width: 100%;" />
</a-form-item>

<!-- 修改后 -->
<a-form-item v-if="formData.type !== 'ping' && formData.category !== 'application' && formData.category !== 'workflow'" 
             label="端口" field="port">
  <VariableInput 
    v-model="formData.port" 
    placeholder="端口号或变量（如 {{mysql_port}}）" 
    :variables="variableOptions" 
  />
</a-form-item>
```

**注意**：
- Port 字段从 `number` 类型改为 `string` 类型
- 需要修改表单数据类型定义
- 后端需要支持 Port 字段的字符串类型

---

#### 修改3：表单数据类型调整

**文件**：`web/src/views/inspection/ProbeManagement.vue`

**位置**：约第700行（formData 定义）

```typescript
// 修改前
const formData = reactive({
  // ...
  target: '',
  port: 0,
  // ...
})

// 修改后
const formData = reactive({
  // ...
  target: '',
  port: '', // 改为字符串类型，支持变量
  // ...
})
```

---

#### 修改4：表单验证调整

**文件**：`web/src/views/inspection/ProbeManagement.vue`

**位置**：约第740行（表单验证规则）

```typescript
// 修改前
const formRules = {
  target: [{ required: formData.category !== 'application' && formData.category !== 'workflow', message: '请输入目标地址' }],
  port: [{ required: formData.type !== 'ping' && formData.category !== 'application', message: '请输入端口' }],
}

// 修改后
const formRules = {
  target: [{ required: formData.category !== 'application' && formData.category !== 'workflow', message: '请输入目标地址' }],
  port: [
    { required: formData.type !== 'ping' && formData.category !== 'application', message: '请输入端口' },
    { 
      validator: (value: string, cb: (error?: string) => void) => {
        // 如果包含变量，跳过验证
        if (value && value.includes('{{')) {
          return cb()
        }
        // 如果是纯数字，验证范围
        const num = parseInt(value)
        if (isNaN(num) || num < 1 || num > 65535) {
          return cb('端口号必须在 1-65535 之间')
        }
        cb()
      }
    }
  ],
}
```

---

### 4.2 后端修改

#### 修改1：ProbeConfig 结构体调整

**文件**：`internal/biz/inspection/models.go`

**位置**：ProbeConfig 结构体定义

```go
// 修改前
type ProbeConfig struct {
    // ...
    Target      string         `gorm:"type:varchar(255);not null" json:"target"`
    Port        int            `gorm:"default:0" json:"port"`
    // ...
}

// 修改后
type ProbeConfig struct {
    // ...
    Target      string         `gorm:"type:varchar(255);not null" json:"target"`
    Port        string         `gorm:"type:varchar(50);default:''" json:"port"` // 改为字符串，支持变量
    // ...
}
```

**影响**：
- 需要数据库迁移（ALTER TABLE）
- 需要更新 init.sql

---

#### 修改2：变量解析器支持 Port 字段

**文件**：`internal/biz/inspection/variable_resolver.go`

**位置**：第124行（ResolveConfig 方法）

```go
// 修改前
func (r *VariableResolver) ResolveConfig(ctx context.Context, cfg *ProbeConfig) (*ProbeConfig, error) {
    texts := []string{cfg.Target, cfg.URL, cfg.Headers, cfg.Params, cfg.Body, cfg.ProxyURL}
    names := ExtractVariableNames(texts...)
    // ...
    resolved.Target = replacer(cfg.Target)
    // ...
}

// 修改后
func (r *VariableResolver) ResolveConfig(ctx context.Context, cfg *ProbeConfig) (*ProbeConfig, error) {
    texts := []string{cfg.Target, cfg.Port, cfg.URL, cfg.Headers, cfg.Params, cfg.Body, cfg.ProxyURL}
    names := ExtractVariableNames(texts...)
    // ...
    resolved.Target = replacer(cfg.Target)
    resolved.Port = replacer(cfg.Port) // 新增：解析 Port 字段
    // ...
}
```

---

#### 修改3：ResolveConfigWithExtra 方法同步修改

**文件**：`internal/biz/inspection/variable_resolver.go`

**位置**：第67行（ResolveConfigWithExtra 方法）

```go
// 修改前
func (r *VariableResolver) ResolveConfigWithExtra(ctx context.Context, cfg *ProbeConfig, extraVars map[string]string) (*ProbeConfig, error) {
    texts := []string{cfg.Target, cfg.URL, cfg.Headers, cfg.Params, cfg.Body, cfg.ProxyURL, cfg.WSMessage}
    // ...
    resolved.Target = resolve(cfg.Target)
    // ...
}

// 修改后
func (r *VariableResolver) ResolveConfigWithExtra(ctx context.Context, cfg *ProbeConfig, extraVars map[string]string) (*ProbeConfig, error) {
    texts := []string{cfg.Target, cfg.Port, cfg.URL, cfg.Headers, cfg.Params, cfg.Body, cfg.ProxyURL, cfg.WSMessage}
    // ...
    resolved.Target = resolve(cfg.Target)
    resolved.Port = resolve(cfg.Port) // 新增：解析 Port 字段
    // ...
}
```

---

#### 修改4：拨测执行时的 Port 类型转换

**文件**：`internal/biz/inspection/executor.go`

**位置**：约第160-180行（executeProbe 方法）

```go
// 修改前
func (e *NetworkProbeExecutor) executeProbe(cfg *ProbeConfig) (*probers.Result, uint) {
    // ...
    prober.Probe(cfg.Target, cfg.Port, cfg.Timeout, cfg.Count, cfg.PacketSize)
    // ...
}

// 修改后
func (e *NetworkProbeExecutor) executeProbe(cfg *ProbeConfig) (*probers.Result, uint) {
    // ...
    // 将 Port 字符串转换为整数
    port := 0
    if cfg.Port != "" {
        if p, err := strconv.Atoi(cfg.Port); err == nil {
            port = p
        } else {
            appLogger.Warn("invalid port value", zap.String("port", cfg.Port), zap.Error(err))
        }
    }
    prober.Probe(cfg.Target, port, cfg.Timeout, cfg.Count, cfg.PacketSize)
    // ...
}
```

**注意**：需要在所有调用 `cfg.Port` 的地方进行类型转换。

---

#### 修改5：指标推送时的 Port 处理

**文件**：`internal/biz/inspection/executor.go`

**位置**：约第1430行（pushMetrics 方法）

```go
// 修改前
switch config.Type {
case "tcp", "udp":
    if config.Port > 0 {
        targetValue = fmt.Sprintf("%s:%d", config.Target, config.Port)
    }
}

// 修改后
switch config.Type {
case "tcp", "udp":
    if config.Port != "" {
        targetValue = fmt.Sprintf("%s:%s", config.Target, config.Port)
    }
}
```

---

### 4.3 数据库迁移

#### 迁移脚本

**文件**：`migrations/xxx_alter_probe_config_port_to_string.sql`（新建）

```sql
-- 将 probe_configs 表的 port 字段从 int 改为 varchar(50)
ALTER TABLE `probe_configs` MODIFY COLUMN `port` varchar(50) DEFAULT '' COMMENT '端口（支持变量）';

-- 将现有的数字端口转换为字符串
UPDATE `probe_configs` SET `port` = CAST(`port` AS CHAR) WHERE `port` > 0;
UPDATE `probe_configs` SET `port` = '' WHERE `port` = 0;
```

#### 更新 init.sql

**文件**：`migrations/init.sql`

```sql
-- 修改前
`port` int DEFAULT 0 COMMENT '端口',

-- 修改后
`port` varchar(50) DEFAULT '' COMMENT '端口（支持变量）',
```

---

## 五、方案 B 详细实现（仅支持 Target）

### 5.1 前端修改

只修改 Target 字段，Port 保持不变。

**文件**：`web/src/views/inspection/ProbeManagement.vue`

**位置**：第128行

```vue
<!-- 修改前 -->
<a-form-item v-if="formData.category !== 'application' && formData.category !== 'workflow'" 
             label="目标地址" field="target">
  <a-input v-model="formData.target" placeholder="IP 或域名" />
</a-form-item>

<!-- 修改后 -->
<a-form-item v-if="formData.category !== 'application' && formData.category !== 'workflow'" 
             label="目标地址" field="target">
  <VariableInput 
    v-model="formData.target" 
    placeholder="IP 或域名（输入 / 插入变量）" 
    :variables="variableOptions" 
  />
</a-form-item>
```

### 5.2 后端修改

无需修改，Target 字段已经支持变量解析。

---

## 六、修改文件清单

### 方案 A（完整支持）

#### 前端
1. `web/src/views/inspection/ProbeManagement.vue`
   - 第128行：Target 字段改用 VariableInput
   - 第129行：Port 字段改用 VariableInput
   - 约第700行：formData.port 类型改为 string
   - 约第740行：Port 字段验证规则调整

#### 后端
1. `internal/biz/inspection/models.go`
   - ProbeConfig.Port 字段类型改为 string

2. `internal/biz/inspection/variable_resolver.go`
   - 第124行：ResolveConfig 方法添加 Port 字段解析
   - 第67行：ResolveConfigWithExtra 方法添加 Port 字段解析

3. `internal/biz/inspection/executor.go`
   - 约第160-180行：executeProbe 方法添加 Port 类型转换
   - 约第1430行：pushMetrics 方法调整 Port 拼接逻辑
   - 其他所有使用 `cfg.Port` 的地方（需要全局搜索）

4. `migrations/xxx_alter_probe_config_port_to_string.sql`（新建）
   - 数据库迁移脚本

5. `migrations/init.sql`
   - 更新 port 字段定义

#### 修改统计
- **前端文件**：1个
- **后端文件**：4个（含1个新建）
- **修改行数**：约50行
- **数据库迁移**：1个表字段类型变更

---

### 方案 B（仅支持 Target）

#### 前端
1. `web/src/views/inspection/ProbeManagement.vue`
   - 第128行：Target 字段改用 VariableInput

#### 后端
无需修改

#### 修改统计
- **前端文件**：1个
- **后端文件**：0个
- **修改行数**：约5行
- **数据库迁移**：无

---

## 七、修复效果

### 方案 A 效果

#### 效果1：Target 变量引用

**前端输入**：
```
Target: {{db_host}}
Port: {{mysql_port}}
```

**后端解析**：
```
Target: 192.168.1.100
Port: 3306
```

**指标显示**：
```prometheus
srehub_inspect_tcp_port_reachable{target="192.168.1.100:3306"} 1
```

**巡检结果**：
```
目标：192.168.1.100:3306
状态：成功
```

---

#### 效果2：混合使用

**前端输入**：
```
Target: {{db_host}}
Port: 3306
```

**后端解析**：
```
Target: 192.168.1.100
Port: 3306
```

**指标显示**：
```prometheus
srehub_inspect_tcp_port_reachable{target="192.168.1.100:3306"} 1
```

---

### 方案 B 效果

#### 效果1：Target 变量引用

**前端输入**：
```
Target: {{ping_host}}
```

**后端解析**：
```
Target: 192.168.1.1
```

**指标显示**：
```prometheus
srehub_inspect_ping_avg_rtt_seconds{target="192.168.1.1"} 0.015
```

---

## 八、注意事项

### 8.1 方案 A 注意事项

1. **数据库迁移风险**：
   - Port 字段类型变更需要停机维护
   - 需要备份数据库
   - 迁移脚本需要充分测试

2. **类型转换**：
   - 所有使用 `cfg.Port` 的地方需要类型转换
   - 需要处理转换失败的情况
   - 建议添加日志记录转换错误

3. **向后兼容**：
   - 旧的数字端口需要转换为字符串
   - 前端需要兼容旧数据（数字类型）

4. **验证逻辑**：
   - 前端需要验证 Port 值（数字或变量）
   - 后端需要验证解析后的 Port 值范围

---

### 8.2 方案 B 注意事项

1. **功能限制**：
   - Port 不支持变量，灵活性较低
   - 无法通过变量统一管理端口

2. **用户体验**：
   - Target 和 Port 的输入方式不一致
   - 可能造成用户困惑

---

## 九、推荐方案

### 推荐：方案 A（完整支持）

**理由**：
1. **功能完整**：Target 和 Port 都支持变量，灵活性高
2. **用户体验一致**：与应用服务类型保持一致
3. **便于管理**：可以通过变量统一管理常用端口（如 `{{mysql_port}}`、`{{redis_port}}`）
4. **扩展性好**：未来可以支持更多字段的变量引用

**风险可控**：
- 数据库迁移风险可通过充分测试降低
- 类型转换逻辑简单，不易出错
- 向后兼容性好

---

## 十、实施步骤

### 阶段1：前端修改（1-2小时）
1. 修改 Target 字段为 VariableInput
2. 修改 Port 字段为 VariableInput
3. 调整表单数据类型和验证规则
4. 前端测试

### 阶段2：后端修改（2-3小时）
1. 修改 ProbeConfig 结构体
2. 修改变量解析器
3. 修改拨测执行逻辑
4. 修改指标推送逻辑
5. 全局搜索并修改所有使用 `cfg.Port` 的地方

### 阶段3：数据库迁移（1小时）
1. 编写迁移脚本
2. 测试环境验证
3. 生产环境备份
4. 执行迁移

### 阶段4：测试验证（2-3小时）
1. 单元测试
2. 集成测试
3. 端到端测试
4. 性能测试

### 总计：6-9小时

---

## 十一、测试建议

### 测试用例1：Target 变量引用
1. 创建变量：`ping_host=192.168.1.1`
2. 创建 Ping 拨测：`Target={{ping_host}}`
3. 执行拨测
4. 验证指标：`target="192.168.1.1"`
5. 验证巡检结果：显示 `192.168.1.1`

### 测试用例2：Port 变量引用
1. 创建变量：`mysql_port=3306`
2. 创建 TCP 拨测：`Target=192.168.1.100`，`Port={{mysql_port}}`
3. 执行拨测
4. 验证指标：`target="192.168.1.100:3306"`
5. 验证巡检结果：显示 `192.168.1.100:3306`

### 测试用例3：混合使用
1. 创建变量：`db_host=192.168.1.100`
2. 创建 TCP 拨测：`Target={{db_host}}`，`Port=3306`
3. 执行拨测
4. 验证指标：`target="192.168.1.100:3306"`

### 测试用例4：变量不存在
1. 创建 TCP 拨测：`Target={{undefined_var}}`，`Port=3306`
2. 执行拨测
3. 验证行为：保持原样 `{{undefined_var}}` 或报错

### 测试用例5：Port 无效值
1. 创建变量：`invalid_port=abc`
2. 创建 TCP 拨测：`Target=192.168.1.100`，`Port={{invalid_port}}`
3. 执行拨测
4. 验证行为：报错或使用默认端口

---

**方案整理完成日期**：2026-05-15  
**整理人员**：Claude (Opus 4.6)  
**状态**：待用户确认
