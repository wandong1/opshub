# Agent NAT 场景快速配置指南

## 问题现象

```
[ERROR] tls: failed to verify certificate: x509: certificate is valid for 0.0.0.0, 127.0.0.1, not 1.2.3.4
```

## 解决步骤

### 1. 编辑配置文件

```bash
vi config/config.yaml
```

### 2. 添加 NAT 公网 IP

```yaml
agent:
  enabled: true
  cert_dir: "data/agent-certs"
  binary_dir: "data/agent-binaries"
  heartbeat_timeout: 180
  deploy_path: "/opt/srehub-agent"
  server_addresses:
    - "1.2.3.4"  # 替换为你的 NAT 公网 IP
```

### 3. 重启服务端

```bash
sudo systemctl restart opshub
```

### 4. 查看日志确认

```bash
sudo journalctl -u opshub -f | grep "生成服务端证书"
```

**预期输出**：
```
[INFO] 生成服务端证书 configIPs=[1.2.3.4] totalIPs=3
```

### 5. Agent 自动重连

无需操作，Agent 会自动重连并验证通过。

---

## 常见配置

### NAT 场景
```yaml
server_addresses:
  - "1.2.3.4"  # NAT 公网 IP
```

### 域名场景
```yaml
server_addresses:
  - "opshub.example.com"
```

### 多网卡场景
```yaml
server_addresses:
  - "192.168.1.100"  # 内网
  - "10.0.0.5"       # 管理网
```

### 混合场景
```yaml
server_addresses:
  - "1.2.3.4"              # 公网 IP
  - "opshub.example.com"   # 域名
  - "192.168.1.100"        # 内网 IP
```

---

## 完成！

配置后重启服务端即可解决 NAT 场景下的 TLS 证书验证问题。
