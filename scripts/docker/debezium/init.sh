#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 共享外部网络 ( 与 pgsql / kafka / es 等其它中间件复用 )。
NETWORK_NAME="jrmarcco_net"

# Kafka 客户端证书来源 ( gateway 仓库的 kafka 部署目录 )。
# 可经 KAFKA_CERTS_DIR 显式指定; 默认探测兄弟仓库路径 ../gateway/scripts/docker/kafka/certs。
DEFAULT_KAFKA_CERTS_DIR="${SCRIPT_DIR}/../../../gateway/scripts/docker/kafka/certs"
KAFKA_CERTS_DIR="${KAFKA_CERTS_DIR:-${DEFAULT_KAFKA_CERTS_DIR}}"

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
        echo "✗ 未找到 Kafka 证书目录: ${KAFKA_CERTS_DIR}"
        echo "  请先在 gateway 仓库执行: cd gateway/scripts/docker/kafka && ./init.sh"
        echo "  或经环境变量指定: KAFKA_CERTS_DIR=/path/to/kafka/certs ./init.sh"
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
