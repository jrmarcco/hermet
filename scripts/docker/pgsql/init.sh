#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 共享外部网络 ( 与 kafka / debezium / es 等其它中间件复用 )。
NETWORK_NAME="jrmarcco_net"

echo "🚀 PostgreSQL 分片集群初始化脚本"
echo ""

# 1. 外部网络 ( 不存在则创建 )
if ! docker network ls --format '{{.Name}}' | grep -q "^${NETWORK_NAME}$"; then
    echo "🌐 创建外部网络 ${NETWORK_NAME}..."
    docker network create "${NETWORK_NAME}"
else
    echo "ℹ️  外部网络 ${NETWORK_NAME} 已存在"
fi
echo ""

# 2. .env ( 不存在则生成一份可用的默认配置，密码随机生成 )
if [ ! -f "./.env" ]; then
    echo "📝 生成 .env..."

    PWD_0="$(openssl rand -hex 12)"
    PWD_1="$(openssl rand -hex 12)"

    cat > ./.env <<EOF
POSTGRES_0_PASSWORD=${PWD_0}
POSTGRES_1_PASSWORD=${PWD_1}
POSTGRES_IMAGE=postgres:18.6
POSTGRES_0_PORT=15432
POSTGRES_1_PORT=25432
EOF

    echo "✅ .env 已生成 ( 密码随机生成，请自行保管 )"
    echo "   ⚠ 密码需同步至: etc/config.yaml 的 db.sharding DSN 与 debezium/connectors/*.json"
else
    echo "ℹ️  .env 已存在，跳过生成"
fi
echo ""

# 3. 数据目录 ( bind mount 持久化 )
echo "📁 准备数据目录..."
for i in 0 1; do
    DATA_DIR="./data-$i"
    if [ -d "${DATA_DIR}" ]; then
        echo "  📁 数据目录 ${DATA_DIR} 已存在"
    else
        echo "  📁 创建数据目录 ${DATA_DIR}"
        mkdir -p "${DATA_DIR}"
    fi
done
echo "✅ 数据目录就绪"
echo ""

echo "🎉 初始化完成！"
echo "👉 启动: docker compose up -d"
echo "👉 建表: 见 scripts/sql/README.md 与 scripts/sql/exec_sql.sh"
