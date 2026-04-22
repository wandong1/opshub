## Agent 连接架构概览

OpsHub 的 Agent 系统采用 **mTLS（双向 TLS 认证）** 架构，通过 gRPC 双向流实现客户端与服务端的安全通信。

------

## 一、证书体系架构

### 1. 证书层级结构



```
CA 根证书 (ca.pem + ca-key.pem)
├── 服务端证书 (动态生成，内存中使用)
│   └── CN: OpsHub gRPC Server
│   └── 用途: ServerAuth
│   └── SAN: localhost, 127.0.0.1, 0.0.0.0
│
└── Agent 客户端证书 (按 AgentID 签发)
    └── CN: {AgentID} (UUID格式)
    └── 用途: ClientAuth
    └── 存储: data/agent-certs/{AgentID}/cert.pem + key.pem
```

### 2. 证书文件分布



```bash
data/agent-certs/
├── ca.pem              # CA 根证书（公钥）
├── ca-key.pem          # CA 私钥（用于签发证书）
├── ca.srl              # 证书序列号记录
└── {AgentID}/          # 每个 Agent 独立目录
    ├── cert.pem        # Agent 客户端证书
    └── key.pem         # Agent 私钥
```

------

## 二、服务端 TLS 配置逻辑

### 1. 服务端启动流程（`internal/server/agent/grpc_server.go`）



```go
// Start() 方法
1. 初始化 CA（如果不存在则生成）
   └── tlsMgr.InitCA()

2. 加载服务端 TLS 配置
   └── tlsMgr.LoadServerTLSConfig()
      ├── 动态生成服务端证书（每次启动重新生成）
      ├── 使用 CA 签发服务端证书
      └── 配置 mTLS 参数

3. 创建 gRPC 服务器
   └── grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))

4. 监听端口（默认 9877）
```

### 2. 服务端 TLS 配置详解（`internal/server/agent/tls.go:184-237`）



```go
func (m *TLSManager) LoadServerTLSConfig() (*tls.Config, error) {
    // 1. 动态生成服务端私钥（ECDSA P-256）
    serverKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    
    // 2. 创建服务端证书模板
    serverTemplate := &x509.Certificate{
        Subject: pkix.Name{
            Organization: []string{"OpsHub"},
            CommonName:   "OpsHub gRPC Server",
        },
        NotBefore: time.Now(),
        NotAfter:  time.Now().Add(10 * 365 * 24 * time.Hour),
        KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
        ExtKeyUsage: []x509.ExtKeyUsage{
            x509.ExtKeyUsageServerAuth,  // 服务端认证
        },
        IPAddresses: []net.IP{
            net.ParseIP("0.0.0.0"),
            net.ParseIP("127.0.0.1"),
        },
        DNSNames: []string{"localhost"},
    }
    
    // 3. 使用 CA 签发服务端证书
    serverCertDER, _ := x509.CreateCertificate(
        rand.Reader, 
        serverTemplate,  // 证书模板
        m.caCert,        // 签发者（CA）
        &serverKey.PublicKey, 
        m.caKey,         // CA 私钥签名
    )
    
    // 4. 配置 mTLS
    return &tls.Config{
        Certificates: []tls.Certificate{cert},
        ClientAuth:   tls.RequireAndVerifyClientCert,  // 强制客户端证书验证
        ClientCAs:    caPool,  // 用于验证客户端证书的 CA 池
    }
}
```

**关键点**：

- `ClientAuth: tls.RequireAndVerifyClientCert` — 强制要求客户端提供证书并验证
- `ClientCAs: caPool` — 只信任由本 CA 签发的客户端证书
- 服务端证书每次启动动态生成，不持久化到磁盘

------

## 三、客户端 TLS 配置逻辑

### 1. Agent 客户端连接流程（`agent/internal/client/grpc_client.go`）



```go
// connectAndServe() 方法
1. 加载 TLS 配置
   └── c.loadTLS()

2. 创建 gRPC 连接
   └── grpc.NewClient(serverAddr, 
        grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))

3. 建立双向流
   └── client.Connect(ctx)

4. 发送注册消息
   └── AgentMessage_Register

5. 启动心跳循环
   └── heartbeatLoop()
```

### 2. 客户端 TLS 配置详解（`agent/internal/client/grpc_client.go:182-213`）



```go
func (c *GRPCClient) loadTLS() (*tls.Config, error) {
    certDir := c.cfg.CertDir  // 配置的证书目录
    
    // 1. 加载 CA 证书（用于验证服务端）
    caCert, _ := os.ReadFile(filepath.Join(certDir, "ca.pem"))
    caPool := x509.NewCertPool()
    caPool.AppendCertsFromPEM(caCert)
    
    // 2. 加载客户端证书和私钥
    cert, _ := tls.LoadX509KeyPair(
        filepath.Join(certDir, "cert.pem"),  // 客户端证书
        filepath.Join(certDir, "key.pem"),   // 客户端私钥
    )
    
    // 3. 提取服务器主机名（用于 SNI）
    serverName := c.cfg.ServerAddr
    if host, _, err := net.SplitHostPort(c.cfg.ServerAddr); err == nil {
        serverName = host
    }
    
    // 4. 配置 TLS
    return &tls.Config{
        Certificates:       []tls.Certificate{cert},  // 客户端证书
        RootCAs:            caPool,                   // 信任的 CA
        ServerName:         serverName,               // SNI 主机名
        InsecureSkipVerify: true,  // ⚠️ 跳过服务器证书验证
    }
}
```

------

## 四、SSL 证书校验逻辑详解

### 1. 服务端校验客户端证书

**校验流程**：



```
1. 客户端发起 TLS 握手
   └── 提供客户端证书 (cert.pem)

2. 服务端验证客户端证书
   ├── 检查证书是否由信任的 CA 签发
   │   └── 使用 ClientCAs (ca.pem) 验证签名
   ├── 检查证书有效期
   │   └── NotBefore <= 当前时间 <= NotAfter
   ├── 检查证书用途
   │   └── ExtKeyUsage 包含 ClientAuth
   └── 检查证书撤销状态（未实现 CRL/OCSP）

3. 验证通过 → 建立连接
   验证失败 → 拒绝连接
```

**代码位置**：`internal/server/agent/tls.go:232-236`



```go
&tls.Config{
    ClientAuth: tls.RequireAndVerifyClientCert,  // 强制验证
    ClientCAs:  caPool,  // 只信任本 CA 签发的证书
}
```

### 2. 客户端校验服务端证书

**当前实现的问题**：



```go
InsecureSkipVerify: true,  // ⚠️ 跳过服务器证书验证
```

**为什么跳过验证？** 根据代码注释：

> "跳过服务器证书验证（因为服务端证书只包含 localhost）"

**实际原因分析**：

1. 服务端证书的 SAN 只包含 `localhost` 和 `127.0.0.1`
2. 当 Agent 通过外网 IP 或域名连接时，主机名不匹配
3. 为了兼容性，选择跳过服务端证书验证

**安全影响**：

- ✅ 客户端仍然验证服务端证书是否由信任的 CA 签发（通过 `RootCAs`）
- ❌ 不验证服务端证书的主机名（容易受到中间人攻击）
- ⚠️ 在内网环境相对安全，但不符合最佳实践

------

## 五、证书签发流程

### 1. Agent 证书签发（`internal/server/agent/tls.go:131-181`）



```go
func (m *TLSManager) SignAgentCert(agentID string, hostIP string) {
    // 1. 生成 Agent 私钥
    agentKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    
    // 2. 创建证书模板
    template := &x509.Certificate{
        SerialNumber: serialNumber,  // 随机序列号
        Subject: pkix.Name{
            Organization: []string{"OpsHub Agent"},
            CommonName:   agentID,  // CN = AgentID (UUID)
        },
        NotBefore: time.Now(),
        NotAfter:  time.Now().Add(365 * 24 * time.Hour),  // 1年有效期
        KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
        ExtKeyUsage: []x509.ExtKeyUsage{
            x509.ExtKeyUsageClientAuth,  // 客户端认证
        },
        IPAddresses: []net.IP{net.ParseIP(hostIP)},  // 可选的 IP SAN
    }
    
    // 3. 使用 CA 签发证书
    certDER, _ := x509.CreateCertificate(
        rand.Reader,
        template,      // 证书模板
        m.caCert,      // 签发者（CA）
        &agentKey.PublicKey,
        m.caKey,       // CA 私钥签名
    )
    
    // 4. 保存到文件
    agentCertDir := filepath.Join(m.certDir, agentID)
    os.MkdirAll(agentCertDir, 0700)
    os.WriteFile(filepath.Join(agentCertDir, "cert.pem"), certPEM, 0644)
    os.WriteFile(filepath.Join(agentCertDir, "key.pem"), keyPEM, 0600)
}
```

### 2. 证书签发时机

**场景 1：SSH 部署 Agent**



```
1. 用户通过平台部署 Agent
   └── deploy_handler.go: DeployAgent()

2. 生成 AgentID (UUID)
   └── uuid.New().String()

3. 签发客户端证书
   └── tlsMgr.SignAgentCert(agentID, hostIP)

4. 通过 SSH 上传证书到目标主机
   └── 上传 ca.pem, cert.pem, key.pem 到 Agent 配置目录

5. 启动 Agent 服务
```

**场景 2：手动安装 Agent**



```
1. 用户手动下载 Agent 安装包

2. 手动生成 AgentID 或使用预生成的 UUID

3. 从服务端获取 CA 证书
   └── scp user@server:/path/to/ca.pem ./certs/

4. 请求服务端签发证书（需要额外实现接口）
   或 使用预签发的通用证书（不推荐）

5. 配置 Agent 并启动
```

------

## 六、连接建立完整流程



```
┌─────────────┐                                    ┌─────────────┐
│   Agent     │                                    │   Server    │
│  (Client)   │                                    │  (gRPC)     │
└──────┬──────┘                                    └──────┬──────┘
       │                                                  │
       │  1. TLS Handshake                               │
       ├─────────────────────────────────────────────────>│
       │     ClientHello + Client Certificate            │
       │                                                  │
       │  2. Server verifies client cert                 │
       │     - Check CA signature ✓                      │
       │     - Check validity period ✓                   │
       │     - Check ExtKeyUsage=ClientAuth ✓            │
       │                                                  │
       │  3. ServerHello + Server Certificate            │
       │<─────────────────────────────────────────────────┤
       │                                                  │
       │  4. Client verifies server cert                 │
       │     - Check CA signature ✓                      │
       │     - Skip hostname verification (InsecureSkipVerify) │
       │                                                  │
       │  5. TLS connection established                  │
       │═════════════════════════════════════════════════>│
       │                                                  │
       │  6. gRPC bidirectional stream                   │
       ├──────────────────────────────────────────────────┤
       │     stream := client.Connect(ctx)               │
       │                                                  │
       │  7. Send RegisterRequest                        │
       ├─────────────────────────────────────────────────>│
       │     AgentID, Hostname, OS, Arch, IPs            │
       │                                                  │
       │  8. Server validates registration               │
       │     - Query agent_info by AgentID              │
       │     - Or auto-register new host                 │
       │                                                  │
       │  9. Send RegisterResponse                       │
       │<─────────────────────────────────────────────────┤
       │     Success=true, HeartbeatInterval=30s         │
       │                                                  │
       │  10. Start heartbeat loop                       │
       ├─────────────────────────────────────────────────>│
       │     HeartbeatRequest (every 30s)                │
       │<─────────────────────────────────────────────────┤
       │     HeartbeatResponse                           │
       │                                                  │
```

------

## 七、安全性分析

### ✅ 优点

1. **双向认证**：服务端和客户端都需要提供证书
2. **CA 签名验证**：所有证书必须由同一 CA 签发
3. **证书隔离**：每个 Agent 拥有独立的证书和私钥
4. **私钥保护**：私钥文件权限 0600，仅所有者可读
5. **证书有效期**：Agent 证书 1 年有效期，需定期更新

### ⚠️ 潜在风险

1. **跳过服务端主机名验证**
   - `InsecureSkipVerify: true` 容易受到中间人攻击
   - **建议**：服务端证书应包含所有可能的访问地址（IP/域名）
2. **CA 私钥安全**
   - `ca-key.pem` 权限应为 0600
   - 应考虑使用 HSM 或密钥管理服务
3. **证书撤销机制缺失**
   - 无 CRL 或 OCSP 支持
   - 被盗证书无法及时撤销
4. **证书轮换**
   - Agent 证书 1 年有效期，需要自动续期机制

------

## 八、改进建议

### 1. 修复服务端证书主机名问题



```go
// 方案 1：配置文件指定所有访问地址
serverTemplate.DNSNames = []string{
    "localhost",
    "opshub.example.com",
    "*.opshub.example.com",
}
serverTemplate.IPAddresses = []net.IP{
    net.ParseIP("127.0.0.1"),
    net.ParseIP("公网IP"),
}

// 方案 2：客户端配置跳过主机名验证但保留 CA 验证
tlsConfig.InsecureSkipVerify = false
tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
    // 自定义验证逻辑
    return nil
}
```

### 2. 实现证书撤销机制



```go
// 维护撤销列表
type CertificateRevocationList struct {
    RevokedSerialNumbers map[string]time.Time
}

// 在验证时检查
tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
    cert := verifiedChains[0][0]
    if isRevoked(cert.SerialNumber) {
        return errors.New("certificate revoked")
    }
    return nil
}
```

### 3. 自动证书续期



```go
// Agent 定期检查证书有效期
func (c *GRPCClient) checkCertExpiry() {
    cert, _ := tls.LoadX509KeyPair(certPath, keyPath)
    x509Cert, _ := x509.ParseCertificate(cert.Certificate[0])
    
    if time.Until(x509Cert.NotAfter) < 30*24*time.Hour {
        // 请求服务端续期
        c.requestCertRenewal()
    }
}
```

------

## 总结

OpsHub 的 Agent 连接采用了 **mTLS 双向认证** 架构，通过 CA 签发的证书实现了较强的安全性。服务端强制验证客户端证书，确保只有授权的 Agent 可以连接。但客户端跳过了服务端主机名验证（`InsecureSkipVerify: true`），这在内网环境相对安全，但在公网部署时存在中间人攻击风险。建议完善服务端证书的 SAN 配置，并实现证书撤销和自动续期机制。