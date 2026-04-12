# API Key 管理功能实现总结

## 功能概述

在系统管理 → 系统配置下新增了 **API Key 管理** 功能，用于生成和管理具有超级管理员权限的永久有效 API Key。

## 核心特性

### 1. 安全机制
- **密钥生成**：40字符随机密钥（格式：`opshub_` + 32位十六进制）
- **哈希存储**：数据库仅存储 SHA256 哈希值，不存明文
- **一次性展示**：完整密钥仅在创建时返回一次，之后永久无法查看
- **脱敏展示**：列表页显示格式为 `前缀***后缀`（如 `opshub_xxx***abcd`）

### 2. 权限控制
- API Key 拥有完全等同于超级管理员的权限
- 可用于所有需要认证的 API 接口
- 删除后立即全局失效

### 3. 使用统计
- **调用次数**：每次使用自动累加（存储在 Redis）
- **最后调用时间**：记录最近一次使用时间（存储在 Redis）
- 列表页实时展示统计数据

## 技术实现

### 后端架构（Go）

#### 1. 数据模型 (`internal/biz/system/model.go`)
```go
type SysAPIKey struct {
    ID          uint
    Name        string  // API Key 名称
    KeyHash     string  // SHA256 哈希值
    KeyPrefix   string  // 前缀（用于脱敏展示）
    KeySuffix   string  // 后缀（用于脱敏展示）
    Description string
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt
}
```

#### 2. Repository 层 (`internal/data/system/apikey_repository.go`)
- `Create()` - 创建 API Key
- `GetByID()` - 根据 ID 查询
- `GetByKeyHash()` - 根据哈希值查询（用于验证）
- `List()` - 分页列表
- `Delete()` - 删除

#### 3. UseCase 层 (`internal/biz/system/apikey_usecase.go`)
- `GenerateAPIKey()` - 生成随机密钥
- `HashAPIKey()` - SHA256 哈希
- `CreateAPIKey()` - 创建并返回完整密钥
- `ListAPIKeys()` - 列表查询（合并 Redis 统计数据）
- `DeleteAPIKey()` - 删除（同时清理 Redis）
- `VerifyAPIKey()` - 验证密钥有效性
- `RecordAPIKeyUsage()` - 记录使用情况

#### 4. Service 层 (`internal/service/system/apikey.go`)
- `POST /api/v1/system/apikeys` - 创建 API Key
- `GET /api/v1/system/apikeys` - 获取列表
- `DELETE /api/v1/system/apikeys/:id` - 删除

#### 5. 认证中间件增强 (`internal/service/rbac/middleware.go`)
```go
// AuthRequired() 中增加 API Key 认证逻辑
if strings.HasPrefix(token, "opshub_") {
    apiKey, err := m.apiKeyUseCase.VerifyAPIKey(ctx, token)
    if err == nil {
        // 记录使用
        m.apiKeyUseCase.RecordAPIKeyUsage(ctx, apiKey.KeyHash)
        // 设置超级管理员标识
        c.Set("is_api_key", true)
        c.Next()
        return
    }
}
```

### 前端实现（Vue 3 + TypeScript）

#### 1. API 客户端 (`web/src/api/apikey.ts`)
```typescript
export const createAPIKey = (data: CreateAPIKeyRequest) => {...}
export const listAPIKeys = (params: ListAPIKeysParams) => {...}
export const deleteAPIKey = (id: number) => {...}
```

#### 2. 管理页面 (`web/src/views/system/APIKeyManagement.vue`)
- **列表展示**：ID、名称、脱敏密钥、描述、调用次数、最后调用时间、创建时间
- **新增功能**：弹窗表单输入名称和描述
- **密钥展示**：创建成功后弹窗显示完整密钥，支持一键复制
- **删除功能**：二次确认，提示删除后立即失效

## 数据库表结构

```sql
CREATE TABLE `sys_api_keys` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL COMMENT 'API Key名称',
  `key_hash` varchar(64) NOT NULL COMMENT 'API Key哈希值(SHA256)',
  `key_prefix` varchar(10) NOT NULL COMMENT '密钥前缀(用于展示)',
  `key_suffix` varchar(10) NOT NULL COMMENT '密钥后缀(用于展示)',
  `description` varchar(500) DEFAULT NULL COMMENT '描述',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sys_api_keys_key_hash` (`key_hash`),
  KEY `idx_sys_api_keys_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

## Redis 数据结构

```
apikey:call_count:{key_hash}    -> 调用次数（整数）
apikey:last_called:{key_hash}   -> 最后调用时间（Unix 时间戳）
```

## 使用示例

### 1. 创建 API Key
```bash
curl -X POST http://localhost:9876/api/v1/system/apikeys \
  -H "Authorization: Bearer <admin_jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "测试密钥",
    "description": "用于自动化脚本"
  }'
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "测试密钥",
    "apiKey": "opshub_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
    "description": "用于自动化脚本",
    "createdAt": "2026-04-07T10:00:00Z"
  }
}
```

### 2. 使用 API Key 调用接口
```bash
curl -X GET http://localhost:9876/api/v1/asset/hosts \
  -H "Authorization: Bearer opshub_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
```

### 3. 查看 API Key 列表
```bash
curl -X GET "http://localhost:9876/api/v1/system/apikeys?page=1&page_size=20" \
  -H "Authorization: Bearer <admin_jwt_token>"
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 1,
    "page": 1,
    "page_size": 20,
    "data": [
      {
        "id": 1,
        "name": "测试密钥",
        "maskedKey": "opshub_xxx***o5p6",
        "description": "用于自动化脚本",
        "totalCalls": 42,
        "lastCalledAt": "2026-04-07T12:30:00Z",
        "createdAt": "2026-04-07T10:00:00Z"
      }
    ]
  }
}
```

## 安全注意事项

1. **密钥保管**：创建后立即复制保存，关闭弹窗后无法再次查看
2. **权限控制**：API Key 拥有超级管理员权限，请谨慎分发
3. **定期轮换**：建议定期删除旧密钥并创建新密钥
4. **监控使用**：通过调用次数和最后调用时间监控异常使用
5. **及时删除**：不再使用的密钥应立即删除

## 文件清单

### 后端文件
- `internal/biz/system/model.go` - 数据模型定义
- `internal/biz/system/apikey_usecase.go` - 业务逻辑层
- `internal/data/system/apikey_repository.go` - 数据访问层
- `internal/service/system/apikey.go` - HTTP 服务层
- `internal/server/system/http.go` - 路由注册
- `internal/service/rbac/middleware.go` - 认证中间件增强
- `cmd/server/server.go` - 数据库迁移

### 前端文件
- `web/src/api/apikey.ts` - API 客户端
- `web/src/views/system/APIKeyManagement.vue` - 管理页面

## 测试建议

1. **功能测试**
   - 创建 API Key 并验证完整密钥返回
   - 使用 API Key 调用各类接口验证权限
   - 删除 API Key 后验证立即失效
   - 验证列表页脱敏展示正确

2. **安全测试**
   - 验证数据库中无明文密钥
   - 验证关闭弹窗后无法再次获取完整密钥
   - 验证 API Key 权限等同于超级管理员

3. **性能测试**
   - 验证 Redis 统计数据更新正常
   - 验证高并发调用时统计准确性

## 后续优化建议

1. **过期机制**：支持设置 API Key 有效期
2. **权限细化**：支持为 API Key 配置特定权限范围
3. **IP 白名单**：限制 API Key 只能从特定 IP 访问
4. **速率限制**：对 API Key 调用频率进行限制
5. **审计日志**：记录 API Key 的所有操作日志
