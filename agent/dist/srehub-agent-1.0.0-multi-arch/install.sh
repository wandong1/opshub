#!/bin/bash
# SREHub Agent 多架构安装脚本
set -e

INSTALL_DIR="${INSTALL_DIR:-/opt/srehub-agent}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "安装 SREHub Agent 到 ${INSTALL_DIR} ..."

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

PLATFORM=$(detect_platform)
OS=$(echo "${PLATFORM}" | cut -d'/' -f1)
ARCH=$(echo "${PLATFORM}" | cut -d'/' -f2)
BINARY_NAME="srehub-agent-${OS}-${ARCH}"

echo "检测到平台: ${PLATFORM}"

# 检查二进制文件是否存在
if [ ! -f "${SCRIPT_DIR}/bin/${BINARY_NAME}" ]; then
    echo "错误: 未找到适用于 ${PLATFORM} 的二进制文件"
    echo "可用的二进制文件:"
    ls -1 "${SCRIPT_DIR}/bin/"
    exit 1
fi

# 创建目录
mkdir -p "${INSTALL_DIR}/certs"

# 复制二进制文件
echo "安装二进制文件: ${BINARY_NAME}"
cp "${SCRIPT_DIR}/bin/${BINARY_NAME}" "${INSTALL_DIR}/srehub-agent"
chmod +x "${INSTALL_DIR}/srehub-agent"

# 如果存在配置文件则复制（不覆盖已有配置）
[ ! -f "${INSTALL_DIR}/agent.yaml" ] && cp "${SCRIPT_DIR}/agent.yaml" "${INSTALL_DIR}/agent.yaml"

# 复制证书（如果存在）
for f in ca.pem cert.pem key.pem; do
    [ -f "${SCRIPT_DIR}/certs/${f}" ] && cp "${SCRIPT_DIR}/certs/${f}" "${INSTALL_DIR}/certs/${f}"
done
[ -f "${INSTALL_DIR}/certs/key.pem" ] && chmod 600 "${INSTALL_DIR}/certs/key.pem"

# 安装服务（根据操作系统）
if [ "${OS}" = "linux" ]; then
    # Linux: 使用 systemd
    if [ -d /etc/systemd/system ]; then
        sed "s|/opt/srehub-agent|${INSTALL_DIR}|g" "${SCRIPT_DIR}/srehub-agent.service" \
            > /etc/systemd/system/srehub-agent.service
        systemctl daemon-reload
        systemctl enable srehub-agent
        echo "systemd 服务已安装，使用 'systemctl start srehub-agent' 启动"
    fi
elif [ "${OS}" = "darwin" ]; then
    # macOS: 使用 launchd
    echo "macOS 平台，请手动配置 launchd 或直接运行: ${INSTALL_DIR}/srehub-agent"
fi

echo "安装完成！"
