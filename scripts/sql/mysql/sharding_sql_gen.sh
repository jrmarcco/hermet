#!/usr/bin/env bash
set -euo pipefail

# 分库分表 SQL 脚本生成工具 ( MySQL 版本 )
# 用法: ./sharding-sql-gen.sh [选项]
#
# 选项:
#   -d, --db-count NUM        数据库分片数量 ( 默认: 2 )
#   -t, --table-count NUM     表分片数量 ( 默认: 4 )
#   -i, --input-file FILE     输入 SQL 文件路径
#   -o, --output-dir DIR      输出目录 ( 默认: ./sharding )
#   -p, --db-prefix PREFIX    数据库名前缀 ( 必填 )
#   --all                     处理所有 ./*.sql 文件
#   -h, --help                显示帮助信息

# 默认配置。
DB_COUNT=2
TABLE_COUNT=4
OUTPUT_DIR="./sharding"
DB_PREFIX=""
INPUT_FILE=""
PROCESS_ALL=false

# 颜色输出。
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 显示帮助信息。
show_help() {
    cat << EOF
分库分表SQL脚本生成工具 ( MySQL 版本 )

用法: $0 [选项]

选项:
  -p, --db-prefix PREFIX    数据库名前缀 ( 必填 )
  -d, --db-count NUM        数据库分片数量 ( 默认: 2 )
  -t, --table-count NUM     表分片数量 ( 默认: 4 )
  -i, --input-file FILE     输入 SQL 文件路径
  -o, --output-dir DIR      输出目录 ( 默认: ./sharding )
  --all                     处理所有 ./*.sql 文件
  -h, --help                显示帮助信息

示例:
  # 生成数据库初始化脚本
  $0 -p hermet --all

  # 生成数据库 + 处理所有表文件
  $0 -p hermet --all -d 2 -t 4
EOF
}

# 解析命令行参数。
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -d|--db-count)
                DB_COUNT="$2"
                shift 2
                ;;
            -t|--table-count)
                TABLE_COUNT="$2"
                shift 2
                ;;
            -i|--input-file)
                INPUT_FILE="$2"
                shift 2
                ;;
            -o|--output-dir)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            -p|--db-prefix)
                DB_PREFIX="$2"
                shift 2
                ;;
            --all)
                PROCESS_ALL=true
                shift
                ;;
            -h|--help)
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
    if [[ -z "$DB_PREFIX" ]]; then
        echo -e "${RED}错误: 必须指定数据库名前缀 -p/--db-prefix${NC}"
        echo ""
        show_help
        exit 1
    fi

    # 如果不是只生成数据库脚本，则需要指定输入文件或 --all。
    if [[ "$PROCESS_ALL" == true ]] || [[ -n "$INPUT_FILE" ]]; then
        if [[ "$PROCESS_ALL" == false ]] && [[ -z "$INPUT_FILE" ]]; then
            echo -e "${RED}错误: 必须指定 -i/--input-file、--all"
            show_help
            exit 1
        fi

        if [[ "$PROCESS_ALL" == false ]] && [[ -n "$INPUT_FILE" ]] && [[ ! -f "$INPUT_FILE" ]]; then
            echo -e "${RED}错误: 文件不存在: $INPUT_FILE${NC}"
            exit 1
        fi
    fi

    if ! [[ "$DB_COUNT" =~ ^[0-9]+$ ]] || [[ "$DB_COUNT" -lt 1 ]]; then
        echo -e "${RED}错误: 数据库分片数量必须是大于0的整数${NC}"
        exit 1
    fi

    if ! [[ "$TABLE_COUNT" =~ ^[0-9]+$ ]] || [[ "$TABLE_COUNT" -lt 1 ]]; then
        echo -e "${RED}错误: 表分片数量必须是大于0的整数${NC}"
        exit 1
    fi
}

# 生成数据库初始化脚本。
generate_db_init_scripts() {
    echo -e "${YELLOW}生成数据库初始化脚本${NC}"
    echo "配置: DB_COUNT=$DB_COUNT, DB_PREFIX=$DB_PREFIX"
    echo ""

    # 创建输出目录。
    mkdir -p "$OUTPUT_DIR"

    for ((db_idx=0; db_idx<DB_COUNT; db_idx++)); do
        local db_name="${DB_PREFIX}_${db_idx}"
        local output_file="${OUTPUT_DIR}/00_db_init_db${db_idx}.sql"

        echo -e "${GREEN}正在生成: $output_file${NC}"

        {
            echo "-- ============================================"
            echo "-- 分库分表SQL脚本 - 数据库初始化 ( MySQL 版本 )"
            echo "-- 数据库: $db_name"
            echo "-- 生成时间: $(date '+%Y-%m-%d %H:%M:%S')"
            echo "-- ============================================"
            echo ""
            echo "-- 创建分片数据库"
            echo "CREATE DATABASE IF NOT EXISTS $db_name DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
            echo ""
        } > "$output_file"

        echo -e "${GREEN}✓ 生成完成: $output_file${NC}"
    done

    echo ""
}

# 提取 SQL 文件中的所有表名。
extract_table_names() {
    local sql_file="$1"
    grep -iE '^[[:space:]]*(CREATE|DROP)[[:space:]]+TABLE[[:space:]]+(IF[[:space:]]+(NOT[[:space:]]+)?EXISTS[[:space:]]+)?[a-zA-Z_][a-zA-Z0-9_]*' "$sql_file" | \
        sed -E 's/.*TABLE[[:space:]]+(IF[[:space:]]+(NOT[[:space:]]+)?EXISTS[[:space:]]+)?([a-zA-Z_][a-zA-Z0-9_]*).*/\3/' | sort -u
}

# 生成单个分库分表 SQL。
generate_sharding_sql() {
    local input_file="$1"
    local db_idx="$2"
    local output_file="$3"

    local db_name="${DB_PREFIX}_${db_idx}"

    echo -e "${GREEN}正在生成: $output_file${NC}"

    # 读取原始 SQL 内容。
    local content
    content=$(cat "$input_file")

    # 获取所有表名。
    local table_names
    table_names=$(extract_table_names "$input_file")

    # 开始生成新的 SQL 文件。
    {
        echo "-- ============================================"
        echo "-- 分库分表SQL脚本 ( MySQL 版本 )"
        echo "-- 数据库: $db_name"
        echo "-- 分表数量: $TABLE_COUNT"
        echo "-- 生成时间: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "-- 原始文件: $input_file"
        echo "-- ============================================"
        echo ""

        # 为每个表生成分表。
        while IFS= read -r table_name; do
            [[ -z "$table_name" ]] && continue

            echo "-- ============================================"
            echo "-- 表: $table_name (分表数: $TABLE_COUNT)"
            echo "-- ============================================"
            echo ""

            for ((table_idx=0; table_idx<TABLE_COUNT; table_idx++)); do
                local sharded_table_name="${table_name}_${table_idx}"

                echo "-- 分表 $table_idx: $sharded_table_name"

                # 提取该表的完整定义 ( 从 DROP 到对应的 ); )。
                local table_section
                table_section=$(echo "$content" | awk -v table="$table_name" '
                    /DROP TABLE.*IF.*EXISTS/ {
                        # 精确匹配表名: IF EXISTS table; 或 IF EXISTS table 后面跟空格/分号
                        if (index($0, "EXISTS " table ";") > 0 ||
                            (index($0, "EXISTS " table " ") > 0) ||
                            (index($0, "EXISTS " table) > 0 && $0 ~ /EXISTS [a-zA-Z_][a-zA-Z0-9_]*[[:space:]]*$/)) {
                            found=1
                        }
                    }
                    found { print }
                    found && /ENGINE=InnoDB/ {
                        found=0
                    }
                ')

                # 替换表名。
                table_section=$(echo "$table_section" | sed -E "s/(^|[^a-zA-Z0-9_])${table_name}([^a-zA-Z0-9_]|$)/\1${sharded_table_name}\2/g")

                # 替换 CONSTRAINT 名称 ( 添加分表后缀 )。
                table_section=$(echo "$table_section" | sed -E "s/(CONSTRAINT[[:space:]]+)([a-zA-Z_][a-zA-Z0-9_]*)/\1\2_${table_idx}/g")

                echo "$table_section"
                echo ""

                # 提取该表的索引创建语句 ( 支持多行 )。
                local index_section
                index_section=$(echo "$content" | awk -v table="$table_name" '
                    # 匹配 CREATE INDEX ... ON table_name(
                    /CREATE[[:space:]]+.*INDEX.*ON[[:space:]]+/ {
                        # 检查是否包含目标表名: ON table( 或 ON table (
                        if (index($0, "ON " table "(") > 0 || index($0, "ON " table " (") > 0) {
                            found=1
                        }
                    }
                    # 如果找到了开始，打印所有行直到分号
                    found {
                        buffer = buffer $0 "\n"
                    }
                    # 遇到分号结束
                    found && /;$/ {
                        print buffer
                        buffer = ""
                        found=0
                    }
                ' | sed -E "s/(^|[^a-zA-Z0-9_])${table_name}([^a-zA-Z0-9_]|$)/\1${sharded_table_name}\2/g" | \
                    sed -E "s/(INDEX[[:space:]]+)([a-zA-Z_][a-zA-Z0-9_]*)/\1\2_${table_idx}/g")

                if [[ -n "$index_section" ]]; then
                    echo "-- 创建索引"
                    echo "$index_section"
                fi

                echo ""
            done
        done <<< "$table_names"

    } > "$output_file"

    echo -e "${GREEN}✓ 生成完成: $output_file${NC}"
}

# 处理单个 SQL 文件。
process_single_file() {
    local input_file="$1"
    local base_name
    base_name=$(basename "$input_file" .sql)

    echo -e "${YELLOW}处理文件: $input_file${NC}"
    echo "配置: DB_COUNT=$DB_COUNT, TABLE_COUNT=$TABLE_COUNT"
    echo ""

    # 为每个数据库分片生成 SQL。
    for ((db_idx=0; db_idx<DB_COUNT; db_idx++)); do
        local db_dir="${OUTPUT_DIR}/db_${db_idx}"
        # 创建数据库子目录。
        mkdir -p "$db_dir"
        local output_file="${db_dir}/${base_name}.sql"
        generate_sharding_sql "$input_file" "$db_idx" "$output_file"
    done

    echo ""
}

# 处理所有 SQL 文件。
process_all_files() {
    local sql_dir="."

    if [[ ! -d "$sql_dir" ]]; then
        echo -e "${RED}错误: SQL目录不存在: $sql_dir${NC}"
        exit 1
    fi

    # 查找所有 *_init.sql 文件 ( 数据库初始化由参数生成 )。
    local sql_files
    sql_files=$(find "$sql_dir" -maxdepth 1 -name "*.sql" -type f | sort)

    if [[ -z "$sql_files" ]]; then
        echo -e "${RED}错误: 未找到SQL文件${NC}"
        exit 1
    fi

    while IFS= read -r sql_file; do
        process_single_file "$sql_file"
    done <<< "$sql_files"
}

main() {
    echo -e "${GREEN}======================================${NC}"
    echo -e "${GREEN}分库分表SQL脚本生成工具 ( MySQL 版本 )${NC}"
    echo -e "${GREEN}======================================${NC}"
    echo ""

    parse_args "$@"
    validate_args

    # 生成数据库初始化脚本。
    generate_db_init_scripts

    # 处理表文件。
    if [[ "$PROCESS_ALL" == true ]]; then
        process_all_files
    elif [[ -n "$INPUT_FILE" ]]; then
        process_single_file "$INPUT_FILE"
    fi

    echo -e "${GREEN}======================================${NC}"
    echo -e "${GREEN}所有任务完成！${NC}"
    echo -e "${GREEN}输出目录: $OUTPUT_DIR${NC}"
    echo -e "${GREEN}======================================${NC}"
}

main "$@"

