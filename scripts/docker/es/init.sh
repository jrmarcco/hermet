#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 共享外部网络 ( 与 pgsql / kafka / debezium 等其它中间件复用 )。
NETWORK_NAME="jrmarcco_net"

echo "🚀 Elasticsearch 初始化脚本"
echo ""

# 1. 外部网络 ( 不存在则创建 )
if ! docker network ls --format '{{.Name}}' | grep -q "^${NETWORK_NAME}$"; then
    echo "🌐 创建外部网络 ${NETWORK_NAME}..."
    docker network create "${NETWORK_NAME}"
else
    echo "ℹ️  外部网络 ${NETWORK_NAME} 已存在"
fi
echo ""

# 2. .env ( 不存在则生成一份，密码随机生成 )
if [ ! -f "./.env" ]; then
    echo "📝 生成 .env..."

    ES_PASSWORD="$(openssl rand -hex 12)"

    cat > ./.env <<EOF
ELASTIC_PASSWORD=${ES_PASSWORD}
ELASTICSEARCH_IMAGE=docker.elastic.co/elasticsearch/elasticsearch:9.5.5
KIBANA_IMAGE=docker.elastic.co/kibana/kibana:9.5.5
ELASTICSEARCH_PORT=19200
KIBANA_PORT=15601
ES_JAVA_OPTS=-Xms1g -Xmx1g
EOF

    echo "✅ .env 已生成 ( 密码随机生成，请自行保管 )"
else
    echo "ℹ️  .env 已存在，跳过生成"
fi
echo ""

# 3. 数据目录 ( ES 镜像以 uid 1000 运行, bind mount 目录需修正属主, 否则启动即失败 )
echo "📁 准备数据目录 ( 属主修正为 1000:1000 )..."
DATA_DIR="./es-data"
[ -d "${DATA_DIR}" ] || mkdir -p "${DATA_DIR}"
if [ "$(id -u)" = "0" ]; then
    chown -R 1000:1000 "${DATA_DIR}"
else
    sudo chown -R 1000:1000 "${DATA_DIR}"
fi
echo "✅ 数据目录就绪"
echo ""

echo "🎉 初始化完成！"
echo "👉 启动:          docker compose up -d"
echo "👉 ( 可选 ) Kibana: docker compose --profile kibana up -d"
echo "👉 健康检查:       curl -u elastic:\$(grep ELASTIC_PASSWORD .env | cut -d= -f2) http://localhost:19200/_cluster/health"
