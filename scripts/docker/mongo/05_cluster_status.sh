#!/bin/bash

echo "=========================================="
echo "MongoDB 数据库初始化脚本"
echo "=========================================="
echo ""

USER="jrmarcco"
PASSWORD="<passwd>"

echo "正在创建数据库 hermet 并配置分片..."
echo ""

docker exec mongodb-mongos-1 mongosh --port 27017 admin \
  --tls \
  --tlsCertificateKeyFile /etc/ssl/mongodb/mongo.pem \
  --tlsCAFile /etc/ssl/mongodb/ca.pem \
  --authenticationDatabase admin -u "$USER" -p "$PASSWORD" \
  --quiet --eval "
// 切换到 hermet 数据库
db = db.getSiblingDB('hermet');

// 启用数据库分片
print('1. 启用数据库 hermet 的分片功能...');
sh.enableSharding('hermet');

// 创建 message collection
print('2. 创建 message collection...');
db.createCollection('message');

// 创建分片键索引
print('3. 为 message collection 创建分片键索引 (cid)...');
db.message.createIndex({ cid: 1 });

// 对 collection 进行分片，使用 cid 作为分片键
print('4. 对 message collection 进行分片，分片键: { cid: 1 }...');
sh.shardCollection('hermet.message', { cid: 1 });

print('');
print('==========================================');
print('数据库初始化完成！');
print('==========================================');
print('数据库: hermet');
print('Collection: message');
print('分片键: { cid: 1 }');
print('');
"
