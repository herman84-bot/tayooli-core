# 🚀 Tayooli ERP — Go-Live Checklist

> **Tujuan:** daftar periksa item-by-item dari kode siap (≈100% fitur) menuju **go-live resmi**.
> Sumber utama: [`production-runbook.md`](production-runbook.md) (§2 env var, §4 deploy, §6 observability, §7 smoke test, §8 checklist, §9 rollback).
> Cara pakai: centang item, catat tanggal/oleh siapa di kolom note. **Jangan** commit secret apa pun.

**Target go-live:** `____ / ____ / ____` · **PIC:** `__________`

**Status ringkas:** 🔴 Belum mulai · 🟡 Sebagian · 🟢 Selesai
| Fase | Status |
|---|---|
| 0. Inventaris & konteks | 🟡 |
| 1. Secrets & konfigurasi | 🟡 |
| 2. Infrastruktur & keamanan | 🟡 |
| 3. Deployment & CI/CD | 🟡 |
| 4. Data & migrasi | 🟡 |
| 5. Monitoring & observability | 🟡 |
| 6. Smoke test end-to-end | 🟡 |
| 7. Cutover go-live | 🔴 |
| 8. Rollback & hypercare | 🔴 |

---

## Fase 0 — Inventaris & Konteks

- [x] Konfirmasi arsitektur target: Frontend Next.js + Backend Go + AI Worker Python + Postgres 15 + Kafka + Redis (lihat `production-runbook.md` §1)
- [x] Konfirmasi server produksi berjalan: VM GCP `tayooli-server` (`104.197.178.237`, `mbg-waste-tracker-503014`, us-central1-c) — **live sejak lama, health OK**
- [ ] Tentukan **opsi deploy** final per komponen (A: docker-compose / B: K8s / C: VM+GCP yang sudah jalan) — campuran boleh, tapi tulis di sini:
  - Frontend: `__________`
  - Backend Go: `__________`
  - AI Worker: `__________`
  - DB/Kafka/Redis: `__________`
- [ ] Tentukan **domain produksi** (contoh: `erp.tayooli.id`) — wajib untuk SSL resmi
- [ ] Buat daftar kontak on-call & war room (channel darurat, PIC deploy, PIC infra, PIC DB)

## Fase 1 — Secrets & Konfigurasi (WAJIB diisi operator, jangan di-commit)

### Backend Go (`backend/go-core`)
- [ ] `JWT_SECRET` — acak ≥ 32 char: `openssl rand -base64 48` → simpan di secret manager / env produksi
- [ ] `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` (kuat), `DB_NAME`, `DB_SSLMODE=require` — SSLMODE saat ini masih `disable` di compose (dev)
- [ ] `KAFKA_BROKERS` → broker produksi (mis. `localhost:9092` atau in-cluster)
- [ ] `FRONTEND_ORIGIN` — origin frontend produksi (CORS whitelist)
- [ ] `APP_ENV=production` — log zerolog JSON
- [ ] `AI_BASE_URL` → URL AI worker (untuk inference HTTP)
- [ ] `SENTRY_DSN` (opsional tapi disarankan) — error tracking
- [ ] `REDIS_URL` (opsional, jika Celery async dipakai)

### Frontend (Next.js)
- [ ] `NEXT_PUBLIC_API_URL` **dan/atau** `BACKEND_URL` → URL backend produksi (pastikan **bukan** `localhost:8081`)
- [ ] `AUTH_DEMO` **tidak diset** (atau `false`) di produksi asli — demo auth hanya untuk preview
- [ ] `GEMINI_API_KEY` — asisten AI "Nara" (opsional; tanpa ini Nara pesan ramah)
- [ ] `APP_URL` → URL publik frontend

### AI Worker (Python)
- [ ] `KAFKA_BROKERS` — consumer `invoice.created` / producer `invoice.ocr_completed`
- [ ] `REDIS_URL` / `CELERY_BROKER` (opsional; tanpa ini fallback sinkron)

### Payment Gateway (Pakasir & Midtrans)
- [ ] `PAKASIR_API_KEY` (+ `PAKASIR_PROJECT_SLUG`, `PAKASIR_BASE_URL`) — produksi, **bukan sandbox**
- [ ] `MIDTRANS_SERVER_KEY` & `MIDTRANS_CLIENT_KEY`, set `MIDTRANS_IS_PRODUCTION=true`
- [ ] Config **per-tenant** di UI `/dashboard/payment-gateways` (Model B) — diverifikasi tiap tenant yang mau go-live
- [ ] IP/webhook endpoint dicatat ke dashboard Pakasir/Midtrans: `/api/webhooks/pakasir` & `/api/webhooks/midtrans`

### CI/CD Secrets (GitHub, environment `production`)
- [ ] `GCR_SA_KEY` — service account push image ke GCR
- [ ] `KUBECONFIG` — akses cluster (jika pakai Opsi B)
- [ ] `CODECOV_TOKEN` — coverage reporting

## Fase 2 — Infrastruktur & Keamanan

### Domain & SSL (gap terbesar saat ini)
- [ ] Beli/daftarkan **domain** (mis. `tayooli.id`) & setup DNS: A record → `104.197.178.237`
- [ ] Konfigurasi Nginx `server_name` → domain baru (saat ini masih IP)
- [ ] **Ganti self-signed SSL dengan sertifikat resmi** (Let's Encrypt via certbot + auto-renew timer):
  ```bash
  sudo apt install certbot python3-certbot-nginx
  sudo certbot --nginx -d erp.tayooli.id --redirect
  sudo systemctl list-timers | grep certbot   # pastikan auto-renew aktif
  ```
- [ ] Verifikasi HTTP → HTTPS redirect & tidak ada sertifikat kedaluwarsa (test di browser + `curl -vI https://<domain>/health`)

### Firewall & Hardening
- [ ] Firewall GCP: hanya buka 80/443 publik; **8081 (backend) dan 5432/9092 jangan diekspos publik** — akses via internal/VPC atau reverse proxy saja
- [ ] Port 3000 (frontend dev) tidak terekspos publik — akses via Nginx saja
- [ ] PostgreSQL hanya bind ke localhost/internal, password kuat, `pg_hba.conf` dibatasi
- [ ] Kafka tidak terekspos publik
- [ ] Backend jalan **non-root** (UID non-zero) + filesystem read-only (sudah ada di Dockerfile/K8s — verifikasi di VM juga)
- [ ] `npm audit` / `go vulncheck` / `pip-audit` — tidak ada critical vulnerability
- [ ] Rate limiter backend aktif (sudah ada di `main.go` — konfirmasi threshold sesuai kebutuhan)
- [ ] Secret tidak ada di repo/git history: `git grep -i "password\|secret\|api_key" -- "*.env*"` + pindahkan `secret.yaml` keluar dari git (CI sudah eksklusi)

## Fase 3 — Deployment & CI/CD

- [ ] Branch kerja final (`feat/remaining-25-percent`) sudah **di-merge ke `main`** (HEAD saat ini sudah `main` — verifikasi semua modul masuk)
- [ ] Pipeline CI hijau di `main` terakhir: test-go, test-frontend, test-ai-worker, e2e (Playwright), build → GCR
- [ ] Image backend & ai-worker ter-build dengan tag `sha` + `latest` di GCR
- [ ] Jika pakai **K8s**: secret riil dibuat out-of-band (CI tidak pernah apply secret):
  ```bash
  kubectl create secret generic tayooli-api-secrets -n tayooli-core \
    --from-literal=DB_USER=tayooli \
    --from-literal=DB_PASSWORD='<STRONG>' \
    --from-literal=JWT_SECRET="$(openssl rand -base64 48)"
  ```
- [ ] Jika pakai **VM/systemd**: binary baru (`/opt/tayooli/tayooli-api`) di-update + `systemctl restart tayooli-backend` tanpa downtime (atau rolling)
- [ ] Staging environment (jika ada) di-deploy **sebelum** produksi — smoke test dulu di sana
- [ ] Pipeline `deploy` job berjalan sukses di `main` (dry-run → pin image → apply → rollout) — atau deploy manual terverifikasi
- [ ] `kubectl rollout status deployment/tayooli-api -n tayooli-core` (jika K8s) / `systemctl status tayooli-backend tayooli-frontend kafka` (jika VM)

## Fase 4 — Data & Migrasi

- [ ] Migrasi **001 → 013** dijalankan semua berurutan di DB produksi (file seed E2E opsional)
- [ ] Password **seed users diganti atau dihapus** — `admin@test.com` / `password123` **TIDAK boleh ada di produksi**:
  ```sql
  -- ganti password / nonaktifkan user seed (12x, 13x) di produksi
  UPDATE users SET password_hash = crypt('<BARU>', gen_salt('bf')) WHERE email LIKE '%@test.com';
  ```
- [ ] Tenant produksi pertama dibuat + admin/accountant/approver asli (bukan seed) diverifikasi bisa login
- [ ] Verifikasi isolasi tenant: user tenant A **tidak bisa** melihat data tenant B (uji `/api/v1/invoices` lintas token)
- [ ] Backup otomatis aktif: timer `tayooli-backup.timer` (pg_dump harian → `/opt/tayooli/backups/`, retensi 7 hari) — sudah ada di VM, **verifikasi log & hasilnya**
- [ ] **Restore backup pernah diuji** (paling penting!): restore ke DB uji + smoke test, catat waktu RTO/RPO:
  - RTO target: `____ menit` · RPO target: `____`
- [ ] Backup ke **lokasi terpisah** (bucket/region lain) — jangan hanya satu VM

## Fase 5 — Monitoring & Observability

- [ ] `/health` & `/health/ready` (Go) dan `/health` (AI worker) merespons 200 di produksi
- [ ] `/metrics` (:9090) di-scrape Prometheus (atau minimal dicek: `curl localhost:9090/metrics | head`)
- [ ] Alerting dasar aktif: backend down, DB down, disk penuh, health check gagal → notifikasi (email/Telegram/Slack)
- [ ] Timer monitoring `tayooli-monitor.timer` (cek 5 menit + auto-restart) — sudah ada di VM, verifikasi log
- [ ] Log aggregation: `journalctl -u tayooli-backend` dikumpulkan terpusat (Loki/Cloud Logging/opsional) — belum ada, disarankan
- [ ] Uptime check eksternal (UptimeRobot/StatusCake dsb.) → domain produksi
- [ ] Sentry error tracking aktif (DSN terpasang, test kirim error)
- [ ] Dashboard dasar (Grafana/opsional) — CPU, memori, disk, DB connections, Kafka lag

## Fase 6 — Smoke Test End-to-End (di environment produksi)

### Backend
- [ ] `curl -sf https://<domain>/health` dan `/health/ready` → 200
- [ ] Login API: `POST /api/v1/auth/login` + `GET /api/v1/auth/me` dengan user produksi

### Frontend (jalankan manual sebagai user produksi)
- [ ] Login → dashboard → buat invoice → approval flow (approved oleh approver)
- [ ] **AI analysis** muncul (anomaly badge) — butuh AI worker + `GEMINI_API_KEY`
- [ ] O2C penuh: Customer → Sales Order → Sales Invoice → Payment Order → approve → pay
- [ ] Payment gateway: config per-tenant aktif, checkout Pakasir (QRIS/VA) & Midtrans Snap (mode **sandbox** dulu, lalu produksi)
- [ ] Webhook: `/api/webhooks/pakasir` & `/api/webhooks/midtrans` menerima payload test → status invoice berubah ke PAID
- [ ] Products/Inventory: buat product, adjust stock, lihat stok akurat

### Data Flow
- [ ] Kafka: buat invoice → AI worker (consumer `invoice.created`) proses OCR → status invoice berubah → event `invoice.ocr_completed` consumed
- [ ] Dashboard analytics (`/api/v1/dashboard/summary`) menampilkan angka sesuai aksi di atas

### Non-Fungsional (sebelum go-live resmi)
- [ ] Load test ringan (k6/artillery): ≥ `____` request/s login+list invoice tanpa error & p95 < `____` ms
- [ ] Uji concurrent: 2 user dari tenant berbeda operasi bersamaan, tidak ada kebocoran data
- [ ] Uji 3-way match tolerance (`MATCH_AMOUNT_TOLERANCE_PCT`) di skenario mismatch

## Fase 7 — Cutover Go-Live

- [ ] Semua Fase 1–6 centang hijau (atau risiko tersisa didokumentasikan + disetujui)
- [ ] `JWT_SECRET` final dipasang (tidak berubah lagi — ganti secret = semua sesi logout)
- [ ] DNS propagasi selesai (`dig <domain> +short` → `104.197.178.237`)
- [ ] HTTPS valid di `https://<domain>` (bukan self-signed), auto-renew berjalan
- [ ] Smoke test §6 **lulus di produksi** dengan data riil terakhir
- [ ] Backup manual terakhir sebelum cutover: `pg_dump` + verifikasi file
- [ ] Traffic diarahkan penuh ke produksi (hapus mode demo/preview)
- [ ] Pengumuman go-live ke stakeholder + dokumentasi akun/akses internal
- [ ] Semua akses darurat (SSH keys, service accounts) tercatat & dibatasi

## Fase 8 — Rollback & Hypercare

### Rollback (harus sudah diuji, bukan cuma ditulis)
- [ ] K8s: `kubectl rollout undo deployment/tayooli-api -n tayooli-core` — **teruji**
- [ ] Compose: `docker compose pull && docker compose up -d` dengan tag lama — **teruji**
- [ ] DB: restore dari backup terakhir — **teruji** (catatan: migrasi ke belakang tidak didukung, blok DOWN; jangan drop kolom manual)
- [ ] Runbook §9 dibaca & dipahami oleh ≥ 2 orang

### Hypercare (48–72 jam pasca go-live)
- [ ] Monitoring dipantau setiap 4 jam; alert on-call aktif
- [ ] Log error (Sentry/journal) dicek harian; bug prioritas di-fix via hotfix branch
- [ ] Backup harian diverifikasi berjalan
- [ ] Review performa (DB connections, Kafka lag, response time) setelah beban riil pertama
- [ ] Retro go-live: catat temuan, update runbook & checklist ini

---

## Catatan Kritis (dari runbook)
1. **Migrasi idempotent** — jalankan semua file `.sql` 001→013 berurutan; ada nomor duplikat (`011_*`, `012_*`, `013_*`) — jangan lewati.
2. **Rollback DB tidak didukung ke belakang** — migrasi ke depan idempotent, DOWN di-blok.
3. **Demo auth mati di produksi** — tanpa `NEXT_PUBLIC_API_URL`/`BACKEND_URL`, login mengembalikan `503 auth backend not configured` (by design).
4. **CI tidak pernah apply secret** — secret selalu out-of-band.
5. **Freebuff hosting hanya frontend** — Go/Python tidak bisa di-build di sana; backend + AI worker wajib di-host terpisah (VM/Docker/K8s).