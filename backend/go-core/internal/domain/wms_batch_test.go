package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllocateFEFO_OrderAndDepletion(t *testing.T) {
	now := time.Now().UTC()
	b1 := uuid.New()
	b2 := uuid.New()
	b3 := uuid.New()
	loc := uuid.New()
	prod := uuid.New()

	dEarly := now.Add(10 * 24 * time.Hour)
	dLate := now.Add(30 * 24 * time.Hour)

	balances := []domain.BatchBalance{
		{
			BatchID:      b3,
			BatchNumber:  "NO-EXP",
			ExpiryDate:   nil,
			Status:       domain.StockBatchStatusReleased,
			LocationID:   loc,
			LocationCode: "A-01",
			ProductID:    prod,
			Quantity:     decimal.NewFromInt(50),
			CreatedAt:    now.Add(-1 * time.Hour),
		},
		{
			BatchID:      b2,
			BatchNumber:  "LATE",
			ExpiryDate:   &dLate,
			Status:       domain.StockBatchStatusReleased,
			LocationID:   loc,
			LocationCode: "A-01",
			ProductID:    prod,
			Quantity:     decimal.NewFromInt(20),
			CreatedAt:    now,
		},
		{
			BatchID:      b1,
			BatchNumber:  "EARLY",
			ExpiryDate:   &dEarly,
			Status:       domain.StockBatchStatusReleased,
			LocationID:   loc,
			LocationCode: "A-01",
			ProductID:    prod,
			Quantity:     decimal.NewFromInt(10),
			CreatedAt:    now,
		},
	}

	// 1. Partial: takes all of b1, part of b2.
	allocs, err := domain.AllocateFEFO(balances, decimal.NewFromInt(15), false)
	require.NoError(t, err)
	require.Len(t, allocs, 2)
	assert.Equal(t, b1, allocs[0].BatchID)
	assert.True(t, allocs[0].Quantity.Equal(decimal.NewFromInt(10)))
	assert.Equal(t, b2, allocs[1].BatchID)
	assert.True(t, allocs[1].Quantity.Equal(decimal.NewFromInt(5)))

	// 2. Exact sum across early + late: consumes both, leaves no-exp untouched.
	allocs, err = domain.AllocateFEFO(balances, decimal.NewFromInt(30), false)
	require.NoError(t, err)
	require.Len(t, allocs, 2)
	assert.Equal(t, b1, allocs[0].BatchID)
	assert.Equal(t, b2, allocs[1].BatchID)

	// 3. Needs no-exp batch too.
	allocs, err = domain.AllocateFEFO(balances, decimal.NewFromInt(50), false)
	require.NoError(t, err)
	require.Len(t, allocs, 3)
	assert.Equal(t, b3, allocs[2].BatchID)
	assert.True(t, allocs[2].Quantity.Equal(decimal.NewFromInt(20)))

	// 4. Over-request returns ErrInsufficientStock with no partial allocation.
	_, err = domain.AllocateFEFO(balances, decimal.NewFromInt(90), false)
	require.ErrorIs(t, err, domain.ErrInsufficientStock)

	// 5. Zero/negative request rejected.
	_, err = domain.AllocateFEFO(balances, decimal.Zero, false)
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestAllocateFEFO_OnHoldFiltering(t *testing.T) {
	now := time.Now().UTC()
	bHold := uuid.New()
	bRel := uuid.New()
	loc := uuid.New()
	prod := uuid.New()
	d := now.Add(5 * 24 * time.Hour)

	balances := []domain.BatchBalance{
		{
			BatchID:      bHold,
			BatchNumber:  "HOLD",
			ExpiryDate:   &d,
			Status:       domain.StockBatchStatusOnHold,
			LocationID:   loc,
			LocationCode: "A-01",
			ProductID:    prod,
			Quantity:     decimal.NewFromInt(100),
		},
		{
			BatchID:      bRel,
			BatchNumber:  "REL",
			ExpiryDate:   nil,
			Status:       domain.StockBatchStatusReleased,
			LocationID:   loc,
			LocationCode: "A-01",
			ProductID:    prod,
			Quantity:     decimal.NewFromInt(10),
		},
	}

	// Sales (includeOnHold=false) skip bHold even though it expires earlier.
	allocs, err := domain.AllocateFEFO(balances, decimal.NewFromInt(5), false)
	require.NoError(t, err)
	require.Len(t, allocs, 1)
	assert.Equal(t, bRel, allocs[0].BatchID)

	// Asking for 15 fails because only 10 released stock is available.
	_, err = domain.AllocateFEFO(balances, decimal.NewFromInt(15), false)
	require.ErrorIs(t, err, domain.ErrInsufficientStock)

	// Adjustments (includeOnHold=true) may touch held stock.
	allocs, err = domain.AllocateFEFO(balances, decimal.NewFromInt(15), true)
	require.NoError(t, err)
	require.Len(t, allocs, 1)
	assert.Equal(t, bHold, allocs[0].BatchID)
	assert.True(t, allocs[0].Quantity.Equal(decimal.NewFromInt(15)))
}
