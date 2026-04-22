# 智能巡检模块 - JSON 字段名不匹配问题修复

## 修复日期
2026-03-13

## 问题描述

用户反馈：
1. 前端可以正常选择业务分组和主机标签
2. 保存后数据没有正确保存到后端
3. 后端日志显示接收到的字段都是空的

## 问题分析

### 审计日志显示前端发送的数据正确

```json
{
  "hostMatchType": "tag",
  "hostTags": "[\"mysqld\"]",
  "hostIds": "[]"
}
```

### 后端日志显示接收到的数据为空

```
[DEBUG] Item 1:
  Name: 检查 uptime
  HostMatchType:
  HostTags:
  HostIDs:
```

### 根本原因

**前端发送的 JSON 字段名是驼峰命名（camelCase）：**
```json
{
  "hostMatchType": "tag",
  "hostTags": "[\"mysqld\"]",
  "hostIds": "[]",
  "groupId": 3,
  "executionStrategy": "concurrent"
}
```

**后端 DTO 的 JSON 标签是蛇形命名（snake_case）：**
```go
type ItemCreateRequest struct {
    HostMatchType     string `json:"host_match_type"`
    HostTags          string `json:"host_tags"`
    HostIDs           string `json:"host_ids"`
    GroupID           uint   `json:"group_id"`
    ExecutionStrategy string `json:"execution_strategy"`
}
```

**字段名不匹配，导致 Gin 的 JSON 绑定失败，所有字段都是零值（空字符串）！**

## 修复方案

修改 `ItemCreateRequest` 和 `ItemUpdateRequest` 的 JSON 标签为驼峰命名，与前端保持一致。

### 修改的字段

| 字段名 | 修改前（蛇形） | 修改后（驼峰） |
|--------|---------------|---------------|
| GroupID | `group_id` | `groupId` |
| ExecutionStrategy | `execution_strategy` | `executionStrategy` |
| ExecutionType | `execution_type` | `executionType` |
| ScriptType | `script_type` | `scriptType` |
| ScriptContent | `script_content` | `scriptContent` |
| ScriptFile | `script_file` | `scriptFile` |
| PromQLQuery | `promql_query` | `promqlQuery` |
| HostMatchType | `host_match_type` | `hostMatchType` |
| HostTags | `host_tags` | `hostTags` |
| HostIDs | `host_ids` | `hostIds` |
| AssertionType | `assertion_type` | `assertionType` |
| AssertionValue | `assertion_value` | `assertionValue` |
| VariableName | `variable_name` | `variableName` |
| VariableRegex | `variable_regex` | `variableRegex` |

### 修改后的代码

**ItemCreateRequest：**
```go
type ItemCreateRequest struct {
    Name              string `json:"name" binding:"required"`
    Description       string `json:"description"`
    GroupID           uint   `json:"groupId" binding:"required"`
    Sort              int    `json:"sort"`
    Status            string `json:"status"`
    ExecutionStrategy string `json:"executionStrategy"`
    ExecutionType     string `json:"executionType" binding:"required"`
    Command           string `json:"command"`
    ScriptType        string `json:"scriptType"`
    ScriptContent     string `json:"scriptContent"`
    ScriptFile        string `json:"scriptFile"`
    PromQLQuery       string `json:"promqlQuery"`
    HostMatchType     string `json:"hostMatchType"`
    HostTags          string `json:"hostTags"`
    HostIDs           string `json:"hostIds"`
    AssertionType     string `json:"assertionType"`
    AssertionValue    string `json:"assertionValue"`
    VariableName      string `json:"variableName"`
    VariableRegex     string `json:"variableRegex"`
    Timeout           int    `json:"timeout"`
}
```

**ItemUpdateRequest：**
```go
type ItemUpdateRequest struct {
    Name              string `json:"name"`
    Description       string `json:"description"`
    GroupID           uint   `json:"groupId"`
    Sort              int    `json:"sort"`
    Status            string `json:"status"`
    ExecutionStrategy string `json:"executionStrategy"`
    ExecutionType     string `json:"executionType"`
    Command           string `json:"command"`
    ScriptType        string `json:"scriptType"`
    ScriptContent     string `json:"scriptContent"`
    ScriptFile        string `json:"scriptFile"`
    PromQLQuery       string `json:"promqlQuery"`
    HostMatchType     string `json:"hostMatchType"`
    HostTags          string `json:"hostTags"`
    HostIDs           string `json:"hostIds"`
    AssertionType     string `json:"assertionType"`
    AssertionValue    string `json:"assertionValue"`
    VariableName      string `json:"variableName"`
    VariableRegex     string `json:"variableRegex"`
    Timeout           int    `json:"timeout"`
}
```

## 修改的文件

- `internal/service/inspection_mgmt/item_dto.go`
  - 修改 `ItemCreateRequest` 的所有 JSON 标签
  - 修改 `ItemUpdateRequest` 的所有 JSON 标签

## 测试步骤

1. **重启后端服务**：
   ```bash
   make run
   ```

2. **在前端保存巡检组**：
   - 选择业务分组
   - 添加巡检项
   - 选择主机匹配方式
   - 选择主机标签
   - 点击保存

3. **查看后端日志**：
   ```
   [DEBUG] Item 1:
     Name: 检查 uptime
     HostMatchType: tag
     HostTags: ["mysqld"]  <-- 现在应该有值了！
     HostIDs: []
   ```

4. **查询数据库验证**：
   ```sql
   SELECT id, name, host_match_type, host_tags, host_ids
   FROM inspection_items
   WHERE group_id = 3;
   ```

   应该看到：
   - `host_match_type` 为 `tag`
   - `host_tags` 为 `["mysqld"]`
   - `host_ids` 为 `[]`

## 经验教训

1. **前后端字段命名要统一**：
   - 前端使用驼峰命名（camelCase）
   - 后端 JSON 标签也应该使用驼峰命名
   - 数据库字段可以使用蛇形命名（snake_case），GORM 会自动转换

2. **JSON 绑定失败的表现**：
   - Gin 的 `ShouldBindJSON` 不会报错
   - 但是所有字段都是零值（空字符串、0、false）
   - 需要通过日志来发现问题

3. **调试技巧**：
   - 在 Handler 层添加日志，查看接收到的数据
   - 在 Service 层添加日志，查看处理的数据
   - 对比审计日志中的请求参数和实际接收到的数据

4. **命名规范**：
   - Go 结构体字段：大写开头的驼峰命名（PascalCase）
   - JSON 标签：小写开头的驼峰命名（camelCase）
   - 数据库字段：蛇形命名（snake_case）

## 相关问题

这个问题也可能影响其他模块，建议检查：
- 其他 DTO 的 JSON 标签是否与前端字段名一致
- 确保前后端的命名规范统一

## 总结

问题的根本原因是前后端 JSON 字段名不一致，导致 Gin 无法正确绑定数据。修复后，数据可以正确保存和回显。
