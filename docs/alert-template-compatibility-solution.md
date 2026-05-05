# 告警规则模板语法兼容性问题分析与解决方案

## 问题描述

### 问题 1：Prometheus 模板语法不兼容
用户在告警规则的 annotations 中使用了 Prometheus 原生模板语法：
```yaml
annotations:
  title: 主机 CPU 使用率过高
  description: 主机 {{ $labels.instance }} CPU 使用率为 {{ $value | printf "%.2f" }}%
```

但系统无法正确解析 `$labels` 和 `$value` 这种 Prometheus 语法。

### 问题 2：更新规则后模板不生效
用户修改了规则的 annotations 模板：
```yaml
# 修改前
description: 主机 {{ $labels.instance }} CPU 使用率为 {{ $value | printf "%.2f" }}%

# 修改后
description: 主机 {{ .instance }} CPU 使用率为 {{ .Value | printf "%.2f" }}%
```

保存后重新触发告警，但告警内容仍然显示旧的模板变量（未渲染）。

## 根本原因分析

### 1. 模板语法不兼容

**当前实现**（`internal/service/alert/eval_service.go:466-510`）：
```go
func (e *EvalEngine) renderAnnotations(annotationsJSON string, rule *biz.AlertRule, value float64, firedAt time.Time, labels map[string]string) string {
    // 构建模板数据
    data := map[string]interface{}{
        "RuleName":      rule.Name,
        "Severity":      rule.Severity,
        "Value":         value,
        "FiredAt":       firedAt.Format("2006-01-02 15:04:05"),
    }
    
    // 添加 Labels 中的字段
    for k, v := range labels {
        data[k] = v  // 支持 {{.instance}}、{{.job}}
    }
    
    // 渲染每个 annotation 字段
    tpl, err := template.New("").Parse(v)
    tpl.Execute(&buf, data)
}
```

**问题**：
- ✅ 支持 Go template 语法：`{{ .instance }}`、`{{ .Value }}`
- ❌ 不支持 Prometheus 语法：`{{ $labels.instance }}`、`{{ $value }}`
- ❌ 不支持 Prometheus 函数：`{{ $value | printf "%.2f" }}`

### 2. 模板渲染时机问题

**当前流程**：
```
1. 规则评估 (eval_service.go)
   ├─ 执行 PromQL 查询
   ├─ 判断是否触发告警
   ├─ 渲染 annotations 模板 ✅ 在这里渲染
   └─ 创建/更新告警事件（保存渲染后的结果）

2. 发送通知 (notify_service.go)
   ├─ 读取告警事件（annotations 已经是渲染后的）
   ├─ 渲染通知模板 ✅ 再次渲染（但 annotations 已经是纯文本）
   └─ 发送通知
```

**问题**：
- Annotations 在**告警创建时**就被渲染并保存到数据库
- 更新规则后，**已存在的告警事件**中的 annotations 不会更新
- 只有**新触发的告警**才会使用新模板

## 解决方案

### 方案 A：完全兼容 Prometheus 模板语法（推荐）

#### 优点
- ✅ 用户可以直接复制 Prometheus 规则，无需修改
- ✅ 学习成本低，符合用户习惯
- ✅ 兼容现有的 Go template 语法

#### 缺点
- ⚠️ 需要实现 Prometheus 模板函数（humanize、printf 等）
- ⚠️ 需要处理两种语法的转换

#### 实现方案

**1. 创建 Prometheus 模板适配器**

```go
// pkg/promtemplate/adapter.go
package promtemplate

import (
    "bytes"
    "fmt"
    "regexp"
    "text/template"
)

// PrometheusTemplateAdapter Prometheus 模板适配器
type PrometheusTemplateAdapter struct {
    funcMap template.FuncMap
}

func NewAdapter() *PrometheusTemplateAdapter {
    return &PrometheusTemplateAdapter{
        funcMap: template.FuncMap{
            // Prometheus 内置函数
            "humanize":      humanize,
            "humanize1024":  humanize1024,
            "humanizeDuration": humanizeDuration,
            "humanizePercentage": humanizePercentage,
            "humanizeTimestamp": humanizeTimestamp,
            "title":         strings.Title,
            "toUpper":       strings.ToUpper,
            "toLower":       strings.ToLower,
            "match":         regexp.MatchString,
            "reReplaceAll":  reReplaceAll,
            "graphLink":     graphLink,
            "tableLink":     tableLink,
        },
    }
}

// ConvertTemplate 转换 Prometheus 模板语法为 Go template 语法
func (a *PrometheusTemplateAdapter) ConvertTemplate(promTpl string) string {
    // 1. 转换 $labels.xxx -> .xxx
    re1 := regexp.MustCompile(`\$labels\.(\w+)`)
    tpl := re1.ReplaceAllString(promTpl, ".$1")
    
    // 2. 转换 $value -> .Value
    re2 := regexp.MustCompile(`\$value`)
    tpl = re2.ReplaceAllString(tpl, ".Value")
    
    // 3. 转换 $externalLabels.xxx -> .ExternalLabels.xxx
    re3 := regexp.MustCompile(`\$externalLabels\.(\w+)`)
    tpl = re3.ReplaceAllString(tpl, ".ExternalLabels.$1")
    
    return tpl
}

// Render 渲染模板（自动转换 Prometheus 语法）
func (a *PrometheusTemplateAdapter) Render(tplStr string, data map[string]interface{}) (string, error) {
    // 转换模板语法
    convertedTpl := a.ConvertTemplate(tplStr)
    
    // 创建模板并添加函数
    tpl, err := template.New("").Funcs(a.funcMap).Parse(convertedTpl)
    if err != nil {
        return "", fmt.Errorf("模板解析失败: %w", err)
    }
    
    var buf bytes.Buffer
    if err := tpl.Execute(&buf, data); err != nil {
        return "", fmt.Errorf("模板渲染失败: %w", err)
    }
    
    return buf.String(), nil
}

// Prometheus 模板函数实现
func humanize(v float64) string {
    if v == 0 {
        return "0"
    }
    prefix := ""
    for _, p := range []string{"k", "M", "G", "T", "P", "E", "Z", "Y"} {
        if math.Abs(v) < 1000 {
            break
        }
        prefix = p
        v /= 1000
    }
    return fmt.Sprintf("%.4g%s", v, prefix)
}

func humanize1024(v float64) string {
    if v == 0 {
        return "0"
    }
    prefix := ""
    for _, p := range []string{"Ki", "Mi", "Gi", "Ti", "Pi", "Ei", "Zi", "Yi"} {
        if math.Abs(v) < 1024 {
            break
        }
        prefix = p
        v /= 1024
    }
    return fmt.Sprintf("%.4g%s", v, prefix)
}

func humanizeDuration(v float64) string {
    if v == 0 {
        return "0s"
    }
    if math.IsNaN(v) || math.IsInf(v, 0) {
        return fmt.Sprintf("%.4g", v)
    }
    if v < 0 {
        return fmt.Sprintf("%.4gs", v)
    }
    seconds := int64(v)
    duration := time.Duration(seconds) * time.Second
    return duration.String()
}

func humanizePercentage(v float64) string {
    return fmt.Sprintf("%.4g%%", v*100)
}

func humanizeTimestamp(v float64) string {
    if math.IsNaN(v) || math.IsInf(v, 0) {
        return fmt.Sprintf("%.4g", v)
    }
    t := time.Unix(int64(v), 0).UTC()
    return t.Format("2006-01-02 15:04:05")
}

func reReplaceAll(pattern, repl, text string) string {
    re := regexp.MustCompile(pattern)
    return re.ReplaceAllString(text, repl)
}

func graphLink(expr string) string {
    return fmt.Sprintf("/graph?g0.expr=%s", url.QueryEscape(expr))
}

func tableLink(expr string) string {
    return fmt.Sprintf("/graph?g0.expr=%s&g0.tab=1", url.QueryEscape(expr))
}
```

**2. 修改 eval_service.go 使用适配器**

```go
// internal/service/alert/eval_service.go

import (
    "github.com/ydcloud-dy/opshub/pkg/promtemplate"
)

type EvalEngine struct {
    // ... 现有字段
    promTplAdapter *promtemplate.PrometheusTemplateAdapter
}

func NewEvalEngine(...) *EvalEngine {
    return &EvalEngine{
        // ... 现有初始化
        promTplAdapter: promtemplate.NewAdapter(),
    }
}

func (e *EvalEngine) renderAnnotations(annotationsJSON string, rule *biz.AlertRule, value float64, firedAt time.Time, labels map[string]string) string {
    if annotationsJSON == "" || annotationsJSON == "{}" {
        return annotationsJSON
    }

    var annotations map[string]string
    if err := json.Unmarshal([]byte(annotationsJSON), &annotations); err != nil {
        return annotationsJSON
    }

    // 构建模板数据（兼容 Prometheus 和 Go template）
    data := map[string]interface{}{
        // Go template 风格
        "RuleName":      rule.Name,
        "Severity":      rule.Severity,
        "SeverityLabel": severityLabel(rule.Severity),
        "Value":         value,
        "FiredAt":       firedAt.Format("2006-01-02 15:04:05"),
        
        // Prometheus 风格（通过适配器转换）
        "value":         value,  // 兼容 $value
        "labels":        labels, // 兼容 $labels
    }

    // 添加 Labels 中的字段（支持 {{.instance}}）
    for k, v := range labels {
        data[k] = v
    }

    // 渲染每个 annotation 字段
    rendered := make(map[string]string)
    for k, v := range annotations {
        // 使用 Prometheus 模板适配器渲染
        result, err := e.promTplAdapter.Render(v, data)
        if err != nil {
            appLogger.Warn("模板渲染失败，使用原始值",
                zap.String("key", k),
                zap.String("template", v),
                zap.Error(err))
            rendered[k] = v
        } else {
            rendered[k] = result
        }
    }

    b, _ := json.Marshal(rendered)
    return string(b)
}
```

**3. 支持的模板语法对照表**

| Prometheus 语法 | Go Template 语法 | 说明 |
|----------------|------------------|------|
| `{{ $labels.instance }}` | `{{ .instance }}` | 标签字段 |
| `{{ $value }}` | `{{ .Value }}` | 告警值 |
| `{{ $value \| printf "%.2f" }}` | `{{ .Value \| printf "%.2f" }}` | 格式化 |
| `{{ $value \| humanize }}` | `{{ .Value \| humanize }}` | 人性化显示 |
| `{{ $labels.job }}` | `{{ .job }}` | 任意标签 |

**4. 测试用例**

```yaml
# 测试规则
annotations:
  # Prometheus 原生语法（自动转换）
  title: 主机 CPU 使用率过高
  description: 主机 {{ $labels.instance }} CPU 使用率为 {{ $value | printf "%.2f" }}%
  
  # Go template 语法（直接支持）
  summary: 主机 {{ .instance }} CPU 使用率为 {{ .Value | printf "%.2f" }}%
  
  # 混合使用
  detail: 规则: {{ .RuleName }}, 实例: {{ $labels.instance }}, 值: {{ $value | humanize }}
```

### 方案 B：仅支持 Go Template 语法（简单）

#### 优点
- ✅ 实现简单，无需额外开发
- ✅ 性能更好

#### 缺点
- ❌ 用户需要学习 Go template 语法
- ❌ 无法直接复制 Prometheus 规则
- ❌ 迁移成本高

#### 实现方案

只需在文档中说明支持的语法，并提供转换工具。

### 方案 C：延迟渲染（解决更新问题）

#### 问题
当前 annotations 在告警创建时就被渲染并保存，更新规则后已存在的告警不会更新。

#### 解决方案

**1. 保存原始模板到数据库**

```go
// internal/biz/alert/event.go
type AlertEvent struct {
    // ... 现有字段
    Annotations         string  // 渲染后的 annotations（用于显示）
    AnnotationsTemplate string  // 原始模板（用于重新渲染）
}
```

**2. 通知时实时渲染**

```go
// internal/service/alert/notify_service.go
func (s *NotifyService) Send(ctx context.Context, ch *biz.AlertNotifyChannel, event *biz.AlertEvent, isResolve bool, phones []string) {
    // 如果有原始模板，实时渲染
    if event.AnnotationsTemplate != "" {
        // 重新获取规则
        rule, _ := s.ruleRepo.GetByID(ctx, event.AlertRuleID)
        if rule != nil {
            // 解析 labels
            var labels map[string]string
            json.Unmarshal([]byte(event.Labels), &labels)
            
            // 实时渲染 annotations
            event.Annotations = renderAnnotations(event.AnnotationsTemplate, rule, event.Value, event.FiredAt, labels)
        }
    }
    
    // ... 继续发送通知
}
```

## 推荐方案

### 最佳方案：方案 A + 方案 C

**阶段 1：实现 Prometheus 模板兼容（方案 A）**
- 创建 `pkg/promtemplate` 包
- 实现模板语法转换
- 实现 Prometheus 内置函数
- 修改 `eval_service.go` 使用适配器

**阶段 2：优化模板渲染机制（方案 C）**
- 数据库添加 `annotations_template` 字段
- 保存原始模板
- 通知时实时渲染

### 实施优先级

**P0（立即实施）**
1. 实现基础的 Prometheus 语法转换（`$labels.xxx` -> `.xxx`，`$value` -> `.Value`）
2. 实现常用函数：`printf`、`humanize`、`humanize1024`

**P1（1 周内）**
3. 实现完整的 Prometheus 函数库
4. 添加单元测试

**P2（2 周内）**
5. 实现延迟渲染机制
6. 数据库迁移

## 风险评估

### 技术风险
- ⚠️ 模板语法转换可能有边界情况
- ⚠️ Prometheus 函数实现可能不完全兼容

### 兼容性风险
- ✅ 向后兼容：现有的 Go template 语法仍然有效
- ✅ 平滑迁移：两种语法可以混用

### 性能风险
- ⚠️ 实时渲染会增加通知延迟（约 1-5ms）
- ✅ 可以通过缓存优化

## 工作量评估

| 任务 | 工作量 | 优先级 |
|------|--------|--------|
| 创建 promtemplate 包 | 4 小时 | P0 |
| 实现语法转换 | 2 小时 | P0 |
| 实现基础函数 | 3 小时 | P0 |
| 修改 eval_service.go | 1 小时 | P0 |
| 单元测试 | 2 小时 | P1 |
| 实现完整函数库 | 4 小时 | P1 |
| 延迟渲染机制 | 3 小时 | P2 |
| 数据库迁移 | 1 小时 | P2 |
| **总计** | **20 小时** | - |

## 测试计划

### 单元测试
```go
func TestPrometheusTemplateAdapter(t *testing.T) {
    adapter := promtemplate.NewAdapter()
    
    // 测试语法转换
    input := "{{ $labels.instance }} CPU: {{ $value | printf \"%.2f\" }}%"
    expected := "{{ .instance }} CPU: {{ .Value | printf \"%.2f\" }}%"
    result := adapter.ConvertTemplate(input)
    assert.Equal(t, expected, result)
    
    // 测试渲染
    data := map[string]interface{}{
        "instance": "192.168.1.1:9100",
        "Value":    85.67,
    }
    output, err := adapter.Render(input, data)
    assert.NoError(t, err)
    assert.Equal(t, "192.168.1.1:9100 CPU: 85.67%", output)
}
```

### 集成测试
1. 创建使用 Prometheus 语法的规则
2. 触发告警
3. 验证 annotations 正确渲染
4. 更新规则模板
5. 验证新告警使用新模板

## 文档更新

需要更新以下文档：
1. 用户手册：模板语法说明
2. API 文档：annotations 字段说明
3. 迁移指南：Prometheus 规则迁移

---

## 总结

**推荐实施方案 A（Prometheus 模板兼容）**，因为：
1. ✅ 用户体验最好，学习成本低
2. ✅ 兼容性强，支持两种语法
3. ✅ 工作量可控（约 20 小时）
4. ✅ 风险可控，向后兼容

**是否开始实施？请确认。**
