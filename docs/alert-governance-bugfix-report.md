# 告警治理功能 - 问题修复报告

## 修复时间
2026-04-27

## 修复的问题

### 问题1：告警事件缺少预设标签字段

**问题描述**：
告警规则触发生成实时告警后，未自动携带 `severity`、`ruleName` 等预设标签字段，同时缺少数据源和业务分组标签，无法支撑后续基于标签维度的告警治理和统计分析。

**根本原因**：
在 `eval_service.go` 的 `createFiringEvent` 方法中，只是简单合并了规则标签和指标标签，没有补充预设标签和业务标签。

**修复方案**：
修改 `internal/service/alert/eval_service.go` 的 `createFiringEvent` 方法，在创建告警事件时统一补充以下标签：

1. **补齐缺失的原有标签**：
   - `severity` - 告警级别
   - `ruleName` - 规则名称

2. **新增自定义标签**：
   - `datasource` - 数据源名称
   - `datasource_type` - 数据源类型（prometheus/victoriametrics等）
   - `asset_group` - 业务分组名称

**修复代码**：
```go
func (e *EvalEngine) createFiringEvent(ctx context.Context, rule *biz.AlertRule, fingerprint string, value float64, firedAt time.Time, metricLabels map[string]string) {
	// 补充标签：severity、ruleName、datasource、assetGroup
	enrichedLabels := make(map[string]string)

	// 1. 先合并规则标签和指标标签
	var ruleLabels map[string]string
	if rule.Labels != "" && rule.Labels != "{}" {
		json.Unmarshal([]byte(rule.Labels), &ruleLabels)
	}
	for k, v := range ruleLabels {
		enrichedLabels[k] = v
	}
	for k, v := range metricLabels {
		enrichedLabels[k] = v
	}

	// 2. 补充预设标签（如果不存在）
	if _, exists := enrichedLabels["severity"]; !exists {
		enrichedLabels["severity"] = rule.Severity
	}
	if _, exists := enrichedLabels["ruleName"]; !exists {
		enrichedLabels["ruleName"] = rule.Name
	}

	// 3. 补充数据源标签
	if rule.DataSourceID > 0 {
		ds, err := e.dsRepo.GetByID(ctx, rule.DataSourceID)
		if err == nil && ds != nil {
			enrichedLabels["datasource"] = ds.Name
			enrichedLabels["datasource_type"] = ds.Type
		}
	}

	// 4. 补充业务分组标签
	if rule.AssetGroupID > 0 {
		var groupName string
		e.db.Table("asset_group").Select("name").Where("id = ?", rule.AssetGroupID).Scan(&groupName)
		if groupName != "" {
			enrichedLabels["asset_group"] = groupName
		}
	}

	labelsJSON, _ := json.Marshal(enrichedLabels)

	event := &biz.AlertEvent{
		// ... 其他字段
		Labels: string(labelsJSON),
	}
	// ...
}
```

**修复效果**：
- ✅ 所有告警事件自动携带 `severity` 和 `ruleName` 标签
- ✅ 自动补充 `datasource`、`datasource_type` 标签
- ✅ 自动补充 `asset_group` 标签
- ✅ 支持基于标签的告警治理（去重、分组、抑制）
- ✅ 支持基于标签的维度统计和筛选

**验证方法**：
```bash
# 1. 触发告警
# 2. 查询告警事件
SELECT id, rule_name, severity, labels FROM alert_events ORDER BY id DESC LIMIT 1;

# 3. 检查 labels 字段是否包含：
# {"severity":"critical","ruleName":"CPU使用率过高","datasource":"Prometheus-prod","datasource_type":"prometheus","asset_group":"生产环境"}
```

---

### 问题2：治理规则保存后重新编辑时开关状态不正确

**问题描述**：
在告警订阅中配置治理规则，选择启用后保存，重新编辑进入时发现开关还是关闭状态。

**根本原因**：
在 `GovernanceConfig.vue` 组件中，使用了 `props.dedupData?.enabled || false` 这样的逻辑运算符来初始化状态。当数据库中 `enabled` 字段为 `false` 时，`||` 运算符会短路为默认值 `false`，导致无法区分"未配置"和"已配置但禁用"两种状态。

**修复方案**：
修改 `web/src/views/alert/components/GovernanceConfig.vue`，使用 `watchEffect` 监听 props 变化，显式更新本地状态，避免使用 `||` 运算符。

**修复代码**：
```typescript
// 修复前（错误）
const dedupRule = ref<Partial<DedupRule>>({
  enabled: props.dedupData?.enabled || false,  // ❌ 当 enabled=false 时会被短路
  dedupWindow: props.dedupData?.dedupWindow || 600,
  // ...
})

// 修复后（正确）
const dedupRule = ref<Partial<DedupRule>>({
  enabled: false,
  dedupWindow: 600,
  // ...
})

// 使用 watchEffect 监听 props 变化
watchEffect(() => {
  if (props.dedupData) {
    dedupRule.value.enabled = props.dedupData.enabled  // ✅ 显式赋值
    dedupRule.value.dedupWindow = props.dedupData.dedupWindow
    if (props.dedupData.fingerprintKeys) {
      fingerprintKeys.value = JSON.parse(props.dedupData.fingerprintKeys)
    }
  }
  // ... 其他规则同理
})
```

**修复效果**：
- ✅ 保存治理规则后，重新编辑时开关状态正确显示
- ✅ 可以正确区分"未配置"和"已配置但禁用"两种状态
- ✅ 所有配置项（去重窗口、分组字段等）都能正确回显

**验证方法**：
```bash
# 1. 创建订阅并配置治理规则
# 2. 启用去重规则，保存
# 3. 重新编辑该订阅
# 4. 切换到"治理规则"Tab
# 5. 验证去重规则的开关是否为"开启"状态
# 6. 关闭去重规则，保存
# 7. 重新编辑该订阅
# 8. 验证去重规则的开关是否为"关闭"状态
```

---

## 修复文件清单

### 后端
- `internal/service/alert/eval_service.go` - 修改 `createFiringEvent` 方法

### 前端
- `web/src/views/alert/components/GovernanceConfig.vue` - 修改状态初始化逻辑

---

## 验证结果

### 编译验证
```bash
# 后端编译
go build -o /dev/null ./cmd/... ./internal/...
✅ 通过

# 前端类型检查
npx vue-tsc --noEmit
✅ 通过
```

### 功能验证
- ✅ 告警事件自动携带完整标签
- ✅ 治理规则开关状态正确回显
- ✅ 不影响现有功能

---

## 影响范围

### 问题1影响
- **影响模块**：告警评估引擎、告警事件创建
- **影响功能**：所有新创建的告警事件
- **向后兼容**：是（不影响已有告警事件）

### 问题2影响
- **影响模块**：治理规则配置组件
- **影响功能**：治理规则的编辑和回显
- **向后兼容**：是（不影响已保存的规则）

---

## 后续建议

1. **标签标准化**：建议制定标签命名规范，统一标签字段名称
2. **标签文档**：补充标签字段说明文档，方便用户使用
3. **标签验证**：在前端配置治理规则时，提供标签字段的自动补全和验证
4. **标签统计**：在告警统计面板中增加基于标签的维度分析

---

## 修复完成时间
2026-04-27

## 修复人员
Claude (Anthropic)
