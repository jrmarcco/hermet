#!/bin/bash

# 默认参数值
CONFIG_SERVERS=3
SHARDS=3
NODES_PER_SHARD=3
MONGOS=1

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --config-servers)
            CONFIG_SERVERS="$2"
            shift 2
            ;;
        --shards)
            SHARDS="$2"
            shift 2
            ;;
        --nodes-per-shard)
            NODES_PER_SHARD="$2"
            shift 2
            ;;
        --mongos)
            MONGOS="$2"
            shift 2
            ;;
        -h|--help)
            echo "用法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  --config-servers NUM     配置服务器数量 ( 默认 3 )"
            echo "  --shards NUM             分片数量 ( 默认 3 )"
            echo "  --nodes-per-shard NUM    每个分片的节点数 ( 默认 3 )"
            echo "  --mongos NUM             路由器数量 ( 默认 1 )"
            echo "  -h, --help               显示此帮助信息"
            echo ""
            echo "示例:"
            echo "  $0                                   # 使用默认值 ( 3 配置服务器 / 3 分片 / 每分片 3 节点 )"
            echo "  $0 --config-servers 3 --shards 5     # 3 配置服务器 / 5 分片"
            echo "  $0 --shards 2 --nodes-per-shard 2    # 2 分片 / 每分片 2 节点"
            exit 0
            ;;
        *)
            echo "未知参数: $1"
            echo "使用 -h 或 --help 查看帮助信息"
            exit 1
            ;;
    esac
done

# 验证参数为正整数
if ! [[ "$CONFIG_SERVERS" =~ ^[1-9][0-9]*$ ]] || ! [[ "$SHARDS" =~ ^[1-9][0-9]*$ ]] || ! [[ "$NODES_PER_SHARD" =~ ^[1-9][0-9]*$ ]]; then
    echo "错误: 所有参数必须为正整数"
    exit 1
fi

echo "创建 MongoDB 数据目录..."
echo "配置服务器数量: $CONFIG_SERVERS"
echo "分片数量: $SHARDS"
echo "每个分片的节点数: $NODES_PER_SHARD"
echo "路由器数量: $MONGOS"
echo ""

# 创建配置服务器目录
echo "创建配置服务器目录..."
for i in $(seq 1 $CONFIG_SERVERS); do
    mkdir -p "config-svr-${i}-data"
    mkdir -p "config-svr-${i}-config-data"
done

# 创建分片目录
echo "创建分片目录..."
NODE_LABELS=($(echo {a..z}))
for shard in $(seq 1 $SHARDS); do
    for node in $(seq 0 $((NODES_PER_SHARD - 1))); do
        node_label=${NODE_LABELS[$node]}
        mkdir -p "shard${shard}-${node_label}-data"
    done
done

# 创建路由器目录
echo "创建路由器目录..."
for i in $(seq 1 $MONGOS); do
    mkdir -p "mongos-${i}-data"
done

echo ""
echo "目录创建完成！"
