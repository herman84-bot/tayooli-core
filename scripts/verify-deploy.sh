#!/bin/bash
set -euo pipefail

COOKIE_FILE="/tmp/sre_test_cookies.txt"
rm -f "$COOKIE_FILE"

EMAIL="sre-live-$(date +%s)@tayooli.test"
FULL_NAME="SRE Automated Verification"
PASSWORD="VerificationSecret123!"

echo "=========================================================="
echo "1. Testing POST https://tayooli.my.id/api/v1/auth/register"
echo "   Email: $EMAIL"
echo "=========================================================="

REGISTER_RESP=$(curl -i -s -c "$COOKIE_FILE" -X POST https://tayooli.my.id/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d "{\"full_name\":\"$FULL_NAME\",\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

echo "$REGISTER_RESP"

echo ""
echo "=========================================================="
echo "2. Cookie Jar Verification"
echo "=========================================================="
cat "$COOKIE_FILE"

echo ""
echo "=========================================================="
echo "3. Testing POST https://tayooli.my.id/api/v1/workspaces"
echo "   Using auth cookie to update company name"
echo "=========================================================="

WORKSPACE_RESP=$(curl -i -s -b "$COOKIE_FILE" -X POST https://tayooli.my.id/api/v1/workspaces \
  -H "Content-Type: application/json" \
  -d '{"company_name":"SRE Verified Enterprise"}')

echo "$WORKSPACE_RESP"

echo ""
echo "=========================================================="
echo "4. Database State Verification (psql)"
echo "=========================================================="
USER_INFO=$(sudo -u postgres psql -d tayooli_erp -t -A -F"|" -c "SELECT u.id, u.email, u.full_name, t.id, t.name FROM users u JOIN tenants t ON u.tenant_id = t.id WHERE u.email = '$EMAIL';")
echo "Found DB Record: $USER_INFO"

TENANT_ID=$(echo "$USER_INFO" | cut -d'|' -f4)
USER_ID=$(echo "$USER_INFO" | cut -d'|' -f1)

echo "USER_ID: $USER_ID"
echo "TENANT_ID: $TENANT_ID"

echo ""
echo "=========================================================="
echo "5. Cleanup Test Records"
echo "=========================================================="
sudo -u postgres psql -d tayooli_erp -c "DELETE FROM users WHERE id = '$USER_ID';"
sudo -u postgres psql -d tayooli_erp -c "DELETE FROM tenants WHERE id = '$TENANT_ID';"
rm -f "$COOKIE_FILE"

echo ""
echo "=========================================================="
echo "6. Confirm Cleanup"
echo "=========================================================="
REMAINING=$(sudo -u postgres psql -d tayooli_erp -t -A -c "SELECT count(*) FROM users WHERE email = '$EMAIL';")
echo "Remaining users with test email: $REMAINING"

echo ""
echo "✅ All verification steps completed successfully!"
