# Implementation Plan: Workspace Autonomous AI Copilot ("Tayooli Copilot")

**Date:** 2026-08-14  
**Spec Reference:** `docs/superpowers/specs/2026-08-14-workspace-ai-copilot-design.md`  
**Status:** Ready to Implement  

---

## 1. Overview
Implement an omnipresent, autonomous AI Copilot for Tayooli ERP that allows users to query, configure, and operate workspace settings (profile, team, payment gateways, master data) via natural language. The architecture features a Hybrid Action Engine (Gemini 2.5/3.5 Flash Function Calling in Next.js, Go REST API validation, PostgreSQL RLS), 3 autonomy levels (Advisory, Assisted, Autopilot), 18 security guardrails, multi-LLM hot-fallback, and zero-AI rule-based fallback.

---

## 2. Architecture Decisions
- **Orchestration Layer:** Next.js Route Handler (`app/api/copilot/chat/route.ts`) calling `@google/genai` with tools declaration, authenticated via session cookies.
- **Backend Single Source of Truth:** Go Chi API handles all database mutations; AI never connects to PostgreSQL directly.
- **Failover Hierarchy:** Primary (Gemini 2.5/3.5 Flash) → Secondary (Groq Llama 3.3 / Fallback Key) → Zero-AI Rule Matcher.
- **Human-in-the-Loop:** Level 2 Assisted requires explicit user button click on `ActionPreviewCard` before mutations execute.
- **Circuit Breaker:** 3 consecutive 429/5xx errors trigger 60s cooldown with transparent UI timer.

---

## 3. Tasks Breakdown

### Phase 1: Database & Backend Foundation
- [ ] **Task 1: Database Migration `015_tenant_ai_permissions.sql`**
  - Create table `tenant_ai_permissions` (autonomy_level, allowed_scopes, emergency_stop, updated_by).
  - Apply RLS policies with `current_setting('app.current_tenant_id')`.
  - Add metadata columns to `audit_logs` (actor_type, prompt_snippet, previous_state, new_state).
  - Files: `backend/go-core/migrations/015_tenant_ai_permissions.sql`
- [ ] **Task 2: Backend AI Permissions Handlers & Endpoints**
  - Add domain model & repository for `tenant_ai_permissions`.
  - Add handler `GET /api/v1/ai/permissions` and `PATCH /api/v1/ai/permissions`.
  - Enforce admin/owner RBAC and RLS.
  - Files: `backend/go-core/internal/domain/ai_permissions.go`, `internal/infra/postgres/ai_permissions_repo.go`, `internal/handler/ai_permissions_handler.go`, `cmd/api/main.go`.

### Checkpoint 1: Foundation
- [ ] Migration applies cleanly.
- [ ] Backend tests/build passes.

---

### Phase 2: Frontend Copilot Core & State Management
- [ ] **Task 3: Copilot State Management & Permission Hook**
  - Create Zustand store `hooks/useCopilot.ts` (messages, isOpen, pendingActions, activeProvider, cooldownTimer).
  - Create `hooks/useAIPermissions.ts` (TanStack Query for fetch & mutate AI permissions/kill switch).
  - Files: `hooks/useCopilot.ts`, `hooks/useAIPermissions.ts`.
- [ ] **Task 4: AI Knowledge Map & Tool Registry Schemas**
  - Create `lib/copilot/knowledge-map.ts` with structured ERP playbook (P2P, O2C, Accounting, Settings).
  - Create `lib/copilot/tools.ts` with Gemini Function Calling declarations & Zod schemas.
  - Create `lib/copilot/executor.ts` (client-side dispatcher calling Go API endpoints).
  - Files: `lib/copilot/knowledge-map.ts`, `lib/copilot/tools.ts`, `lib/copilot/executor.ts`.

---

### Phase 3: AI Orchestration & Failover Engine
- [ ] **Task 5: Next.js Copilot Chat API Route with Multi-LLM Relay**
  - Implement `POST /app/api/copilot/chat/route.ts`.
  - Wrap workspace data in `<workspace_data readonly="true">` to prevent indirect prompt injection.
  - Add try-catch fallback: if Gemini 429/timeout > 5s → fallback to secondary / rule matcher.
  - Files: `app/api/copilot/chat/route.ts`.
- [ ] **Task 6: Deterministic Zero-AI Rule-Based Fallback Engine**
  - Create `lib/copilot/fallback-matcher.ts` (regex/keyword matching for quick actions when AI is offline/limited).
  - Files: `lib/copilot/fallback-matcher.ts`.

### Checkpoint 2: Orchestration & Logic
- [ ] Tool calling schemas validate with Zod.
- [ ] Fallback matcher correctly identifies ERP quick actions.

---

### Phase 4: UI Components & ZenSpace Experience
- [ ] **Task 7: Action Preview Card Component**
  - Build `components/copilot/ActionPreviewCard.tsx` with old vs new diff view, risk badge, and `Type-to-Confirm` for destructive actions.
  - Files: `components/copilot/ActionPreviewCard.tsx`.
- [ ] **Task 8: Copilot Slide-Over Drawer & Omnipresent Launcher**
  - Build `components/copilot/CopilotDrawer.tsx` (420px width, chat stream, message history, clear chat).
  - Build `components/copilot/CopilotTrigger.tsx` (floating bottom-right button + `Ctrl+K` / `Cmd+K` global listener).
  - Build `components/copilot/ProviderStatusBadge.tsx` (🟢 Gemini / 🟡 Fallback / 🟠 Offline Mode).
  - Files: `components/copilot/CopilotDrawer.tsx`, `components/copilot/CopilotTrigger.tsx`, `components/copilot/ProviderStatusBadge.tsx`.
- [ ] **Task 9: AI Autonomy & Permission Settings Modal**
  - Build `components/copilot/AutonomySettingsModal.tsx` (Advisory / Assisted / Autopilot selector, granular scope checkboxes, Emergency Kill Switch button).
  - Files: `components/copilot/AutonomySettingsModal.tsx`.

---

### Phase 5: Shell Integration, Cache Invalidation & Testing
- [ ] **Task 10: App Shell Wiring & Undo Toast**
  - Mount `CopilotDrawer` and `CopilotTrigger` into `app/(app)/layout.tsx`.
  - Integrate 15-second Undo toast upon action execution.
  - Invalidate TanStack Query caches automatically per modified module.
  - Files: `app/(app)/layout.tsx`.
- [ ] **Task 11: Verification & Security Hardening**
  - Run `npx tsc --noEmit` to ensure zero TypeScript errors.
  - Verify RBAC denial on non-admin user trying to execute setting actions.
  - Verify Emergency Kill Switch blocks tool calling immediately.
  - Verify Circuit Breaker timer when 429 simulated.

### Checkpoint 3: Complete
- [ ] All 11 tasks completed.
- [ ] Full end-to-end user experience verified.
