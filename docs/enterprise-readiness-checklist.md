# Daftar Periksa Kesiapan Jual ke Perusahaan Besar — Tayooli

Dibuat: sesi audit setelah komit `a616177`. Status berdasarkan bukti di repo dan server live, bukan klaim.

**Legenda status**
- ✅ SUDAH — ada bukti (file / tes / uji live)
- 🟡 SEBAGIAN — ada, tapi belum lengkap atau belum terbukti
- ❌ BELUM — tidak ditemukan
- ❓ TIDAK DIKETAHUI — tidak bisa dicek dari repo (perlu dicek manual)

**Prioritas:** P0 = wajib sebelum menawarkan ke perusahaan besar · P1 = wajib sebelum kontrak ditandatangani · P2 = nilai tambah

---

## Ringkasan

| Area | ✅ | 🟡 | ❌ | ❓ |
|---|---|---|---|---|
| A. Keamanan | 5 | 3 | 2 | 0 |
| B. Keandalan data | 4 | 1 | 1 | 0 |
| C. Skala & performa | 3 | 1 | 1 | 0 |
| D. Operasional | 1 | 2 | 3 | 0 |
| E. Mutu kode & tes | 3 | 2 | 1 | 0 |
| F. Legal & bisnis | 1 | 0 | 4 | 0 |

**Kesimpulan:** fondasi teknis cukup kuat, tapi area **Operasional** dan **Legal & bisnis** hampir kosong — dua area itu yang paling sering membuat perusahaan besar menolak.

---

## A. Keamanan

| # | Item | Status | Bukti / catatan | Prio |
|---|---|---|---|---|
| A1 | Pemisahan data antar perusahaan (tenant) di database | ✅ | PostgreSQL RLS + `FORCE`, `setTenantLocally` per transaksi | — |
| A2 | Password disimpan aman | ✅ | bcrypt cost 12 (`auth` usecase) | — |
| A3 | Sesi login aman | ✅ | Cookie HttpOnly + Secure + SameSite=Strict; JWT HS256, secret min. 32 karakter (`docs/security/2026-09-05-security-audit.md`) | — |
| A4 | Batas jumlah permintaan | ✅ | Per-IP + per-tenant (komit `f3d0436`), diuji live: 429 + `Retry-After` | — |
| A5 | Error internal tidak bocor ke pengguna | ✅ | Komit `a616177`; tes `internal_error_test.go` | — |
| A6 | Format error seragam | 🟡 | Masih 81 `http.Error` teks polos | P2 |
| A7 | Header keamanan web | 🟡 | HSTS + X-Frame-Options ada di `next.config.ts`; **Content-Security-Policy belum ada** | P1 |
| A8 | Audit keamanan | 🟡 | Ada audit internal (2026-09-05), tapi ditulis untuk server GCP lama, bukan Zeabur | P0 |
| A9 | Pentest oleh pihak luar independen | ❌ | Tidak ditemukan | P0 |
| A10 | Login dua langkah (2FA) / SSO perusahaan (SAML/OIDC) | ❌ | Tidak ditemukan di kode | P1 |

## B. Keandalan data

| # | Item | Status | Bukti / catatan | Prio |
|---|---|---|---|---|
| B1 | Setiap mutasi stok terhubung ke batch | ✅ | Aturan ADR-014; `batch_id` NOT NULL di `stock_movements` | — |
| B2 | Kasir aman dari klik ganda | ✅ | Idempotency key di `pos_handler.go` + tes `pos_idempotency_test.go` | — |
| B3 | Jejak audit perubahan data | ✅ | Tabel `audit_logs` + `copilot_audit` (cakupan per-modul belum diperiksa) | — |
| B4 | Transfer / Opname / Scrap / Surat Jalan aman dari klik ganda | ✅ | Idempotent middleware di Chi router WMS (`wms_handler.go`, `cmd/api/main.go`) + forward header di `lib/api.ts` dan auto-generate UUID di `hooks/useWMS.ts`. Diuji di `idempotency_test.go` dan `wms-idempotency-client.test.ts` | — |
| B5 | Aman dari saling-kunci (deadlock) saat ramai | 🟡 | `ORDER BY id` sebelum `FOR UPDATE` di `wms_manifest_repo.go:53`; retry error 40P01 di usecase belum ada | P1 |
| B6 | Daftar produk tanpa batas 1.000 di semua modul | 🟡 | Modul WMS (Opname, Scrap, Marketplace, Transfer, Barang Masuk, Arus Barang, Surat Jalan, Pengaturan WMS, LPN) memakai `ProductCombobox` / `useProductSearch` (pencarian server); scan barcode lewat `barcodes/resolve`; nama produk & pelapor di daftar Scrap/Opname di-join di backend (`wms_repo.go`). Sisa `useProducts()`: Kasir POS (`pos/page.tsx`), Produk (`products/page.tsx`), `app/dashboard/products` | P1 |

## C. Skala & performa

| # | Item | Status | Bukti / catatan | Prio |
|---|---|---|---|---|
| C1 | Dasbor cepat | ✅ | 59–147 ms live (komit `491ee82`) | — |
| C2 | Pencarian produk berindeks | ✅ | Migrasi 052 (trigram + indeks komposit) | — |
| C3 | Timeout berantai ditangani | ✅ | Komit `4ee05d5`; proxy membalas 504 jujur | — |
| C4 | Query stok lain masih pola OR-join yang lambat | 🟡 | `product_repo.go` ListInventoryFromWMS, `wms_batch_repo.go`, `wms_repo.go` | P1 |
| C5 | Uji beban (berapa pengguna serentak sanggup) | ❌ | Tidak ada skrip k6/load test | P0 |

## D. Operasional

| # | Item | Status | Bukti / catatan | Prio |
|---|---|---|---|---|
| D1 | Deploy otomatis hanya jika tes lulus | ✅ | `.github/workflows/ci.yml`: test-go + test-frontend → deploy | — |
| D2 | Pelacakan error (Sentry) | 🟡 | Kode ada di `main.go`, aktif hanya jika `SENTRY_DSN` diisi — tidak terlihat aktif di log produksi | P0 |
| D3 | Runbook produksi | 🟡 | `docs/production-runbook.md` ada, tapi masih menulis GCP/k8s, bukan Zeabur | P1 |
| D4 | Backup otomatis + **restore pernah diuji** | ❌ | Runbook sendiri menulis "belum ada" (baris 150, 198) | P0 |
| D5 | Pemantauan uptime + peringatan otomatis | ❌ | Runbook baris 150: belum ada | P0 |
| D6 | Backup Zeabur bawaan aktif? | ❌ | Dicek langsung di dashboard (tab *Backup & Restore* service `postgresql`): fitur Auto Backup terkunci di paket **Dev+** ("Upgrade plan"); *Backup List* kosong. Database = `postgres:18` dengan volume tunggal di 1 server — jika server/volume rusak, data hilang | P0 |

## E. Mutu kode & tes

| # | Item | Status | Bukti / catatan | Prio |
|---|---|---|---|---|
| E1 | Tes frontend otomatis | ✅ | 51 suite, 638 tes, jalan di CI | — |
| E2 | Tes backend otomatis | ✅ | 85 file tes Go, jalan di CI | — |
| E3 | Cek tipe (TypeScript strict) | ✅ | `tsc --noEmit` di CI | — |
| E4 | Tes end-to-end (Playwright) | 🟡 | 8 file di `frontend/e2e*`, tapi **tidak dijalankan di CI** | P1 |
| E5 | Tes integrasi database benar-benar jalan di CI | 🟡 | CI menjalankan migrasi + `go test`, tapi log job tidak bisa dibuka untuk memastikan tes integrasi tidak di-skip | P1 |
| E6 | Pemindaian celah library (govulncheck / npm audit / Dependabot) | ❌ | Tidak ada di `.github/` | P1 |

## F. Legal & bisnis

| # | Item | Status | Bukti / catatan | Prio |
|---|---|---|---|---|
| F1 | Kepatuhan UU PDP (No. 27/2022): kebijakan privasi, lokasi data, DPA | ❌ | Tidak ditemukan dokumen | P0 |
| F2 | SLA tertulis (mis. uptime 99,5%, waktu respons dukungan) | ❌ | Tidak ditemukan | P0 |
| F3 | Syarat & Ketentuan / kontrak (kepemilikan data, keluar layanan, batas tanggung jawab) | ❌ | Tidak ditemukan | P0 |
| F4 | Tim dukungan / on-call di luar satu orang | ❌ | Tidak ditemukan | P0 |
| F5 | Lokasi server data (region Zeabur) memenuhi syarat pembeli | ✅ | Dashboard Zeabur: server **Tencent Jakarta 2C 2GB** — data berada di Indonesia (cocok untuk UU PDP). Catatan: hanya 2 vCPU / 2 GB RAM untuk DB + backend + frontend sekaligus | P1 |

---

## Urutan kerja yang disarankan

**Tahap 1 — teknis yang bisa dikerjakan sekarang (P0 teknis)**
1. B6 — modul WMS selesai; sisa Kasir POS & halaman Produk.
2. B4 — idempotency key untuk Transfer, Opname, Scrap, Surat Jalan.
3. B5 — urutkan kunci `FOR UPDATE` + retry deadlock.
4. D2 — aktifkan Sentry (isi `SENTRY_DSN`).
5. D4 / D6 — aktifkan backup harian + **uji restore** ke database terpisah.
6. D5 — pasang uptime monitor (mis. UptimeRobot) ke `/health`.
7. C5 — uji beban k6 (lokal / staging, bukan produksi).

**Tahap 2 — sebelum kontrak (P1)**
CSP (A7), 2FA (A10), E2E di CI (E4), govulncheck + Dependabot (E6), runbook Zeabur (D3), query OR-join (C4).

**Tahap 3 — dokumen & organisasi (P0 non-teknis, butuh pemilik bisnis)**
Pentest pihak luar (A9), dokumen UU PDP (F1), SLA (F2), kontrak (F3), tim dukungan (F4).
