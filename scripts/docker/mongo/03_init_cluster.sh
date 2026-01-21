#!/bin/bash

# MongoDB 分片集群初始化脚本

# 默认参数值
CONFIG_SERVERS=3
SHARDS=3
NODES_PER_SHARD=3

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
        -h|--help)
            echo "用法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  --config-servers NUM     配置服务器数量 ( 默认 3 )"
            echo "  --shards NUM             分片数量 ( 默认 3 )"
            echo "  --nodes-per-shard NUM    每个分片的节点数 ( 默认 3 )"
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

echo "=========================================="
echo "开始初始化 MongoDB 分片集群"
echo "=========================================="

# 等待所有容器启动完成
echo ""
echo "等待所有容器启动完成..."
sleep 10

echo ""
echo "=========================================="
echo "1. 初始化配置服务器副本集 (configReplSet)"
echo "=========================================="

CONFIG_MEMBERS=""
for i in $(seq 1 $CONFIG_SERVERS); do
  CONFIG_MEMBERS="$CONFIG_MEMBERS{ _id: $((i-1)), host: \"mongodb-config-server-$i:27019\" },"
done
CONFIG_MEMBERS=${CONFIG_MEMBERS%,}

docker exec mongodb-config-server-1 mongosh --port 27019 \
  --tls \
  --tlsCertificateKeyFile /etc/ssl/mongodb/mongo.pem \
  --tlsCAFile /etc/ssl/mongodb/ca.pem \
  --quiet --eval "
rs.initiate({
  _id: \"configReplSet\",
  configsvr: true,
  members: [ $CONFIG_MEMBERS ]
})
"

echo "等待配置服务器副本集选举主节点..."
sleep 10

# 初始化分片副本集
for i in $(seq 1 $SHARDS); do
  echo ""
  echo "=========================================="
  echo "初始化分片 $i 副本集 ( shard${i}ReplSet )"
  echo "=========================================="

  SHARD_MEMBERS=""
  # 节点命名规则：1-a, 1-b, 1-c ... ( 通常不会超过 26 个节点 )
  for j in $(seq 1 $NODES_PER_SHARD); do
    LETTER=$(printf "\\$(printf '%03o' $((96+j)))")
    SHARD_MEMBERS="$SHARD_MEMBERS{ _id: $((j-1)), host: \"mongodb-shard$i-$LETTER:27018\" },"
  done
  SHARD_MEMBERS=${SHARD_MEMBERS%,}

  # 使用该分片的第一个节点执行初始化
  FIRST_NODE_LETTER=$(printf "\\$(printf '%03o' $((96+1)))")

  docker exec mongodb-shard$i-$FIRST_NODE_LETTER mongosh --port 27018 \
    --tls \
    --tlsCertificateKeyFile /etc/ssl/mongodb/admin-client.pem \
    --tlsCAFile /etc/ssl/mongodb/ca.pem \
    --quiet --eval "
rs.initiate({
  _id: \"shard${i}ReplSet\",
  members: [ $SHARD_MEMBERS ]
})
"
  echo "等待分片 $i 副本集选举主节点..."
  sleep 10
done

# 将分片添加到集群
echo ""
echo "=========================================="
echo "5. 将分片添加到集群"
echo "=========================================="

ADD_SHARDS_CMD=""
for i in $(seq 1 $SHARDS); do
  SHARD_HOSTS=""
  for j in $(seq 1 $NODES_PER_SHARD); do
    LETTER=$(printf "\\$(printf '%03o' $((96+j)))")
    SHARD_HOSTS="$SHARD_HOSTS,mongodb-shard$i-$LETTER:27018"
  done
  SHARD_HOSTS=${SHARD_HOSTS#,}
  ADD_SHARDS_CMD="$ADD_SHARDS_CMD sh.addShard(\"shard${i}ReplSet/$SHARD_HOSTS\");"
done

docker exec mongodb-mongos-1 mongosh --port 27017 \
  --tls --tlsCertificateKeyFile /etc/ssl/mongodb/admin-client.pem --tlsCAFile /etc/ssl/mongodb/ca.pem \
  --quiet --eval "$ADD_SHARDS_CMD"

echo ""
echo "=========================================="
echo "MongoDB 分片集群初始化完成 ( 已启用 TLS )"
echo "=========================================="
echo ""
echo "您可以使用以下命令连接到 mongos "
echo "  docker exec -it mongodb-mongos-1 mongosh --port 27017 --tls --tlsCertificateKeyFile /etc/ssl/mongodb/admin-client.pem --tlsCAFile /etc/ssl/mongodb/ca.pem"
echo ""
echo "或者从主机连接 ( 需要证书文件 )"
echo "  mongosh 'mongodb://localhost:27017/?tls=true&tlsCertificateKeyFile=./certs/mongo.pem&tlsCAFile=./certs/ca.pem'"
echo ""

