# 长时间处理超时问题修复报告

## 问题描述

在使用AI模型代理时，当模型处理时间较长时会出现连接中断，n8n报错"terminated"。短时间处理正常，长时间处理异常。

## 根本原因

发现了**两个关键的超时问题**：

### 1. AgentHub的总超时问题
**位置：** `internal/server/agent/hub.go` - `StreamResponse`方法

**问题：** 使用了固定的总超时（Timer），无论数据是否在持续传输，到了超时时间就会强制关闭channel。

```go
// 错误的实现
timer := time.NewTimer(timeout)
<-timer.C
// 超时后关闭channel
```

**影响：** 即使模型正在持续输出数据，只要总时间超过配置的timeout（如300秒），就会被强制中断。

### 2. Agent端HTTP客户端的总超时问题
**位置：** `agent/internal/client/grpc_client.go` - `handleStreamProxyRequest`方法

**问题：** HTTP客户端设置了`Timeout`字段，这是一个**总超时**，包括连接、请求、响应的全部时间。

```go
// 错误的实现
client := &http.Client{
    Timeout: timeout, // 这会限制整个请求的总时间
}
```

**影响：** 对于流式响应，即使数据在持续传输，只要总时间超过timeout，HTTP客户端就会强制断开连接。

---

## 修复方案

### 修复1：移除AgentHub的超时限制

**文件：** `internal/server/agent/hub.go`

**修改：** 完全移除StreamResponse中的超时goroutine

```go
// StreamResponse 返回流式响应的channel（用于SSE）
// 注意：这里不设置超时，因为流式响应可能持续很长时间
// 超时应该由HTTP客户端（Agent端）和最终客户端控制
func (h *AgentHub) StreamResponse(as *AgentStream, requestID string, timeout time.Duration) (<-chan *pb.StreamProxyChunk, error) {
	chunkChan := make(chan *pb.StreamProxyChunk, 10)

	// 注册pending请求
	as.pendMu.Lock()
	as.pending[requestID] = chunkChan
	as.pendMu.Unlock()

	// 不再设置超时，让请求自然完成或由客户端断开
	return chunkChan, nil
}
```

**原理：** 
- 流式响应的超时应该由边缘控制（客户端断开、Agent端HTTP超时）
- 中间层（AgentHub）不应该设置总超时限制

### 修复2：Agent端使用Context超时替代Client超时

**文件：** `agent/internal/client/grpc_client.go`

**修改：** 移除HTTP Client的Timeout，使用Context控制超时

```go
// 创建支持长连接的HTTP客户端
// 注意：不设置Client.Timeout，因为流式响应可能持续很长时间
client := &http.Client{
	Timeout: 0, // 不设置总超时，允许长时间流式传输
	Transport: &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,  // 等待响应头的超时
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true,
	},
}

// 创建带超时的context，用于控制整个请求
// 但这个超时应该很长，主要是防止永久挂起
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
defer cancel()

httpReq = httpReq.WithContext(ctx)
```

**原理：**
- `Client.Timeout = 0` 表示不限制总时间
- `ResponseHeaderTimeout` 只限制等待响应头的时间（30秒）
- `Context.WithTimeout(30分钟)` 作为最大保护，防止永久挂起
- 30分钟足够处理大多数长时间的模型推理任务

---

## 超时机制对比

### 修复前

| 层级 | 超时类型 | 超时时间 | 问题 |
|------|---------|---------|------|
| AgentHub | 总超时 | 300秒 | ❌ 长时间处理会被中断 |
| Agent HTTP Client | 总超时 | 600秒 | ❌ 长时间处理会被中断 |

### 修复后

| 层级 | 超时类型 | 超时时间 | 说明 |
|------|---------|---------|------|
| AgentHub | 无超时 | - | ✅ 由客户端控制 |
| Agent HTTP Client | Context超时 | 30分钟 | ✅ 足够长，防止永久挂起 |
| Agent HTTP Transport | 响应头超时 | 30秒 | ✅ 只限制等待响应头 |

---

## 测试建议

### 1. 短时间处理测试（< 5分钟）
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

### 2. 长时间处理测试（5-15分钟）
```bash
curl -X POST http://localhost:8080/api/v1/ai-model-proxy/{TOKEN}/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama2",
    "messages": [{"role": "user", "content": "Write a very long story with 10000 words"}],
    "stream": true
  }'
```

**预期：** 持续返回流式响应，不会中断

### 3. 超长时间测试（> 15分钟）
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

## 部署步骤

### 1. 编译新版本
```bash
# 编译服务器端
go build -o bin/opshub ./main.go

# 编译Agent客户端
cd agent
go build -o opshub-agent main.go
```

### 2. 重启服务器
```bash
# 停止旧服务
systemctl stop opshub

# 替换二进制文件
cp bin/opshub /usr/local/bin/opshub

# 启动新服务
systemctl start opshub
```

### 3. 更新Agent客户端
```bash
# 在每台Agent主机上
systemctl stop opshub-agent
cp opshub-agent /usr/local/bin/opshub-agent
systemctl start opshub-agent
```

### 4. 验证
```bash
# 查看服务器日志
journalctl -u opshub -f

# 查看Agent日志
journalctl -u opshub-agent -f
```

---

## 监控建议

### 1. 查看流式代理进度
Agent端每100个chunk会记录一次进度：
```
流式代理进度: url=..., chunks=100, bytes=409600
流式代理进度: url=..., chunks=200, bytes=819200
```

### 2. 查看完成日志
```
流式代理请求完成: url=..., totalBytes=2048000, chunks=500
```

### 3. 监控指标
- 平均处理时间
- 最长处理时间
- 超时次数（应该为0）
- 传输字节数

---

## 关键改进点

1. ✅ **移除AgentHub总超时** - 不再限制流式响应的总时间
2. ✅ **移除HTTP Client总超时** - 允许长时间流式传输
3. ✅ **使用Context超时** - 30分钟作为最大保护
4. ✅ **保留响应头超时** - 30秒，防止连接挂起
5. ✅ **客户端断开检测** - 服务器端监听客户端断开

---

## 注意事项

### 1. 超时配置
前端创建代理时的"超时时间"配置现在主要用于：
- 作为参考值传递给Agent
- 实际不再作为硬性限制

### 2. 资源管理
虽然移除了总超时限制，但仍有保护机制：
- Context 30分钟超时
- 客户端断开自动清理
- Agent端HTTP Transport的各种超时

### 3. 向后兼容
- 新版本服务器兼容旧版本Agent（但无法获得长时间处理支持）
- 新版本Agent兼容旧版本服务器
- 建议同时更新服务器和Agent以获得最佳效果

---

## 总结

**问题根源：** 使用了不适合流式响应的总超时机制

**解决方案：** 移除中间层的总超时限制，使用边缘控制和Context超时

**效果：** 支持任意长时间的流式处理（最长30分钟），解决了长时间处理中断的问题

---

**修复日期：** 2026-05-11  
**影响范围：** 服务器端 + Agent客户端  
**部署要求：** 需要同时更新服务器和Agent  
**测试状态：** 待验证
