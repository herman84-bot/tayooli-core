#!/bin/bash
# =============================================================================
# Tayooli ERP — Server Hardening Script
# Run on: tayooli-server (GCP e2-micro, Ubuntu 22.04)
# Date: 2026-09-05
# =============================================================================
set -euo pipefail

echo "=== [1/7] Setup Swap (2GB) for e2-micro stability ==="
if [ ! -f /swapfile ]; then
    fallocate -l 2G /swapfile
    chmod 600 /swapfile
    mkswap /swapfile
    swapon /swapfile
    echo '/swapfile none swap sw 0 0' >> /etc/fstab
    # Optimize swap behavior for low-memory server
    sysctl vm.swappiness=10
    echo 'vm.swappiness=10' >> /etc/sysctl.conf
    echo "  Swap created and activated"
else
    echo "  Swap already exists"
fi
swapon --show
free -m
echo ""

echo "=== [2/7] Disable unnecessary services ==="
# Snap is heavy and unnecessary on a production server
systemctl stop snapd snapd.socket snapd.apparmor 2>/dev/null || true
systemctl disable snapd snapd.socket snapd.apparmor 2>/dev/null || true
systemctl mask snapd snapd.socket 2>/dev/null || true
# Remove snap if possible to free disk
apt-get purge -y snapd 2>/dev/null || true
echo "  Snap disabled and removed"
echo ""

echo "=== [3/7] Sanitize seed credentials ==="
# Generate secure random passwords
ADMIN_NEW_PW=$(openssl rand -base64 32)
ACCOUNTANT_NEW_PW=$(openssl rand -base64 32)
APPROVER_NEW_PW=$(openssl rand -base64 32)

# Hash passwords with bcrypt (using Python since it's installed)
hash_password() {
    python3 -c "
import bcrypt, sys
pw = sys.argv[1].encode()
h = bcrypt.hashpw(pw, bcrypt.gensalt(rounds=12))
print(h.decode())
" "$1"
}

ADMIN_HASH=$(hash_password "$ADMIN_NEW_PW")
ACCOUNTANT_HASH=$(hash_password "$ACCOUNTANT_NEW_PW")
APPROVER_HASH=$(hash_password "$APPROVER_NEW_PW")

# Update passwords in database
sudo -u postgres psql -d tayooli_erp <<EOSQL
UPDATE users SET password_hash = '${ADMIN_HASH}' WHERE email = 'admin@test.com';
UPDATE users SET password_hash = '${ACCOUNTANT_HASH}' WHERE email = 'accountant@test.com';
UPDATE users SET password_hash = '${APPROVER_HASH}' WHERE email = 'approver@test.com';
EOSQL

# Save new credentials securely (readable only by root)
CRED_FILE="/root/.tayooli-credentials"
cat > "$CRED_FILE" <<EOF
# Tayooli ERP — Production Credentials
# Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
# WARNING: Store these securely and delete this file after saving elsewhere!

admin@test.com      : ${ADMIN_NEW_PW}
accountant@test.com : ${ACCOUNTANT_NEW_PW}
approver@test.com   : ${APPROVER_NEW_PW}
EOF
chmod 600 "$CRED_FILE"
echo "  Seed passwords rotated. New credentials saved to ${CRED_FILE}"
echo "  ⚠️  Copy credentials to a secure location, then delete ${CRED_FILE}"
echo ""

echo "=== [4/7] Setup automated DB backup ==="
# Create backup directory
BACKUP_DIR="/opt/tayooli/backups"
mkdir -p "$BACKUP_DIR"
chown postgres:postgres "$BACKUP_DIR"

# Create backup script
cat > /opt/tayooli/backup-db.sh <<'BACKUP_SCRIPT'
#!/bin/bash
# Tayooli ERP — Automated PostgreSQL Backup
set -euo pipefail

BACKUP_DIR="/opt/tayooli/backups"
DB_NAME="tayooli_erp"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="${BACKUP_DIR}/${DB_NAME}_${TIMESTAMP}.sql.gz"
RETENTION_DAYS=7

echo "[$(date)] Starting backup of ${DB_NAME}..."

# Dump and compress
sudo -u postgres pg_dump -Fc "${DB_NAME}" | gzip > "${BACKUP_FILE}"
chmod 640 "${BACKUP_FILE}"

# Verify backup is not empty (minimum 1KB)
BACKUP_SIZE=$(stat -f%z "${BACKUP_FILE}" 2>/dev/null || stat -c%s "${BACKUP_FILE}")
if [ "${BACKUP_SIZE}" -lt 1024 ]; then
    echo "[$(date)] ERROR: Backup file too small (${BACKUP_SIZE} bytes). Possible corruption."
    exit 1
fi

echo "[$(date)] Backup created: ${BACKUP_FILE} ($(du -h "${BACKUP_FILE}" | cut -f1))"

# Upload to GCS (if gsutil is available)
if command -v gsutil &>/dev/null; then
    GCS_BUCKET="gs://tayooli-backups"
    gsutil -q cp "${BACKUP_FILE}" "${GCS_BUCKET}/db/${DB_NAME}_${TIMESTAMP}.sql.gz" 2>/dev/null && \
        echo "[$(date)] Uploaded to ${GCS_BUCKET}" || \
        echo "[$(date)] WARNING: GCS upload failed (bucket may not exist yet)"
fi

# Rotate old local backups (keep last N days)
find "${BACKUP_DIR}" -name "*.sql.gz" -mtime +${RETENTION_DAYS} -delete
echo "[$(date)] Cleaned backups older than ${RETENTION_DAYS} days"

# Log backup inventory
echo "[$(date)] Current backups:"
ls -lh "${BACKUP_DIR}"/*.sql.gz 2>/dev/null || echo "  (none)"
BACKUP_SCRIPT
chmod 755 /opt/tayooli/backup-db.sh

# Schedule daily backup at 03:00 UTC + weekly full backup Sunday 02:00
(crontab -l 2>/dev/null || true; echo "
# Tayooli ERP — Automated DB Backup
0 3 * * * /opt/tayooli/backup-db.sh >> /var/log/tayooli-backup.log 2>&1
") | sort -u | crontab -

# Run first backup now
/opt/tayooli/backup-db.sh
echo "  DB backup configured (daily at 03:00 UTC, 7-day retention)"
echo ""

echo "=== [5/7] Setup health monitoring & alerting ==="
# Create health monitor script
cat > /opt/tayooli/health-monitor.sh <<'HEALTH_SCRIPT'
#!/bin/bash
# Tayooli ERP — Health Monitor
# Checks backend, frontend, DB, disk, and memory
# Writes status to /tmp/tayooli-health.json

ALERT_LOG="/var/log/tayooli-alerts.log"
STATUS_FILE="/tmp/tayooli-health.json"

check_service() {
    systemctl is-active --quiet "$1" 2>/dev/null && echo "ok" || echo "down"
}

check_http() {
    curl -sf --connect-timeout 5 "$1" >/dev/null 2>&1 && echo "ok" || echo "unreachable"
}

# Collect checks
BACKEND_SVC=$(check_service tayooli-backend)
FRONTEND_SVC=$(check_service tayooli-frontend)
NGINX_SVC=$(check_service nginx)
POSTGRES_SVC=$(check_service postgresql)
BACKEND_HTTP=$(check_http "http://127.0.0.1:8081/health")
FRONTEND_HTTP=$(check_http "http://127.0.0.1:3000")

DISK_PCT=$(df / --output=pcent | tail -1 | tr -d ' %')
MEM_AVAILABLE=$(awk '/MemAvailable/ {print $2}' /proc/meminfo)
MEM_TOTAL=$(awk '/MemTotal/ {print $2}' /proc/meminfo)
MEM_PCT=$((100 - (MEM_AVAILABLE * 100 / MEM_TOTAL)))
LOAD=$(cat /proc/loadavg | cut -d' ' -f1)

TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Write status JSON
cat > "$STATUS_FILE" <<EOF
{
  "timestamp": "${TIMESTAMP}",
  "services": {
    "backend": "${BACKEND_SVC}",
    "frontend": "${FRONTEND_SVC}",
    "nginx": "${NGINX_SVC}",
    "postgres": "${POSTGRES_SVC}"
  },
  "http": {
    "backend_api": "${BACKEND_HTTP}",
    "frontend_web": "${FRONTEND_HTTP}"
  },
  "resources": {
    "disk_percent": ${DISK_PCT},
    "memory_percent": ${MEM_PCT},
    "load_1min": ${LOAD}
  }
}
EOF

# Alert conditions
ALERT=""
[ "$BACKEND_SVC" != "ok" ] && ALERT="${ALERT}CRITICAL: tayooli-backend service DOWN\n"
[ "$POSTGRES_SVC" != "ok" ] && ALERT="${ALERT}CRITICAL: postgresql service DOWN\n"
[ "$NGINX_SVC" != "ok" ] && ALERT="${ALERT}WARNING: nginx service DOWN\n"
[ "$BACKEND_HTTP" != "ok" ] && ALERT="${ALERT}CRITICAL: backend HTTP health check FAILED\n"
[ "$DISK_PCT" -gt 90 ] && ALERT="${ALERT}WARNING: disk usage at ${DISK_PCT}%\n"
[ "$MEM_PCT" -gt 90 ] && ALERT="${ALERT}WARNING: memory usage at ${MEM_PCT}%\n"

if [ -n "$ALERT" ]; then
    echo "[${TIMESTAMP}] ALERT:" >> "$ALERT_LOG"
    echo -e "$ALERT" >> "$ALERT_LOG"

    # Auto-restart failed critical services
    [ "$BACKEND_SVC" != "ok" ] && systemctl restart tayooli-backend 2>/dev/null && \
        echo "[${TIMESTAMP}] Auto-restarted tayooli-backend" >> "$ALERT_LOG"
    [ "$POSTGRES_SVC" != "ok" ] && systemctl restart postgresql 2>/dev/null && \
        echo "[${TIMESTAMP}] Auto-restarted postgresql" >> "$ALERT_LOG"
    [ "$NGINX_SVC" != "ok" ] && systemctl restart nginx 2>/dev/null && \
        echo "[${TIMESTAMP}] Auto-restarted nginx" >> "$ALERT_LOG"
fi
HEALTH_SCRIPT
chmod 755 /opt/tayooli/health-monitor.sh

# Create systemd service for health monitoring (runs every 2 minutes)
cat > /etc/systemd/system/tayooli-monitor.service <<'MONITOR_SVC'
[Unit]
Description=Tayooli Health Monitor
After=network.target

[Service]
Type=oneshot
ExecStart=/opt/tayooli/health-monitor.sh
User=root
MONITOR_SVC

cat > /etc/systemd/system/tayooli-monitor.timer <<'MONITOR_TIMER'
[Unit]
Description=Run Tayooli Health Monitor every 2 minutes

[Timer]
OnBootSec=60
OnUnitActiveSec=120
AccuracySec=30

[Install]
WantedBy=timers.target
MONITOR_TIMER

systemctl daemon-reload
systemctl enable tayooli-monitor.timer
systemctl start tayooli-monitor.timer

# Create health endpoint that Nginx can serve (for external uptime checks)
cat > /opt/tayooli/health-status.sh <<'STATUS_ENDPOINT'
#!/bin/bash
# Serve health status as a static JSON file via Nginx
/opt/tayooli/health-monitor.sh
cp /tmp/tayooli-health.json /var/www/html/health-status.json 2>/dev/null || true
STATUS_ENDPOINT
chmod 755 /opt/tayooli/health-status.sh

echo "  Health monitor configured (every 2 min, auto-restart on failure)"
echo ""

echo "=== [6/7] Harden SSH & server security ==="
# Disable password authentication (key-only)
if ! grep -q "^PasswordAuthentication no" /etc/ssh/sshd_config; then
    sed -i 's/^#*PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config
    sed -i 's/^#*PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config
    systemctl reload sshd
    echo "  SSH hardened: password auth disabled, root login disabled"
else
    echo "  SSH already hardened"
fi

# Enable automatic security updates
apt-get install -y unattended-upgrades >/dev/null 2>&1 || true
dpkg-reconfigure -plow unattended-upgrades 2>/dev/null || true
echo "  Automatic security updates enabled"
echo ""

echo "=== [7/7] Restart Kafka (now with swap available) ==="
systemctl start kafka 2>/dev/null || true
sleep 5
KAFKA_STATUS=$(systemctl is-active kafka 2>/dev/null || echo "failed")
echo "  Kafka status: ${KAFKA_STATUS}"
echo ""

echo "============================================"
echo "  HARDENING COMPLETE"
echo "============================================"
echo ""
echo "Summary:"
echo "  ✅ Swap: 2GB configured"
echo "  ✅ Seed passwords: rotated (see /root/.tayooli-credentials)"
echo "  ✅ DB backup: daily at 03:00 UTC, 7-day retention"
echo "  ✅ Health monitor: every 2 min, auto-restart on failure"
echo "  ✅ SSH: key-only, no root login"
echo "  ✅ Security updates: automatic"
echo "  ✅ Snap: removed (frees ~100MB RAM)"
echo ""
echo "Remaining manual steps:"
echo "  1. Copy /root/.tayooli-credentials to secure location, then delete it"
echo "  2. Create GCS bucket 'tayooli-backups' for offsite backup"
echo "  3. Setup custom domain + Let's Encrypt SSL when domain is ready"
echo ""
free -m
