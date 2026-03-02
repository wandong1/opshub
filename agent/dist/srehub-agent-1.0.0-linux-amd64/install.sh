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
