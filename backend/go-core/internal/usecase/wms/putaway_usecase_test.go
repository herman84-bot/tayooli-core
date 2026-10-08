package wms_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/wms"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPutawayAndSprint1Usecase(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	adminID := uuid.New()
	staffID := uuid.New()
	auditorID := uuid.New()

	repo := newMockWMSRepo()
	usecase := uc.New(repo)

	whID := uuid.New()
	repo.warehouses[whID] = domain.Warehouse{ID: whID, TenantID: tenantID, Name: "Gudang Utama"}
	repo.userWarehouses[fmt.Sprintf("%s:%s", tenantID, staffID)] = []uuid.UUID{whID}

	stgLoc, err := repo.GetOrCreateStagingLocation(ctx, tenantID, whID)
	require.NoError(t, err)

	rackA := uuid.New()
	repo.locations[rackA] = domain.WarehouseLocation{
		ID:          rackA,
		TenantID:    tenantID,
		WarehouseID: &whID,
		Code:        "RCK-A1",
		Type:        domain.LocationTypeInternal,
	}
	rackB := uuid.New()
	repo.locations[rackB] = domain.WarehouseLocation{
		ID:          rackB,
		TenantID:    tenantID,
		WarehouseID: &whID,
		Code:        "RCK-B1",
		Type:        domain.LocationTypeInternal,
	}

	prodID := uuid.New()
	batchID := uuid.New()
	now := time.Now().UTC()
	b := &domain.StockBatch{
		ID:          batchID,
		TenantID:    tenantID,
		ProductID:   prodID,
		BatchNumber: "LOT-2026-001",
		Status:      domain.StockBatchStatusReleased,
		CreatedAt:   now,
	}
	_, err = repo.GetOrCreateBatch(ctx, b)
	require.NoError(t, err)

	// Put 10 units into staging location
	repo.stockLevels[fmt.Sprintf("%s:%s:%s", tenantID, stgLoc.ID, prodID)] = decimal.NewFromInt(10)

	// 1. Settings (PDF-06)
	t.Run("WMS settings toggle require_release_approval", func(t *testing.T) {
		settings, err := usecase.GetWMSSettings(ctx, tenantID, adminID, "admin")
		require.NoError(t, err)
		assert.False(t, settings.RequireReleaseApproval)

		_, err = usecase.UpdateWMSSettings(ctx, tenantID, staffID, "warehouse", uc.UpdateWMSSettingsRequest{RequireReleaseApproval: true})
		assert.ErrorIs(t, err, domain.ErrUnauthorized)

		updated, err := usecase.UpdateWMSSettings(ctx, tenantID, adminID, "admin", uc.UpdateWMSSettingsRequest{RequireReleaseApproval: true})
		require.NoError(t, err)
		assert.True(t, updated.RequireReleaseApproval)
	})

	// 2. Default Rack (CR-03 / ADR-014 Invariant 4)
	t.Run("Product default rack CRUD", func(t *testing.T) {
		err := usecase.SetDefaultLocation(ctx, tenantID, adminID, "admin", uc.SetDefaultLocationRequest{
			ProductID:   prodID,
			WarehouseID: whID,
			LocationID:  rackA,
		})
		require.NoError(t, err)

		defs, err := usecase.ListDefaultLocations(ctx, tenantID, adminID, "admin", &prodID)
		require.NoError(t, err)
		require.Len(t, defs, 1)
		assert.Equal(t, rackA, defs[0].LocationID)
	})

	// 3. Putaway Pending List (sentry-wms §1.2)
	t.Run("Get putaway pending suggests default rack", func(t *testing.T) {
		pending, err := usecase.GetPutawayPending(ctx, tenantID, staffID, "warehouse", whID)
		require.NoError(t, err)
		require.Len(t, pending, 1)
		assert.Equal(t, "DEFAULT_RACK", pending[0].SuggestionSource)
		require.NotNil(t, pending[0].SuggestedLocationID)
		assert.Equal(t, rackA, *pending[0].SuggestedLocationID)
	})

	// 4. Invariant 4: Overriding default rack requires reason
	t.Run("Invariant 4: Overriding default rack without reason is rejected", func(t *testing.T) {
		_, err := usecase.ConfirmPutaway(ctx, tenantID, staffID, "warehouse", uc.PutawayRequest{
			WarehouseID:    whID,
			ProductID:      prodID,
			BatchID:        batchID,
			Quantity:       decimal.NewFromInt(5),
			DestLocationID: rackB, // Deviates from default rackA!
			Reason:         nil,   // No reason!
		})
		assert.ErrorIs(t, err, domain.ErrPutawayReasonRequired)

		// With reason -> succeeds
		reason := "Rak A1 sedang penuh maintenance"
		mov, err := usecase.ConfirmPutaway(ctx, tenantID, staffID, "warehouse", uc.PutawayRequest{
			WarehouseID:    whID,
			ProductID:      prodID,
			BatchID:        batchID,
			Quantity:       decimal.NewFromInt(5),
			DestLocationID: rackB,
			Reason:         &reason,
		})
		require.NoError(t, err)
		assert.Equal(t, rackB, mov.DestLocationID)
		assert.Equal(t, stgLoc.ID, mov.SourceLocationID)
		assert.True(t, mov.Quantity.Equal(decimal.NewFromInt(5)))
	})

	// 5. Auditor is blocked from confirming putaway
	t.Run("Auditor role cannot confirm putaway", func(t *testing.T) {
		_, err := usecase.ConfirmPutaway(ctx, tenantID, auditorID, "auditor", uc.PutawayRequest{
			WarehouseID:    whID,
			ProductID:      prodID,
			BatchID:        batchID,
			Quantity:       decimal.NewFromInt(1),
			DestLocationID: rackA,
		})
		assert.ErrorIs(t, err, domain.ErrUnauthorized)
	})

	// 6. Release approval (PDF-06)
	t.Run("Release approval promotes on hold lots", func(t *testing.T) {
		rcID := uuid.New()
		rc := &domain.StockReceipt{
			ID:          rcID,
			TenantID:    tenantID,
			WarehouseID: whID,
			Status:      domain.StockReceiptStatusPosted,
		}
		repo.stockReceipts[rcID] = rc

		heldBatchID := uuid.New()
		repo.batches[heldBatchID] = &domain.StockBatch{
			ID:              heldBatchID,
			TenantID:        tenantID,
			ProductID:       prodID,
			BatchNumber:     "HELD-001",
			SourceReceiptID: &rcID,
			Status:          domain.StockBatchStatusOnHold,
		}

		// Staff is rejected
		_, err := usecase.ReleaseStockReceipt(ctx, tenantID, staffID, "warehouse", rcID)
		assert.ErrorIs(t, err, domain.ErrUnauthorized)

		// Admin succeeds
		released, err := usecase.ReleaseStockReceipt(ctx, tenantID, adminID, "admin", rcID)
		require.NoError(t, err)
		assert.NotNil(t, released.ReleasedBy)
		assert.Equal(t, domain.StockBatchStatusReleased, repo.batches[heldBatchID].Status)
	})
}
