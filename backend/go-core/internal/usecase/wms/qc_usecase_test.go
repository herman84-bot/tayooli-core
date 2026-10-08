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

// --- mock QC repo methods (record params; real SQL is covered by the PG15 integration test) ---

var mockQCState = struct {
	submits []domain.SubmitQCParams
	moves   []domain.QuarantineMoveParams
}{}

func (m *mockWMSRepo) GetOrCreateQuarantineLocation(ctx context.Context, tenantID, warehouseID uuid.UUID) (*domain.WarehouseLocation, error) {
	for _, l := range m.locations {
		if l.TenantID == tenantID && l.WarehouseID != nil && *l.WarehouseID == warehouseID && l.Type == domain.LocationTypeQuarantine {
			cp := l
			return &cp, nil
		}
	}
	wh := warehouseID
	l := domain.WarehouseLocation{ID: uuid.New(), TenantID: tenantID, WarehouseID: &wh, Code: domain.QuarantineCode, Type: domain.LocationTypeQuarantine}
	m.locations[l.ID] = l
	return &l, nil
}

func (m *mockWMSRepo) SubmitQCInspection(ctx context.Context, p domain.SubmitQCParams) (*domain.QCInspectionDetail, error) {
	mockQCState.submits = append(mockQCState.submits, p)
	return &domain.QCInspectionDetail{Inspection: domain.QCInspection{ReceiptID: p.ReceiptID, InspectionMode: p.Input.InspectionMode}}, nil
}

func (m *mockWMSRepo) GetQCInspectionByReceipt(ctx context.Context, tenantID, receiptID uuid.UUID) (*domain.QCInspectionDetail, error) {
	return nil, nil
}

func (m *mockWMSRepo) ListQCInspections(ctx context.Context, tenantID, warehouseID uuid.UUID) ([]domain.QCInspection, error) {
	return nil, nil
}

func (m *mockWMSRepo) MoveQuarantineStock(ctx context.Context, p domain.QuarantineMoveParams) (*domain.StockMovement, error) {
	mockQCState.moves = append(mockQCState.moves, p)
	return &domain.StockMovement{ID: uuid.New(), SourceLocationID: p.SourceLocID, DestLocationID: p.DestLocID, Quantity: p.Input.Quantity}, nil
}

func TestQCUsecase(t *testing.T) {
	ctx := context.Background()
	tenantID, adminID, outsiderID := uuid.New(), uuid.New(), uuid.New()
	repo := newMockWMSRepo()
	repo.initReceipts()
	u := uc.New(repo)

	whID := uuid.New()
	repo.warehouses[whID] = domain.Warehouse{ID: whID, TenantID: tenantID, Name: "Gudang Utama"}
	repo.userWarehouses[fmt.Sprintf("%s:%s", tenantID, outsiderID)] = []uuid.UUID{}

	posted := &domain.StockReceipt{ID: uuid.New(), TenantID: tenantID, WarehouseID: whID, ReceiptNumber: "GR-1", Status: domain.StockReceiptStatusPosted, CreatedAt: time.Now()}
	draft := &domain.StockReceipt{ID: uuid.New(), TenantID: tenantID, WarehouseID: whID, ReceiptNumber: "GR-2", Status: domain.StockReceiptStatusDraft, CreatedAt: time.Now()}
	repo.stockReceipts[posted.ID] = posted
	repo.stockReceipts[draft.ID] = draft

	in := domain.QCInspectionInput{InspectionMode: " full ", Items: []domain.QCLineInput{{BatchID: uuid.New(), CheckedQty: decimal.NewFromInt(1)}}}

	t.Run("submit passes staging + quarantine bins and normalizes mode", func(t *testing.T) {
		mockQCState.submits = nil
		_, err := u.SubmitQCInspection(ctx, tenantID, adminID, "admin", posted.ID, in)
		require.NoError(t, err)
		require.Len(t, mockQCState.submits, 1)
		p := mockQCState.submits[0]
		assert.Equal(t, domain.QCModeFull, p.Input.InspectionMode)
		stg, _ := repo.GetOrCreateStagingLocation(ctx, tenantID, whID)
		qrn, _ := repo.GetOrCreateQuarantineLocation(ctx, tenantID, whID)
		assert.Equal(t, stg.ID, p.StagingLocID)
		assert.Equal(t, qrn.ID, p.QuarantineLoc)
		assert.NotEqual(t, p.StagingLocID, p.QuarantineLoc)
	})
	t.Run("draft receipt rejected", func(t *testing.T) {
		_, err := u.SubmitQCInspection(ctx, tenantID, adminID, "admin", draft.ID, in)
		assert.ErrorIs(t, err, domain.ErrQCReceiptNotPosted)
	})
	t.Run("missing actor rejected", func(t *testing.T) {
		_, err := u.SubmitQCInspection(ctx, tenantID, uuid.Nil, "admin", posted.ID, in)
		assert.ErrorIs(t, err, domain.ErrActorRequired)
	})
	t.Run("staff without warehouse access rejected", func(t *testing.T) {
		_, err := u.SubmitQCInspection(ctx, tenantID, outsiderID, "staff", posted.ID, in)
		assert.Error(t, err)
	})

	act := domain.QuarantineActionInput{WarehouseID: whID, ProductID: uuid.New(), BatchID: uuid.New(), Quantity: decimal.NewFromInt(2)}
	t.Run("release goes QRN -> STG-IN", func(t *testing.T) {
		mockQCState.moves = nil
		_, err := u.ReleaseQuarantine(ctx, tenantID, adminID, "admin", act)
		require.NoError(t, err)
		stg, _ := repo.GetOrCreateStagingLocation(ctx, tenantID, whID)
		qrn, _ := repo.GetOrCreateQuarantineLocation(ctx, tenantID, whID)
		require.Len(t, mockQCState.moves, 1)
		assert.Equal(t, qrn.ID, mockQCState.moves[0].SourceLocID)
		assert.Equal(t, stg.ID, mockQCState.moves[0].DestLocID)
		assert.Equal(t, "released", mockQCState.moves[0].Action)
	})
	t.Run("scrap requires notes and goes to @SCRAP", func(t *testing.T) {
		mockQCState.moves = nil
		_, err := u.ScrapQuarantine(ctx, tenantID, adminID, "admin", act)
		assert.ErrorIs(t, err, domain.ErrScrapNotesRequired)
		notes := "  "
		act2 := act
		act2.Notes = &notes
		_, err = u.ScrapQuarantine(ctx, tenantID, adminID, "admin", act2)
		assert.ErrorIs(t, err, domain.ErrScrapNotesRequired)
		notes = "kemasan pecah"
		_, err = u.ScrapQuarantine(ctx, tenantID, adminID, "admin", act2)
		require.NoError(t, err)
		scrap, _ := repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeScrap)
		assert.Equal(t, scrap.ID, mockQCState.moves[0].DestLocID)
	})
	t.Run("zero/negative qty rejected", func(t *testing.T) {
		bad := act
		bad.Quantity = decimal.Zero
		_, err := u.ReleaseQuarantine(ctx, tenantID, adminID, "admin", bad)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		bad.Quantity = decimal.NewFromInt(-1)
		_, err = u.ReleaseQuarantine(ctx, tenantID, adminID, "admin", bad)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})
}
