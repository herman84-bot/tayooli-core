package wms

// Sprint 2 QC & quarantine use cases (ADR-014 Invariant 3; sentry-wms 1.1 quarantine
// bin not pickable; OCA quality step between receiving and storage).

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// SubmitQCInspection inspects a posted receipt. Damaged qty moves STG-IN -> QRN in the same tx.
func (u *Usecase) SubmitQCInspection(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID, in domain.QCInspectionInput) (*domain.QCInspectionDetail, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrActorRequired
	}
	rc, _, err := u.repo.GetStockReceiptByID(ctx, tenantID, receiptID)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, rc.WarehouseID); err != nil {
		return nil, err
	}
	if rc.Status != domain.StockReceiptStatusPosted {
		return nil, domain.ErrQCReceiptNotPosted
	}
	in.InspectionMode = domain.QCInspectionMode(strings.ToUpper(strings.TrimSpace(string(in.InspectionMode))))
	stg, err := u.repo.GetOrCreateStagingLocation(ctx, tenantID, rc.WarehouseID)
	if err != nil {
		return nil, fmt.Errorf("SubmitQCInspection: staging: %w", err)
	}
	qrn, err := u.repo.GetOrCreateQuarantineLocation(ctx, tenantID, rc.WarehouseID)
	if err != nil {
		return nil, fmt.Errorf("SubmitQCInspection: quarantine: %w", err)
	}
	return u.repo.SubmitQCInspection(ctx, domain.SubmitQCParams{
		TenantID: tenantID, ReceiptID: receiptID, UserID: userID,
		StagingLocID: stg.ID, QuarantineLoc: qrn.ID, Input: in,
	})
}

func (u *Usecase) GetReceiptQC(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID) (*domain.QCInspectionDetail, error) {
	rc, _, err := u.repo.GetStockReceiptByID(ctx, tenantID, receiptID)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, rc.WarehouseID); err != nil {
		return nil, err
	}
	return u.repo.GetQCInspectionByReceipt(ctx, tenantID, receiptID)
}

func (u *Usecase) ListQCInspections(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID uuid.UUID) ([]domain.QCInspection, error) {
	if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, warehouseID); err != nil {
		return nil, err
	}
	return u.repo.ListQCInspections(ctx, tenantID, warehouseID)
}

// ListQuarantineStock lists batch balances sitting in the warehouse quarantine bin.
func (u *Usecase) ListQuarantineStock(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID uuid.UUID) ([]domain.BatchBalance, error) {
	if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, warehouseID); err != nil {
		return nil, err
	}
	t := domain.LocationTypeQuarantine
	return u.repo.ListBatchBalances(ctx, tenantID, domain.BatchBalanceFilter{WarehouseID: &warehouseID, LocationType: &t})
}

// ReleaseQuarantine returns stock to inbound staging so it goes through putaway again.
func (u *Usecase) ReleaseQuarantine(ctx context.Context, tenantID, userID uuid.UUID, role string, in domain.QuarantineActionInput) (*domain.StockMovement, error) {
	return u.moveQuarantine(ctx, tenantID, userID, role, in, "released")
}

// ScrapQuarantine writes quarantined stock off to @SCRAP (notes required).
func (u *Usecase) ScrapQuarantine(ctx context.Context, tenantID, userID uuid.UUID, role string, in domain.QuarantineActionInput) (*domain.StockMovement, error) {
	if in.Notes == nil || strings.TrimSpace(*in.Notes) == "" {
		return nil, domain.ErrScrapNotesRequired
	}
	return u.moveQuarantine(ctx, tenantID, userID, role, in, "scrapped")
}

func (u *Usecase) moveQuarantine(ctx context.Context, tenantID, userID uuid.UUID, role string, in domain.QuarantineActionInput, action string) (*domain.StockMovement, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrActorRequired
	}
	if in.WarehouseID == uuid.Nil || in.ProductID == uuid.Nil || in.BatchID == uuid.Nil || !in.Quantity.IsPositive() {
		return nil, domain.ErrInvalidInput
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, in.WarehouseID); err != nil {
		return nil, err
	}
	qrn, err := u.repo.GetOrCreateQuarantineLocation(ctx, tenantID, in.WarehouseID)
	if err != nil {
		return nil, fmt.Errorf("moveQuarantine: quarantine: %w", err)
	}
	var dest uuid.UUID
	if action == "scrapped" {
		loc, err := u.repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeScrap)
		if err != nil {
			return nil, fmt.Errorf("moveQuarantine: scrap: %w", err)
		}
		dest = loc.ID
	} else {
		loc, err := u.repo.GetOrCreateStagingLocation(ctx, tenantID, in.WarehouseID)
		if err != nil {
			return nil, fmt.Errorf("moveQuarantine: staging: %w", err)
		}
		dest = loc.ID
	}
	return u.repo.MoveQuarantineStock(ctx, domain.QuarantineMoveParams{
		TenantID: tenantID, UserID: userID, Action: action, Input: in,
		SourceLocID: qrn.ID, DestLocID: dest,
	})
}
