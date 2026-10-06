# Tayooli ERP - Project Status & Codebase Map

> **UNTUK AI / AGEN BARU**: Baca dokumen ini pertama kali sebelum menjelajahi codebase. Dokumen ini adalah satu-satunya sumber kebenaran (Source of Truth) mengenai status proyek saat ini. Jika dokumen lain di `docs/` bertentangan dengan ini, ikuti file ini.

## 🎯 Status Saat Ini (Branch: `main`)
**Kesiapan produksi ~92%.** Semua fitur bisnis inti selesai. Hardening operasional server selesai (2026-09-05). Yang tersisa: custom domain + SSL Let's Encrypt, dan test coverage modul baru.

> 🚀 **Go-Live Checklist:** [`docs/go-live-checklist.md`](docs/go-live-checklist.md) · **Rollback Procedure:** [`docs/rollback-procedure.md`](docs/rollback-procedure.md)

## ✅ Hardening Operasional (2026-09-05) — SELESAI
- **Swap 3GB** aktif di server e2-micro (mencegah OOM crash)
- **Seed passwords dirotasi** — `admin@test.com`, `accountant@test.com`, `approver@test.com` password diganti acak (bcrypt, disimpan di `/root/.tayooli-credentials`)
- **Backup DB otomatis** — `pg_dump` harian jam 03:00 UTC, retensi 7 hari, di `/opt/tayooli/backups/`
- **Health monitor** — systemd timer tiap 2 menit, cek backend/frontend/nginx/postgres/disk/memory, auto-restart service yang down, log ke `/var/log/tayooli-alerts.log`
- **SSH hardened** — password auth disabled, root login disabled (key-only)
- **Snap dihapus** — membebaskan ~126MB RAM + disk
- **Security updates otomatis** — `unattended-upgrades` aktif
- **Merge conflict artifacts** — semua `.orig`/`.rej` dibersihkan, ditambahkan ke `.gitignore`
- **Rollback procedure** — didokumentasikan di `docs/rollback-procedure.md`

## ✅ Baru Selesai (Batch ini)

### Workspace Autonomous AI Copilot (APPROVED)
- **Database & RLS:** Migrasi `backend/go-core/migrations/015_tenant_ai_permissions.sql` (tabel `tenant_ai_permissions`, isolasi tenant RLS, audit log metadata).
- **Backend Go:** Domain `internal/domain/ai_permission.go`, repository `internal/infra/postgres/ai_permissions_repo.go`, handler `internal/handler/ai_permissions_handler.go` (`GET/PATCH /api/v1/ai/permissions`).
- **Hybrid Action Engine:** `app/api/copilot/chat/route.ts` dengan Gemini 2.5 Flash Function Calling, XML Context Isolation anti-injection, dan fallback otomatis ke Zero-AI Matcher `lib/copilot/fallback-matcher.ts`.
- **Governance & Guardrails:** 3 Autonomy Levels (Advisory, Assisted, Autopilot), Emergency Kill-Switch instan, High-Risk Type-to-Confirm ("KONFIRMASI"), dan 15-detik Undo window.
- **Frontend ZenSpace / Anti-Slop:** `components/copilot/` (`CopilotDrawer`, `CopilotTrigger` shortcut `Ctrl+K`, `ActionPreviewCard` diff tabular, `ProviderStatusBadge`, `AutonomySettingsModal`, `UndoToast`).
- **Verifikasi:** `npx tsc --noEmit` pass (0 errors), test suite `__tests__/copilot-matcher.test.ts`.

### Module 2 — AI Serving (APPROVED)
- `ai-worker/app.py` — FastAPI serving: `POST /api/v1/inference/ingest` (202 + job_id), `GET /api/v1/inference/status/{id}?tenant_id=` (PENDING/STARTED/SUCCESS/FAILURE + anomaly_score)
- `ai-worker/celery_app.py` — Celery app (Redis broker dari `REDIS_URL`), task `ai_worker.process_ocr` reuse `run_ocr` dari `main.py`; fallback sinkron otomatis jika `REDIS_URL` tidak diset
- `ai-worker/requirements.txt` + `README.md` (cara menjalankan uvicorn + celery worker)
- Frontend: `frontend/lib/api.ts` — `ANOMALY_THRESHOLD=0.7`, helper `isAnomaly()` dari `ai_confidence_score`; `app/(shell)/invoices/page.tsx` menampilkan badge dari data nyata

### Wire Products/Inventory (APPROVED)
- Backend: `internal/usecase/product/product.go`, `internal/handler/product_handler.go`, `internal/infra/postgres/product_repo.go` & `inventory_repo.go`, `internal/domain/product.go` & `inventory.go`
- Migrasi `backend/go-core/migrations/012_products_inventory.sql` — tabel `products` (SKU unik per tenant) & `inventory` (FK komposit anti cross-tenant, qty ≥ 0), RLS enable+force, 8 policy idempotent, blok DOWN
- Route: `GET/POST /api/v1/products` (POST: admin,accountant), `GET /api/v1/products/{id}`, `GET/POST /api/v1/inventory` (POST: admin,warehouse), `GET /api/v1/inventory/{id}`
- Frontend: `hooks/useProducts.ts`, `components/products/ProductDrawer.tsx` & `InventoryDrawer.tsx`, halaman `app/(shell)/products/page.tsx`, link di `app/(shell)/layout.tsx`
- Verifikasi: `go build/test/vet` ✅, `npx tsc --noEmit` ✅

### Module 4 — O2C E2E Test
- `frontend/e2e/o2c-cycle.spec.ts` — login → customer → sales order → sales invoice → payment order → approve → pay (2 skenario: happy path + validation error). `tsc --noEmit` ✅

### Module 5 — Production Readiness (APPROVED setelah perbaikan)
- `k8s/`: `namespace.yaml`, `configmap.yaml`, `secret.yaml` (template placeholder), `service.yaml` (http:8081 + metrics:9090), `deployment.yaml`, `deployment-ai-worker.yaml` — probe `/health` + `/health/ready`, non-root, ROFS, image pinned `${IMAGE_TAG}`
- `backend/go-core/Dockerfile` — multi-stage, `-trimpath -ldflags="-s -w"`, non-root UID 10001, HEALTHCHECK
- `docker-compose.yml` — healthcheck `/health`, `JWT_SECRET` default dev, `DB_SSLMODE: disable` (Postgres lokal tanpa SSL)
- `.github/workflows/ci.yml` — `go vet`, job `test-ai-worker`, build image via Buildx, deploy job dengan **step `Pin image tags` (sed `${IMAGE_TAG}` → `github.sha`)** + dry-run + eksklusi `secret.yaml`
- `backend/go-core/internal/config/config.go` — enforce `len(JWT_SECRET) >= 32` (fails fast saat placeholder)

## 🏗️ Peta Arsitektur & Direktori

### 1. Backend Go (`/backend/go-core`)
Hexagonal / Clean Architecture, Chi router, `database/sql` murni (NO ORM).
- **Titik Masuk:** `cmd/api/main.go` — wiring repo → usecase → handler, 2 Kafka consumer (OCR + Accounting), graceful shutdown SIGINT/SIGTERM (30s), Sentry (opsional), rate limiter, `/metrics` Prometheus, `/health` + `/health/ready`
- **Migrasi:** `migrations/` — `001_init_schema` s/d `012_products_inventory.sql` (RLS, seed, OCR fields, PO/GR, payment orders, vendors, accounting, auth functions, products/inventory)
- **Pola Multi-Tenant:** RLS dua lapis — `WHERE tenant_id = $n` di tiap query + `SET LOCAL app.current_tenant_id` di transaksi
- **Modul (`/internal`):** auth, invoice (OCR + 3-way match), PO/GR, payment-orders, vendors, customers, sales-orders, sales-invoices, approvals, accounting (journal + CoA), dashboard, ai (inference proxy), products, inventory
- **Kafka:** producer event + consumer `invoice.ocr_completed` (allowlist status `ai_processed`/`ai_failed` anti poison-pill) + consumer accounting

### 2. AI Worker Python (`/ai-worker`)
- `main.py` — consumer Kafka `invoice.created` → OCR Tesseract → publish `invoice.ocr_completed` (pipeline utama, TIDAK disentuh)
- `app.py` (BARU) — FastAPI serving + Celery (`celery_app.py`) untuk permintaan OCR sinkron/async via HTTP
- `proto/invoice_service.proto` — kontrak gRPC

### 3. Frontend Next.js (`/frontend`)
Next.js 15 (App Router), Tailwind, Zustand/TanStack Query, komponen shadcn-style di `components/ui/` (data-table, drawer, button, anomaly-badge).
- Halaman yang ada: `app/login`, `app/(shell)/` (`invoices`, `approvals`, `accounting/*`, `products`), `app/customers`, `app/sales-orders`, `app/sales-invoices` — plus `app/api/*/route.ts` sebagai **proxy server-side ke backend Go** (`BACKEND_URL` env, default `http://localhost:8081`, forward cookie/Authorization)
- **O2C Frontend (BARU selesai):** modul Customers, Sales Orders, Sales Invoices terhubung ke API nyata — mock (`setTimeout` + data palsu) dihapus dari `app/api/{customers,sales-orders,sales-invoices}/route.ts`; hook `useCustomers`/`useSalesOrders`/`useSalesInvoices` disesuaikan kontrak Go (snake_case → display), punya `create*` + `refresh` + `error`; tombol "+ New" membuka form fungsional; kolom tabel sesuai data nyata (UUID, format IDR, status UNPAID/PARTIAL/PAID/CANCELLED). Jest `__tests__/o2c.test.tsx` diperbarui — 3/3 PASS. `tsc --noEmit` ✅, `npm run build` ✅

### 4. Infrastruktur & CI/CD
- **DB:** PostgreSQL 15 (RLS). **Broker:** Apache Kafka KRaft (`apache/kafka:3.7.0`, in-cluster `kafka:29092`)
- **Live Production Server (GCP):**
  - VM Instance: `tayooli-server` (e2-micro, Ubuntu 22.04 LTS, zone `us-central1-c`, Project `mbg-waste-tracker-503014`)
  - Public IP (Static): `104.197.178.237` (reserved, tidak berubah saat restart), Port `8081` (Firewall `allow-tayooli-api`)
  - HTTPS: `https://104.197.178.237` (Nginx reverse proxy + self-signed SSL)
  - Frontend: Next.js 15 production (port 3000 via systemd `tayooli-frontend.service`)
  - Nginx: Port `80` → `443` redirect, reverse proxy to frontend (3000) + backend (8081)
  - Kafka: KRaft mode (port 9092/9093, systemd `kafka`, storage **persisten di `/var/lib/kafka`** — sebelumnya di `/tmp` sehingga hilang tiap reboot)
  - Swap: 3GB aktif (mencegah OOM crash pada e2-micro)
  - Monitoring: systemd timer `tayooli-monitor.timer` (health check tiap 2 menit, auto-restart service down, alert log `/var/log/tayooli-alerts.log`)
  - Backup: crontab `pg_dump` harian 03:00 UTC, simpan `/opt/tayooli/backups/`, retensi 7 hari
  - SSH: key-only (password auth disabled, root login disabled)
  - Security updates: `unattended-upgrades` aktif
  - Services: `tayooli-backend`, `tayooli-frontend`, `kafka`, `nginx`
  - App Binary: `/opt/tayooli/tayooli-api` managed via systemd `tayooli-backend.service`
  - Health check endpoint: `http://104.197.178.237:8081/health` (Returns HTTP 200 OK & DB healthy)
  - HTTPS health: `https://104.197.178.237/health`
- **CI (`.github/workflows/ci.yml`):** `test-go` (Postgres+Kafka service, migrasi, RLS idempotent), `test-frontend` (jest), `test-ai-worker` (compileall), `e2e` (Playwright), `build` (Buildx → GCR, tag `sha`+`latest`), `deploy` (dry-run → pin image → apply → rollout — hanya di push ke main)
- **K8s:** namespace `tayooli-core`, backend + ai-worker (detail di atas)

## 📝 Kesepakatan Pengembangan (CTO Rules)
1. **DILARANG MENGGUNAKAN ORM** (GORM dll). Gunakan `database/sql` murni atau `sqlc`.
2. Jangan hardcode rahasia; selalu baca dari environment (`JWT_SECRET` ≥ 32 char wajib).
3. Selalu filter `tenant_id` (isolasi data antar-tenant adalah harga mati) — RLS + query-level.
4. Kode "Production Ready": handle error eksplisit, log terstruktur (zerolog), timeout context.
5. Jangan tinggalkan stub/TODO untuk endpoint yang dikerjakan — selesaikan sampai tuntas.
6. UI/UX: ZenSpace (warna tenang, progressive disclosure, empathetic errors).

---
*Dokumen ini adalah intisari status terbaru. Update file ini jika ada fitur mayor yang selesai atau arsitektur yang berubah.*
*Terakhir diperbarui: 2026-09-05 — Hardening operasional selesai.*

## 🆕 2026-10-06 — Security Fix POS + Reset Password (Zeabur prod)
- **POS tenant-leak fixed + `owner` role 403 fixed** (commit `ff1c554`):
  hardcoded `DEFAULT_PRODUCTS` dihapus; tenant kosong dapat katalog kosong;
  stok default 0; scan barcode asing ditolak; cache query dibersihkan saat
  identitas auth berubah (`app/providers.tsx`).
- **Email reset-password aktif di produksi**: env `BREVO_API_KEY` /
  `BREVO_SENDER_EMAIL` / `BREVO_SENDER_NAME` diset di service backend Zeabur
  (sender `noreply@tayooli.my.id`, verified). Diagnosis lengkap ada di
  [`docs/production-runbook.md`](docs/production-runbook.md) §10.
- **Tracer untuk "200 tapi email tidak sampai"**: endpoint forgot-password
  sengaja selalu balas 200 (anti-enumeration); periksa log backend untuk
  baris `[AUTH] forgot-password requested for non-existent email`.
- Infrastruktur untuk 5 gudang: estimasi upgrade ada di
  [`docs/zeabur-5-warehouse-upgrade-estimate.md`](docs/zeabur-5-warehouse-upgrade-estimate.md)
  (belum disetujui klien).
