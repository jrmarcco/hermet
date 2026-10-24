#!/bin/sh
# 生成网关连接鉴权 JWT 的 RSA 密钥对 ( 契约见 synp-api proto/auth/v1/auth.proto )。
#
# 用法:
#   sh ./scripts/gen_jwt_rsa_keys.sh          # 输出到 stdout
#   sh ./scripts/gen_jwt_rsa_keys.sh /tmp/gw  # 输出到 /tmp/gw_private.pem 与 /tmp/gw_public.pem
#
# kid 为 DER 公钥 SHA-256 的前 16 位 hex，需与 etc/config.yaml 的 jwt.gateway.kid 一致。
# 密钥轮换: 生成新密钥对 -> 新公钥加入 JWKS ( 并存重叠期 ) -> 签发切换到新私钥 -> 下线旧公钥。

set -e

BITS=2048
OUT_PREFIX=""

if [ "$#" -gt 1 ]; then
    echo "usage: $0 [output_prefix]" >&2
    exit 1
fi

if [ "$#" -eq 1 ]; then
    OUT_PREFIX="$1"
fi

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:$BITS -out "$TMP_DIR/private.pem" 2>/dev/null
openssl pkey -in "$TMP_DIR/private.pem" -pubout -out "$TMP_DIR/public.pem" 2>/dev/null
openssl pkey -in "$TMP_DIR/private.pem" -pubout -outform DER -out "$TMP_DIR/public.der" 2>/dev/null

KID=$(sha256sum "$TMP_DIR/public.der" | cut -c1-16)

if [ -n "$OUT_PREFIX" ]; then
    cp "$TMP_DIR/private.pem" "${OUT_PREFIX}_private.pem"
    cp "$TMP_DIR/public.pem" "${OUT_PREFIX}_public.pem"
    echo "private key: ${OUT_PREFIX}_private.pem"
    echo "public key:  ${OUT_PREFIX}_public.pem"
else
    echo "===== private key ====="
    cat "$TMP_DIR/private.pem"
    echo "===== public key ====="
    cat "$TMP_DIR/public.pem"
fi

echo "===== kid ====="
echo "$KID"
echo "把公私钥填入 etc/config.yaml 的 jwt.gateway 段，kid 填入 jwt.gateway.kid。"
