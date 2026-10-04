# Tayooli ERP — Onboarding Wizard Design Spec

**Date:** 2026-08-31
**Status:** Approved
**Author:** Buffy (CTO Agent)

---

## 1. Overview

### Problem
Current login page only supports email+password login. No registration flow exists. Users cannot self-onboard.

### Goal
Implement a complete 5-step onboarding wizard that guides new users from registration to productive dashboard usage.

### Target Users
- Business owners
- Finance managers
- Direct users

---

## 2. Architecture

### Frontend Flow

/landing → /register → /verify-email → /onboarding/workspace → /onboarding/trial → /dashboard

### Backend Endpoints (New)

| Endpoint | Method | Purpose |
|---|---|---|
| /api/v1/auth/register | POST | Create new user account |
| /api/v1/auth/verify-email | GET | Verify email via token |
| /api/v1/auth/resend-verification | POST | Resend verification email |
| /api/v1/workspaces | POST | Create tenant/workspace |
| /api/v1/subscriptions | POST | Create trial subscription |

---

## 3. Step-by-Step Design

### Step 1: Register (/register)

Fields: Nama Lengkap, Email, Kata Sandi
UI: Centered card, Step indicator 1/5, Green button, Link to login

### Step 2: Email Verification (/verify-email)

States: Pending, Success, Error, Expired
UI: Email icon, email address, Resend button with cooldown

### Step 3: Workspace Setup (/onboarding/workspace)

Fields: Nama Perusahaan only
UI: Centered card, Step indicator 3/5, Continue button

### Step 4: Trial Activated (/onboarding/trial)

UI: Success animation, Trial 14 days info, Go to Dashboard button

### Step 5: Tutorial (Dashboard)

UI: 5-step tooltip walkthrough, Skip link
Storage: localStorage flag

---

## 4. Error Handling

| Scenario | Handling |
|---|---|
| Email already registered | Link to /login |
| Invalid token | Resend button |
| Expired token | Auto-resend |
| Network error | Retry button |

---

## 5. Responsive

| Viewport | Behavior |
|---|---|
| Desktop 1366+ | Centered card, max 480px |
| Tablet 768-1365 | Full width card |
| Mobile 360-767 | Stack layout, smaller text |

---

## 6. Security

- Password: bcrypt cost 12
- Token: UUID v4, 24h expiry
- Rate limiting: 5 register/IP/hour
- CSRF protection
- HttpOnly cookies

---

## 7. Files

New: register/page, verify-email/page, onboarding/workspace/page, onboarding/trial/page, RegisterForm.tsx, StepIndicator.tsx, auth_handler.go, register.go, email/sender.go

Modified: login/page (add daftar link), LoginForm.tsx (unverified error), login/route.ts (unverified check), dashboard/page (tutorial trigger), main.go (new routes)

---

## 8. Acceptance Criteria

1. User can register
2. Verification email sent
3. User can verify email
4. Workspace setup works
5. Trial auto-created
6. Dashboard with tutorial
7. Existing login works
8. Unverified users blocked
9. Responsive works
10. No console errors

---

*Spec approved. Ready for implementation.*