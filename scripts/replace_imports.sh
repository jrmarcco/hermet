#!/bin/bash

# 替换Go文件中import块内的指定前缀
# 用法: ./replace_imports.sh <目录路径> <原前缀> <新前缀>

set -e

# 颜色定义。
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查参数。
if [ $# -ne 3 ]; then
    echo -e "${RED}错误: 参数不足${NC}"
    echo "用法: $0 <目录路径> <原前缀> <新前缀>"
    echo "示例: $0 . github.com/jrmarcco/hermet gitee.com/jrmarcco/hermet-go"
    exit 1
fi

DIR_PATH="$1"
OLD_PREFIX="$2"
NEW_PREFIX="$3"

# 检查目录是否存在。
if [ ! -d "$DIR_PATH" ]; then
    echo -e "${RED}错误: 目录 '$DIR_PATH' 不存在${NC}"
    exit 1
fi

# 转义特殊字符用于 sed ( 已定义变量但未使用，保留以备将来扩展 )。
OLD_PREFIX_ESCAPED=$(printf '%s\n' "$OLD_PREFIX" | sed 's/[\/&]/\\&/g')
NEW_PREFIX_ESCAPED=$(printf '%s\n' "$NEW_PREFIX" | sed 's/[\/&]/\\&/g')

echo -e "${GREEN}开始扫描目录: $DIR_PATH${NC}"
echo -e "${YELLOW}原前缀: $OLD_PREFIX${NC}"
echo -e "${YELLOW}新前缀: $NEW_PREFIX${NC}"
echo ""

# 统计变量。
TOTAL_FILES=0
MODIFIED_FILES=0

# 创建临时文件来存储修改后的内容。
TEMP_FILE=$(mktemp "${TMPDIR:-/tmp}/replace_imports.XXXXXX")

# 查找所有 .go 文件并处理。
while IFS= read -r -d '' go_file; do
    TOTAL_FILES=$((TOTAL_FILES + 1))

    echo -e "${YELLOW}处理: $go_file${NC}"

    # 使用 awk 处理文件，只在 import 块内进行替换。
    awk -v old_prefix="$OLD_PREFIX" -v new_prefix="$NEW_PREFIX" '
    BEGIN {
        in_import = 0
        modified = 0
    }

    # 检测单行 import。
    /^import[[:space:]]+"/ {
        if (index($0, "\"" old_prefix) > 0) {
            # 完全匹配检查。
            # 使用正则表达式确保完整匹配: "prefix" 或 "prefix/
            line = $0
            regex = "\"" old_prefix "([\"/]|$)"
            if (match(line, regex)) {
                gsub("\"" old_prefix, "\"" new_prefix, line)
                $0 = line
                modified = 1
            }
        }
        print
        next
    }

    # 检测 import 块开始。
    /^import[[:space:]]*\(/ {
        in_import = 1
        print
        next
    }

    # 检测 import 块结束。
    /^\)/ && in_import {
        in_import = 0
        print
        next
    }

    # 在 import 块内处理。
    in_import {
        if (index($0, old_prefix) > 0) {
            # 完全匹配检查 - 确保是完整的路径匹配。
            # 匹配 "old_prefix 或 "old_prefix/ 或 old_prefix" 或 old_prefix/
            line = $0
            regex1 = "\"" old_prefix "([\"/]|$)"
            regex2 = "[[:space:]]" old_prefix "([\"/]|$)"
            if (match(line, regex1) || match(line, regex2)) {
                gsub(old_prefix, new_prefix, line)
                $0 = line
                modified = 1
            }
        }
        print
        next
    }

    # 其他行直接输出。
    {
        print
    }

    END {
        exit modified
    }
    ' "$go_file" > "$TEMP_FILE" || awk_exit_code=$?

    # 检查是否有修改 ( 通过 awk 的退出状态 )。
    awk_exit_code=${awk_exit_code:-0}
    if [ $awk_exit_code -eq 1 ]; then
        # 有修改则替换原文件。
        mv "$TEMP_FILE" "$go_file"
        echo -e "${GREEN}✓ 已修改: $go_file${NC}"
        MODIFIED_FILES=$((MODIFIED_FILES + 1))
    else
        # 没有修改则删除临时文件。
        rm -f "$TEMP_FILE"
    fi

done < <(find "$DIR_PATH" -type f -name "*.go" -print0)

# 清理可能残留的临时文件。
rm -f "$TEMP_FILE"

# 输出统计信息。
echo ""
echo -e "${GREEN}========== 完成 ==========${NC}"
echo -e "扫描文件数: ${YELLOW}$TOTAL_FILES${NC}"
echo -e "修改文件数: ${GREEN}$MODIFIED_FILES${NC}"
echo ""

if [ $MODIFIED_FILES -gt 0 ]; then
    echo -e "${YELLOW}提示: 建议运行以下命令验证修改:${NC}"
    echo "  go mod tidy"
    echo "  go build ./..."
fi
