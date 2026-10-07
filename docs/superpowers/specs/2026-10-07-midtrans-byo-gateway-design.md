# Design Spec: Midtrans BYO Payment Gateway untuk Enterprise (POS + Invoice) — Step-by-Step

**Tanggal:** 2026-10-07
**Status:** Draft — menunggu review tertulis pemilik produk
**Prinsip kunci:** Tayooli tidak mengelola uang. Tenant input kredensial gateway miliknya sendiri, dana settlement langsung ke rekening bank tenant (Model B / BYO). Platform hanya bikin tagihan/kode bayar dan mencatat status lunas via webhook.
**Scope V1:** POS kasir + Payment Link Invoice, tenant = perusahaan besar (jawaban A di 3 pertanyaan brainstorming), settlement = BYO akun tenant (jawaban A), provider V1 = Midtrans saja. Pakasir tetap ada sebagai fallback UMKM. Xendit/DOKU menyusul sebagai provider tambahan — tidak di V1.

## 1. Ringkasan — kita mau apa

- QRIS di POS sekarang palsu: `app/(app)/pos/page.tsx:362` generate string hardcode `ID.LINKAJA.WWW…E85C` + render QR lokal. CRC mismatch (E85C vs C27F), TLV broken. `backend/usecase/pos/pos.go:Checkout` tidak pernah call gateway. Tombol konfirmasi manual bisa salah pakai. Uang tidak bergerak.
- V1: ganti QRIS palsu jadi QRIS dinamis asli dari Midtrans Snap/Core API, status lunas diverifikasi server-to-server sebelum cetak struk & potong stok. Invoice penjualan juga dapat payment link Midtrans. Semua dana masuk ke akun Midtrans milik tenant, bukan platform.
- Rollout step-by-step: Midtrans dulu, stabil, baru tambah Xendit/DOKU sebagai provider baru tanpa ubah handler/UI.

## 2. Bukan tujuan V1 (Non-goals)

- Tidak bikin gateway sendiri, tidak direct ke API bank.
- Tidak ubah model settlement: tetap BYO, tidak bikin split-settlement/platform-aggregator.
- Tidak sentuh modul enterprise lama (P2P/O2C/CoA/Jurnal) — tetap 13 modul inti.
- Tidak migrasi paksa tenant Pakasir; mereka tetap LIVE sampai migrasi mandiri.

## 3. Konteks yang dibaca (biar tidak ngarang)

- Abstraksi provider sudah ada: `lib/payments/types.ts` (interface `PaymentProvider`), `lib/payments/registry.ts` (pakasir + midtrans), `lib/payments/providers/midtrans.ts` (Snap, SHA512 webhook, verifyTransactionStatus, sudah pakai `serverKey/clientKey/isProduction`), `lib/payments/providers/pakasir.ts`, `lib/payments/webhook-handler.ts` (gateway-agnostic, cek signature + re-verify server-to-server).
- FE: `app/dashboard/payment-gateways/page.tsx` (form mentah per-tenant + settlement bank fields), `components/billing/QRISPaymentModal.tsx` (QRIS langganan via `/api/v1/subscription/pay`), `app/(app)/pos/page.tsx` (dummy QR), `app/payments/demo/[orderId]/page.tsx` (demo checkout).
- BE: `backend/go-core/internal/domain/payment_gateway.go` (TenantPaymentConfig), `backend/go-core/migrations/014_payment_gateways.sql` (tenant_payment_configs + payment_transactions + RLS), `backend/go-core/internal/handler/payment_gateway_handler.go` (Get/Upsert, pakai `middleware.GetTenantID`), `backend/go-core/cmd/api/main.go:634-638` (route `/payments/configs`), `backend/go-core/internal/handler/subscription_handler.go:361` (webhook Pakasir lama — belum ada handler Midtrans terpisah).
- ADR: `docs/adr/008-pakasir-payment-gateway.md` (Model B BYO, settlement H+1, KYC), `docs/api/pakasir.md` (webhook & config API).
- Design system (ui-ux-pro-max, query `enterprise ERP payment gateway checkout`): Minimalism/Swiss, Light+Dark, primary #2563EB / CTA #EA580C, typography Outfit+Work Sans, hover 200–250ms, anti-pattern: jangan pakai emoji sebagai ikon.

## 4. Keputusan yang sudah disepakati

- Settlement: BYO — tenant daftar akun Midtrans sendiri (KYC perusahaan), simpan `serverKey/clientKey/isProduction` di `tenant_payment_configs`. Webhook `signature_key = SHA512(order_id+status_code+gross_amount+ServerKey)` diverifikasi + re-check `GET /v2/{orderId}/status`. Uang settlement ke rekening tenant sesuai setting di dashboard Midtrans tenant — field `settlement_bank_*` di Tayooli dihapus dari sumber kebenaran (opsional display-only jika perlu).
- Scope: POS + Payment Link Invoice. POS butuh QRIS dinamis dengan masa berlaku + polling status; invoice butuh redirect_url Snap.
- Provider V1: Midtrans. Go SDK resmi `midtrans-go`, sandbox+production, biaya transparan QRIS 0.7%, VA Rp4.000, dokumentasi POS dinamis `GoPay QRIS POS Integration` + `validityPeriod`. Xendit/DOKU dicatat sebagai provider berikutnya — abstraksi tetap terbuka (`PaymentProviderId` union diperluas, tidak ubah kontrak).

## 5. Arsitektur V1

### 5.1 Kontrak tidak berubah
`PaymentProvider` tetap 7 method (`isConfigured`, `globalConfig`, `createPayment`, `parseWebhook`, `verifyWebhook`, `verifyTransactionStatus`, `isPaidStatus`, `demoPaymentLink`). Route handler, `webhook-handler.ts`, dan UI tetap provider-agnostic. Tambah provider = tambah 1 file `providers/xendit.ts` + 1 baris di `registry.ts`.

### 5.2 Yang dirapikan untuk Midtrans POS (beda dari Snap langganan)
- Snap `redirect_url` cocok untuk invoice (customer klik link). POS butuh QR string dinamis (charge QRIS MPM / Core API) supaya QR bisa dirender langsung di modal kasir. `midtrans.ts:createPayment` diperluas: bila dipanggil dari POS (flag `channel=qris`), pakai Charge QRIS, kembalikan `qrContent + expiry`; bila dari invoice, tetap Snap. Alternatif yang tetap valid: pakai Snap juga untuk POS lalu embed `redirect_url` — keputusan final saat implementasi, kontrak luar sama (return `paymentLink|qrContent`).
- Tenant tanpa kredensial → `demo:true`, arahkan ke `/payments/demo/{orderId}` (perilaku sekarang dipertahankan).

### 5.3 Backend wiring
- Tabel tidak perlu migrasi baru untuk V1 (`tenant_payment_configs.provider='midtrans'` sudah ada). Bila butuh kolom QRIS POS (expiry), tambah di `payment_transactions` saja — bukan breaking.
- Tambah endpoint Next.js tipis untuk POS: `POST /api/pos/payments` (auth tenant, panggil `registry.resolveProviderConfig` → `provider.createPayment` → simpan `payment_transactions` → return QR/expiry) dan `GET /api/pos/payments/{orderId}/status` (polling, panggil `verifyTransactionStatus`). Atau reuse `POST /api/v1/subscription/pay` — pilih yang paling sedikit ubah permukaan API (dicek saat implementasi).
- Webhook Midtrans: `POST /api/webhooks/midtrans` → `lib/payments/webhook-handler.ts:handleProviderWebhook('midtrans', request)` — sudah ada kerangka, tinggal pastikan route ada (di Go: tambah `r.Post("/webhooks/midtrans", ...)` mirroring `HandlePakasirWebhook`; atau route Next.js bila tetap di FE). Verifikasi signature wajib sebelum `markInvoicePaid` / sebelum `Checkout` potong stok.
- Frontend proxy `app/api/v1/[...path]/route.ts` tetap meneruskan ke `tayooli-backend.zeabur.app` — tidak dirubah.

## 6. Alur

### 6.1 POS kasir (pengganti dummy)
1. Kasir pilih metode: Tunai / QRIS / VA / E-wallet. Untuk non-tunai: FE `POST /api/pos/payments` → provider `createPayment(channel=qris)` → terima `qrContent` asli + `expiry ~5 menit`.
2. Render QR (`qrcode`), tampil nominal + countdown + `role="status"` ("Menunggu pembayaran…").
3. Poll `GET /api/pos/payments/{orderId}/status` tiap 3 dtk (maks 5 menit) + webhook settlement sebagai sumber kebenaran. Bila `capture|settlement` terverifikasi server-to-server → cetak struk + `posUsecase.Checkout` potong stok (`inventoryRepo.UpdateQuantity` + `wmsRepo.CreateStockMovement`) baru dijalankan. Tombol konfirmasi manual dihapus untuk QRIS; tunai tetap manual.
4. State: Lunas (hijau, auto-cetak), Kedaluwarsa (tombol "Buat kode baru" 1 klik), Gagal (tombol "Coba lagi / Ganti tunai"), Gateway timeout (tawarkan fallback tunai).

### 6.2 Payment Link Invoice
Reuse alur langganan: tenant pilih Midtrans → invoice `CreatePayment(provider=midtrans, amount)` → `redirect_url` Snap → customer bayar → webhook settlement → `payment_transactions.status=completed` → invoice PAID.

### 6.3 Settlement
Customer scan/bayar → dana ditampung gateway Midtrans tenant → settlement H+1 ke rekening bank tenant (diatur di dashboard Midtrans tenant). Tayooli tidak pernah memegang dana; hanya mencatat.

## 7. UX (tenant tidak kesulitan, kasir tidak salah pencet)

- **Onboarding tenant — wizard 3 langkah** ganti form mentah `payment-gateways/page.tsx`:
  1. Pilih gateway (kartu Midtrans/Pakasir berisi channel+fee+link daftar+ panduan 5 menit).
  2. Tempel kredensial (Server Key + Client Key), tombol "Test koneksi" hit sandbox — Loading → Berhasil/Gagal + langkah pemulihan (pedoman ux: Submit Feedback + Error Recovery).
  3. Tes Rp 1.000–10.000 sandbox → lunasi → webhook kembali settlement → status DEMO→LIVE. Gagal: error summary di atas form, fokus otomatis ke field bermasalah (Focusable Error Summary). Settlement bank tidak diinput di Tayooli.
- **POS modal**: nominal + timer + status live (`aria-live` contextual, bukan angka mentah), 2 tombol recovery saja. Token desain: Minimalism/Swiss, primary #2563EB, CTA #EA580C, tap target 48px, kontras 4.5:1, hormati `prefers-reduced-motion`, SVG Phosphor/Lucide (jangan emoji).
- **Error & timeout**: gateway 5xx/timeout → banner kuning "Gateway sibuk, coba lagi atau pakai tunai" + retry idempoten per `orderId`.

## 8. Keamanan & kepatuhan

- Kredensial terenkripsi at-rest (pgcrypto/KMS — ADR-008 §5, ditandai sebagai follow-up bila belum ada; minimal tidak log `[REDACTED]` dan `sanitizeURL`).
- Webhook: `parseWebhook` → cek `provider` vs transaksi (`403 provider mismatch` di `webhook-handler.ts:61`), `verifyWebhook` SHA512, lalu `verifyTransactionStatus` server-to-server sebelum state change. Demo fallback: cek amount equality.
- RLS `tenant_payment_configs` + `payment_transactions` sudah ada — handler wajib lewat `middleware.GetTenantID`.
- Pakasir vs Midtrans terisolasi: 1 transaksi tidak bisa di-settle oleh webhook provider lain (invariant `webhook-handler.ts`).

## 9. Observabilitas

- Metric: `payment_create_total{provider,status}`, `webhook_verify_failed_total`, `qris_expiry_total`, `pos_checkout_after_settlement_total`.
- Log terstruktur per `tenant_id/order_id/provider` (tanpa kunci rahasia).
- Checklist go-live: IP/webhook endpoint `/api/webhooks/midtrans` terdaftar di dashboard Midtrans tenant, sandbox → production toggle `isProduction`.

## 10. Migrasi

- Hapus/disable jalur QR dummy di `pos/page.tsx:362` untuk channel QRIS (tunai tetap). Fallback: bila `!provider.isConfigured(config)` → DEMO link (perilaku sekarang).
- Tenant Pakasir existing tetap LIVE; tidak migrasi paksa. Wizard tawarkan "Pindah ke Midtrans" bila mereka mau.
- Tidak perlu migrasi data massal; 1 tenant 1 provider (`uq_tenant_provider`).

## 11. Pengujian

- Unit: `webhook-handler.ts` (signature, amount mismatch, provider mismatch), `providers/midtrans.ts` (sandbox vs production, `isPaidStatus`), `pos/page.tsx` (3 state UI: lunas/kedaluwarsa/gagal), `payment_gateway_handler.go`.
- E2E (Playwright, port 3000 per Aturan 3080): wizard BYO → test sandbox → POS QRIS lunas → POS QRIS kedaluwarsa→buat baru → invoice payment link lunas. Verifikasi stok hanya terpotong setelah settlement (regression untuk bug QR dummy).
- Manual: bayar QRIS sandbox Midtrans, pastikan settlement masuk dashboard sandbox tenant, bukan dashboard platform.

## 12. Rollout & sukses

- Rollout: merge ke `main` → Zeabur auto-deploy (Building→Running) → verifikasi `https://tayooli.my.id` + `https://tayooli-backend.zeabur.app`.
- Kriteria sukses V1:
  1. Tenant baru LIVE <5 menit tanpa CS (wizard 3 langkah lolos).
  2. 0 struk tercetak / stok terpotong sebelum `settlement` terverifikasi.
  3. Dana QRIS tercermin di rekening/dashboard Midtrans tenant, bukan platform (BYO terbukti).
  4. Tidak ada regresi tenant Pakasir.
  5. Aksesibilitas: error summary fokus, live status terbaca screen reader, tap target ≥44pt.

## 13. Next step — Xendit/DOKU (di luar V1)

- Tambah `lib/payments/providers/xendit.ts` + `doku.ts`, daftarkan di `registry.ts`, tambah opsi di wizard. Tidak ubah handler/webhook/UI.
- Urutan setelah Midtrans stabil — diputuskan terpisah.

## 14. Persetujuan (hard gate)

Dokumen ini baru design. Implementasi (kode, dependency `midtrans-go` bila di Go, wiring POS) hanya setelah pemilik produk review tertulis dan menyetujui. Setelah disetujui, invoke `writing-plans` untuk memecah tugas implementasi.

## 15. Referensi silang (agar tidak ngarang field/endpoint)

- `lib/payments/*`, `lib/payment-store.ts`, `app/dashboard/payment-gateways/page.tsx`, `components/billing/QRISPaymentModal.tsx`, `app/(app)/pos/page.tsx:362`, `backend/go-core/internal/domain/payment_gateway.go`, `backend/go-core/migrations/014_payment_gateways.sql`, `backend/go-core/internal/handler/payment_gateway_handler.go`, `backend/go-core/cmd/api/main.go`, `docs/adr/008-pakasir-payment-gateway.md`, `docs/api/pakasir.md`.

---

*Self-review checklist penulis: placeholder (TBD/TODO) — tidak ada; konsistensi arsitektur vs alur — BYO di semua bagian; scope — V1 hanya Midtrans, Xendit/DOKU eksplisit di luar V1; ambiguitas shipping — hard gate "review tertulis sebelum coding" ditegaskan di §14.*
