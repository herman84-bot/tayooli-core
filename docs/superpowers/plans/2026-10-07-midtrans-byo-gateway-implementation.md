# Midtrans BYO Payment Gateway Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans or superpowers:subagent-driven-development. Each task uses checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Midtrans as the enterprise-grade payment gateway for POS and invoices, replacing the dummy QRIS with real dynamic QRIS and real settlement tracking.

**Architecture:** Backend Go handles tenant config CRUD + Midtrans API calls + webhook verification + settlement recording. Frontend Next.js onboarding wizard (3 steps) + POS modal with real QR + polling + refactored payment abstraction remains provider-agnostic.

**Tech Stack:** Go 1.24 (Chi, Midtrans SDK `github.com/midtrans/midtrans-go`), Next.js 15 (TanStack Query, `qrcode` lib), PostgreSQL 15 (tenant_payment_configs, payment_transactions already exist).

**Spec:** [2026-10-07-midtrans-byo-gateway-design.md](../specs/2026-10-07-midtrans-byo-gateway-design.md)

## Global Constraints

- Scope V1: Midtrans only — Xendit/DOKU provider abstraction open but not implemented.
- No settlement_bank_* fields stored in Tayooli (removed from UI form).
- Tenant credentials encrypted at-rest (defer to follow-up ADR-008 §5; this plan marks placeholder, not implements).
- RLS on `tenant_payment_configs` + `payment_transactions` — handler wajib pakai `middleware.GetTenantID`.
- POS QRIS dummy at `app/(app)/pos/page.tsx:362` disabled for QRIS channel; CASH channel retained.
- Webhook verification: SHA512(order_id+status_code+gross_amount+ServerKey) checked before state change.
- Model B BYO: settlement langsung ke rekening tenant (configured di dashboard Midtrans tenant, tidak di Tayooli).

## Review Focus

1. **Midtrans API contract correctness** — exact field names (qrContent, expiry, validityPeriod) source-verified against Midtrans docs.
2. **Webhook idempotency** — same order_id replayed never double-settles invoice or double-deducts stock.
3. **POS checkout atomicity** — struk + stock deduction only if `settlement` verified server-to-server.
4. **UX accessibility** — error summary, live status, tap targets, dark-mode, reduce-motion.
5. **No data leaks** — tenant config queries scoped to tenant_id via RLS; test with two tenants.

---

## File Structure & Changes Map

```
BACKEND (Go)
  internal/
    domain/
      payment_gateway.go         ← TenantPaymentConfig (no changes; schema exists)
    infra/
      postgres/
        payment_gateway_repo.go   ← GetConfig/UpsertConfig (no changes; repo exists)
    handler/
      payment_gateway_handler.go ← GetConfig/UpsertConfig HTTP (no changes; already wired)
      payment_webhook_handler.go ← NEW: POST /api/v1/webhooks/midtrans
      pos_handler.go             ← UpdateCheckout to call payment provider + verify webhook
    usecase/
      pos/
        pos.go                    ← Checkout() adds payment verification gate
    infra/
      midtrans/
        client.go                 ← NEW: wrapper around midtrans-go SDK
  migrations/
    015_midtrans_payment_qris.sql ← NEW: add qris_expiry, qris_string to payment_transactions
  cmd/api/
    main.go                        ← register /webhooks/midtrans route + Midtrans client init

FRONTEND (Next.js)
  lib/payments/
    providers/
      midtrans.ts                ← MODIFY: split Snap (invoice) vs Core API (POS QRIS)
  components/
    payment-gateway-wizard/
      SettingsPage.tsx           ← NEW: wizard 3 steps (pilih → tempel → tes)
  app/dashboard/
    payment-gateways/
      page.tsx                   ← REPLACE form with wizard
      loading.tsx                ← (loading skeleton)
  app/(app)/pos/
    page.tsx                     ← MODIFY: disable dummy QRIS, add real charge + polling
    hooks/
      use-pos-payment.ts         ← NEW: hook for charge + polling + status
  app/api/pos/
    payments/
      route.ts                   ← NEW: POST /api/pos/payments (charge), GET /api/pos/payments/{orderId}/status
```

---

## Task Breakdown (Bite-Sized Steps)

### Phase 1: Backend Infrastructure (Midtrans SDK + Migration)

- [ ] **1.1** Install `github.com/midtrans/midtrans-go` dependency + go.mod sync  
  - `go get github.com/midtrans/midtrans-go@latest && go mod tidy`
  - Verify: `go list -m github.com/midtrans/midtrans-go`

- [ ] **1.2** Create migration `015_midtrans_payment_qris.sql`  
  - Add columns to `payment_transactions`: `qris_string TEXT`, `qris_expiry TIMESTAMPTZ`
  - Backfill existing rows: `qris_string = NULL`, `qris_expiry = NULL`
  - Run migration locally against dev DB
  - Verify schema: `\d payment_transactions` in psql

- [ ] **1.3** Create `backend/go-core/internal/infra/midtrans/client.go`  
  - Wrap `midtrans.NewSnapV2Client(serverKey)` for Snap API
  - Wrap `midtrans.NewCoreAPIClient(serverKey, clientKey, isProduction)` for Core API (QRIS)
  - Implement `CreateChargeQRIS(orderId, amount, expiryMinutes) → (qrString, expiry, error)`
  - Implement `VerifyTransaction(orderId) → (status, amount, error)` (re-check server-to-server)
  - Write unit test stubs (defer full mocking to 1.5)
  - Verify: `go test ./internal/infra/midtrans`

- [ ] **1.4** Create `backend/go-core/internal/handler/payment_webhook_handler.go`  
  - Implement `HandleMidtransWebhook(w http.ResponseWriter, r *http.Request)`
  - Parse webhook body (order_id, status_code, gross_amount, signature_key)
  - Verify SHA512 signature: `sha512.Sum512(order_id+status_code+gross_amount+serverKey)`
  - On success: call `verifyTransactionStatus` server-to-server (re-check, best practice)
  - If paid (`capture|settlement`): call `markInvoicePaid` from payment_store
  - Return JSON `{ok:true}` or error with 403 (bad signature) / 502 (verify failed)
  - Unit test: mock webhook payload + signature check
  - Verify: `go test ./internal/handler -run TestHandleMidtransWebhook`

- [ ] **1.5** Wire Midtrans webhook route in `main.go`  
  - Add `r.Post("/webhooks/midtrans", paymentWebhookHandler.HandleMidtransWebhook)` under `/api/v1`
  - Test route: curl `-X POST http://localhost:8081/api/v1/webhooks/midtrans -H "Content-Type: application/json" -d '{...}'`

- [ ] **1.6** Create `.env.example` entries for Midtrans (if not present)  
  - `MIDTRANS_SERVER_KEY=` (for production/demo Midtrans merchant)
  - `MIDTRANS_CLIENT_KEY=`
  - `MIDTRANS_IS_PRODUCTION=false` (default sandbox)
  - Verify: `cat .env.example | grep MIDTRANS`

---

### Phase 2: Backend POS Payment Integration

- [ ] **2.1** Modify `pos_handler.go:UpdateCheckout` to accept payment method + amount  
  - Add field `PaymentMethod string` and `PaymentAmount float64` to request body
  - For non-CASH: call `midtrans.CreateChargeQRIS(orderId, amount, 5)` → return QR string + expiry
  - Store in `payment_transactions`: order_id, amount, status='pending', qris_string, qris_expiry
  - Return JSON: `{order_id, qris_string, qris_expiry, ...existing fields}`
  - Unit test: mock Midtrans charge success + failure
  - Verify: `go test ./internal/handler -run TestUpdateCheckoutWithPayment`

- [ ] **2.2** Modify `pos.Checkout()` in usecase to gate stock deduction on payment status  
  - Before `inventoryRepo.UpdateQuantity()` and `wmsRepo.CreateStockMovement()`:
    - If `paymentMethod != CASH`: check `payment_transactions.status` is `completed|settlement|capture`
    - If not paid, return error (don't deduct stock)
  - Unit test: mock stock deduction only happens after paid
  - Verify: `go test ./internal/usecase/pos -run TestCheckoutStockGate`

- [ ] **2.3** Add POS status endpoint `GET /api/v1/pos/payments/{orderId}/status`  
  - Query `payment_transactions` by order_id
  - Call `midtrans.VerifyTransaction(orderId)` (server-to-server re-check)
  - Return JSON: `{status, amount, payment_method, completed_at}`
  - Polling safety: idempotent, no side effects
  - Unit test: verify response matches transaction + Midtrans API
  - Verify: `curl http://localhost:8081/api/v1/pos/payments/ORD123/status`

---

### Phase 3: Frontend Onboarding Wizard (Settings → Payment Gateway)

- [ ] **3.1** Create `app/dashboard/payment-gateways/components/PaymentGatewayWizard.tsx`  
  - 3 steps: Step1 (pilih gateway kartu), Step2 (tempel key), Step3 (tes Rp1000)
  - Step1: show kartu Midtrans (+ Pakasir fallback) dengan fee + link daftar + status badge
  - Step2: form ServerKey + ClientKey, button "Test Koneksi", error summary + inline field error
  - Step3: create Rp1000 charge, poll status tiap 1s, show Loading/Berhasil/Gagal, set wizard state LIVE if success
  - Accessibility: `role="alert"` for error summary, `aria-live="polite"` for status, tab-navigable form
  - Design tokens: primary #2563EB, error #DC2626, success #10B981, tap target ≥48px
  - Verify: render wizard in Storybook (if exists) or test page

- [ ] **3.2** Replace `payment-gateways/page.tsx` form with wizard component  
  - Remove old form (settlement_bank_* fields, all inputs)
  - Import wizard component, render in modal or full-page
  - Add "Close" / "Back" buttons
  - Call `GET /api/v1/payments/configs?provider=midtrans` to prefill existing config (show current keys masked)
  - On wizard success: POST config to backend, show "Configured ✓" toast, redirect to dashboard
  - Verify: load page, see wizard, test all 3 steps

- [ ] **3.3** Add loading skeleton `payment-gateways/loading.tsx`  
  - Suspend boundary for wizard + card skeleton
  - Verify: build, check for hydration errors

---

### Phase 4: Frontend POS Payment Modal (Real QRIS)

- [ ] **4.1** Create `app/(app)/pos/hooks/use-pos-payment.ts`  
  - Hook `usePOSPayment()`: manages charge creation + polling state
  - Methods:
    - `createCharge(orderId, amount, method) → {qrString, expiry}`
    - `pollStatus(orderId, intervalMs=3000, maxTimeMs=5*60*1000) → {status, amount}`
    - `reset()` to clear state
  - State: `{loading, error, qrString, expiry, status, remaining_ms}`
  - Cleanup: cancel polling on unmount
  - Unit test: mock `fetch /api/pos/payments`, verify polling stops
  - Verify: `npm test -- pos.test.ts`

- [ ] **4.2** Modify `app/(app)/pos/page.tsx:handleOpenPayment`  
  - Remove hardcoded `qrisPayload` at line 362
  - For QRIS: call `usePOSPayment().createCharge(orderId, grandTotal, 'QRIS')`
  - For CASH: keep existing flow (manual cashTendered input)
  - Start polling in modal component (not in handler)
  - Verify: function signature unchanged, just delegates to hook

- [ ] **4.3** Refactor `POS QRISModal` to use real QR + polling  
  - Show real `qrString` from charge (not dummy payload)
  - Show countdown timer (expiry) with warning at 1min remaining
  - Poll status every 3s, update `role="status"` aria-live region: "Menunggu pembayaran…" → "Pembayaran diterima" (green) / "Kadaluwarsa" (yellow) / "Gagal" (red)
  - Buttons:
    - Lunas: show "Cetak Struk" + auto-close modal after 2s
    - Kedaluwarsa: show "Buat Kode Baru" (call createCharge again)
    - Gagal: show "Coba Lagi" + "Ganti ke Tunai"
    - Timeout (5min, no status): show "Gateway Sibuk, coba lagi atau pakai tunai" + fallback CASH button
  - Design: Minimalism/Swiss, #2563EB primary, #EA580C CTA, light/dark mode, Outfit font, no emoji
  - Accessibility: color-independent status (text labels), keyboard nav, tab-trapped modal
  - Verify: Playwright test, 3 scenarios (lunas, kedaluwarsa, gagal)

- [ ] **4.4** Remove manual "Konfirmasi Pembayaran QRIS Berhasil" button (line 1033)  
  - For QRIS: button disabled, auto-advances on poll `status=settlement`
  - For CASH: button remains "Konfirmasi Pembayaran Tunai"
  - Verify: build, visual check in POS page

---

### Phase 5: Integration & Testing

- [ ] **5.1** End-to-end: wizard → POS → settlement  
  - Playwright test: 
    1. Login tenant
    2. Settings → Payment Gateway → wizard → enter keys → test Rp1000 → LIVE
    3. Navigate POS, add product, open payment, select QRIS
    4. Wait for charge, mock webhook settlement (send test payload from Midtrans dashboard or curl)
    5. Verify status updates to "paid"
    6. Verify stock was NOT deducted until paid (regression test)
    7. Close modal, verify struk printed (or queued)
  - Run: `npm run test:e2e -- payment.spec.ts`
  - Verify: all steps pass, no timeout

- [ ] **5.2** Webhook idempotency test  
  - Send same webhook twice with same order_id
  - Verify: invoice marked paid once (no double-settle)
  - Verify: stock deducted once
  - Query DB: count payments with same order_id = 1, invoice.status = PAID
  - Verify: `go test ./internal/handler -run TestWebhookIdempotency`

- [ ] **5.3** Tenant isolation test (RLS)  
  - Create 2 tenants, assign different Midtrans keys
  - Tenant A charges, Tenant B tries to query A's payment_transactions
  - Verify: Tenant B sees empty list (RLS blocks cross-tenant reads)
  - Query DB with `app.current_tenant_id = TenantB_ID`: `SELECT * FROM payment_transactions WHERE tenant_id = TenantA_ID;` → 0 rows
  - Verify: `go test ./internal/infra/postgres -run TestTenantIsolation`

- [ ] **5.4** Error recovery test  
  - Midtrans API 500 error: charge fails, modal shows "Gateway Sibuk…", offer fallback tunai
  - Webhook signature bad: handler returns 403, invoice NOT marked paid
  - Amount mismatch in webhook: handler returns 400, invoice stays pending
  - Test each: curl webhook with bad sig / amount, verify response + DB state
  - Verify: `go test ./internal/handler -run TestWebhookErrors`

---

### Phase 6: Migration & Cleanup (Existing Tenants)

- [ ] **6.1** Mark Pakasir config as deprecated in settings UI  
  - Add banner: "Pakasir tidak lagi didukung untuk POS baru. Silakan migrasi ke Midtrans."
  - Keep config readable (tenant can still see old keys), but wizard only offers Midtrans
  - Existing Pakasir subscriptions still work (no breaking change)
  - Verify: dashboard shows banner when `provider=pakasir`

- [ ] **6.2** Test backward compat: existing Pakasir LIVE tenant  
  - Deploy code, verify old tenant can still use Pakasir for subscriptions (if they have active subscription)
  - Verify: no regressions in subscription billing flow
  - Query: SELECT * FROM tenant_payment_configs WHERE provider='pakasir' AND is_active=true; → should have rows

---

### Phase 7: Documentation & Commit

- [ ] **7.1** Write inline code comments  
  - `midtrans/client.go`: explain Core API vs Snap, why re-verify server-to-server
  - `payment_webhook_handler.go`: explain SHA512 sig + idempotency guard
  - `use-pos-payment.ts`: explain polling cleanup, expiry countdown
  - POS modal: explain accessibility roles + countdown logic
  - Verify: no TODOs left (only TBD future enhancements)

- [ ] **7.2** Update docs/api/payments.md or create docs/integration/midtrans-setup.md  
  - How to register Midtrans account + get keys
  - Webhook URL to register: `https://tayooli.my.id/api/v1/webhooks/midtrans`
  - Sandbox vs production toggle (`MIDTRANS_IS_PRODUCTION`)
  - Tenant onboarding flow (3-step wizard)
  - Example: curl POS charge + webhook test
  - Verify: docs render in `docs/` folder

- [ ] **7.3** Commit & review  
  - Atomic commits per task group (Phase 1 backend infra, Phase 2 POS, Phase 3 wizard, Phase 4 modal, Phase 5 tests, Phase 6 migration, Phase 7 docs)
  - Commit message: `feat(payments): add Midtrans BYO gateway for POS and invoices`
  - Include co-author if pair-coded
  - Verify: `git log --oneline` shows clean history

- [ ] **7.4** Code review checklist  
  - [ ] All tests pass: `go test ./... && npm test`
  - [ ] No hardcoded keys (env only)
  - [ ] No console.logs in prod code
  - [ ] Webhook endpoint handles all Midtrans status codes (settlement, capture, pending, expire, deny, cancel)
  - [ ] POS modal accessible (WCAG 2.2 AA, tested with keyboard nav + screen reader)
  - [ ] RLS enforced: tenant config/transactions scoped by tenant_id
  - [ ] Dashboard shows 0 breaking changes for existing Pakasir tenants
  - Verify: all checkboxes ✓

---

## Estimated Effort

- **Backend:** ~4–6h (infra + handler + usecase changes + tests)
- **Frontend:** ~5–7h (wizard + modal + hook + tests)
- **Testing & docs:** ~2–3h
- **Total:** ~12–16h (can parallelize backend + frontend = ~8–10h wall-clock)

## Success Criteria (from spec §12)

1. ✓ Tenant baru LIVE <5 menit (wizard 3 langkah lolos)
2. ✓ 0 struk/stok sebelum settlement terverifikasi server-to-server
3. ✓ Dana QRIS masuk rekening tenant (BYO proven), bukan platform
4. ✓ 0 regresi Pakasir LIVE tenant
5. ✓ Aksesibilitas: error summary fokus, live status terbaca, tap ≥44pt

---

**Next:** Approved? Pick execution method:
- **Native:** Saya execute tiap task inline (faster, riskier jika banyak context-switch)
- **Subagent-driven:** Delegate parallel tasks to subagents (slower start, better isolation + parallel)

Pilih mana?
