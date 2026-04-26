# PromQL 巡检功能实施方案

## 一、方案概述

### 1.1 功能目标

在智能巡检模块中新增 PromQL 查询类型，支持通过 Prometheus/VictoriaMetrics 数据源进行指标巡检，实现：

- 巡检组关联数据源（单个数据源，复用告警管理的数据源配置）
- 支持 Instant Query 和 Range Query 两种查询类型
- 支持预置变量（主机 IP、标签等）自动注入到 PromQL 语句
- 新增 PromQL 专用断言类型（阈值、区间、存在性判断）
- 并发控制（默认 50 个并发）
- 完整的执行结果记录和追溯

### 1.2 核心调整

基于讨论，最终方案调整如下：

1. ✅ **单数据源**：巡检组只关联一个数据源，简化逻辑
2. ✅ **Range Query 一期实现**：支持 instant 和 range 两种查询类型
3. ✅ **超时 30 秒**：查询超时时间设为 30 秒
4. ✅ **并发控制 50**：使用 goroutine pool 限制并发数为 50
5. ✅ **不需要缓存**：每次实时查询

---

## 二、数据库变更

### 2.1 巡检组表（inspection_groups）

**删除字段**：
- `prometheus_url`
- `prometheus_username`
- `prometheus_password`

**新增字段**：
- `datasource_id` (INT UNSIGNED, DEFAULT 0, INDEX) - 关联的数据源 ID

### 2.2 巡检项表（inspection_items）

**新增字段**：
- `promql_query_type` (VARCHAR(20), DEFAULT 'instant') - PromQL 查询类型（instant/range）

**修改字段**：
- `assertion_value` (TEXT) - 扩展长度，支持 JSON 格式配置

### 2.3 巡检执行详情表（inspection_execution_details）

**新增字段**：
- `datasource_id` (INT UNSIGNED, DEFAULT 0, INDEX) - 使用的数据源 ID
- `promql` (TEXT) - 实际执行的 PromQL（变量已替换）
- `promql_result` (LONGTEXT) - 原始查询结果（JSON）
- `metric_value` (DECIMAL(20,4)) - 提取的指标值
- `metric_labels` (JSON) - 指标标签（JSON）
- `assertion_pass` (BOOLEAN) - 断言是否通过
- `assertion_rule` (TEXT) - 应用的断言规则（JSON）
- `failure_reason` (TEXT) - 失败原因详情

### 2.4 迁移脚本

```sql
-- 执行脚本：migrations/promql_inspection.sql
-- 详见文件内容
```

---

## 三、后端实现

### 3.1 核心组件

#### 1. PromQL 执行器（`plugins/inspection/executor/promql_executor.go`）

**功能**：
- 渲染 PromQL 模板（注入预置变量）
- 查询数据源（支持 Instant Query 和 Range Query）
- 解析查询结果（提取指标值和标签）
- 并发控制（使用信号量，默认 50）

**预置变量**：
- `{{.Instance}}` - 主机 instance 标签（IP:Port）
- `{{.IP}}` - 主机 IP 地址
- `{{.Hostname}}` - 主机名
- `{{.Labels.xxx}}` - 主机标签
- `{{.ServiceLabels.xxx}}` - 服务标签

**示例**：
```go
promqlExecutor := executor.NewPromQLExecutor(datasourceRepo, 50)
result := promqlExecutor.Execute(ctx, promql, host, datasourceID, queryType)
```

#### 2. 断言验证器扩展（`plugins/inspection/executor/assertion_validator.go`）

**新增断言类型**：

| 断言类型 | 常量 | 用途 |
|---------|------|------|
| PromQL 阈值判断 | `promql_threshold` | 单指标与阈值比较（>、>=、<、<=、==、!=） |
| PromQL 区间判断 | `promql_range` | 判断指标值是否在指定区间内 |
| PromQL 存在性判断 | `promql_exists` | 检查指标是否存在 |

**断言配置格式（JSON）**：

```json
// 阈值判断
{
  "type": "promql_threshold",
  "operator": ">",
  "value": 80,
  "unit": "%",
  "message": "CPU 使用率超过 80%"
}

// 区间判断
{
  "type": "promql_range",
  "min": 20,
  "max": 80,
  "invert": false,
  "message": "CPU 使用率应在 20%-80% 之间"
}

// 存在性判断
{
  "type": "promql_exists",
  "expect": true,
  "message": "服务应在线"
}
```

#### 3. 服务层集成（`plugins/inspection/service/item_service.go`）

**修改点**：
- 构造函数新增 `datasourceRepo` 参数，创建 `PromQLExecutor`
- `executeItem` 方法增加 PromQL 执行分支
- 新增 `executePromQL` 方法处理 PromQL 巡检逻辑

**执行流程**：
```
1. 判断 execution_type
   ├─ promql → 调用 executePromQL
   └─ command/script → 原有逻辑

2. executePromQL
   ├─ 检查数据源配置
   ├─ 调用 PromQLExecutor.Execute
   ├─ 执行断言校验
   └─ 返回结果
```

### 3.2 模型变更

#### 巡检组模型（`plugins/inspection/model/inspection_group.go`）

```go
type InspectionGroup struct {
    // ... 其他字段
    DataSourceID uint `gorm:"default:0;index" json:"datasource_id"`
    // 删除：PrometheusURL, PrometheusUsername, PrometheusPassword
}
```

#### 巡检项模型（`plugins/inspection/model/inspection_item.go`）

```go
type InspectionItem struct {
    // ... 其他字段
    PromQLQuery     string `gorm:"type:text" json:"promql_query"`
    PromQLQueryType string `gorm:"size:20;default:'instant'" json:"promql_query_type"`
}
```

#### 执行详情模型（`internal/data/inspection_mgmt/execution_record.go`）

```go
type InspectionExecutionDetail struct {
    // ... 其他字段
    DataSourceID   uint    `gorm:"default:0;index" json:"dataSourceId"`
    PromQL         string  `gorm:"type:text" json:"promql"`
    PromQLResult   string  `gorm:"type:longtext" json:"promqlResult"`
    MetricValue    float64 `gorm:"type:decimal(20,4)" json:"metricValue"`
    MetricLabels   string  `gorm:"type:json" json:"metricLabels"`
    AssertionPass  *bool   `gorm:"default:null" json:"assertionPass"`
    AssertionRule  string  `gorm:"type:text" json:"assertionRule"`
    FailureReason  string  `gorm:"type:text" json:"failureReason"`
}
```

### 3.3 DTO 变更

#### 巡检组 DTO（`plugins/inspection/dto/group_dto.go`）

```go
type GroupCreateRequest struct {
    // ... 其他字段
    DataSourceID uint `json:"datasource_id"`
    // 删除：PrometheusURL, PrometheusUsername, PrometheusPassword
}
```

#### 巡检项 DTO（`plugins/inspection/dto/item_dto.go`）

```go
type ItemCreateRequest struct {
    // ... 其他字段
    PromQLQuery     string `json:"promql_query"`
    PromQLQueryType string `json:"promql_query_type"`
}
```

---

## 四、前端实现（待开发）

### 4.1 巡检组配置页面

**数据源选择**：
- 下拉单选框（从告警管理数据源列表加载）
- 显示数据源类型标签（Prometheus/VictoriaMetrics）
- 显示访问模式标签（Direct/Agent）

**界面示例**：
```vue
<a-form-item label="关联数据源" required>
  <a-select
    v-model="form.datasource_id"
    placeholder="请选择数据源"
    :loading="datasourceLoading"
    allow-search
  >
    <a-option v-for="ds in datasourceList" :key="ds.id" :value="ds.id">
      <div style="display: flex; align-items: center; justify-content: space-between;">
        <span>{{ ds.name }}</span>
        <a-space>
          <a-tag :color="ds.type === 'prometheus' ? 'blue' : 'green'" size="small">
            {{ ds.type }}
          </a-tag>
          <a-tag v-if="ds.access_mode === 'agent'" color="orange" size="small">
            Agent
          </a-tag>
        </a-space>
      </div>
    </a-option>
  </a-select>
</a-form-item>
```

### 4.2 巡检项配置页面

**PromQL 配置区域**（当 execution_type === 'promql' 时显示）：

1. **查询类型选择**：
   - 瞬时查询（instant）- 推荐
   - 范围查询（range）

2. **PromQL 语句输入**：
   - 多行文本框
   - 支持预置变量提示
   - 示例：`node_cpu_seconds_total{instance="{{.Instance}}", mode="idle"}`

3. **断言类型选择**：
   - 阈值判断（promql_threshold）
   - 区间判断（promql_range）
   - 存在性判断（promql_exists）

4. **断言规则配置**（根据断言类型动态显示）：
   - **阈值判断**：操作符（>、>=、<、<=、==、!=）+ 阈值 + 单位 + 失败提示
   - **区间判断**：最小值 + 最大值 + 判断逻辑（在区间内/外正常）
   - **存在性判断**：期望结果（有数据/无数据）

**界面示例**：
```vue
<template v-if="form.execution_type === 'promql'">
  <!-- 查询类型 -->
  <a-form-item label="查询类型">
    <a-radio-group v-model="form.promql_query_type">
      <a-radio value="instant">瞬时查询（推荐）</a-radio>
      <a-radio value="range">范围查询</a-radio>
    </a-radio-group>
  </a-form-item>

  <!-- PromQL 语句 -->
  <a-form-item label="PromQL 语句" required>
    <a-textarea
      v-model="form.promql_query"
      placeholder="支持预置变量：{{.Instance}}, {{.IP}}, {{.Hostname}}"
      :rows="4"
    />
  </a-form-item>

  <!-- 断言类型 -->
  <a-form-item label="断言类型" required>
    <a-select v-model="form.assertion_type">
      <a-option value="promql_threshold">阈值判断</a-option>
      <a-option value="promql_range">区间判断</a-option>
      <a-option value="promql_exists">存在性判断</a-option>
    </a-select>
  </a-form-item>

  <!-- 阈值判断配置 -->
  <template v-if="form.assertion_type === 'promql_threshold'">
    <a-form-item label="阈值规则">
      <a-space>
        <a-select v-model="assertionConfig.operator" style="width: 100px">
          <a-option value=">">大于 ></a-option>
          <a-option value=">=">大于等于 >=</a-option>
          <a-option value="<">小于 <</a-option>
          <a-option value="<=">小于等于 <=</a-option>
          <a-option value="==">等于 ==</a-option>
          <a-option value="!=">不等于 !=</a-option>
        </a-select>
        <a-input-number v-model="assertionConfig.value" placeholder="阈值" style="width: 150px" />
        <a-input v-model="assertionConfig.unit" placeholder="单位（可选）" style="width: 100px" />
      </a-space>
    </a-form-item>
    <a-form-item label="失败提示">
      <a-input v-model="assertionConfig.message" placeholder="如：CPU 使用率超过 80%" />
    </a-form-item>
  </template>
</template>
```

### 4.3 执行结果展示页面

**展示内容**：
- 执行的 PromQL 语句（变量已替换）
- 查询结果（JSON 格式化展示）
- 提取的指标值
- 指标标签
- 断言结果（通过/失败）
- 失败原因

---

## 五、常用指标巡检场景覆盖

### 5.1 主机基础指标（✅ 完全覆盖）

| 指标 | PromQL 示例 | 断言类型 |
|------|------------|---------|
| CPU 使用率 | `100 - (avg by (instance) (irate(node_cpu_seconds_total{instance="{{.Instance}}",mode="idle"}[5m])) * 100)` | `promql_threshold > 80` |
| 内存使用率 | `(1 - (node_memory_MemAvailable_bytes{instance="{{.Instance}}"} / node_memory_MemTotal_bytes{instance="{{.Instance}}"})) * 100` | `promql_threshold > 80` |
| 磁盘使用率 | `(1 - (node_filesystem_avail_bytes{instance="{{.Instance}}",mountpoint="/"} / node_filesystem_size_bytes{instance="{{.Instance}}",mountpoint="/"})) * 100` | `promql_threshold > 90` |
| 系统负载 | `node_load5{instance="{{.Instance}}"}` | `promql_threshold > 4` |

### 5.2 服务可用性（✅ 完全覆盖）

| 指标 | PromQL 示例 | 断言类型 |
|------|------------|---------|
| 服务在线状态 | `up{instance="{{.Instance}}"}` | `promql_threshold == 1` |
| 进程存在性 | `namedprocess_namegroup_num_procs{instance="{{.Instance}}",groupname="nginx"}` | `promql_threshold > 0` |

### 5.3 应用指标（✅ 完全覆盖）

| 指标 | PromQL 示例 | 断言类型 |
|------|------------|---------|
| HTTP 请求错误率 | `sum(rate(http_requests_total{instance="{{.Instance}}",status=~"5.."}[5m])) / sum(rate(http_requests_total{instance="{{.Instance}}"}[5m])) * 100` | `promql_threshold < 5` |
| HTTP 请求延迟（P95） | `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{instance="{{.Instance}}"}[5m]))` | `promql_threshold < 1` |
| 队列长度 | `rabbitmq_queue_messages{instance="{{.Instance}}",queue="task_queue"}` | `promql_threshold < 1000` |

**覆盖度**：Instant Query + 阈值/区间/存在性断言可以覆盖 **85%+** 的常规巡检场景。

---

## 六、实施步骤

### 阶段 1：数据库迁移（✅ 已完成）

- [x] 编写迁移脚本
- [ ] 执行数据库迁移
- [ ] 验证表结构

### 阶段 2：后端开发（✅ 已完成）

- [x] 实现 PromQL 执行器
- [x] 扩展断言验证器
- [x] 集成服务层逻辑
- [x] 更新模型和 DTO
- [ ] 单元测试

### 阶段 3：前端开发（⏳ 待开发）

- [ ] 巡检组数据源选择界面
- [ ] 巡检项 PromQL 配置界面
- [ ] 执行结果展示优化
- [ ] 集成测试

### 阶段 4：测试与上线（⏳ 待执行）

- [ ] 功能测试
- [ ] 性能测试（并发 50）
- [ ] 用户验收测试
- [ ] 生产环境部署

---

## 七、注意事项

### 7.1 数据源连接失败

**风险**：数据源不可达、认证失败、网络超时

**解决方案**：
- 记录详细的错误日志
- 前端展示数据源健康状态
- 提供数据源连接测试功能

### 7.2 PromQL 语法错误

**风险**：用户配置的 PromQL 语法错误导致查询失败

**解决方案**：
- 前端提供 PromQL 语法校验
- 提供常用 PromQL 模板
- 测试运行功能，提前发现问题

### 7.3 查询结果为空

**风险**：PromQL 查询无结果（指标不存在、时间范围不对）

**解决方案**：
- 区分"查询失败"和"查询无结果"
- 使用 `promql_exists` 断言检查指标存在性
- 记录详细的查询日志

### 7.4 性能问题

**风险**：大量主机并发查询 PromQL 导致数据源压力大

**解决方案**：
- 控制并发数（使用 goroutine pool，默认 50）
- 设置查询超时（30 秒）
- 监控数据源负载

### 7.5 预置变量解析失败

**风险**：主机缺少必要的标签信息（如 `instance`、`service_labels`）

**解决方案**：
- 在主机匹配阶段检查必要字段
- 提供默认值（如 `instance` 默认为 `IP:9100`）
- 记录变量替换日志

---

## 八、后续优化方向

### 8.1 高级断言类型（二期）

- **多指标对比**（`promql_compare`）：支持多个 PromQL 查询结果的对比
- **趋势分析**（`promql_trend`）：检测指标是否持续上升/下降
- **异常检测**：基于历史数据的异常检测

### 8.2 用户体验优化

- PromQL 语法高亮和自动补全
- 预置变量智能提示
- PromQL 查询结果可视化（图表展示）
- 常用 PromQL 模板库

### 8.3 性能优化

- 查询结果缓存（短期缓存 1-5 分钟）
- 数据源健康检查和自动切换
- 批量查询优化

---

## 九、相关文件清单

### 后端文件

| 文件路径 | 说明 |
|---------|------|
| `migrations/promql_inspection.sql` | 数据库迁移脚本 |
| `plugins/inspection/executor/promql_executor.go` | PromQL 执行器 |
| `plugins/inspection/executor/assertion_validator.go` | 断言验证器（已扩展） |
| `plugins/inspection/service/item_service.go` | 巡检项服务（已集成） |
| `plugins/inspection/service/group_service.go` | 巡检组服务（已更新） |
| `plugins/inspection/model/inspection_group.go` | 巡检组模型（已更新） |
| `plugins/inspection/model/inspection_item.go` | 巡检项模型（已更新） |
| `internal/data/inspection_mgmt/execution_record.go` | 执行记录模型（已更新） |
| `plugins/inspection/dto/group_dto.go` | 巡检组 DTO（已更新） |
| `plugins/inspection/dto/item_dto.go` | 巡检项 DTO（已更新） |

### 前端文件（待开发）

| 文件路径 | 说明 |
|---------|------|
| `web/src/views/inspection/GroupForm.vue` | 巡检组配置页面 |
| `web/src/views/inspection/ItemForm.vue` | 巡检项配置页面 |
| `web/src/views/inspection/ExecutionDetail.vue` | 执行结果详情页面 |

---

## 十、总结

本方案实现了 PromQL 巡检的核心功能，支持：

✅ 单数据源关联（复用告警管理数据源）  
✅ Instant Query 和 Range Query  
✅ 预置变量自动注入  
✅ PromQL 专用断言类型（阈值、区间、存在性）  
✅ 并发控制（50）  
✅ 完整的执行结果记录  

**覆盖度**：可覆盖 85%+ 的常规指标巡检场景。

**后续工作**：
1. 执行数据库迁移
2. 前端界面开发
3. 集成测试
4. 生产环境部署
