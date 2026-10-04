# Tayooli ERP — Rollback Procedure

> **Tujuan:** panduan step-by-step untuk rollback deployment yang gagal, baik backend, frontend, maupun database.

---

## 1. Rollback Backend (Go API)

### Skenario: deploy binary baru → error / crash

```bash
# 1. Cek status backend saat ini
gcloud compute ssh tayooli-server --zone=us-central1-c --command="sudo systemctl status tayooli-backend --no-pager"

# 2. Lihat log error
gcloud compute ssh tayooli-server --zone=us-central1-c --command="sudo journalctl -u tayooli-backend --since '10 min ago' --no-pager"

# 3. Rollback ke binary sebelumnya
# Binary backup otomatis disimpan sebelum deploy:
gcloud compute ssh tayooli-server --zone=us-central1-c --command="
  sudo cp /opt/tayooli/tayooli-api.bak /opt/tayooli/tayooli-api
  sudo systemctl restart tayooli-backend
  sudo systemctl status tayooli-backend --no-pager
"

# 4. Verifikasi
curl -sf http://104.197.178.237:8081/health
```

### Pre-deploy checklist (sebelum deploy berikutnya)
```bash
# SELALU backup binary sebelum replace
gcloud compute ssh tayooli-server --zone=us-central1-c --command="
  sudo cp /opt/tayooli/tayooli-api /opt/tayooli/tayooli-api.bak
"
```

---

## 2. Rollback Frontend (Next.js)

### Skenario: deploy frontend baru → white screen / error

```bash
# 1. Cek log frontend
gcloud compute ssh tayooli-server --zone=us-central1-c --command="sudo journalctl -u tayooli-frontend --since '10 min ago' --no-pager"

# 2. Rollback ke build sebelumnya
gcloud compute ssh tayooli-server --zone=us-central1-c --command="
  cd /opt/tayooli/frontend
  # Jika pakai git:
  sudo git log --oneline -5
  sudo git checkout HEAD~1
  sudo npm run build
  sudo systemctl restart tayooli-frontend
"

# 3. Verifikasi
curl -sf http://104.197.178.237:3000
```

---

## 3. Rollback Database (PostgreSQL)

### Skenario: migration baru merusak data / schema

```bash
# 1. List backup yang tersedia
gcloud compute ssh tayooli-server --zone=us-central1-c --command="ls -lh /opt/tayooli/backups/"

# 2. Stop backend (mencegah write saat restore)
gcloud compute ssh tayooli-server --zone=us-central1-c --command="sudo systemctl stop tayooli-backend"

# 3. Restore dari backup terakhir yang bagus
# PERINGATAN: Ini MENGHAPUS semua data setelah backup dibuat!
gcloud compute ssh tayooli-server --zone=us-central1-c --command="
  BACKUP_FILE='/opt/tayooli/backups/tayooli_erp_YYYYMMDD_HHMMSS.sql.gz'
  
  # Drop dan recreate database
  sudo -u postgres psql -c 'DROP DATABASE IF EXISTS tayooli_erp_restore;'
  sudo -u postgres psql -c 'CREATE DATABASE tayooli_erp_restore OWNER tayooli_app;'
  
  # Restore
  gunzip -c \$BACKUP_FILE | sudo -u postgres pg_restore -d tayooli_erp_restore -Fc 2>/dev/null || \
  gunzip -c \$BACKUP_FILE | sudo -u postgres psql tayooli_erp_restore
  
  # Swap databases
  sudo -u postgres psql -c 'ALTER DATABASE tayooli_erp RENAME TO tayooli_erp_broken;'
  sudo -u postgres psql -c 'ALTER DATABASE tayooli_erp_restore RENAME TO tayooli_erp;'
  
  echo 'Database restored. Old broken DB available as tayooli_erp_broken'
"

# 4. Restart backend
gcloud compute ssh tayooli-server --zone=us-central1-c --command="
  sudo systemctl start tayooli-backend
  sudo systemctl status tayooli-backend --no-pager
"

# 5. Verifikasi data
curl -sf http://104.197.178.237:8081/health
```

### Setelah yakin restore berhasil:
```bash
# Hapus database broken
gcloud compute ssh tayooli-server --zone=us-central1-c --command="
  sudo -u postgres psql -c 'DROP DATABASE IF EXISTS tayooli_erp_broken;'
"
```

---

## 4. Full System Rollback (Nuclear Option)

Jika semua service bermasalah setelah perubahan besar:

```bash
# 1. Stop semua services
gcloud compute ssh tayooli-server --zone=us-central1-c --command="
  sudo systemctl stop tayooli-backend tayooli-frontend kafka
"

# 2. Restore DB dari backup
# (ikuti langkah Rollback Database di atas)

# 3. Restore backend binary
gcloud compute ssh tayooli-server --zone=us-central1-c --command="
  sudo cp /opt/tayooli/tayooli-api.bak /opt/tayooli/tayooli-api
"

# 4. Start semua services
gcloud compute ssh tayooli-server --zone=us-central1-c --command="
  sudo systemctl start kafka
  sleep 5
  sudo systemctl start tayooli-backend
  sudo systemctl start tayooli-frontend
  sudo systemctl status tayooli-backend tayooli-frontend kafka --no-pager
"

# 5. Verifikasi
curl -sf http://104.197.178.237:8081/health
```

---

## 5. Kubernetes Rollback (jika sudah migrasi ke K8s)

```bash
# Rollback deployment ke revision sebelumnya
kubectl rollout undo deployment/tayooli-api -n tayooli-core
kubectl rollout status deployment/tayooli-api -n tayooli-core --timeout=120s

# Atau rollback ke revision spesifik
kubectl rollout history deployment/tayooli-api -n tayooli-core
kubectl rollout undo deployment/tayooli-api -n tayooli-core --to-revision=<N>
```

---

## Quick Reference

| Masalah | Waktu Rollback | Risiko Data Loss |
|---|---|---|
| Backend crash setelah deploy | < 2 menit | Tidak ada |
| Frontend broken | < 5 menit | Tidak ada |
| Migration rusak (schema only) | < 10 menit | Tidak ada |
| Migration rusak (data corrupt) | 5-15 menit | Data setelah backup terakhir |
| Full system rollback | 15-30 menit | Data setelah backup terakhir |

---

*Terakhir diperbarui: 2026-09-05*
