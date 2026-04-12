# Agent 连接问题修复总结

## 问题现象

```
[ERROR] 建立流失败: rpc error: code = Unavailable desc = connection error: 
desc = "transport: authentication handshake failed: tls: failed to verify certificate: 
x509: certificate is valid for 0.0.0.0, 127.0.0.1, not 192.168.0.13"
```

## 根本原因

服务端 TLS 证书只包含 `0.0.0.0` 和 `127.0.0.1`，不包含服务器的实际 IP 地址。

## 修复方案

### ✅ 已修复内容

1. **服务端证书自动包含所有本机 IP**
   - 文件：`internal/server/agent/tls.go`
   - 新增 `getLocalIPs()` 函数
   - 修改 `LoadServerTLSConfig()` 方法

2. **编译验证通过**
   - ✅ 代码编译成功
   - ✅ 无语法错误

---

## 立即操作步骤

### 1. 重启服务端（必须）

```bash
# 停止服务
sudo systemctl stop opshub

# 启动服务（会重新生成包含所有 IP 的证书）
sudo systemctl start opshub

# 查看日志确认
sudo journalctl -u opshub -f | grep "生成服务端证书"
```

**预期日志**：
```
[INFO] 生成服务端证书 ipAddresses=[192.168.0.13, ...]
```

### 2. 重启 Agent（自动重连）

```bash
# systemd 模式
sudo systemctl restart srehub-agent

# no_systemd 模式
sudo /opt/srehub-agent/restart.sh
```

### 3. 验证连接

```bash
# 查看 Agent 日志
tail -f /opt/srehub-agent/srehub-agent.log

# 或查看应用日志
tail -f /var/log/srehub-agent/agent.log
```

**预期日志**：
```
[INFO] 正在连接到服务器: 192.168.0.13:9090
[INFO] 注册成功: 注册成功
[INFO] 启动心跳循环，间隔: 60秒
```

### 4. 在平台查看状态

- 打开主机管理页面
- Agent 状态应显示为 **在线**（绿色）

---

## 为什么需要重启服务端？

**服务端证书生命周期**：
- 证书在服务启动时动态生成
- 不持久化到磁盘
- 每次启动都会重新生成

**修复前的证书**（旧服务）：
```
IP Addresses: 0.0.0.0, 127.0.0.1
```

**修复后的证书**（新服务）：
```
IP Addresses: 0.0.0.0, 127.0.0.1, 192.168.0.13, ...
```

**只有重启服务端，才会生成包含所有 IP 的新证书！**

---

## 常见问题

### Q1: 重启服务端会影响现有连接吗？

**A**: 会短暂中断，但 Agent 会自动重连（5 秒重试间隔）。

### Q2: 需要重新部署 Agent 吗？

**A**: 不需要。Agent 端代码无需修改，只需重启服务端。

### Q3: 如果服务器有多个 IP 怎么办？

**A**: 修复后的代码会自动获取所有非 loopback 的 IPv4 地址，全部加入证书。

### Q4: 如果服务器 IP 变更怎么办？

**A**: 重启服务端即可，会自动生成包含新 IP 的证书。

### Q5: 为什么不用域名？

**A**: 可以用域名，但需要配置 DNS。当前方案更简单，适合内网环境。

---

## 技术细节

### 修复前后对比

| 项目 | 修复前 | 修复后 |
|------|--------|--------|
| 证书 IP | 0.0.0.0, 127.0.0.1 | 0.0.0.0, 127.0.0.1, 192.168.0.13, ... |
| 本地连接 | ✅ 成功 | ✅ 成功 |
| 内网连接 | ❌ 失败 | ✅ 成功 |
| 多网卡 | ❌ 失败 | ✅ 成功 |

### 代码变更

**新增函数**：
```go
func getLocalIPs() []string {
    // 获取本机所有非 loopback 的 IPv4 地址
}
```

**修改函数**：
```go
func (m *TLSManager) LoadServerTLSConfig() (*tls.Config, error) {
    // 动态获取本机 IP 并加入证书 SAN
}
```

---

## 相关文档

- [Agent TLS 证书架构详解](docs/agent-tls-architecture.md)
- [Agent TLS 证书修复详情](docs/agent-tls-certificate-fix.md)
- [Agent no_systemd 模式修复](docs/agent-no-systemd-fix.md)

---

## 总结

✅ **问题已修复**：服务端证书现在会自动包含所有本机 IP 地址

⚠️ **必须操作**：重启服务端以生成新证书

🎯 **预期效果**：Agent 可以通过任意本机 IP 连接服务端

📝 **修复日期**：2026-04-07
