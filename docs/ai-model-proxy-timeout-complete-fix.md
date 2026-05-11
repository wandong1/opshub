# AI模型代理超时问题完整修复报告

## 问题现象

使用AI模型代理时，长时间处理会中断：
- **短时间处理（< 30秒）**：正常
- **长时间处理（> 30秒）**：连接中断，n8n报错"terminated"

**Agent日志显示：**
```
[18:39:27] 流式代理进度: chunks=700, bytes=88245
# 数据还在传输，但客户端已断开
```

---

## 根本原因分析

经过深入排查，发现了**三个层级的超时问题**：

### 1. ❌ HTTP服务器WriteTimeout（最关键）
**位置：** `internal/server/http.go`

**问题：** HTTP服务器设置了`WriteTimeout`，这会限制整个响应的写入时间。

```go
// 错误配置
s.server = &http.Server{
    WriteTimeout: time.Duration(conf.Server.WriteTimeout) * time.Millisecond,
}
```

**影响：** 无论数据是否在持续传输，只要写入时间超过配置值（通常30-60秒），连接就会被Go的HTTP服务器强制关闭。

**这是导致客户端提前断开的直接原因！**

### 2. ❌ Agent端ResponseHeaderTimeout
**位置：** `agent/internal/client/grpc_client.go`

**问题：** 设置了30秒的响应头超时。

```go
// 错误配置
Transport: &http.Transport{
    ResponseHeaderTimeout: 30 * time.Second,
}
```

**影响：** 如果AI模型思考时间超过30秒才开始返回第一个token，就会超时。

### 3. ❌ AgentHub的总超时
**位置：** `internal/server/agent/hub.go`

**问题：** 使用固定的总超时Timer。

```go
// 错误实现
timer := time.NewTimer(timeout)
<-timer.C
// 超时后关闭channel
```

**影响：** 无论数据是否在传输，到时间就强制关闭。

---

## 完整修复方案

### 修复1：移除HTTP服务器WriteTimeout ⭐ 最关键

**文件：** `internal/server/http.go`

```go
// 修复后
s.server = &http.Server{
    Addr:         fmt.Sprintf(":%d", conf.Server.HttpPort),
    Handler:      router,
    ReadTimeout:  time.Duration(conf.Server.ReadTimeout) * time.Millisecond,
    WriteTimeout: 0, // 不限制写入超时，支持流式响应
    IdleTimeout:  120 * time.Second, // 空闲连接超时
}
```

**原理：**
- `WriteTimeout: 0` 表示不限制写入时间
- `IdleTimeout` 控制空闲连接的超时
- 流式响应可以持续任意长时间

### 修复2：移除Agent端ResponseHeaderTimeout

**文件：** `agent/internal/client/grpc_client.go`

```go
// 修复后
client := &http.Client{
    Timeout: 0, // 不设置总超时
    Transport: &http.Transport{
        MaxIdleConns:          100,
        MaxIdleConnsPerHost:   100,
        IdleConnTimeout:       90 * time.Second,
        ResponseHeaderTimeout: 0,  // 不限制等待响应头，允许模型慢慢思考
        ExpectContinueTimeout: 1 * time.Second,
        DisableCompression:    true,
    },
}

// 使用Context控制最大时间
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
defer cancel()
httpReq = httpReq.WithContext(ctx)
```

**原理：**
- 移除所有HTTP层面的超时限制
- 使用Context的30分钟超时作为最大保护
- 允许模型任意长时间思考和输出

### 修复3：移除AgentHub的超时限制

**文件：** `internal/server/agent/hub.go`

```go
// 修复后
func (h *AgentHub) StreamResponse(as *AgentStream, requestID string, timeout time.Duration) (<-chan *pb.StreamProxyChunk, error) {
    chunkChan := make(chan *pb.StreamProxyChunk, 10)
    
    as.pendMu.Lock()
    as.pending[requestID] = chunkChan
    as.pendMu.Unlock()
    
    // 不再设置超时，让请求自然完成或由客户端断开
    return chunkChan, nil
}
```

**原理：**
- 中间层不应该设置总超时
- 超时由边缘控制（客户端、Agent端）

---

## 超时层级对比

### 修复前（多层超时限制）

| 层级 | 超时类型 | 超时时间 | 问题 |
|------|---------|---------|------|
| HTTP服务器 | WriteTimeout | 30-60秒 | ❌ **最致命**，强制断开连接 |
| AgentHub | 总超时 | 300秒 | ❌ 到时间就关闭 |
| Agent HTTP Client | 总超时 | 600秒 | ❌ 限制总时间 |
| Agent HTTP Transport | ResponseHeaderTimeout | 30秒 | ❌ 模型思考超时 |

### 修复后（边缘控制）

| 层级 | 超时类型 | 超时时间 | 说明 |
|------|---------|---------|------|
| HTTP服务器 | WriteTimeout | 0（无限制） | ✅ 支持流式响应 |
| HTTP服务器 | IdleTimeout | 120秒 | ✅ 空闲连接超时 |
| AgentHub | 无超时 | - | ✅ 由边缘控制 |
| Agent HTTP Client | Context超时 | 30分钟 | ✅ 最大保护 |
| Agent HTTP Transport | ResponseHeaderTimeout | 0（无限制） | ✅ 允许慢思考 |

---

## 部署步骤

### 1. 编译新版本

```bash
# 编译服务器端
go build -o bin/opshub ./main.go

# 编译Agent客户端
cd agent
go build -o opshub-agent main.go
```

### 2. 部署服务器端

```bash
# 停止服务
systemctl stop opshub

# 备份旧版本
cp /usr/local/bin/opshub /usr/local/bin/opshub.bak

# 替换新版本
cp bin/opshub /usr/local/bin/opshub

# 启动服务
systemctl start opshub

# 查看日志
journalctl -u opshub -f
```

### 3. 部署Agent客户端

```bash
# 在每台Agent主机上执行
systemctl stop opshub-agent
cp opshub-agent /usr/local/bin/opshub-agent
systemctl start opshub-agent

# 查看日志
journalctl -u opshub-agent -f
```

---

## 测试验证

### 测试1：短时间处理（< 1分钟）

```bash
curl -X POST http://localhost:8080/api/v1/ai-model-proxy/{TOKEN}/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama2",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'
```

**预期：** 正常返回流式响应

### 测试2：中等时间处理（1-5分钟）

```bash
curl -X POST http://localhost:8080/api/v1/ai-model-proxy/{TOKEN}/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama2",
    "messages": [{"role": "user", "content": "Write a detailed story with 5000 words"}],
    "stream": true
  }'
```

**预期：** 持续返回流式响应，不会中断

### 测试3：长时间处理（5-15分钟）

```bash
curl -X POST http://localhost:8080/api/v1/ai-model-proxy/{TOKEN}/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama2",
    "messages": [{"role": "user", "content": "Generate a very long document with 20000 words"}],
    "stream": true
  }'
```

**预期：** 持续返回流式响应，直到完成

### 测试4：超长时间处理（15-30分钟）

```bash
curl -X POST http://localhost:8080/api/v1/ai-model-proxy/{TOKEN}/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama2",
    "messages": [{"role": "user", "content": "Generate 50000 words"}],
    "stream": true
  }'
```

**预期：** 持续返回流式响应，直到完成或达到30分钟上限

---

## 日志监控

### Agent端日志（正常情况）

```
[INFO] 收到流式代理请求: requestID=xxx
[INFO] 流式代理请求开始: status=200
[INFO] 流式代理进度: chunks=100, bytes=12573
[INFO] 流式代理进度: chunks=200, bytes=25158
[INFO] 流式代理进度: chunks=300, bytes=37760
...
[INFO] 流式代理请求完成: totalBytes=xxx, chunks=xxx
```

### 服务器端日志（正常情况）

```
[INFO] AI模型代理请求开始
[INFO] AI模型代理请求完成: total_bytes=xxx
```

### 异常情况

如果看到以下日志，说明还有问题：
```
[ERROR] timeout awaiting response headers  ← ResponseHeaderTimeout问题
[ERROR] context deadline exceeded          ← Context超时（30分钟）
[WARN] 客户端断开连接                     ← 客户端主动断开
```

---

## 关键改进点总结

1. ✅ **移除HTTP服务器WriteTimeout** - 最关键的修复
2. ✅ **移除Agent端ResponseHeaderTimeout** - 允许模型慢思考
3. ✅ **移除AgentHub总超时** - 中间层不限制
4. ✅ **使用Context超时** - 30分钟作为最大保护
5. ✅ **添加IdleTimeout** - 120秒空闲连接超时
6. ✅ **客户端断开检测** - 服务器端监听客户端断开

---

## 技术原理

### Go HTTP服务器的WriteTimeout

`WriteTimeout` 是从**读取完请求头**到**写完整个响应**的时间限制。

对于流式响应：
- 开始写入第一个字节 → 计时开始
- 持续写入数据 → 计时继续
- 超过WriteTimeout → **连接被强制关闭**

**这就是为什么Agent端日志显示数据还在传输，但客户端已经断开的原因！**

### 正确的流式响应配置

```go
WriteTimeout: 0  // 不限制写入时间
IdleTimeout: 120 * time.Second  // 空闲连接超时
```

- `WriteTimeout: 0` 允许无限长的写入时间
- `IdleTimeout` 控制空闲连接（没有读写活动）的超时

---

## 注意事项

### 1. 配置文件中的WriteTimeout

如果配置文件中设置了`write_timeout`，现在会被忽略（代码中硬编码为0）。

### 2. 资源管理

虽然移除了WriteTimeout，但仍有保护机制：
- Context 30分钟超时
- IdleTimeout 120秒
- 客户端断开自动清理

### 3. 向后兼容

- 新版本服务器兼容旧版本Agent
- 新版本Agent兼容旧版本服务器
- 但要获得完整的长时间处理支持，需要同时更新

---

## 总结

**问题根源：** HTTP服务器的WriteTimeout限制了整个响应的写入时间

**解决方案：** 移除所有不适合流式响应的超时限制，使用边缘控制

**效果：** 支持任意长时间的流式处理（最长30分钟）

**部署要求：** 需要同时更新服务器端和Agent客户端

---

**修复日期：** 2026-05-11  
**影响范围：** 服务器端 + Agent客户端  
**测试状态：** 待验证  
**优先级：** 🔥 高优先级（影响核心功能）
