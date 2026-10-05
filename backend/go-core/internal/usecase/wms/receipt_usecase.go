package wms

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/shopspring/decimal"
)

// -----------------------------------------------------------------------------
// Stock Receipts (Barang Masuk / inbound goods receipt)
// -----------------------------------------------------------------------------

type StockReceiptItemRequest struct {
	ProductID    uuid.UUID        `json:"product_id"`
	ExpectedQty  *decimal.Decimal `json:"expected_qty,omitempty"`
	AcceptedQty  decimal.Decimal  `json:"accepted_qty"`
	RejectedQty  decimal.Decimal  `json:"rejected_qty"`
	RejectReason *string          `json:"reject_reason,omitempty"`
}

type StockReceiptRequest struct {
	WarehouseID    uuid.UUID                 `json:"warehouse_id"`
	DestLocationID uuid.UUID                 `json:"dest_location_id"`
	SupplierName   string                    `json:"supplier_name"`
	SupplierRef    *string                   `json:"supplier_ref,omitempty"`
	Notes          *string                   `json:"notes,omitempty"`
	Items          []StockReceiptItemRequest `json:"items"`
}

func receiptInvalid(msg string) error {
	return &domain.StockReceiptValidationError{Msg: msg}
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

// validateStockReceiptRequest checks the payload, write access and the destination location.
func (u *Usecase) validateStockReceiptRequest(ctx context.Context, tenantID, userID uuid.UUID, role string, req StockReceiptRequest) ([]domain.StockReceiptItem, error) {
	if req.WarehouseID == uuid.Nil {
		return nil, receiptInvalid("Gudang wajib dipilih")
	}
	if req.DestLocationID == uuid.Nil {
		return nil, receiptInvalid("Lokasi tujuan wajib dipilih")
	}
	if strings.TrimSpace(req.SupplierName) == "" {
		return nil, receiptInvalid("Nama pemasok wajib diisi")
	}
	if len(req.Items) == 0 {
		return nil, receiptInvalid("Penerimaan harus memiliki minimal satu barang")
	}

	seen := make(map[uuid.UUID]bool, len(req.Items))
	items := make([]domain.StockReceiptItem, 0, len(req.Items))
	for i, it := range req.Items {
		line := i + 1
		if it.ProductID == uuid.Nil {
			return nil, receiptInvalid(fmt.Sprintf("Baris %d: produk wajib dipilih", line))
		}
		if seen[it.ProductID] {
			return nil, receiptInvalid(fmt.Sprintf("Baris %d: produk yang sama tidak boleh diinput dua kali", line))
		}
		seen[it.ProductID] = true
		if it.ExpectedQty != nil && it.ExpectedQty.IsNegative() {
			return nil, receiptInvalid(fmt.Sprintf("Baris %d: jumlah dipesan tidak boleh negatif", line))
		}
		if it.AcceptedQty.IsNegative() || it.RejectedQty.IsNegative() {
			return nil, receiptInvalid(fmt.Sprintf("Baris %d: jumlah diterima/ditolak tidak boleh negatif", line))
		}
		if !it.AcceptedQty.Add(it.RejectedQty).IsPositive() {
			return nil, receiptInvalid(fmt.Sprintf("Baris %d: jumlah diterima atau ditolak harus lebih dari 0", line))
		}
		reason := trimPtr(it.RejectReason)
		if it.RejectedQty.IsPositive() && reason == nil {
			return nil, receiptInvalid(fmt.Sprintf("Baris %d: alasan penolakan wajib diisi jika ada barang ditolak", line))
		}
		if !it.RejectedQty.IsPositive() {
			reason = nil
		}
		items = append(items, domain.StockReceiptItem{
			ProductID:    it.ProductID,
			ExpectedQty:  it.ExpectedQty,
			AcceptedQty:  it.AcceptedQty,
			RejectedQty:  it.RejectedQty,
			RejectReason: reason,
		})
	}

	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, req.WarehouseID); err != nil {
		return nil, err
	}

	loc, err := u.repo.GetLocationByID(ctx, tenantID, req.DestLocationID)
	if err != nil {
		return nil, err
	}
	if loc.WarehouseID == nil || *loc.WarehouseID != req.WarehouseID {
		return nil, receiptInvalid("Lokasi tujuan tidak berada di gudang yang dipilih")
	}
	if loc.Type != domain.LocationTypeInternal {
		return nil, receiptInvalid("Lokasi tujuan harus berupa lokasi penyimpanan internal")
	}
	return items, nil
}

func generateReceiptNumber() string {
	now := time.Now().UTC()
	return fmt.Sprintf("GR-%s-%s", now.Format("20060102"), strings.ToUpper(uuid.New().String()[:8]))
}

func (u *Usecase) CreateStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, req StockReceiptRequest) (*domain.StockReceipt, []domain.StockReceiptItem, error) {
	items, err := u.validateStockReceiptRequest(ctx, tenantID, userID, role, req)
	if err != nil {
		return nil, nil, err
	}
	rc := &domain.StockReceipt{
		ID:             uuid.New(),
		TenantID:       tenantID,
		ReceiptNumber:  generateReceiptNumber(),
		WarehouseID:    req.WarehouseID,
		DestLocationID: req.DestLocationID,
		SupplierName:   strings.TrimSpace(req.SupplierName),
		SupplierRef:    trimPtr(req.SupplierRef),
		Notes:          trimPtr(req.Notes),
		Status:         domain.StockReceiptStatusDraft,
		CreatedBy:      userID,
	}
	if err := u.repo.CreateStockReceipt(ctx, rc, items); err != nil {
		return nil, nil, fmt.Errorf("CreateStockReceipt: %w", err)
	}
	return u.repo.GetStockReceiptByID(ctx, tenantID, rc.ID)
}

func (u *Usecase) UpdateStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID, req StockReceiptRequest) (*domain.StockReceipt, []domain.StockReceiptItem, error) {
	existing, _, err := u.repo.GetStockReceiptByID(ctx, tenantID, receiptID)
	if err != nil {
		return nil, nil, err
	}
	// Must have write access to the warehouse the receipt currently belongs to.
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, existing.WarehouseID); err != nil {
		return nil, nil, err
	}
	if existing.Status != domain.StockReceiptStatusDraft {
		return nil, nil, domain.ErrStockReceiptNotDraft
	}
	items, err := u.validateStockReceiptRequest(ctx, tenantID, userID, role, req)
	if err != nil {
		return nil, nil, err
	}
	rc := &domain.StockReceipt{
		ID:             receiptID,
		TenantID:       tenantID,
		WarehouseID:    req.WarehouseID,
		DestLocationID: req.DestLocationID,
		SupplierName:   strings.TrimSpace(req.SupplierName),
		SupplierRef:    trimPtr(req.SupplierRef),
		Notes:          trimPtr(req.Notes),
	}
	// Repo re-checks DRAFT under a row lock, so a concurrent post cannot slip through.
	if err := u.repo.UpdateDraftStockReceipt(ctx, rc, items); err != nil {
		return nil, nil, err
	}
	return u.repo.GetStockReceiptByID(ctx, tenantID, receiptID)
}

func (u *Usecase) GetStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID) (*domain.StockReceipt, []domain.StockReceiptItem, error) {
	rc, items, err := u.repo.GetStockReceiptByID(ctx, tenantID, receiptID)
	if err != nil {
		return nil, nil, err
	}
	if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, rc.WarehouseID); err != nil {
		return nil, nil, err
	}
	if items == nil {
		items = []domain.StockReceiptItem{}
	}
	return rc, items, nil
}

func (u *Usecase) ListStockReceipts(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID, status *domain.StockReceiptStatus) ([]domain.StockReceipt, error) {
	if status != nil {
		switch *status {
		case domain.StockReceiptStatusDraft, domain.StockReceiptStatusPosted, domain.StockReceiptStatusCancelled:
		default:
			return nil, receiptInvalid("Status penerimaan tidak dikenal")
		}
	}
	if warehouseID != nil {
		if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, *warehouseID); err != nil {
			return nil, err
		}
		list, err := u.repo.ListStockReceipts(ctx, tenantID, warehouseID, status)
		if err != nil {
			return nil, err
		}
		if list == nil {
			list = []domain.StockReceipt{}
		}
		return list, nil
	}

	accessibleWHs, err := u.FilterAccessibleWarehouses(ctx, tenantID, userID, role)
	if err != nil {
		return nil, err
	}
	whSet := make(map[uuid.UUID]bool)
	for _, w := range accessibleWHs {
		whSet[w.ID] = true
	}
	all, err := u.repo.ListStockReceipts(ctx, tenantID, nil, status)
	if err != nil {
		return nil, err
	}
	result := []domain.StockReceipt{}
	for _, rc := range all {
		if whSet[rc.WarehouseID] {
			result = append(result, rc)
		}
	}
	return result, nil
}

// systemReceiptLocations resolves @VENDOR and @SCRAP before the posting tx is opened
// (GetOrCreateSystemLocation runs in its own transaction).
func (u *Usecase) systemReceiptLocations(ctx context.Context, tenantID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	vendorLoc, err := u.repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeVendor)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("stock receipt: get virtual vendor loc: %w", err)
	}
	scrapLoc, err := u.repo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeScrap)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("stock receipt: get virtual scrap loc: %w", err)
	}
	return vendorLoc.ID, scrapLoc.ID, nil
}

func (u *Usecase) PostStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID) (*domain.StockReceipt, error) {
	existing, _, err := u.repo.GetStockReceiptByID(ctx, tenantID, receiptID)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, existing.WarehouseID); err != nil {
		return nil, err
	}
	if existing.Status != domain.StockReceiptStatusDraft {
		return nil, domain.ErrStockReceiptNotDraft
	}
	vendorID, scrapID, err := u.systemReceiptLocations(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return u.repo.PostStockReceipt(ctx, tenantID, receiptID, userID, vendorID, scrapID)
}

func (u *Usecase) CancelStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID, reason string) (*domain.StockReceipt, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, receiptInvalid("Alasan pembatalan wajib diisi")
	}
	existing, _, err := u.repo.GetStockReceiptByID(ctx, tenantID, receiptID)
	if err != nil {
		return nil, err
	}
	if err := u.ValidateWarehouseWriteAccess(ctx, tenantID, userID, role, existing.WarehouseID); err != nil {
		return nil, err
	}
	if existing.Status == domain.StockReceiptStatusCancelled {
		return nil, domain.ErrStockReceiptAlreadyCancelled
	}
	vendorID, scrapID, err := u.systemReceiptLocations(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return u.repo.CancelStockReceipt(ctx, tenantID, receiptID, userID, vendorID, scrapID, reason)
}
