#!/bin/bash

# ====== 配置参数 ======
URI="http://localhost:8080"
ACCOUNT="jrmarcco@gmail.com"
ACCOUNT_TYPE="email"
CREDENTIAL="<passwd>"

# 用户创建参数
START_INDEX=${1:-1}         # 起始序号 ( 默认 1 )
COUNT=${2:-100}             # 创建数量 ( 默认 100 )
GENDER=${3:-"female"}       # 性别 ( 默认 female )

# ====== 登录获取 Token ======
echo ">>> 正在登录..."
LOGIN_RESPONSE=$(curl -s -X POST "${URI}/api/v1/auth/sign-in" \
  -H "Content-Type: application/json" \
  -d "{
    \"account\": \"${ACCOUNT}\",
    \"accountType\": \"${ACCOUNT_TYPE}\",
    \"credential\": \"${CREDENTIAL}\"
  }")

# 提取 accessToken ( 需要 jq 工具 )
ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.accessToken')

if [ "$ACCESS_TOKEN" == "null" ] || [ -z "$ACCESS_TOKEN" ]; then
  echo ">>> 登录失败，响应: $LOGIN_RESPONSE"
  exit 1
fi

echo ">>> 登录成功 [ Token: ${ACCESS_TOKEN:0:20} ]"

# ====== 循环创建用户 ======
END_INDEX=$((START_INDEX + COUNT - 1))

for i in $(seq $START_INDEX $END_INDEX); do
  EMAIL="user-${i}@example.com"
  MOBILE="1310013100${i}"
  NICKNAME="nickname-${i}"

  echo ""
  echo ">>> 正在创建用户 #${i}: ${EMAIL}"

  RESPONSE=$(curl -s -X POST "${URI}/api/v1/admin/user" \
    -H "Content-Type: application/json" \
    -H "x-access-token: Bearer ${ACCESS_TOKEN}" \
    -d "{
      \"email\": \"${EMAIL}\",
      \"mobile\": \"${MOBILE}\",
      \"nickname\": \"${NICKNAME}\",
      \"avatar\": \"\",
      \"gender\": \"${GENDER}\",
      \"region\": \"\",
      \"birthday\": 0,
      \"tagline\": \"\"
    }")

  echo ">>> 响应: $RESPONSE"
done

echo ""
echo ">>> 共创建 ${COUNT} 个用户 ( 序号 ${START_INDEX} ~ ${END_INDEX} )"
