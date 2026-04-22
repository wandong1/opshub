# Agent TLS 证书主机名验证问题修复

## 问题描述

Agent 客户端连接服务端时报错：

```
[ERROR] 建立流失败: rpc error: code = Unavailable desc = connection error: 
desc = "transport: authentication handshake failed: tls: failed to verify certificate: 
x509: certificate is valid for 0.0.0.0, 127.0.0.1, not 192.168.0.13"
```

**错误原因**：
- 服务端证书只包含 `0.0.0.0` 和 `127.0.0.1`
- 客户端通过 `192.168.0.13:9090` 连接
- 证书主机名不匹配，TLS 握手失败

---

## 根本原因

### 服务端证书生成逻辑

**位置**：`internal/server/agent/tls.go:183-210`

**原始代码**：
```go
serverTemplate := &x509.Certificate{
    // ...
    IPAddresses: []net.IP{
        net.ParseIP("0.0.0.0"),
        net.ParseIP("127.0.0.1"),  // ← 只有这两个 IP
    },
    DNSNames: []string{"localhost"},
}
```

**问题**：
- 服务端证书在启动时动态生成
- 只包含 `0.0.0.0` 和 `127.0.0.1`
- 不包含服务器的实际 IP 地址（如 `192.168.0.13`）

### 客户端验证逻辑

**位置**：`agent/internal/client/grpc_client.go:182-213`

**原始代码**：
```go
return &tls.Config{
    Certificates:       []tls.Certificate{cert},
    RootCAs:            caPool,
    ServerName:         serverName,  // ← 从 server_addr 提取
    InsecureSkipVerify: true,        // ← 虽然跳过验证，但仍然失败
}
```

**问题**：
- `InsecureSkipVerify: true` 只跳过主机名验证
- 但 Go 的 TLS 库在证书 SAN 不匹配时仍然会报错
- 需要服务端证书包含正确的 IP 地址

---

## 解决方案

### 修复服务端证书生成逻辑

**文件**：`internal/server/agent/tls.go`

**修改内容**：

1. **添加获取本机 IP 的函数**：

```go
// getLocalIPs 获取本机所有非loopback IP地址
func getLocalIPs() []string {
    var ips []string
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        appLogger.Warn("获取本机IP失败", zap.Error(err))
        return ips
    }
    for _, addr := range addrs {
        if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
            if ipNet.IP.To4() != nil {
                ips = append(ips, ipNet.IP.String())
            }
        }
    }
    return ips
}
```

2. **修改证书生成逻辑**：

```go
func (m *TLSManager) LoadServerTLSConfig() (*tls.Config, error) {
    // ... 省略前面的代码 ...

    // 获取本机所有 IP 地址
    localIPs := getLocalIPs()
    ipAddresses := []net.IP{
        net.ParseIP("0.0.0.0"),
        net.ParseIP("127.0.0.1"),
    }
    for _, ipStr := range localIPs {
        if ip := net.ParseIP(ipStr); ip != nil {
            ipAddresses = append(ipAddresses, ip)
        }
    }

    serverTemplate := &x509.Certificate{
        // ...
        IPAddresses: ipAddresses,  // ← 包含所有本机 IP
        DNSNames:    []string{"localhost"},
    }

    appLogger.Info("生成服务端证书", zap.Strings("ipAddresses", localIPs))
    
    // ... 省略后面的代码 ...
}
```

---

## 修复效果

### 修复前

**服务端证书 SAN**：
```
IP Addresses:
  - 0.0.0.0
  - 127.0.0.1
DNS Names:
  - localhost
```

**客户端连接**：
- ❌ `192.168.0.13:9090` → 证书验证失败
- ✅ `127.0.0.1:9090` → 成功
- ✅ `localhost:9090` → 成功

---

### 修复后

**服务端证书 SAN**（示例）：
```
IP Addresses:
  - 0.0.0.0
  - 127.0.0.1
  - 192.168.0.13    ← 自动添加
  - 192.168.1.100   ← 自动添加（如果有多个网卡）
  - 10.0.0.5        ← 自动添加（如果有多个网卡）
DNS Names:
  - localhost
```

**客户端连接**：
- ✅ `192.168.0.13:9090` → 成功
- ✅ `192.168.1.100:9090` → 成功
- ✅ `10.0.0.5:9090` → 成功
- ✅ `127.0.0.1:9090` → 成功
- ✅ `localhost:9090` → 成功

---

## 验证步骤

### 1. 重新编译服务端

```bash
make build
```

### 2. 重启服务端

```bash
# 停止旧服务
sudo systemctl stop opshub

# 启动新服务
sudo systemctl start opshub

# 查看日志（确认证书生成）
sudo journalctl -u opshub -f | grep "生成服务端证书"
```

**预期日志**：
```
[INFO] 生成服务端证书 ipAddresses=[192.168.0.13, 192.168.1.100]
```

### 3. 测试 Agent 连接

```bash
# 在 Agent 主机上查看日志
tail -f /opt/srehub-agent/srehub-agent.log
```

**预期日志**：
```
[INFO] 正在连接到服务器: 192.168.0.13:9090
[INFO] 发送注册请求 - AgentID: xxx, Hostname: xxx
[INFO] 注册成功: 注册成功
[INFO] 启动心跳循环，间隔: 60秒
```

### 4. 在平台查看 Agent 状态

- 主机管理页面
- Agent 状态显示为 **在线**（绿色）

---

## 技术细节

### 为什么 InsecureSkipVerify 不起作用？

**Go TLS 库的验证流程**：

1. **验证证书链**：检查证书是否由信任的 CA 签发 ✅
2. **验证证书有效期**：检查 NotBefore 和 NotAfter ✅
3. **验证主机名**：检查 ServerName 是否在证书的 SAN 中 ❌

`InsecureSkipVerify: true` 只跳过第 3 步，但 Go 的 TLS 实现在 SAN 完全不匹配时仍然会报错。

**正确的做法**：
- 让服务端证书包含所有可能的访问地址
- 或者使用自定义的 `VerifyPeerCertificate` 回调

### 为什么不使用配置文件指定 IP？

**动态获取的优势**：
- ✅ 自动适应多网卡环境
- ✅ 无需手动配置
- ✅ 支持 DHCP 动态 IP
- ✅ 支持容器/虚拟机迁移

**配置文件的劣势**：
- ❌ 需要手动维护
- ❌ IP 变更后需要重启服务
- ❌ 多网卡环境容易遗漏

### 证书生命周期

**服务端证书**：
- 每次服务启动时动态生成
- 有效期 10 年
- 不持久化到磁盘
- 重启后自动包含最新的 IP 地址

**客户端证书**：
- 部署时签发
- 有效期 1 年
- 持久化到 `data/agent-certs/{agentID}/`
- 需要定期续期

---

## 其他解决方案（不推荐）

### 方案二：客户端完全跳过验证

**修改**：`agent/internal/client/grpc_client.go`

```go
return &tls.Config{
    Certificates:       []tls.Certificate{cert},
    RootCAs:            caPool,
    InsecureSkipVerify: true,
    VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
        // 完全跳过验证
        return nil
    },
}
```

**缺点**：
- ❌ 安全性降低
- ❌ 容易受到中间人攻击
- ❌ 不符合最佳实践

---

### 方案三：使用域名而非 IP

**修改**：
1. 配置 DNS 解析：`opshub.example.com` → `192.168.0.13`
2. 服务端证书添加域名：`DNSNames: []string{"localhost", "opshub.example.com"}`
3. Agent 配置使用域名：`server_addr: "opshub.example.com:9090"`

**优点**：
- ✅ 更符合 TLS 最佳实践
- ✅ IP 变更时只需更新 DNS

**缺点**：
- ❌ 需要 DNS 服务器
- ❌ 内网环境配置复杂

---

## 相关文件

- `internal/server/agent/tls.go` - TLS 证书管理
- `internal/server/agent/grpc_server.go` - gRPC 服务器
- `agent/internal/client/grpc_client.go` - Agent gRPC 客户端
- `docs/agent-tls-architecture.md` - TLS 架构详解（之前的文档）

---

## 修复日期

2026-04-07

## 修复人员

Claude (AI Assistant)
