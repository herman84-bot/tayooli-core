# LuminaFlow ERP — Sisa 25% Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement remaining 25% of LuminaFlow ERP: Auth (Zustand + Secure Cookie), Approval routes, AI Serving integration, Accounting UI (Drawer), E2E tests, hardening, and production readiness.

**Architecture:** Go backend (Chi, Hexagonal) + Next.js 15 frontend (App Router, TanStack Query, Zustand) + Python AI service (FastAPI, Celery). Modular, tenant-isolated via RLS. Sequential module execution: 1→2→3→4→5.

**Tech Stack:** Go 1.25, Chi v5, shopspring/decimal, golang-jwt/v5, segmentio/kafka-go, lib/pq, zerolog. Next.js 15, React 19, TanStack Query v5, Zustand v5, Zod v4, Lucide React, Tailwind CSS 4. Python 3.11, FastAPI, Celery, Redis, pytesseract.

---

## Global Constraints

- All Go code: explicit error handling, context for timeouts, no GORM (database/sql only)
- All TS code: strict mode, no `any`, Zod for validation
- Multi-tenant: ALWAYS filter by `tenant_id` from middleware, never hardcode
- UI: ZenSpace (calm colors, progressive disclosure, empathetic errors), Lucide icons (Claude web style: 1.5px stroke, rounded caps, currentColor)
- Forms: Drawer/Slide-over pattern (not modals, not full pages)
- Security: JWT in HttpOnly Secure cookie (`lumina_auth`), SameSite=Strict, rate limit login/inference
- Tests: TDD — write failing test first, then implement
- Commits: frequent, conventional commit messages (`feat:`, `fix:`, `test:`, `refactor:`)
- Branch: work on feature branches, PR to main after each module passes z-audit

---

## Module 1: Wire Approval Routes + Auth System

### Files to Create/Modify

| Path | Action | Purpose |
|------|--------|---------|
| `backend/go-core/internal/handler/auth_handler.go` | Create | Login, Me, Logout HTTP handlers |
| `backend/go-core/internal/usecase/auth/auth_usecase.go` | Create | Auth business logic (verify creds, issue JWT) |
| `backend/go-core/internal/infra/postgres/user_repo.go` | Modify | Add `GetByEmail`, `VerifyPassword` |
| `backend/go-core/internal/middleware/auth_middleware.go` | Modify | Read JWT from cookie `lumina_auth` |
| `backend/go-core/cmd/api/main.go` | Modify | Register `/auth/*` and `/approvals/*` routes |
| `backend/go-core/internal/handler/approval_handler.go` | Create | Approval CRUD + approve/reject |
| `backend/go-core/internal/usecase/approval/approval_usecase.go` | Create | Approval workflow logic |
| `frontend/app/login/page.tsx` | Create | Login page with form |
| `frontend/hooks/useAuth.ts` | Create | Zustand auth store + hook |
| `frontend/components/auth/LoginForm.tsx` | Create | Reusable login form component |
| `frontend/app/approvals/page.tsx` | Create | Approvals list page |
| `frontend/components/approvals/ApprovalDrawer.tsx` | Create | Drawer for approve/reject |
| `frontend/components/ui/icon.tsx` | Create | Lucide wrapper (Claude style) |
| `frontend/components/ui/drawer.tsx` | Create | Reusable drawer component |
| `frontend/lib/api.ts` | Modify | Add auth, approval, inference endpoints |

---

### Task 1.1: Backend — User Repository Extensions

**Files:**
- Modify: `backend/go-core/internal/infra/postgres/user_repo.go`
- Test: `backend/go-core/internal/infra/postgres/user_repo_test.go`

**Interfaces:**
- Produces: `GetByEmail(ctx, tenantID, email) (*domain.User, error)`, `VerifyPassword(hash, password) bool`

```go
// backend/go-core/internal/infra/postgres/user_repo.go (add to existing UserRepo)

func (r *UserRepo) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.User, error) {
    const q = `SELECT id, tenant_id, email, password_hash, role, created_at, updated_at 
               FROM users WHERE tenant_id = $1 AND email = $2`
    var u domain.User
    err := r.db.QueryRowContext(ctx, q, tenantID, email).Scan(
        &u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
    if err == sql.ErrNoRows {
        return nil, domain.ErrNotFound
    }
    return &u, err
}

func VerifyPassword(hash, password string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

- [ ] **Step 1: Write failing test**

```go
// backend/go-core/internal/infra/postgres/user_repo_test.go
func TestUserRepo_GetByEmail(t *testing.T) {
    db := setupTestDB(t)
    repo := postgres.NewUserRepo(db)
    tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
    
    // Seed user
    _, err := db.ExecContext(context.Background(), `
        INSERT INTO users (id, tenant_id, email, password_hash, role)
        VALUES ($1, $2, $3, $4, $5)`,
        uuid.New(), tenantID, "test@example.com", 
        "$2a$10$hashedpassword", "admin")
    require.NoError(t, err)
    
    // Test GetByEmail
    u, err := repo.GetByEmail(context.Background(), tenantID, "test@example.com")
    require.NoError(t, err)
    assert.Equal(t, "test@example.com", u.Email)
    
    // Test not found
    _, err = repo.GetByEmail(context.Background(), tenantID, "none@example.com")
    assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestVerifyPassword(t *testing.T) {
    hash, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
    assert.True(t, postgres.VerifyPassword(string(hash), "secret123"))
    assert.False(t, postgres.VerifyPassword(string(hash), "wrong"))
}
```

- [ ] **Step 2: Run test to verify it fails**
```bash
cd backend/go-core && go test -v ./internal/infra/postgres/... -run TestUserRepo_GetByEmail
# Expected: FAIL - GetByEmail not defined
```

- [ ] **Step 3: Write minimal implementation** (code above)
- [ ] **Step 4: Run test to verify it passes**
```bash
cd backend/go-core && go test -v ./internal/infra/postgres/... -run TestUserRepo
# Expected: PASS
```

- [ ] **Step 5: Commit**
```bash
git add backend/go-core/internal/infra/postgres/user_repo.go backend/go-core/internal/infra/postgres/user_repo_test.go
git commit -m "feat(auth): add GetByEmail and VerifyPassword to UserRepo"
```

---

### Task 1.2: Backend — Auth Usecase

**Files:**
- Create: `backend/go-core/internal/usecase/auth/auth_usecase.go`
- Test: `backend/go-core/internal/usecase/auth/auth_usecase_test.go`

**Interfaces:**
- Consumes: `UserRepo.GetByEmail`, `VerifyPassword`
- Produces: `Login(ctx, tenantID, email, password) (token string, user *domain.User, error)`, `GetMe(ctx, userID, tenantID) (*domain.User, error)`

```go
// backend/go-core/internal/usecase/auth/auth_usecase.go
package auth

import (
    "context"
    "time"
    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
    "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
    "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
)

type Usecase struct {
    userRepo *postgres.UserRepo
    jwtSecret string
    tokenTTL time.Duration
}

func New(userRepo *postgres.UserRepo, jwtSecret string, tokenTTL time.Duration) *Usecase {
    return &Usecase{userRepo: userRepo, jwtSecret: jwtSecret, tokenTTL: tokenTTL}
}

func (u *Usecase) Login(ctx context.Context, tenantID uuid.UUID, email, password string) (string, *domain.User, error) {
    user, err := u.userRepo.GetByEmail(ctx, tenantID, email)
    if err != nil {
        return "", nil, domain.ErrInvalidCredentials
    }
    if !postgres.VerifyPassword(user.PasswordHash, password) {
        return "", nil, domain.ErrInvalidCredentials
    }
    token, err := u.generateToken(user.ID, user.TenantID, user.Role)
    return token, user, err
}

func (u *Usecase) GetMe(ctx context.Context, userID, tenantID uuid.UUID) (*domain.User, error) {
    return u.userRepo.GetByID(ctx, userID, tenantID)
}

func (u *Usecase) generateToken(userID, tenantID uuid.UUID, role string) (string, error) {
    claims := jwt.MapClaims{
        "sub":       userID.String(),
        "tenant_id": tenantID.String(),
        "role":      role,
        "exp":       time.Now().Add(u.tokenTTL).Unix(),
        "iat":       time.Now().Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(u.jwtSecret))
}
```

- [ ] **Step 1: Write failing test**

```go
// backend/go-core/internal/usecase/auth/auth_usecase_test.go
package auth

import (
    "context"
    "testing"
    "time"
    "github.com/google/uuid"
    "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
    "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
    "github.com/stretchr/testify/require"
)

func TestAuthUsecase_Login(t *testing.T) {
    db := setupTestDB(t)
    userRepo := postgres.NewUserRepo(db)
    uc := New(userRepo, "test-secret", time.Hour)
    tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
    
    // Seed user with bcrypt hash of "password123"
    hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
    userID := uuid.New()
    _, err := db.ExecContext(context.Background(), `
        INSERT INTO users (id, tenant_id, email, password_hash, role)
        VALUES ($1, $2, $3, $4, $5)`,
        userID, tenantID, "admin@test.com", string(hash), "admin")
    require.NoError(t, err)
    
    // Success
    token, user, err := uc.Login(context.Background(), tenantID, "admin@test.com", "password123")
    require.NoError(t, err)
    assert.NotEmpty(t, token)
    assert.Equal(t, "admin@test.com", user.Email)
    
    // Wrong password
    _, _, err = uc.Login(context.Background(), tenantID, "admin@test.com", "wrong")
    assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
    
    // Non-existent user
    _, _, err = uc.Login(context.Background(), tenantID, "none@test.com", "password123")
    assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthUsecase_GetMe(t *testing.T) {
    db := setupTestDB(t)
    userRepo := postgres.NewUserRepo(db)
    uc := New(userRepo, "test-secret", time.Hour)
    tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
    userID := uuid.New()
    // ... seed user ...
    user, err := uc.GetMe(context.Background(), userID, tenantID)
    require.NoError(t, err)
    assert.Equal(t, userID, user.ID)
}
```

- [ ] **Step 2: Run test to verify it fails**
```bash
cd backend/go-core && go test -v ./internal/usecase/auth/... -run TestAuthUsecase_Login
# Expected: FAIL - package doesn't exist
```

- [ ] **Step 3: Write minimal implementation** (code above)
- [ ] **Step 4: Run test to verify it passes**
```bash
cd backend/go-core && go test -v ./internal/usecase/auth/...
# Expected: PASS
```

- [ ] **Step 5: Commit**
```bash
git add backend/go-core/internal/usecase/auth/
git commit -m "feat(auth): add AuthUsecase with Login and GetMe"
```

---

### Task 1.3: Backend — Auth Handler

**Files:**
- Create: `backend/go-core/internal/handler/auth_handler.go`
- Test: `backend/go-core/internal/handler/auth_handler_test.go`

**Interfaces:**
- Consumes: `AuthUsecase.Login`, `AuthUsecase.GetMe`
- Produces: HTTP handlers `Login`, `Me`, `Logout`

```go
// backend/go-core/internal/handler/auth_handler.go
package handler

import (
    "context"
    "net/http"
    "time"
    "github.com/google/uuid"
    "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/auth"
)

type AuthHandler struct {
    authUC *auth.Usecase
}

func NewAuthHandler(authUC *auth.Usecase) *AuthHandler {
    return &AuthHandler{authUC: authUC}
}

type loginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    tenantID := uuid.MustParse(ctx.Value("tenant_id").(string))
    
    var req loginRequest
    if err := decodeJSON(r, &req); err != nil {
        respondError(w, r, http.StatusBadRequest, "invalid request body")
        return
    }
    if err := validate.Struct(req); err != nil {
        respondError(w, r, http.StatusBadRequest, "validation failed")
        return
    }
    
    token, user, err := h.authUC.Login(ctx, tenantID, req.Email, req.Password)
    if err != nil {
        respondError(w, r, http.StatusUnauthorized, "invalid credentials")
        return
    }
    
    // Set HTTP-only cookie
    http.SetCookie(w, &http.Cookie{
        Name:     "lumina_auth",
        Value:    token,
        Path:     "/",
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
        MaxAge:   int(time.Hour.Seconds()),
    })
    
    respondJSON(w, r, http.StatusOK, map[string]any{
        "user": map[string]string{
            "id":    user.ID.String(),
            "email": user.Email,
            "role":  user.Role,
        },
    })
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := uuid.MustParse(ctx.Value("user_id").(string))
    tenantID := uuid.MustParse(ctx.Value("tenant_id").(string))
    
    user, err := h.authUC.GetMe(ctx, userID, tenantID)
    if err != nil {
        respondError(w, r, http.StatusNotFound, "user not found")
        return
    }
    respondJSON(w, r, http.StatusOK, map[string]any{
        "user": map[string]string{
            "id":    user.ID.String(),
            "email": user.Email,
            "role":  user.Role,
        },
    })
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
    http.SetCookie(w, &http.Cookie{
        Name:     "lumina_auth",
        Value:    "",
        Path:     "/",
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
        MaxAge:   0,
    })
    respondJSON(w, r, http.StatusOK, map[string]string{"message": "logged out"})
}
```

- [ ] **Step 1: Write failing test**

```go
// backend/go-core/internal/handler/auth_handler_test.go
package handler

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/google/uuid"
    "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/auth"
    "github.com/stretchr/testify/require"
)

func TestAuthHandler_Login(t *testing.T) {
    // Mock AuthUsecase
    mockUC := &mockAuthUsecase{
        loginFn: func(ctx context.Context, tenantID uuid.UUID, email, password string) (string, *domain.User, error) {
            if email == "admin@test.com" && password == "password123" {
                return "mock-jwt-token", &domain.User{ID: uuid.New(), Email: email, Role: "admin"}, nil
            }
            return "", nil, domain.ErrInvalidCredentials
        },
    }
    h := NewAuthHandler(mockUC)
    
    // Test success
    body := `{"email":"admin@test.com","password":"password123"}`
    req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
    req = req.WithContext(context.WithValue(req.Context(), "tenant_id", "550e8400-e29b-41d4-a716-446655440000"))
    rec := httptest.NewRecorder()
    
    h.Login(rec, req)
    
    assert.Equal(t, http.StatusOK, rec.Code)
    cookie := rec.Result().Cookies()[0]
    assert.Equal(t, "lumina_auth", cookie.Name)
    assert.Equal(t, "mock-jwt-token", cookie.Value)
    assert.True(t, cookie.HttpOnly)
    assert.True(t, cookie.Secure)
    
    // Test invalid credentials
    body = `{"email":"admin@test.com","password":"wrong"}`
    req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
    req = req.WithContext(context.WithValue(req.Context(), "tenant_id", "550e8400-e29b-41d4-a716-446655440000"))
    rec = httptest.NewRecorder()
    h.Login(rec, req)
    assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
```

- [ ] **Step 2: Run test to verify it fails**
```bash
cd backend/go-core && go test -v ./internal/handler/... -run TestAuthHandler_Login
# Expected: FAIL
```

- [ ] **Step 3: Write minimal implementation** (code above)
- [ ] **Step 4: Run test to verify it passes**
```bash
cd backend/go-core && go test -v ./internal/handler/... -run TestAuthHandler
# Expected: PASS
```

- [ ] **Step 5: Commit**
```bash
git add backend/go-core/internal/handler/auth_handler.go backend/go-core/internal/handler/auth_handler_test.go
git commit -m "feat(auth): add AuthHandler with Login, Me, Logout"
```

---

### Task 1.4: Backend — Update Auth Middleware for Cookie

**Files:**
- Modify: `backend/go-core/internal/middleware/auth_middleware.go`
- Test: `backend/go-core/internal/middleware/tenant_test.go` (extend)

**Interfaces:**
- Produces: Middleware that reads JWT from `lumina_auth` cookie OR `Authorization` header (backward compat)

```go
// backend/go-core/internal/middleware/auth_middleware.go (modify extractToken func)

func extractToken(r *http.Request) (string, error) {
    // Try cookie first
    if cookie, err := r.Cookie("lumina_auth"); err == nil && cookie.Value != "" {
        return cookie.Value, nil
    }
    // Fallback to Authorization header
    auth := r.Header.Get("Authorization")
    if auth == "" {
        return "", domain.ErrUnauthorized
    }
    const prefix = "Bearer "
    if len(auth) < len(prefix) || auth[:len(prefix)] != prefix {
        return "", domain.ErrInvalidToken
    }
    return auth[len(prefix):], nil
}
```

- [ ] **Step 1: Write failing test** (extend existing `tenant_test.go`)
```go
func TestAuthMiddleware_CookieToken(t *testing.T) {
    req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil)
    req.AddCookie(&http.Cookie{Name: "lumina_auth", Value: "valid.jwt.token"})
    req = req.WithContext(context.WithValue(req.Context(), "jwt_secret", "test-secret"))
    
    rec := httptest.NewRecorder()
    TenantMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "valid.jwt.token", r.Header.Get("X-JWT-Token"))
    })).ServeHTTP(rec, req)
    
    assert.Equal(t, http.StatusOK, rec.Code)
}
```

- [ ] **Step 2: Run test to verify it fails**
```bash
cd backend/go-core && go test -v ./internal/middleware/... -run TestAuthMiddleware_CookieToken
# Expected: FAIL
```

- [ ] **Step 3: Write minimal implementation** (code above)
- [ ] **Step 4: Run test to verify it passes**
```bash
cd backend/go-core && go test -v ./internal/middleware/...
# Expected: PASS
```

- [ ] **Step 5: Commit**
```bash
git add backend/go-core/internal/middleware/auth_middleware.go backend/go-core/internal/middleware/tenant_test.go
git commit -m "feat(auth): middleware reads JWT from HttpOnly cookie"
```

---

### Task 1.5: Backend — Register Auth Routes in Main

**Files:**
- Modify: `backend/go-core/cmd/api/main.go`

**Interfaces:**
- Consumes: `AuthHandler`, `ApprovalHandler` (later)
- Produces: Registered routes

```go
// backend/go-core/cmd/api/main.go (add to main())

// ... after existing imports
import authUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/auth"
import approvalUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/approval"

// ... inside main() after existing wiring

// Auth
authUsecase := authUC.New(userRepo, jwtSecret, time.Hour)
authHandler := handler.NewAuthHandler(authUsecase)

// Approval (placeholder for now - will implement in Task 1.7)
approvalRepo := postgres.NewApprovalRepo(db)
approvalUsecase := approvalUC.New(approvalRepo)
approvalHandler := handler.NewApprovalHandler(approvalUsecase)

// ... in router setup, add PUBLIC auth routes BEFORE tenantMiddleware group
r.Route("/api/v1/auth", func(r chi.Router) {
    r.Post("/login", authHandler.Login)
})

// Protected auth routes (require valid cookie)
r.Group(func(r chi.Router) {
    r.Use(tenantMiddleware.TenantMiddleware)
    r.Get("/auth/me", authHandler.Me)
    r.Post("/auth/logout", authHandler.Logout)
})

// Approval routes
r.Group(func(r chi.Router) {
    r.Use(tenantMiddleware.TenantMiddleware)
    r.Get("/approvals", approvalHandler.List)
    r.Get("/approvals/{id}", approvalHandler.Get)
    r.Post("/approvals/{id}/approve", approvalHandler.Approve)
    r.Post("/approvals/{id}/reject", approvalHandler.Reject)
})
```

- [ ] **Step 1: Verify compiles**
```bash
cd backend/go-core && go build ./cmd/api/...
# Expected: SUCCESS (after ApprovalHandler exists in Task 1.7)
```

- [ ] **Step 2: Commit**
```bash
git add backend/go-core/cmd/api/main.go
git commit -m "feat(auth): register /auth/* routes; add approval routes placeholder"
```

---

### Task 1.6: Frontend — Icon Wrapper (Claude Style)

**Files:**
- Create: `frontend/components/ui/icon.tsx`

**Interfaces:**
- Produces: `<Icon name="LayoutDashboard" size="md" />` component

```tsx
// frontend/components/ui/icon.tsx
"use client"

import * as Lucide from "lucide-react"

type IconName = keyof typeof Lucide

interface IconProps {
  name: IconName
  size?: "sm" | "md" | "lg"
  className?: string
}

const sizeMap = {
  sm: "w-4 h-4",
  md: "w-5 h-5",
  lg: "w-6 h-6",
} as const

export function Icon({ name, size = "md", className }: IconProps) {
  const IconComponent = Lucide[name]
  if (!IconComponent) {
    console.warn(`Icon "${name}" not found in Lucide`)
    return null
  }
  return <IconComponent className={`${sizeMap[size]} text-current ${className ?? ""}`} strokeWidth={1.5} />
}
```

- [ ] **Step 1: Create file**
- [ ] **Step 2: Test import works**
```bash
cd frontend && npm run build 2>&1 | head -20
# Expected: no icon.tsx errors
```

- [ ] **Step 3: Commit**
```bash
git add frontend/components/ui/icon.tsx
git commit -m "feat(ui): add Icon wrapper with Claude-style Lucide icons"
```

---

### Task 1.7: Frontend — Drawer Component

**Files:**
- Create: `frontend/components/ui/drawer.tsx`

**Interfaces:**
- Produces: `<Drawer open={true} onOpenChange={setOpen}><DrawerContent>...</DrawerContent></Drawer>`

```tsx
// frontend/components/ui/drawer.tsx
"use client"

import { X } from "lucide-react"
import { createContext, useContext, useEffect, useState } from "react"
import { cn } from "@/lib/utils"

interface DrawerContextType {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const DrawerContext = createContext<DrawerContextType | null>(null)

export function DrawerProvider({ children, open, onOpenChange }: { 
  children: React.ReactNode
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  return (
    <DrawerContext.Provider value={{ open, onOpenChange }}>
      {children}
    </DrawerContext.Provider>
  )
}

function useDrawer() {
  const ctx = useContext(DrawerContext)
  if (!ctx) throw new Error("Drawer components must be used within DrawerProvider")
  return ctx
}

export function Drawer({ children }: { children: React.ReactNode }) {
  const { open, onOpenChange } = useDrawer()
  if (!open) return null
  
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => { if (e.key === "Escape") onOpenChange(false) }
    document.addEventListener("keydown", handleEscape)
    document.body.style.overflow = "hidden"
    return () => { document.removeEventListener("keydown", handleEscape); document.body.style.overflow = "" }
  }, [onOpenChange])
  
  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="fixed inset-0 bg-black/30" onClick={() => onOpenChange(false)} />
      <div className="ml-auto w-full max-w-2xl bg-background shadow-xl flex flex-col">
        {children}
      </div>
    </div>
  )
}

export function DrawerHeader({ title, icon }: { title: string; icon?: React.ReactNode }) {
  const { onOpenChange } = useDrawer()
  return (
    <div className="flex items-center justify-between p-4 border-b">
      <div className="flex items-center gap-3">
        {icon && <span className="text-muted-foreground">{icon}</span>}
        <h2 className="text-lg font-semibold">{title}</h2>
      </div>
      <button onClick={() => onOpenChange(false)} className="p-1 rounded-md hover:bg-accent">
        <X className="w-5 h-5" />
      </button>
    </div>
  )
}

export function DrawerContent({ children }: { children: React.ReactNode }) {
  return <div className="flex-1 overflow-y-auto p-4">{children}</div>
}

export function DrawerFooter({ children }: { children: React.ReactNode }) {
  return <div className="p-4 border-t flex justify-end gap-2">{children}</div>
}
```

- [ ] **Step 1: Create file**
- [ ] **Step 2: Commit**
```bash
git add frontend/components/ui/drawer.tsx
git commit -m "feat(ui): add Drawer component for slide-over pattern"
```

---

### Task 1.8: Frontend — Auth Store (Zustand)

**Files:**
- Create: `frontend/hooks/useAuth.ts`

**Interfaces:**
- Produces: `useAuthStore` with `user`, `isAuthenticated`, `isLoading`, `login`, `logout`, `hydrate`

```tsx
// frontend/hooks/useAuth.ts
"use client"

import { create } from "zustand"
import { persist, createJSONStorage } from "zustand/middleware"

interface User {
  id: string
  email: string
  role: string
  tenantId: string
}

interface AuthState {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  hydrate: () => Promise<void>
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      isAuthenticated: false,
      isLoading: true,
      
      login: async (email: string, password: string) => {
        set({ isLoading: true })
        const res = await fetch("/api/v1/auth/login", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include", // important for cookies
          body: JSON.stringify({ email, password }),
        })
        if (!res.ok) {
          set({ isLoading: false })
          const err = await res.json()
          throw new Error(err.error || "Login failed")
        }
        const data = await res.json()
        set({ 
          user: data.user, 
          isAuthenticated: true, 
          isLoading: false 
        })
      },
      
      logout: async () => {
        await fetch("/api/v1/auth/logout", { 
          method: "POST", 
          credentials: "include" 
        })
        set({ user: null, isAuthenticated: false })
      },
      
      hydrate: async () => {
        set({ isLoading: true })
        try {
          const res = await fetch("/api/v1/auth/me", { credentials: "include" })
          if (res.ok) {
            const data = await res.json()
            set({ user: data.user, isAuthenticated: true, isLoading: false })
          } else {
            set({ user: null, isAuthenticated: false, isLoading: false })
          }
        } catch {
          set({ user: null, isAuthenticated: false, isLoading: false })
        }
      },
    }),
    {
      name: "lumina-auth",
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({ user: state.user, isAuthenticated: state.isAuthenticated }),
    }
  )
)

export function useAuth() {
  const { user, isAuthenticated, isLoading, login, logout, hydrate } = useAuthStore()
  return { user, isAuthenticated, isLoading, login, logout, hydrate }
}
```

- [ ] **Step 1: Create file**
- [ ] **Step 2: Commit**
```bash
git add frontend/hooks/useAuth.ts
git commit -m "feat(auth): add Zustand auth store with cookie-based auth"
```

---

### Task 1.9: Frontend — Login Page

**Files:**
- Create: `frontend/app/login/page.tsx`
- Create: `frontend/components/auth/LoginForm.tsx`

**Interfaces:**
- Consumes: `useAuth`, `Icon`, `Drawer` (not used here but pattern)
- Produces: Login page with form

```tsx
// frontend/components/auth/LoginForm.tsx
"use client"

import { useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Mail, Lock, AlertCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Icon } from "@/components/ui/icon"
import { useAuth } from "@/hooks/useAuth"

const loginSchema = z.object({
  email: z.string().email("Invalid email address"),
  password: z.string().min(8, "Password must be at least 8 characters"),
})
type LoginFormData = z.infer<typeof loginSchema>

export function LoginForm() {
  const { login } = useAuth()
  const [error, setError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  
  const { register, handleSubmit, formState: { errors } } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
  })
  
  const onSubmit = async (data: LoginFormData) => {
    setError(null)
    setIsLoading(true)
    try {
      await login(data.email, data.password)
    } catch (e) {
      setError(e instanceof Error ? e.message : "Login failed")
    } finally {
      setIsLoading(false)
    }
  }
  
  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      {error && (
        <div className="flex items-center gap-2 text-sm text-destructive bg-destructive/10 p-3 rounded-md">
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          <span>{error}</span>
        </div>
      )}
      
      <div className="space-y-1.5">
        <label htmlFor="email" className="text-sm font-medium">Email</label>
        <div className="relative">
          <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-muted-foreground" />
          <Input
            id="email"
            type="email"
            placeholder="admin@company.com"
            className="pl-10"
            {...register("email")}
          />
        </div>
        {errors.email && (
          <p className="text-sm text-destructive">{errors.email.message}</p>
        )}
      </div>
      
      <div className="space-y-1.5">
        <label htmlFor="password" className="text-sm font-medium">Password</label>
        <div className="relative">
          <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-muted-foreground" />
          <Input
            id="password"
            type="password"
            placeholder="••••••••"
            className="pl-10"
            {...register("password")}
          />
        </div>
        {errors.password && (
          <p className="text-sm text-destructive">{errors.password.message}</p>
        )}
      </div>
      
      <Button type="submit" className="w-full" disabled={isLoading}>
        {isLoading ? "Signing in..." : "Sign in"}
      </Button>
    </form>
  )
}
```

```tsx
// frontend/app/login/page.tsx
import { Metadata } from "next"
import { Building2 } from "lucide-react"
import { LoginForm } from "@/components/auth/LoginForm"
import { Icon } from "@/components/ui/icon"

export const metadata: Metadata = {
  title: "Sign in | LuminaFlow ERP",
}

export default function LoginPage() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background px-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-primary text-primary-foreground mb-4">
            <Building2 className="w-8 h-8" />
          </div>
          <h1 className="text-2xl font-bold">Welcome back</h1>
          <p className="text-muted-foreground mt-1">Sign in to your LuminaFlow account</p>
        </div>
        
        <div className="bg-card border rounded-xl p-6 shadow-sm">
          <LoginForm />
        </div>
        
        <p className="text-center text-sm text-muted-foreground mt-6">
          Demo: admin@test.com / password123
        </p>
      </div>
    </div>
  )
}
```

- [ ] **Step 1: Create files**
- [ ] **Step 2: Test build**
```bash
cd frontend && npm run build 2>&1 | grep -E "(login|error)" | head -10
# Expected: no errors
```

- [ ] **Step 3: Commit**
```bash
git add frontend/app/login/page.tsx frontend/components/auth/LoginForm.tsx
git commit -m "feat(auth): add login page with form"
```

---

### Task 1.10: Frontend — Auth Provider & Route Protection

**Files:**
- Modify: `frontend/app/providers.tsx`
- Modify: `frontend/app/(shell)/layout.tsx` (or root layout)

**Interfaces:**
- Consumes: `useAuth` hook
- Produces: Client provider that hydrates auth on mount

```tsx
// frontend/app/providers.tsx (modify)
"use client"

import { SessionProvider } from "next-auth/react" // if using, else remove
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ReactNode, useEffect, useState } from "react"
import { useAuth } from "@/hooks/useAuth"

export function Providers({ children }: { children: ReactNode }) {
  const [queryClient] = useState(() => new QueryClient({
    defaultOptions: { queries: { staleTime: 5 * 60 * 1000 } }
  }))
  const { hydrate, isLoading } = useAuth()
  
  useEffect(() => { hydrate() }, [hydrate])
  
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }
  
  return (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  )
}
```

```tsx
// frontend/app/(shell)/layout.tsx (add auth check)
"use client"

import { useEffect } from "react"
import { useRouter, usePathname } from "next/navigation"
import { useAuth } from "@/hooks/useAuth"

const publicPaths = ["/login"]

export default function ShellLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const pathname = usePathname()
  const { isAuthenticated, isLoading } = useAuth()
  
  useEffect(() => {
    if (!isLoading && !isAuthenticated && !publicPaths.includes(pathname)) {
      router.push("/login")
    }
  }, [isAuthenticated, isLoading, pathname, router])
  
  if (isLoading) return null
  if (!isAuthenticated && !publicPaths.includes(pathname)) return null
  
  return <>{children}</>
}
```

- [ ] **Step 1: Modify files**
- [ ] **Step 2: Commit**
```bash
git add frontend/app/providers.tsx frontend/app/\(shell\)/layout.tsx
git commit -m "feat(auth): add auth provider and route protection"
```

---

### Task 1.11: Backend — Approval Handler + Usecase

**Files:**
- Create: `backend/go-core/internal/handler/approval_handler.go`
- Create: `backend/go-core/internal/usecase/approval/approval_usecase.go`
- Test: `backend/go-core/internal/handler/approval_handler_test.go`

**Interfaces:**
- Consumes: `ApprovalRepo` (CRUD), `AuditLogRepo`
- Produces: Handlers for List, Get, Approve, Reject

```go
// backend/go-core/internal/usecase/approval/approval_usecase.go
package approval

import (
    "context"
    "github.com/google/uuid"
    "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
    "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
)

type Usecase struct {
    approvalRepo *postgres.ApprovalRepo
    auditLogRepo *postgres.AuditLogRepo
}

func New(approvalRepo *postgres.ApprovalRepo, auditLogRepo *postgres.AuditLogRepo) *Usecase {
    return &Usecase{approvalRepo: approvalRepo, auditLogRepo: auditLogRepo}
}

func (u *Usecase) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Approval, error) {
    return u.approvalRepo.List(ctx, tenantID)
}

func (u *Usecase) Get(ctx context.Context, id, tenantID uuid.UUID) (*domain.Approval, error) {
    return u.approvalRepo.GetByID(ctx, id, tenantID)
}

func (u *Usecase) Approve(ctx context.Context, id, tenantID, userID uuid.UUID) (*domain.Approval, error) {
    approval, err := u.approvalRepo.GetByID(ctx, id, tenantID)
    if err != nil { return nil, err }
    if approval.Status != "pending" { return nil, domain.ErrInvalidStatus }
    
    approval.Status = "approved"
    approval.ApprovedBy = &userID
    now := time.Now()
    approval.ApprovedAt = &now
    
    if err := u.approvalRepo.Update(ctx, approval); err != nil { return nil, err }
    
    // Audit log
    u.auditLogRepo.Create(ctx, domain.AuditLog{
        TenantID: tenantID, UserID: &userID, EntityType: "approval",
        EntityID: id, Action: "approve", NewData: toJSON(approval),
    })
    return approval, nil
}

func (u *Usecase) Reject(ctx context.Context, id, tenantID, userID uuid.UUID, reason string) (*domain.Approval, error) {
    approval, err := u.approvalRepo.GetByID(ctx, id, tenantID)
    if err != nil { return nil, err }
    if approval.Status != "pending" { return nil, domain.ErrInvalidStatus }
    
    approval.Status = "rejected"
    approval.RejectedBy = &userID
    approval.RejectionReason = reason
    now := time.Now()
    approval.RejectedAt = &now
    
    if err := u.approvalRepo.Update(ctx, approval); err != nil { return nil, err }
    
    u.auditLogRepo.Create(ctx, domain.AuditLog{
        TenantID: tenantID, UserID: &userID, EntityType: "approval",
        EntityID: id, Action: "reject", NewData: toJSON(approval),
    })
    return approval, nil
}
```

- [ ] **Step 1: Write failing test** (approval_usecase_test.go)
- [ ] **Step 2: Run test to verify it fails**
- [ ] **Step 3: Write minimal implementation** (code above)
- [ ] **Step 4: Run test to verify it passes**
- [ ] **Step 5: Create approval_handler.go** (similar pattern to auth_handler)
- [ ] **Step 6: Run handler tests**
- [ ] **Step 7: Commit**
```bash
git add backend/go-core/internal/usecase/approval/ backend/go-core/internal/handler/approval_handler.go backend/go-core/internal/handler/approval_handler_test.go
git commit -m "feat(approval): add approval usecase and handler"
```

---

### Task 1.12: Frontend — Approvals Page + Drawer

**Files:**
- Create: `frontend/app/approvals/page.tsx`
- Create: `frontend/components/approvals/ApprovalDrawer.tsx`

**Interfaces:**
- Consumes: `useApprovals` hook (TanStack Query), `Drawer`, `Icon`, `Button`
- Produces: Approvals list with slide-over approve/reject

```tsx
// frontend/hooks/useApprovals.ts
"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/lib/api"

export function useApprovals() {
  return useQuery({ queryKey: ["approvals"], queryFn: api.approvals.list })
}

export function useApprovalActions() {
  const qc = useQueryClient()
  const approve = useMutation({ 
    mutationFn: (id: string) => api.approvals.approve(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["approvals"] })
  })
  const reject = useMutation({ 
    mutationFn: ({ id, reason }: { id: string; reason: string }) => api.approvals.reject(id, reason),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["approvals"] })
  })
  return { approve, reject }
}
```

```tsx
// frontend/app/approvals/page.tsx
"use client"

import { useState } from "react"
import { CheckCircle, XCircle, FileText, Clock } from "lucide-react"
import { ColumnDef } from "@tanstack/react-table"
import { DataTable } from "@/components/ui/data-table"
import { Drawer, DrawerHeader, DrawerContent, DrawerFooter } from "@/components/ui/drawer"
import { ApprovalDrawer } from "@/components/approvals/ApprovalDrawer"
import { Icon } from "@/components/ui/icon"
import { useApprovals, useApprovalActions } from "@/hooks/useApprovals"

const columns: ColumnDef<Approval>[] = [
  { accessorKey: "id", header: "ID", cell: ({ row }) => row.getValue().slice(0,8) + "..." },
  { accessorKey: "entityType", header: "Type" },
  { accessorKey: "entityId", header: "Entity ID", cell: ({ row }) => row.getValue().slice(0,8) + "..." },
  { accessorKey: "status", header: "Status", cell: ({ row }) => {
    const s = row.getValue()
    return <span className={cn("px-2 py-1 rounded-full text-xs", 
      s==="pending" && "bg-yellow-100 text-yellow-800",
      s==="approved" && "bg-green-100 text-green-800",
      s==="rejected" && "bg-red-100 text-red-800")}>{s}</span>
  }},
  { accessorKey: "createdAt", header: "Created", cell: ({ row }) => 
    new Date(row.getValue()).toLocaleDateString() },
]

export default function ApprovalsPage() {
  const { data: approvals, isLoading } = useApprovals()
  const [selected, setSelected] = useState<Approval | null>(null)
  const { approve, reject } = useApprovalActions()
  
  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <Icon name="FileText" /> Approvals
        </h1>
      </div>
      
      <DataTable columns={columns} data={approvals ?? []} isLoading={isLoading} />
      
      <Drawer open={!!selected} onOpenChange={open => !open && setSelected(null)}>
        <DrawerProvider open={!!selected} onOpenChange={open => !open && setSelected(null)}>
          <DrawerHeader title="Approval Detail" icon={<Icon name="FileText" />} />
          <DrawerContent>
            {selected && <ApprovalDrawer approval={selected} onApprove={approve.mutate} onReject={reject.mutate} />}
          </DrawerContent>
        </DrawerProvider>
      </Drawer>
    </div>
  )
}
```

```tsx
// frontend/components/approvals/ApprovalDrawer.tsx
"use client"

import { useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { CheckCircle, XCircle, AlertCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { DrawerHeader, DrawerContent, DrawerFooter } from "@/components/ui/drawer"
import { Icon } from "@/components/ui/icon"

const rejectSchema = z.object({ reason: z.string().min(5, "Reason required (min 5 chars)") })

interface ApprovalDrawerProps {
  approval: Approval
  onApprove: (id: string) => void
  onReject: (id: string, reason: string) => void
}

export function ApprovalDrawer({ approval, onApprove, onReject }: ApprovalDrawerProps) {
  const [action, setAction] = useState<"approve" | "reject">("approve")
  const { register, handleSubmit, formState: { errors } } = useForm({ resolver: zodResolver(rejectSchema) })
  
  return (
    <div className="space-y-4">
      <dl className="grid grid-cols-3 gap-4 text-sm">
        <dt className="text-muted-foreground">Type</dt><dd className="col-span-2 font-medium">{approval.entityType}</dd>
        <dt className="text-muted-foreground">Entity</dt><dd className="col-span-2 font-mono text-xs">{approval.entityId}</dd>
        <dt className="text-muted-foreground">Requested By</dt><dd className="col-span-2">{approval.requestedBy}</dd>
        <dt className="text-muted-foreground">Created</dt><dd className="col-span-2">{new Date(approval.createdAt).toLocaleString()}</dd>
      </dl>
      
      <div className="flex gap-2">
        <Button variant="outline" onClick={() => setAction("approve")} className={action==="approve" && "bg-green-100 text-green-800"}>
          <Icon name="CheckCircle" className="mr-1" /> Approve
        </Button>
        <Button variant="outline" onClick={() => setAction("reject")} className={action==="reject" && "bg-red-100 text-red-800"}>
          <Icon name="XCircle" className="mr-1" /> Reject
        </Button>
      </div>
      
      {action === "reject" && (
        <form onSubmit={handleSubmit((d) => onReject(approval.id, d.reason))} className="space-y-2">
          <Textarea {...register("reason")} placeholder="Rejection reason..." rows={3} />
          {errors.reason && <p className="text-sm text-destructive">{errors.reason.message}</p>}
          <Button type="submit" variant="destructive">Reject</Button>
        </form>
      )}
      
      {action === "approve" && (
        <div className="flex justify-end">
          <Button onClick={() => onApprove(approval.id)} className="bg-green-600 hover:bg-green-700">
            <Icon name="CheckCircle" className="mr-1" /> Confirm Approve
          </Button>
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 1: Create files**
- [ ] **Step 2: Commit**
```bash
git add frontend/hooks/useApprovals.ts frontend/app/approvals/page.tsx frontend/components/approvals/ApprovalDrawer.tsx
git commit -m "feat(approval): add approvals page with drawer actions"
```

---

### Task 1.13: Module 1 Integration Test & z-audit

**Files:**
- Test: `backend/go-core/integration/auth_approval_test.go`
- Run: `z-audit` security review

```bash
# Step 1: Run all Go tests
cd backend/go-core && go test ./... -v
# Expected: All PASS

# Step 2: Run frontend build + tests
cd frontend && npm run build && npm run test:ci
# Expected: PASS

# Step 3: z-audit security review (manual or automated)
# Check: JWT in HttpOnly cookie, SameSite=Strict, bcrypt passwords, 
#        tenant isolation on approval endpoints, rate limit on /login
```

- [ ] **Step 1: Run full test suite**
- [ ] **Step 2: Fix any failures**
- [ ] **Step 3: z-audit review** (check off each item)
- [ ] **Step 4: Commit**
```bash
git add -A
git commit -m "test: module 1 integration tests pass; security review complete"
```

---

## Module 2: gRPC/AI Serving Integration

*(Similar detailed task breakdown for Tasks 2.1–2.8)*

**Summary of Tasks:**
- 2.1: Verify/extend `ai_serving_service.py` endpoints (`/ingest`, `/status/{invoice_id}`)
- 2.2: Go AI client (`internal/infra/ai/ai_client.go`) with HTTP calls to FastAPI
- 2.3: AI Usecase (`internal/usecase/ai/ai_usecase.go`) — orchestrate inference
- 2.4: Inference Handler (`internal/handler/inference_handler.go`) — REST endpoints
- 2.5: Register `/inference/*` routes in `main.go`
- 2.6: Update Invoice handler → trigger anomaly detection after OCR
- 2.7: Frontend: Add anomaly badge to invoice list/detail
- 2.8: Integration test + z-audit

---

## Module 3: Accounting Detail UI

**Summary of Tasks:**
- 3.1: Frontend `ChartOfAccountsPage` with `CoATable` + `CoADrawer` (create/edit)
- 3.2: Frontend `JournalEntriesPage` with `JETable` + `JEDrawer` (create + detail)
- 3.3: Validation: debit == credit on client & server
- 3.4: `frontend-design` agent review for UI/UX
- 3.5: `executing-plans` implement
- 3.6: `z-audit` review

---

## Module 4: Integration Tests & Hardening

**Summary of Tasks:**
- 4.1: Playwright E2E tests for O2P cycle, O2C cycle, Auth, Approval
- 4.2: Rate limiter middleware (chi Throttle)
- 4.3: CORS middleware (whitelist FRONTEND_ORIGIN)
- 4.4: Health check endpoint comprehensive (DB, Kafka, Redis, AI)
- 4.5: Standardized error response format
- 4.6: `systematic-debugging` for flaky tests
- 4.7: `z-audit` review

---

## Module 5: Production Readiness

**Summary of Tasks:**
- 5.1: Structured logging with correlation IDs (zerolog)
- 5.2: Prometheus metrics endpoint (`/metrics`)
- 5.3: Config validation at startup (all required env vars)
- 5.4: Graceful shutdown (wait for Kafka consumer, HTTP server)
- 5.5: Sentry error tracking integration
- 5.6: GitHub Actions CI/CD pipeline (test → build → deploy)
- 5.7: K8s manifests update (resource limits, health probes)
- 5.8: `dev-pipeline` agent setup pipeline
- 5.9: `z-audit` final security review
- 5.10: Deploy to staging, smoke test

---

## Execution Handoff

**Plan complete and saved to `docs/superpowers/plans/2026-07-16-luminaflow-remaining-25-percent.md`.**

**Two execution options:**

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration  
**REQUIRED SUB-SKILL:** Use `superpowers:subagent-driven-development`

**2. Inline Execution** — Execute tasks in this session using `executing-plans`, batch execution with checkpoints  
**REQUIRED SUB-SKILL:** Use `superpowers:executing-plans`

**Which approach?**