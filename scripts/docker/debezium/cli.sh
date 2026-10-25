#!/usr/bin/env bash
set -euo pipefail

# Debezium Connector 管理脚本。
# 用于便捷地注册、更新、查看和删除 Debezium 连接器。

# 检测终端是否支持颜色。
if [ -t 1 ] && [ "$TERM" != "dumb" ] && command -v tput &> /dev/null && tput setaf 1 &> /dev/null; then
    # 终端支持颜色
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    # 终端不支持颜色，使用空字符串。
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

# 默认配置。
DEFAULT_HOST="127.0.0.1"
DEFAULT_PORT="8083"

# Kafka TLS 密码: 渲染 connector JSON 中 __KAFKA_TLS_KEY_PASSWORD__ 占位符
# ( schema history 客户端不继承 worker TLS 配置, 密码需写进 connector 配置 )。
# 优先取环境变量, 其次脚本同目录 .env 的 KAFKA_TLS_KEY_PASSWORD。
CLI_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -z "${KAFKA_TLS_KEY_PASSWORD:-}" ] && [ -f "${CLI_SCRIPT_DIR}/.env" ]; then
    KAFKA_TLS_KEY_PASSWORD="$(grep -E '^KAFKA_TLS_KEY_PASSWORD=' "${CLI_SCRIPT_DIR}/.env" | tail -1 | cut -d= -f2-)"
fi

# render_json <in_file> <out_file>: 原地渲染占位符到 out_file。
render_json() {
    local in_file="$1" out_file="$2"
    if [ -z "${KAFKA_TLS_KEY_PASSWORD:-}" ]; then
        print_warning "KAFKA_TLS_KEY_PASSWORD 未设置 ( 环境变量或 .env ), __KAFKA_TLS_KEY_PASSWORD__ 将按原样提交, schema history TLS 会握手失败。"
    fi
    sed "s|__KAFKA_TLS_KEY_PASSWORD__|${KAFKA_TLS_KEY_PASSWORD:-__KAFKA_TLS_KEY_PASSWORD__}|g" "$in_file" > "$out_file"
}

# 打印帮助信息。
print_help() {
    printf "${BLUE}Debezium Connector 管理脚本${NC}\n\n"
    printf "用法: %s [选项] <操作> [参数]\n\n" "$0"

    printf "${YELLOW}选项:${NC}\n"
    printf "    -h, --host <host>       Debezium Connect 服务地址 (默认: %s)\n" "${DEFAULT_HOST}"
    printf "    -p, --port <port>       Debezium Connect 服务端口 (默认: %s)\n" "${DEFAULT_PORT}"
    printf "    --help                  显示此帮助信息\n\n"

    printf "${YELLOW}操作:${NC}\n"
    printf "    register <json_file>    注册新的连接器\n"
    printf "                            示例: %s register ./connectors/pgsql-connector.json\n\n" "$0"

    printf "    update <json_file>      更新现有连接器配置\n"
    printf "                            示例: %s update ./connectors/pgsql-connector.json\n\n" "$0"

    printf "    list                    列出所有连接器\n"
    printf "                            示例: %s list\n\n" "$0"

    printf "    status <connector_name> 查看指定连接器的状态\n"
    printf "                            示例: %s status pgsql-connector\n\n" "$0"

    printf "    config <connector_name> 查看指定连接器的配置\n"
    printf "                            示例: %s config pgsql-connector\n\n" "$0"

    printf "    delete <connector_name> 删除指定连接器\n"
    printf "                            示例: %s delete pgsql-connector\n\n" "$0"

    printf "    restart <connector_name> 重启指定连接器\n"
    printf "                            示例: %s restart pgsql-connector\n\n" "$0"

    printf "    pause <connector_name>  暂停指定连接器\n"
    printf "                            示例: %s pause pgsql-connector\n\n" "$0"

    printf "    resume <connector_name> 恢复指定连接器\n"
    printf "                            示例: %s resume pgsql-connector\n\n" "$0"

    printf "${YELLOW}示例:${NC}\n"
    printf "    # 使用默认地址注册连接器\n"
    printf "    %s register ./connectors/pgsql-connector.json\n\n" "$0"

    printf "    # 指定服务地址和端口\n"
    printf "    %s -h 192.168.1.100 -p 8083 list\n\n" "$0"

    printf "    # 查看连接器状态\n"
    printf "    %s status pgsql-connector\n\n" "$0"

    printf "    # 删除连接器\n"
    printf "    %s delete pgsql-connector\n\n" "$0"
}

# 打印错误信息。
print_error() {
    printf "${RED}[错误]${NC} %s\n" "$1" >&2
}

# 打印成功信息。
print_success() {
    printf "${GREEN}[成功]${NC} %s\n" "$1"
}

# 打印信息。
print_info() {
    printf "${BLUE}[信息]${NC} %s\n" "$1"
}

# 打印警告信息。
print_warning() {
    printf "${YELLOW}[警告]${NC} %s\n" "$1"
}

# 检查 curl 是否安装。
check_curl() {
    if ! command -v curl &> /dev/null; then
        print_error "curl 未安装，请先安装 curl。"
        exit 1
    fi
}

# 检查 jq 是否安装 ( 用于格式化 JSON 输出 )。
check_jq() {
    if ! command -v jq &> /dev/null; then
        print_warning "jq 未安装 JSON 输出将不会格式化。建议安装 jq 以获得更好的显示效果。"
        return 1
    fi
    return 0
}

# 检查服务是否可用。
check_service() {
    local url="$1"
    if ! curl -s -f "${url}/" > /dev/null; then
        print_error "无法连接到 Debezium Connect 服务: ${url}"
        print_info "请检查服务是否正在运行，以及地址和端口是否正确。"
        exit 1
    fi
}

# 格式化 JSON 输出。
format_json() {
    if check_jq; then
        jq '.'
    else
        cat
    fi
}

# 注册连接器。
register_connector() {
    local base_url="$1"
    local json_file="$2"

    if [ -z "$json_file" ]; then
        print_error "请指定连接器配置文件。"
        echo "用法: $0 register <json_file>"
        exit 1
    fi

    if [ ! -f "$json_file" ]; then
        print_error "配置文件不存在: $json_file。"
        exit 1
    fi

    # 从 JSON 文件中提取连接器名称
    local connector_name
    if check_jq; then
        connector_name=$(jq -r '.name' "$json_file")
    else
        # 使用 grep 和 cut 提取名称，兼容 Linux 和 macOS
        connector_name=$(grep -o '"name"[[:space:]]*:[[:space:]]*"[^"]*"' "$json_file" | cut -d'"' -f4)
    fi

    print_info "正在注册连接器: $connector_name。"
    print_info "配置文件: $json_file"

    # 渲染 __KAFKA_TLS_KEY_PASSWORD__ 占位符后提交。
    local rendered_json
    rendered_json="$(mktemp)"
    render_json "$json_file" "$rendered_json"

    response=$(curl -s -w "\n%{http_code}" -X POST "${base_url}/connectors" \
        -H "Content-Type: application/json" \
        -d @"$rendered_json")
    rm -f "$rendered_json"

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" -eq 201 ] || [ "$http_code" -eq 200 ]; then
        print_success "连接器注册成功: $connector_name"
        echo "$body" | format_json
    else
        print_error "连接器注册失败 (HTTP $http_code)"
        echo "$body" | format_json
        exit 1
    fi
}

# 更新连接器。
update_connector() {
    local base_url="$1"
    local json_file="$2"

    if [ -z "$json_file" ]; then
        print_error "请指定连接器配置文件。"
        echo "用法: $0 update <json_file>"
        exit 1
    fi

    if [ ! -f "$json_file" ]; then
        print_error "配置文件 [ $json_file ] 不存在。"
        exit 1
    fi

    # 从 JSON 文件中提取连接器名称和配置 ( 渲染 __KAFKA_TLS_KEY_PASSWORD__ 占位符后提取 )。
    local connector_name
    local config
    local rendered_json
    rendered_json="$(mktemp)"
    render_json "$json_file" "$rendered_json"
    if check_jq; then
        connector_name=$(jq -r '.name' "$rendered_json")
        config=$(jq '.config' "$rendered_json")
    else
        # 使用 grep 和 cut 提取名称，兼容 Linux 和 macOS
        connector_name=$(grep -o '"name"[[:space:]]*:[[:space:]]*"[^"]*"' "$rendered_json" | cut -d'"' -f4)
        config=$(cat "$rendered_json")
    fi
    rm -f "$rendered_json"

    print_info "正在更新连接器: $connector_name。"

    response=$(curl -s -w "\n%{http_code}" -X PUT "${base_url}/connectors/${connector_name}/config" \
        -H "Content-Type: application/json" \
        -d "$config")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" -eq 200 ] || [ "$http_code" -eq 201 ]; then
        print_success "连接器更新成功: $connector_name。"
        echo "$body" | format_json
    else
        print_error "连接器更新失败 (HTTP $http_code)。"
        echo "$body" | format_json
        exit 1
    fi
}

# 列出所有连接器。
list_connectors() {
    local base_url="$1"

    print_info "正在获取连接器列表..."

    response=$(curl -s -w "\n%{http_code}" -X GET "${base_url}/connectors")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" -eq 200 ]; then
        if check_jq; then
            local count=$(echo "$body" | jq '. | length')
            print_success "找到 $count 个连接器:"
            echo "$body" | jq -r '.[]' | while read -r connector; do
                echo "  - $connector"
            done
        else
            print_success "连接器列表:"
            echo "$body"
        fi
    else
        print_error "获取连接器列表失败 (HTTP $http_code)"
        echo "$body" | format_json
        exit 1
    fi
}

# 查看连接器状态。
get_connector_status() {
    local base_url="$1"
    local connector_name="$2"

    if [ -z "$connector_name" ]; then
        print_error "请指定连接器名称。"
        echo "用法: $0 status <connector_name>"
        exit 1
    fi

    print_info "正在获取连接器状态: $connector_name。"

    response=$(curl -s -w "\n%{http_code}" -X GET "${base_url}/connectors/${connector_name}/status")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" -eq 200 ]; then
        print_success "连接器状态:。"
        echo "$body" | format_json
    else
        print_error "获取连接器状态失败 (HTTP $http_code)。"
        echo "$body" | format_json
        exit 1
    fi
}

# 查看连接器配置。
get_connector_config() {
    local base_url="$1"
    local connector_name="$2"

    if [ -z "$connector_name" ]; then
        print_error "请指定连接器名称。"
        echo "用法: $0 config <connector_name>"
        exit 1
    fi

    print_info "正在获取连接器配置: $connector_name。"

    response=$(curl -s -w "\n%{http_code}" -X GET "${base_url}/connectors/${connector_name}/config")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" -eq 200 ]; then
        print_success "连接器配置:"
        echo "$body" | format_json
    else
        print_error "获取连接器配置失败 (HTTP $http_code)。"
        echo "$body" | format_json
        exit 1
    fi
}

# 删除连接器。
delete_connector() {
    local base_url="$1"
    local connector_name="$2"

    if [ -z "$connector_name" ]; then
        print_error "请指定连接器名称。"
        echo "用法: $0 delete <connector_name>"
        exit 1
    fi

    print_warning "即将删除连接器: $connector_name。"
    read -p "确认删除? (y/N): " -n 1 -r
    echo

    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "已取消删除操作。"
        exit 0
    fi

    print_info "正在删除连接器: $connector_name。"

    response=$(curl -s -w "\n%{http_code}" -X DELETE "${base_url}/connectors/${connector_name}")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" -eq 204 ] || [ "$http_code" -eq 200 ]; then
        print_success "连接器删除成功: $connector_name。"
    else
        print_error "连接器删除失败 (HTTP $http_code)。"
        if [ -n "$body" ]; then
            echo "$body" | format_json
        fi
        exit 1
    fi
}

# 重启连接器。
restart_connector() {
    local base_url="$1"
    local connector_name="$2"

    if [ -z "$connector_name" ]; then
        print_error "请指定连接器名称。"
        echo "用法: $0 restart <connector_name>"
        exit 1
    fi

    print_info "正在重启连接器: $connector_name。"

    response=$(curl -s -w "\n%{http_code}" -X POST "${base_url}/connectors/${connector_name}/restart")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" -eq 204 ] || [ "$http_code" -eq 200 ]; then
        print_success "连接器重启成功: $connector_name。"
    else
        print_error "连接器重启失败 (HTTP $http_code)。"
        if [ -n "$body" ]; then
            echo "$body" | format_json
        fi
        exit 1
    fi
}

# 暂停连接器。
pause_connector() {
    local base_url="$1"
    local connector_name="$2"

    if [ -z "$connector_name" ]; then
        print_error "请指定连接器名称。"
        echo "用法: $0 pause <connector_name>"
        exit 1
    fi

    print_info "正在暂停连接器: $connector_name。"

    response=$(curl -s -w "\n%{http_code}" -X PUT "${base_url}/connectors/${connector_name}/pause")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" -eq 202 ] || [ "$http_code" -eq 200 ]; then
        print_success "连接器暂停成功: $connector_name。"
    else
        print_error "连接器暂停失败 (HTTP $http_code)。"
        if [ -n "$body" ]; then
            echo "$body" | format_json
        fi
        exit 1
    fi
}

# 恢复连接器。
resume_connector() {
    local base_url="$1"
    local connector_name="$2"

    if [ -z "$connector_name" ]; then
        print_error "请指定连接器名称。"
        echo "用法: $0 resume <connector_name>"
        exit 1
    fi

    print_info "正在恢复连接器: $connector_name。"

    response=$(curl -s -w "\n%{http_code}" -X PUT "${base_url}/connectors/${connector_name}/resume")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)

    if [ "$http_code" -eq 202 ] || [ "$http_code" -eq 200 ]; then
        print_success "连接器恢复成功: $connector_name。"
    else
        print_error "连接器恢复失败 (HTTP $http_code)。"
        if [ -n "$body" ]; then
            echo "$body" | format_json
        fi
        exit 1
    fi
}

# 主函数。
main() {
    # 检查必要的工具。
    check_curl

    # 解析参数。
    local host="$DEFAULT_HOST"
    local port="$DEFAULT_PORT"
    local operation=""
    local args=()

    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--host)
                host="$2"
                shift 2
                ;;
            -p|--port)
                port="$2"
                shift 2
                ;;
            --help)
                print_help
                exit 0
                ;;
            register|update|list|status|config|delete|restart|pause|resume)
                operation="$1"
                shift
                args=("$@")
                break
                ;;
            *)
                print_error "未知选项: $1"
                echo "使用 --help 查看帮助信息"
                exit 1
                ;;
        esac
    done

    # 检查是否指定了操作。
    if [ -z "$operation" ]; then
        print_error "请指定操作"
        echo "使用 --help 查看帮助信息"
        exit 1
    fi

    # 构建基础 URL。
    local base_url="http://${host}:${port}"

    # 检查服务是否可用。
    check_service "$base_url"

    # 执行操作。
    case $operation in
        register)
            register_connector "$base_url" "${args[0]}"
            ;;
        update)
            update_connector "$base_url" "${args[0]}"
            ;;
        list)
            list_connectors "$base_url"
            ;;
        status)
            get_connector_status "$base_url" "${args[0]}"
            ;;
        config)
            get_connector_config "$base_url" "${args[0]}"
            ;;
        delete)
            delete_connector "$base_url" "${args[0]}"
            ;;
        restart)
            restart_connector "$base_url" "${args[0]}"
            ;;
        pause)
            pause_connector "$base_url" "${args[0]}"
            ;;
        resume)
            resume_connector "$base_url" "${args[0]}"
            ;;
        *)
            print_error "未知操作: $operation"
            exit 1
            ;;
    esac
}

# 运行主函数。
main "$@"
