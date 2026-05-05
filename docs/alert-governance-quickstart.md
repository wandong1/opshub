# 告警治理功能快速使用指南

## 快速开始

### 1. 启动服务
```bash
# 服务会自动执行数据库迁移
./bin/opshub server -c config/config.yaml
```

### 2. 配置去重规则（减少重复告警）

**场景**：CPU告警频繁重复触发

```bash
curl -X POST http://localhost:9876/api/v1/alert/dedup-rules \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subscriptionId": 1,
    "name": "CPU告警去重",
    "enabled": true,
    "fingerprintKeys": "[\"severity\", \"ruleName\", \"instance\"]",
    "dedupWindow": 600
  }'
```

**效果**：相同的CPU告警在10分钟内只发送一次

### 3. 配置分组规则（减少通知次数）

**场景**：多个告警同时触发，希望聚合后发送

```bash
curl -X POST http://localhost:9876/api/v1/alert/group-rules \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subscriptionId": 1,
    "name": "按级别分组",
    "enabled": true,
    "groupBy": "[\"severity\", \"ruleName\"]",
    "groupWait": 30,
    "groupInterval": 300,
    "maxGroupSize": 20
  }'
```

**效果**：相同级别和规则的告警等待30秒后聚合发送

### 4. 配置抑制规则（避免级联告警）

**场景**：节点宕机时，该节点上的服务告警应该被抑制

```bash
curl -X POST http://localhost:9876/api/v1/alert/inhibit-rules \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subscriptionId": 1,
    "name": "节点宕机抑制服务告警",
    "enabled": true,
    "sourceMatchers": "{\"severity\": \"critical\", \"ruleName\": \"节点宕机\"}",
    "targetMatchers": "{\"severity\": \"warning\", \"ruleName\": \"服务不可用\"}",
    "equalLabels": "[\"instance\"]"
  }'
```

**效果**：当节点宕机告警触发时，同一节点的服务不可用告警被抑制

## 常见场景配置

### 场景1：高频告警去重

**问题**：磁盘使用率告警每分钟触发一次

**解决方案**：
```json
{
  "subscriptionId": 1,
  "name": "磁盘告警去重",
  "enabled": true,
  "fingerprintKeys": "[\"severity\", \"ruleName\", \"instance\", \"mountpoint\"]",
  "dedupWindow": 1800
}
```

**效果**：相同磁盘的告警30分钟内只发送一次

### 场景2：批量告警聚合

**问题**：集群重启时产生大量告警

**解决方案**：
```json
{
  "subscriptionId": 1,
  "name": "集群告警分组",
  "enabled": true,
  "groupBy": "[\"severity\", \"cluster\"]",
  "groupWait": 60,
  "groupInterval": 600,
  "maxGroupSize": 50
}
```

**效果**：同一集群的告警等待1分钟后聚合发送，最多50条一组

### 场景3：上下游依赖抑制

**问题**：数据库宕机导致所有应用告警

**解决方案**：
```json
{
  "subscriptionId": 1,
  "name": "数据库宕机抑制应用告警",
  "enabled": true,
  "sourceMatchers": "{\"severity\": \"critical\", \"ruleName\": \"数据库不可用\"}",
  "targetMatchers": "{\"severity\": \"warning\", \"ruleName\": \"应用连接失败\"}",
  "equalLabels": "[\"database\"]"
}
```

**效果**：数据库宕机时，依赖该数据库的应用告警被抑制

## 查询和管理

### 查询规则列表
```bash
# 查询某个订阅的去重规则
curl http://localhost:9876/api/v1/alert/dedup-rules?subscriptionId=1 \
  -H "Authorization: Bearer YOUR_TOKEN"

# 查询所有分组规则
curl http://localhost:9876/api/v1/alert/group-rules \
  -H "Authorization: Bearer YOUR_TOKEN"

# 查询抑制规则
curl http://localhost:9876/api/v1/alert/inhibit-rules?subscriptionId=1 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 更新规则
```bash
# 更新去重窗口为20分钟
curl -X PUT http://localhost:9876/api/v1/alert/dedup-rules/1 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subscriptionId": 1,
    "name": "CPU告警去重",
    "enabled": true,
    "fingerprintKeys": "[\"severity\", \"ruleName\", \"instance\"]",
    "dedupWindow": 1200
  }'
```

### 删除规则
```bash
curl -X DELETE http://localhost:9876/api/v1/alert/dedup-rules/1 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## 验证效果

### 1. 查看日志
```bash
# 查看去重日志
tail -f logs/app.log | grep "告警已去重"

# 查看抑制日志
tail -f logs/app.log | grep "告警被抑制"

# 查看分组日志
tail -f logs/app.log | grep "告警已加入分组"
```

### 2. 查看指纹记录
```sql
-- 查看去重指纹记录
SELECT * FROM alert_fingerprints 
WHERE subscription_id = 1 
ORDER BY last_seen_at DESC 
LIMIT 10;

-- 查看分组缓存
SELECT * FROM alert_group_cache 
WHERE subscription_id = 1 AND sent = 0;
```

## 最佳实践

### 1. 去重规则
- **指纹字段**：至少包含 `severity` 和 `ruleName`
- **去重窗口**：根据告警频率调整，建议 5-30 分钟
- **适用场景**：高频重复告警、抖动告警

### 2. 分组规则
- **分组字段**：通常按 `severity` 或 `cluster` 分组
- **等待时间**：建议 30-60 秒，给告警收集时间
- **发送间隔**：建议 5-10 分钟，避免频繁通知
- **适用场景**：批量告警、集群故障

### 3. 抑制规则
- **源告警**：选择根因告警（如节点宕机、网络故障）
- **目标告警**：选择衍生告警（如服务不可用、连接失败）
- **相等标签**：确保是同一资源（如 `instance`、`cluster`）
- **适用场景**：级联故障、依赖关系明确的告警

## 注意事项

1. **规则优先级**：静默 > 去重 > 抑制 > 分组
2. **订阅隔离**：每个订阅的治理规则独立生效
3. **性能影响**：大量规则会增加处理时间，建议每个订阅不超过10条规则
4. **测试验证**：新规则上线前先在测试环境验证
5. **定期清理**：系统会自动清理30天前的指纹记录和7天前的分组缓存

## 故障排查

### 问题1：去重不生效
**检查项**：
- 规则是否启用（`enabled: true`）
- 指纹字段是否正确
- 去重窗口是否过小
- 查看日志确认是否命中规则

### 问题2：分组未聚合
**检查项**：
- 分组字段是否匹配
- 等待时间是否足够
- 是否达到最大分组数
- 查看 `alert_group_cache` 表

### 问题3：抑制未生效
**检查项**：
- 源告警是否存在且为 firing 状态
- 匹配条件是否正确（JSON格式）
- 相等标签是否匹配
- 查看日志确认匹配逻辑

## 获取帮助

- 查看完整文档：`docs/alert-governance-implementation.md`
- 查看设计方案：`.claude/plans/sequential-wiggling-platypus.md`
- 提交Issue：https://github.com/your-repo/opshub/issues
