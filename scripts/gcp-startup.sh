#!/bin/bash
# Tayooli ERP — GCP Startup Script
# Runs on every instance boot. Idempotent.
# HARDENED: waits for DNS/network, retries GCS download until success,
# applies idempotent DB migrations, restarts backend after Kafka is up.

LOG=/tmp/startup.log
exec > $LOG 2>&1

# 0a. Keep prior hardening: stop snapd noise and refresh sshd (idempotent)
systemctl stop snapd snapd.socket 2>/dev/null || true
systemctl disable snapd snapd.socket 2>/dev/null || true
systemctl restart sshd 2>/dev/null || true

echo "=== Startup script started at $(date) ==="

# 0. Wait for network + DNS (startup scripts often race DNS on boot)
for i in $(seq 1 30); do
  if curl -sI --max-time 5 https://storage.googleapis.com >/dev/null 2>&1; then
    echo "Network/DNS ready after ${i} tries."
    break
  fi
  echo "Waiting for network... (${i}/30)"
  sleep 5
done

# 1. System dependencies
apt-get update -qq 2>/dev/null || true
apt-get install -y -qq python3-pip tesseract-ocr poppler-utils curl postgresql-client 2>/dev/null || true

# 2. AI Worker setup
mkdir -p /opt/tayooli/ai-worker
if [ -f /tmp/ai-worker.tar.gz ]; then
  tar -xzf /tmp/ai-worker.tar.gz -C /opt/tayooli/ai-worker
fi

cd /opt/tayooli/ai-worker
pip3 install --break-system-packages -q \
  fastapi==0.115.6 uvicorn[standard]==0.34.0 pydantic==2.8.2 \
  kafka-python==2.0.2 pytesseract==0.3.13 pdf2image==1.17.0 \
  Pillow==10.4.0 celery[redis]==5.4.0 redis==5.2.1 2>/dev/null || true

# Create AI worker service
cat > /etc/systemd/system/tayooli-ai-worker.service << 'UNIT'
[Unit]
Description=Tayooli AI Worker (FastAPI + OCR)
After=network.target kafka.service
Wants=kafka.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/tayooli/ai-worker
Environment=PYTHONUNBUFFERED=1
Environment=KAFKA_BROKERS=localhost:9092
ExecStart=/usr/bin/python3 -m uvicorn service:app --host 0.0.0.0 --port 8000
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable tayooli-ai-worker 2>/dev/null || true
systemctl restart tayooli-ai-worker 2>/dev/null || true

# 3. Deploy latest binary from GCS (retry until success)
DEPLOYED=0
for i in $(seq 1 12); do
  if curl -sSL --max-time 120 -o /opt/tayooli/tayooli-api-new \
    "https://storage.googleapis.com/tayooli-deployments/tayooli-api" 2>/dev/null; then
    chmod 755 /opt/tayooli/tayooli-api-new
    cp /opt/tayooli/tayooli-api /opt/tayooli/tayooli-api-old 2>/dev/null || true
    mv /opt/tayooli/tayooli-api-new /opt/tayooli/tayooli-api
    echo "Latest binary deployed from GCS (attempt ${i})."
    DEPLOYED=1
    break
  fi
  echo "GCS download attempt ${i}/12 FAILED. Retrying in 10s..."
  sleep 10
done

if [ "$DEPLOYED" = "0" ]; then
  echo "GCS download FAILED after all attempts. Keeping existing binary."
fi

# 4. Apply idempotent DB migrations (safe to run on every boot)
ENV_FILE=/opt/tayooli/.env
dbhost=$(grep -E '^DB_HOST=' "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2-)
dbport=$(grep -E '^DB_PORT=' "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2-)
dbuser=$(grep -E '^DB_USER=' "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2-)
dbpass=$(grep -E '^DB_PASSWORD=' "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2-)
dbname=$(grep -E '^DB_NAME=' "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2-)
# DDL on the users table requires table ownership — the app user cannot run it.
# Run as the postgres superuser when available; fall back to the app user.
if [ -n "$dbname" ]; then
  MIGRATE_SQL="
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS password_reset_token TEXT,
  ADD COLUMN IF NOT EXISTS password_reset_expires_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_users_password_reset_token
  ON users (password_reset_token) WHERE password_reset_token IS NOT NULL;
"
  if command -v sudo >/dev/null 2>&1 && sudo -n -u postgres true 2>/dev/null; then
    echo "$MIGRATE_SQL" | sudo -u postgres psql -d "$dbname" -v ON_ERROR_STOP=1 >/tmp/migrate018.log 2>&1
  else
    PGPASSWORD="$dbpass" psql -h "$dbhost" -p "${dbport:-5432}" -U "$dbuser" -d "$dbname" -v ON_ERROR_STOP=1 -c "$MIGRATE_SQL" >/tmp/migrate018.log 2>&1
  fi
  if [ $? -eq 0 ]; then
    echo "Migration 018 (password reset) applied/verified."
    logger "STARTUP: migration_018 OK"
  else
    echo "Migration 018 FAILED — see /tmp/migrate018.log"
    logger "STARTUP: migration_018 FAILED"
    tail -5 /tmp/migrate018.log > /dev/console 2>/dev/null || true
  fi
else
  echo "WARN: DB_NAME not found in /opt/tayooli/.env — migration skipped."
  logger "STARTUP: migration_018 SKIPPED (no DB_NAME env)"
fi

# 5. Restart backend (wait for Kafka to come up first)
(sleep 45 && systemctl restart tayooli-backend && echo "Backend restarted after Kafka window." && sleep 6 && \
  code=$(curl -s -o /dev/null -w '%{http_code}' -X POST http://localhost:8081/api/v1/auth/forgot-password -H 'Content-Type: application/json' -d '{"email":"admin@test.com"}') && \
  logger "STARTUP: forgot-password route check HTTP=$code" && echo "Route check HTTP=$code") &

echo "=== Startup script completed at $(date) ==="
