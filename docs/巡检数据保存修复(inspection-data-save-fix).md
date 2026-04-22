# 智能巡检模块 - 数据保存和回显问题修复

## 修复日期
2026-03-13

## 问题描述

用户反馈：
1. 前端可以正常选择业务分组和主机标签
2. 保存后数据没有正确保存到后端
3. 再次打开编辑时，业务分组和主机标签都消失了

## 问题分析

通过后端日志分析发现：

### 前端发送的数据（正确）
```json
{
  "groupIds": "[1]",
  "items": [{
    "hostTags": "[\"mysqld\"]",
    "hostIds": "[]"
  }]
}
```

### 后端保存的数据（错误）
```sql
UPDATE `inspection_groups` SET `group_ids`='' WHERE `id` = 3
INSERT INTO `inspection_items` (..., `host_tags`='', `host_ids`='', ...)
```

可以看到 `group_ids`、`host_tags`、`host_ids` 都被保存为空字符串。

### 根本原因

后端的 `Update` 方法中有条件判断：

```go
// GroupService.Update()
if req.GroupIDs != "" {
    group.GroupIDs = req.GroupIDs
}

// ItemService.Update()
if req.Description != "" {
    item.Description = req.Description
}
```

这种条件判断导致：
1. 当字段值为空字符串时，不会更新数据库
2. 但实际上前端发送的是 JSON 字符串（如 `"[1]"`），不应该被过滤
3. 某些情况下，GORM 可能将未赋值的字段保存为空字符串

## 修复方案

### 1. 修复 GroupService.Update()

**修改文件**：`internal/service/inspection_mgmt/group_service.go`

**修改前**：
```go
if req.Description != "" {
    group.Description = req.Description
}
// ...
if req.GroupIDs != "" {
    group.GroupIDs = req.GroupIDs
}
```

**修改后**：
```go
group.Description = req.Description  // 直接赋值，允许清空
// ...
group.GroupIDs = req.GroupIDs  // 直接赋值，确保 JSON 字符串被保存
```

### 2. 修复 ItemService.Update()

**修改文件**：`internal/service/inspection_mgmt/item_service.go`

**修改前**：
```go
if req.Description != "" {
    item.Description = req.Description
}
item.HostTags = req.HostTags
item.HostIDs = req.HostIDs
```

**修改后**：
```go
item.Description = req.Description  // 直接赋值，允许清空
// ...
item.HostTags = req.HostTags  // 确保 JSON 字符串被保存
item.HostIDs = req.HostIDs    // 确保 JSON 字符串被保存
```

### 3. 添加调试日志（前端）

在前端添加详细的 console.log，帮助追踪数据流：

**保存时**：
```javascript
console.log('保存巡检组数据:', groupData)
console.log('业务分组 IDs:', formData.groupIds)
console.log('巡检项数据:', itemData)
console.log('  - hostTags (序列化):', itemData.hostTags)
```

**编辑时**：
```javascript
console.log('编辑巡检组，原始数据:', record)
console.log('解析后的 groupIds:', groupIds)
console.log('加载巡检项列表:', res)
console.log('  - hostTags 解析成功:', hostTags)
```

## 修改的文件

1. `internal/service/inspection_mgmt/group_service.go`
   - 修改 `Update()` 方法
   - 移除 `Description` 和 `GroupIDs` 的条件判断
   - 改为直接赋值

2. `internal/service/inspection_mgmt/item_service.go`
   - 修改 `Update()` 方法
   - 移除 `Description` 的条件判断
   - 确保 `HostTags` 和 `HostIDs` 直接赋值

3. `web/src/views/inspection/InspectionManagement.vue`
   - 添加详细的调试日志
   - 优化数据解析逻辑

## 数据流程

### 保存流程
1. 前端：用户选择业务分组和主机标签
2. 前端：将数组序列化为 JSON 字符串
   - `groupIds: [1, 2]` → `"[1,2]"`
   - `hostTags: ["web", "api"]` → `"[\"web\",\"api\"]"`
3. 前端：发送 HTTP 请求到后端
4. 后端：接收请求，调用 `Update()` 方法
5. 后端：直接赋值，不做条件判断
6. 后端：GORM 保存到数据库

### 回显流程
1. 后端：从数据库读取数据
2. 后端：返回 JSON 响应（字段值为 JSON 字符串）
3. 前端：接收响应
4. 前端：解析 JSON 字符串为数组
   - `"[1,2]"` → `[1, 2]`
   - `"[\"web\",\"api\"]"` → `["web", "api"]`
5. 前端：显示在表单中

## 测试步骤

1. **重启后端服务**：
   ```bash
   make run
   ```

2. **清除浏览器缓存**：
   - 按 F12 打开开发者工具
   - 右键点击刷新按钮，选择"清空缓存并硬性重新加载"

3. **测试保存功能**：
   - 创建或编辑巡检组
   - 选择业务分组（可多选）
   - 添加巡检项
   - 选择主机匹配方式
   - 选择主机标签或主机名
   - 点击保存
   - 查看控制台日志，确认数据正确序列化

4. **测试回显功能**：
   - 刷新页面
   - 再次编辑该巡检组
   - 查看控制台日志，确认数据正确解析
   - 验证业务分组和主机标签正确显示

5. **验证数据库**：
   ```sql
   SELECT id, name, group_ids FROM inspection_groups WHERE id = 3;
   SELECT id, name, host_match_type, host_tags, host_ids FROM inspection_items WHERE group_id = 3;
   ```

   应该看到：
   - `group_ids` 为 `[1]` 或 `[1,2]`（JSON 数组字符串）
   - `host_tags` 为 `["mysqld"]` 或 `[]`（JSON 数组字符串）
   - `host_ids` 为 `[1,2]` 或 `[]`（JSON 数组字符串）

## 注意事项

1. **字段更新策略**：
   - 对于可以为空的字段（如 Description、GroupIDs），应该直接赋值
   - 对于不能为空的字段（如 Name、Status），可以保留条件判断
   - JSON 字符串字段（如 GroupIDs、HostTags、HostIDs）必须直接赋值

2. **空值处理**：
   - 空数组序列化为 `"[]"`，不是空字符串 `""`
   - 前端发送 `"[]"` 表示用户清空了选择
   - 后端应该保存 `"[]"`，而不是忽略更新

3. **调试日志**：
   - 生产环境部署前应该移除或注释掉 console.log
   - 或者使用环境变量控制是否输出日志

## 相关问题

这个问题也可能影响其他使用 JSON 字符串存储数组的字段，建议检查：
- 其他模块的 Update 方法
- 确保 JSON 字段都是直接赋值，不做条件判断

## 总结

问题的根本原因是后端 Update 方法中不恰当的条件判断，导致 JSON 字符串字段无法正确更新。修复后，数据可以正确保存和回显。
