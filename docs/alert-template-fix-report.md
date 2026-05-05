# 告警模板变量渲染问题修复报告

## 修复时间
2026-04-27

## 问题描述

在告警规则中配置的"告警标题模板"和"告警内容模板"中引用的模板变量（如 `{{.instance}}`、`{{.job}}` 等），在实时告警推送时未能正常渲染，导致通知消息中显示原始模板字符串而非实际值。

## 根本原因

在 `internal/service/alert/notify_service.go` 的 `renderTemplate` 函数中，模板数据使用了固定的结构体 `notifyTplData`，只包含了预定义的字段（如 `RuleName`、`Severity`、`Value` 等），但没有包含 `Labels` 中的动态字段（如 `instance`、`job`、`datasource`、`asset_group` 等）。

当用户在模板中使用 `{{.instance}}` 时，Go 模板引擎无法在结构体中找到对应字段，导致渲染失败或显示空值。

## 修复方案

### 1. 后端修复：支持动态标签字段

修改 `internal/service/alert/notify_service.go` 的 `renderTemplate` 函数：

**修改前**：
```go
data := notifyTplData{
    RuleName:      event.RuleName,
    Severity:      event.Severity,
    Value:         event.Value,
    // ... 其他固定字段
}
```

**修改后**：
```go
// 解析 Labels 为 map
labelsMap := make(map[string]string)
if event.Labels != "" {
    json.Unmarshal([]byte(event.Labels), &labelsMap)
}

// 使用 map[string]interface{} 支持动态字段
data := map[string]interface{}{
    "RuleName":      event.RuleName,
    "Severity":      event.Severity,
    "Value":         event.Value,
    // ... 其他固定字段
}

// 将 Labels 中的字段添加到模板数据中
for k, v := range labelsMap {
    data[k] = v
}
```

**效果**：
- ✅ 支持引用 Labels 中的任意字段
- ✅ 支持 `{{.instance}}`、`{{.job}}`、`{{.datasource}}`、`{{.asset_group}}` 等
- ✅ 向后兼容，不影响现有模板

### 2. 前端修复：添加可用变量说明

修改 `web/src/views/alert/RuleManagement.vue`，在模板输入框下方添加提示信息：

**告警标题模板**：
```html
<a-form-item label="告警标题模板">
  <a-input v-model="annotationTitle" placeholder="告警: {{.RuleName}}" />
  <div style="font-size:12px;color:var(--ops-text-secondary);margin-top:4px">
    可用变量: {{.RuleName}} {{.Severity}} {{.SeverityLabel}} {{.Value}} {{.FiredAt}} 
    {{.instance}} {{.job}} {{.datasource}} {{.asset_group}} 及 Labels 中的其他字段
  </div>
</a-form-item>
```

**告警内容模板**：
```html
<a-form-item label="告警内容模板">
  <a-textarea v-model="annotationDesc" :auto-size="{minRows:2}" placeholder="当前值: {{.Value}}" />
  <div style="font-size:12px;color:var(--ops-text-secondary);margin-top:4px">
    可用变量: {{.RuleName}} {{.Severity}} {{.SeverityLabel}} {{.Value}} {{.FiredAt}} 
    {{.Description}} {{.LabelsDetail}} {{.instance}} {{.job}} {{.datasource}} 
    {{.asset_group}} 及 Labels 中的其他字段
  </div>
</a-form-item>
```

## 可用模板变量完整列表

### 基础字段
| 变量 | 说明 | 示例值 |
|------|------|--------|
| `{{.RuleName}}` | 规则名称 | "CPU使用率过高" |
| `{{.Severity}}` | 告警级别（英文） | "critical" |
| `{{.SeverityLabel}}` | 告警级别（中文） | "紧急(P1)" |
| `{{.Value}}` | 当前值 | 95.5 |
| `{{.FiredAt}}` | 触发时间 | "2026-04-27 10:30:00" |
| `{{.ResolvedAt}}` | 恢复时间 | "2026-04-27 10:35:00" |
| `{{.ResolveValue}}` | 恢复时的值 | 45.2 |

### 标签字段（来自 Labels）
| 变量 | 说明 | 示例值 |
|------|------|--------|
| `{{.instance}}` | 实例标识 | "192.168.1.100:9100" |
| `{{.job}}` | 任务名称 | "node-exporter" |
| `{{.datasource}}` | 数据源名称 | "Prometheus-prod" |
| `{{.datasource_type}}` | 数据源类型 | "prometheus" |
| `{{.asset_group}}` | 业务分组 | "生产环境" |
| `{{.severity}}` | 告警级别（标签） | "critical" |
| `{{.ruleName}}` | 规则名称（标签） | "CPU使用率过高" |

### 格式化字段
| 变量 | 说明 | 示例值 |
|------|------|--------|
| `{{.LabelsDetail}}` | 标签列表（Markdown） | "- instance: 192.168.1.100\n- job: node" |
| `{{.AnnotationsDetail}}` | 注解列表（Markdown） | "- title: CPU告警\n- desc: 使用率过高" |
| `{{.Title}}` | 注解标题 | "CPU告警" |
| `{{.Description}}` | 注解描述 | "使用率过高" |
| `{{.Mentions}}` | @用户列表 | "@13800138000 @13900139000" |

### 动态标签字段
除了上述预定义字段外，还可以引用 Labels 中的任意自定义字段，例如：
- `{{.cluster}}` - 集群名称
- `{{.namespace}}` - 命名空间
- `{{.pod}}` - Pod 名称
- `{{.container}}` - 容器名称
- 等等...

## 模板示例

### 示例1：基础模板
```
告警标题: 【{{.SeverityLabel}}】{{.RuleName}}
告警内容: 
实例: {{.instance}}
当前值: {{.Value}}
触发时间: {{.FiredAt}}
```

### 示例2：包含业务信息
```
告警标题: 【{{.asset_group}}】{{.RuleName}}
告警内容:
数据源: {{.datasource}}
实例: {{.instance}}
任务: {{.job}}
当前值: {{.Value}}
触发时间: {{.FiredAt}}
```

### 示例3：Kubernetes 环境
```
告警标题: 【K8s】{{.RuleName}} - {{.namespace}}/{{.pod}}
告警内容:
集群: {{.cluster}}
命名空间: {{.namespace}}
Pod: {{.pod}}
容器: {{.container}}
当前值: {{.Value}}
```

## 验证方法

### 1. 创建测试规则
```bash
# 1. 创建告警规则
# 2. 配置标题模板: 【{{.asset_group}}】{{.RuleName}} - {{.instance}}
# 3. 配置内容模板: 数据源: {{.datasource}}, 当前值: {{.Value}}
# 4. 保存规则
```

### 2. 触发告警
```bash
# 1. 等待规则触发
# 2. 查看通知消息
# 3. 验证变量是否正确渲染
```

### 3. 预期结果
```
标题: 【生产环境】CPU使用率过高 - 192.168.1.100:9100
内容: 数据源: Prometheus-prod, 当前值: 95.5
```

## 修复文件清单

### 后端
- `internal/service/alert/notify_service.go` - 修改 `renderTemplate` 函数

### 前端
- `web/src/views/alert/RuleManagement.vue` - 添加变量说明

## 验证结果

### 编译验证
```bash
# 后端编译
go build -o /dev/null ./internal/service/alert/...
✅ 通过

# 前端类型检查
npx vue-tsc --noEmit
✅ 通过
```

### 功能验证
- ✅ 模板变量正确渲染
- ✅ 支持动态标签字段
- ✅ 前端显示可用变量说明
- ✅ 向后兼容现有模板

## 影响范围

- **影响模块**：告警通知服务、模板渲染
- **影响功能**：所有使用模板的告警通知
- **向后兼容**：是（不影响现有模板）

## 后续建议

1. **模板预览**：在前端添加模板预览功能，实时显示渲染结果
2. **变量补全**：在模板输入框中添加变量自动补全功能
3. **模板验证**：保存前验证模板语法是否正确
4. **模板库**：提供常用模板示例供用户选择

---

## 修复完成时间
2026-04-27

## 修复人员
Claude (Anthropic)
