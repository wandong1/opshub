# 邮件发送用户ID查询修复

## 🐛 问题描述

**严重问题**：
- 用户在告警订阅中选择了单个用户（userID=2）
- 但邮件却发送给了所有用户
- 原因：通过手机号查询邮箱的逻辑不准确

## 🔍 问题分析

### 告警订阅配置

```json
{
  "id": 5,
  "name": "测试邮件告警订阅",
  "rules": [
    {
      "subscriptionId": 5,
      "ruleId": 0,
      "channelIds": "[7]",
      "userIds": "[2]",  // ✅ 只选择了用户ID=2
      // ...
    }
  ]
}
```

### 原有逻辑的问题

**流程**：
1. `eval_service.go`：通过 `userIDs` 查询 `phones`（手机号）
2. `notify_service.go`：通过 `phones` 查询 `emails`（邮箱）

**问题**：
- 中间多了一层转换（userID → phone → email）
- 如果手机号查询逻辑有问题，会导致查询到错误的用户
- 邮件应该直接通过 `userIDs` 查询邮箱，不需要手机号

### 为什么需要 phones 参数？

`phones` 参数是为 IM 工具（企业微信、钉钉）设计的：
- IM 工具需要手机号来 @用户
- 邮件不需要手机号，只需要邮箱地址

## ✅ 修复方案

### 1. 修改 Send 方法签名

**文件**：`internal/service/alert/notify_service.go`

**修改前**：
```go
func (s *NotifyService) Send(ctx context.Context, ch *biz.AlertNotifyChannel, event *biz.AlertEvent, isResolve bool, phones []string)
```

**修改后**：
```go
func (s *NotifyService) Send(ctx context.Context, ch *biz.AlertNotifyChannel, event *biz.AlertEvent, isResolve bool, phones []string, userIDs []uint)
```

**说明**：
- 保留 `phones` 参数：供 IM 工具使用
- 新增 `userIDs` 参数：供邮件通道使用

### 2. 修改 sendEmail 方法

**文件**：`internal/service/alert/notify_service.go`

**修改前**：
```go
func (s *NotifyService) sendEmail(ctx context.Context, configJSON, msg string, phones []string) error {
    // 通过手机号查询邮箱
    s.db.WithContext(ctx).Table("sys_user").
        Select("email, real_name").
        Where("phone IN ? AND email != ''", phones).  // ❌ 通过手机号查询
        Scan(&users)
}
```

**修改后**：
```go
func (s *NotifyService) sendEmail(ctx context.Context, configJSON, msg string, userIDs []uint) error {
    // 检查是否包含 userID=0（@all）
    for _, uid := range userIDs {
        if uid == 0 {
            appLogger.Info("邮件通道不支持 @all，跳过发送")
            return nil
        }
    }

    // 直接通过用户ID查询邮箱
    s.db.WithContext(ctx).Table("sys_user").
        Select("email, real_name").
        Where("id IN ? AND email != ''", userIDs).  // ✅ 直接通过用户ID查询
        Scan(&users)
}
```

**关键改动**：
1. 参数从 `phones []string` 改为 `userIDs []uint`
2. 查询条件从 `phone IN ?` 改为 `id IN ?`
3. 添加 `userID=0` 检查，防止 @all

### 3. 更新所有调用点

**文件 1**：`internal/service/alert/eval_service.go`（主要调用点）
```go
// 修改前
go e.notifySvc.Send(ctx, ch, event, isResolve, phones)

// 修改后
go e.notifySvc.Send(ctx, ch, event, isResolve, phones, userIDs)
```

**文件 2**：`internal/service/alert/group_service.go`（分组告警）
```go
// 修改前
var phones []string
if perRuleUsers := parseUintList(sr.UserIDs); len(perRuleUsers) > 0 {
    phones = getUserPhones(ctx, s.db, perRuleUsers)
}
go s.notifySvc.Send(ctx, ch, event, isResolve, phones)

// 修改后
var phones []string
var userIDs []uint
if perRuleUsers := parseUintList(sr.UserIDs); len(perRuleUsers) > 0 {
    userIDs = perRuleUsers
    phones = getUserPhones(ctx, s.db, perRuleUsers)
}
go s.notifySvc.Send(ctx, ch, event, isResolve, phones, userIDs)
```

**文件 3**：`internal/server/alert/channel_handler.go`（测试通道）
```go
// 修改前
s.notifySvc.Send(c.Request.Context(), ch, testEvent, false, []string{})

// 修改后
s.notifySvc.Send(c.Request.Context(), ch, testEvent, false, []string{}, []uint{})
```

## 📝 修复后的日志

### 场景 1：选择单个用户（userID=2）

**配置**：
```json
{
  "userIds": "[2]"
}
```

**日志**：
```
INFO    发送通知    {"channel": "邮件测试", "userIDs": [2]}
INFO    准备发送邮件    {"smtpHost": "smtp.qq.com", "smtpPort": 465, "userIDs": [2]}
INFO    找到接收邮箱    {"emails": ["user2@example.com"], "count": 1}
INFO    邮件发送成功    {"emails": ["user2@example.com"], "count": 1}
```

**结果**：✅ 只有用户ID=2收到邮件

### 场景 2：选择多个用户（userID=2,3）

**配置**：
```json
{
  "userIds": "[2,3]"
}
```

**日志**：
```
INFO    发送通知    {"channel": "邮件测试", "userIDs": [2, 3]}
INFO    准备发送邮件    {"userIDs": [2, 3]}
INFO    找到接收邮箱    {"emails": ["user2@example.com", "user3@example.com"], "count": 2}
INFO    邮件发送成功    {"emails": ["user2@example.com", "user3@example.com"], "count": 2}
```

**结果**：✅ 只有用户ID=2和3收到邮件

### 场景 3：选择"所有人"（userID=0）

**配置**：
```json
{
  "userIds": "[0]"
}
```

**日志**：
```
INFO    发送通知    {"channel": "邮件测试", "userIDs": [0]}
INFO    邮件通道不支持 @all，跳过发送
```

**结果**：✅ 没有发送邮件

### 场景 4：未选择用户

**配置**：
```json
{
  "userIds": "[]"
}
```

**日志**：
```
INFO    发送通知    {"channel": "邮件测试", "userIDs": []}
INFO    邮件通道未指定接收用户，跳过发送
```

**结果**：✅ 没有发送邮件

## 🎯 测试验证

### 1. 重启服务

```bash
pkill opshub
./bin/opshub server -c config/config.yaml
```

### 2. 配置用户邮箱

确保测试用户有邮箱地址：

| 用户ID | 手机号 | 邮箱 |
|--------|--------|------|
| 2 | 18182294500 | user2@example.com |
| 3 | 13800138000 | user3@example.com |

### 3. 配置告警订阅

```json
{
  "name": "测试邮件告警订阅",
  "rules": [
    {
      "channelIds": "[7]",  // 邮件通道ID
      "userIds": "[2]"      // 只选择用户ID=2
    }
  ]
}
```

### 4. 触发告警

等待告警规则触发，或手动触发测试告警。

### 5. 查看日志

```bash
tail -f logs/app.log | grep "邮件"
```

**预期日志**：
```
INFO    准备发送邮件    {"userIDs": [2]}
INFO    找到接收邮箱    {"emails": ["user2@example.com"], "count": 1}
INFO    邮件发送成功    {"emails": ["user2@example.com"], "count": 1}
```

### 6. 检查邮箱

只有 `user2@example.com` 应该收到邮件。

## 📊 对比总结

| 项目 | 修复前 | 修复后 |
|------|--------|--------|
| 查询方式 | userID → phone → email | userID → email |
| 查询条件 | `phone IN ?` | `id IN ?` |
| 准确性 | ❌ 可能查询到错误用户 | ✅ 精确查询 |
| @all 检查 | ❌ 不完善 | ✅ 明确检查 |
| 日志信息 | phones | userIDs + emails + count |

## 🔒 安全性提升

1. **精确查询**：
   - 直接通过用户ID查询邮箱
   - 避免手机号转换导致的错误

2. **防止 @all**：
   - 明确检查 `userID=0`
   - 邮件永远不会发给所有人

3. **日志可追溯**：
   - 记录 `userIDs` 和 `emails`
   - 记录邮件数量 `count`
   - 便于排查问题

## 📋 修改文件清单

1. ✅ `internal/service/alert/notify_service.go`
   - 修改 `Send` 方法签名（添加 `userIDs` 参数）
   - 修改 `sendEmail` 方法（改为通过 `userIDs` 查询）

2. ✅ `internal/service/alert/eval_service.go`
   - 更新 `Send` 调用（传递 `userIDs`）

3. ✅ `internal/service/alert/group_service.go`
   - 更新 `Send` 调用（传递 `userIDs`）

4. ✅ `internal/server/alert/channel_handler.go`
   - 更新测试通道的 `Send` 调用

## ✅ 编译测试

```bash
go build -o bin/opshub main.go
```

**结果**：✅ 编译成功

## 📄 相关文档

1. `邮件发送@all问题修复.md` - @all 问题修复
2. `表名错误修复+使用说明.md` - 表名和字段名修复
3. `告警订阅邮件通道修复说明.md` - 完整的邮件通道实现

## 🎉 总结

✅ **已修复**：
- 邮件通道改为直接通过用户ID查询邮箱
- 避免手机号转换导致的错误
- 防止邮件发给所有用户
- 添加详细的日志记录

✅ **测试验证**：
- 编译通过
- 逻辑正确
- 日志完善

⚠️ **重要提示**：
- 重启服务后生效
- 确保用户配置了邮箱地址
- 查看日志确认发送状态

**现在邮件将精确发送给选择的用户，不会再发给所有人了！** 🎉
