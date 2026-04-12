# API Key 管理功能 - 测试指南

## 功能已完成

✅ 后端 API 实现完成
✅ 前端页面实现完成
✅ 数据库迁移已添加
✅ 认证中间件已增强
✅ 编译错误已修复

## 测试步骤

### 1. 启动服务

```bash
# 启动后端服务
cd /Users/Zhuanz/golang_project/src/opshub
./bin/opshub server -c config/config.yaml

# 前端已在运行（端口 5173）
# 如需重启：cd web && npm run dev
```

### 2. 访问功能

1. 打开浏览器访问：http://localhost:5173
2. 使用管理员账号登录（admin / 123456）
3. 进入"系统管理 → 系统配置"
4. 在左侧导航点击"API Key管理"

### 3. 功能测试

#### 3.1 创建 API Key

1. 点击"新增 API Key"按钮
2. 输入名称（必填）：如"测试密钥"
3. 输入描述（可选）：如"用于自动化脚本"
4. 点击"确定"
5. **重要**：弹窗显示完整密钥，立即复制保存
6. 点击"我已复制，关闭"

**预期结果**：
- 创建成功提示
- 完整密钥格式：`opshub_` + 32位十六进制字符
- 列表中新增一条记录，密钥显示为脱敏格式（`opshub_xxx***abcd`）

#### 3.2 查看列表

列表应显示以下信息：
- ID
- 名称
- 脱敏密钥（前缀+***+后缀）
- 描述
- 调用次数（初始为 0）
- 最后调用时间（初始为"从未调用"）
- 创建时间
- 删除按钮

#### 3.3 使用 API Key

```bash
# 使用刚创建的 API Key 调用接口
curl -X GET http://localhost:9876/api/v1/asset/hosts \
  -H "Authorization: Bearer opshub_你的密钥"
```

**预期结果**：
- 接口正常返回数据（与使用 JWT Token 效果相同）
- 列表中该 Key 的调用次数 +1
- 最后调用时间更新为当前时间

#### 3.4 删除 API Key

1. 点击某个 API Key 的"删除"按钮
2. 确认删除提示："删除后该 API Key 将立即失效，无法恢复，确认删除？"
3. 点击"确定"

**预期结果**：
- 删除成功提示
- 列表中该记录消失
- 再次使用该 Key 调用接口返回 401 未授权

### 4. 安全性验证

#### 4.1 密钥不可再次查看

1. 创建 API Key 后关闭弹窗
2. 刷新页面
3. 尝试查看完整密钥

**预期结果**：无任何方式可以查看完整密钥，只能看到脱敏格式

#### 4.2 数据库存储验证

```bash
# 进入 MySQL 容器
docker exec -it opshub-mysql mysql -uroot -p'OpsHub@2026' opshub

# 查询 API Key 表
SELECT id, name, key_hash, key_prefix, key_suffix FROM sys_api_keys;
```

**预期结果**：
- `key_hash` 字段存储的是 SHA256 哈希值（64位十六进制）
- 没有任何字段存储完整明文密钥

#### 4.3 权限验证

使用 API Key 调用各类接口，验证拥有超级管理员权限：

```bash
# 查询主机列表
curl -H "Authorization: Bearer opshub_xxx" http://localhost:9876/api/v1/asset/hosts

# 创建主机
curl -X POST http://localhost:9876/api/v1/asset/hosts \
  -H "Authorization: Bearer opshub_xxx" \
  -H "Content-Type: application/json" \
  -d '{"name":"test","ip":"192.168.1.1","port":22}'

# 查询用户列表
curl -H "Authorization: Bearer opshub_xxx" http://localhost:9876/api/v1/users

# 查询系统配置
curl -H "Authorization: Bearer opshub_xxx" http://localhost:9876/api/v1/system/config
```

**预期结果**：所有接口均可正常访问，无权限限制

### 5. Redis 统计验证

```bash
# 进入 Redis 容器
docker exec -it opshub-redis redis-cli -a '1ujasdJ67Ps'

# 查看调用次数
GET apikey:call_count:{key_hash}

# 查看最后调用时间
GET apikey:last_called:{key_hash}
```

**预期结果**：
- 调用次数为整数
- 最后调用时间为 Unix 时间戳

### 6. 边界测试

#### 6.1 空名称测试
- 创建时不输入名称，点击确定
- **预期**：提示"请输入 API Key 名称"

#### 6.2 删除后立即失效
- 删除一个 API Key
- 立即使用该 Key 调用接口
- **预期**：返回 401 未授权

#### 6.3 并发调用统计
- 使用同一个 API Key 快速调用多次接口
- 查看列表中的调用次数
- **预期**：调用次数准确累加

## 常见问题排查

### 问题1：前端页面报错 "IconKey is not exported"

**原因**：Arco Design 没有 IconKey 图标
**解决**：已修复，使用 IconSafe 替代

### 问题2：后端编译错误

**检查**：
```bash
go build -o /dev/null ./internal/biz/system/...
go build -o /dev/null ./internal/data/system/...
go build -o /dev/null ./internal/service/system/...
```

### 问题3：API Key 认证失败

**检查清单**：
1. 确认 API Key 格式正确（以 `opshub_` 开头）
2. 确认 API Key 未被删除
3. 检查 Authorization header 格式：`Bearer opshub_xxx`
4. 查看后端日志是否有错误信息

### 问题4：统计数据不更新

**检查**：
1. Redis 是否正常运行
2. 后端是否正确连接 Redis
3. 查看 Redis 中的 key 是否存在

## 性能测试建议

```bash
# 使用 ab 进行压力测试
ab -n 1000 -c 10 -H "Authorization: Bearer opshub_xxx" \
  http://localhost:9876/api/v1/asset/hosts

# 验证统计数据准确性
```

## 安全建议

1. **定期轮换**：建议每3-6个月轮换一次 API Key
2. **最小权限**：虽然当前 API Key 拥有超级管理员权限，但建议后续版本支持权限细化
3. **监控异常**：定期检查调用次数异常的 API Key
4. **及时删除**：不再使用的 API Key 应立即删除
5. **安全存储**：API Key 应存储在安全的密钥管理系统中，不要硬编码在代码中

## 后续优化方向

1. **过期时间**：支持设置 API Key 有效期
2. **权限范围**：支持为 API Key 配置特定权限
3. **IP 白名单**：限制 API Key 只能从特定 IP 访问
4. **速率限制**：对 API Key 调用频率进行限制
5. **审计日志**：详细记录 API Key 的所有操作
6. **批量管理**：支持批量创建、删除 API Key
7. **使用分析**：提供 API Key 使用情况的可视化报表
