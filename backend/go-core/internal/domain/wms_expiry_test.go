package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withClock pins domain.ExpiryClock for one test.
func withClock(t *testing.T, now time.Time) {
	t.Helper()
	prev := domain.ExpiryClock
	domain.ExpiryClock = func() time.Time { return now }
	t.Cleanup(func() { domain.ExpiryClock = prev })
}

func date(y int, m time.Month, d int) *time.Time {
	t := time.Date(y, m, d, 0, 0, 0, 0, time.UTC) // Postgres DATE as scanned by lib/pq
	return &t
}

func TestIsBatchExpired_UsesWIBCalendarDay(t *testing.T) {
	// 16 Oct 00:30 WIB == 15 Oct 17:30 UTC. A lot expiring 15 Oct is already
	// expired in Indonesia even though the UTC date is still 15 Oct.
	now := time.Date(2026, 10, 15, 17, 30, 0, 0, time.UTC)

	assert.True(t, domain.IsBatchExpired(date(2026, 10, 15), now), "expired the day after its expiry date (WIB)")
	assert.False(t, domain.IsBatchExpired(date(2026, 10, 16), now), "usable on its expiry date itself")
	assert.False(t, domain.IsBatchExpired(nil, now), "no expiry date never expires")

	assert.Equal(t, -1, domain.DaysUntilExpiry(*date(2026, 10, 15), now))
	assert.Equal(t, 0, domain.DaysUntilExpiry(*date(2026, 10, 16), now))
	assert.Equal(t, 30, domain.DaysUntilExpiry(*date(2026, 11, 15), now))
}

func TestExpirySeverityBuckets(t *testing.T) {
	assert.Equal(t, domain.ExpirySeverityExpired, domain.ExpirySeverity(-1))
	assert.Equal(t, domain.ExpirySeverityCritical, domain.ExpirySeverity(0))
	assert.Equal(t, domain.ExpirySeverityCritical, domain.ExpirySeverity(domain.ExpiryCriticalDays))
	assert.Equal(t, domain.ExpirySeverityWarning, domain.ExpirySeverity(domain.ExpiryCriticalDays+1))
}

func TestIsOutboundStockRef(t *testing.T) {
	for _, ref := range []string{domain.StockRefDeliveryOrder, domain.StockRefPOS, domain.StockRefTransfer, domain.StockRefMarketplace} {
		assert.True(t, domain.IsOutboundStockRef(ref), ref)
	}
	for _, ref := range []string{domain.StockRefScrap, domain.StockRefOpname, domain.StockRefPutaway, domain.StockRefQC, "PICKING_DAMAGED"} {
		assert.False(t, domain.IsOutboundStockRef(ref), ref)
	}
}

func expiryBalances(loc, prod uuid.UUID) (expired, fresh domain.BatchBalance) {
	created := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	expired = domain.BatchBalance{
		BatchID: uuid.New(), BatchNumber: "EXP-OLD", ExpiryDate: date(2026, 10, 10),
		Status: domain.StockBatchStatusReleased, LocationID: loc, LocationCode: "A-01",
		ProductID: prod, Quantity: decimal.NewFromInt(40), CreatedAt: created,
	}
	fresh = domain.BatchBalance{
		BatchID: uuid.New(), BatchNumber: "FRESH", ExpiryDate: date(2026, 12, 31),
		Status: domain.StockBatchStatusReleased, LocationID: loc, LocationCode: "A-01",
		ProductID: prod, Quantity: decimal.NewFromInt(10), CreatedAt: created,
	}
	return expired, fresh
}

func TestAllocateFEFO_SkipsExpiredBatchesForOutbound(t *testing.T) {
	withClock(t, time.Date(2026, 10, 20, 3, 0, 0, 0, time.UTC))
	loc, prod := uuid.New(), uuid.New()
	expired, fresh := expiryBalances(loc, prod)
	balances := []domain.BatchBalance{expired, fresh}

	// Before the fix FEFO picked EXP-OLD first because it expires earliest.
	allocs, err := domain.AllocateFEFO(balances, decimal.NewFromInt(5), false)
	require.NoError(t, err)
	require.Len(t, allocs, 1)
	assert.Equal(t, fresh.BatchID, allocs[0].BatchID, "expired batch must never be allocated for outbound")

	// Only expired stock could cover the request -> explicit expired error.
	_, err = domain.AllocateFEFO(balances, decimal.NewFromInt(30), false)
	assert.True(t, errors.Is(err, domain.ErrExpiredStockBlocked), "got %v", err)

	// Not enough stock even counting expired lots -> plain insufficient stock.
	_, err = domain.AllocateFEFO(balances, decimal.NewFromInt(100), false)
	assert.True(t, errors.Is(err, domain.ErrInsufficientStock), "got %v", err)
	assert.False(t, errors.Is(err, domain.ErrExpiredStockBlocked))
}

func TestAllocateFEFO_ScrapAndOpnameMayTakeExpiredStock(t *testing.T) {
	withClock(t, time.Date(2026, 10, 20, 3, 0, 0, 0, time.UTC))
	loc, prod := uuid.New(), uuid.New()
	expired, fresh := expiryBalances(loc, prod)

	allocs, err := domain.AllocateFEFO([]domain.BatchBalance{expired, fresh}, decimal.NewFromInt(40), true)
	require.NoError(t, err)
	require.Len(t, allocs, 1)
	assert.Equal(t, expired.BatchID, allocs[0].BatchID, "scrap/opname drains the expired lot first")
}

func TestAllocateFEFO_BatchUsableOnItsExpiryDate(t *testing.T) {
	// 10 Oct 23:00 WIB: a lot expiring 10 Oct is still usable.
	withClock(t, time.Date(2026, 10, 10, 16, 0, 0, 0, time.UTC))
	loc, prod := uuid.New(), uuid.New()
	expired, _ := expiryBalances(loc, prod)

	allocs, err := domain.AllocateFEFO([]domain.BatchBalance{expired}, decimal.NewFromInt(5), false)
	require.NoError(t, err)
	assert.Equal(t, expired.BatchID, allocs[0].BatchID)

	// One hour later (11 Oct 00:00 WIB) it is blocked.
	withClock(t, time.Date(2026, 10, 10, 17, 0, 0, 0, time.UTC))
	_, err = domain.AllocateFEFO([]domain.BatchBalance{expired}, decimal.NewFromInt(5), false)
	assert.True(t, errors.Is(err, domain.ErrExpiredStockBlocked), "got %v", err)
}
