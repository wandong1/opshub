# AI模型代理功能实施方案

## 一、需求概述

在现有的资产管理、Web站点管理基础上，新增**AI模型代理**功能，实现以下目标：

1. **通过Agent主机代理访问AI模型服务**（如Ollama、OpenAI等）
2. **提供永久有效的Token认证代理URL**
3. **支持SSE流式响应**，解决"Did not receive done or success response in stream"问题
4. **完全不影响现有功能**，采用独立的代码路径和数据表

## 二、技术架构对比

### 2.1 现有Web站点代理架构（参考）

```
客户端 → 服务器(Token认证) → AgentHub → Agent(gRPC) → 目标网站
         ↓ 缓冲完整响应
客户端 ← 服务器 ← AgentHub ← Agent ← 完整响应(io.ReadAll)
```

**特点：**
- 使用 `io.ReadAll` 缓冲完整响应
- 适合普通HTTP请求
- **不支持SSE流式传输**

### 2.2 新AI模型代理架构（流式）

```
客户端 → 服务器(Token认证) → AgentHub → Agent(gRPC) → Ollama API
         ↓ SSE实时分块
客户端 ← 服务器(Flush) ← AgentHub ← Agent ← 流式分块(4KB buffer)
```

**特点：**
- Agent端使用 `bufio.Reader` 分块读取
- gRPC使用新的流式消息类型
- 服务器端使用 `gin.Writer.Flush()` 实时推送
- **完全支持SSE协议透传**

## 三、数据库设计

### 3.1 主表：ai_model_proxies

```sql
CREATE TABLE `ai_model_proxies` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除时间',
  
  -- 基础信息
  `name` varchar(100) NOT NULL COMMENT 'AI模型代理名称',
  `description` varchar(500) DEFAULT NULL COMMENT '描述',
  `model_type` varchar(50) NOT NULL DEFAULT 'ollama' COMMENT '模型类型: ollama/openai/custom',
  `status` tinyint NOT NULL DEFAULT 1 COMMENT '状态: 1=启用 0=禁用',
  
  -- 连接配置
  `target_url` varchar(500) NOT NULL COMMENT '目标URL(如http://localhost:11434)',
  `api_key` varchar(500) DEFAULT NULL COMMENT 'API密钥(加密存储)',
  `timeout` int NOT NULL DEFAULT 300 COMMENT '超时时间(秒),默认5分钟',
  
  -- 代理访问Token（永久有效）
  `proxy_token` varchar(64) NOT NULL COMMENT '代理访问Token(UUID)',
  
  -- 分组关联
  `group_id` bigint unsigned NOT NULL COMMENT '资产分组ID',
  
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_proxy_token` (`proxy_token`),
  KEY `idx_group_id` (`group_id`),
  KEY `idx_status` (`status`),
  KEY `idx_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_ai_model_proxy_group` FOREIGN KEY (`group_id`) REFERENCES `asset_group` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI模型代理配置表';
```

### 3.2 关联表：ai_model_proxy_agents

```sql
CREATE TABLE `ai_model_proxy_agents` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `proxy_id` bigint unsigned NOT NULL COMMENT 'AI模型代理ID',
  `host_id` bigint unsigned NOT NULL COMMENT 'Agent主机ID',
  
  PRIMARY KEY (`id`),
  KEY `idx_proxy_id` (`proxy_id`),
  KEY `idx_host_id` (`host_id`),
  CONSTRAINT `fk_ai_model_proxy_agents_proxy` FOREIGN KEY (`proxy_id`) REFERENCES `ai_model_proxies` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_ai_model_proxy_agents_host` FOREIGN KEY (`host_id`) REFERENCES `hosts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI模型代理与Agent主机关联表';
```

### 3.3 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `model_type` | varchar(50) | 模型类型，支持：ollama、openai、custom |
| `target_url` | varchar(500) | 目标地址，如 `http://localhost:11434` |
| `api_key` | varchar(500) | API密钥，使用AES加密存储（可选） |
| `timeout` | int | 超时时间（秒），默认300秒（5分钟） |
| `proxy_token` | varchar(64) | UUID格式，永久有效，用于生成代理URL |
| `group_id` | bigint | 资产分组ID，用于权限控制 |

## 四、gRPC协议扩展

### 4.1 新增消息类型（api/proto/agent.proto）

```protobuf
// 流式代理请求
message StreamProxyRequest {
    string request_id = 1;      // 请求ID(UUID)
    string method = 2;          // HTTP方法
    string url = 3;             // 目标URL
    map<string, string> headers = 4;  // 请求头
    bytes body = 5;             // 请求体
    int32 timeout = 6;          // 超时时间(秒)
}

// 流式代理响应块
message StreamProxyChunk {
    string request_id = 1;      // 请求ID
    bytes data = 2;             // 数据块
    bool is_final = 3;          // 是否最后一块
    string error = 4;           // 错误信息
    int32 status_code = 5;      // HTTP状态码(仅首块)
    map<string, string> headers = 6;  // 响应头(仅首块)
}

// 添加到 AgentMessage.payload
message AgentMessage {
    oneof payload {
        // ... 现有消息类型
        StreamProxyRequest stream_proxy_request = 15;
        StreamProxyChunk stream_proxy_chunk = 16;
    }
}
```

### 4.2 协议特点

- **分块传输**：每次传输最多4KB数据
- **首块包含元信息**：status_code和headers仅在第一块中传输
- **最后一块标记**：is_final=true表示传输完成
- **错误处理**：error字段用于传递错误信息

## 五、核心代码实现

### 5.1 Agent端流式处理器（agent/internal/client/grpc_client.go）

```go
func (c *GrpcClient) handleStreamProxyRequest(req *pb.StreamProxyRequest) {
    // 1. 构建HTTP请求
    httpReq, err := http.NewRequest(req.Method, req.Url, bytes.NewReader(req.Body))
    if err != nil {
        c.sendStreamError(req.RequestId, err)
        return
    }
    
    // 2. 设置请求头
    for k, v := range req.Headers {
        httpReq.Header.Set(k, v)
    }
    
    // 3. 发送请求
    client := &http.Client{Timeout: time.Duration(req.Timeout) * time.Second}
    resp, err := client.Do(httpReq)
    if err != nil {
        c.sendStreamError(req.RequestId, err)
        return
    }
    defer resp.Body.Close()
    
    // 4. 发送首块（包含状态码和响应头）
    headers := make(map[string]string)
    for k, v := range resp.Header {
        headers[k] = strings.Join(v, ",")
    }
    
    c.stream.Send(&pb.AgentMessage{
        Payload: &pb.AgentMessage_StreamProxyChunk{
            StreamProxyChunk: &pb.StreamProxyChunk{
                RequestId:  req.RequestId,
                StatusCode: int32(resp.StatusCode),
                Headers:    headers,
                Data:       []byte{},
                IsFinal:    false,
            },
        },
    })
    
    // 5. 流式读取并转发（关键：不缓冲）
    reader := bufio.NewReader(resp.Body)
    buffer := make([]byte, 4096)
    
    for {
        n, err := reader.Read(buffer)
        if n > 0 {
            // 立即发送数据块
            c.stream.Send(&pb.AgentMessage{
                Payload: &pb.AgentMessage_StreamProxyChunk{
                    StreamProxyChunk: &pb.StreamProxyChunk{
                        RequestId: req.RequestId,
                        Data:      buffer[:n],
                        IsFinal:   false,
                    },
                },
            })
        }
        
        if err == io.EOF {
            break
        }
        if err != nil {
            c.sendStreamError(req.RequestId, err)
            return
        }
    }
    
    // 6. 发送最后一块
    c.stream.Send(&pb.AgentMessage{
        Payload: &pb.AgentMessage_StreamProxyChunk{
            StreamProxyChunk: &pb.StreamProxyChunk{
                RequestId: req.RequestId,
                Data:      []byte{},
                IsFinal:   true,
            },
        },
    })
}
```

### 5.2 服务器端SSE处理器（internal/server/asset/ai_model_proxy_handler.go）

```go
func (h *AIModelProxyHandler) ProxyStream(c *gin.Context) {
    // 1. 提取Token
    token := c.Param("token")
    
    // 2. 验证Token并获取配置
    proxyConfig, err := h.validateToken(token)
    if err != nil {
        c.JSON(404, gin.H{"error": "Invalid token"})
        return
    }
    
    // 3. 选择在线Agent
    agent, err := h.selectOnlineAgent(proxyConfig.AgentHostIDs)
    if err != nil {
        c.JSON(503, gin.H{"error": "No agent available"})
        return
    }
    
    // 4. 构建目标URL
    targetURL := proxyConfig.TargetURL + c.Param("path")
    
    // 5. 读取请求体
    body, _ := io.ReadAll(c.Request.Body)
    
    // 6. 构建gRPC请求
    requestID := uuid.New().String()
    streamReq := &pb.StreamProxyRequest{
        RequestId: requestID,
        Method:    c.Request.Method,
        Url:       targetURL,
        Headers:   extractHeaders(c.Request.Header),
        Body:      body,
        Timeout:   int32(proxyConfig.Timeout),
    }
    
    // 7. 发送到Agent
    agent.Send(&pb.AgentMessage{
        Payload: &pb.AgentMessage_StreamProxyRequest{
            StreamProxyRequest: streamReq,
        },
    })
    
    // 8. 设置SSE响应头
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("X-Accel-Buffering", "no")  // 禁用nginx缓冲
    
    // 9. 获取chunk通道
    chunkChan, err := h.agentHub.StreamResponse(agent, requestID, 
        time.Duration(proxyConfig.Timeout)*time.Second)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // 10. 流式转发到客户端
    flusher, ok := c.Writer.(http.Flusher)
    if !ok {
        c.JSON(500, gin.H{"error": "Streaming not supported"})
        return
    }
    
    firstChunk := true
    for chunk := range chunkChan {
        // 首块：设置状态码和响应头
        if firstChunk {
            c.Status(int(chunk.StatusCode))
            for k, v := range chunk.Headers {
                c.Header(k, v)
            }
            firstChunk = false
        }
        
        // 写入数据
        if len(chunk.Data) > 0 {
            c.Writer.Write(chunk.Data)
            flusher.Flush()  // 立即刷新
        }
        
        // 检查是否结束
        if chunk.IsFinal {
            break
        }
        
        // 检查错误
        if chunk.Error != "" {
            log.Printf("Stream error: %s", chunk.Error)
            break
        }
    }
}
```

### 5.3 AgentHub扩展（internal/server/agent/hub.go）

```go
// StreamResponse 返回流式响应的channel
func (h *AgentHub) StreamResponse(
    as AgentStreamInterface,
    requestID string,
    timeout time.Duration,
) (<-chan *pb.StreamProxyChunk, error) {
    chunkChan := make(chan *pb.StreamProxyChunk, 10)
    
    // 注册pending请求
    as.RegisterPending(requestID, chunkChan)
    
    // 超时处理
    go func() {
        timer := time.NewTimer(timeout)
        defer timer.Stop()
        
        <-timer.C
        // 超时后发送错误chunk并关闭channel
        chunkChan <- &pb.StreamProxyChunk{
            RequestId: requestID,
            Error:     "Request timeout",
            IsFinal:   true,
        }
        close(chunkChan)
        as.RemovePending(requestID)
    }()
    
    return chunkChan, nil
}
```

## 六、API接口设计

### 6.1 管理接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/ai-model-proxies` | 列表查询 |
| POST | `/api/v1/ai-model-proxies` | 创建代理 |
| GET | `/api/v1/ai-model-proxies/:id` | 获取详情 |
| PUT | `/api/v1/ai-model-proxies/:id` | 更新配置 |
| DELETE | `/api/v1/ai-model-proxies/:id` | 删除代理 |
| POST | `/api/v1/ai-model-proxies/:id/regenerate-token` | 重新生成Token |
| GET | `/api/v1/ai-model-proxies/:id/test` | 测试连接 |

### 6.2 代理接口

| 方法 | 路径 | 说明 |
|------|------|------|
| ANY | `/api/v1/ai-model-proxy/:token/*path` | 流式代理转发 |

**示例：**
```
原始Ollama地址: http://localhost:11434/api/chat
代理后地址: https://opshub.com/api/v1/ai-model-proxy/abc123-uuid-token/api/chat
```

### 6.3 请求/响应示例

**创建代理：**
```json
POST /api/v1/ai-model-proxies
{
  "name": "本地Ollama",
  "description": "开发环境Ollama服务",
  "modelType": "ollama",
  "targetUrl": "http://localhost:11434",
  "apiKey": "",
  "timeout": 300,
  "groupId": 1,
  "agentHostIds": [10, 11]
}
```

**响应：**
```json
{
  "id": 1,
  "name": "本地Ollama",
  "proxyToken": "550e8400-e29b-41d4-a716-446655440000",
  "proxyUrl": "https://opshub.com/api/v1/ai-model-proxy/550e8400-e29b-41d4-a716-446655440000",
  "status": 1,
  "agentOnline": true
}
```

## 七、前端菜单配置

### 7.1 菜单结构

```
资产管理 (id=15)
  ├─ 主机管理
  ├─ 中间件管理
  ├─ Web站点管理 (id=90)
  └─ AI模型代理 (id=91, 新增)
       ├─ 查看列表
       ├─ 新增代理 (button)
       ├─ 编辑代理 (button)
       ├─ 删除代理 (button)
       ├─ 测试连接 (button)
       └─ 重新生成Token (button)
```

### 7.2 SQL迁移（菜单部分）

```sql
-- 插入AI模型代理菜单
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `path`, `component`, `icon`, `sort`, `visible`, `status`)
VALUES (91, 'AI模型代理', 'asset_ai_model_proxies', 2, 15, '/asset/ai-model-proxies', 'asset/AIModelProxies', 'Cpu', 10, 1, 1);

-- 插入按钮权限
INSERT INTO `sys_menu` (`id`, `name`, `code`, `type`, `parent_id`, `api_path`, `api_method`, `sort`, `visible`, `status`)
VALUES
  (380, '新增代理', 'ai_model_proxies:create', 3, 91, '/api/v1/ai-model-proxies', 'POST', 1, 1, 1),
  (381, '编辑代理', 'ai_model_proxies:update', 3, 91, '/api/v1/ai-model-proxies/:id', 'PUT', 2, 1, 1),
  (382, '删除代理', 'ai_model_proxies:delete', 3, 91, '/api/v1/ai-model-proxies/:id', 'DELETE', 3, 1, 1),
  (383, '测试连接', 'ai_model_proxies:test', 3, 91, '/api/v1/ai-model-proxies/:id/test', 'GET', 4, 1, 1),
  (384, '重新生成Token', 'ai_model_proxies:regenerate', 3, 91, '/api/v1/ai-model-proxies/:id/regenerate-token', 'POST', 5, 1, 1);

-- 为管理员角色分配权限
INSERT INTO `sys_role_menu` (`role_id`, `menu_id`)
VALUES (1, 91), (1, 380), (1, 381), (1, 382), (1, 383), (1, 384);
```

## 八、实施步骤

### 阶段1：数据库和协议（不影响现有功能）
1. 创建数据库迁移文件
2. 扩展gRPC协议（新增消息类型）
3. 生成protobuf代码

### 阶段2：领域模型和数据访问
4. 创建领域模型（Model、Request、VO、Repository接口）
5. 实现数据访问层（Repository实现）
6. 实现业务逻辑层（UseCase）

### 阶段3：Agent端实现
7. 实现Agent端流式处理器
8. 注册消息处理器

### 阶段4：服务器端实现
9. 扩展AgentHub支持流式响应
10. 实现SSE流式代理处理器
11. 实现HTTP服务层（CRUD接口）
12. 注册路由和依赖注入

### 阶段5：测试验证
13. 单元测试
14. 集成测试（Ollama SSE流式）
15. 性能测试（内存泄漏、长连接）

## 九、关键技术点

### 9.1 SSE协议透传

**问题：** 现有实现使用 `io.ReadAll` 缓冲完整响应，导致SSE流被阻塞

**解决方案：**
- Agent端：使用 `bufio.Reader` 分块读取（4KB buffer）
- gRPC：使用流式消息类型传输
- 服务器端：使用 `http.Flusher` 立即刷新到客户端

### 9.2 Token永久有效

**实现方式：**
- 使用UUID作为proxy_token
- 数据库唯一索引保证不重复
- 不设置过期时间
- 支持手动重新生成

### 9.3 Agent选择策略

**复用现有逻辑：**
```go
// 从绑定的Agent列表中选择第一个在线的
for _, hostID := range agentHostIDs {
    if agent, ok := agentHub.GetByHostID(hostID); ok {
        if agentHub.IsOnline(hostID) {
            return agent, nil
        }
    }
}
return nil, errors.New("no agent available")
```

### 9.4 错误处理

| 场景 | HTTP状态码 | 处理方式 |
|------|-----------|---------|
| Token无效 | 404 | 返回错误JSON |
| 无可用Agent | 503 | 返回错误JSON |
| Agent超时 | 504 | 发送error chunk并关闭流 |
| 目标服务错误 | 502 | 透传原始状态码 |
| 流中断 | - | 发送error chunk，客户端重连 |

## 十、向后兼容性保证

### 10.1 独立代码路径
- 新文件：不修改现有文件
- 新路由：独立的路由前缀
- 新表：不影响现有表结构

### 10.2 gRPC协议扩展
- 新增消息类型（不修改现有类型）
- 向后兼容：老Agent忽略新消息

### 10.3 AgentHub扩展
- 新增方法 `StreamResponse()`
- 不修改现有方法 `WaitResponse()`

## 十一、测试计划

### 11.1 功能测试

**测试用例1：Ollama流式对话**
```bash
curl -X POST https://opshub.com/api/v1/ai-model-proxy/{token}/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama2",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'
```

**预期结果：**
- 实时接收SSE事件
- 最后收到 `{"done": true}` 消息
- 无 "Did not receive done" 错误

**测试用例2：Token永久有效**
- 创建代理并获取Token
- 30天后仍可正常访问

**测试用例3：Agent故障转移**
- 绑定2个Agent
- 关闭主Agent
- 请求自动路由到备用Agent

### 11.2 性能测试

**测试用例4：内存泄漏检测**
```bash
# 持续发送100个并发流式请求，每个持续5分钟
ab -n 100 -c 10 -t 300 https://opshub.com/api/v1/ai-model-proxy/{token}/api/chat
```

**监控指标：**
- 服务器内存使用量
- Goroutine数量
- 连接数

**测试用例5：长连接稳定性**
- 单个SSE连接保持10分钟
- 验证无超时断开
- 验证数据完整性

### 11.3 兼容性测试

**测试用例6：现有功能回归**
- Web站点代理功能正常
- 主机管理功能正常
- 中间件管理功能正常

## 十二、风险评估

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|---------|
| 内存泄漏 | 高 | 中 | 实现超时机制、context取消、监控goroutine |
| Agent崩溃 | 中 | 低 | 检测流关闭、返回错误chunk、日志记录 |
| 破坏现有功能 | 高 | 低 | 独立代码路径、全面回归测试、灰度发布 |
| nginx缓冲SSE | 中 | 中 | 设置 `X-Accel-Buffering: no` 响应头 |
| 客户端不支持SSE | 低 | 低 | 文档说明、提供测试工具 |

## 十三、上线计划

### 阶段1：开发环境验证（1-2天）
- 完成代码开发
- 本地Ollama测试
- 单元测试通过

### 阶段2：测试环境部署（1天）
- 部署到测试环境
- 集成测试
- 性能测试

### 阶段3：生产环境灰度（1天）
- 仅对管理员开放
- 监控日志和性能指标
- 收集反馈

### 阶段4：全量发布（1天）
- 开放给所有用户
- 持续监控
- 准备回滚方案

## 十四、文档输出

1. **用户手册**：如何创建AI模型代理、获取代理URL
2. **API文档**：接口说明、请求示例、响应格式
3. **运维手册**：部署步骤、监控指标、故障排查
4. **开发文档**：架构设计、代码结构、扩展指南

---

## 附录：与现有Web站点代理的对比

| 特性 | Web站点代理 | AI模型代理 |
|------|------------|-----------|
| 响应方式 | 缓冲完整响应 | 流式实时响应 |
| gRPC消息 | HttpProxyRequest/Response | StreamProxyRequest/Chunk |
| Agent读取 | io.ReadAll | bufio.Reader分块 |
| 服务器写入 | 一次性写入 | Flush实时刷新 |
| 适用场景 | 普通网页 | SSE/流式API |
| Token策略 | 双Token（短期+长期） | 单Token（永久） |
| 内容重写 | HTML/CSS/JS路径重写 | 无需重写（透传） |

---

**方案制定人：** Claude  
**制定时间：** 2026-05-10  
**版本：** v1.0
