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

func TestOutboundQCUsecase(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	packerID := uuid.New()
	qcInspectorID := uuid.New()
	warehouseID := uuid.New()
	doID := uuid.New()

	repo := newMockWMSRepo()
	repo.initReceipts()

	// Setup warehouse
	repo.warehouses[warehouseID] = domain.Warehouse{
		ID:       warehouseID,
		TenantID: tenantID,
		Name:     "Gudang Utama",
		Code:     "WH-MAIN",
		IsActive: true,
	}
	repo.userWarehouses[tenantID.String()+":"+packerID.String()] = []uuid.UUID{warehouseID}
	repo.userWarehouses[tenantID.String()+":"+qcInspectorID.String()] = []uuid.UUID{warehouseID}

	// Setup Delivery Order packed by packerID
	now := time.Now().UTC()
	packedStatus := domain.DeliveryOrderStatusPacked
	trackingNum := "AWB-TEST-888"
	repo.deliveryOrders[doID] = &domain.DeliveryOrder{
		ID:             doID,
		TenantID:       tenantID,
		WarehouseID:    warehouseID,
		DONumber:       "DO-20261101-0001",
		TrackingNumber: &trackingNum,
		Status:         packedStatus,
		PackedBy:       &packerID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	repo.doItems[doID] = []domain.DeliveryOrderItem{
		{
			ID:              uuid.New(),
			TenantID:        tenantID,
			DeliveryOrderID: doID,
			ProductID:       uuid.New(),
			Quantity:        decimal.NewFromInt(5),
			LocationID:      uuid.New(),
			PackedQty:       decimal.NewFromInt(5),
		},
	}

	usecase := uc.New(repo)

	t.Run("Segregation of duties: packer cannot QC their own packed order", func(t *testing.T) {
		req := domain.PassOutboundQCRequest{
			ScannedAWB: "AWB-TEST-888",
		}
		_, err := usecase.PassDeliveryOrderQC(ctx, tenantID, packerID, "warehouse", doID, req)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrSelfApprovalForbidden, "packer cannot approve their own packed order")

		rejectReq := domain.RejectOutboundQCRequest{
			Reason: "Kemasan rusak",
		}
		_, err = usecase.RejectDeliveryOrderQC(ctx, tenantID, packerID, "warehouse", doID, rejectReq)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrSelfApprovalForbidden, "packer cannot reject their own packed order")
	})

	t.Run("Reject QC requires reason", func(t *testing.T) {
		rejectReq := domain.RejectOutboundQCRequest{
			Reason: "   ",
		}
		_, err := usecase.RejectDeliveryOrderQC(ctx, tenantID, qcInspectorID, "warehouse", doID, rejectReq)
		require.ErrorIs(t, err, domain.ErrQCRejectionReasonRequired)
	})

	t.Run("Different QC officer can pass order", func(t *testing.T) {
		req := domain.PassOutboundQCRequest{
			ScannedAWB: "AWB-TEST-888",
		}
		res, err := usecase.PassDeliveryOrderQC(ctx, tenantID, qcInspectorID, "warehouse", doID, req)
		require.NoError(t, err)
		assert.Equal(t, domain.DeliveryOrderStatusQCPassed, res.Status)
	})

	t.Run("Different QC officer can reject order", func(t *testing.T) {
		rejectReq := domain.RejectOutboundQCRequest{
			Reason: "AWB rusak atau tidak terbaca",
		}
		res, err := usecase.RejectDeliveryOrderQC(ctx, tenantID, qcInspectorID, "warehouse", doID, rejectReq)
		require.NoError(t, err)
		assert.Equal(t, domain.DeliveryOrderStatusQCFailed, res.Status)
	})
}
