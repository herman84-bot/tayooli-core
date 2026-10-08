package domain

// Sprint 2: inbound QC, quarantine and Berita Acara Kerusakan (BAK).
// Patterns: OCA/wms quality step between receiving and storage; sentry-wms 1.1
// non-pickable quarantine bin; ADR-014 Invariant 3 (quarantine requires QC + BAK
// signed by the driver; inbound never goes straight to SCRAP).

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrQCAlreadyInspected     = errors.New("receipt already has a QC inspection")
	ErrQCReceiptNotPosted     = errors.New("only posted receipts can be inspected")
	ErrQCSamplingFailed       = errors.New("sampling found damaged goods; repeat with FULL inspection")
	ErrQCBAKDriverRequired    = errors.New("damaged goods require driver name and driver signature on the BAK")
	ErrQCStagedQtyChanged     = errors.New("staged quantity no longer covers the damaged quantity")
	ErrQuarantineQtyInvalid   = errors.New("quantity exceeds quarantined stock")
	ErrScrapNotesRequired     = errors.New("notes are required to scrap quarantined stock")
	ErrQuarantineStockBlocked = errors.New("quarantined stock cannot be picked, sold or transferred")
)

type QCInspectionMode string

const (
	QCModeFull     QCInspectionMode = "FULL"
	QCModeSampling QCInspectionMode = "SAMPLING"
)

type QCInspectionStatus string

const (
	QCStatusPassed      QCInspectionStatus = "QC_PASSED"
	QCStatusQuarantined QCInspectionStatus = "QUARANTINED"
	QCStatusRejected    QCInspectionStatus = "QC_REJECTED"
)

// QuarantineCode is the per-warehouse quarantine bin code.
const QuarantineCode = "QRN"

// Ledger reference type for QC/quarantine movements.
const StockRefQC = "QC_INSPECTION"

type QCInspection struct {
	ID              uuid.UUID          `json:"id"`
	TenantID        uuid.UUID          `json:"tenant_id"`
	ReceiptID       uuid.UUID          `json:"receipt_id"`
	ReceiptNumber   string             `json:"receipt_number"`
	SupplierName    string             `json:"supplier_name"`
	WarehouseID     uuid.UUID          `json:"warehouse_id"`
	InspectionMode  QCInspectionMode   `json:"inspection_mode"`
	SampleQty       *decimal.Decimal   `json:"sample_qty,omitempty"`
	GrossCartons    int                `json:"gross_cartons"`
	Status          QCInspectionStatus `json:"status"`
	TotalCheckedQty decimal.Decimal    `json:"total_checked_qty"`
	TotalPassedQty  decimal.Decimal    `json:"total_passed_qty"`
	TotalDamagedQty decimal.Decimal    `json:"total_damaged_qty"`
	ShortageQty     decimal.Decimal    `json:"shortage_qty"`
	OverageQty      decimal.Decimal    `json:"overage_qty"`
	BAKNumber       *string            `json:"bak_number,omitempty"`
	BAKNotes        *string            `json:"bak_notes,omitempty"`
	DriverName      *string            `json:"driver_name,omitempty"`
	DriverSigned    bool               `json:"driver_signed"`
	Notes           *string            `json:"notes,omitempty"`
	InspectorID     uuid.UUID          `json:"inspector_id"`
	InspectorName   string             `json:"inspector_name"`
	CreatedAt       time.Time          `json:"created_at"`
}

type QCInspectionItem struct {
	ID           uuid.UUID       `json:"id"`
	ProductID    uuid.UUID       `json:"product_id"`
	ProductName  string          `json:"product_name"`
	ProductSKU   string          `json:"product_sku"`
	BatchID      uuid.UUID       `json:"batch_id"`
	BatchNumber  string          `json:"batch_number"`
	ExpiryDate   *time.Time      `json:"expiry_date,omitempty"`
	StagedQty    decimal.Decimal `json:"staged_qty"`
	CheckedQty   decimal.Decimal `json:"checked_qty"`
	PassedQty    decimal.Decimal `json:"passed_qty"`
	DamagedQty   decimal.Decimal `json:"damaged_qty"`
	ShortageQty  decimal.Decimal `json:"shortage_qty"`
	OverageQty   decimal.Decimal `json:"overage_qty"`
	DamageReason *string         `json:"damage_reason,omitempty"`
}

type QCInspectionDetail struct {
	Inspection QCInspection       `json:"inspection"`
	Items      []QCInspectionItem `json:"items"`
}

// QCLineInput is one inspected batch line (blind count: checked_qty is what the
// inspector physically counted; expected comes from the receipt).
type QCLineInput struct {
	BatchID      uuid.UUID       `json:"batch_id"`
	CheckedQty   decimal.Decimal `json:"checked_qty"`
	DamagedQty   decimal.Decimal `json:"damaged_qty"`
	DamageReason *string         `json:"damage_reason,omitempty"`
}

type QCInspectionInput struct {
	InspectionMode QCInspectionMode `json:"inspection_mode"`
	SampleQty      *decimal.Decimal `json:"sample_qty,omitempty"`
	GrossCartons   int              `json:"gross_cartons"`
	DriverName     *string          `json:"driver_name,omitempty"`
	DriverSigned   bool             `json:"driver_signed"`
	BAKNotes       *string          `json:"bak_notes,omitempty"`
	Notes          *string          `json:"notes,omitempty"`
	Items          []QCLineInput    `json:"items"`
}

// QCExpectedLine is a receipt batch line as booked into staging at post time.
type QCExpectedLine struct {
	ProductID   uuid.UUID
	BatchID     uuid.UUID
	ExpectedQty decimal.Decimal // receipt accepted_qty
}

// QCComputed is the validated, derived result of an inspection.
type QCComputed struct {
	Status   QCInspectionStatus
	Items    []QCInspectionItem
	Checked  decimal.Decimal
	Passed   decimal.Decimal
	Damaged  decimal.Decimal
	Shortage decimal.Decimal
	Overage  decimal.Decimal
}

func trimPtr(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

// EvaluateQC validates an inspection against the expected receipt lines and
// derives per-line discrepancies and the overall status. Pure, no I/O.
//
// Rules:
//   - every expected batch line must be inspected exactly once; unknown batches rejected
//   - quantities >= 0, damaged <= checked, damaged needs a reason
//   - SAMPLING needs sample_qty > 0; a failed sample (any damage) forces FULL (PDF-02)
//   - damaged goods need a BAK: driver name + driver signature (Invariant 3)
//   - shortage = max(expected - checked, 0), overage = max(checked - expected, 0)
//     (discrepancy report only; ledger moves only the damaged qty out of staging)
//   - damaged cannot exceed expected (the staged stock); otherwise nothing to quarantine
//   - status: no damage = QC_PASSED; all expected damaged = QC_REJECTED; else QUARANTINED
func EvaluateQC(in QCInspectionInput, expected []QCExpectedLine) (*QCComputed, error) {
	if in.InspectionMode != QCModeFull && in.InspectionMode != QCModeSampling {
		return nil, ErrInvalidInput
	}
	if in.GrossCartons < 0 || len(in.Items) == 0 || len(expected) == 0 {
		return nil, ErrInvalidInput
	}
	if in.InspectionMode == QCModeSampling && (in.SampleQty == nil || !in.SampleQty.IsPositive()) {
		return nil, ErrInvalidInput
	}
	exp := make(map[uuid.UUID]QCExpectedLine, len(expected))
	for _, e := range expected {
		exp[e.BatchID] = e
	}
	seen := make(map[uuid.UUID]bool, len(in.Items))
	out := &QCComputed{}
	expectedTotal := decimal.Zero
	for _, line := range in.Items {
		e, ok := exp[line.BatchID]
		if !ok || seen[line.BatchID] {
			return nil, ErrInvalidInput
		}
		seen[line.BatchID] = true
		if line.CheckedQty.IsNegative() || line.DamagedQty.IsNegative() || line.DamagedQty.GreaterThan(line.CheckedQty) {
			return nil, ErrInvalidInput
		}
		if line.DamagedQty.IsPositive() && trimPtr(line.DamageReason) == "" {
			return nil, ErrInvalidInput
		}
		if line.DamagedQty.GreaterThan(e.ExpectedQty) {
			return nil, ErrQCStagedQtyChanged
		}
		item := QCInspectionItem{
			ProductID:  e.ProductID,
			BatchID:    e.BatchID,
			StagedQty:  e.ExpectedQty,
			CheckedQty: line.CheckedQty,
			DamagedQty: line.DamagedQty,
			PassedQty:  line.CheckedQty.Sub(line.DamagedQty),
		}
		if diff := e.ExpectedQty.Sub(line.CheckedQty); diff.IsPositive() {
			item.ShortageQty = diff
		} else {
			item.OverageQty = diff.Neg()
		}
		if line.DamagedQty.IsPositive() {
			r := trimPtr(line.DamageReason)
			item.DamageReason = &r
		}
		out.Items = append(out.Items, item)
		out.Checked = out.Checked.Add(item.CheckedQty)
		out.Passed = out.Passed.Add(item.PassedQty)
		out.Damaged = out.Damaged.Add(item.DamagedQty)
		out.Shortage = out.Shortage.Add(item.ShortageQty)
		out.Overage = out.Overage.Add(item.OverageQty)
		expectedTotal = expectedTotal.Add(e.ExpectedQty)
	}
	if len(seen) != len(exp) {
		return nil, ErrInvalidInput
	}
	if out.Damaged.IsPositive() {
		if in.InspectionMode == QCModeSampling {
			return nil, ErrQCSamplingFailed
		}
		if trimPtr(in.DriverName) == "" || !in.DriverSigned {
			return nil, ErrQCBAKDriverRequired
		}
	}
	switch {
	case !out.Damaged.IsPositive():
		out.Status = QCStatusPassed
	case out.Damaged.Equal(expectedTotal):
		out.Status = QCStatusRejected
	default:
		out.Status = QCStatusQuarantined
	}
	return out, nil
}

// BAKNumber formats the BAK registration number from the receipt number.
func BAKNumber(receiptNumber string) string {
	return "BAK-" + receiptNumber
}

// SubmitQCParams carries everything the repo needs for the atomic QC tx. The repo
// locks the receipt, loads the expected lines and runs EvaluateQC inside the tx.
type SubmitQCParams struct {
	TenantID      uuid.UUID
	ReceiptID     uuid.UUID
	UserID        uuid.UUID
	StagingLocID  uuid.UUID
	QuarantineLoc uuid.UUID
	Input         QCInspectionInput
}

// QuarantineActionInput moves stock out of quarantine (release back to staging, or scrap).
type QuarantineActionInput struct {
	WarehouseID uuid.UUID       `json:"warehouse_id"`
	ProductID   uuid.UUID       `json:"product_id"`
	BatchID     uuid.UUID       `json:"batch_id"`
	Quantity    decimal.Decimal `json:"quantity"`
	Notes       *string         `json:"notes,omitempty"`
}

type QuarantineMoveParams struct {
	TenantID    uuid.UUID
	UserID      uuid.UUID
	Action      string // "released" | "scrapped"
	Input       QuarantineActionInput
	SourceLocID uuid.UUID // quarantine bin
	DestLocID   uuid.UUID // STG-IN or @SCRAP
}

type WMSQCRepository interface {
	GetOrCreateQuarantineLocation(ctx context.Context, tenantID, warehouseID uuid.UUID) (*WarehouseLocation, error)
	SubmitQCInspection(ctx context.Context, p SubmitQCParams) (*QCInspectionDetail, error)
	GetQCInspectionByReceipt(ctx context.Context, tenantID, receiptID uuid.UUID) (*QCInspectionDetail, error)
	ListQCInspections(ctx context.Context, tenantID, warehouseID uuid.UUID) ([]QCInspection, error)
	MoveQuarantineStock(ctx context.Context, p QuarantineMoveParams) (*StockMovement, error)
}
