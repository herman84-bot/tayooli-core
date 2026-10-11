package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
)

// memIdemStore is an in-memory domain.IdempotencyStore with the same
// semantics as the Postgres one: Reserve is atomic per (tenant, key).
type memIdemStore struct {
	mu   sync.Mutex
	rows map[string]*domain.IdempotencyRecord
}

func newMemIdemStore() *memIdemStore {
	return &memIdemStore{rows: map[string]*domain.IdempotencyRecord{}}
}

func (s *memIdemStore) k(t uuid.UUID, key string) string { return t.String() + "|" + key }

func (s *memIdemStore) Reserve(ctx context.Context, tenantID uuid.UUID, key, requestHash string) (*domain.IdempotencyRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.rows[s.k(tenantID, key)]; ok {
		cpy := *r
		return &cpy, false, nil
	}
	s.rows[s.k(tenantID, key)] = &domain.IdempotencyRecord{Key: key, RequestHash: requestHash}
	return nil, true, nil
}

func (s *memIdemStore) Complete(ctx context.Context, tenantID uuid.UUID, key string, status int, body []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.rows[s.k(tenantID, key)]
	r.Completed, r.StatusCode, r.ResponseBody = true, status, append([]byte(nil), body...)
	return nil
}

func (s *memIdemStore) Release(ctx context.Context, tenantID uuid.UUID, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rows, s.k(tenantID, key))
	return nil
}

func idemBody(e *e2eEnv, qty int) []byte {
	return []byte(`{"items":[{"product_id":"` + e.productID.String() + `","qty":` + itoa(qty) + `,"price":10000}],"payments":[{"method":"CASH","amount":10000}]}`)
}

func idemCheckout(e *e2eEnv, h *handler.POSHandler, key string, qty int) *httptest.ResponseRecorder {
	body := idemBody(e, qty)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	req = tenantCtx(req, e.tenantID, e.userID)
	rr := httptest.NewRecorder()
	h.Checkout(rr, req)
	return rr
}

func itoa(n int) string { return string(rune('0' + n)) }

func stockOf(e *e2eEnv) float64 {
	var total float64
	for _, inv := range e.inventory.stocks {
		total += inv.Quantity
	}
	return total
}

// A retry with the same key (network dropped after commit) must replay the
// first receipt, not sell and deduct stock a second time.
func TestPOSCheckout_SameIdempotencyKeyIsRecordedOnce(t *testing.T) {
	e := buildE2E(t, false)
	h := e.handler.WithIdempotency(newMemIdemStore())

	first := idemCheckout(e, h, "cart-abc-1", 1)
	if first.Code != http.StatusCreated {
		t.Fatalf("first checkout = %d %s", first.Code, first.Body.String())
	}
	second := idemCheckout(e, h, "cart-abc-1", 1)
	if second.Code != http.StatusCreated {
		t.Fatalf("retry = %d %s, want replayed 201", second.Code, second.Body.String())
	}
	if second.Header().Get("Idempotent-Replayed") != "true" {
		t.Error("retry must be marked as replayed")
	}
	if !bytes.Equal(first.Body.Bytes(), second.Body.Bytes()) {
		t.Errorf("retry body differs:\n%s\n%s", first.Body.String(), second.Body.String())
	}
	if len(e.posRepo.orders) != 1 {
		t.Fatalf("orders = %d, want 1", len(e.posRepo.orders))
	}
	if got := stockOf(e); got != 99 {
		t.Fatalf("stock = %v, want 99 (deducted once)", got)
	}
}

// Reusing a key for a different cart is a client bug; refuse it.
func TestPOSCheckout_KeyReusedWithDifferentBodyIsRejected(t *testing.T) {
	e := buildE2E(t, false)
	h := e.handler.WithIdempotency(newMemIdemStore())

	if rr := idemCheckout(e, h, "k1", 1); rr.Code != http.StatusCreated {
		t.Fatalf("first = %d", rr.Code)
	}
	if rr := idemCheckout(e, h, "k1", 2); rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("reuse with other body = %d, want 422", rr.Code)
	}
	if len(e.posRepo.orders) != 1 {
		t.Fatalf("orders = %d, want 1", len(e.posRepo.orders))
	}
}

// A failed checkout must free the key so the cashier can retry.
func TestPOSCheckout_FailedCheckoutReleasesKey(t *testing.T) {
	e := buildE2E(t, false)
	store := newMemIdemStore()
	h := e.handler.WithIdempotency(store)

	if rr := idemCheckout(e, h, "k2", 0); rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid qty = %d, want 400", rr.Code)
	}
	if _, fresh, _ := store.Reserve(context.Background(), e.tenantID, "k2", "x"); !fresh {
		t.Fatal("key must be released after a failed checkout")
	}
}

// A request that arrives while the first one is still running gets 409.
func TestPOSCheckout_InFlightDuplicateGets409(t *testing.T) {
	e := buildE2E(t, false)
	store := newMemIdemStore()
	h := e.handler.WithIdempotency(store)
	// Simulate an in-flight first request holding the key.
	if _, fresh, _ := store.Reserve(context.Background(), e.tenantID, "k3", handler.CheckoutBodyHash(idemBody(e, 1))); !fresh {
		t.Fatal("setup")
	}
	if rr := idemCheckout(e, h, "k3", 1); rr.Code != http.StatusConflict {
		t.Fatalf("in-flight duplicate = %d, want 409", rr.Code)
	}
	if len(e.posRepo.orders) != 0 {
		t.Fatal("duplicate must not create an order")
	}
}

// Without a key the old behavior is unchanged (backward compatible).
func TestPOSCheckout_NoKeyKeepsOldBehavior(t *testing.T) {
	e := buildE2E(t, false)
	h := e.handler.WithIdempotency(newMemIdemStore())
	idemCheckout(e, h, "", 1)
	idemCheckout(e, h, "", 1)
	if len(e.posRepo.orders) != 2 {
		t.Fatalf("orders = %d, want 2", len(e.posRepo.orders))
	}
}

// Key from another tenant must not replay this tenant's receipt.
func TestPOSCheckout_KeysAreTenantScoped(t *testing.T) {
	store := newMemIdemStore()
	if _, fresh, _ := store.Reserve(context.Background(), e2eOther, "shared", "h"); !fresh {
		t.Fatal("setup")
	}
	e := buildE2E(t, false)
	h := e.handler.WithIdempotency(store)
	if rr := idemCheckout(e, h, "shared", 1); rr.Code != http.StatusCreated {
		t.Fatalf("other tenant key = %d, want 201", rr.Code)
	}
}
