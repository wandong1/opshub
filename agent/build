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

# 创建智能安装脚本（自动检测架构）
cat > "${MULTI_ARCH_DIR}/install.sh" << 'INSTALL_EOF'
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

