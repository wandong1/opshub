# 告警管理 - 实时告警屏蔽逻辑优化 - 实施完成报告

## 📋 项目概述

本次优化实现了告警管理系统的多维度屏蔽功能，支持批量操作、灵活时效配置和标签搜索。

---

## ✅ 已完成功能

### 1. 后端实现（100%）

#### 1.1 数据模型扩展
- ✅ 扩展 `AlertEvent` 模型，添加 `SilenceDimension` 和 `SilenceType` 字段
- ✅ 创建 `AlertSilenceRule` 屏蔽规则表，支持固定时长和周期性屏蔽
- ✅ 数据库迁移已配置（`cmd/server/server.go`）

#### 1.2 核心功能模块
- ✅ **标签匹配引擎** (`internal/service/alert/label_matcher.go`)
  - 子集匹配：规则标签必须是告警标签的子集
  - 模糊搜索：支持通配符 `*`（如 `job=prome*`）
  
- ✅ **屏蔽规则仓储** (`internal/data/alert/silence_repo.go`)
  - CRUD 操作
  - 规则匹配查询
  - 活跃规则列表

- ✅ **评估引擎扩展** (`internal/service/alert/eval_service.go`)
  - `shouldSilence()` - 检查告警是否应被屏蔽
  - `matchSilenceRule()` - 三元组匹配（等级 + 规则名 + 标签）
  - 支持固定时长和周期性屏蔽检查

#### 1.3 API 接口
**批量屏蔽 API：**
- `POST /api/v1/alert/events/batch-silence` - 批量屏蔽告警
- `POST /api/v1/alert/events/batch-unsilence` - 批量取消屏蔽
- `GET /api/v1/alert/events/silenced` - 查询已屏蔽告警列表

**屏蔽规则管理 API：**
- `GET /api/v1/alert/silence-rules` - 查询规则列表
- `POST /api/v1/alert/silence-rules` - 创建规则
- `GET /api/v1/alert/silence-rules/:id` - 获取规则详情
- `PUT /api/v1/alert/silence-rules/:id` - 更新规则
- `DELETE /api/v1/alert/silence-rules/:id` - 删除规则
- `PUT /api/v1/alert/silence-rules/:id/toggle` - 启用/禁用规则

#### 1.4 标签搜索功能
- ✅ `event_repo.go` 中实现 `applyLabelFilter()` 方法
- ✅ 使用 MySQL `JSON_EXTRACT` + `LIKE` 实现模糊匹配
- ✅ 支持前缀、后缀、包含匹配

#### 1.5 权限配置
- ✅ 菜单按钮权限已添加：
  - `alert:events:batch-silence` - 批量屏蔽
  - `alert:events:batch-unsilence` - 批量取消屏蔽

---

### 2. 前端实现（100%）

#### 2.1 ActiveEvents.vue 扩展
- ✅ **表格行选择**：支持 checkbox 多选
- ✅ **批量操作栏**：显示已选数量，提供批量屏蔽/处理按钮
- ✅ **标签搜索**：新增标签过滤输入框，支持模糊匹配
- ✅ **批量屏蔽弹窗**：
  - 屏蔽维度说明（三元组）
  - 标签编辑功能（可移除部分标签扩大屏蔽范围）
  - 屏蔽类型选择（固定时长 / 周期性）
  - 固定时长选项（1h, 2h, 6h, 12h, 24h, 168h）
  - 周期性配置（工作日选择 + 时间段）
  - 屏蔽原因输入
- ✅ **已屏蔽告警入口**：新增"已屏蔽告警"按钮

#### 2.2 SilencedAlertsModal.vue 组件
- ✅ **过滤功能**：
  - 告警等级过滤
  - 告警状态过滤（告警中/已恢复）
  - 规则名称搜索
  - 标签搜索
- ✅ **批量操作**：
  - 表格行选择
  - 批量取消屏蔽
- ✅ **单条操作**：取消屏蔽
- ✅ **UI 样式**：完全对齐 ActiveEvents.vue，保持视觉统一

#### 2.3 API 客户端扩展
- ✅ `web/src/api/alert.ts` 添加：
  - `batchSilenceEvents()` - 批量屏蔽
  - `batchUnsilenceEvents()` - 批量取消屏蔽
  - `getSilencedEvents()` - 查询已屏蔽告警
  - `AlertSilenceRule` 接口定义
  - 屏蔽规则管理接口

---

## 🎯 核心特性

### 1. 多维度屏蔽（三元组）
默认按 `(告警等级, 规则名称, 标签)` 三元组进行屏蔽：
- 相同三元组的所有告警（包括反复触发）都会被屏蔽
- 用户可编辑标签：移除部分标签后，屏蔽范围扩大
- 示例：移除 `pod` 标签后，所有相同 `job` 和 `instance` 的告警都被屏蔽

### 2. 灵活的屏蔽时效
**固定时长屏蔽：**
- 支持 1小时、2小时、6小时、12小时、1天、1周
- 到期自动失效

**周期性屏蔽：**
- 支持工作日/周末选择
- 支持时间段配置（如 09:00-18:00）
- 每天自动生效/失效

### 3. 批量操作
- 支持批量选择多条告警
- 一键批量屏蔽
- 一键批量取消屏蔽
- 后端自动按三元组分组，避免创建重复规则

### 4. 标签搜索
- 支持精确匹配：`job=prometheus`
- 支持前缀匹配：`job=prome*`
- 支持后缀匹配：`instance=*:9090`
- 支持包含匹配：`instance=*:90*`

### 5. 已屏蔽告警管理
- 独立弹窗查看所有已屏蔽告警
- 区分告警中/已恢复状态
- 支持批量取消屏蔽
- 实时同步到实时告警列表

---

## 🧪 测试指南

### 1. 后端测试

#### 1.1 启动服务
```bash
# 启动服务（会自动执行数据库迁移）
make run

# 或者
go run main.go server -c config/config.yaml
```

#### 1.2 验证数据库迁移
```bash
docker exec -it opshub-mysql mysql -uroot -p'OpsHub@2026' opshub

# 检查新表
SHOW TABLES LIKE 'alert_silence_rules';
DESC alert_silence_rules;

# 检查扩展字段
DESC alert_events;
SHOW COLUMNS FROM alert_events LIKE 'silence%';
```

#### 1.3 测试 API（需要先登录获取 token）
```bash
# 1. 登录获取 token
TOKEN=$(curl -X POST http://localhost:9876/api/v1/identity/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"123456"}' | jq -r '.data.token')

# 2. 测试批量屏蔽
curl -X POST http://localhost:9876/api/v1/alert/events/batch-silence \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "eventIds": [1, 2],
    "type": "fixed",
    "duration": "2h",
    "reason": "测试屏蔽"
  }'

# 3. 测试标签搜索
curl "http://localhost:9876/api/v1/alert/events/active?labelFilter=job=prome*" \
  -H "Authorization: Bearer $TOKEN"

# 4. 测试已屏蔽告警列表
curl "http://localhost:9876/api/v1/alert/events/silenced?page=1&pageSize=20" \
  -H "Authorization: Bearer $TOKEN"

# 5. 测试批量取消屏蔽
curl -X POST http://localhost:9876/api/v1/alert/events/batch-unsilence \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"eventIds": [1, 2]}'
```

### 2. 前端测试

#### 2.1 启动前端开发服务器
```bash
cd web
npm install  # 首次运行
npm run dev
```

#### 2.2 功能测试清单

**测试场景 1：批量屏蔽（固定时长）**
1. 访问 http://localhost:5173
2. 登录系统（admin / 123456）
3. 进入"告警管理" -> "实时告警"
4. 勾选 2-3 条告警
5. 点击"批量屏蔽"按钮
6. 选择"固定时长" -> "2小时"
7. 输入屏蔽原因："测试批量屏蔽"
8. 点击"确定"
9. ✅ 验证：告警列表中选中的告警应显示"屏蔽中"标签

**测试场景 2：批量屏蔽（周期性）**
1. 勾选告警
2. 点击"批量屏蔽"
3. 选择"周期性"
4. 勾选"周一"到"周五"
5. 设置时间段：09:00 - 18:00
6. 输入原因："工作时间屏蔽"
7. 点击"确定"
8. ✅ 验证：屏蔽规则已创建

**测试场景 3：标签编辑（扩大屏蔽范围）**
1. 勾选一条告警
2. 点击"批量屏蔽"
3. 勾选"自定义标签"
4. 移除 `pod` 标签（点击标签的 X）
5. 选择固定时长 1小时
6. 点击"确定"
7. ✅ 验证：所有相同 job 和 instance 的告警都被屏蔽

**测试场景 4：标签搜索**
1. 在标签搜索框输入：`job=prome*`
2. 点击"查询"
3. ✅ 验证：只显示 job 以 "prome" 开头的告警

**测试场景 5：已屏蔽告警管理**
1. 点击"已屏蔽告警"按钮
2. ✅ 验证：弹窗显示所有已屏蔽的告警
3. 勾选几条告警
4. 点击"批量取消屏蔽"
5. 确认操作
6. ✅ 验证：告警从已屏蔽列表中移除，实时告警列表更新

**测试场景 6：过滤功能**
1. 在已屏蔽告警弹窗中
2. 测试等级过滤（选择 P1-紧急）
3. 测试状态过滤（选择"告警中"）
4. 测试规则名称搜索
5. 测试标签搜索
6. ✅ 验证：过滤结果正确

---

## 📊 数据库表结构

### alert_silence_rules（屏蔽规则表）
```sql
CREATE TABLE `alert_silence_rules` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `severity` varchar(20) NOT NULL COMMENT '告警等级',
  `rule_name` varchar(200) NOT NULL COMMENT '规则名称',
  `labels` text COMMENT 'JSON标签',
  `type` varchar(20) NOT NULL COMMENT 'fixed/periodic',
  `duration` varchar(20) DEFAULT NULL COMMENT '屏蔽时长',
  `silence_until` datetime(3) DEFAULT NULL COMMENT '屏蔽截止时间',
  `time_ranges` text COMMENT '周期性时间段JSON',
  `reason` varchar(1000) DEFAULT NULL COMMENT '屏蔽原因',
  `created_by` bigint unsigned DEFAULT NULL COMMENT '创建人',
  `enabled` tinyint(1) DEFAULT '1' COMMENT '是否启用',
  PRIMARY KEY (`id`),
  KEY `idx_severity` (`severity`),
  KEY `idx_rule_name` (`rule_name`),
  KEY `idx_silence_until` (`silence_until`),
  KEY `idx_created_by` (`created_by`),
  KEY `idx_alert_silence_rules_deleted_at` (`deleted_at`)
);
```

### alert_events（扩展字段）
```sql
ALTER TABLE `alert_events` 
ADD COLUMN `silence_dimension` text COMMENT '屏蔽维度JSON',
ADD COLUMN `silence_type` varchar(20) COMMENT 'fixed/periodic';
```

---

## 🔧 技术实现细节

### 1. 屏蔽匹配算法
```go
func (e *EvalEngine) matchSilenceRule(rule *biz.AlertSilenceRule, event *biz.AlertEvent, now time.Time) bool {
    // 1. 检查告警等级
    if rule.Severity != event.Severity {
        return false
    }
    
    // 2. 检查规则名称
    if rule.RuleName != event.RuleName {
        return false
    }
    
    // 3. 检查标签（子集匹配）
    if !MatchLabels(event.Labels, rule.Labels) {
        return false
    }
    
    // 4. 检查时效
    if rule.Type == "fixed" {
        return rule.SilenceUntil != nil && now.Before(*rule.SilenceUntil)
    } else if rule.Type == "periodic" {
        return isInTimeRanges(rule.TimeRanges, now)
    }
    
    return false
}
```

### 2. 标签子集匹配
```go
func MatchLabels(eventLabelsJSON string, ruleLabelsJSON string) bool {
    eventMap := parseLabels(eventLabelsJSON)
    ruleMap := parseLabels(ruleLabelsJSON)
    
    // 规则标签必须是告警标签的子集
    for k, v := range ruleMap {
        if eventMap[k] != v {
            return false
        }
    }
    return true
}
```

### 3. 标签模糊搜索（MySQL）
```go
func (r *EventRepo) applyLabelFilter(q *gorm.DB, filter string) *gorm.DB {
    parts := strings.SplitN(filter, "=", 2)
    key := parts[0]
    pattern := parts[1]
    jsonPath := fmt.Sprintf("$.%s", key)
    
    if strings.HasSuffix(pattern, "*") {
        // 前缀匹配：job=prome*
        prefix := strings.TrimSuffix(pattern, "*")
        q = q.Where("JSON_EXTRACT(labels, ?) LIKE ?", jsonPath, prefix+"%")
    }
    // ... 其他匹配模式
    
    return q
}
```

---

## 🎨 UI 设计规范

### 批量操作栏样式
```css
.batch-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  background: #e8f3ff;
  border: 1px solid #bedaff;
  border-radius: 6px;
  padding: 8px 14px;
  margin-bottom: 12px;
}
```

### 告警等级颜色
- P1-紧急：`#f53f3f` (红色)
- P2-重要：`#d95c2c` (橙红)
- P3-次要：`#ff7d00` (橙色)
- P4-警告：`#165dff` (蓝色)

---

## 📝 注意事项

1. **向后兼容**：保留了现有的单条屏蔽 API 和字段，新功能作为扩展
2. **性能优化**：批量屏蔽时按三元组分组，避免创建大量重复规则
3. **权限控制**：所有新 API 都需要登录认证，管理员自动拥有所有权限
4. **数据一致性**：屏蔽规则和事件状态保持同步
5. **用户体验**：UI 完全对齐现有风格，保持视觉统一

---

## 🚀 部署建议

1. **数据库备份**：升级前备份 `alert_events` 表
2. **灰度发布**：建议先在测试环境验证
3. **监控告警**：关注屏蔽规则的创建和匹配性能
4. **用户培训**：提供操作手册，说明三元组屏蔽逻辑

---

## 📞 技术支持

如有问题，请查看：
- 实施计划：`/Users/Zhuanz/.claude/plans/enumerated-watching-hamming.md`
- 后端代码：`internal/service/alert/`, `internal/server/alert/`
- 前端代码：`web/src/views/alert/ActiveEvents.vue`, `web/src/views/alert/SilencedAlertsModal.vue`

---

**实施完成时间**：2026-04-03
**实施人员**：Claude (Opus 4.6)
**状态**：✅ 已完成，待测试验证
