# Web站点管理 - 密码复制问题修复

## 问题描述

用户设置站点访问密码并保存成功后，点击"复制密码"按钮时提示"该站点未设置访问密码"。

## 问题分析

### 1. 前端接口定义缺失

**问题**: `web/src/api/website.ts` 中的 `Website` 接口缺少 `accessPassword` 字段

**影响**: TypeScript 类型检查无法识别密码字段，导致数据处理不完整

**修复**: 添加 `accessPassword?: string` 字段到接口定义

```typescript
export interface Website {
  // ... 其他字段
  accessPassword?: string  // 访问密码（仅在详情接口返回）
}
```

### 2. 后端密码加密逻辑

**现有逻辑**:
- 创建站点时：如果提供密码，则加密后存储
- 更新站点时：如果提供密码，则加密后更新；如果不提供，保留原密码
- 获取详情时：通过 `toVOWithPassword` 方法解密密码返回

**工作流程**:
```
创建/更新 → 加密存储 → 数据库(加密密码)
获取详情 → 解密返回 → 前端(明文密码)
```

### 3. 密码加密实现

使用 AES-256-GCM 加密算法：

```go
// 加密密钥（32字节）
encryptionKey := []byte("opshub-enc-key-32-bytes-long!!!!")

// 加密流程
plaintext → AES-GCM → base64编码 → 存储

// 解密流程
存储 → base64解码 → AES-GCM → plaintext
```

### 4. API 接口行为

#### 列表接口 `/api/v1/websites`
- 不返回 `accessPassword` 字段（安全考虑）
- 只返回 `accessUser`（用户名）

#### 详情接口 `/api/v1/websites/:id`
- 返回完整信息，包括解密后的 `accessPassword`
- 通过 `toVOWithPassword` 方法处理

## 修复方案

### 1. 前端修复

#### 添加类型定义
```typescript
// web/src/api/website.ts
export interface Website {
  accessPassword?: string  // 新增字段
}
```

#### 优化复制逻辑
```typescript
const handleCopyPassword = async (record: Website) => {
  try {
    // 调用详情接口获取完整信息（包括解密后的密码）
    const res = await getWebsite(record.id)
    const website = res

    // 添加调试日志
    console.log('获取到的站点信息:', website)
    console.log('访问密码:', website.accessPassword)

    // 检查密码是否存在
    if (!website.accessPassword || website.accessPassword === '') {
      Message.warning('该站点未设置访问密码')
      return
    }

    // 复制到剪贴板
    await navigator.clipboard.writeText(website.accessPassword)
    Message.success('密码已复制到剪贴板')
  } catch (error: any) {
    console.error('复制密码失败:', error)
    Message.error('复制失败: ' + error.message)
  }
}
```

### 2. 后端修复

#### 更新逻辑优化
```go
// 更新站点时，只有明确提供了新密码才更新
if req.AccessPassword != "" {
    encrypted, err := uc.encrypt(req.AccessPassword)
    if err != nil {
        return fmt.Errorf("加密访问密码失败: %w", err)
    }
    website.AccessPassword = encrypted
}
// 注意：如果 req.AccessPassword 为空，保留原有密码不变
```

**说明**: 这样可以避免编辑站点时，如果不修改密码（密码字段为空），不会清空原有密码。

## 测试步骤

### 1. 创建站点并设置密码
```
1. 点击"新增站点"
2. 填写站点信息
3. 设置访问用户名和密码
4. 保存
```

### 2. 验证密码复制
```
1. 找到刚创建的站点
2. 点击"复制凭据"下拉菜单
3. 点击"复制密码"
4. 应该提示"密码已复制到剪贴板"
5. 粘贴验证密码内容正确
```

### 3. 验证编辑不影响密码
```
1. 编辑站点（不修改密码字段）
2. 保存
3. 再次复制密码
4. 密码应该仍然可以正常复制
```

### 4. 验证密码更新
```
1. 编辑站点
2. 修改密码为新密码
3. 保存
4. 复制密码
5. 应该复制到新密码
```

## 调试方法

### 前端调试
打开浏览器控制台，查看日志输出：
```
获取到的站点信息: { id: 1, name: "测试站点", accessPassword: "123456", ... }
访问密码: 123456
```

### 后端调试
查看后端日志，确认：
1. 密码是否正确加密存储
2. 详情接口是否正确解密返回

### 数据库验证
```sql
-- 查看加密后的密码（应该是 base64 编码的密文）
SELECT id, name, access_user, access_password FROM websites;
```

## 安全考虑

1. **传输安全**: 使用 HTTPS 传输密码
2. **存储安全**: 使用 AES-256-GCM 加密存储
3. **访问控制**: 只有详情接口返回密码，列表接口不返回
4. **权限验证**: 需要通过 JWT 认证才能访问
5. **密钥管理**: 加密密钥应该从配置文件或环境变量读取（待优化）

## 后续优化建议

1. **密钥管理**: 将加密密钥移到配置文件，支持密钥轮换
2. **密码强度**: 添加密码强度验证
3. **操作审计**: 记录密码复制操作到审计日志
4. **权限细化**: 添加"查看密码"权限，不是所有人都能复制密码
5. **密码过期**: 支持密码过期提醒
6. **密码历史**: 记录密码修改历史

## 相关文件

### 前端
- `web/src/api/website.ts` - API 接口定义
- `web/src/views/asset/Websites.vue` - 站点管理页面

### 后端
- `internal/biz/asset/website.go` - 站点模型定义
- `internal/biz/asset/website_usecase.go` - 站点业务逻辑（加密/解密）
- `internal/service/asset/website_service.go` - 站点服务层
- `internal/data/asset/website.go` - 站点数据访问层
