# Tayooli ERP Architecture

## Core DNA
Multi-tenant B2B ERP with AI-driven autonomous finance (Paper.id style).

## Tech Stack
- Backend: Go (Chi router, Hexagonal/Clean Architecture)
- AI Worker: Python (FastAPI, gRPC, OCR/LLM)
- Frontend: Next.js 14 (App Router), Zustand, TanStack Query, Tailwind
- DB: PostgreSQL 15 (Row-Level Security for multi-tenancy)
- Broker: Apache Kafka (KRaft mode)

## DB Schema Core (Postgres)
- `tenants` (id UUID PK, name, plan)
- `users` (id UUID PK, tenant_id FK, email, role)
- `invoices` (id UUID PK, tenant_id FK, vendor_id, amount, status, ai_confidence_score)
- `audit_logs` (id UUID PK, tenant_id FK, entity, action, prev_hash, current_hash)

## Key Workflows
- Order-to-Pay: Ingestion -> Python OCR -> Go 3-way match -> Kafka Event -> Auto-approve.
