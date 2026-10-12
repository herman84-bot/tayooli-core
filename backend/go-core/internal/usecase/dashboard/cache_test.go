package dashboard_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/dashboard"
)

type countingRepo struct {
	calls atomic.Int64
	delay time.Duration
}

func (r *countingRepo) GetSummary(ctx context.Context, tenantID uuid.UUID, days int) (*domain.DashboardSummary, error) {
	r.calls.Add(1)
	time.Sleep(r.delay)
	return &domain.DashboardSummary{}, nil
}

func TestCachedSummary_ConcurrentSameTenantHitsDBOnce(t *testing.T) {
	repo := &countingRepo{delay: 50 * time.Millisecond}
	uc := dashboard.NewWithCache(repo, time.Minute)
	tenant := uuid.New()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := uc.GetSummary(context.Background(), tenant, 30); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if got := repo.calls.Load(); got != 1 {
		t.Fatalf("expected 1 DB call for 50 concurrent requests, got %d", got)
	}
}

func TestCachedSummary_IsolatedPerTenantAndPeriod(t *testing.T) {
	repo := &countingRepo{}
	uc := dashboard.NewWithCache(repo, time.Minute)
	a, b := uuid.New(), uuid.New()
	ctx := context.Background()

	_, _ = uc.GetSummary(ctx, a, 30)
	_, _ = uc.GetSummary(ctx, b, 30)
	_, _ = uc.GetSummary(ctx, a, 7)
	_, _ = uc.GetSummary(ctx, a, 30) // cached
	if got := repo.calls.Load(); got != 3 {
		t.Fatalf("expected 3 DB calls (tenant a/30, b/30, a/7), got %d", got)
	}
}

func TestCachedSummary_ExpiresAfterTTL(t *testing.T) {
	repo := &countingRepo{}
	uc := dashboard.NewWithCache(repo, 20*time.Millisecond)
	tenant := uuid.New()
	ctx := context.Background()

	_, _ = uc.GetSummary(ctx, tenant, 30)
	time.Sleep(40 * time.Millisecond)
	_, _ = uc.GetSummary(ctx, tenant, 30)
	if got := repo.calls.Load(); got != 2 {
		t.Fatalf("expected refetch after TTL, got %d calls", got)
	}
}

func TestCachedSummary_ReturnsCopy(t *testing.T) {
	repo := &countingRepo{}
	uc := dashboard.NewWithCache(repo, time.Minute)
	tenant := uuid.New()
	ctx := context.Background()

	s1, _ := uc.GetSummary(ctx, tenant, 30)
	s1.Vendors.Active = 999 // a caller mutating its result must not poison the cache
	s2, _ := uc.GetSummary(ctx, tenant, 30)
	if s2.Vendors.Active == 999 {
		t.Fatal("cache returned a shared pointer")
	}
}

type panicRepo struct{ calls atomic.Int64 }

func (r *panicRepo) GetSummary(ctx context.Context, tenantID uuid.UUID, days int) (*domain.DashboardSummary, error) {
	if r.calls.Add(1) == 1 {
		panic("boom")
	}
	return &domain.DashboardSummary{}, nil
}

// Regression: a panicking fetch must not leave an in-flight entry that blocks
// every later request for that tenant forever.
func TestCachedSummary_PanicDoesNotWedgeKey(t *testing.T) {
	repo := &panicRepo{}
	uc := dashboard.NewWithCache(repo, time.Minute)
	tenant := uuid.New()
	if _, err := uc.GetSummary(context.Background(), tenant, 30); err == nil {
		t.Fatal("expected error from panicking fetch")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := uc.GetSummary(ctx, tenant, 30); err != nil {
		t.Fatalf("second call must refetch, got %v", err)
	}
}
