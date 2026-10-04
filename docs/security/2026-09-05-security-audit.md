# Tayooli ERP — Security Audit Report (Pra-Produksi)

- **Tanggal audit:** 2026-09-05
- **Scope:** Backend Go (`backend/go-core`), Frontend Next.js (`app/`, `components/`), konfigurasi GCP/nginx
- **Metode:** Audit kode berbasis bukti (membaca file langsung + `go build`/`go test`/`tsc --noEmit`) + cek konfigurasi server
- **Catatan keterbatasan:** SSH ke `tayooli-server` tidak dapat dijangkau saat audit (timeout ×2). Item bertanda ⏳ = status server perlu verifikasi ulang — tidak diklaim tanpa bukti.

---

## Ringkasan Eksekutif

| Kontrol | Status | Catatan |
|---|---|---|
| Autentikasi & Otorisasi | ✅ **Kuat** | JWT HS256 ketat + cookie aman + RLS per-tenant + RBAC per-endpoint |
| Rate limiting | 🟡 **Parsial → membaik** | Token bucket per-IP; XFF-spoofing sudah ditutup; cakupan belum penuh (daftar read/write CRUD) |
| Validasi & sanitasi input | 🟡 **Membaik** | Email normalize + trim baru; body cap di semua write; Zod merata di form utama; masih ada form kecil tanpa Zod |
| CSRF protection | ✅ **Baik** | SameSite=Strict + CORS whitelist (tanpa token eksplisit — memadai untuk arsitektur ini) |
| CORS | ✅ **Diperketat** | Single-origin via `FRONTEND_ORIGIN`; tidak pernah wildcard |
| Transport & headers | 🟡 **Parsial** | HTTP di port 8081 terekspos langsung; belum ada security headers di nginx; TLS self-signed ⏳ |

**Kesimpulan:** Fondasi keamanan solid untuk rilis awal (auth, tenancy, CSRF, CORS, SQL). Yang **harus** dibereskan sebelum produksi publik: P0 di bawah (pengetatan paparan jaringan + verifikasi secret). Tidak ditemukan kerentanan kritis kelas SQLi / RLS-bypass / auth-bypass di kode yang diaudit.

---

## 1. Autentikasi & Otorisasi — ✅

**Bukti (file):**
- JWT: `internal/middleware/tenant.go` — `jwt.WithValidMethods(["HS256"])`, `WithExpirationRequired()`, verifikasi `tenant_id` UUID.
- Secret gate: `internal/config/config.go:11,93` — `MinJWTSecretLength = 32`, `config.Load` gagal cepat bila pendek; `tenant.go` mengulang gate sebagai pertahanan kedua.
- Cookie: `internal/handler/auth_handler.go` — `tayooli_auth` **HttpOnly + Secure (APP_ENV=production) + SameSite=Strict**; fallback Bearer header (backward compat).
- Password: bcrypt cost 12 (`usecase/auth/auth_usecase.go:119,227`); tidak ada log password (grep bersih).
- Tenancy: `TenantMiddleware` di seluruh grup `/api/v1` (tenant_id dari JWT, bukan klien) + `SET LOCAL app.current_tenant_id` per transaksi (`infra/postgres/shared.go`) → RLS aktif.
- RBAC: `RequireRole` per endpoint di `cmd/api/main.go` — create invoice (admin/accountant), approve (approver), payment (treasury/cfo), team admin-only, dst.
- PlanGuard: membatasi aksi tulis sesuai plan subscription.

**Risiko sisa:** a) Bearer header sebagai fallback memungkinkan token di-copy ke luar cookie (minor; header tetap divalidasi sama); b) ⏳ verifikasi panjang `JWT_SECRET` di server (>32 char) belum diulang saat audit ini.

## 2. Rate Limiting — 🟡 (Parsial → membaik)

**Sudah (bukti: `internal/middleware/ratelimit.go`, `cmd/api/main.go`):**
- Token bucket per-IP (100 req/min, burst 100) → `auth/login`, `register`, `resend-verification`, `forgot-password`, `reset-password`, `/inference/*`, dan (perbaikan sesi ini) **`/subscription/pay` + `/chat/message`**.
- **XFF-spoofing ditutup** (perbaikan sesi ini): IP klien di-resolve via `TRUSTED_PROXY_IPS` — header `X-Forwarded-For` hanya dipercaya dari proxy terdaftar; default pakai `RemoteAddr` (6 unit test baru hijau).

**Gap:**
- Endpoint daftar & CRUD lain (`GET/POST /invoices`, `/purchase-orders`, `/vendors`, `/settings/team`, `/accounts`, dsb.) **belum** dilimit — DoS terbatas masih mungkin via satu IP (dampak diredam pool DB cap 25).
- Limiter in-memory → tidak terdistribusi (OK untuk single-instance saat ini; wajib Redis bila multi-instance).
- ⏳ `TRUSTED_PROXY_IPS` belum ter-set di server; Next proxy **tidak meneruskan XFF** (lihat rekomendasi P1) → per-IP efektif hanya untuk akses langsung, semua user via proxy tampak `127.0.0.1`.

## 3. Validasi & Sanitasi Input — 🟡 (Membaik)

**Sudah (bukti):**
- Body cap `MaxBytesReader`: 64 KiB–1 MB di semua handler write (invoice, PO, GR, vendor, payment, approval, inference, auth — auth baru ditambah sesi ini).
- Email: `normalizeEmail()` baru (`auth_handler.go`) — trim + lowercase + validasi `net/mail` + domain ber-titik; dipakai login/register/resend/forgot; fallback case-legacy di login.
- Register: full name di-trim; duplikat dicek pada email ternormalisasi.
- SQL 100% parameterized; satu-satunya `fmt.Sprintf` = `SET LOCAL` dengan UUID tervalidasi (aman).
- UUID param selalu di-`uuid.Parse` sebelum dipakai.
- Frontend: Zod merata — `lib/schemas/*` (vendor/invoice/PO/GR/payment-order/dashboard), RHF+zod (auth 4 + accounting 2), dan **baru** customers, sales-orders, sales-invoices, products (safeParse + per-field).

**Gap / catatan:**
- Rate & jumlah: `/chat/message` tanpa batas **per-user** (limiter per-IP 100/min masih longgar utk LLM yang berbiaya) — rekomendasi P1.
- Form kecil lain belum Zod (billing amount input, help/ReportForm, onboarding workspace sudah manual min-2) — P2.
- Belum ada trim/lowercase normalisasi email untuk data lama (migrasi) — P3.

## 4. CSRF Protection — ✅

- Cookie `SameSite=Strict` memblokir pengiriman cookie lintas-situs; CORS whitelist mencegah pembacaan respons dari origin lain.
- Tidak ada token CSRF eksplisit — **keputusan desain yang memadai** untuk API cookie-based modern. Catatan: tetap JANGAN pernah set `SameSite=None` tanpa token CSRF.

## 5. CORS & Transport — 🟡

**Sudah:** `internal/middleware/cors.go` — `Access-Control-Allow-Origin` hanya di-set jika Origin **persis** `FRONTEND_ORIGIN`; methods/headers terbatas; tidak ada wildcard; dipasang global di `main.go:359`.

**Gap:**
- Firewall `allow-tayooli-api` membuka **port 8081 langsung ke internet** → bypass nginx (rate limit XFF, logging, header). P0: batasi 8081 ke `127.0.0.1` (nginx→Next→backend) atau IP internal; akses publik cukup lewat 443.
- ⏳ TLS: sertifikat self-signed (per catatan infra); bila domain `tayooli.com` dipasang → wajib Let's Encrypt + redirect HSTS. P0/P1.
- ⏳ `APP_ENV=production` di server harus aktif agar cookie `Secure` (belum diverifikasi ulang).

## 6. Secrets & Konfigurasi — 🟡

- ⏳ `JWT_SECRET` (wajib ≥32 char via gate runtime) — panjang aktual di server belum diverifikasi ulang (SSH down).
- ⏳ `FRONTEND_ORIGIN` (harus domain produksi, bukan localhost) — belum diverifikasi ulang.
- `GROQ_API_KEY`, `PAKASIR_API_KEY`, `TELEGRAM_BOT_TOKEN` ter-set (terverifikasi sesi sebelumnya); `PAKASIR_WEBHOOK_KEY` & `TELEGRAM_WEBHOOK_SECRET` = **opsional** di kode — wajib di-set di produksi.
- `.env` tidak pernah di-commit (`.gitignore` lokal; komit `.freebuff/` dsb sudah beres).
- Saran P2: rotasi JWT secret berkala + simpan secret via Secret Manager GCP bila tim bertambah.

## 7. Webhook & Integrasi

- Pakasir: `subscription_handler.go:347-361` — verifikasi `X-Pakasir-Key` **hanya bila** `PAKASIR_WEBHOOK_KEY` di-set → P0: set key-nya.
- Telegram: `telegram_bot_handler.go` — verifikasi `X-Telegram-Bot-Api-Secret-Token` bila `TELEGRAM_WEBHOOK_SECRET` di-set → P1: set secret-nya.
- Endpoint `/metrics` publik (tanpa auth) — bocor metadata request. P2: proteksi (allowlist IP atau pindah port internal).

## 8. Lain-lain

- Security headers (nginx): belum ada `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Content-Security-Policy`. P1.
- Tidak ada logging password/rahasia (grep bersih). ✅
- Timeout global 15 s (`chiMiddleware.Timeout`) + pool DB cap 25 + timeouts server — ✅ (dari hardening sesi sebelumnya).

---

## Rekomendasi Berprioritas

### 🔴 P0 — Wajib sebelum produksi publik
| # | Aksi | File/Konfigurasi |
|---|---|---|
| P0-1 | Batasi port 8081 ke loopback/internal — publik hanya via 443/nginx | Firewall GCP (`allow-tayooli-api`) |
| P0-2 | Set `PAKASIR_WEBHOOK_KEY` + pastikan webhook Pakasir memakai header-nya | `/opt/tayooli/.env` |
| P0-3 | Verifikasi `JWT_SECRET` ≥32 char acak, `APP_ENV=production`, `FRONTEND_ORIGIN` = domain produksi | `/opt/tayooli/.env` |

### 🟠 P1 — Sebelum/awal produksi
| # | Aksi | File/Konfigurasi |
|---|---|---|
| P1-1 | Nginx: tambah security headers (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `CSP` dasar) | `/etc/nginx/sites-enabled/default` |
| P1-2 | Set `TELEGRAM_WEBHOOK_SECRET` | `/opt/tayooli/.env` |
| P1-3 | Perluas rate limit ke daftar CRUD utama (atau terapkan limiter global ringan) | `cmd/api/main.go` |
| P1-4 | Buat limiter per-user lebih ketat untuk `/chat/message` (biaya LLM) | `internal/middleware/ratelimit.go` |
| P1-5 | Next proxy meneruskan `X-Forwarded-For` asli + set `TRUSTED_PROXY_IPS=127.0.0.1` | `app/api/**/route.ts`, `/opt/tayooli/.env` |
| P1-6 | Pasang domain + Let's Encrypt (bila `www.tayooli.com`) → hapus cert self-signed | nginx/certbot |

### 🟡 P2 — Secepatnya setelah stabil
| # | Aksi |
|---|---|
| P2-1 | Proteksi `/metrics` (IP allowlist / internal) |
| P2-2 | Zod untuk form kecil tersisa (billing, ReportForm/help) + hapus `.orig`/`.rej` di `components/accounting/` |
| P2-3 | CSRF token eksplisit opsional bila kelak cookie memakai `SameSite=None`/cross-site |
| P2-4 | Secret Manager GCP + rotasi JWT berkala |

### 🟢 P3 — Backlog
| # | Aksi |
|---|---|
| P3-1 | Migrasi normalisasi email data lama (lowercase/trim) |
| P3-2 | Limiter Redis untuk multi-instance |
| P3-3 | Penetration test terjadwal + dependency scan (govulncheck, npm audit) di CI |

---

## Lampiran — Cara verifikasi ulang (SSH saat tersedia)

```bash
# 1) Secret & env kunci
gcloud compute ssh tayooli-server --zone=us-central1-c --command="\
  JWT=\$(grep JWT_SECRET /opt/tayooli/.env | head -1 | sed 's/.*=//' | tr -d '\"'); \
  echo jwt_len=\${#JWT}; grep -E '^(APP_ENV|FRONTEND_ORIGIN|PAKASIR_WEBHOOK_KEY|TELEGRAM_WEBHOOK_SECRET|TRUSTED_PROXY_IPS)=' /opt/tayooli/.env"
# 2) Metrics exposure
gcloud compute ssh ... --command="curl -s -o /dev/null -w '%{http_code}' http://localhost:8081/metrics"
# 3) Security headers
gcloud compute ssh ... --command="grep -c add_header /etc/nginx/sites-enabled/default"
# 4) Uji rate limit (dari luar, setelah deploy fix): >100 request cepat ke /api/v1/chat/message → harap 429
```
