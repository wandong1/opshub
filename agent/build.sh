#!/bin/bash
# SREHub Agent 构建脚本
# 用法: ./build.sh [linux/amd64] [linux/arm64] [darwin/amd64] [darwin/arm64]
# 默认构建所有支持的平台

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION="${VERSION:-1.0.0}"
BUILD_DIR="${SCRIPT_DIR}/dist"
AGENT_DIR="${SCRIPT_DIR}"

# 默认目标平台（如果没有指定参数，构建所有平台）
if [ $# -eq 0 ]; then
    TARGETS="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64"
else
    TARGETS="$@"
fi

echo "========================================="
echo "  SREHub Agent 构建工具 v${VERSION}"
echo "========================================="

# 清理旧构建
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

# 创建多架构安装包目录
MULTI_ARCH_DIR="${BUILD_DIR}/srehub-agent-${VERSION}-multi-arch"
mkdir -p "${MULTI_ARCH_DIR}/bin"
mkdir -p "${MULTI_ARCH_DIR}/certs"

for TARGET in ${TARGETS}; do
    OS=$(echo "${TARGET}" | cut -d'/' -f1)
    ARCH=$(echo "${TARGET}" | cut -d'/' -f2)
    BINARY_NAME="srehub-agent-${OS}-${ARCH}"

    echo ""
    echo ">>> 构建 ${OS}/${ARCH} ..."

    # 编译二进制到多架构目录
    echo "  编译二进制..."
    (cd "${AGENT_DIR}" && CGO_ENABLED=0 GOOS="${OS}" GOARCH="${ARCH}" \
        go build -ldflags "-s -w -X main.Version=${VERSION}" \
        -o "${MULTI_ARCH_DIR}/bin/${BINARY_NAME}" \
        ./cmd/main.go)

    echo "  完成: bin/${BINARY_NAME}"
done

# 复制配置模板到多架构目录
cp "${AGENT_DIR}/config/agent.yaml" "${MULTI_ARCH_DIR}/agent.yaml"
cp "${AGENT_DIR}/config/srehub-agent.service" "${MULTI_ARCH_DIR}/srehub-agent.service"

# 复制通用证书到多架构目录（如果存在）
COMMON_CERT_DIR="${SCRIPT_DIR}/../data/agent-certs/common"
if [ -f "${COMMON_CERT_DIR}/cert.pem" ] && [ -f "${COMMON_CERT_DIR}/key.pem" ]; then
    echo ""
    echo ">>> 打包通用客户端证书..."
    cp "${SCRIPT_DIR}/../data/agent-certs/ca.pem" "${MULTI_ARCH_DIR}/certs/"
    cp "${COMMON_CERT_DIR}/cert.pem" "${MULTI_ARCH_DIR}/certs/"
    cp "${COMMON_CERT_DIR}/key.pem" "${MULTI_ARCH_DIR}/certs/"
    echo "  已包含通用证书（有效期 10 年）"
else
    echo ""
    echo ">>> 警告: 未找到通用证书，安装包将不包含证书"
    echo "    运行以下命令生成通用证书:"
    echo "    bash scripts/generate-common-agent-cert.sh"
fi

# 创建智能安装脚本（自动检测架构）
cat > "${MULTI_ARCH_DIR}/install.sh" << 'INSTALL_EOF'
#!/bin/bash
# SREHub Agent 多架构安装脚本
# 用法: ./install.sh [SERVER_ADDR] [no_systemd]
# 示例: ./install.sh 192.168.1.100:9090
#       ./install.sh 192.168.1.100:9090 no_systemd
set -e

INSTALL_DIR="${INSTALL_DIR:-/opt/srehub-agent}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVER_ADDR="${1:-}"
USE_SYSTEMD="yes"

# 检查是否禁用 systemd
if [ "$2" = "no_systemd" ] || [ "$1" = "no_systemd" ]; then
    USE_SYSTEMD="no"
    # 如果第一个参数是 no_systemd，则没有指定服务端地址
    [ "$1" = "no_systemd" ] && SERVER_ADDR=""
fi

echo "========================================="
echo "  SREHub Agent 安装程序"
echo "========================================="

# 检测操作系统和架构
detect_platform() {
    local os=""
    local arch=""

    # 检测操作系统
    case "$(uname -s)" in
        Linux*)  os="linux" ;;
        Darwin*) os="darwin" ;;
        *)
            echo "错误: 不支持的操作系统 $(uname -s)"
            exit 1
            ;;
    esac

    # 检测架构
    case "$(uname -m)" in
        x86_64|amd64)  arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *)
            echo "错误: 不支持的架构 $(uname -m)"
            exit 1
            ;;
    esac

    echo "${os}/${arch}"
}

# 生成 UUID（兼容 Linux 和 macOS）
generate_uuid() {
    if command -v uuidgen >/dev/null 2>&1; then
        uuidgen | tr '[:upper:]' '[:lower:]'
    elif [ -f /proc/sys/kernel/random/uuid ]; then
        cat /proc/sys/kernel/random/uuid
    else
        # 降级方案：使用随机数生成伪 UUID
        printf '%08x-%04x-%04x-%04x-%012x\n' \
            $RANDOM$RANDOM $RANDOM $RANDOM $RANDOM $RANDOM$RANDOM$RANDOM
    fi
}

PLATFORM=$(detect_platform)
OS=$(echo "${PLATFORM}" | cut -d'/' -f1)
ARCH=$(echo "${PLATFORM}" | cut -d'/' -f2)
BINARY_NAME="srehub-agent-${OS}-${ARCH}"

echo "检测到平台: ${PLATFORM}"

# 检查二进制文件是否存在
if [ ! -f "${SCRIPT_DIR}/bin/${BINARY_NAME}" ]; then
    echo "错误: 未找到适用于 ${PLATFORM} 的二进制文件"
    echo "可用的二进制文件:"
    ls -1 "${SCRIPT_DIR}/bin/" 2>/dev/null || echo "  (无)"
    exit 1
fi

# 创建目录
echo "创建安装目录: ${INSTALL_DIR}"
mkdir -p "${INSTALL_DIR}/certs"
mkdir -p "$(dirname /var/log/srehub-agent/agent.log)" 2>/dev/null || true

# 复制二进制文件
echo "安装二进制文件: ${BINARY_NAME}"
cp "${SCRIPT_DIR}/bin/${BINARY_NAME}" "${INSTALL_DIR}/srehub-agent"
chmod +x "${INSTALL_DIR}/srehub-agent"

# 处理配置文件
if [ -f "${INSTALL_DIR}/agent.yaml" ]; then
    echo "配置文件已存在，跳过覆盖"
else
    echo "生成配置文件..."

    # 生成 Agent ID
    AGENT_ID=$(generate_uuid)
    echo "  生成 Agent ID: ${AGENT_ID}"

    # 确定服务端地址
    if [ -z "${SERVER_ADDR}" ]; then
        # 如果安装包中已有预配置的地址，则使用它
        if [ -f "${SCRIPT_DIR}/agent.yaml" ]; then
            CONFIGURED_ADDR=$(grep '^server_addr:' "${SCRIPT_DIR}/agent.yaml" | awk '{print $2}' | tr -d '"')
            if [ "${CONFIGURED_ADDR}" != "your-server-ip:9090" ] && [ -n "${CONFIGURED_ADDR}" ]; then
                SERVER_ADDR="${CONFIGURED_ADDR}"
                echo "  使用预配置的服务端地址: ${SERVER_ADDR}"
            else
                echo "警告: 未指定服务端地址，请手动编辑 ${INSTALL_DIR}/agent.yaml"
                SERVER_ADDR="your-server-ip:9090"
            fi
        else
            echo "警告: 未指定服务端地址，请手动编辑 ${INSTALL_DIR}/agent.yaml"
            SERVER_ADDR="your-server-ip:9090"
        fi
    else
        echo "  使用指定的服务端地址: ${SERVER_ADDR}"
    fi

    # 生成配置文件
    cat > "${INSTALL_DIR}/agent.yaml" << EOF
agent_id: "${AGENT_ID}"
server_addr: "${SERVER_ADDR}"
cert_dir: "${INSTALL_DIR}/certs"
log_file: "/var/log/srehub-agent/agent.log"
log_max_size: 100
log_max_backups: 3
log_max_age: 30
log_level: "info"
EOF
    echo "  配置文件已生成: ${INSTALL_DIR}/agent.yaml"
fi

# 复制证书（如果存在）
CERT_COPIED=0
for f in ca.pem cert.pem key.pem; do
    if [ -f "${SCRIPT_DIR}/certs/${f}" ]; then
        cp "${SCRIPT_DIR}/certs/${f}" "${INSTALL_DIR}/certs/${f}"
        CERT_COPIED=$((CERT_COPIED + 1))
        echo "  复制证书: ${f}"
    fi
done

if [ ${CERT_COPIED} -eq 0 ]; then
    echo "警告: 未找到证书文件，请手动将证书放置到 ${INSTALL_DIR}/certs/ 目录"
    echo "  需要的文件: ca.pem, cert.pem, key.pem"
elif [ ${CERT_COPIED} -lt 3 ]; then
    echo "警告: 证书文件不完整（需要 ca.pem, cert.pem, key.pem）"
fi

# 设置证书权限
[ -f "${INSTALL_DIR}/certs/key.pem" ] && chmod 600 "${INSTALL_DIR}/certs/key.pem"

# 创建管理脚本
create_management_scripts() {
    echo "创建管理脚本..."

    # start.sh
    cat > "${INSTALL_DIR}/start.sh" << 'STARTEOF'
#!/bin/bash
INSTALL_DIR="$(cd "$(dirname "$0")" && pwd)"
PID_FILE="${INSTALL_DIR}/srehub-agent.pid"
LOG_FILE="${INSTALL_DIR}/srehub-agent.log"

if [ -f "${PID_FILE}" ]; then
    PID=$(cat "${PID_FILE}")
    if kill -0 "${PID}" 2>/dev/null; then
        echo "SREHub Agent 已在运行 (PID: ${PID})"
        exit 1
    else
        rm -f "${PID_FILE}"
    fi
fi

echo "启动 SREHub Agent..."
cd "${INSTALL_DIR}"
nohup "${INSTALL_DIR}/srehub-agent" run -c "${INSTALL_DIR}/agent.yaml" >> "${LOG_FILE}" 2>&1 &
echo $! > "${PID_FILE}"
echo "SREHub Agent 已启动 (PID: $(cat ${PID_FILE}))"
echo "日志文件: ${LOG_FILE}"
STARTEOF

    # stop.sh
    cat > "${INSTALL_DIR}/stop.sh" << 'STOPEOF'
#!/bin/bash
INSTALL_DIR="$(cd "$(dirname "$0")" && pwd)"
PID_FILE="${INSTALL_DIR}/srehub-agent.pid"

if [ ! -f "${PID_FILE}" ]; then
    echo "SREHub Agent 未运行"
    exit 1
fi

PID=$(cat "${PID_FILE}")
if ! kill -0 "${PID}" 2>/dev/null; then
    echo "SREHub Agent 未运行 (PID 文件已过期)"
    rm -f "${PID_FILE}"
    exit 1
fi

echo "停止 SREHub Agent (PID: ${PID})..."
kill "${PID}"

# 等待进程退出
for i in {1..10}; do
    if ! kill -0 "${PID}" 2>/dev/null; then
        rm -f "${PID_FILE}"
        echo "SREHub Agent 已停止"
        exit 0
    fi
    sleep 1
done

# 强制终止
echo "强制终止 SREHub Agent..."
kill -9 "${PID}" 2>/dev/null || true
rm -f "${PID_FILE}"
echo "SREHub Agent 已停止"
STOPEOF

    # restart.sh
    cat > "${INSTALL_DIR}/restart.sh" << 'RESTARTEOF'
#!/bin/bash
INSTALL_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "重启 SREHub Agent..."
"${INSTALL_DIR}/stop.sh"
sleep 2
"${INSTALL_DIR}/start.sh"
RESTARTEOF

    # status.sh
    cat > "${INSTALL_DIR}/status.sh" << 'STATUSEOF'
#!/bin/bash
INSTALL_DIR="$(cd "$(dirname "$0")" && pwd)"
PID_FILE="${INSTALL_DIR}/srehub-agent.pid"
LOG_FILE="${INSTALL_DIR}/srehub-agent.log"

if [ ! -f "${PID_FILE}" ]; then
    echo "SREHub Agent 未运行"
    exit 1
fi

PID=$(cat "${PID_FILE}")
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
else
    echo "SREHub Agent 未运行 (PID 文件已过期)"
    rm -f "${PID_FILE}"
    exit 1
fi
STATUSEOF

    chmod +x "${INSTALL_DIR}/start.sh"
    chmod +x "${INSTALL_DIR}/stop.sh"
    chmod +x "${INSTALL_DIR}/restart.sh"
    chmod +x "${INSTALL_DIR}/status.sh"

    echo "  管理脚本已创建:"
    echo "    ${INSTALL_DIR}/start.sh"
    echo "    ${INSTALL_DIR}/stop.sh"
    echo "    ${INSTALL_DIR}/restart.sh"
    echo "    ${INSTALL_DIR}/status.sh"
}

# 安装服务（根据操作系统和用户选择）
if [ "${USE_SYSTEMD}" = "yes" ] && [ "${OS}" = "linux" ] && [ -d /etc/systemd/system ]; then
    # Linux: 使用 systemd
    echo "安装 systemd 服务..."
    sed "s|/opt/srehub-agent|${INSTALL_DIR}|g" "${SCRIPT_DIR}/srehub-agent.service" \
        > /etc/systemd/system/srehub-agent.service
    systemctl daemon-reload
    systemctl enable srehub-agent
    echo "  systemd 服务已安装并设置为开机自启"
    echo ""
    echo "使用以下命令管理服务:"
    echo "  启动: systemctl start srehub-agent"
    echo "  停止: systemctl stop srehub-agent"
    echo "  重启: systemctl restart srehub-agent"
    echo "  状态: systemctl status srehub-agent"
    echo "  日志: journalctl -u srehub-agent -f"
else
    # 使用脚本管理
    if [ "${USE_SYSTEMD}" = "no" ]; then
        echo "使用脚本管理模式（已禁用 systemd）"
    elif [ "${OS}" = "darwin" ]; then
        echo "macOS 平台，使用脚本管理模式"
    else
        echo "systemd 不可用，使用脚本管理模式"
    fi

    create_management_scripts

    echo ""
    echo "使用以下命令管理服务:"
    echo "  启动: ${INSTALL_DIR}/start.sh"
    echo "  停止: ${INSTALL_DIR}/stop.sh"
    echo "  重启: ${INSTALL_DIR}/restart.sh"
    echo "  状态: ${INSTALL_DIR}/status.sh"
fi

echo ""
echo "========================================="
echo "  安装完成！"
echo "========================================="
echo "安装目录: ${INSTALL_DIR}"
echo "配置文件: ${INSTALL_DIR}/agent.yaml"
echo "证书目录: ${INSTALL_DIR}/certs"
echo "管理模式: $([ "${USE_SYSTEMD}" = "yes" ] && echo "systemd" || echo "脚本管理")"
echo ""

# 检查配置是否完整
if [ "${SERVER_ADDR}" = "your-server-ip:9090" ]; then
    echo "⚠️  请编辑配置文件设置正确的服务端地址:"
    echo "   vi ${INSTALL_DIR}/agent.yaml"
    echo ""
fi

if [ ${CERT_COPIED} -lt 3 ]; then
    echo "⚠️  请将证书文件放置到 ${INSTALL_DIR}/certs/ 目录"
    echo "   需要的文件: ca.pem, cert.pem, key.pem"
    echo ""
fi

if [ "${USE_SYSTEMD}" = "yes" ] && [ "${OS}" = "linux" ] && [ -d /etc/systemd/system ]; then
    echo "启动服务: systemctl start srehub-agent"
elif [ "${USE_SYSTEMD}" = "no" ] || [ "${OS}" = "darwin" ]; then
    echo "启动服务: ${INSTALL_DIR}/start.sh"
fi
INSTALL_EOF
chmod +x "${MULTI_ARCH_DIR}/install.sh"

# 打包多架构 tar.gz
echo ""
echo ">>> 打包多架构安装包..."
PACKAGE_NAME="srehub-agent-${VERSION}-multi-arch"
(cd "${BUILD_DIR}" && tar czf "${PACKAGE_NAME}.tar.gz" "${PACKAGE_NAME}")

# 复制到 data/agent-binaries 供服务端部署使用
DATA_DIR="${SCRIPT_DIR}/../data/agent-binaries"
mkdir -p "${DATA_DIR}"
cp "${BUILD_DIR}/${PACKAGE_NAME}.tar.gz" "${DATA_DIR}/"

SIZE=$(du -h "${BUILD_DIR}/${PACKAGE_NAME}.tar.gz" | cut -f1)
echo "  完成: dist/${PACKAGE_NAME}.tar.gz (${SIZE})"

echo ""
echo "========================================="
echo "  构建完成！输出目录: ${BUILD_DIR}"
echo "========================================="
echo "多架构安装包: ${PACKAGE_NAME}.tar.gz"
echo "包含平台:"
ls -1 "${MULTI_ARCH_DIR}/bin/" | sed 's/srehub-agent-/  - /'
echo ""
echo "已复制到: ${DATA_DIR}/"

