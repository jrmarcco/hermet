#!/bin/bash

# MongoDB 客户端证书生成工具

KEY_SIZE=2048

if [ -z "$1" ]; then
    echo "用法: $0 <客户端名称> [有效期天数]"
    echo ""
    echo "示例:"
    echo "  $0 myapp-client          # 生成 myapp-client 证书 ( 默认10年 )"
    echo "  $0 monitoring-client 365 # 生成 monitoring-client 证书 ( 1年有效期 )"
    echo ""
    echo "已存在的客户端证书:"
    ls certs/*-client.pem 2>/dev/null | while read cert; do
        CERT_NAME=$(basename "$cert" .pem)
        EXPIRY=$(openssl x509 -in "$cert" -noout -enddate 2>/dev/null | cut -d= -f2)
        echo "  • $CERT_NAME (有效至: $EXPIRY)"
    done
    exit 1
fi

CLIENT_NAME=$1
DAYS=${2:-3650}  # 默认 10 年

cd certs || exit 1

echo "=========================================="
echo "生成客户端证书: $CLIENT_NAME"
echo "有效期: $DAYS 天"
echo "=========================================="
echo ""

# 检查证书是否已存在
if [ -f "${CLIENT_NAME}.pem" ]; then
    echo "⚠️  警告：${CLIENT_NAME}.pem 已存在"
    read -p "是否覆盖? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "已取消"
        exit 0
    fi
fi

# 1. 生成私钥
echo "1. 生成私钥..."
openssl genrsa -out "${CLIENT_NAME}.key" ${KEY_SIZE}
echo "   ✓ 私钥生成完成"

# 2. 创建证书配置
echo "2. 创建证书配置..."
cat > "${CLIENT_NAME}.cnf" << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
CN = ${CLIENT_NAME}
O = MongoDB-Client
C = CN

[v3_req]
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${CLIENT_NAME}
DNS.2 = localhost
EOF
echo "   ✓ 配置文件创建完成"

# 3. 生成 CSR
echo "3. 生成证书请求..."
openssl req -new -key "${CLIENT_NAME}.key" \
  -out "${CLIENT_NAME}.csr" \
  -config "${CLIENT_NAME}.cnf" 2>/dev/null
echo "   ✓ CSR 生成完成"

# 4. 签名证书
echo "4. 签名证书..."
openssl x509 -req -in "${CLIENT_NAME}.csr" \
  -CA ca.pem -CAkey ca.key -CAcreateserial \
  -out "${CLIENT_NAME}.crt" \
  -days $DAYS -sha256 \
  -extensions v3_req \
  -extfile "${CLIENT_NAME}.cnf" 2>/dev/null
echo "   ✓ 证书签名完成"

# 5. 合并为 PEM
echo "5. 合并证书和私钥..."
cat "${CLIENT_NAME}.key" "${CLIENT_NAME}.crt" > "${CLIENT_NAME}.pem"
chmod 644 "${CLIENT_NAME}.pem"
echo "   ✓ PEM 文件生成完成"

cd ..

echo ""
echo "=========================================="
echo "✅ 客户端证书生成成功！"
echo "=========================================="
echo ""
echo "证书文件: certs/${CLIENT_NAME}.pem"
echo "证书信息:"
openssl x509 -in "certs/${CLIENT_NAME}.pem" -noout -subject -enddate
echo ""
echo "使用此证书连接："
echo "  docker exec -it mongodb-mongos mongosh --port 27017 \\"
echo "    -u <username> -p <password> \\"
echo "    --authenticationDatabase admin \\"
echo "    --tls \\"
echo "    --tlsCertificateKeyFile /path/to/${CLIENT_NAME}.pem \\"
echo "    --tlsCAFile /path/to/ca.pem"
echo ""
echo "从主机连接（需要先复制证书）："
echo "  cp certs/${CLIENT_NAME}.pem /path/to/"
echo "  cp certs/ca.pem /path/to/"
echo "  mongosh \"mongodb://username:password@localhost:27017/admin\" \\"
echo "    --tls \\"
echo "    --tlsCertificateKeyFile /path/to/${CLIENT_NAME}.pem \\"
echo "    --tlsCAFile /path/to/ca.pem"
echo ""
echo "提示: 记得为此证书创建对应的 MongoDB 用户"
echo ""
