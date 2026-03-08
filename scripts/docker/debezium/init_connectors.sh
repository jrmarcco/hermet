#!/usr/bin/env bash
set -euo pipefail

CONNECT_HOST="${CONNECT_HOST:-debezium-connect-1}"
CONNECT_PORT="${CONNECT_PORT:-8083}"
CONNECTORS_DIR="${CONNECTORS_DIR:-/kafka/connectors}"
MAX_RETRIES=30
RETRY_INTERVAL=10

echo "========================================"
echo "Debezium Connector Auto-Registration"
echo "========================================"
echo "Connect Host: ${CONNECT_HOST}:${CONNECT_PORT}"
echo "Connectors Dir: ${CONNECTORS_DIR}"
echo ""

# 等待 Kafka Connect 就绪。
echo "[1/3] Waiting for Kafka Connect to be ready..."
retry_count=0
while [ $retry_count -lt $MAX_RETRIES ]; do
    if curl -s -o /dev/null -w "%{http_code}" "http://${CONNECT_HOST}:${CONNECT_PORT}/" | grep -q "200"; then
        echo "✓ Kafka Connect is ready!"
        break
    fi
    retry_count=$((retry_count + 1))
    echo "  Attempt ${retry_count}/${MAX_RETRIES}: Kafka Connect not ready, waiting ${RETRY_INTERVAL}s..."
    sleep $RETRY_INTERVAL
done

if [ $retry_count -eq $MAX_RETRIES ]; then
    echo "✗ ERROR: Kafka Connect failed to start after ${MAX_RETRIES} attempts"
    exit 1
fi

# 获取已注册的 connector 列表。
echo ""
echo "[2/3] Fetching registered connectors..."
registered_connectors=$(curl -s "http://${CONNECT_HOST}:${CONNECT_PORT}/connectors" | jq -r '.[]' 2>/dev/null || echo "")
if [ -z "$registered_connectors" ]; then
    echo "  No connectors currently registered"
else
    echo "  Registered connectors:"
    echo "$registered_connectors" | sed 's/^/    - /'
fi

# 遍历 connectors 目录下的所有 JSON 文件。
echo ""
echo "[3/3] Processing connector configurations..."
connector_count=0
skipped_count=0
registered_count=0
failed_count=0

if [ ! -d "$CONNECTORS_DIR" ]; then
    echo "✗ ERROR: Connectors directory not found: ${CONNECTORS_DIR}"
    exit 1
fi

for config_file in "${CONNECTORS_DIR}"/*.json; do
    if [ ! -f "$config_file" ]; then
        echo "  No JSON files found in ${CONNECTORS_DIR}"
        break
    fi

    connector_count=$((connector_count + 1))
    filename=$(basename "$config_file")

    # 从 JSON 文件中提取 connector name。
    connector_name=$(jq -r '.name' "$config_file" 2>/dev/null)

    if [ -z "$connector_name" ] || [ "$connector_name" = "null" ]; then
        echo "  ⚠ WARN: ${filename} - Invalid JSON or missing 'name' field, skipping"
        failed_count=$((failed_count + 1))
        continue
    fi

    # 检查 connector 是否已注册。
    if echo "$registered_connectors" | grep -q "^${connector_name}$"; then
        echo "  ⊙ SKIP: ${filename} (${connector_name}) - Already registered"
        skipped_count=$((skipped_count + 1))
        continue
    fi

    # 注册 connector。
    echo "  ➜ Registering: ${filename} (${connector_name})..."
    response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        --data @"$config_file" \
        "http://${CONNECT_HOST}:${CONNECT_PORT}/connectors")

    http_code=$(echo "$response" | tail -n1)
    response_body=$(echo "$response" | sed '$d')

    if [ "$http_code" -eq 201 ] || [ "$http_code" -eq 200 ]; then
        echo "    ✓ SUCCESS: ${connector_name} registered successfully"
        registered_count=$((registered_count + 1))
    else
        echo "    ✗ FAILED: ${connector_name} registration failed (HTTP ${http_code})"
        echo "    Response: ${response_body}"
        failed_count=$((failed_count + 1))
    fi
done

# 输出统计信息。
echo ""
echo "========================================"
echo "Registration Summary"
echo "========================================"
echo "Total files found:        ${connector_count}"
echo "Successfully registered:  ${registered_count}"
echo "Skipped (existing):       ${skipped_count}"
echo "Failed:                   ${failed_count}"
echo "========================================"

if [ $failed_count -gt 0 ]; then
    echo "⚠ Warning: Some connectors failed to register"
    exit 0  # 不中断其他服务启动。
else
    echo "✓ All connectors processed successfully"
fi
