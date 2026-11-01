package wms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/wms"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pinExpiryClock(t *testing.T, now time.Time) {
	t.Helper()
	prev := domain.ExpiryClock
	domain.ExpiryClock = func() time.Time { return now }
	t.Cleanup(func() { domain.ExpiryClock = prev })
}

func expLine(batch string, y int, m time.Month, d int, qty int64) domain.ExpiringBatchLine {
	return domain.ExpiringBatchLine{
		BatchID: uuid.New(), BatchNumber: batch,
		ExpiryDate: time.Date(y, m, d, 0, 0, 0, 0, time.UTC),
		Quantity:   decimal.NewFromInt(qty),
	}
}

func TestGetExpiryReport_ClassifiesAndSummarises(t *testing.T) {
	pinExpiryClock(t, time.Date(2026, 11, 1, 2, 0, 0, 0, time.UTC)) // 1 Nov 09:00 WIB
	repo := newMockWMSRepo()
	tenantID, userID := uuid.New(), uuid.New()
	wh := domain.Warehouse{ID: uuid.New(), TenantID: tenantID, Code: "WH1", Name: "Gudang 1"}
	repo.warehouses[wh.ID] = wh
	repo.expiringLines = []domain.ExpiringBatchLine{
		expLine("EXPIRED", 2026, 10, 31, 12), // -1 day
		expLine("TODAY", 2026, 11, 1, 3),     // 0 days -> critical
		expLine("WEEK", 2026, 11, 8, 4),      // 7 days -> critical
		expLine("MONTH", 2026, 11, 20, 5),    // 19 days -> warning
	}
	u := uc.New(repo)

	rep, err := u.GetExpiryReport(context.Background(), tenantID, userID, "admin", nil, 0)
	require.NoError(t, err)

	assert.Equal(t, uc.DefaultExpiryWindowDays, rep.Days)
	assert.Equal(t, "2026-11-01", rep.AsOf)
	assert.Equal(t, []uuid.UUID{wh.ID}, repo.expiringWarehouseIDs)
	assert.Equal(t, "2026-12-01", repo.expiringUntil.Format("2006-01-02"))

	require.Len(t, rep.Lines, 4)
	assert.Equal(t, domain.ExpirySeverityExpired, rep.Lines[0].Severity)
	assert.Equal(t, -1, rep.Lines[0].DaysLeft)
	assert.Equal(t, domain.ExpirySeverityCritical, rep.Lines[1].Severity)
	assert.Equal(t, domain.ExpirySeverityCritical, rep.Lines[2].Severity)
	assert.Equal(t, domain.ExpirySeverityWarning, rep.Lines[3].Severity)

	assert.Equal(t, 1, rep.Summary.ExpiredBatches)
	assert.True(t, rep.Summary.ExpiredQty.Equal(decimal.NewFromInt(12)))
	assert.Equal(t, 2, rep.Summary.CriticalBatches)
	assert.True(t, rep.Summary.CriticalQty.Equal(decimal.NewFromInt(7)))
	assert.Equal(t, 1, rep.Summary.WarningBatches)
	assert.True(t, rep.Summary.WarningQty.Equal(decimal.NewFromInt(5)))
}

func TestGetExpiryReport_ClampsWindowAndChecksWarehouseAccess(t *testing.T) {
	pinExpiryClock(t, time.Date(2026, 11, 1, 2, 0, 0, 0, time.UTC))
	repo := newMockWMSRepo()
	tenantID, userID := uuid.New(), uuid.New()
	wh := domain.Warehouse{ID: uuid.New(), TenantID: tenantID, Code: "WH1", Name: "Gudang 1"}
	repo.warehouses[wh.ID] = wh
	u := uc.New(repo)

	rep, err := u.GetExpiryReport(context.Background(), tenantID, userID, "admin", &wh.ID, 9999)
	require.NoError(t, err)
	assert.Equal(t, uc.MaxExpiryWindowDays, rep.Days)
	assert.NotNil(t, rep.Lines)

	// Staff without an assignment to this warehouse may not read its report.
	_, err = u.GetExpiryReport(context.Background(), tenantID, userID, "staff", &wh.ID, 30)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrUnauthorizedWarehouse) || errors.Is(err, domain.ErrForbidden), "got %v", err)

	// Without warehouse_id an unassigned staff user sees an empty report, not other warehouses.
	repo.expiringWarehouseIDs = nil
	rep, err = u.GetExpiryReport(context.Background(), tenantID, userID, "staff", nil, 30)
	require.NoError(t, err)
	assert.Empty(t, rep.Lines)
	assert.Nil(t, repo.expiringWarehouseIDs, "repo must not be queried with no accessible warehouse")
}
