# Web站点代理静态资源加载修复

## 问题描述

通过 Agent 代理访问内部站点时，出现以下问题：
1. CSS、JavaScript 等静态资源无法正确加载，页面显示不完整
2. 某些网站的 HTTP 重定向请求无法正确处理

## 根本原因

1. **Agent 端 HTTP 客户端不支持重定向**：`agent/internal/prober/http.go` 中的 `http.Client` 没有配置 `CheckRedirect` 处理函数，导致默认不跟随重定向（Go 的 http.Client 默认最多跟随 10 次重定向，但需要正确配置）

2. **响应头处理不当**：某些响应头（如 `Content-Encoding`、`Transfer-Encoding`）在代理场景下不应该直接转发，否则会导致浏览器解析错误

## 修复方案

### 1. Agent 端支持 HTTP 重定向

**文件**：`agent/internal/prober/http.go`

**修改内容**：
```go
// 重定向计数器
redirectCount := 0

client := &http.Client{
    Transport: transport,
    Timeout:   time.Duration(req.Timeout) * time.Second,
    // 支持自动跟随重定向（最多10次）
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
        redirectCount = len(via)
        if len(via) >= 10 {
            return http.ErrUseLastResponse
        }
        return nil
    },
}
```

**效果**：
- 自动跟随 HTTP 301/302/303/307/308 重定向
- 最多跟随 10 次重定向，防止无限循环
- 记录重定向次数到 `result.RedirectCount`

### 2. 服务端优化响应头处理

**文件**：`internal/server/asset/website_proxy.go`

**修改内容**：
```go
// 设置响应头
for key, value := range result.ResponseHeaders {
    // 跳过某些不应该转发的响应头
    lowerKey := strings.ToLower(key)
    if lowerKey == "content-encoding" || lowerKey == "content-length" || lowerKey == "transfer-encoding" {
        continue
    }
    c.Header(key, value)
}
```

**跳过的响应头说明**：
- `Content-Encoding`：压缩编码由 Agent 端处理，服务端不应再次声明
- `Content-Length`：由 Gin 框架自动计算，避免长度不匹配
- `Transfer-Encoding`：分块传输编码由框架处理

## 测试验证

### 测试场景

1. **静态资源加载**：访问包含 CSS/JS/图片的内部站点，验证所有资源正确加载
2. **HTTP 重定向**：访问会产生 301/302 重定向的站点，验证能正确跟随重定向
3. **多次重定向**：验证重定向链（A → B → C）能正确处理
4. **重定向循环保护**：验证超过 10 次重定向时能正确终止

### 测试步骤

1. 重新部署 Agent（使用新编译的多架构安装包）
2. 重启后端服务
3. 通过平台访问内部站点，检查：
   - 页面样式是否正常显示
   - JavaScript 功能是否正常工作
   - 图片等静态资源是否正常加载
   - 重定向是否正确跟随

## 部署说明

### Agent 端

```bash
cd agent
./build.sh
```

新的多架构安装包已自动复制到 `data/agent-binaries/srehub-agent-1.0.0-multi-arch.tar.gz`

### 服务端

```bash
go build -o bin/opshub .
./bin/opshub server -c config/config.yaml
```

或使用 Docker：
```bash
docker-compose up -d --build
```

## 技术细节

### HTTP 重定向类型

- **301 Moved Permanently**：永久重定向
- **302 Found**：临时重定向
- **303 See Other**：使用 GET 方法重定向
- **307 Temporary Redirect**：临时重定向（保持原方法）
- **308 Permanent Redirect**：永久重定向（保持原方法）

### 重定向处理流程

1. Agent 端发起 HTTP 请求
2. 目标服务器返回 3xx 状态码 + Location 头
3. Agent 端 `CheckRedirect` 函数被调用
4. 自动向 Location 指定的 URL 发起新请求
5. 重复步骤 2-4，直到返回非 3xx 状态码或达到 10 次上限
6. 返回最终响应给服务端

### 响应头过滤原因

- **Content-Encoding**：Agent 端已经解压缩响应体，服务端不应再声明 gzip/deflate
- **Content-Length**：响应体长度可能在传输过程中变化，由框架自动计算更准确
- **Transfer-Encoding**：chunked 编码由框架处理，避免冲突

## 相关文件

- `agent/internal/prober/http.go` - Agent 端 HTTP 拨测器
- `internal/server/asset/website_proxy.go` - 服务端代理处理器
- `api/proto/agent.proto` - Agent 通信协议定义（已包含 redirect_count 字段）

## 版本信息

- 修复日期：2026-03-11
- Agent 版本：1.0.0
- 影响范围：Web站点管理 - 内部站点代理访问
