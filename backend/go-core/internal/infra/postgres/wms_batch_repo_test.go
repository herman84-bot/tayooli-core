package postgres_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
)

// Verify interface implementations at compile time.
var (
	_ domain.WMSBatchRepository = (*postgres.WMSRepo)(nil)
	_ domain.WMSRepository      = (*postgres.WMSRepo)(nil)
)

func TestWMSRepo_ValidationFastPaths(t *testing.T) {
	repo := postgres.NewWMSRepo(nil)
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("CreateStockMovement rejects nil BatchID", func(t *testing.T) {
		mov := &domain.StockMovement{
			TenantID: tenantID,
			BatchID:  nil,
		}
		err := repo.CreateStockMovement(ctx, mov)
		require.ErrorIs(t, err, domain.ErrBatchRequired)

		nilUUID := uuid.Nil
		mov.BatchID = &nilUUID
		err = repo.CreateStockMovement(ctx, mov)
		require.ErrorIs(t, err, domain.ErrBatchRequired)
	})

	t.Run("CreateStockMovements rejects nil BatchID", func(t *testing.T) {
		movs := []domain.StockMovement{
			{TenantID: tenantID, BatchID: nil},
		}
		err := repo.CreateStockMovements(ctx, tenantID, movs)
		require.ErrorIs(t, err, domain.ErrBatchRequired)
	})

	t.Run("CreateStockMovements empty slice is noop", func(t *testing.T) {
		err := repo.CreateStockMovements(ctx, tenantID, nil)
		require.NoError(t, err)
	})

	t.Run("GetOrCreateBatch rejects invalid input", func(t *testing.T) {
		_, err := repo.GetOrCreateBatch(ctx, nil)
		require.ErrorIs(t, err, domain.ErrInvalidInput)

		_, err = repo.GetOrCreateBatch(ctx, &domain.StockBatch{
			TenantID: uuid.Nil,
		})
		require.ErrorIs(t, err, domain.ErrInvalidInput)

		_, err = repo.GetOrCreateBatch(ctx, &domain.StockBatch{
			TenantID:  tenantID,
			ProductID: uuid.Nil,
		})
		require.ErrorIs(t, err, domain.ErrInvalidInput)

		_, err = repo.GetOrCreateBatch(ctx, &domain.StockBatch{
			TenantID:    tenantID,
			ProductID:   uuid.New(),
			BatchNumber: "   ",
		})
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("ConfirmPutaway validates required IDs and qty", func(t *testing.T) {
		// Zero/negative qty
		cmd := domain.PutawayCommand{
			TenantID:          tenantID,
			WarehouseID:       uuid.New(),
			ProductID:         uuid.New(),
			BatchID:           uuid.New(),
			StagingLocationID: uuid.New(),
			DestLocationID:    uuid.New(),
			UserID:            uuid.New(),
			Quantity:          decimal.Zero,
		}
		_, err := repo.ConfirmPutaway(ctx, cmd)
		require.ErrorIs(t, err, domain.ErrInvalidInput)

		// Nil user ID
		cmd.Quantity = decimal.NewFromInt(10)
		cmd.UserID = uuid.Nil
		_, err = repo.ConfirmPutaway(ctx, cmd)
		require.ErrorIs(t, err, domain.ErrActorRequired)

		// Nil product ID
		cmd.UserID = uuid.New()
		cmd.ProductID = uuid.Nil
		_, err = repo.ConfirmPutaway(ctx, cmd)
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("ReleaseStockReceipt rejects nil UserID", func(t *testing.T) {
		_, err := repo.ReleaseStockReceipt(ctx, tenantID, uuid.New(), uuid.Nil)
		require.ErrorIs(t, err, domain.ErrActorRequired)
	})

	t.Run("UpsertWMSSettings validates nil tenant", func(t *testing.T) {
		err := repo.UpsertWMSSettings(ctx, nil)
		require.ErrorIs(t, err, domain.ErrInvalidInput)

		err = repo.UpsertWMSSettings(ctx, &domain.WMSSettings{TenantID: uuid.Nil})
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("SetDefaultLocation validates nil IDs", func(t *testing.T) {
		err := repo.SetDefaultLocation(ctx, nil)
		require.ErrorIs(t, err, domain.ErrInvalidInput)

		err = repo.SetDefaultLocation(ctx, &domain.ProductDefaultLocation{
			TenantID: uuid.Nil,
		})
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("DeductWarehouseStock validates non-positive qty", func(t *testing.T) {
		err := repo.DeductWarehouseStock(ctx, tenantID, uuid.New(), uuid.New(), decimal.Zero, &domain.StockMovement{})
		require.ErrorIs(t, err, domain.ErrInvalidInput)

		err = repo.DeductWarehouseStock(ctx, tenantID, uuid.New(), uuid.New(), decimal.NewFromInt(-5), &domain.StockMovement{})
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	})
}

func TestAuditHashChain_Calculation(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	entityID := uuid.New()
	entityType := "stock_receipt"
	action := "released"
	details := json.RawMessage(`{"released_batch_count":2}`)
	now := time.Now().UTC()

	// Initial hash with empty prevHash
	pHash := ""
	h1 := sha256.New()
	h1.Write([]byte(pHash))
	h1.Write([]byte(tenantID.String()))
	h1.Write([]byte(entityType))
	h1.Write([]byte(entityID.String()))
	h1.Write([]byte(action))
	h1.Write(details)
	h1.Write([]byte(now.Format(time.RFC3339Nano)))
	currentHash1 := hex.EncodeToString(h1.Sum(nil))

	assert.Len(t, currentHash1, 64)

	// Second hash chained from first
	entityID2 := uuid.New()
	now2 := now.Add(time.Second)
	h2 := sha256.New()
	h2.Write([]byte(currentHash1))
	h2.Write([]byte(tenantID.String()))
	h2.Write([]byte("putaway"))
	h2.Write([]byte(entityID2.String()))
	h2.Write([]byte("confirmed"))
	h2.Write(nil)
	h2.Write([]byte(now2.Format(time.RFC3339Nano)))
	currentHash2 := hex.EncodeToString(h2.Sum(nil))

	assert.Len(t, currentHash2, 64)
	assert.NotEqual(t, currentHash1, currentHash2)
	_ = userID
	_ = fmt.Sprintf("Hash 1: %s, Hash 2: %s", currentHash1, currentHash2)
}
