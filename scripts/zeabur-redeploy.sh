#!/usr/bin/env bash
# Redeploy Zeabur services (pulls latest commit of the configured branch).
# Usage: ZEABUR_API_TOKEN=... ZEABUR_ENV_ID=... scripts/zeabur-redeploy.sh <serviceID>...
# Exits non-zero if any redeploy is not acknowledged with {"data":{"redeployService":true}}.
set -euo pipefail

: "${ZEABUR_API_TOKEN:?ZEABUR_API_TOKEN is required}"
: "${ZEABUR_ENV_ID:?ZEABUR_ENV_ID is required}"
[ "$#" -gt 0 ] || { echo "usage: $0 <serviceID>..." >&2; exit 2; }

fail=0
for svc in "$@"; do
  body=$(printf '{"query":"mutation($s: ObjectID!, $e: ObjectID!) { redeployService(serviceID: $s, environmentID: $e) }","variables":{"s":"%s","e":"%s"}}' "$svc" "$ZEABUR_ENV_ID")
  resp=$(curl -sS --max-time 60 --retry 2 https://api.zeabur.com/graphql \
    -H "Authorization: Bearer ${ZEABUR_API_TOKEN}" \
    -H 'Content-Type: application/json' \
    --data "$body") || { echo "service $svc: request failed" >&2; fail=1; continue; }
  if printf '%s' "$resp" | grep -q '"redeployService":true'; then
    echo "service $svc: redeploy triggered"
  else
    echo "service $svc: redeploy NOT triggered: $resp" >&2
    fail=1
  fi
done
exit "$fail"
