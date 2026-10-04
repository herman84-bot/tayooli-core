# PRD: Subscription Billing System — Tayooli ERP

**Status:** Final  
**Date:** 2026-08-28  
**Target:** Production-ready subscription + AI customer service

---

## 1. Problem Statement

Tayooli ERP punya pricing tiers di landing page tapi **tidak ada billing system**. User bisa akses semua fitur tanpa batas. Trial tidak di-enforce. Tidak ada revenue collection.

**Yang harus dibangun:**
1. Subscription billing (tagih user untuk langganan)
2. Trial management (14 hari, auto-expire)
3. Plan enforcement (limit users, invoices, fitur)
4. AI customer service yang tidak terbedakan dari manusia

---

## 2. Confirmed Decisions

| # | Topik | Keputusan |
|---|---|---|
| 1 | Trial | 14 hari, tanpa extend. Habis = expired. |
| 2 | Refund | No refund (bulanan). Pro-rated (annual). |
| 3 | Enterprise sales | AI-powered. Tidak ada human sales. |
| 4 | PPN | Belum termasuk harga. Rp 199.000 + PPN 11%. |
| 5 | Multi-currency | IDR only. USD fase 2. |
| 6 | Email provider | Resend |
| 7 | LLM API | Groq (free tier, auto-detect model) |
| 8 | AI personality | 100% manusia. Training lengkap. |

---

## 3. Payment Gateway Separation

### FLOW 1: Subscription (User → Tayooli)

```
User langganan Tayooli → Bayar pakai PAKASIR
                          (QRIS / Virtual Account)

Gateway: Pakasir saja (pilihan tunggal)
Untuk: tagihan Starter, Bisnis, Enterprise
Tayooli terima uang dari user.
```

### FLOW 2: ERP Transaction (Customer → User)

```
Customer user beli barang → Bayar pakai PAKASIR atau MIDTRANS
                            (tergantung user pilih)

Gateway: Pakasir ATAU Midtrans (user konfigurasi)
Untuk: invoice penjualan, pembayaran vendor
User terima uang dari customer-nya.
```

### Ringkasan

| Flow | Gateway | Pilihan |
|---|---|---|
| User → Bayar Tayooli (subscription) | Pakasir | Wajib (satu-satunya) |
| Customer → Bayar User (ERP) | Pakasir atau Midtrans | User pilih salah satu |

---

## 4. Pricing Tiers

| Plan | Monthly | Annual (hemat 20%) | Users | Vendor | Invoice/bulan | OCR/bulan |
|---|---|---|---|---|---|---|
| **Trial** | Gratis | Gratis | 3 | 50 | 100 | 20 |
| **Starter** | Rp 199.000 | Rp 159.200 | 3 | 50 | 100 | 20 |
| **Bisnis** | Rp 449.000 | Rp 359.200 | 10 | ∞ | ∞ | ∞ |
| **Enterprise** | Custom | Custom | ∞ | ∞ | ∞ | ∞ |

**Harga belum termasuk PPN 11%.**

### Feature Matrix

| Feature | Trial | Starter | Bisnis | Enterprise |
|---|---|---|---|---|
| Dashboard | ✅ | ✅ | ✅ | ✅ |
| Invoice Management | ✅ | ✅ | ✅ | ✅ |
| Purchase Orders | ✅ | ✅ | ✅ | ✅ |
| Approval Workflows | Basic | Basic | Multi-step | Custom |
| OCR + AI Analysis | 20/bulan | 20/bulan | ∞ | ∞ |
| Payment Gateway | 1 gateway | 1 gateway | Multi | Custom |
| Accounting | ✅ | ✅ | ✅ | ✅ |
| Multi-entity | ❌ | ❌ | ❌ | ✅ |
| API Access | ❌ | ❌ | ✅ | ✅ |

---

## 5. Subscription Status Flow

```
TRIALING → (bayar) → ACTIVE
TRIALING → (14 hari tanpa bayar) → EXPIRED
ACTIVE → (tagihan tidak dibayar 7 hari) → PAST_DUE
ACTIVE → (user cancel) → CANCELLED (berakhir di akhir siklus)
PAST_DUE → (bayar) → ACTIVE
PAST_DUE → (7 hari tanpa bayar) → EXPIRED
CANCELLED → (akhir siklus) → EXPIRED
EXPIRED → (bayar) → ACTIVE
```

---

## 6. Business Logic

### 6.1 Trial

```
Hari 0: Daftar → trial aktif, semua fitur penuh
Hari 12: Email reminder "Trial berakhir 2 hari lagi"
Hari 14: Expired → blokir akses tulis → redirect /upgrade
          TIDAK ADA PERPANJANGAN
Hari 21: Full block
```

### 6.2 Payment (Subscription)

```
User klik "Pilih Starter"
  → Pilih bulanan/tahunan
  → POST /api/v1/subscription
  → Generate Pakasir payment link
  → User bayar (QRIS/VA)
  → Webhook diterima
  → Verifikasi server-to-server
  → Status = ACTIVE, plan = starter
```

### 6.3 Renewal

```
Siklus berakhir → generate invoice baru → kirim link pembayaran
3 hari tanpa bayar → status = PAST_DUE
7 hari tanpa bayar → status = EXPIRED
```

### 6.4 Upgrade

```
Starter → Bisnis: prorated, bayar selisih, langsung aktif
```

### 6.5 Downgrade

```
Bisnis → Starter: efektif akhir siklus, cek limit
```

### 6.6 Plan Enforcement (Middleware)

```
Setiap write request:
1. Cek status subscription (active/expired/cancelled)
2. Cek trial masih berlaku
3. Cek limit plan (users, invoices, OCR)
4. Cek fitur tersedia di plan
5. Jika semua lolos → lanjut
6. Jika ada gagal → blokir 403
```

---

## 7. Database Schema

```sql
CREATE TABLE tenant_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    plan VARCHAR(50) NOT NULL DEFAULT 'trial',
    status VARCHAR(30) NOT NULL DEFAULT 'trialing',
    trial_started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    trial_ends_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '14 days'),
    billing_period VARCHAR(10) NOT NULL DEFAULT 'monthly',
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    payment_provider VARCHAR(32) DEFAULT 'pakasir',
    cancel_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id)
);

CREATE TABLE subscription_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL REFERENCES tenant_subscriptions(id),
    invoice_number VARCHAR(50) NOT NULL UNIQUE,
    amount INTEGER NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    payment_link TEXT,
    payment_provider VARCHAR(32) DEFAULT 'pakasir',
    payment_reference VARCHAR(255),
    due_date TIMESTAMPTZ NOT NULL,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE plan_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan VARCHAR(50) NOT NULL UNIQUE,
    max_users INTEGER NOT NULL,
    max_vendors INTEGER NOT NULL,
    max_invoices_per_month INTEGER NOT NULL,
    max_ocr_per_month INTEGER NOT NULL,
    multi_entity BOOLEAN NOT NULL DEFAULT FALSE,
    api_access BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO plan_limits (plan, max_users, max_vendors, max_invoices_per_month, max_ocr_per_month, multi_entity, api_access) VALUES
('trial',     3,   50,  100,  20,  FALSE, FALSE),
('starter',   3,   50,  100,  20,  FALSE, FALSE),
('bisnis',   10,   -1,   -1,  -1,  FALSE, TRUE),
('enterprise', -1,  -1,   -1,  -1,  TRUE,  TRUE);
```

---

## 8. API Endpoints

```
# Subscription
GET    /api/v1/subscription              — Get current
POST   /api/v1/subscription              — Create (from trial)
PATCH  /api/v1/subscription              — Update plan
DELETE /api/v1/subscription              — Cancel

# Billing
GET    /api/v1/subscription/invoices     — List invoices
POST   /api/v1/subscription/invoices/:id/pay — Generate payment link

# Plan
GET    /api/v1/plans                     — List plans
GET    /api/v1/usage                     — Usage vs limits

# Webhooks
POST   /api/webhooks/subscription/pakasir — Payment notification

# AI
POST   /api/v1/ai/chat                   — Chat with Nara
```

---

## 9. AI Customer Service (Nara)

### Architecture

```
Chat Widget (Tiledesk)
  → Dify (AI Platform, self-hosted)
    → Groq LLM (auto-detect model)
    → Mem0 (memory)
    → RAG (knowledge base Tayooli)
```

### Personality

- Nama: Nara
- Gaya: Santai tapi profesional
- Bahasa: Indonesia natural
- Karakter: Sabar, detail-oriented, proactive

### Security

- Tidak bypass payment
- Tidak akses secrets
- Tidak generate plan baru
- Data dari knowledge base, bukan ngarang

### Groq Auto-Detect

- Fetch daftar model setiap 5 menit
- Auto-fallback jika model hilang
- Rate limit handling
- Tidak pernah berhenti

---

## 10. Tech Stack (Free/MVP)

| Komponen | Teknologi | Biaya |
|---|---|---|
| Backend | Go (Chi) | Existing |
| Frontend | Next.js 14 | Existing |
| Database | PostgreSQL 15 | Existing |
| AI Platform | Dify (self-hosted) | Gratis |
| AI Memory | Mem0 (self-hosted) | Gratis |
| Chat Widget | Tiledesk (self-hosted) | Gratis |
| LLM API | Groq (free tier) | Gratis |
| Payment (subscription) | Pakasir | Per transaksi |
| Payment (ERP user) | Pakasir atau Midtrans | Per transaksi |
| Email | Resend | 3k gratis/bulan |

---

## 11. Implementation Phases

### Phase 1: Schema + Backend Subscription
- Migration: tenant_subscriptions, subscription_invoices, plan_limits
- Extend Tenant domain model
- Subscription CRUD endpoints
- PlanGuard middleware
- Trial management

### Phase 2: Payment Integration (Pakasir only)
- Pakasir payment link untuk subscription
- Webhook handler
- Payment verification

### Phase 3: Frontend
- Registration → trial flow
- Upgrade page
- Billing dashboard
- Trial banners

### Phase 4: AI Customer Service
- Setup Dify + Mem0 + Tiledesk
- Train Nara persona
- Upload knowledge base
- Integrate Groq auto-detect

---

## 12. Success Metrics

| Metric | Target |
|---|---|
| Trial → Paid conversion | > 10% |
| Payment success rate | > 95% |
| Plan enforcement accuracy | 100% |
| AI resolution rate | > 70% |
| AI response time | < 2 detik |
