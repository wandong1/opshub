# 告警规则删除后残留问题修复报告

## 问题描述

用户删除了 CPU 利用率告警规则，但在"实时告警"页面仍然看到两条该规则的告警事件。

## 问题分析

### 数据库查询结果

```sql
SELECT id, alert_rule_id, rule_name, status, fired_at, resolved_at 
FROM alert_events 
WHERE rule_name LIKE '%cpu%';
```

| ID | Rule ID | Status | Fired At | Resolved At |
|----|---------|--------|----------|-------------|
| 9008 | 16 | firing | 2026-05-02 23:46:11 | NULL |
| 9007 | 16 | firing | 2026-05-02 23:46:11 | NULL |

规则 ID 16 的删除时间：`2026-05-02 23:45:59`

### 时序分析

```
23:45:59 - 用户删除规则 ID 16
           ├─ 步骤1: 标记现有告警为 resolved
           ├─ 步骤2: 清理 Redis 状态
           ├─ 步骤3: 删除规则（软删除）
           └─ 步骤4: 发布规则重载事件

23:46:11 - 评估引擎触发新告警（使用旧缓存）
           ├─ 告警 ID 9007 产生
           └─ 告警 ID 9008 产生
```

### 根本原因

**时序竞态问题**：

1. 删除规则时，先标记现有告警为 `resolved`
2. 然后删除规则
3. 最后发布规则重载事件
4. **问题**：在步骤 2-3 之间，评估引擎可能还在使用旧缓存
5. 导致在规则删除后又产生了新的告警（ID 9007, 9008）

## 修复方案

### 1. 调整删除顺序

**修复前**：
```go
func (s *HTTPServer) deleteRule(c *gin.Context) {
    // 1. 标记现有告警为 resolved
    s.eventRepo.ResolveActiveByRuleID(ctx, ruleID)
    
    // 2. 清理 Redis 状态
    s.evalEngine.ClearRuleStates(ctx, ruleID)
    
    // 3. 删除规则
    s.ruleRepo.Delete(ctx, ruleID)
    
    // 4. 发布规则重载事件 ❌ 太晚了
    s.evalEngine.GetRuleCache().PublishReloadEvent(ctx)
}
```

**修复后**：
```go
func (s *HTTPServer) deleteRule(c *gin.Context) {
    // 1. 标记现有告警为 resolved
    s.eventRepo.ResolveActiveByRuleID(ctx, ruleID)
    
    // 2. 清理 Redis 状态
    s.evalEngine.ClearRuleStates(ctx, ruleID)
    
    // 3. 发布规则重载事件 ✅ 提前发布，让评估引擎停止评估
    s.evalEngine.GetRuleCache().PublishReloadEvent(ctx)
    
    // 4. 删除规则
    s.ruleRepo.Delete(ctx, ruleID)
    
    // 5. 再次清理可能新产生的告警 ✅ 防止竞态
    s.eventRepo.ResolveActiveByRuleID(ctx, ruleID)
}
```

### 2. 前端显示修复

**问题**：前端表格的 PromQL 列显示为空

**原因**：表格显示的是 `record.expr` 字段，但新格式使用的是 `record.queryExpr`

**修复**：
```vue
<!-- 修复前 -->
<code class="expr-code">{{ record.expr }}</code>

<!-- 修复后 -->
<code class="expr-code">{{ record.queryExpr || record.expr }}</code>
```

## 修复步骤

### 1. 手动清理遗留告警

```sql
UPDATE alert_events 
SET status = 'resolved', 
    resolve_type = 'manual', 
    resolved_at = NOW() 
WHERE id IN (9007, 9008);
```

### 2. 修改代码逻辑

- ✅ 调整删除规则的执行顺序
- ✅ 提前发布规则重载事件
- ✅ 删除后再次清理告警

### 3. 修复前端显示

- ✅ 优先显示 `queryExpr` 字段
- ✅ 向后兼容 `expr` 字段

## 为什么会出现竞态

### 评估引擎的工作机制

```
评估引擎（每 15 秒一次）
  ├─ 从缓存读取规则列表
  ├─ 对每个规则执行 PromQL 查询
  ├─ 判断是否触发告警
  └─ 创建告警事件

规则缓存刷新机制
  ├─ 本地缓存（内存）
  ├─ Redis 缓存（共享）
  └─ Pub/Sub 通知（异步）
```

### 竞态窗口

```
T0: 用户点击删除规则
T1: 标记现有告警为 resolved
T2: 删除规则
T3: 发布 Pub/Sub 通知
T4: 评估引擎收到通知，刷新缓存

问题：在 T2-T4 之间，评估引擎可能还在使用旧缓存
```

## 测试验证

### 1. 验证遗留告警已清理

```bash
# 刷新前端页面，确认实时告警列表为空
```

### 2. 测试删除规则

1. 创建一个测试规则并启用
2. 等待触发告警
3. 删除规则
4. 检查实时告警列表，确认告警已自动恢复

### 3. 验证前端显示

1. 导入包含 `queryExpr` 的规则
2. 检查规则列表的 PromQL 列是否正确显示

## 预防措施

### 1. 代码层面

- ✅ 提前发布规则重载事件
- ✅ 删除后再次清理告警
- ✅ 使用事务确保原子性（未来优化）

### 2. 监控层面

- 建议添加监控：检测 `deleted_at IS NOT NULL` 的规则是否还有 `firing` 状态的告警
- 定期清理孤儿告警（规则已删除但告警仍为 firing）

### 3. 用户提示

- 删除规则时提示用户：相关告警将自动恢复
- 如果删除失败，提示用户重试

## 相关文件

- ✅ `internal/server/alert/rule_handler.go` - 删除规则逻辑（已修复）
- ✅ `internal/data/alert/event_repo.go` - 告警事件仓储（无需修改）
- ✅ `web/src/views/alert/RuleManagement.vue` - 前端规则列表（已修复）
- ✅ `web/src/views/alert/ActiveEvents.vue` - 前端实时告警（无需修改）

## 后续优化建议

### 短期（1 周内）

1. 添加定时任务，清理孤儿告警
   ```go
   // 每小时执行一次
   func cleanOrphanEvents() {
       // 查找规则已删除但告警仍为 firing 的记录
       // 标记为 resolved
   }
   ```

2. 添加监控指标
   ```go
   // 统计孤儿告警数量
   orphanEventsCount := countEventsWithDeletedRules()
   ```

### 中期（1 个月内）

1. 使用数据库事务确保原子性
   ```go
   tx.Begin()
   defer tx.Rollback()
   
   // 1. 标记告警
   // 2. 删除规则
   // 3. 再次标记告警
   
   tx.Commit()
   ```

2. 优化规则缓存刷新机制
   - 减少 Pub/Sub 延迟
   - 使用同步刷新而非异步

### 长期（3 个月内）

1. 重构评估引擎
   - 使用规则版本号
   - 评估前检查规则是否已删除

2. 添加告警生命周期管理
   - 告警创建时记录规则版本
   - 规则删除时自动清理所有版本的告警

## 总结

本次修复解决了删除规则后告警残留的问题，主要通过：

1. ✅ 调整删除顺序，提前发布规则重载事件
2. ✅ 删除后再次清理告警，防止竞态
3. ✅ 手动清理了遗留的告警
4. ✅ 修复了前端 PromQL 列显示问题

**建议尽快部署到生产环境，避免用户困惑。**
