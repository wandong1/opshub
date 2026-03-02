#!/bin/bash
# SREHub Agent 构建脚本
# 用法: ./build.sh [linux/amd64] [linux/arm64]
# 默认构建 linux/amd64

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION="${VERSION:-1.0.0}"
BUILD_DIR="${SCRIPT_DIR}/dist"
AGENT_DIR="${SCRIPT_DIR}"

# 默认目标平台
TARGETS="${@:-linux/amd64}"

echo "========================================="
echo "  SREHub Agent 构建工具 v${VERSION}"
echo "========================================="

# 清理旧构建
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

for TARGET in ${TARGETS}; do
    OS=$(echo "${TARGET}" | cut -d'/' -f1)
    ARCH=$(echo "${TARGET}" | cut -d'/' -f2)
    BINARY_NAME="srehub-agent"
    PACKAGE_NAME="srehub-agent-${VERSION}-${OS}-${ARCH}"
    STAGING_DIR="${BUILD_DIR}/${PACKAGE_NAME}"

    echo ""
    echo ">>> 构建 ${OS}/${ARCH} ..."

    # 创建临时打包目录
    mkdir -p "${STAGING_DIR}/certs"

    # 编译二进制
    echo "  编译二进制..."
    CGO_ENABLED=0 GOOS="${OS}" GOARCH="${ARCH}" \
        go build -ldflags "-s -w -X main.Version=${VERSION}" \
        -o "${STAGING_DIR}/${BINARY_NAME}" \
        "${AGENT_DIR}/cmd/"

    # 复制配置模板
    cp "${AGENT_DIR}/config/agent.yaml" "${STAGING_DIR}/agent.yaml"
    cp "${AGENT_DIR}/config/srehub-agent.service" "${STAGING_DIR}/srehub-agent.service"

    # 创建安装脚本
    cat > "${STAGING_DIR}/install.sh" << 'INSTALL_EOF'
#!/bin/bash
# SREHub Agent 安装脚本
set -e

INSTALL_DIR="${INSTALL_DIR:-/opt/srehub-agent}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "安装 SREHub Agent 到 ${INSTALL_DIR} ..."

# 创建目录
mkdir -p "${INSTALL_DIR}/certs"

# 复制文件
cp "${SCRIPT_DIR}/srehub-agent" "${INSTALL_DIR}/srehub-agent"
chmod +x "${INSTALL_DIR}/srehub-agent"

# 如果存在配置文件则复制（不覆盖已有配置）
[ ! -f "${INSTALL_DIR}/agent.yaml" ] && cp "${SCRIPT_DIR}/agent.yaml" "${INSTALL_DIR}/agent.yaml"

# 复制证书（如果存在）
for f in ca.pem cert.pem key.pem; do
    [ -f "${SCRIPT_DIR}/certs/${f}" ] && cp "${SCRIPT_DIR}/certs/${f}" "${INSTALL_DIR}/certs/${f}"
done
[ -f "${INSTALL_DIR}/certs/key.pem" ] && chmod 600 "${INSTALL_DIR}/certs/key.pem"

# 安装 systemd 服务
if [ -d /etc/systemd/system ]; then
    sed "s|/opt/srehub-agent|${INSTALL_DIR}|g" "${SCRIPT_DIR}/srehub-agent.service" \
        > /etc/systemd/system/srehub-agent.service
    systemctl daemon-reload
    systemctl enable srehub-agent
    echo "systemd 服务已安装，使用 'systemctl start srehub-agent' 启动"
fi

echo "安装完成！"
INSTALL_EOF
    chmod +x "${STAGING_DIR}/install.sh"

    # 打包 tar.gz
    echo "  打包 ${PACKAGE_NAME}.tar.gz ..."
    (cd "${BUILD_DIR}" && tar czf "${PACKAGE_NAME}.tar.gz" "${PACKAGE_NAME}")

    # 同时复制到 data/agent-binaries 供服务端部署使用
    DATA_DIR="${SCRIPT_DIR}/../data/agent-binaries"
    mkdir -p "${DATA_DIR}"
    cp "${BUILD_DIR}/${PACKAGE_NAME}.tar.gz" "${DATA_DIR}/"
    # 保留一份裸二进制供兼容
    cp "${STAGING_DIR}/${BINARY_NAME}" "${DATA_DIR}/${BINARY_NAME}"

    SIZE=$(du -h "${BUILD_DIR}/${PACKAGE_NAME}.tar.gz" | cut -f1)
    echo "  完成: dist/${PACKAGE_NAME}.tar.gz (${SIZE})"
done

echo ""
echo "========================================="
echo "  构建完成！输出目录: ${BUILD_DIR}"
echo "========================================="
ls -lh "${BUILD_DIR}"/*.tar.gz
