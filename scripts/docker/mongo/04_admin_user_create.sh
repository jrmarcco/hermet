#!/usr/bin/env bash
set -euo pipefail

echo "=========================================="
echo "MongoDB 管理员用户创建脚本"
echo "=========================================="
echo ""

docker exec mongodb-mongos-1 mongosh --port 27017 admin \
  --tls \
  --tlsCertificateKeyFile /etc/ssl/mongodb/mongo.pem \
  --tlsCAFile /etc/ssl/mongodb/ca.pem \
  --quiet --eval "
db.createUser({
  user: 'jrmarcco',
  pwd: '<passwd>',
  roles: [ { role: 'root', db: 'admin' } ]
})
"
