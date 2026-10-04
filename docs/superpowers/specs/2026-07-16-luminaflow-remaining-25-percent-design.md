# LuminaFlow ERP — Rencana Penyelesaian Sisa 25% Fitur

**Tanggal:** 2026-07-16  
**Versi:** 1.0  
**Status:** Approved for Implementation

---

## 📋 Ringkasan Eksekutif

Proyek LuminaFlow ERP sudah mencapai ~75% kelengkapan. Dokumen ini merincikan rencana implementasi untuk 5 modul sisa yang akan diselesaikan secara berurutan:

1. **Wire Approval Routes + Auth System** — JWT cookie-based auth + approval workflow endpoints
2. **gRPC/AI Serving Integration** — Hubungkan Go backend dengan Python AI service (FastAPI/Celery)
3. **Accounting Detail UI** — Chart of Accounts & Journal Entry management dengan drawer pattern
4. **Integration Tests & Hardening** — E2E tests, rate limiting, CORS, health checks
5. **Production Readiness** — Logging, monitoring, config validation, CI/CD pipeline

Setiap modul akan dikerjakan dengan agent/skill yang ditentukan:
- `frontend-design` → UI/UX design
- `executing-plans` → Implementasi kode
- `systematic-debugging` → Debugging & testing
- `dev-pipeline` → CI/CD & infra
- `z-audit` → Security review (wajib setiap tahap)

---

## 🎨 Design System: Icon Style (Claude Web-Inspired)

Semua icon menggunakan **Lucide React** dengan karakteristik:

| Property | Value |
|----------|-------|
| Stroke width | 1.5px |
| Caps | Rounded |
| Default size | 20px (`w-5 h-5`) |
| Compact size | 16px (`w-4 h-4`) |
| Emphasis size | 24px (`w-6 h-6`) |
| Color | `currentColor` (inherit) |
| Style | Clean, minimal, geometric |

**Mapping Icon:**

| Feature | Lucide Icon | Size Class |
|---------|-------------|------------|
| Dashboard | `LayoutDashboard` | `w-5 h-5` |
| Invoice | `FileText` | `w-5 h-5` |
| Purchase Order | `ClipboardList` | `w-5 h-5` |
| Goods Receipt | `PackageCheck` | `w-5 h-5` |
| Payment | `Banknote` | `w-5 h-5` |
| Vendor | `Building2` | `w-5 h-5` |
| Customer | `Users` | `w-5 h-5` |
| Sales | `TrendingUp` | `w-5 h-5` |
| Accounting | `Calculator` | `w-5 h-5` |
| Approval | `CheckCircle` | `w-5 h-5` |
| Settings | `Settings` | `w-5 h-5` |
| Logout | `LogOut` | `w-5 h-5` |
| Search | `Search` | `w-4 h-4` |
| Filter | `Filter` | `w-4 h-4` |
| Create/Add | `Plus` | `w-5 h-5` |
| Edit | `Pencil` | `w-4 h-4` |
| Delete | `Trash2` | `w-4 h-4` |
| Back | `ArrowLeft` | `w-4 h-4` |
| Menu | `Menu` | `w-5 h-5` |

**Komponen Wrapper:**
```tsx
// components/ui/icon.tsx
import * as Lucide from "lucide-react"

type IconName = keyof typeof Lucide

interface IconProps {
  name: IconName
  size?: "sm" | "md" | "lg"
  className?: string
}

const sizeMap = { sm: "w-4 h-4", md: "w-5 h-5", lg: "w-6 h-6" }

export function Icon({ name, size = "md", className }: IconProps) {
  const IconComponent = Lucide[name]
  if (!IconComponent) return null
  return <IconComponent className={`${sizeMap[size]} text-current ${className}`} />
}
```

---

## 🔧 MODUL 1: Wire Approval Routes + Auth System

### 1.1 Backend (Go)

**Files to Create/Modify:**

| File | Action | Description |
|------|--------|-------------|
| `internal/handler/auth_handler.go` | Create | Login, Me, Logout handlers |
| `internal/usecase/auth/auth_usecase.go` | Create | Auth business logic |
| `internal/infra/postgres/user_repo.go` | Extend | Add GetByEmail, VerifyPassword |
| `internal/middleware/auth_middleware.go` | Modify | Read JWT from cookie `lumina_auth` |
| `cmd/api/main.go` | Modify | Register auth routes + approval routes |
| `internal/handler/approval_handler.go` | Create | Approval CRUD + approve/reject |
| `internal/usecase/approval/approval_usecase.go` | Create | Approval workflow logic |

**Endpoints:**

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/login` | Public | Login, set HTTP-only cookie |
| GET | `/api/v1/auth/me` | Cookie | Get current user profile |
| POST | `/api/v1/auth/logout` | Cookie | Clear cookie |
| GET | `/api/v1/approvals` | Cookie | List approvals (tenant-scoped) |
| GET | `/api/v1/approvals/{id}` | Cookie | Get approval detail |
| POST | `/api/v1/approvals/{id}/approve` | Cookie | Approve request |
| POST | `/api/v1/approvals/{id}/reject` | Cookie | Reject request |

**Cookie Config:**
```go
http.SetCookie(w, &http.Cookie{
    Name:     "lumina_auth",
    Value:    token,
    Path:     "/",
    HttpOnly: true,
    Secure:   true,           // HTTPS only in production
    SameSite: http.SameSiteStrictMode,
    MaxAge:   3600,           // 1 hour
})
```

### 1.2 Frontend (Next.js)

**Files to Create:**

| File | Description |
|------|-------------|
| `app/login/page.tsx` | Login form page |
| `hooks/useAuth.ts` | Zustand store + auth hook |
| `components/auth/LoginForm.tsx` | Reusable login form |
| `app/approvals/page.tsx` | Approvals list page |
| `components/approvals/ApprovalDrawer.tsx` | Drawer approve/reject |

**Auth Store (Zustand):**
```typescript
// hooks/useAuth.ts
interface User { id: string; email: string; role: string; tenantId: string }
interface AuthState {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  hydrate: () => Promise<void>  // calls /auth/me
}
```

**UI Components (Claude-style):**
- Login page: centered card, subtle shadow, Lucide icons for email/lock
- Drawer: slide from right, header with icon + title, form inside
- All buttons: `bg-primary text-primary-foreground hover:bg-primary/90`
- Inputs: `border-input focus:ring-2 focus:ring-ring`

---

## 🤖 MODUL 2: gRPC/AI Serving Integration

### 2.1 Architecture

```
┌─────────────┐     HTTP/JSON      ┌──────────────────┐
│  Go Backend │ ─────────────────► │  FastAPI Service │
│  (Client)   │   /inference/*     │  (ai_serving)    │
└─────────────┘                    └────────┬─────────┘
                                            │ Celery
                                            ▼
                                     ┌──────────────┐
                                     │   Redis      │
                                     │  (Broker)    │
                                     └──────────────┘
```

### 2.2 Go Client

**New Files:**
- `internal/infra/ai/ai_client.go` — HTTP client ke FastAPI
- `internal/usecase/ai/ai_usecase.go` — Business logic inference

**Endpoints Go:**
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/inference/ingest` | Kirim invoice untuk anomaly detection |
| GET | `/api/v1/inference/status/{invoice_id}` | Poll status hasil |

**Request/Response:**
```go
// POST /inference/ingest
type IngestRequest struct {
  InvoiceID   string `json:"invoice_id"`
  TenantID    string `json:"tenant_id"`
  Amount      string `json:"amount"`
  VendorID    string `json:"vendor_id"`
  ExtractedText string `json:"extracted_text"`
}

// Response
type IngestResponse struct {
  JobID string `json:"job_id"`
  Status string `json:"status"` // "queued"
}

// GET /inference/status/{invoice_id}
type StatusResponse struct {
  InvoiceID   string  `json:"invoice_id"`
  Status      string  `json:"status"` // "completed" | "failed" | "processing"
  AnomalyScore float64 `json:"anomaly_score,omitempty"`
  SuggestedGLAccount string `json:"suggested_gl_account,omitempty"`
  Error       string  `json:"error,omitempty"`
}
```

### 2.3 Python Service (Existing)

**Verify `ai_serving_service.py` has:**
- FastAPI endpoints: `/ingest`, `/status/{invoice_id}`
- Celery task: `run_anomaly_detection`, `suggest_gl_account`
- Redis result backend

**If missing → implement:**
- `POST /ingest` → queue Celery task, return `job_id`
- `GET /status/{invoice_id}` → check Celery AsyncResult

---

## 📊 MODUL 3: Accounting Detail UI

### 3.1 Pages (Drawer Pattern)

| Route | Component | Drawer Purpose |
|-------|-----------|----------------|
| `/accounting/chart-of-accounts` | `ChartOfAccountsPage` | List CoA, "Create" → drawer form |
| `/accounting/journal-entries` | `JournalEntriesPage` | List JE, "Create" → drawer form, click row → drawer detail |
| `/accounting/journal-entries/[id]` | — | Redirect to list with drawer open |

### 3.2 Components

| Component | Description |
|-----------|-------------|
| `CoATable` | TanStack Table: Code, Name, Type, Balance |
| `CoADrawer` | Create/Edit: Code, Name, Type (Asset/Liability/Equity/Revenue/Expense), Parent Account |
| `JETable` | TanStack Table: Date, Reference, Description, Total Debit/Credit |
| `JEDrawer` | Create: multiple line items (account + debit/credit), running balance validation |

### 3.3 Validation Rules
- Journal Entry: Total Debit == Total Credit (client + server)
- Account Type: Balance direction (Asset/Expense = Debit normal, Liability/Equity/Revenue = Credit normal)
- CoA Code uniqueness per tenant

---

## 🧪 MODUL 4: Integration Tests & Hardening

### 4.1 E2E Test Scenarios (Playwright)

| Scenario | Steps | Expected |
|----------|-------|----------|
| O2P Full Cycle | Login → Create PO → Create GR → Create Invoice → 3-Way Match → Approve → Payment | Invoice auto-approved if match pass |
| O2C Full Cycle | Login → Create Customer → Create Sales Order → Create Sales Invoice → Payment | Revenue recognized |
| Auth Flow | Visit /dashboard → redirect /login → login → redirect /dashboard | Token in cookie, Zustand hydrated |
| Approval Flow | Create invoice > threshold → approval created → approver approve → invoice approved | Status changes correctly |

### 4.2 Hardening Checklist

| Area | Implementation |
|------|----------------|
| Rate Limiting | `chi/middleware.Throttle(100, time.Minute)` per IP + tenant |
| CORS | Allow only `FRONTEND_ORIGIN` env, credentials: true |
| Health Check | `/health` → check DB, Kafka, Redis, AI service |
| Error Handling | Standardized error response: `{error: {code, message, details}}` |
| Input Validation | Zod schemas on frontend, Go struct tags + custom validators on backend |

---

## 🚀 MODUL 5: Production Readiness

### 5.1 Logging & Monitoring

| Component | Implementation |
|-----------|----------------|
| Structured Logging | zerolog + request_id correlation |
| Metrics | Prometheus `/metrics` (http_requests_total, db_latency, kafka_lag) |
| Error Tracking | Sentry DSN via env var |
| Alerting | PrometheusRule for: error_rate > 5%, p99_latency > 2s, consumer_lag > 1000 |

### 5.2 Config Validation

```go
// internal/config/config.go
type Config struct {
  DBHost     string `env:"DB_HOST,required"`
  DBPort     string `env:"DB_PORT,required"`
  DBUser     string `env:"DB_USER,required"`
  DBPassword string `env:"DB_PASSWORD,required"`
  DBName     string `env:"DB_NAME,required"`
  DBSSLMode  string `env:"DB_SSLMODE,required"`
  JWTSecret  string `env:"JWT_SECRET,required"`
  KafkaBrokers string `env:"KAFKA_BROKERS,required"`
  RedisURL   string `env:"REDIS_URL,required"`
  AIServingURL string `env:"AI_SERVING_URL,required"`
  FrontendOrigin string `env:"FRONTEND_ORIGIN,required"`
  ServerPort string `env:"SERVER_PORT" envDefault:"8081"`
}

func Load() (*Config, error) { /* use cleanenv or similar */ }
```

### 5.3 CI/CD Pipeline (GitHub Actions)

```yaml
# .github/workflows/ci.yml
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres: ...
      kafka: ...
      redis: ...
    steps:
      - uses: actions/checkout@v4
      - name: Run Go tests
        run: make test-go
      - name: Run Frontend tests
        run: make test-frontend
      - name: Run E2E tests
        run: make test-e2e
  build:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build Docker images
        run: docker compose -f docker-compose.prod.yml build
      - name: Push to registry
        run: docker push ...
  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment: production
    steps:
      - name: Deploy to K8s
        run: kubectl apply -f k8s/
```

---

## 🔐 Security Review (z-audit per modul)

| Modul | Checklist |
|-------|-----------|
| Auth | JWT in HttpOnly cookie, CSRF protection (SameSite=Strict), password hashing (bcrypt), rate limit login |
| Approval | Tenant isolation (RLS), role-based access (approver/admin), audit log on approve/reject |
| AI Integration | Input validation (invoice_id UUID, amount regex), rate limit inference, no PII in logs |
| Accounting | Balance validation, tenant isolation, audit trail on JE create/post |
| Tests | No secrets in test fixtures, test DB isolated |

---

## 📦 File Structure Summary (New/Modified)

```
backend/go-core/
├── cmd/api/main.go                    # +auth +approval +inference routes
├── internal/handler/
│   ├── auth_handler.go                # NEW
│   ├── approval_handler.go            # NEW
│   └── inference_handler.go           # NEW
├── internal/usecase/
│   ├── auth/auth_usecase.go           # NEW
│   ├── approval/approval_usecase.go   # NEW
│   └── ai/ai_usecase.go               # NEW
├── internal/infra/
│   ├── ai/ai_client.go                # NEW
│   └── postgres/user_repo.go          # +GetByEmail
├── internal/middleware/
│   └── auth_middleware.go             # +cookie support
├── internal/config/config.go          # NEW - validation
└── internal/domain/errors.go          # +new error codes

frontend/
├── app/
│   ├── login/page.tsx                 # NEW
│   ├── approvals/page.tsx             # NEW
│   ├── accounting/
│   │   ├── chart-of-accounts/page.tsx # NEW
│   │   └── journal-entries/page.tsx   # NEW
│   └── layout.tsx                     # +AuthProvider
├── components/
│   ├── ui/icon.tsx                    # NEW - Lucide wrapper
│   ├── auth/LoginForm.tsx             # NEW
│   ├── approvals/ApprovalDrawer.tsx   # NEW
│   ├── accounting/
│   │   ├── CoATable.tsx               # NEW
│   │   ├── CoADrawer.tsx              # NEW
│   │   ├── JETable.tsx                # NEW
│   │   └── JEDrawer.tsx               # NEW
│   └── ui/Drawer.tsx                  # NEW - reusable
├── hooks/
│   ├── useAuth.ts                     # NEW - Zustand store
│   └── useApprovals.ts                # NEW
└── lib/api.ts                         # +auth +approval +inference endpoints

ai_serving_service.py                  # VERIFY/EXTEND endpoints
docker-compose.yml                     # +redis service
.github/workflows/ci.yml               # NEW
```

---

## ✅ Approval Gates & Handoff

| Gate | Criteria | Agent |
|------|----------|-------|
| Modul 1 Done | Login works, cookie set, approvals CRUD works, all tests pass | `z-audit` → `executing-plans` |
| Modul 2 Done | Go → Python inference roundtrip works, anomaly score returned | `systematic-debugging` → `z-audit` |
| Modul 3 Done | CoA & JE pages render, drawers work, balance validation passes | `frontend-design` → `executing-plans` |
| Modul 4 Done | All E2E scenarios green, rate limit/CORS/health verified | `systematic-debugging` |
| Modul 5 Done | CI/CD passes, metrics exposed, config validated, deployed to staging | `dev-pipeline` → `z-audit` |

---

## 📌 Catatan Implementasi

1. **Urutan ketat:** Modul 1 → 2 → 3 → 4 → 5. Tidak melompat.
2. **Setiap modul:** `executing-plans` implement → `z-audit` review → lanjut modul berikutnya.
3. **Frontend Design:** `frontend-design` agent dipanggil di awal Modul 1 & 3 untuk produce mockup/component spec sebelum coding.
4. **Debugging:** `systematic-debugging` dipanggil saat ada test flaky atau integration error.
5. **CI/CD:** `dev-pipeline` dipanggil di Modul 5 setup pipeline, tapi config CI ditulis di Modul 1 agar test jalan sejak awal.

---

**Document Version:** 1.0  
**Next Action:** Invoke `writing-plans` skill untuk generate implementation plan per modul.