package dashboard

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// cacheKey isolates entries per tenant and period; a tenant can never read
// another tenant's summary.
type cacheKey struct {
	tenant uuid.UUID
	days   int
}

type cacheEntry struct {
	ready   chan struct{} // closed when the fetch finished
	summary *domain.DashboardSummary
	err     error
	expires time.Time
}

// summaryCache is a per-process TTL cache with request coalescing: concurrent
// requests for the same (tenant, days) share ONE database fetch instead of each
// running the ~15 aggregate queries (dashboard auto-refresh from many open tabs
// would otherwise multiply DB load). Errors are not cached.
type summaryCache struct {
	ttl     time.Duration
	mu      sync.Mutex
	entries map[cacheKey]*cacheEntry
}

func (c *summaryCache) get(ctx context.Context, key cacheKey, fetch func() (*domain.DashboardSummary, error)) (*domain.DashboardSummary, error) {
	now := time.Now()
	c.mu.Lock()
	if e, ok := c.entries[key]; ok {
		select {
		case <-e.ready:
			if e.err == nil && now.Before(e.expires) {
				c.mu.Unlock()
				return cloneSummary(e.summary), nil
			}
			// expired or failed: fall through and refetch
		default:
			// in flight: wait for the leader
			c.mu.Unlock()
			select {
			case <-e.ready:
				if e.err != nil {
					return nil, e.err
				}
				return cloneSummary(e.summary), nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	e := &cacheEntry{ready: make(chan struct{})}
	c.entries[key] = e
	c.evictExpiredLocked(now)
	c.mu.Unlock()

	c.runLeader(key, e, fetch)
	if e.err != nil {
		return nil, e.err
	}
	return cloneSummary(e.summary), nil
}

// runLeader always releases waiters and never leaves a failed entry cached,
// even if fetch panics (the panic is converted into an error).
func (c *summaryCache) runLeader(key cacheKey, e *cacheEntry, fetch func() (*domain.DashboardSummary, error)) {
	defer func() {
		if r := recover(); r != nil {
			e.summary, e.err = nil, fmt.Errorf("dashboard summary: panic: %v", r)
		}
		e.expires = time.Now().Add(c.ttl)
		if e.err != nil {
			c.mu.Lock()
			if c.entries[key] == e {
				delete(c.entries, key)
			}
			c.mu.Unlock()
		}
		close(e.ready)
	}()
	e.summary, e.err = fetch()
}

// evictExpiredLocked bounds memory to the set of recently active tenants.
func (c *summaryCache) evictExpiredLocked(now time.Time) {
	for k, e := range c.entries {
		select {
		case <-e.ready:
			if now.After(e.expires) {
				delete(c.entries, k)
			}
		default:
		}
	}
}

// cloneSummary deep-copies the slices so a caller mutating its result cannot
// corrupt the cached value shared with other requests.
func cloneSummary(s *domain.DashboardSummary) *domain.DashboardSummary {
	if s == nil {
		return nil
	}
	c := *s
	c.MonthlyTrend = slices.Clone(s.MonthlyTrend)
	c.TopVendors = slices.Clone(s.TopVendors)
	c.POS.RecentOrders = slices.Clone(s.POS.RecentOrders)
	c.WMS.LowStockItems = slices.Clone(s.WMS.LowStockItems)
	c.Outbound.TopProducts = slices.Clone(s.Outbound.TopProducts)
	c.Outbound.TopCustomers = slices.Clone(s.Outbound.TopCustomers)
	for i := range c.Outbound.TopCustomers {
		if id := c.Outbound.TopCustomers[i].CustomerID; id != nil {
			v := *id
			c.Outbound.TopCustomers[i].CustomerID = &v
		}
	}
	return &c
}
