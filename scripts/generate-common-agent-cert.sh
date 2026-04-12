#!/bin/bash
# 生成通用 Agent 客户端证书（用于手动安装）
# 有效期：10 年

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CERT_DIR="${PROJECT_ROOT}/data/agent-certs"
COMMON_CERT_DIR="${CERT_DIR}/common"

echo "========================================="
echo "  生成通用 Agent 客户端证书"
echo "========================================="

# 检查 CA 证书是否存在
if [ ! -f "${CERT_DIR}/ca.pem" ] || [ ! -f "${CERT_DIR}/ca-key.pem" ]; then
    echo "错误: CA 证书不存在，请先启动服务端生成 CA"
    echo "路径: ${CERT_DIR}/ca.pem"
    exit 1
fi

# 创建通用证书目录
mkdir -p "${COMMON_CERT_DIR}"

# 检查是否已存在通用证书
if [ -f "${COMMON_CERT_DIR}/cert.pem" ] && [ -f "${COMMON_CERT_DIR}/key.pem" ]; then
    echo "通用证书已存在，是否重新生成？(y/N)"
    read -r answer
    if [ "$answer" != "y" ] && [ "$answer" != "Y" ]; then
        echo "跳过生成"
        exit 0
    fi
fi

echo "生成通用 Agent 客户端证书..."

# 生成私钥
openssl ecparam -genkey -name prime256v1 -out "${COMMON_CERT_DIR}/key.pem"

# 创建证书签名请求配置
cat > "${COMMON_CERT_DIR}/cert.conf" << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
O = OpsHub Agent
CN = common-agent

[v3_req]
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
EOF

# 生成 CSR
openssl req -new -key "${COMMON_CERT_DIR}/key.pem" \
    -out "${COMMON_CERT_DIR}/cert.csr" \
    -config "${COMMON_CERT_DIR}/cert.conf"

# 使用 CA 签发证书（有效期 10 年）
openssl x509 -req -in "${COMMON_CERT_DIR}/cert.csr" \
    -CA "${CERT_DIR}/ca.pem" \
    -CAkey "${CERT_DIR}/ca-key.pem" \
    -CAcreateserial \
    -out "${COMMON_CERT_DIR}/cert.pem" \
    -days 3650 \
    -extensions v3_req \
    -extfile "${COMMON_CERT_DIR}/cert.conf"

# 设置权限
chmod 600 "${COMMON_CERT_DIR}/key.pem"
chmod 644 "${COMMON_CERT_DIR}/cert.pem"

# 清理临时文件
rm -f "${COMMON_CERT_DIR}/cert.csr" "${COMMON_CERT_DIR}/cert.conf"

echo ""
echo "========================================="
echo "  通用证书生成完成！"
echo "========================================="
echo "证书目录: ${COMMON_CERT_DIR}"
echo "文件列表:"
echo "  - cert.pem (客户端证书)"
echo "  - key.pem  (私钥)"
echo ""
echo "证书信息:"
openssl x509 -in "${COMMON_CERT_DIR}/cert.pem" -noout -subject -dates
echo ""
echo "该证书将在构建 Agent 安装包时自动打包"
