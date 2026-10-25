#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 共享外部网络 ( 与 pgsql / kafka / es 等其它中间件复用 )。
NETWORK_NAME="jrmarcco_net"

# Kafka 客户端证书目录 ( 必须手动指定, 不做探测 )。
#
# 目录内需包含以下文件 ( 由 gateway 仓库 scripts/docker/kafka 的 certs_gen.sh 生成 ):
#   - kafka.client.keystore.jks     客户端密钥库 ( connect 的 worker/producer/consumer/
#                                   schema history 连 kafka 的 mTLS 身份 )
#   - kafka.client.truststore.jks   客户端信任库 ( 校验 kafka broker 服务端证书 )
# 缺任一文件本脚本会直接报错退出。
#
# 示例:
#   KAFKA_CERTS_DIR=~/Projects/github.com/synp/gateway/scripts/docker/kafka/certs ./init.sh
if [ -z "${KAFKA_CERTS_DIR:-}" ]; then
    echo "✗ 必须设置 KAFKA_CERTS_DIR ( 指向含 kafka.client.keystore.jks / kafka.client.truststore.jks 的证书目录 )"
    echo "  通常是 gateway 仓库 kafka 部署的 certs 目录:"
    echo "  KAFKA_CERTS_DIR=/path/to/gateway/scripts/docker/kafka/certs ./init.sh"
    exit 1
fi

echo "🚀 Debezium Connect 集群初始化脚本"
echo ""

# 1. 外部网络 ( 不存在则创建 )
if ! docker network ls --format '{{.Name}}' | grep -q "^${NETWORK_NAME}$"; then
    echo "🌐 创建外部网络 ${NETWORK_NAME}..."
    docker network create "${NETWORK_NAME}"
else
    echo "ℹ️  外部网络 ${NETWORK_NAME} 已存在"
fi
echo ""

# 2. .env ( 不存在则从模板生成 )
if [ ! -f "./.env" ]; then
    echo "📝 生成 .env..."
    cp ./.env.example ./.env
    echo "✅ .env 已生成 ( 默认 KAFKA_TLS_KEY_PASSWORD=kafka-secret，与 gateway kafka 部署保持一致 )"
    echo "   ⚠ 若 kafka 侧密码已修改，请同步修改这里"
else
    echo "ℹ️  .env 已存在，跳过生成"
fi
echo ""

# 3. Kafka 客户端证书 ( connect 以客户端身份连 kafka，需 mTLS keystore/truststore )
NEED_KEYSTORE="./certs/kafka.client.keystore.jks"
NEED_TRUSTSTORE="./certs/kafka.client.truststore.jks"
if [ ! -f "${NEED_KEYSTORE}" ] || [ ! -f "${NEED_TRUSTSTORE}" ]; then
    echo "🔐 准备 Kafka 客户端证书 ( 来源: ${KAFKA_CERTS_DIR} )..."

    if [ ! -d "${KAFKA_CERTS_DIR}" ]; then
        echo "✗ KAFKA_CERTS_DIR 指向的目录不存在: ${KAFKA_CERTS_DIR}"
        echo "  请检查路径 ( 目录内需有 kafka.client.keystore.jks / kafka.client.truststore.jks )"
        exit 1
    fi

    mkdir -p ./certs
    for cert_file in "${KAFKA_CERTS_DIR}"/kafka.client.*.jks; do
        if [ -f "${cert_file}" ]; then
            cp -v "${cert_file}" ./certs/
        fi
    done

    if [ ! -f "${NEED_KEYSTORE}" ] || [ ! -f "${NEED_TRUSTSTORE}" ]; then
        echo "✗ 证书目录中缺少 kafka.client.keystore.jks 或 kafka.client.truststore.jks"
        exit 1
    fi
    echo "✅ 客户端证书就绪"
else
    echo "ℹ️  客户端证书已存在，跳过复制"
fi
echo ""

# 4. connector 配置检查 ( 提醒同步数据库密码 )
for connector_file in ./connectors/*.json; do
    [ -f "${connector_file}" ] || continue
    if grep -q '"database.password": "<passwd>"' "${connector_file}"; then
        echo "⚠ ${connector_file} 中的 database.password 仍为占位符，请改为 pgsql 实际密码 ( 见 pgsql/.env )"
    fi
done
echo ""

echo "🎉 初始化完成！"
echo "👉 启动: docker compose up -d ( connector 自动注册 )"
echo "👉 管理: ./cli.sh list / ./cli.sh status <name>"
