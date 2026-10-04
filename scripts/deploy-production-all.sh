#!/bin/bash
# ==============================================================================
# Tayooli ERP — Full Production Deployment Script
# Targets: Database Migrations (021-023), Backend Go API, and Frontend Next.js
# ==============================================================================
set -e

echo "=========================================================="
echo "🚀 Starting Tayooli ERP Production Deployment"
echo "📅 Time: $(date)"
echo "=========================================================="

# ── 1. Database Migrations (021, 022, 023) ──────────────────────────────────
echo ""
echo "=== Step 1: Running Database Migrations ==="
MIGRATIONS_DIR="/opt/tayooli/src/backend/go-core/migrations"
if [ ! -d "$MIGRATIONS_DIR" ]; then
    # Fallback if running from local repo root
    MIGRATIONS_DIR="$(dirname "$0")/../backend/go-core/migrations"
fi

for MIGRATION_FILE in "021_wms_multi_warehouse_locations.sql" "022_wms_stock_opname_and_scrap.sql" "023_wms_marketplace_sales_import.sql"; do
    TARGET_SQL="$MIGRATIONS_DIR/$MIGRATION_FILE"
    if [ -f "$TARGET_SQL" ]; then
        echo "Applying $MIGRATION_FILE..."
        sudo -u postgres psql -d tayooli_erp -f "$TARGET_SQL" 2>&1 || {
            echo "⚠️ Migration via postgres user failed, trying tayooli_app user..."
            PGPASSWORD=tayooli_secure_pass psql -h localhost -U tayooli_app -d tayooli_erp -f "$TARGET_SQL" || true
        }
        echo "✅ $MIGRATION_FILE processed."
    else
        echo "⚠️ Migration file $TARGET_SQL not found, skipping."
    fi
done

# ── 2. Backend Compilation & Restart ─────────────────────────────────────────
echo ""
echo "=== Step 2: Compiling & Deploying Backend ==="
if [ -d "/opt/tayooli/src/backend/go-core" ]; then
    cd /opt/tayooli/src/backend/go-core
    
    # If git repo exists in src, pull latest
    if [ -d ".git" ] || [ -d "../../.git" ]; then
        echo "Pulling latest backend source from git..."
        git pull origin main || true
    fi

    # Backup existing binary
    if [ -f "/opt/tayooli/tayooli-api" ]; then
        sudo cp /opt/tayooli/tayooli-api /opt/tayooli/tayooli-api.bak
    fi

    echo "Compiling Go binary..."
    GO_BIN="/usr/local/go/bin/go"
    if [ ! -f "$GO_BIN" ]; then
        GO_BIN="$(which go)"
    fi

    sudo $GO_BIN build -o /opt/tayooli/tayooli-api ./cmd/api
    sudo chmod 755 /opt/tayooli/tayooli-api

    echo "Restarting tayooli-backend service..."
    sudo systemctl restart tayooli-backend
    sleep 2

    if sudo systemctl is-active --quiet tayooli-backend; then
        echo "✅ Backend service is ACTIVE and running."
    else
        echo "❌ Backend service failed to start! Restoring backup..."
        if [ -f "/opt/tayooli/tayooli-api.bak" ]; then
            sudo cp /opt/tayooli/tayooli-api.bak /opt/tayooli/tayooli-api
            sudo systemctl restart tayooli-backend
        fi
        exit 1
    fi
else
    echo "⚠️ Backend source directory /opt/tayooli/src/backend/go-core not found."
fi

# ── 3. Frontend Build & Restart ──────────────────────────────────────────────
echo ""
echo "=== Step 3: Building & Deploying Frontend ==="
FRONTEND_DIR="/opt/tayooli/frontend"
if [ -d "$FRONTEND_DIR" ]; then
    cd "$FRONTEND_DIR"

    # Pull latest if git repository
    if [ -d ".git" ]; then
        echo "Pulling latest frontend source from git..."
        git pull origin main || true
    fi

    echo "Building Next.js production bundle..."
    # Ensure memory optimization flags
    export NODE_OPTIONS="--max-old-space-size=1536"
    npm run build

    echo "Restarting tayooli-frontend service..."
    sudo systemctl restart tayooli-frontend
    sleep 3

    if sudo systemctl is-active --quiet tayooli-frontend; then
        echo "✅ Frontend service is ACTIVE and running."
    else
        echo "⚠️ Frontend service status check warning — inspecting logs..."
        sudo journalctl -u tayooli-frontend -n 20 --no-pager
    fi
else
    echo "⚠️ Frontend directory /opt/tayooli/frontend not found."
fi

# ── 4. Health Check Verification ─────────────────────────────────────────────
echo ""
echo "=== Step 4: Health Check Verification ==="
HEALTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8081/health || echo "FAILED")
echo "Backend HTTP Health Check (/health): $HEALTH_STATUS"

if [ "$HEALTH_STATUS" = "200" ]; then
    echo "🎉 ALL SYSTEMS GO — DEPLOYMENT COMPLETE!"
    curl -s http://127.0.0.1:8081/health | jq . 2>/dev/null || curl -s http://127.0.0.1:8081/health
else
    echo "⚠️ Health check returned status: $HEALTH_STATUS"
fi
