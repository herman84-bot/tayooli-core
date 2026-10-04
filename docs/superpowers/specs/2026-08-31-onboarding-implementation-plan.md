# Tayooli ERP — Onboarding Wizard Implementation Plan

**Date:** 2026-08-31
**Status:** Approved Spec
**Priority:** Anti-AI-Slop

---

## Anti-AI-Slop Rules

### DILARANG:
- Gradient besar, glassmorphism, neon, glow
- Floating blobs, animated shapes
- Selamat datang di perjalanan Anda
- Mulai petualangan bisnis Anda
- Solusi cerdas untuk...
- Tingkatkan produktivitas...
- Icon dekoratif yang tidak perlu
- Badge AI-powered Smart Next-gen
- Color palette di luar green existing
- Animation berlebihan
- Copy marketing yang tidak jelas

### WAJIB:
- Copy langsung ke inti
- Professional B2B tone
- Hijau konsisten (#16a34a)
- Whitespace sebagai design element
- Typography hierarchy yang jelas
- Minimal, restrained, credible

---

## Phase 1: Backend Registration Auth

### 1.1 Database Migration

File: migrations/017_user_registration.sql

ALTER TABLE users ADD COLUMN full_name, email_verified_at, verification_token, verification_expires_at

### 1.2 Registration Endpoint

File: internal/handler/auth_handler.go

POST /api/v1/auth/register

Request: full_name, email, password
Response 201: message, email
Response 409: email sudah terdaftar

Logic:
1. Validate input
2. Check email uniqueness
3. Hash password bcrypt cost 12
4. Insert user status unverified
5. Generate verification_token UUID
6. Set expiry now + 24h
7. Send verification email async
8. Return success

### 1.3 Email Verification

GET /api/v1/auth/verify-email?token=xxx

Logic:
1. Find user by token
2. Check expiry 24h
3. Update email_verified_at
4. Clear token
5. Set auth cookie
6. Return success

### 1.4 Resend Verification

POST /api/v1/auth/resend-verification

Logic:
1. Find user by email
2. Rate limit 3x per hour
3. Generate new token
4. Send email async

### 1.5 Email Sender

File: internal/infra/email/sender.go

SMTP config via env: SMTP_HOST PORT USER PASS FROM

### 1.6 Login Modification

Add check: if email_verified_at nil return 403

---

## Phase 2: Frontend Registration

### 2.1 Register Page

File: app/register/page.tsx

Layout: centered card, Logo, Step 1/5, RegisterForm, Link to login

### 2.2 RegisterForm

File: components/auth/RegisterForm.tsx

Fields: Nama Lengkap, Email, Kata Sandi
Validation: Zod schema
Submit: POST /api/v1/auth/register
On success: redirect /verify-email?email=xxx

Design: bg-primary button, no icons di input, no decoration

---

## Phase 3: Frontend Email Verification

### 3.1 Verify Email Page

File: app/verify-email/page.tsx

States: Loading, Success, Error, Pending
Layout: centered card, Step 2/5, Mail icon, email display, Resend button

### 3.2 Auto-verify on mount

Read token from URL, call API, on success redirect workspace

---

## Phase 4: Frontend Workspace Setup

### 4.1 Workspace Page

File: app/onboarding/workspace/page.tsx

Layout: centered card, Step 3/5, WorkspaceForm

### 4.2 WorkspaceForm

File: components/onboarding/WorkspaceForm.tsx

Field: Nama Perusahaan only
Submit: POST /api/v1/workspaces
On success: redirect /onboarding/trial

---

## Phase 5: Frontend Trial Activated

### 5.1 Trial Page

File: app/onboarding/trial/page.tsx

Layout: centered card, Step 4/5 completed, checkmark, Trial 14 hari aktif
Button: Masuk ke Dashboard
Link: Lihat harga

### 5.2 Auto-create trial

Call POST /api/v1/subscriptions on mount

---

## Phase 6: Backend Workspace Subscription

### 6.1 Create Workspace

POST /api/v1/workspaces

Logic: get user, create tenant, link user as owner

### 6.2 Trial Subscription

POST /api/v1/subscriptions modified

Logic: create trial, set 14 days, set limits

---

## Phase 7: Frontend Tutorial

### 7.1 Tutorial Component

File: components/tutorial/TooltipWalkthrough.tsx

Logic: check localStorage, show 5 tooltips, set flag on complete

---

## Phase 8: Integration Testing

### 8.1 Login Modification

Add Belum punya akun Daftar link

### 8.2 Route Protection

middleware.ts rules for onboarding flow

### 8.3 Testing

10 point checklist

---

## Implementation Order

1. Backend migration + register
2. Backend email verification
3. Backend workspace
4. Backend subscription modify
5. Frontend register page
6. Frontend verify email
7. Frontend workspace
8. Frontend trial
9. Frontend tutorial
10. Integration login link
11. Route protection
12. Testing

---

## Copy Guidelines Anti-AI-Slop

Register: Buat Akun, Isi data di bawah untuk mulai
Verify: Cek Email Anda, Kami kirim link verifikasi
Workspace: Nama Perusahaan, Tampilan di dashboard dan invoice
Trial: Trial Aktif, 14 hari. Semua fitur. Tanpa kartu kredit
Tutorial: Ini dashboard Anda, Upload invoice pertama

---

*Plan ready. No AI slop. Professional B2B copy.*