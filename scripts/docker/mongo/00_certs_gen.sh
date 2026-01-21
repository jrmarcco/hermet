#!/bin/bash

# 配置变量
KEY_SIZE=2048              # RSA 密钥长度
CERT_VALIDITY_DAYS=3650    # 证书有效期（天）

mkdir -p certs && cd certs

# 生成 CA 私钥和证书
echo ""
echo "1. 生成 CA 证书..."
if [ ! -f ca.key ]; then
  openssl genrsa -out ca.key ${KEY_SIZE}
  openssl req -x509 -new -nodes -key ca.key -sha256 -days ${CERT_VALIDITY_DAYS} \
    -out ca.pem \
    -subj "/CN=MongoDB-CA/O=MongoDB/C=CN"
  echo "✓ CA 证书生成完成"
else
  echo "✓ CA 证书已存在，跳过"
fi

# 生成服务器私钥
openssl genrsa -out mongo.key ${KEY_SIZE}

# 创建 SAN 配置文件
cat > san.cnf << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
CN = MongoDB
O = MongoDB
C = CN

[v3_req]
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth, clientAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost

DNS.2 = mongodb-config-server-1
DNS.3 = mongodb-config-server-2
DNS.4 = mongodb-config-server-3

DNS.5 = mongodb-shard1-a
DNS.6 = mongodb-shard1-b
DNS.7 = mongodb-shard1-c

DNS.8 = mongodb-shard2-a
DNS.9 = mongodb-shard2-b
DNS.10 = mongodb-shard2-c

DNS.11 = mongodb-shard3-a
DNS.12 = mongodb-shard3-b
DNS.13 = mongodb-shard3-c

DNS.14 = mongodb-mongos-1

# ===== 如果需要外部访问，添加主机名 或 IP =====
# DNS.xx = mongos.yourdomain.com
IP.1 = 127.0.0.1
IP.2 = 192.168.3.3
EOF

# 生成 CSR 和签名证书
echo ""
echo "2. 生成 MongoDB 证书..."
openssl req -new -key mongo.key \
  -out mongo.csr \
  -config san.cnf

openssl x509 -req -in mongo.csr \
  -CA ca.pem -CAkey ca.key -CAcreateserial \
  -out mongo.crt \
  -days ${CERT_VALIDITY_DAYS} -sha256 \
  -extensions v3_req \
  -extfile san.cnf

# 合并为 PEM 文件
echo ""
echo "3. 合并为 PEM 文件..."
cat mongo.key mongo.crt > mongo.pem
chmod 644 mongo.pem

# 生成 keyfile 用于集群内部认证
openssl rand -base64 756 > keyfile
chmod 400 keyfile

echo ""
echo "4. 生成客户端证书..."

# 生成客户端证书函数
generate_client_cert() {
    local CLIENT_NAME=$1

    echo "  → 生成 ${CLIENT_NAME} 证书..."

    # 生成客户端私钥
    openssl genrsa -out "${CLIENT_NAME}.key" ${KEY_SIZE}

    # 创建客户端证书配置
    cat > "${CLIENT_NAME}.cnf" << CLIENTEOF
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
CLIENTEOF

    # 生成 CSR
    openssl req -new -key "${CLIENT_NAME}.key" \
      -out "${CLIENT_NAME}.csr" \
      -config "${CLIENT_NAME}.cnf"

    # 签名证书
    openssl x509 -req -in "${CLIENT_NAME}.csr" \
      -CA ca.pem -CAkey ca.key -CAcreateserial \
      -out "${CLIENT_NAME}.crt" \
      -days ${CERT_VALIDITY_DAYS} -sha256 \
      -extensions v3_req \
      -extfile "${CLIENT_NAME}.cnf"

    # 合并为 PEM
    cat "${CLIENT_NAME}.key" "${CLIENT_NAME}.crt" > "${CLIENT_NAME}.pem"
    chmod 644 "${CLIENT_NAME}.pem"

    echo "  ✓ ${CLIENT_NAME} 证书生成完成"
}

# 生成默认客户端证书
generate_client_cert "admin-client"

echo ""
echo "=========================================="
echo "证书生成完成！"
echo "=========================================="
echo ""
echo "服务端证书: certs/mongo.pem (所有节点共享)"
echo "客户端证书:"
echo "  • certs/admin-client.pem (管理员)"
echo "CA 证书: certs/ca.pem"
echo "KeyFile: certs/keyfile"
echo ""
