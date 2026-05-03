# RSA 密码加密方案

## 概述

为提升登录安全性，OpsHub 采用 RSA 非对称加密方案对登录密码进行加密传输。即使在 HTTPS 被破解的情况下，攻击者也无法直接获取明文密码。

## 技术方案

### 架构设计

```
┌─────────────┐                    ┌─────────────┐
│   前端      │                    │   后端      │
│  (Vue 3)    │                    │   (Go)      │
└─────────────┘                    └─────────────┘
      │                                   │
      │  1. 获取 RSA 公钥                │
      │ ──────────────────────────────>  │
      │                                   │
      │  2. 返回公钥 (PEM 格式)          │
      │ <──────────────────────────────  │
      │                                   │
      │  3. 使用公钥加密密码             │
      │     (JSEncrypt)                   │
      │                                   │
      │  4. 发送加密密码                 │
      │ ──────────────────────────────>  │
      │                                   │
      │                    5. 使用私钥解密密码
      │                       (crypto/rsa)
      │                                   │
      │  6. 返回登录结果                 │
      │ <──────────────────────────────  │
```

### 核心组件

#### 后端

1. **RSA 密钥管理器** (`pkg/crypto/rsa.go`)
   - 单例模式，全局唯一
   - 启动时自动生成 2048 位 RSA 密钥对
   - 提供公钥获取和密码解密方法

2. **公钥接口** (`internal/server/rbac/http.go`)
   - `GET /api/v1/public/rsa-public-key`
   - 返回 PEM 格式的 RSA 公钥
   - 无需认证，公开访问

3. **登录接口改造** (`internal/service/rbac/user.go`)
   - 支持 `encryptedPassword` 字段（RSA 加密）
   - 保留 `password` 字段（明文，向后兼容）
   - 优先使用加密密码，解密后验证

#### 前端

1. **依赖库**
   - `jsencrypt`: 轻量级 RSA 加密库

2. **API 定义** (`web/src/api/auth.ts`)
   - `getRsaPublicKey()`: 获取 RSA 公钥
   - `LoginParams` 接口支持 `encryptedPassword` 字段

3. **登录页面** (`web/src/views/Login.vue`)
   - 页面加载时获取 RSA 公钥
   - 登录时使用公钥加密密码
   - 发送加密后的密码到后端

## 使用方式

### 前端加密流程

```typescript
import JSEncrypt from 'jsencrypt'
import { getRsaPublicKey } from '@/api/auth'

// 1. 获取公钥
const res = await getRsaPublicKey()
const publicKey = res.publicKey

// 2. 加密密码
const encrypt = new JSEncrypt()
encrypt.setPublicKey(publicKey)
const encryptedPassword = encrypt.encrypt(password)

// 3. 发送加密密码
await login({
  username: 'admin',
  encryptedPassword: encryptedPassword,
  captchaId: '...',
  captchaCode: '...'
})
```

### 后端解密流程

```go
import "github.com/ydcloud-dy/opshub/pkg/crypto"

// 1. 获取 RSA 管理器
rsaManager := crypto.GetRSAManager()

// 2. 解密密码
password, err := rsaManager.DecryptPassword(encryptedPassword)
if err != nil {
    return errors.New("密码解密失败")
}

// 3. 验证密码
user, err := userUseCase.ValidatePassword(ctx, username, password)
```

## 向后兼容

为确保平滑升级，系统同时支持加密和明文密码：

1. **优先级**：`encryptedPassword` > `password`
2. **兼容期**：旧客户端仍可使用明文密码登录
3. **迁移建议**：所有客户端升级后，可移除明文密码支持

## 安全特性

### 优势

1. **传输安全**：密码在客户端加密，传输过程中为密文
2. **防重放攻击**：每次加密结果不同（RSA PKCS#1 v1.5 填充）
3. **密钥安全**：私钥仅存在于服务端内存，不落盘
4. **向后兼容**：不影响现有系统运行

### 注意事项

1. **密钥管理**
   - 私钥存储在服务端内存中
   - 服务重启会生成新的密钥对
   - 生产环境建议持久化密钥（可选）

2. **密钥轮换**
   - 当前实现：服务重启自动轮换
   - 未来可扩展：定期轮换机制

3. **性能影响**
   - RSA 加密/解密仅在登录时执行
   - 对系统整体性能影响可忽略

## 测试

### 单元测试

```bash
# 运行 RSA 加密测试
go test -v ./pkg/crypto/
```

### 集成测试

1. 启动后端服务
2. 访问登录页面
3. 输入用户名密码
4. 查看网络请求，确认密码已加密

### 验证方法

```bash
# 1. 获取公钥
curl http://localhost:9876/api/v1/public/rsa-public-key

# 2. 使用 JSEncrypt 加密密码（在浏览器控制台）
const encrypt = new JSEncrypt()
encrypt.setPublicKey('公钥内容')
const encrypted = encrypt.encrypt('123456')
console.log(encrypted)

# 3. 使用加密密码登录
curl -X POST http://localhost:9876/api/v1/public/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "encryptedPassword": "加密后的密码",
    "captchaId": "...",
    "captchaCode": "..."
  }'
```

## 未来优化

1. **密钥持久化**
   - 将私钥加密存储到配置文件
   - 支持从环境变量加载密钥

2. **密钥版本管理**
   - 支持多版本密钥共存
   - 平滑过渡到新密钥

3. **更强的加密算法**
   - 升级到 RSA OAEP 填充
   - 支持 ECC 椭圆曲线加密

4. **移除明文密码支持**
   - 所有客户端升级后
   - 移除 `password` 字段
   - 强制使用加密密码

## 相关文件

### 后端

- `pkg/crypto/rsa.go` - RSA 密钥管理器
- `pkg/crypto/rsa_test.go` - 单元测试
- `internal/service/rbac/user.go` - 登录服务（解密逻辑）
- `internal/server/rbac/http.go` - 公钥接口

### 前端

- `web/src/api/auth.ts` - 认证 API
- `web/src/views/Login.vue` - 登录页面（加密逻辑）
- `web/package.json` - 依赖配置（jsencrypt）

## 更新日志

### 2026-05-03

- ✅ 实现 RSA 密钥管理器
- ✅ 添加公钥获取接口
- ✅ 改造登录接口支持加密密码
- ✅ 前端集成 JSEncrypt 加密库
- ✅ 登录页面实现密码加密
- ✅ 编写单元测试
- ✅ 向后兼容明文密码
