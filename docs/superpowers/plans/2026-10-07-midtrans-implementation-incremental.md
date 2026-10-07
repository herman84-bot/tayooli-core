# Midtrans BYO Gateway — Incremental Implementation (Opsi A)

**Approach:** 3 endpoint terpisah — charge (create QRIS) → poll (check status) → confirm (settle + checkout).  
**Zero regresi:** Existing `Checkout()` untuk CASH **tidak berubah sama sekali**.

---

## Increment 0: Prepare (no code change, just verify)

- [ ] Verify Go.mod and dependencies: `go list -m all | grep midtrans` (should be empty, we add it next increment)
- [ ] Verify migration 014 exists: `cat backend/go-core/migrations/014_payment_gateways.sql | grep payment_transactions` 
- [ ] Verify payment_gateway_repo.go has all 5 methods: GetConfig, UpsertConfig, CreateTransaction, GetTransactionByOrderID, UpdateTransactionStatus
- [ ] Verify pos.go Checkout() does stock deduction inline (line 128–143)
- [ ] Verify POS handler Checkout() calls pos.go Checkout() (line 24–55)

**Commit:** None (verification only)

---

## Increment 1: Add Midtrans SDK + Basic Client Wrapper

**What:** Install `github.com/midtrans/midtrans-go` + create `backend/go-core/internal/infra/midtrans/client.go`

**Files:**
- `backend/go-core/go.mod` — add dependency
- `backend/go-core/go.sum` — generated
- `backend/go-core/internal/infra/midtrans/client.go` — NEW wrapper

**Scope:** 
- Install SDK
- Create `MidtransClient` struct wrapping Snap + Core API clients
- Implement `NewMidtransClient(serverKey, clientKey, isProduction)`
- Implement `CreateChargeQRIS(orderId, amount, expiryMinutes) → (qrString, expiry, error)` — **stub with TODO comment**
- Write unit test stub (mock Midtrans response)

**Why this order?** Dependency first, so we can test import.

**Steps:**
1. `cd backend/go-core && go get github.com/midtrans/midtrans-go@latest && go mod tidy`
2. Create file `internal/infra/midtrans/client.go` with basic struct + NewMidtransClient
3. Write test skeleton `internal/infra/midtrans/client_test.go`
4. Run `go test ./internal/infra/midtrans` — should pass (stubs only)
5. Commit: `feat(payments): add midtrans-go dependency + basic client wrapper`

**Verify:** `go build ./cmd/api && go test ./internal/infra/midtrans`

---

## Increment 2: Implement Midtrans Core API QRIS Charge

**What:** Fill in `CreateChargeQRIS` to actually call Midtrans API

**Files:**
- `backend/go-core/internal/infra/midtrans/client.go` — implement CreateChargeQRIS method

**Scope:**
- Call Midtrans Core API `POST /v2/charge` with `payment_type: qris`
- Return qrString (from `actions[]` containing `generate-qr-code`)
- Return expiry (calculate from response + 5min default)
- Handle errors (invalid key, amount zero, API down)

**Why:** Most risky part — if Midtrans API contract wrong, we catch here.

**Steps:**
1. Read Midtrans Go SDK docs (via godoc or source)
2. Implement CreateChargeQRIS using Snap/Core API client
3. Write unit test: mock Midtrans response, verify qrString extracted
4. Run test: `go test -v ./internal/infra/midtrans -run TestCreateChargeQRIS`
5. Commit: `feat(payments): implement Midtrans QRIS charge via Core API`

**Verify:** `go test ./internal/infra/midtrans`

---

## Increment 3: Implement VerifyTransaction (Server-to-Server Check)

**What:** Implement server-to-server status verification (re-check best practice)

**Files:**
- `backend/go-core/internal/infra/midtrans/client.go` — add VerifyTransaction method

**Scope:**
- Call `GET /v2/{orderId}/status` to re-check actual status
- Return status (settlement, capture, pending, expire, deny, cancel)
- Return amount to verify no tampering
- Handle errors gracefully

**Why:** Security — never trust webhook body alone.

**Steps:**
1. Add VerifyTransaction method to MidtransClient
2. Test: mock response with various statuses
3. Run: `go test ./internal/infra/midtrans -run TestVerifyTransaction`
4. Commit: `feat(payments): add server-to-server transaction verification`

**Verify:** `go test ./internal/infra/midtrans`

---

## Increment 4: Implement Midtrans Webhook Handler

**What:** Create `backend/go-core/internal/handler/payment_webhook_handler.go` to handle Midtrans notifications

**Files:**
- `backend/go-core/internal/handler/payment_webhook_handler.go` — NEW
- `backend/go-core/internal/handler/payment_webhook_handler_test.go` — NEW tests

**Scope:**
- Handle `POST /api/v1/webhooks/midtrans`
- Parse webhook: order_id, status_code, gross_amount, signature_key
- Verify SHA512 signature: `sha512(order_id+status_code+gross_amount+serverKey)` == signature_key
- Call `midtrans.VerifyTransaction(orderId)` (re-verify)
- If paid (settlement or capture): call `paymentGatewayRepo.UpdateTransactionStatus(orderId, 'completed')`
- Return 200 OK or error

**Why:** Webhook is entry point for settlement confirmation.

**Steps:**
1. Create handler with SHA512 signature check
2. Test: mock webhook payload + bad signature → 403
3. Test: mock webhook payload + good signature → call repo
4. Run: `go test ./internal/handler -run TestHandleMidtransWebhook`
5. Commit: `feat(payments): implement Midtrans webhook handler with signature verification`

**Verify:** `go test ./internal/handler`

---

## Increment 5: Wire Webhook Route in main.go

**What:** Register `POST /api/v1/webhooks/midtrans` in Chi router

**Files:**
- `backend/go-core/cmd/api/main.go` — add route

**Scope:**
- Register handler under `/api/v1` routes
- No auth required (webhook comes from external Midtrans, signed)
- Keep existing `/api/v1/webhooks/pakasir` route

**Steps:**
1. Add import for payment_webhook_handler
2. Create handler instance with paymentGatewayRepo + midtransClient
3. Register route: `r.Post("/webhooks/midtrans", paymentWebhookHandler.HandleMidtransWebhook)`
4. Build: `go build ./cmd/api`
5. Commit: `feat(payments): wire Midtrans webhook route`

**Verify:** `go build ./cmd/api && grep "webhooks/midtrans" cmd/api/main.go`

---

## Increment 6: Add POS Payment Endpoints

**What:** Create endpoints for POS: charge → status poll → confirm

**Files:**
- `backend/go-core/internal/handler/pos_handler.go` — add 2 new methods
- `backend/go-core/internal/handler/pos_handler_test.go` — tests

**Endpoints:**
1. `POST /api/v1/pos/payments` → create charge, return QR + expiry
2. `GET /api/v1/pos/payments/{orderId}/status` → poll status (idempotent)

**Scope (NOT modifying Checkout yet):**
- Handler CreatePayment: accept orderId, amount, method, tenantID
- Call midtrans.CreateChargeQRIS
- Save to paymentGatewayRepo.CreateTransaction (status=pending)
- Return {orderId, qrString, qrsExpiry, method}
- Handler GetPaymentStatus: query paymentGatewayRepo.GetTransactionByOrderID
- Call midtrans.VerifyTransaction for live check
- Return {status, amount, completed_at}

**Why:** Separate concerns — payment charge is independent of stock/invoice, reduces coupling.

**Steps:**
1. Add CreatePayment handler method
2. Add GetPaymentStatus handler method
3. Register routes in main.go (under `/api/v1`)
4. Test: POST → check response, GET → check idempotency
5. Run: `go test ./internal/handler -run TestCreatePayment`
6. Commit: `feat(payments): add POS payment charge + status polling endpoints`

**Verify:** `go test ./internal/handler && go build ./cmd/api`

---

## Increment 7: Add POS Confirm Endpoint (Trigger Checkout)

**What:** Final step — trigger actual checkout + stock deduction after payment confirmed

**Files:**
- `backend/go-core/internal/handler/pos_handler.go` — add ConfirmPayment method
- Test

**Endpoint:**
- `POST /api/v1/pos/payments/{orderId}/confirm` → verify status + call Checkout

**Scope:**
- Query paymentGatewayRepo.GetTransactionByOrderID (status must be 'completed')
- Call pos.Checkout() with stored items + CASH method (payment already done)
- Stock deduction happens here (inside Checkout)
- Return CheckoutResponse (struk data)

**Why:** **Two-phase** — payment first (might fail), then checkout (must succeed given paid status).

**Steps:**
1. Add ConfirmPayment handler
2. Verify transaction status is 'completed' (gate check)
3. Call pos.Checkout() — reuse existing logic
4. Test: mock transaction pending → 400 error, transaction completed → success
5. Commit: `feat(payments): add POS payment confirmation endpoint`

**Verify:** `go test ./internal/handler && go build ./cmd/api`

---

## Increment 8: Add Migration for QRIS Fields (Optional)

**What:** If needed, extend payment_transactions schema with qris-specific fields

**Files:**
- `backend/go-core/migrations/032_payment_qris_fields.sql` — NEW

**Scope:**
- Add columns: `qris_string TEXT`, `qris_expiry TIMESTAMPTZ`
- Backfill existing rows: `qris_string = NULL`, `qris_expiry = NULL`
- Add comment for idempotency

**Why:** Optional — depends on whether we want to store QR for audit/receipt.

**Condition:** Implement only if POS modal needs to re-render same QR (recovery scenario).

---

## Increment 9: Frontend — Onboarding Wizard (3 Steps)

**What:** Replace form with wizard: Pilih → Tempel Key → Test

**Files:**
- `app/dashboard/payment-gateways/components/PaymentGatewayWizard.tsx` — NEW
- `app/dashboard/payment-gateways/page.tsx` — MODIFY to use wizard
- `app/dashboard/payment-gateways/loading.tsx` — NEW (skeleton)

**Scope:**
- Step 1: Show kartu Midtrans (fee, link daftar, status)
- Step 2: Form ServerKey + ClientKey, "Test Koneksi" button
- Step 3: Create Rp1000 charge, poll 1s, show result
- On success: POST config to backend, show toast "Configured ✓"

**Why:** Most user-facing — if wrong here, tenant confused.

**Steps:**
1. Create WizardComponent with 3-step state machine
2. Implement Step1: card selection
3. Implement Step2: form + test button (call POST /api/v1/payments/configs)
4. Implement Step3: charge Rp1000 + polling
5. Test: Playwright — step through all 3 steps
6. Commit: `feat(ui): add payment gateway onboarding wizard`

**Verify:** `npm test -- payment-gateways.spec.ts (Playwright)`

---

## Increment 10: Frontend — POS Modal + Hook

**What:** Replace dummy QRIS with real QR + polling

**Files:**
- `app/(app)/pos/hooks/use-pos-payment.ts` — NEW
- `app/(app)/pos/components/PaymentModal.tsx` — MODIFY
- `app/(app)/pos/page.tsx` — update handleOpenPayment

**Scope:**
- Hook: createCharge(), pollStatus(), reset()
- Modal: show real QR, countdown timer, status live-region
- Buttons: Lunas (auto-close), Kedaluwarsa (buat kode baru), Gagal (retry / tunai fallback)
- Accessibility: WCAG 2.2 AA (color-independent status, keyboard nav)

**Steps:**
1. Create hook with charge/poll logic
2. Refactor modal to use hook
3. Add polling cleanup on unmount
4. Test: Playwright — 3 scenarios (lunas, kedaluwarsa, gagal)
5. Commit: `feat(pos): implement real QRIS charge + polling modal`

**Verify:** `npm test -- pos.e2e.spec.ts`

---

## Increment 11: Integration Test (E2E)

**What:** Full flow: wizard → POS → webhook → settle

**Files:**
- `e2e/payment-integration.spec.ts` — NEW

**Scope:**
- Login tenant
- Settings → wizard → Midtrans key (sandbox)
- POS → add product → QRIS payment
- Mock webhook settle (curl or Midtrans sandbox)
- Verify status updates
- Verify struk printed (or returned in response)

**Steps:**
1. Write test scenario
2. Run against local backend + Zeabur sandbox
3. Test idempotency (send webhook 2x, verify no double-settle)
4. Commit: `test(e2e): add payment integration test`

**Verify:** `npm run test:e2e -- payment-integration.spec.ts`

---

## Increment 12: Documentation + ADR Update

**What:** Record decisions, setup guide

**Files:**
- `docs/integration/midtrans-setup.md` — NEW (how to register, get keys, webhook URL)
- `docs/adr/008-pakasir-payment-gateway.md` — APPEND note about Midtrans V1

**Scope:**
- Midtrans account creation steps
- Webhook URL: `https://tayooli.my.id/api/v1/webhooks/midtrans`
- Sandbox vs production toggle
- Tenant onboarding flow
- Example curl test for webhook

**Steps:**
1. Write setup guide
2. Update ADR-008 with Midtrans decision
3. Commit: `docs(payments): add Midtrans setup guide + ADR update`

---

## Execution Order

**Sequential (safe, simple):** 0 → 1 → 2 → 3 → 4 → 5 → 6 → 7 → 9 → 10 → 11 → 12

**Parallel (faster, if confident):**
- Backend path: 0 → 1 → 2 → 3 → 4 → 5 → 6 → 7 (1 developer)
- Frontend path: 9 → 10 → 11 (another developer)
- Meet at Increment 12 (docs)

**Recommended for you (solo):** Sequential, but **skip Increment 8** (migration) unless you really need to store QR in DB — the response from charge has it, that's enough.

---

## Success Criteria

After all increments:
- [ ] `go test ./...` — all backend tests pass
- [ ] `npm test` — all frontend tests pass
- [ ] `go build ./cmd/api` — no compile errors
- [ ] `npm run build` — Next.js builds
- [ ] E2E test passes: wizard → POS → settled → struk
- [ ] No regressions: existing CASH flow still works
- [ ] Tenant isolation: RLS blocks cross-tenant reads
- [ ] Webhook idempotency: double-send = single settle

---

**Ready to start Increment 0 (verification)?**
