#!/usr/bin/env bash
set -euo pipefail

# ./scripts/genpkey.sh -d internal/api/grpc/interceptor/jwt/key

# 生成私钥
# openssl genpkey -algorithm ed25519 -out private.pem
# 从私钥生成公钥
# openssl pkey -in private.pem -pubout -out public.pem


# Default directory for keys
KEY_DIR="keys"

# Parse command line arguments
while getopts "d:" opt; do
  case $opt in
    d) KEY_DIR="$OPTARG" ;;
    \?) echo "Invalid option -$OPTARG" >&2; exit 1 ;;
  esac
done

# Create directory if it doesn't exist
mkdir -p "$KEY_DIR"
chmod 700 "$KEY_DIR"

# Generate private key
echo "Generating private key..."
openssl genpkey -algorithm Ed25519 -out "$KEY_DIR/private.pem"
chmod 600 "$KEY_DIR/private.pem"

# Extract public key
echo "Extracting public key..."
openssl pkey -in "$KEY_DIR/private.pem" -pubout -out "$KEY_DIR/public.pem"
chmod 644 "$KEY_DIR/public.pem"

echo "Successfully generated key pair in directory: $KEY_DIR"
echo "Private key: $KEY_DIR/private.pem"
echo "Public key: $KEY_DIR/public.pem"
