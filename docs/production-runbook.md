# Tayooli ERP — Production Runbook

> **Tujuan dokumen:** panduan langkah-demi-langkah untuk membawa Tayooli ERP dari kode (≈100% fitur) ke *go-live*. Dokumen ini adalah sumber kebenaran untuk operasional produksi. Untuk status pengembangan, lihat `PROJECT_STATUS.md`.

**Audit tanggal:** 2026-08-12 · **Status:** kode siap, operasional belum (lihat §8 Checklist Go-Live).

---

## 1. Arsitektur & Komponen

| Komponen | Teknologi | Lokasi | Port (default) |
|---|---|---|---|
| Frontend | Next.js 15 (App Router), TS, Tailwind, Zustand, TanStack Query | Repo root (`app/`, `components/`, `hooks/`, `lib/`) | 3000 |
| Backend API | Go (Chi, hexagonal), database/sql (NO ORM) | `backend/go-core` | 8081 (+ metrics 9090) |
| AI Worker | Python FastAPI + Celery + Tesseract OCR | `ai-worker` | 8000 |
| Database | PostgreSQL 15 (Row-Level Security) | — | 5432 |
| Broker | Apache Kafka (KRaft mode) | — | 9092 (host) / 29092 (in-cluster) |
| Redis | Broker Celery (opsional; ada fallback sinkron) | — | 6379 |

Alur inti: Frontend memproksi `/api/v1/*` server-side ke Go API → Go publish event Kafka (`invoice.created`, dst.) → AI Worker OCR → publish `invoice.ocr_completed` → Go update status + 3-way match → approval flow → payment order → payment gateway (Pakasir/Midtrans).

---

## 2. Env Var Matrix

### 2.1 Frontend (Next.js)

| Variabel | Wajib? | Fungsi |
|---|---|---|
| `NEXT_PUBLIC_API_URL` **atau** `API_BASE_URL` | Opsional (lihat catatan) | Base URL backend Go. **Kosong = demo mode** (hanya aktif di non-produksi). Di produksi tanpa ini, login mengembalikan `503 auth backend not configured` |
| `BACKEND_URL` | Opsional | Target proxy catch-all `app/api/v1/[...path]/route.ts` (default `http://localhost:8081`). Set ke URL backend produksi |
| `GEMINI_API_KEY` | Opsional | Asisten AI "Nara" (`app/api/gemini/nara/route.ts`). Tanpa ini Nara menampilkan pesan ramah, app tetap jalan |
| `APP_URL` | Opsional | URL host untuk link self-referential (OAuth callback, dll.) |
| `AUTH_DEMO=true` | Khusus demo | Mengaktifkan demo auth di produksi. **Jangan set di produksi asli** — hanya untuk preview/demo publik |
| `AUTH_DEMO_SECRET` | Opsional | HMAC secret sesi demo. Tanpa ini secret acak per-boot (sesi invalid saat restart) |
| `PAKASIR_PROJECT_SLUG`, `PAKASIR_API_KEY`, `PAKASIR_BASE_URL`, `PAKASIR_API_BASE_URL` | Opsional | Fallback global gateway Pakasir (Model B: per-tenant lebih disarankan, dikelola via UI `/dashboard/payment-gateways`) |
| `MIDTRANS_SERVER_KEY`, `MIDTRANS_CLIENT_KEY`, `MIDTRANS_IS_PRODUCTION` | Opsional | Fallback global Midtrans |

### 2.2 Backend Go (`backend/go-core`)

| Variabel | Wajib? | Fungsi |
|---|---|---|
| `JWT_SECRET` | **WAJIB** | ≥ 32 karakter (config fails fast). Generate: `openssl rand -base64 48` |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` | **WAJIB** | Koneksi Postgres. `DB_SSLMODE=require` di produksi |
| `KAFKA_BROKERS` | **WAJIB** | Contoh: `kafka:29092` (in-cluster) atau `localhost:9092` |
| `SERVER_PORT` | Opsional | Default 8081 |
| `APP_ENV` | Opsional | `production` / `development` (mengubah format log zerolog) |
| `AI_BASE_URL` | Opsional | Base URL AI worker, mis. `http://ai-worker:8000` (untuk inference HTTP) |
| `REDIS_URL` | Opsional | Broker Celery AI worker |
| `FRONTEND_ORIGIN` | Opsional | Whitelist CORS origin frontend |
| `SENTRY_DSN` | Opsional | Error tracking |
| `MATCH_AMOUNT_TOLERANCE_PCT` | Opsional | Toleransi 3-way match (default 2.0) |

### 2.3 AI Worker (Python)

| Variabel | Wajib? | Fungsi |
|---|---|---|
| `KAFKA_BROKERS` | **WAJIB** | Consumer `invoice.created`, producer `invoice.ocr_completed` |
| `REDIS_URL` / `CELERY_BROKER` | Opsional | Celery broker. **Tanpa ini** endpoint `/api/v1/inference/*` jalan sinkron (fallback) |

---

## 3. Provisioning Infrastruktur

### 3.1 Database (PostgreSQL 15)

```bash
# 1. Buat DB + user (ganti password!)
CREATE DATABASE tayooli;
CREATE USER tayooli WITH PASSWORD '<STRONG_PASSWORD>';

# 2. Apply semua migrasi secara berurutan (001 → 013)
for f in backend/go-core/migrations/*.sql; do
  psql -h <host> -U tayooli -d tayooli -f "$f"
done
```

> Migrasi idempotent (RLS policy `DROP IF EXISTS` + `CREATE`). Ada pasangan duplikat nomor (`011_security_definer_auth_functions.sql`, `012_e2e_test_users.sql`, `013_seed_approval_users.sql`) — jalankan **semua** file `.sql` sesuai urutan nama. Migrasi seed E2E (`012_e2e_test_users.sql`) opsional di produksi; jika dipakai, **wajib ganti password seed** (lihat §8).

### 3.2 Kafka (KRaft)

Topics yang dipakai (lihat `docs/kafka-topics.md`): `invoice.created`, `invoice.ocr_completed`, `invoice.approved`, `payment_order.*`, `vendor.*`, plus topic accounting. Pastikan ada sebelum backend start, atau aktifkan auto-create (default broker).

### 3.3 Redis (opsional)

Hanya dibutuhkan untuk Celery async di AI worker. Tanpa Redis, AI worker tetap berfungsi (fallback sinkron) — cocok untuk MVP.

---

## 4. Deploy Options

### Opsi A — Self-host via docker-compose

```bash
POSTGRES_PASSWORD=<STRONG> docker compose up -d          # postgres + redis + kafka + kafka-ui
cd backend/go-core && go run ./cmd/api                   # atau build image
# ai-worker: cd ai-worker && uvicorn app:app --host 0.0.0.0 --port 8000
# frontend: NEXT_PUBLIC_API_URL=http://localhost:8081 npm run build && npm start
```

### Opsi B — Kubernetes (manifests sudah ada di `k8s/`)

1. Provision secret **out-of-band** (CI sengaja mengeksklusi `secret.yaml`):
   ```bash
   kubectl create secret generic tayooli-api-secrets -n tayooli-core \
     --from-literal=DB_USER=tayooli \
     --from-literal=DB_PASSWORD='<STRONG>' \
     --from-literal=JWT_SECRET="$(openssl rand -base64 48)"
   ```
2. `kubectl apply -f k8s/00-namespace.yaml -f k8s/10-configmap.yaml -f k8s/30-finance-service.yaml -f k8s/31-ai-worker.yaml -f k8s/32-tayooli-api.yaml -f k8s/40-ingress.yaml`
3. Tunggu rollout: `kubectl rollout status deployment/tayooli-api -n tayooli-core`

### Opsi C — Freebuff-managed hosting (frontend only) ⚠️

Freebuff hosting membangun **hanya frontend Next.js** (builder Node-only; Go/Python tidak bisa jalan di sana). Command terkonfigurasi: install `npm install`, build `npm run build`, preview `npm run dev` :3000 (verified `freebuff-deploy check`: deployable, 0 problems).

**Peringatan penting:** deploy frontend **tanpa env var** akan menampilkan `503 auth backend not configured` di halaman login, karena demo auth dimatikan di produksi (`lib/auth/demo-auth.ts`). Dua pilihan:

- **Demo publik:** set `AUTH_DEMO=true` (+ opsional `GEMINI_API_KEY`) → app bisa dicoba dengan akun seed.
- **Produksi asli:** set `NEXT_PUBLIC_API_URL` / `BACKEND_URL` ke URL backend Go yang di-host terpisah (Opsi A/B) dan expose via HTTPS. Go API + AI worker harus di-host di tempat lain (Docker/K8s/VM) karena tidak bisa di-build oleh Freebuff hosting.

Env var produksi di Freebuff diatur via `freebuff-deploy env set '{"KEY":"value"}'` (terpisah dari `.env` sandbox).

---

## 5. CI/CD (`.github/workflows/ci.yml`)

Pipeline pada push ke `main`: `test-go` (Postgres+Kafka service, migrasi, RLS) → `test-frontend` (lint, jest, `next build`) → `test-ai-worker` (compileall + import check) → `e2e` (Playwright) → `build` (image backend + ai-worker → GCR, tag `sha`+`latest`) → `deploy` (pin image tag, dry-run validate, apply, rollout).

Catatan produksi:
- `kubectl apply` hanya berjalan di `main`; **secrets tidak pernah di-apply oleh CI** (by design).
- `secrets.GCR_SA_KEY`, `secrets.KUBECONFIG`, `secrets.CODECOV_TOKEN` harus diisi di GitHub (environment `production`).

---

## 6. Health, Monitoring, Observability

| Endpoint | Komponen | Fungsi |
|---|---|---|
| `GET /health` | Go API | Liveness (HTTP 200) |
| `GET /health/ready` | Go API | Readiness (DB, Kafka) |
| `GET /metrics` | Go API (:9090) | Prometheus metrics |
| `GET /health` | AI worker | Liveness FastAPI |
| Sentry (`SENTRY_DSN`) | Go API + frontend (opsional) | Error tracking |

Yang belum ada dan direkomendasikan sebelum go-live: backup otomatis Postgres (pg_dump/WAL), alerting Prometheus, log aggregation (Loki/Cloud Logging), uptime check.

---

## 7. Smoke Test Checklist (pasca-deploy)

```bash
# 1. Backend hidup
curl -sf http://<api>/health && curl -sf http://<api>/health/ready

# 2. Auth + sesi
curl -sf -c /tmp/cj -X POST http://<api>/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@<domain>","password":"<password>"}'
curl -sf -b /tmp/cj http://<api>/api/v1/auth/me

# 3. Data tenant-isolated
curl -sf -b /tmp/cj http://<api>/api/v1/invoices   # hanya data tenant sendiri

# 4. Frontend
#   - Login → dashboard → buat invoice → approval flow
#   - AI analysis (anomaly badge) → butuh AI worker + GEMINI_API_KEY
#   - Payment gateway: config per-tenant di /dashboard/payment-gateways
#   - Webhook Pakasir/Midtrans: /api/webhooks/pakasir, /api/webhooks/midtrans

# 5. Kafka: buat invoice → cek consumer OCR memproses → status berubah
```

---

## 8. Checklist Go-Live

### Secrets (WAJIB diisi operator, jangan di-commit)
- [ ] `JWT_SECRET` ≥ 32 char acak (backend) — `openssl rand -base64 48`
- [ ] `DB_PASSWORD` kuat (backend + compose) dan `DB_SSLMODE=require`
- [ ] `GEMINI_API_KEY` (frontend, untuk Nara)
- [ ] `PAKASIR_API_KEY` / `MIDTRANS_SERVER_KEY` (+ per-tenant config via UI)
- [ ] `SENTRY_DSN` (opsional)

### Konfigurasi
- [ ] `NEXT_PUBLIC_API_URL`/`BACKEND_URL` menunjuk backend produksi (atau `AUTH_DEMO=true` untuk demo publik)
- [ ] `FRONTEND_ORIGIN` diset (CORS)
- [ ] Migrasi 001–013 applied
- [ ] Topics Kafka ada
- [ ] Password seed users diganti jika memakai `012_e2e_test_users.sql` (admin@test.com / password123 — **jangan dibawa ke produksi**)

### Operasional
- [ ] HTTPS (ingress/load balancer) + redirect HTTP→HTTPS
- [ ] Backup Postgres terjadwal (WAL/pg_dump) + restore pernah diuji
- [ ] `/metrics` di-scrape; alerting dasar
- [ ] Smoke test §7 lulus end-to-end di environment produksi
- [ ] Prosedur rollback teruji (rollback image tag + restore DB)

---

## 9. Rollback

1. **K8s:** `kubectl rollout undo deployment/tayooli-api -n tayooli-core` (atau apply image tag versi sebelumnya dari CI).
2. **Compose:** `docker compose pull && docker compose up -d` dengan tag lama.
3. **DB:** restore dari backup terakhir; migrasi ke depan bersifat idempotent, migrasi ke belakang TIDAK didukung (blok DOWN) — jangan drop kolom/table secara manual.
