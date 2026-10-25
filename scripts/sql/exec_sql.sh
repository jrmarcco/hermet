#!/usr/bin/env bash
set -euo pipefail

# PostgreSQL SQL 脚本批量执行工具 ( 仅 pgsql，MySQL 支持已移除 )。
#
# 用法: ./exec_sql.sh [选项]
#
# 选项:
#   -h, --host HOST           数据库主机 ( 默认: 127.0.0.1 )
#   -P, --port PORT           数据库端口 ( 默认: 5432 )
#   -u, --user USER           数据库用户名 ( 必填 )
#   -p, --password PASSWORD   数据库密码 ( 缺省读 PGPASSWORD 环境变量 )
#   -d, --database DATABASE   数据库名 ( 必填 )
#   -D, --directory DIR       SQL 文件目录 ( 与 -f 二选一 )
#   -f, --file FILE           单个 SQL 文件 ( 与 -f 二选一 )
#   --dry-run                 仅列出将要执行的文件
#   --help                    显示帮助

# 默认配置。
DB_HOST="127.0.0.1"
DB_PORT="5432"
DB_USER=""
DB_PASSWORD="${PGPASSWORD:-}"
DB_NAME=""
SQL_DIR=""
SQL_FILE=""
DRY_RUN=false

show_help() {
    grep '^#' "$0" | sed 's/^# \{0,1\}//' | tail -n +2
}

# 解析参数。
while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--host) DB_HOST="$2"; shift 2 ;;
        -P|--port) DB_PORT="$2"; shift 2 ;;
        -u|--user) DB_USER="$2"; shift 2 ;;
        -p|--password) DB_PASSWORD="$2"; shift 2 ;;
        -d|--database) DB_NAME="$2"; shift 2 ;;
        -D|--directory) SQL_DIR="$2"; shift 2 ;;
        -f|--file) SQL_FILE="$2"; shift 2 ;;
        --dry-run) DRY_RUN=true; shift ;;
        --help) show_help; exit 0 ;;
        *) echo "未知选项: $1 ( --help 查看用法 )" >&2; exit 1 ;;
    esac
done

if [[ -z "$DB_USER" || -z "$DB_NAME" ]]; then
    echo "错误: -u ( 用户 ) 与 -d ( 数据库 ) 必填" >&2
    exit 1
fi
if [[ -z "$SQL_DIR" && -z "$SQL_FILE" ]]; then
    echo "错误: -D ( 目录 ) 与 -f ( 文件 ) 必须提供其一" >&2
    exit 1
fi
if [[ -n "$SQL_DIR" && -n "$SQL_FILE" ]]; then
    echo "错误: -D 与 -f 互斥" >&2
    exit 1
fi

# 收集待执行文件 ( 目录模式按文件名升序 )。
mapfile -t FILES < <(
    if [[ -n "$SQL_FILE" ]]; then
        echo "$SQL_FILE"
    else
        find "$SQL_DIR" -maxdepth 1 -type f -name '*.sql' | sort
    fi
)
if [[ ${#FILES[@]} -eq 0 ]]; then
    echo "未找到 SQL 文件" >&2
    exit 1
fi

echo "目标: ${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
for f in "${FILES[@]}"; do
    echo "  - $f"
done

if [[ "$DRY_RUN" == true ]]; then
    echo "( dry-run, 未执行 )"
    exit 0
fi

# 逐个执行 ( ON_ERROR_STOP: 任一失败即中止 )。
for f in "${FILES[@]}"; do
    echo "执行: $f"
    PGPASSWORD="$DB_PASSWORD" psql \
        -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        -v ON_ERROR_STOP=1 -f "$f"
done

echo "完成: ${#FILES[@]} 个文件"
