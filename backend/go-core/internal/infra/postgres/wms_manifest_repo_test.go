package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
	"github.com/stretchr/testify/require"
)

// Verify interface implementations at compile time.
var (
	_ domain.WMSManifestRepository = (*postgres.WMSRepo)(nil)
)

func TestWMSRepo_ManifestValidationFastPaths(t *testing.T) {
	repo := postgres.NewWMSRepo(nil)
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("CreateShippingManifest rejects empty DOs", func(t *testing.T) {
		req := domain.CreateShippingManifestRequest{
			WarehouseID:      uuid.New(),
			ExpeditionName:   "JNE",
			DriverName:       "Budi",
			VehiclePlate:     "B 1234 CD",
			DeliveryOrderIDs: nil,
		}
		_, err := repo.CreateShippingManifest(ctx, tenantID, userID, req)
		require.ErrorIs(t, err, domain.ErrManifestEmpty)
	})

	t.Run("CreateShippingManifest rejects missing warehouse", func(t *testing.T) {
		req := domain.CreateShippingManifestRequest{
			WarehouseID:      uuid.Nil,
			ExpeditionName:   "JNE",
			DriverName:       "Budi",
			VehiclePlate:     "B 1234 CD",
			DeliveryOrderIDs: []uuid.UUID{uuid.New()},
		}
		_, err := repo.CreateShippingManifest(ctx, tenantID, userID, req)
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("ScanDOLoading rejects blank barcode", func(t *testing.T) {
		manifestID := uuid.New()
		_, err := repo.ScanDOLoading(ctx, tenantID, manifestID, "   ", userID)
		require.ErrorIs(t, err, domain.ErrInvalidInput)
	})

	t.Run("DispatchShippingManifest rejects missing driver signature", func(t *testing.T) {
		manifestID := uuid.New()
		req := domain.DispatchShippingManifestRequest{
			DriverSignatureSVG: "",
		}
		_, err := repo.DispatchShippingManifest(ctx, tenantID, manifestID, userID, req)
		require.ErrorIs(t, err, domain.ErrManifestSignatureRequired)
	})

	t.Run("DispatchShippingManifest rejects short signature", func(t *testing.T) {
		manifestID := uuid.New()
		req := domain.DispatchShippingManifestRequest{
			DriverSignatureSVG: "abc",
		}
		_, err := repo.DispatchShippingManifest(ctx, tenantID, manifestID, userID, req)
		require.ErrorIs(t, err, domain.ErrManifestSignatureRequired)
	})
}
