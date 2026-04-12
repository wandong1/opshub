# Agent NAT 场景解决方案

## 功能说明

支持在配置文件中指定额外的服务端地址（IP 或域名），用于解决 NAT 转换、负载均衡、多网卡等场景下的 TLS 证书验证问题。

---

## 使用场景

### 场景 1：NAT 转换

**网络拓扑**：
```
Agent (内网)          NAT 网关              服务端 (内网)
192.168.1.100    →   公网IP: 1.2.3.4   →   10.0.0.5:9090
```

**问题**：
- 服务端只能检测到内网 IP `10.0.0.5`
- Agent 通过公网 IP `1.2.3.4` 连接
- 证书不包含 `1.2.3.4`，TLS 验证失败

**解决方案**：
```yaml
# config/config.yaml
agent:
  server_addresses:
    - "1.2.3.4"  # NAT 公网 IP
```

---

### 场景 2：负载均衡

**网络拓扑**：
```
Agent → 负载均衡器 (lb.example.com) → 服务端1 (10.0.0.5)
                                    → 服务端2 (10.0.0.6)
```

**解决方案**：
```yaml
agent:
  server_addresses:
    - "lb.example.com"  # 负载均衡器域名
```

---

### 场景 3：多网卡环境

**服务端网卡**：
- eth0: 192.168.1.100 (内网)
- eth1: 10.0.0.5 (管理网)
- eth2: 172.16.0.5 (业务网)

**解决方案**：
```yaml
agent:
  server_addresses:
    - "10.0.0.5"    # 管理网 IP
    - "172.16.0.5"  # 业务网 IP
```

---

### 场景 4：公网 + 内网混合部署

**部署方式**：
- 内网 Agent 通过内网 IP 连接
- 公网 Agent 通过公网域名连接

**解决方案**：
```yaml
agent:
  server_addresses:
    - "opshub.example.com"  # 公网域名
    - "192.168.1.100"       # 内网 IP
```

---

## 配置方法

### 1. 编辑配置文件

**文件位置**：`config/config.yaml`

**配置示例**：
```yaml
agent:
  enabled: true
  cert_dir: "data/agent-certs"
  binary_dir: "data/agent-binaries"
  heartbeat_timeout: 180
  deploy_path: "/opt/srehub-agent"
  
  # 额外的服务端地址（用于证书SAN）
  server_addresses:
    - "1.2.3.4"              # NAT 公网 IP
    - "opshub.example.com"   # 域名
    - "192.168.100.1"        # 其他网段 IP
```

**配置说明**：
- `server_addresses` 是一个数组，支持多个地址
- 支持 IPv4 地址（如 `1.2.3.4`）
- 支持域名（如 `opshub.example.com`）
- 空数组 `[]` 表示不添加额外地址（仅使用自动检测的 IP）

---

### 2. 重启服务端

```bash
# 停止服务
sudo systemctl stop opshub

# 启动服务（会重新生成包含配置地址的证书）
sudo systemctl start opshub

# 查看日志确认
sudo journalctl -u opshub -f | grep "生成服务端证书"
```

**预期日志**：
```
[INFO] 生成服务端证书 
  autoDetectedIPs=[10.0.0.5] 
  configIPs=[1.2.3.4, 192.168.100.1] 
  configDomains=[opshub.example.com] 
  totalIPs=4 
  totalDomains=2
```

---

### 3. 验证证书内容

**查看证书 SAN**：
```bash
# 使用 openssl 查看证书
openssl s_client -connect 1.2.3.4:9090 -showcerts 2>/dev/null | \
  openssl x509 -text -noout | grep -A 10 "Subject Alternative Name"
```

**预期输出**：
```
X509v3 Subject Alternative Name:
    DNS:localhost, DNS:opshub.example.com, 
    IP Address:0.0.0.0, IP Address:127.0.0.1, 
    IP Address:10.0.0.5, IP Address:1.2.3.4, 
    IP Address:192.168.100.1
```

---

## 工作原理

### 证书生成流程

```
1. 服务启动
   ↓
2. 自动检测本机 IP
   → 获取所有非 loopback 的 IPv4 地址
   ↓
3. 读取配置文件
   → 解析 server_addresses 配置项
   ↓
4. 合并所有地址
   → 基础地址: 0.0.0.0, 127.0.0.1, localhost
   → 自动检测: 10.0.0.5
   → 配置文件: 1.2.3.4, opshub.example.com, 192.168.100.1
   ↓
5. 生成服务端证书
   → IPAddresses: [0.0.0.0, 127.0.0.1, 10.0.0.5, 1.2.3.4, 192.168.100.1]
   → DNSNames: [localhost, opshub.example.com]
   ↓
6. 启动 gRPC 服务
```

### 客户端验证流程

```
1. Agent 连接服务端
   → server_addr: 1.2.3.4:9090
   ↓
2. TLS 握手
   → 服务端发送证书
   ↓
3. 客户端验证证书
   → 检查证书是否由信任的 CA 签发 ✅
   → 检查证书有效期 ✅
   → 检查 ServerName (1.2.3.4) 是否在证书 SAN 中 ✅
   ↓
4. 验证通过，建立连接
```

---

## 常见问题

### Q1: 配置后仍然连接失败？

**A**: 检查以下几点：
1. 是否重启了服务端？（必须重启才能生成新证书）
2. 配置的地址是否正确？（检查拼写和格式）
3. 查看服务端日志，确认证书包含了配置的地址

```bash
sudo journalctl -u opshub -f | grep "生成服务端证书"
```

---

### Q2: 如何添加多个地址？

**A**: 在 `server_addresses` 数组中添加多行：

```yaml
server_addresses:
  - "1.2.3.4"
  - "5.6.7.8"
  - "opshub.example.com"
  - "opshub-backup.example.com"
```

---

### Q3: 支持 IPv6 吗？

**A**: 当前版本仅支持 IPv4。如需 IPv6 支持，需要修改 `getLocalIPs()` 函数。

---

### Q4: 配置域名需要 DNS 解析吗？

**A**: 
- 证书生成时不需要 DNS 解析
- 客户端连接时需要能够解析域名到服务端 IP

---

### Q5: 可以使用通配符域名吗？

**A**: 可以，例如：

```yaml
server_addresses:
  - "*.opshub.example.com"
```

这样 `agent1.opshub.example.com`、`agent2.opshub.example.com` 等都可以连接。

---

### Q6: 配置变更后需要重新部署 Agent 吗？

**A**: 不需要。只需重启服务端，Agent 会自动重连并使用新证书。

---

### Q7: 如何查看当前证书包含哪些地址？

**A**: 查看服务端启动日志：

```bash
sudo journalctl -u opshub -n 100 | grep "生成服务端证书"
```

或使用 openssl 命令查看证书内容（见上文"验证证书内容"）。

---

## 配置示例

### 示例 1：简单 NAT 场景

```yaml
agent:
  enabled: true
  cert_dir: "data/agent-certs"
  binary_dir: "data/agent-binaries"
  heartbeat_timeout: 180
  deploy_path: "/opt/srehub-agent"
  server_addresses:
    - "1.2.3.4"  # NAT 公网 IP
```

---

### 示例 2：多网卡 + 域名

```yaml
agent:
  enabled: true
  cert_dir: "data/agent-certs"
  binary_dir: "data/agent-binaries"
  heartbeat_timeout: 180
  deploy_path: "/opt/srehub-agent"
  server_addresses:
    - "opshub.example.com"   # 主域名
    - "192.168.1.100"        # 内网 IP
    - "10.0.0.5"             # 管理网 IP
```

---

### 示例 3：负载均衡 + 备份

```yaml
agent:
  enabled: true
  cert_dir: "data/agent-certs"
  binary_dir: "data/agent-binaries"
  heartbeat_timeout: 180
  deploy_path: "/opt/srehub-agent"
  server_addresses:
    - "lb.opshub.example.com"      # 负载均衡器
    - "opshub1.example.com"        # 节点1
    - "opshub2.example.com"        # 节点2
    - "*.opshub.example.com"       # 通配符（覆盖所有子域名）
```

---

### 示例 4：仅使用自动检测（默认）

```yaml
agent:
  enabled: true
  cert_dir: "data/agent-certs"
  binary_dir: "data/agent-binaries"
  heartbeat_timeout: 180
  deploy_path: "/opt/srehub-agent"
  server_addresses: []  # 空数组，仅使用自动检测的 IP
```

---

## 技术细节

### 代码变更

**1. 配置结构体**（`internal/conf/conf.go`）：
```go
type AgentConfig struct {
    Enabled          bool     `mapstructure:"enabled"`
    CertDir          string   `mapstructure:"cert_dir"`
    BinaryDir        string   `mapstructure:"binary_dir"`
    HeartbeatTimeout int      `mapstructure:"heartbeat_timeout"`
    DeployPath       string   `mapstructure:"deploy_path"`
    ServerAddresses  []string `mapstructure:"server_addresses"` // 新增
}
```

**2. 证书生成逻辑**（`internal/server/agent/tls.go`）：
```go
func (m *TLSManager) LoadServerTLSConfig(configAddresses []string) (*tls.Config, error) {
    // 1. 自动检测本机 IP
    localIPs := getLocalIPs()
    
    // 2. 从配置文件读取额外地址
    for _, addr := range configAddresses {
        if ip := net.ParseIP(addr); ip != nil {
            ipAddresses = append(ipAddresses, ip)  // IP 地址
        } else {
            dnsNames = append(dnsNames, addr)      // 域名
        }
    }
    
    // 3. 生成证书
    serverTemplate := &x509.Certificate{
        IPAddresses: ipAddresses,
        DNSNames:    dnsNames,
    }
}
```

**3. 调用方式**（`internal/server/agent/grpc_server.go`）：
```go
func (s *GRPCServer) Start() error {
    // 传入配置的额外地址
    tlsConfig, err := s.tlsMgr.LoadServerTLSConfig(s.conf.Agent.ServerAddresses)
}
```

---

## 相关文档

- [Agent TLS 证书架构详解](agent-tls-architecture.md)
- [Agent TLS 证书修复详情](agent-tls-certificate-fix.md)
- [Agent 快速操作指南](agent-fix-summary.md)

---

## 修复日期

2026-04-07

## 修复人员

Claude (AI Assistant)
