package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// fakeIdemStore mirrors the Postgres semantics: Reserve is atomic per (tenant, key).
type fakeIdemStore struct {
	mu         sync.Mutex
	rows       map[string]*domain.IdempotencyRecord
	reserveErr error
}

func newFakeIdemStore() *fakeIdemStore {
	return &fakeIdemStore{rows: map[string]*domain.IdempotencyRecord{}}
}

func (s *fakeIdemStore) k(t uuid.UUID, key string) string { return t.String() + "|" + key }

func (s *fakeIdemStore) Reserve(_ context.Context, t uuid.UUID, key, hash string) (*domain.IdempotencyRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reserveErr != nil {
		return nil, false, s.reserveErr
	}
	if r, ok := s.rows[s.k(t, key)]; ok {
		cpy := *r
		return &cpy, false, nil
	}
	s.rows[s.k(t, key)] = &domain.IdempotencyRecord{Key: key, RequestHash: hash}
	return nil, true, nil
}

func (s *fakeIdemStore) Complete(_ context.Context, t uuid.UUID, key string, status int, body []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.rows[s.k(t, key)]
	r.Completed, r.StatusCode, r.ResponseBody = true, status, append([]byte(nil), body...)
	return nil
}

func (s *fakeIdemStore) Release(_ context.Context, t uuid.UUID, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rows, s.k(t, key))
	return nil
}

// countingHandler simulates a stock-mutating endpoint: every call that reaches
// it is one more side effect.
type countingHandler struct {
	calls  atomic.Int32
	status int
}

func (c *countingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.calls.Add(1)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c.status)
	_, _ = w.Write([]byte(`{"n":` + string(rune('0'+c.calls.Load())) + `}`))
}

var idemTenant = uuid.MustParse("11111111-1111-1111-1111-111111111111")

func idemReq(t uuid.UUID, path, key, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	if t != uuid.Nil {
		req = req.WithContext(context.WithValue(req.Context(), appMiddleware.TenantIDKey, t))
	}
	return req
}

func serve(h http.Handler, r *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	return rr
}

func TestIdempotent_RetryReplaysWithoutSecondEffect(t *testing.T) {
	inner := &countingHandler{status: http.StatusCreated}
	h := Idempotent(newFakeIdemStore())(inner)

	first := serve(h, idemReq(idemTenant, "/wms/scraps", "k1", `{"q":1}`))
	second := serve(h, idemReq(idemTenant, "/wms/scraps", "k1", `{"q":1}`))

	if inner.calls.Load() != 1 {
		t.Fatalf("handler ran %d times, want 1", inner.calls.Load())
	}
	if second.Code != http.StatusCreated || second.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatalf("retry = %d replayed=%q", second.Code, second.Header().Get("Idempotent-Replayed"))
	}
	if !bytes.Equal(first.Body.Bytes(), second.Body.Bytes()) {
		t.Fatalf("replayed body differs: %s vs %s", first.Body, second.Body)
	}
}

func TestIdempotent_ConcurrentDuplicatesRunOnce(t *testing.T) {
	inner := &countingHandler{status: http.StatusCreated}
	h := Idempotent(newFakeIdemStore())(inner)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			serve(h, idemReq(idemTenant, "/wms/scraps", "same", `{"q":1}`))
		}()
	}
	wg.Wait()
	if inner.calls.Load() != 1 {
		t.Fatalf("handler ran %d times under concurrency, want 1", inner.calls.Load())
	}
}

func TestIdempotent_KeyReusedForDifferentBodyOrEndpointIs422(t *testing.T) {
	inner := &countingHandler{status: http.StatusCreated}
	h := Idempotent(newFakeIdemStore())(inner)

	serve(h, idemReq(idemTenant, "/wms/scraps", "k", `{"q":1}`))
	if rr := serve(h, idemReq(idemTenant, "/wms/scraps", "k", `{"q":2}`)); rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("other body = %d, want 422", rr.Code)
	}
	// Same key + same body on a different endpoint must not replay the scrap.
	if rr := serve(h, idemReq(idemTenant, "/wms/transfers", "k", `{"q":1}`)); rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("other endpoint = %d, want 422", rr.Code)
	}
	if inner.calls.Load() != 1 {
		t.Fatalf("handler ran %d times, want 1", inner.calls.Load())
	}
}

func TestIdempotent_FailureReleasesKeySoUserCanRetry(t *testing.T) {
	store := newFakeIdemStore()
	inner := &countingHandler{status: http.StatusConflict}
	h := Idempotent(store)(inner)

	serve(h, idemReq(idemTenant, "/wms/scraps", "k", `{}`))
	inner.status = http.StatusCreated
	if rr := serve(h, idemReq(idemTenant, "/wms/scraps", "k", `{}`)); rr.Code != http.StatusCreated {
		t.Fatalf("retry after failure = %d, want 201", rr.Code)
	}
	if inner.calls.Load() != 2 {
		t.Fatalf("handler ran %d times, want 2 (failure must not be cached)", inner.calls.Load())
	}
}

func TestIdempotent_InFlightDuplicateGets409(t *testing.T) {
	store := newFakeIdemStore()
	inner := &countingHandler{status: http.StatusCreated}
	h := Idempotent(store)(inner)
	if _, fresh, _ := store.Reserve(context.Background(), idemTenant, "k", requestFingerprint(http.MethodPost, "/wms/scraps", []byte(`{}`))); !fresh {
		t.Fatal("setup")
	}
	rr := serve(h, idemReq(idemTenant, "/wms/scraps", "k", `{}`))
	if rr.Code != http.StatusConflict || rr.Header().Get("Retry-After") == "" {
		t.Fatalf("in-flight = %d retry-after=%q, want 409 with Retry-After", rr.Code, rr.Header().Get("Retry-After"))
	}
	if inner.calls.Load() != 0 {
		t.Fatal("in-flight duplicate must not run the handler")
	}
}

func TestIdempotent_NoKeyPassesThrough(t *testing.T) {
	inner := &countingHandler{status: http.StatusCreated}
	h := Idempotent(newFakeIdemStore())(inner)
	serve(h, idemReq(idemTenant, "/wms/scraps", "", `{}`))
	serve(h, idemReq(idemTenant, "/wms/scraps", "", `{}`))
	if inner.calls.Load() != 2 {
		t.Fatalf("handler ran %d times, want 2", inner.calls.Load())
	}
}

func TestIdempotent_KeysAreTenantScoped(t *testing.T) {
	inner := &countingHandler{status: http.StatusCreated}
	h := Idempotent(newFakeIdemStore())(inner)
	other := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	serve(h, idemReq(idemTenant, "/wms/scraps", "k", `{}`))
	if rr := serve(h, idemReq(other, "/wms/scraps", "k", `{}`)); rr.Header().Get("Idempotent-Replayed") == "true" {
		t.Fatal("another tenant must never receive this tenant's replay")
	}
	if inner.calls.Load() != 2 {
		t.Fatalf("handler ran %d times, want 2", inner.calls.Load())
	}
}

func TestIdempotent_RejectsBadInputWithoutRunning(t *testing.T) {
	inner := &countingHandler{status: http.StatusCreated}
	h := Idempotent(newFakeIdemStore())(inner)

	long := string(bytes.Repeat([]byte("a"), maxIdempotencyKeyLen+1))
	if rr := serve(h, idemReq(idemTenant, "/x", long, `{}`)); rr.Code != http.StatusBadRequest {
		t.Fatalf("long key = %d, want 400", rr.Code)
	}
	if rr := serve(h, idemReq(uuid.Nil, "/x", "k", `{}`)); rr.Code != http.StatusUnauthorized {
		t.Fatalf("no tenant = %d, want 401", rr.Code)
	}
	big := string(bytes.Repeat([]byte("x"), maxIdempotentBody+1))
	if rr := serve(h, idemReq(idemTenant, "/x", "k2", big)); rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body = %d, want 413", rr.Code)
	}
	if inner.calls.Load() != 0 {
		t.Fatalf("handler ran %d times, want 0", inner.calls.Load())
	}
}

func TestIdempotent_StoreFailureIsGeneric500(t *testing.T) {
	store := newFakeIdemStore()
	store.reserveErr = errors.New("pq: connection refused 10.0.0.9")
	inner := &countingHandler{status: http.StatusCreated}
	rr := serve(Idempotent(store)(inner), idemReq(idemTenant, "/x", "k", `{}`))
	if rr.Code != http.StatusInternalServerError || bytes.Contains(rr.Body.Bytes(), []byte("10.0.0.9")) {
		t.Fatalf("store failure = %d %s, want generic 500", rr.Code, rr.Body)
	}
	if inner.calls.Load() != 0 {
		t.Fatal("must not run the write when the guard is unavailable")
	}
}
