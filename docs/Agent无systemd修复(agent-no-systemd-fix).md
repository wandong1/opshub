# Agent no_systemd 模式启动脚本修复

## 问题描述

在 `no_systemd` 模式下，`start.sh` 脚本无法正确启动 Agent 应用程序。

**错误的启动命令**：
```bash
nohup "${INSTALL_DIR}/srehub-agent" > /dev/null 2>&1 &
```

**问题原因**：
- 缺少 `run` 子命令
- 缺少 `-c` 配置文件参数
- Agent 使用 Cobra 命令行框架，需要明确指定子命令

**正确的启动命令**：
```bash
/opt/srehub-agent/srehub-agent run -c /opt/srehub-agent/agent.yaml
```

---

## 修复内容

### 1. 修复 start.sh 脚本

**文件位置**：`agent/build.sh:230-251`

**修改前**：
```bash
echo "启动 SREHub Agent..."
cd "${INSTALL_DIR}"
nohup "${INSTALL_DIR}/srehub-agent" > /dev/null 2>&1 &
echo $! > "${PID_FILE}"
echo "SREHub Agent 已启动 (PID: $(cat ${PID_FILE}))"
```

**修改后**：
```bash
echo "启动 SREHub Agent..."
cd "${INSTALL_DIR}"
nohup "${INSTALL_DIR}/srehub-agent" run -c "${INSTALL_DIR}/agent.yaml" >> "${LOG_FILE}" 2>&1 &
echo $! > "${PID_FILE}"
echo "SREHub Agent 已启动 (PID: $(cat ${PID_FILE}))"
echo "日志文件: ${LOG_FILE}"
```

**改进点**：
- ✅ 添加 `run` 子命令
- ✅ 添加 `-c "${INSTALL_DIR}/agent.yaml"` 配置文件参数
- ✅ 将日志输出到 `${INSTALL_DIR}/srehub-agent.log`（追加模式）
- ✅ 显示日志文件路径

---

### 2. 增强 status.sh 脚本

**文件位置**：`agent/build.sh:304-325`

**修改前**：
```bash
if kill -0 "${PID}" 2>/dev/null; then
    echo "SREHub Agent 正在运行 (PID: ${PID})"
    ps -p "${PID}" -o pid,ppid,cmd,etime
    exit 0
fi
```

**修改后**：
```bash
if kill -0 "${PID}" 2>/dev/null; then
    echo "SREHub Agent 正在运行 (PID: ${PID})"
    ps -p "${PID}" -o pid,ppid,cmd,etime
    echo ""
    echo "日志文件: ${LOG_FILE}"
    if [ -f "${LOG_FILE}" ]; then
        echo "最近日志:"
        tail -n 10 "${LOG_FILE}"
    fi
    exit 0
fi
```

**改进点**：
- ✅ 显示日志文件路径
- ✅ 自动显示最近 10 行日志
- ✅ 方便快速排查问题

---

## 验证步骤

### 1. 重新构建安装包

```bash
cd agent/
bash build.sh
```

### 2. 测试安装（no_systemd 模式）

```bash
cd /tmp
tar -xzf /path/to/srehub-agent-1.0.0-multi-arch.tar.gz
cd srehub-agent-1.0.0-multi-arch
sudo bash install.sh 192.168.1.100:9090 no_systemd
```

### 3. 测试启动

```bash
# 启动 Agent
sudo /opt/srehub-agent/start.sh

# 输出示例：
# 启动 SREHub Agent...
# SREHub Agent 已启动 (PID: 12345)
# 日志文件: /opt/srehub-agent/srehub-agent.log
```

### 4. 查看状态

```bash
# 查看运行状态
sudo /opt/srehub-agent/status.sh

# 输出示例：
# SREHub Agent 正在运行 (PID: 12345)
#   PID  PPID CMD                          ELAPSED
# 12345     1 /opt/srehub-agent/srehub-... 00:01:23
# 
# 日志文件: /opt/srehub-agent/srehub-agent.log
# 最近日志:
# 2026-04-07 20:35:00 INFO SREHub Agent 启动中...
# 2026-04-07 20:35:00 INFO AgentID: ae5e4a95-aaeb-498c-acb5-b802bbe4b550
# 2026-04-07 20:35:00 INFO Server: localhost:9090
# ...
```

### 5. 查看日志

```bash
# 实时查看日志
tail -f /opt/srehub-agent/srehub-agent.log
```

### 6. 停止服务

```bash
sudo /opt/srehub-agent/stop.sh
```

---

## 影响范围

### 受影响的部署方式

1. ✅ **通过平台生成安装包** → 手动部署（no_systemd 模式）
2. ✅ **使用通用证书手动部署** → no_systemd 模式
3. ✅ **完全手动部署** → no_systemd 模式
4. ✅ **macOS 平台部署**（默认使用 no_systemd 模式）

### 不受影响的部署方式

- ❌ systemd 模式（使用 systemd 服务文件，不依赖 start.sh）
- ❌ 通过平台 SSH 批量部署（默认使用 systemd）

---

## 相关文件

- `agent/build.sh` - 构建脚本（包含 install.sh 模板）
- `agent/cmd/main.go` - Agent 命令行入口（Cobra 框架）
- `agent/README.md` - Agent 部署文档

---

## 技术细节

### Agent 命令行结构

```
srehub-agent
├── run          # 启动 Agent（需要 -c 参数指定配置文件）
├── version      # 显示版本信息
└── help         # 帮助信息
```

### 正确的启动方式

```bash
# 方式 1: 指定配置文件路径
srehub-agent run -c /opt/srehub-agent/agent.yaml

# 方式 2: 在配置文件所在目录执行（默认查找 agent.yaml）
cd /opt/srehub-agent
srehub-agent run

# 方式 3: 使用相对路径
cd /opt/srehub-agent
srehub-agent run -c ./agent.yaml
```

### 错误的启动方式

```bash
# ❌ 缺少 run 子命令
srehub-agent

# ❌ 缺少配置文件参数
srehub-agent run

# ❌ 配置文件路径错误
srehub-agent run -c agent.yaml  # 如果不在配置文件目录
```

---

## 日志管理

### 日志文件位置

- **systemd 模式**：通过 journalctl 查看
  ```bash
  journalctl -u srehub-agent -f
  ```

- **no_systemd 模式**：
  - 启动日志：`${INSTALL_DIR}/srehub-agent.log`
  - 应用日志：配置文件中的 `log_file`（默认 `/var/log/srehub-agent/agent.log`）

### 日志配置

在 `agent.yaml` 中配置：
```yaml
log_file: "/var/log/srehub-agent/agent.log"
log_max_size: 100      # 单个日志文件最大大小（MB）
log_max_backups: 3     # 保留的旧日志文件数量
log_max_age: 30        # 日志文件最大保留天数
log_level: "info"      # 日志级别：debug/info/warn/error
```

---

## 后续改进建议

1. **日志轮转**：考虑使用 logrotate 管理 no_systemd 模式的日志
2. **健康检查**：添加 Agent 健康检查端点，status.sh 可以调用
3. **自动重启**：添加守护进程监控，Agent 异常退出时自动重启
4. **日志级别动态调整**：支持通过信号量动态调整日志级别

---

## 修复日期

2026-04-07

## 修复人员

Claude (AI Assistant)
