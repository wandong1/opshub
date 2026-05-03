#!/bin/bash

# RSA 密码加密功能测试脚本

echo "=========================================="
echo "RSA 密码加密功能测试"
echo "=========================================="
echo ""

# 服务地址
BASE_URL="http://localhost:9876"

echo "1. 测试获取 RSA 公钥接口..."
PUBLIC_KEY_RESPONSE=$(curl -s "${BASE_URL}/api/v1/public/rsa-public-key")

if echo "$PUBLIC_KEY_RESPONSE" | grep -q "publicKey"; then
    echo "✅ 公钥接口正常"
    echo "公钥内容（前 100 字符）:"
    echo "$PUBLIC_KEY_RESPONSE" | head -c 100
    echo "..."
else
    echo "❌ 公钥接口异常"
    echo "响应: $PUBLIC_KEY_RESPONSE"
    exit 1
fi

echo ""
echo ""
echo "2. 测试说明："
echo "   - 前端会在登录页面加载时自动获取公钥"
echo "   - 用户输入密码后，使用 JSEncrypt 加密"
echo "   - 加密后的密码通过 encryptedPassword 字段发送"
echo "   - 后端使用私钥解密后验证"
echo ""
echo "3. 手动测试步骤："
echo "   a. 启动服务: ./bin/opshub server -c config/config.yaml"
echo "   b. 访问登录页面: http://localhost:9876"
echo "   c. 打开浏览器开发者工具 -> Network"
echo "   d. 输入用户名密码并登录"
echo "   e. 查看 /api/v1/public/login 请求"
echo "   f. 确认请求体中包含 encryptedPassword 字段"
echo "   g. 确认 encryptedPassword 是 Base64 编码的密文"
echo ""
echo "4. 向后兼容性："
echo "   - 旧客户端仍可使用 password 字段（明文）"
echo "   - 新客户端使用 encryptedPassword 字段（密文）"
echo "   - 后端优先使用 encryptedPassword"
echo ""
echo "=========================================="
echo "测试完成"
echo "=========================================="
