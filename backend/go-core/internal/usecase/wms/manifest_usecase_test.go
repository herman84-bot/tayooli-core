package wms_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/wms"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock methods for WMSManifestRepository on mockWMSRepo

func (m *mockWMSRepo) CreateShippingManifest(ctx context.Context, tenantID, userID uuid.UUID, req domain.CreateShippingManifestRequest) (*domain.ShippingManifest, error) {
	now := time.Now().UTC()
	return &domain.ShippingManifest{
		ID:             uuid.New(),
		TenantID:       tenantID,
		WarehouseID:    req.WarehouseID,
		ManifestNumber: "SM-TEST-001",
		ExpeditionName: req.ExpeditionName,
		DriverName:     req.DriverName,
		VehiclePlate:   req.VehiclePlate,
		DriverPhone:    req.DriverPhone,
		TotalPackages:  len(req.DeliveryOrderIDs),
		TotalWeightKg:  decimal.NewFromFloat(12.5),
		Status:         domain.ShippingManifestStatusStaged,
		Notes:          req.Notes,
		CreatedBy:      &userID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (m *mockWMSRepo) GetShippingManifestByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.ShippingManifestDetail, error) {
	now := time.Now().UTC()
	return &domain.ShippingManifestDetail{
		Manifest: domain.ShippingManifest{
			ID:             id,
			TenantID:       tenantID,
			ManifestNumber: "SM-TEST-001",
			Status:         domain.ShippingManifestStatusStaged,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		Items: []domain.ShippingManifestItem{},
	}, nil
}

func (m *mockWMSRepo) ListShippingManifests(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID, status *domain.ShippingManifestStatus, expeditionName *string) ([]domain.ShippingManifest, error) {
	return []domain.ShippingManifest{
		{
			ID:             uuid.New(),
			TenantID:       tenantID,
			ManifestNumber: "SM-TEST-001",
			Status:         domain.ShippingManifestStatusStaged,
		},
	}, nil
}

func (m *mockWMSRepo) ScanDOLoading(ctx context.Context, tenantID, manifestID uuid.UUID, barcode string, userID uuid.UUID) (*domain.ShippingManifestDetail, error) {
	if barcode == "MISLOAD-123" {
		return nil, domain.ErrDOMisload
	}
	now := time.Now().UTC()
	return &domain.ShippingManifestDetail{
		Manifest: domain.ShippingManifest{
			ID:             manifestID,
			TenantID:       tenantID,
			Status:         domain.ShippingManifestStatusLoaded,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		Items: []domain.ShippingManifestItem{
			{
				DONumber: barcode,
				Scanned:  true,
			},
		},
	}, nil
}

func (m *mockWMSRepo) DispatchShippingManifest(ctx context.Context, tenantID, manifestID, userID uuid.UUID, req domain.DispatchShippingManifestRequest) (*domain.ShippingManifest, error) {
	now := time.Now().UTC()
	return &domain.ShippingManifest{
		ID:                 manifestID,
		TenantID:           tenantID,
		Status:             domain.ShippingManifestStatusDispatched,
		DriverSignatureSVG: &req.DriverSignatureSVG,
		DispatchedBy:       &userID,
		DispatchedAt:       &now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}, nil
}

func (m *mockWMSRepo) GetWMSOutboundKPIs(ctx context.Context, tenantID uuid.UUID, warehouseID *uuid.UUID) (*domain.WMSOutboundKPISummary, error) {
	return &domain.WMSOutboundKPISummary{
		DockToStockAvgMinutes:   45.2,
		ReceivingAccuracyPct:    99.5,
		POCompliancePct:         98.0,
		BacklogInboundCount:     2,
		OrderToDispatchAvgHours: 3.5,
		PickingAccuracyPct:      99.8,
		OnTimeShipmentPct:       97.5,
		BacklogOutboundCount:    5,
	}, nil
}

func TestManifestUsecase(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	adminID := uuid.New()
	auditorID := uuid.New()
	whID := uuid.New()

	repo := newMockWMSRepo()
	repo.warehouses[whID] = domain.Warehouse{
		ID:       whID,
		TenantID: tenantID,
		Code:     "WH-01",
		Name:     "Main Warehouse",
		IsActive: true,
	}

	usecase := uc.New(repo)

	t.Run("CreateShippingManifest validation and access", func(t *testing.T) {
		// Empty DOs
		_, err := usecase.CreateShippingManifest(ctx, tenantID, adminID, "admin", domain.CreateShippingManifestRequest{
			WarehouseID:    whID,
			ExpeditionName: "JNE",
			DriverName:     "Budi",
			VehiclePlate:   "B 1234 CD",
		})
		require.ErrorIs(t, err, domain.ErrManifestEmpty)

		// Auditor role forbidden
		_, err = usecase.CreateShippingManifest(ctx, tenantID, auditorID, "auditor", domain.CreateShippingManifestRequest{
			WarehouseID:      whID,
			ExpeditionName:   "JNE",
			DriverName:       "Budi",
			VehiclePlate:     "B 1234 CD",
			DeliveryOrderIDs: []uuid.UUID{uuid.New()},
		})
		require.ErrorIs(t, err, domain.ErrForbidden)

		// Success
		manifest, err := usecase.CreateShippingManifest(ctx, tenantID, adminID, "admin", domain.CreateShippingManifestRequest{
			WarehouseID:      whID,
			ExpeditionName:   "JNE",
			DriverName:       "Budi",
			VehiclePlate:     "B 1234 CD",
			DeliveryOrderIDs: []uuid.UUID{uuid.New()},
		})
		require.NoError(t, err)
		assert.Equal(t, "SM-TEST-001", manifest.ManifestNumber)
		assert.Equal(t, domain.ShippingManifestStatusStaged, manifest.Status)
	})

	t.Run("ScanDOLoading validation and dispatch", func(t *testing.T) {
		manifestID := uuid.New()

		// Empty barcode
		_, err := usecase.ScanDOLoading(ctx, tenantID, adminID, "admin", manifestID, domain.LoadingScanRequest{
			Barcode: "   ",
		})
		require.ErrorIs(t, err, domain.ErrInvalidInput)

		// Misload error from repo
		_, err = usecase.ScanDOLoading(ctx, tenantID, adminID, "admin", manifestID, domain.LoadingScanRequest{
			Barcode: "MISLOAD-123",
		})
		require.ErrorIs(t, err, domain.ErrDOMisload)

		// Success
		detail, err := usecase.ScanDOLoading(ctx, tenantID, adminID, "admin", manifestID, domain.LoadingScanRequest{
			Barcode: "DO-2026-001",
		})
		require.NoError(t, err)
		assert.Equal(t, domain.ShippingManifestStatusLoaded, detail.Manifest.Status)
	})

	t.Run("DispatchShippingManifest signature validation", func(t *testing.T) {
		manifestID := uuid.New()

		// Short/empty signature
		_, err := usecase.DispatchShippingManifest(ctx, tenantID, adminID, "admin", manifestID, domain.DispatchShippingManifestRequest{
			DriverSignatureSVG: "   ",
		})
		require.ErrorIs(t, err, domain.ErrManifestSignatureRequired)

		// Valid signature
		validSig := "<svg width='100' height='50'><path d='M10 10 L20 20'/></svg>"
		dispatched, err := usecase.DispatchShippingManifest(ctx, tenantID, adminID, "admin", manifestID, domain.DispatchShippingManifestRequest{
			DriverSignatureSVG: validSig,
		})
		require.NoError(t, err)
		assert.Equal(t, domain.ShippingManifestStatusDispatched, dispatched.Status)
	})

	t.Run("GetShippingManifest and ListShippingManifests", func(t *testing.T) {
		manifestID := uuid.New()
		detail, err := usecase.GetShippingManifest(ctx, tenantID, adminID, "admin", manifestID)
		require.NoError(t, err)
		assert.Equal(t, manifestID, detail.Manifest.ID)

		list, err := usecase.ListShippingManifests(ctx, tenantID, adminID, "admin", nil, nil, nil)
		require.NoError(t, err)
		assert.Len(t, list, 1)
	})

	t.Run("GetWMSOutboundKPIs", func(t *testing.T) {
		kpi, err := usecase.GetWMSOutboundKPIs(ctx, tenantID, adminID, "admin", &whID)
		require.NoError(t, err)
		assert.Equal(t, 99.8, kpi.PickingAccuracyPct)
		assert.Equal(t, 8, int(kpi.OrderToDispatchAvgHours)+kpi.BacklogOutboundCount)
	})
}
