#!/bin/bash
set -euo pipefail

COOKIE_FILE="/tmp/nextjs_proxy_cookie.txt"
rm -f "$COOKIE_FILE"

EMAIL="proxy-test-$(date +%s)@tayooli.test"
FULL_NAME="Proxy Verification User"
PASSWORD="ProxySecret123!"

echo "=========================================================="
echo "Testing through Next.js Proxy: http://127.0.0.1:3000/api/v1/auth/register"
echo "=========================================================="

REGISTER_RESP=$(curl -i -s -c "$COOKIE_FILE" -X POST http://127.0.0.1:3000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Real-IP: 203.0.113.195" \
  -d "{\"full_name\":\"$FULL_NAME\",\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

echo "$REGISTER_RESP"

echo ""
echo "Testing Next.js Proxy: http://127.0.0.1:3000/api/v1/workspaces"
WORKSPACE_RESP=$(curl -i -s -b "$COOKIE_FILE" -X POST http://127.0.0.1:3000/api/v1/workspaces \
  -H "Content-Type: application/json" \
  -H "X-Real-IP: 203.0.113.195" \
  -d '{"company_name":"NextJS Proxy Company"}')

echo "$WORKSPACE_RESP"

echo ""
echo "Cleaning up DB..."
sudo -u postgres psql -d tayooli_erp -c "
DELETE FROM users WHERE email = '$EMAIL';
DELETE FROM tenants WHERE name = 'NextJS Proxy Company';
"
rm -f "$COOKIE_FILE"
echo "Proxy test cleanup complete."
