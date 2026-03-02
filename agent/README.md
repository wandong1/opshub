# SREHub Agent 手动部署指南

## 方式一：通过平台生成安装包（推荐）

### 1. 生成安装包

在 OpsHub 平台的 **主机管理** 页面，点击 **生成Agent安装包** 按钮，填写服务端地址（如 `192.168.1.100`），点击生成后下载 `.tar.gz` 安装包。

该安装包已包含：
- Agent 二进制文件
- 预签发的 mTLS 证书（ca.pem、cert.pem、key.pem）
- 预填好的 agent.yaml 配置文件
- install.sh 安装脚本
- systemd 服务文件

### 2. 上传并安装

```bash
# 上传安装包到目标机器后解压
tar xzf srehub-agent-*.tar.gz
cd srehub-agent-*

# 执行安装（默认安装到 /opt/srehub-agent）
sudo bash install.sh

# 启动服务
sudo systemctl start srehub-agent
sudo systemctl status srehub-agent
```

安装完成后，Agent 会自动连接服务端并注册，主机将自动出现在平台的主机管理列表中，同时根据运行的进程自动匹配服务标签。

---

## 方式二：完全手动部署

适用于无法通过平台生成安装包的场景。

### 1. 获取安装包

从服务端的 `data/agent-binaries/` 目录复制 `srehub-agent-*.tar.gz` 到目标机器。

```bash
scp user@opshub-server:/path/to/opshub/data/agent-binaries/srehub-agent-*.tar.gz ./
```

### 2. 解压

```bash
tar xzf srehub-agent-*.tar.gz
cd srehub-agent-*
```

### 3. 获取证书

从服务端的 `data/agent-certs/` 目录复制 CA 证书，并为该 Agent 签发客户端证书。

**方法 A**：通过平台 API 生成（需要有效 Token）

```bash
# 调用生成安装包 API 获取证书
curl -X POST http://opshub-server:9876/api/v1/agents/generate-install \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{"serverAddr": "192.168.1.100"}'

# 返回中包含 agentId 和 downloadUrl，下载后从中提取证书
```

**方法 B**：手动复制服务端 CA 证书

```bash
# 从服务端复制 CA 证书
scp user@opshub-server:/path/to/opshub/data/agent-certs/ca.pem ./certs/
scp user@opshub-server:/path/to/opshub/data/agent-certs/ca-key.pem /tmp/

# 使用 openssl 签发客户端证书
AGENT_ID=$(uuidgen)
openssl genrsa -out certs/key.pem 2048
openssl req -new -key certs/key.pem -out /tmp/agent.csr -subj "/CN=${AGENT_ID}"
openssl x509 -req -in /tmp/agent.csr -CA certs/ca.pem -CAkey /tmp/ca-key.pem \
  -CAcreateserial -out certs/cert.pem -days 3650
chmod 600 certs/key.pem
rm -f /tmp/agent.csr /tmp/ca-key.pem

echo "Agent ID: ${AGENT_ID}"
```

### 4. 配置 agent.yaml

```bash
cat > agent.yaml << EOF
agent_id: "<上一步生成的 AGENT_ID>"
server_addr: "<服务端IP>:9090"
cert_dir: "/opt/srehub-agent/certs"
log_file: "/var/log/srehub-agent.log"
EOF
```

参数说明：
- `agent_id`：Agent 唯一标识，UUID 格式
- `server_addr`：OpsHub 服务端 gRPC 地址，默认端口 9090
- `cert_dir`：mTLS 证书目录
- `log_file`：日志文件路径

### 5. 安装

```bash
sudo bash install.sh
# 或手动安装：
sudo mkdir -p /opt/srehub-agent/certs
sudo cp srehub-agent /opt/srehub-agent/
sudo chmod +x /opt/srehub-agent/srehub-agent
sudo cp agent.yaml /opt/srehub-agent/
sudo cp certs/* /opt/srehub-agent/certs/
sudo chmod 600 /opt/srehub-agent/certs/key.pem
sudo cp srehub-agent.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now srehub-agent
```

### 6. 验证

```bash
# 查看服务状态
sudo systemctl status srehub-agent

# 查看日志
sudo journalctl -u srehub-agent -f

# 或查看日志文件
tail -f /var/log/srehub-agent.log
```

注册成功后日志会显示 `注册成功`，主机将自动出现在平台主机管理列表中。

---

## 自动注册说明

手动部署的 Agent 连接服务端后会自动完成以下操作：

1. **自动创建主机记录** — 使用 Agent 上报的 hostname、IP、OS、架构信息创建主机
2. **自动匹配服务标签** — 执行 `ps -eo comm=` 获取进程列表，与平台"服务标签管理"中配置的规则匹配，自动为主机打上标签（如 mysqld、docker、k8s-node 等）
3. **自动采集主机信息** — CPU、内存、磁盘等基础信息

自动注册的主机默认不属于任何分组（GroupID=0），可在平台中手动分配。

---

## 目录结构

```
/opt/srehub-agent/
├── srehub-agent          # Agent 二进制
├── agent.yaml            # 配置文件
├── certs/
│   ├── ca.pem            # CA 证书
│   ├── cert.pem          # 客户端证书
│   └── key.pem           # 客户端私钥
```

## 常用命令

```bash
sudo systemctl start srehub-agent     # 启动
sudo systemctl stop srehub-agent      # 停止
sudo systemctl restart srehub-agent   # 重启
sudo systemctl status srehub-agent    # 状态
sudo journalctl -u srehub-agent -f    # 查看日志
```
