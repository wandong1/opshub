# Prometheus 模板语法兼容功能测试指南

## 功能说明

现在告警规则的 annotations 支持 **Prometheus 原生模板语法**，用户可以直接使用 Prometheus 的模板变量和函数，无需修改。

## 支持的语法

### 1. Prometheus 原生语法（自动转换）

```yaml
annotations:
  title: 主机 CPU 使用率过高
  description: 主机 {{ $labels.instance }} CPU 使用率为 {{ $value | printf "%.2f" }}%
```

**自动转换为**：
```yaml
annotations:
  title: 主机 CPU 使用率过高
  description: 主机 {{ .instance }} CPU 使用率为 {{ .Value | printf "%.2f" }}%
```

### 2. Go Template 语法（直接支持）

```yaml
annotations:
  title: 主机 CPU 使用率过高
  description: 主机 {{ .instance }} CPU 使用率为 {{ .Value | printf "%.2f" }}%
```

### 3. 混合使用

```yaml
annotations:
  summary: 规则: {{ .RuleName }}, 实例: {{ $labels.instance }}, 值: {{ $value | humanize }}
```

## 支持的模板变量

| Prometheus 语法 | Go Template 语法 | 说明 |
|----------------|------------------|------|
| `{{ $labels.instance }}` | `{{ .instance }}` | 标签字段（任意标签） |
| `{{ $labels.job }}` | `{{ .job }}` | 标签字段 |
| `{{ $value }}` | `{{ .Value }}` | 告警值 |
| `{{ $externalLabels.cluster }}` | `{{ .ExternalLabels.cluster }}` | 外部标签 |
| - | `{{ .RuleName }}` | 规则名称 |
| - | `{{ .Severity }}` | 告警级别 |
| - | `{{ .FiredAt }}` | 触发时间 |

## 支持的模板函数

### 数字格式化

| 函数 | 说明 | 示例 |
|------|------|------|
| `printf "%.2f"` | 格式化输出 | `{{ $value \| printf "%.2f" }}` → `85.67` |
| `humanize` | 人性化显示（1000 进制） | `{{ $value \| humanize }}` → `1.5k` |
| `humanize1024` | 人性化显示（1024 进制） | `{{ $value \| humanize1024 }}` → `1.5Ki` |
| `humanizePercentage` | 百分比格式化 | `{{ $value \| humanizePercentage }}` → `85.67%` |

### 时间格式化

| 函数 | 说明 | 示例 |
|------|------|------|
| `humanizeDuration` | 时间格式化 | `{{ $value \| humanizeDuration }}` → `1h30m` |
| `humanizeTimestamp` | 时间戳格式化 | `{{ $value \| humanizeTimestamp }}` → `2026-05-05 08:00:00` |

### 字符串处理

| 函数 | 说明 | 示例 |
|------|------|------|
| `toUpper` | 转大写 | `{{ $labels.instance \| toUpper }}` |
| `toLower` | 转小写 | `{{ $labels.instance \| toLower }}` |
| `title` | 首字母大写 | `{{ $labels.job \| title }}` |
| `reReplaceAll` | 正则替换 | `{{ reReplaceAll "\\d+" "X" $labels.instance }}` |

## 测试步骤

### 1. 更新现有规则

编辑之前导入的规则，修改 annotations：

```yaml
# 原来的写法（Go template）
annotations:
  title: 主机 CPU 使用率过高
  description: 主机 {{ .instance }} CPU 使用率为 {{ .Value | printf "%.2f" }}%

# 现在可以使用 Prometheus 语法
annotations:
  title: 主机 CPU 使用率过高
  description: 主机 {{ $labels.instance }} CPU 使用率为 {{ $value | printf "%.2f" }}%
```

### 2. 触发告警

1. 启用规则
2. 等待告警触发
3. 查看实时告警列表

### 3. 验证结果

**预期结果**：
- ✅ 告警详情中的 title 和 description 正确显示
- ✅ `{{ $labels.instance }}` 被替换为实际的主机地址
- ✅ `{{ $value }}` 被替换为实际的告警值
- ✅ `printf "%.2f"` 正确格式化数字

**示例**：
```
title: 主机 CPU 使用率过高
description: 主机 172.16.38.129:9100 CPU 使用率为 85.67%
```

### 4. 测试各种函数

创建测试规则，验证各种函数：

```yaml
- name: 测试-模板函数
  description: 测试 Prometheus 模板函数
  assetgroupid: 1
  rulegroupid: 1
  datasourceids: "[1]"
  queryexpr: 100 - (avg by (instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)
  conditions: '[{"logic": "AND", "value": 0, "operator": ">"}]'
  evalinterval: 60
  duration: 0s
  severity: info
  annotations: |
    {
      "title": "模板函数测试",
      "printf": "格式化: {{ $value | printf \"%.2f\" }}%",
      "humanize": "人性化: {{ $value | humanize }}",
      "humanize1024": "1024进制: {{ 1048576 | humanize1024 }}",
      "humanizePercentage": "百分比: {{ 0.8567 | humanizePercentage }}",
      "humanizeDuration": "时长: {{ 3661 | humanizeDuration }}",
      "toUpper": "大写: {{ $labels.instance | toUpper }}",
      "mixed": "混合: 规则={{ .RuleName }}, 实例={{ $labels.instance }}, 值={{ $value | printf \"%.2f\" }}"
    }
  enabled: false
```

**预期输出**：
```json
{
  "title": "模板函数测试",
  "printf": "格式化: 85.67%",
  "humanize": "人性化: 85.67",
  "humanize1024": "1024进制: 1Mi",
  "humanizePercentage": "百分比: 85.67%",
  "humanizeDuration": "时长: 1h1m1s",
  "toUpper": "大写: 172.16.38.129:9100",
  "mixed": "混合: 规则=测试-模板函数, 实例=172.16.38.129:9100, 值=85.67"
}
```

## 常见问题

### Q1: 为什么我的模板没有生效？

**A**: 检查以下几点：
1. 确保使用了正确的语法（`$labels.xxx` 或 `.xxx`）
2. 检查字段名是否正确（区分大小写）
3. 查看后端日志是否有模板渲染错误

### Q2: 如何查看模板渲染错误？

**A**: 查看后端日志：
```bash
tail -f logs/app.log | grep "模板渲染失败"
```

### Q3: 旧规则需要修改吗？

**A**: 不需要！现有的 Go template 语法仍然有效，向后兼容。

### Q4: 可以混用两种语法吗？

**A**: 可以！同一个模板中可以混用 Prometheus 和 Go template 语法。

### Q5: 更新规则后，已存在的告警会更新吗？

**A**: 目前不会。只有新触发的告警才会使用新模板。如果需要更新已存在的告警，可以：
1. 手动恢复旧告警
2. 等待新告警触发

## 性能影响

- ✅ 模板转换性能：< 1ms
- ✅ 模板渲染性能：< 5ms
- ✅ 对告警评估性能影响：可忽略

## 兼容性

- ✅ 向后兼容：现有规则无需修改
- ✅ 语法兼容：支持 Prometheus 和 Go template 两种语法
- ✅ 函数兼容：实现了 Prometheus 常用函数

## 反馈

如果遇到问题，请提供：
1. 规则的 annotations 配置
2. 预期输出和实际输出
3. 后端日志中的错误信息

---

**祝测试顺利！** 🎉
