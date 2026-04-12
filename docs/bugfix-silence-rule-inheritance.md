# 告警屏蔽规则继承问题修复说明

## 问题描述

### 用户反馈的问题

当用户对某条告警进行屏蔽后：
1. ✅ 该告警会进入"已屏蔽告警"列表
2. ✅ 告警恢复时，已屏蔽列表中显示"已恢复"
3. ❌ **告警再次触发时，屏蔽规则失效**，新告警显示为"告警中"并推送通知

### 期望行为

屏蔽某条告警后，只要用户没有从"已屏蔽告警"列表中取消屏蔽，那么今后该告警再次产生时：
- 应该自动保持"屏蔽中"状态
- 不应该推送告警通知
- 应该出现在"已屏蔽告警"列表中

---

## 问题根源分析

### 核心问题

系统存在**两套屏蔽机制**，但它们没有协同工作：

#### 1. 告警事件级别的屏蔽（`alert_events` 表）
- 字段：`silenced`, `silence_until`, `silence_reason` 等
- 用途：前端显示"屏蔽中"标签
- 问题：告警恢复后，新产生的告警事件是全新记录，不会继承屏蔽状态

#### 2. 屏蔽规则级别的屏蔽（`alert_silence_rules` 表）
- 字段：`severity`, `rule_name`, `labels`, `type`, `duration` 等
- 用途：按"告警等级 + 规则名称 + 标签"三元组匹配告警
- 问题：只在通知阶段检查，不会自动更新告警事件的屏蔽状态

### 代码层面的问题

**位置**：`internal/service/alert/eval_service.go:366-384`

```go
func (e *EvalEngine) createFiringEvent(...) {
    event := &biz.AlertEvent{
        // ...
        Status: "firing",  // ❌ 硬编码为 firing
        // ...
    }
    if err := e.eventRepo.Create(ctx, event); err != nil {
        return
    }
    e.sendNotifications(ctx, rule, event, false)  // ❌ 直接发送通知
}
```

**问题**：
1. 创建新告警事件时，没有检查是否存在匹配的屏蔽规则
2. 直接将状态设置为 `firing`，`silenced` 字段默认为 `false`
3. 直接调用 `sendNotifications()`，虽然通知会被拦截，但事件状态已经错误

### 业务流程对比

#### 问题流程
```
告警触发 → 创建事件(firing, silenced=false) → 用户屏蔽 → 创建屏蔽规则
    ↓
告警恢复 → 旧事件(resolved)
    ↓
告警再次触发 → 创建新事件(firing, silenced=false) ❌ 
    ↓
前端显示"告警中" ❌ + 通知被拦截 ✅
```

#### 修复后流程
```
告警触发 → 创建事件(firing, silenced=false) → 用户屏蔽 → 创建屏蔽规则
    ↓
告警恢复 → 旧事件(resolved)
    ↓
告警再次触发 → 检查屏蔽规则 → 创建新事件(firing, silenced=true) ✅
    ↓
前端显示"屏蔽中" ✅ + 不发送通知 ✅
```

---

## 修复方案

### 修改文件

`internal/service/alert/eval_service.go`

### 修改内容

#### 1. 修改 `createFiringEvent()` 函数

**修改前**：
```go
func (e *EvalEngine) createFiringEvent(ctx context.Context, rule *biz.AlertRule, fingerprint string, value float64, firedAt time.Time, metricLabels map[string]string) {
    event := &biz.AlertEvent{
        AlertRuleID:  rule.ID,
        RuleName:     rule.Name,
        AssetGroupID: rule.AssetGroupID,
        Fingerprint:  fingerprint,
        Severity:     rule.Severity,
        Status:       "firing",
        Labels:       mergeLabels(rule.Labels, metricLabels),
        Annotations:  rule.Annotations,
        Value:        value,
        FiredAt:      firedAt,
    }
    if err := e.eventRepo.Create(ctx, event); err != nil {
        appLogger.Error("创建告警事件失败", zap.Error(err))
        return
    }
    e.sendNotifications(ctx, rule, event, false)
}
```

**修改后**：
```go
func (e *EvalEngine) createFiringEvent(ctx context.Context, rule *biz.AlertRule, fingerprint string, value float64, firedAt time.Time, metricLabels map[string]string) {
    event := &biz.AlertEvent{
        AlertRuleID:  rule.ID,
        RuleName:     rule.Name,
        AssetGroupID: rule.AssetGroupID,
        Fingerprint:  fingerprint,
        Severity:     rule.Severity,
        Status:       "firing",
        Labels:       mergeLabels(rule.Labels, metricLabels),
        Annotations:  rule.Annotations,
        Value:        value,
        FiredAt:      firedAt,
    }

    // ✅ 新增：检查是否匹配屏蔽规则
    if matchedRule := e.findMatchingSilenceRule(ctx, event); matchedRule != nil {
        event.Silenced = true
        event.SilenceType = matchedRule.Type
        event.SilenceReason = matchedRule.Reason
        now := time.Now()
        event.SilencedAt = &now

        if matchedRule.Type == "fixed" {
            event.SilenceUntil = matchedRule.SilenceUntil
        } else if matchedRule.Type == "periodic" {
            event.SilenceTimeRanges = matchedRule.TimeRanges
        }

        appLogger.Info("新告警匹配到屏蔽规则，自动设置为屏蔽状态",
            zap.Uint("ruleID", rule.ID),
            zap.String("fingerprint", fingerprint),
            zap.Uint("silenceRuleID", matchedRule.ID),
            zap.String("silenceType", matchedRule.Type))
    }

    if err := e.eventRepo.Create(ctx, event); err != nil {
        appLogger.Error("创建告警事件失败", zap.Error(err))
        return
    }

    // ✅ 修改：只有未屏蔽的告警才发送通知
    if !event.Silenced {
        e.sendNotifications(ctx, rule, event, false)
    }
}
```

#### 2. 新增 `findMatchingSilenceRule()` 辅助函数

```go
// findMatchingSilenceRule 查找匹配的屏蔽规则（返回第一个匹配的规则）
func (e *EvalEngine) findMatchingSilenceRule(ctx context.Context, event *biz.AlertEvent) *biz.AlertSilenceRule {
    rules, err := e.silenceCache.GetActiveRules(ctx)
    if err != nil || len(rules) == 0 {
        return nil
    }

    now := time.Now()
    for _, rule := range rules {
        if e.matchSilenceRule(rule, event, now) {
            return rule
        }
    }
    return nil
}
```

---

## 修复效果

### 修复前

| 场景 | 告警状态 | 前端显示 | 是否推送 | 是否符合预期 |
|------|---------|---------|---------|-------------|
| 首次触发 | firing, silenced=false | 告警中 | ✅ 推送 | ✅ |
| 用户屏蔽 | firing, silenced=true | 屏蔽中 | ❌ 不推送 | ✅ |
| 告警恢复 | resolved, silenced=true | 已恢复 | - | ✅ |
| 再次触发 | firing, silenced=false | **告警中** ❌ | ❌ 不推送 | ❌ |

### 修复后

| 场景 | 告警状态 | 前端显示 | 是否推送 | 是否符合预期 |
|------|---------|---------|---------|-------------|
| 首次触发 | firing, silenced=false | 告警中 | ✅ 推送 | ✅ |
| 用户屏蔽 | firing, silenced=true | 屏蔽中 | ❌ 不推送 | ✅ |
| 告警恢复 | resolved, silenced=true | 已恢复 | - | ✅ |
| 再次触发 | firing, silenced=true | **屏蔽中** ✅ | ❌ 不推送 | ✅ |

---

## 测试验证

### 测试步骤

1. **创建测试告警规则**
   - 规则名称：`test_cpu_high`
   - 告警等级：`critical`
   - 表达式：`node_cpu_usage > 80`

2. **触发告警**
   - 等待告警触发
   - 确认前端显示"告警中"
   - 确认收到告警通知

3. **屏蔽告警**
   - 点击"屏蔽"按钮
   - 选择"固定时长" → "1小时"
   - 输入屏蔽原因："测试屏蔽功能"
   - 确认屏蔽成功

4. **验证屏蔽状态**
   - 前端显示"屏蔽中"标签
   - "已屏蔽告警"列表中可以看到该告警
   - 不再收到新的告警通知

5. **模拟告警恢复**
   - 降低 CPU 使用率至 80% 以下
   - 等待告警自动恢复
   - 确认"已屏蔽告警"列表中显示"已恢复"

6. **再次触发告警（关键测试）**
   - 提高 CPU 使用率至 80% 以上
   - 等待告警再次触发
   - **验证点**：
     - ✅ 前端应显示"屏蔽中"（而不是"告警中"）
     - ✅ 不应收到告警通知
     - ✅ "已屏蔽告警"列表中应出现新的告警记录

7. **取消屏蔽**
   - 在"已屏蔽告警"列表中点击"取消屏蔽"
   - 确认屏蔽规则被删除
   - 确认告警状态变为"告警中"
   - 确认开始收到告警通知

### 预期结果

- ✅ 屏蔽规则在告警恢复后仍然有效
- ✅ 新产生的告警自动继承屏蔽状态
- ✅ 前端显示正确的"屏蔽中"标签
- ✅ 不会推送被屏蔽的告警通知
- ✅ 取消屏蔽后，告警恢复正常推送

---

## 注意事项

### 1. 屏蔽规则匹配逻辑

屏蔽规则按以下三元组匹配：
- **告警等级**（`severity`）：必须完全匹配
- **规则名称**（`rule_name`）：必须完全匹配
- **标签**（`labels`）：子集匹配（告警的标签必须包含屏蔽规则的所有标签）

### 2. 屏蔽类型

#### 固定时长屏蔽（`type=fixed`）
- 从屏蔽时刻开始计时
- 到达 `silence_until` 时间后自动失效
- 适用场景：临时维护、短期问题

#### 周期性屏蔽（`type=periodic`）
- 按星期几 + 时间段生效
- 例如：周一至周五 09:00-18:00
- 适用场景：工作时间屏蔽、定期维护窗口

### 3. 性能优化

- 屏蔽规则从 Redis 缓存读取，避免频繁查询数据库
- 使用 `silenceCache.GetActiveRules()` 获取已启用的规则
- 缓存自动订阅 Pub/Sub 事件，规则变更时实时更新

### 4. 日志记录

修复后会在日志中记录屏蔽规则匹配信息：
```
新告警匹配到屏蔽规则，自动设置为屏蔽状态
  ruleID: 123
  fingerprint: abc123def456
  silenceRuleID: 456
  silenceType: fixed
```

---

## 相关代码位置

- **告警评估引擎**：`internal/service/alert/eval_service.go`
- **屏蔽规则缓存**：`internal/service/alert/silence_cache.go`
- **告警事件仓储**：`internal/data/alert/event_repo.go`
- **屏蔽规则仓储**：`internal/data/alert/silence_repo.go`
- **前端实时告警页面**：`web/src/views/alert/ActiveEvents.vue`
- **前端已屏蔽告警弹窗**：`web/src/views/alert/SilencedAlertsModal.vue`

---

## 版本信息

- **修复日期**：2026-04-08
- **修复版本**：v1.x.x
- **影响范围**：告警管理模块
- **向后兼容**：是（不影响现有功能）

---

## 总结

本次修复解决了告警屏蔽规则在告警恢复后失效的严重问题。通过在创建新告警事件时检查屏蔽规则，确保了屏蔽状态的正确继承，实现了用户期望的"一次屏蔽，持续生效"的行为。

修复后，用户体验得到显著提升：
- 不再需要重复屏蔽相同的告警
- 屏蔽规则真正起到了"规则"的作用
- 前端显示状态与实际屏蔽状态保持一致
