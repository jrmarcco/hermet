#!/bin/bash

echo "=========================================="
echo "MongoDB 数据库初始化脚本"
echo "=========================================="
echo ""

USER="jrmarcco"
PASSWORD="<passwd>"

docker exec mongodb-mongos-1 mongosh --port 27017 admin \
  --tls \
  --tlsCertificateKeyFile /etc/ssl/mongodb/mongo.pem \
  --tlsCAFile /etc/ssl/mongodb/ca.pem \
  --authenticationDatabase admin -u "$USER" -p "$PASSWORD" \
  --quiet --eval "
db.createUser({
  user: 'jrmarcco',
  pwd: '<passwd>',
  roles: [ { role: 'root', db: 'admin' } ]
})
"
