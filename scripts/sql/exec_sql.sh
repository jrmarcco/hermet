#!/bin/bash

# SQL 脚本批量执行工具
# 用法: ./execute-sql.sh [选项]
#
# 选项:
#   -t, --db-type TYPE        数据库类型 ( mysql | postgresql，必填 )
#   -h, --host HOST           数据库主机 ( 默认: localhost )
#   -P, --port PORT           数据库端口 ( MySQL默认: 3306, PostgreSQL默认: 5432 )
#   -u, --user USER           数据库用户名 ( 必填 )
#   -p, --password PASSWORD   数据库密码 ( 可选，不提供则提示输入 )
#   -d, --database DATABASE   数据库名 ( 必填 )
#   -D, --directory DIR       SQL文件目录 ( 必填 )
#   -f, --file FILE           单个SQL文件 ( 与 -D 二选一 )
#   --dry-run                 仅显示将要执行的文件，不实际执行
#   --help                    显示帮助信息

set -e

# 默认配置。
DB_TYPE=""
DB_HOST="localhost"
DB_PORT=""
DB_USER=""
DB_PASSWORD=""
DB_NAME=""
SQL_DIR=""
SQL_FILE=""
DRY_RUN=false

# 颜色输出。
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 显示帮助信息。
show_help() {
    cat << EOF
SQL 脚本批量执行工具

用法: $0 [选项]

选项:
  -t, --db-type TYPE        数据库类型 ( mysql | postgresql, 必填 )
  -h, --host HOST           数据库主机 ( 默认: localhost )
  -P, --port PORT           数据库端口 ( MySQL默认: 3306, PostgreSQL默认: 5432 )
  -u, --user USER           数据库用户名 ( 必填 )
  -p, --password PASSWORD   数据库密码 ( 可选，不提供则提示输入 )
  -d, --database DATABASE   数据库名 ( 必填 )
  -D, --directory DIR       SQL文件目录 ( 必填 )
  -f, --file FILE           单个SQL文件 ( 与 -D 二选一 )
  --dry-run                 仅显示将要执行的文件，不实际执行
  --help                    显示帮助信息

示例:
  # 执行目录下所有SQL文件 ( MySQL )
  $0 -t mysql -h localhost -u root -p password -d mydb -D ./sharding/db_0

  # 执行目录下所有SQL文件 ( PostgreSQL )
  $0 -t postgresql -h localhost -u postgres -p password -d mydb -D ./sharding/db_0

  # 执行单个SQL文件
  $0 -t mysql -h localhost -u root -p password -d mydb -f ./init.sql

  # 预览将要执行的文件
  $0 -t mysql -h localhost -u root -d mydb -D ./sharding/db_0 --dry-run

注意:
  - SQL 文件将按照文件名字典序执行
  - 建议使用数字前缀来控制执行顺序 ( 如 00_init.sql, 01_tables.sql )
  - 如果不提供密码参数，系统会提示输入密码
EOF
}

# 解析命令行参数。
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -t|--db-type)
                DB_TYPE="$2"
                shift 2
                ;;
            -h|--host)
                DB_HOST="$2"
                shift 2
                ;;
            -P|--port)
                DB_PORT="$2"
                shift 2
                ;;
            -u|--user)
                DB_USER="$2"
                shift 2
                ;;
            -p|--password)
                DB_PASSWORD="$2"
                shift 2
                ;;
            -d|--database)
                DB_NAME="$2"
                shift 2
                ;;
            -D|--directory)
                SQL_DIR="$2"
                shift 2
                ;;
            -f|--file)
                SQL_FILE="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                echo -e "${RED}错误: 未知选项 $1${NC}"
                show_help
                exit 1
                ;;
        esac
    done
}

# 验证参数。
validate_args() {
    # 验证数据库类型。
    if [[ -z "$DB_TYPE" ]]; then
        echo -e "${RED}错误: 必须指定数据库类型 -t/--db-type${NC}"
        echo ""
        show_help
        exit 1
    fi

    if [[ "$DB_TYPE" != "mysql" ]] && [[ "$DB_TYPE" != "postgresql" ]]; then
        echo -e "${RED}错误: 数据库类型必须是 mysql 或 postgresql${NC}"
        exit 1
    fi

    # 设置默认端口。
    if [[ -z "$DB_PORT" ]]; then
        if [[ "$DB_TYPE" == "mysql" ]]; then
            DB_PORT=3306
        else
            DB_PORT=5432
        fi
    fi

    # 验证用户名。
    if [[ -z "$DB_USER" ]]; then
        echo -e "${RED}错误: 必须指定数据库用户名 -u/--user${NC}"
        exit 1
    fi

    # 验证数据库名。
    if [[ -z "$DB_NAME" ]]; then
        echo -e "${RED}错误: 必须指定数据库名 -d/--database${NC}"
        exit 1
    fi

    # 验证SQL目录或文件。
    if [[ -z "$SQL_DIR" ]] && [[ -z "$SQL_FILE" ]]; then
        echo -e "${RED}错误: 必须指定 SQL 文件目录 -D/--directory 或单个文件 -f/--file${NC}"
        exit 1
    fi

    if [[ -n "$SQL_DIR" ]] && [[ -n "$SQL_FILE" ]]; then
        echo -e "${RED}错误: -D/--directory 和 -f/--file 不能同时使用${NC}"
        exit 1
    fi

    if [[ -n "$SQL_DIR" ]] && [[ ! -d "$SQL_DIR" ]]; then
        echo -e "${RED}错误: 目录不存在: $SQL_DIR${NC}"
        exit 1
    fi

    if [[ -n "$SQL_FILE" ]] && [[ ! -f "$SQL_FILE" ]]; then
        echo -e "${RED}错误: 文件不存在: $SQL_FILE${NC}"
        exit 1
    fi

    # 检查数据库客户端是否安装。
    if [[ "$DB_TYPE" == "mysql" ]]; then
        if ! command -v mysql &> /dev/null; then
            echo -e "${RED}错误: 未找到 mysql 客户端，请先安装${NC}"
            exit 1
        fi
    else
        if ! command -v psql &> /dev/null; then
            echo -e "${RED}错误: 未找到 psql 客户端，请先安装${NC}"
            exit 1
        fi
    fi

    # 如果没有提供密码，提示输入。
    if [[ -z "$DB_PASSWORD" ]] && [[ "$DRY_RUN" == false ]]; then
        read -s -p "请输入数据库密码: " DB_PASSWORD
        echo ""
    fi
}

# 测试数据库连接。
test_connection() {
    echo -e "${YELLOW}测试数据库连接...${NC}"

    if [[ "$DB_TYPE" == "mysql" ]]; then
        if mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" -e "SELECT 1;" &> /dev/null; then
            echo -e "${GREEN}✓ 数据库连接成功${NC}"
            return 0
        else
            echo -e "${RED}✗ 数据库连接失败${NC}"
            return 1
        fi
    else
        if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "SELECT 1;" &> /dev/null; then
            echo -e "${GREEN}✓ 数据库连接成功${NC}"
            return 0
        else
            echo -e "${RED}✗ 数据库连接失败${NC}"
            return 1
        fi
    fi
}

# 执行单个SQL文件。
execute_sql_file() {
    local sql_file="$1"
    local file_name
    file_name=$(basename "$sql_file")

    echo -e "${BLUE}正在执行: $file_name${NC}"

    if [[ "$DRY_RUN" == true ]]; then
        echo -e "${YELLOW}[DRY-RUN] 跳过实际执行${NC}"
        return 0
    fi

    local start_time
    start_time=$(date +%s)

    if [[ "$DB_TYPE" == "mysql" ]]; then
        if mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" < "$sql_file"; then
            local end_time
            end_time=$(date +%s)
            local duration=$((end_time - start_time))
            echo -e "${GREEN}✓ 执行完成 (耗时: ${duration}s)${NC}"
            return 0
        else
            echo -e "${RED}✗ 执行失败${NC}"
            return 1
        fi
    else
        if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$sql_file"; then
            local end_time
            end_time=$(date +%s)
            local duration=$((end_time - start_time))
            echo -e "${GREEN}✓ 执行完成 (耗时: ${duration}s)${NC}"
            return 0
        else
            echo -e "${RED}✗ 执行失败${NC}"
            return 1
        fi
    fi
}

# 执行目录下所有SQL文件。
execute_sql_directory() {
    local sql_dir="$1"

    # 查找所有 .sql 文件并排序。
    local sql_files
    sql_files=$(find "$sql_dir" -maxdepth 1 -name "*.sql" -type f | sort)

    if [[ -z "$sql_files" ]]; then
        echo -e "${YELLOW}警告: 未找到任何 SQL 文件${NC}"
        exit 0
    fi

    local total_files
    total_files=$(echo "$sql_files" | wc -l)
    local current=0
    local failed=0

    echo -e "${YELLOW}找到 $total_files 个 SQL 文件${NC}"
    echo ""

    while IFS= read -r sql_file; do
        current=$((current + 1))
        echo -e "${BLUE}[${current}/${total_files}]${NC}"

        if ! execute_sql_file "$sql_file"; then
            failed=$((failed + 1))
            echo -e "${RED}是否继续执行剩余文件？(y/n)${NC}"
            read -r answer
            if [[ "$answer" != "y" ]] && [[ "$answer" != "Y" ]]; then
                echo -e "${RED}执行已中止${NC}"
                exit 1
            fi
        fi

        echo ""
    done <<< "$sql_files"

    return $failed
}

main() {
    echo -e "${GREEN}======================================${NC}"
    echo -e "${GREEN}SQL 脚本批量执行工具${NC}"
    echo -e "${GREEN}======================================${NC}"
    echo ""

    parse_args "$@"
    validate_args

    # 显示配置信息。
    echo "执行配置:"
    echo "  数据库类型: $DB_TYPE"
    echo "  主机: $DB_HOST"
    echo "  端口: $DB_PORT"
    echo "  用户: $DB_USER"
    echo "  数据库: $DB_NAME"
    if [[ -n "$SQL_DIR" ]]; then
        echo "  SQL目录: $SQL_DIR"
    else
        echo "  SQL文件: $SQL_FILE"
    fi
    if [[ "$DRY_RUN" == true ]]; then
        echo -e "  ${YELLOW}模式: 预览模式 (不实际执行)${NC}"
    fi
    echo ""

    # 测试连接 ( 非 dry-run 模式 )。
    if [[ "$DRY_RUN" == false ]]; then
        if ! test_connection; then
            echo -e "${RED}请检查数据库连接参数${NC}"
            exit 1
        fi
        echo ""
    fi

    # 执行SQL。
    local start_time
    start_time=$(date +%s)

    if [[ -n "$SQL_DIR" ]]; then
        execute_sql_directory "$SQL_DIR"
        local failed=$?
    else
        execute_sql_file "$SQL_FILE"
        local failed=$?
    fi

    local end_time
    end_time=$(date +%s)
    local total_duration=$((end_time - start_time))

    echo -e "${GREEN}======================================${NC}"
    if [[ $failed -eq 0 ]]; then
        echo -e "${GREEN}所有任务完成！${NC}"
    else
        echo -e "${YELLOW}完成，但有 $failed 个文件执行失败${NC}"
    fi
    echo -e "${GREEN}总耗时: ${total_duration}s${NC}"
    echo -e "${GREEN}======================================${NC}"

    exit $failed
}

main "$@"

