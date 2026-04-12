# SREHub Agent 手动部署指南

## 方式一：通过平台生成安装包（推荐）

### 1. 生成安装包

在 OpsHub 平台的 **主机管理** 页面，点击 **生成Agent安装包** 按钮，填写服务端地址（如 `192.168.1.100`），点击生成后下载 `.tar.gz` 安装包。

该安装包已包含：
- Agent 二进制文件（多架构）
- 预签发的 mTLS 证书（ca.pem、cert.pem、key.pem，专用证书）
- 预填好的 agent.yaml 配置文件
- install.sh 安装脚本
- systemd 服务文件

### 2. 上传并安装

```bash
# 上传安装包到目标机器后解压
tar xzf srehub-agent-*.tar.gz
cd srehub-agent-*

# 方式 1: 使用 systemd 管理（推荐，Linux 系统）
sudo bash install.sh 192.168.1.100:9090

# 方式 2: 使用脚本管理（适用于无 systemd 的环境或 macOS）
sudo bash install.sh 192.168.1.100:9090 no_systemd

# 启动服务
# systemd 模式:
sudo systemctl start srehub-agent
sudo systemctl status srehub-agent

# 脚本管理模式:
sudo /opt/srehub-agent/start.sh
sudo /opt/srehub-agent/status.sh
```

**安装脚本功能**：
- 自动检测操作系统和架构，选择对应的二进制文件
- 自动生成 UUID 作为 agent_id
- 支持通过参数指定服务端地址：`./install.sh <SERVER_ADDR>`
- 支持禁用 systemd：`./install.sh <SERVER_ADDR> no_systemd`
- 自动复制证书文件（如果存在）
- 自动安装 systemd 服务或生成管理脚本

**管理模式**：
- **systemd 模式**（默认）：适用于支持 systemd 的 Linux 系统，自动设置开机自启
- **脚本管理模式**：适用于无 systemd 的环境、macOS 或需要手动控制的场景，生成 start.sh/stop.sh/restart.sh/status.sh 脚本

安装完成后，Agent 会自动连接服务端并注册，主机将自动出现在平台的主机管理列表中，同时根据运行的进程自动匹配服务标签。

---

## 方式二：使用通用证书手动部署（快速部署）

适用于批量部署、内网环境或快速测试场景。使用预打包的通用客户端证书（有效期 10 年），所有 Agent 共享同一个证书。

### 1. 获取安装包

从服务端的 `data/agent-binaries/` 目录复制多架构安装包到目标机器：

```bash
scp user@opshub-server:/path/to/opshub/data/agent-binaries/srehub-agent-1.0.0-multi-arch.tar.gz ./
```

该安装包已包含：
- Agent 二进制文件（多架构：linux-amd64/arm64, darwin-amd64/arm64）
- 通用 mTLS 证书（ca.pem、cert.pem、key.pem，有效期 10 年）
- agent.yaml 配置模板
- install.sh 智能安装脚本
- systemd 服务文件

### 2. 解压并安装

```bash
# 解压
tar xzf srehub-agent-1.0.0-multi-arch.tar.gz
cd srehub-agent-1.0.0-multi-arch

# 安装（指定服务端地址，脚本会自动生成 agent_id 并复制证书）
sudo bash install.sh 192.168.1.100:9090

# 或使用脚本管理模式
sudo bash install.sh 192.168.1.100:9090 no_systemd

# 启动服务
sudo systemctl start srehub-agent
# 或
sudo /opt/srehub-agent/start.sh
```

**通用证书说明**：
- 所有手动部署的 Agent 使用相同的客户端证书
- 证书有效期 10 年（2026-2036）
- 适用于内网环境、信任度高的场景
- 简化部署流程，无需为每个 Agent 单独生成证书

**安全建议**：
- 生产环境建议使用"方式一"为每个 Agent 生成独立证书
- 通用证书适用于开发、测试或内网环境
- 如需更高安全性，可在部署后通过平台更新为独立证书

---

## 方式三：完全手动部署（高级）

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

**选项 A**：使用安装包中的通用证书（如果已打包）

安装包中已包含通用证书，可直接使用。

**选项 B**：通过平台 API 生成独立证书（需要有效 Token）

```bash
# 调用生成安装包 API 获取证书
curl -X POST http://opshub-server:9876/api/v1/agents/generate-install \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{"serverAddr": "192.168.1.100"}'

# 返回中包含 agentId 和 downloadUrl，下载后从中提取证书
```

**选项 C**：手动复制服务端 CA 证书并签发

```bash
# 从服务端复制 CA 证书
scp user@opshub-server:/path/to/opshub/data/agent-certs/ca.pem ./certs/
scp user@opshub-server:/path/to/opshub/data/agent-certs/ca-key.pem /tmp/

# 使用 openssl 签发客户端证书
AGENT_ID=$(uuidgen | tr '[:upper:]' '[:lower:]')
openssl ecparam -genkey -name prime256v1 -out certs/key.pem
openssl req -new -key certs/key.pem -out /tmp/agent.csr -subj "/O=OpsHub Agent/CN=${AGENT_ID}"
openssl x509 -req -in /tmp/agent.csr -CA certs/ca.pem -CAkey /tmp/ca-key.pem \
  -CAcreateserial -out certs/cert.pem -days 3650
chmod 600 certs/key.pem
rm -f /tmp/agent.csr /tmp/ca-key.pem

echo "Agent ID: ${AGENT_ID}"
```

### 4. 配置 agent.yaml

如果使用的是未预配置的安装包，需要手动配置：

```bash
cat > agent.yaml << EOF
agent_id: "$(uuidgen | tr '[:upper:]' '[:lower:]')"
server_addr: "<服务端IP>:9090"
cert_dir: "/opt/srehub-agent/certs"
log_file: "/var/log/srehub-agent/agent.log"
log_max_size: 100
log_max_backups: 3
log_max_age: 30
log_level: "info"
EOF
```

参数说明：
- `agent_id`：Agent 唯一标识，UUID 格式（安装脚本会自动生成）
- `server_addr`：OpsHub 服务端 gRPC 地址，默认端口 9090
- `cert_dir`：mTLS 证书目录（绝对路径）
- `log_file`：日志文件路径
- `log_max_size`：单个日志文件最大大小（MB）
- `log_max_backups`：保留的旧日志文件数量
- `log_max_age`：日志文件最大保留天数
- `log_level`：日志级别（debug/info/warn/error）

### 5. 安装

```bash
# 使用安装脚本（推荐）
sudo bash install.sh <服务端地址>
# 示例: sudo bash install.sh 192.168.1.100:9090

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

**install.sh 脚本用法**：
```bash
# 语法
./install.sh [SERVER_ADDR] [no_systemd]

# 示例 1: 使用 systemd 管理（默认）
sudo bash install.sh 192.168.1.100:9090

# 示例 2: 使用脚本管理
sudo bash install.sh 192.168.1.100:9090 no_systemd

# 示例 3: 不指定地址，使用安装包预配置
sudo bash install.sh

# 示例 4: 仅禁用 systemd
sudo bash install.sh no_systemd

# 示例 5: 自定义安装目录
sudo INSTALL_DIR=/usr/local/srehub-agent bash install.sh 192.168.1.100:9090
```

**管理脚本说明**（no_systemd 模式）：
- `start.sh` - 启动 Agent（后台运行，PID 保存到 srehub-agent.pid）
- `stop.sh` - 停止 Agent（优雅关闭，超时后强制终止）
- `restart.sh` - 重启 Agent
- `status.sh` - 查看 Agent 运行状态

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

**systemd 模式**：
```bash
sudo systemctl start srehub-agent     # 启动
sudo systemctl stop srehub-agent      # 停止
sudo systemctl restart srehub-agent   # 重启
sudo systemctl status srehub-agent    # 状态
sudo journalctl -u srehub-agent -f    # 查看日志
```

**脚本管理模式**：
```bash
sudo /opt/srehub-agent/start.sh       # 启动
sudo /opt/srehub-agent/stop.sh        # 停止
sudo /opt/srehub-agent/restart.sh     # 重启
sudo /opt/srehub-agent/status.sh      # 状态
tail -f /var/log/srehub-agent/agent.log  # 查看日志
```
