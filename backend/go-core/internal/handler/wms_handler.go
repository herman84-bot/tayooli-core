package handler

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/wms"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

type WMSUsecase interface {
	// Warehouse
	CreateWarehouse(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateWarehouseRequest) (*domain.Warehouse, error)
	GetWarehouse(ctx context.Context, tenantID, userID uuid.UUID, role string, id uuid.UUID) (*domain.Warehouse, error)
	ListWarehouses(ctx context.Context, tenantID, userID uuid.UUID, role string) ([]domain.Warehouse, error)
	AssignUserWarehouse(ctx context.Context, tenantID, userID uuid.UUID, role string, targetUserID, warehouseID uuid.UUID) error

	// Location
	CreateLocation(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateLocationRequest) (*domain.WarehouseLocation, error)
	ListLocations(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.WarehouseLocation, error)

	// Barcode / SKU
	ResolveBarcode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.ResolvedProduct, error)
	CreateBarcode(ctx context.Context, tenantID uuid.UUID, req uc.CreateBarcodeRequest) (*domain.ProductBarcode, error)
	CreateSKUMapping(ctx context.Context, tenantID uuid.UUID, req uc.CreateSKUMappingRequest) (*domain.ProductSKUMapping, error)

	// Transfers
	CreateTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateTransferRequest) (*domain.StockTransfer, error)
	SubmitTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error)
	ApproveTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error)
	RejectTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID, reason string) (*domain.StockTransfer, error)
	DispatchTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error)
	ReceiveTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error)
	GetTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, []domain.StockTransferItem, error)
	ListTransfers(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.StockTransfer, error)
	CancelTransfer(ctx context.Context, tenantID, userID uuid.UUID, role string, transferID uuid.UUID) (*domain.StockTransfer, error)

	// Delivery Orders
	CreateDeliveryOrder(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateDeliveryOrderRequest) (*domain.DeliveryOrder, error)
	DispatchDeliveryOrder(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID) (*domain.DeliveryOrder, error)
	GetDeliveryOrder(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID) (*domain.DeliveryOrder, []domain.DeliveryOrderItem, error)
	ListDeliveryOrders(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.DeliveryOrder, error)

	// Stock Opnames
	CreateStockOpname(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateStockOpnameRequest) (*domain.StockOpname, error)
	GetStockOpname(ctx context.Context, tenantID, userID uuid.UUID, role string, opnameID uuid.UUID) (*domain.StockOpname, []domain.StockOpnameItem, error)
	ListStockOpnames(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.StockOpname, error)
	AddOpnameItem(ctx context.Context, tenantID, userID uuid.UUID, role string, opnameID uuid.UUID, req uc.AddOpnameItemRequest) (*domain.StockOpnameItem, error)
	CompleteStockOpname(ctx context.Context, tenantID, userID uuid.UUID, role string, opnameID uuid.UUID) (*domain.StockOpname, error)

	// Stock Scraps
	CreateStockScrap(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateStockScrapRequest) (*domain.StockScrap, error)
	ListStockScraps(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.StockScrap, error)

	// Stock Receipts (Barang Masuk)
	CreateStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.StockReceiptRequest) (*domain.StockReceipt, []domain.StockReceiptItem, error)
	UpdateStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID, req uc.StockReceiptRequest) (*domain.StockReceipt, []domain.StockReceiptItem, error)
	GetStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID) (*domain.StockReceipt, []domain.StockReceiptItem, error)
	ListStockReceipts(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID, status *domain.StockReceiptStatus, receiptType *domain.StockReceiptType) ([]domain.StockReceipt, error)
	PostStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID) (*domain.StockReceipt, error)
	CancelStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID, reason string) (*domain.StockReceipt, error)

	// Stock movements & Ledger
	ListStockMovements(ctx context.Context, tenantID, userID uuid.UUID, role string, productID, locationID *uuid.UUID, limit int) ([]domain.StockMovement, error)
	ListStockSummary(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.StockSummary, error)

	// Marketplace Sales Orders & SKU Mappings
	ImportMarketplaceOrders(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.ImportMarketplaceOrdersRequest) (*uc.ImportMarketplaceOrdersResponse, error)
	ResolveSKUMapping(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.CreateSKUMappingRequest) (*domain.ProductSKUMapping, error)
	ListMarketplaceBatches(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID) ([]domain.MarketplaceImportBatch, error)
	ListMarketplaceOrders(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID, batchID *uuid.UUID, status *domain.MarketplaceOrderStatus) ([]domain.MarketplaceOrder, error)
	GetMarketplaceOrder(ctx context.Context, tenantID, userID uuid.UUID, role string, orderID uuid.UUID) (*domain.MarketplaceOrder, error)
	ListSKUMappings(ctx context.Context, tenantID uuid.UUID, channelName string) ([]domain.ProductSKUMapping, error)

	// Sprint 1: Putaway, Release, Settings, Default Locations, Trace
	GetPutawayPending(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID uuid.UUID) ([]domain.PutawayPendingLine, error)
	ConfirmPutaway(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.PutawayRequest) (*domain.StockMovement, error)
	SubmitQCInspection(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID, in domain.QCInspectionInput) (*domain.QCInspectionDetail, error)
	GetReceiptQC(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID) (*domain.QCInspectionDetail, error)
	ListQCInspections(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID uuid.UUID) ([]domain.QCInspection, error)
	ListQuarantineStock(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID uuid.UUID) ([]domain.BatchBalance, error)
	ReleaseQuarantine(ctx context.Context, tenantID, userID uuid.UUID, role string, in domain.QuarantineActionInput) (*domain.StockMovement, error)
	ScrapQuarantine(ctx context.Context, tenantID, userID uuid.UUID, role string, in domain.QuarantineActionInput) (*domain.StockMovement, error)
	// Outbound Sprint 3: Picking Tasks & Pack Station
	GetPickingTask(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID) (*domain.PickingTaskDetail, error)
	StartPickingTask(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID) (*domain.PickingTaskDetail, error)
	RecordPickingItem(ctx context.Context, tenantID, userID uuid.UUID, role string, doID, taskItemID uuid.UUID, pickedQty decimal.Decimal) (*domain.PickingTaskDetail, error)
	ReportPickingDamaged(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID, req domain.PickingDamagedReportRequest) (*domain.PickingDamagedReportResult, error)
	ScanPackStationItem(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID, req domain.PackScanRequest) (*domain.PackScanResult, error)
	CompletePackStation(ctx context.Context, tenantID, userID uuid.UUID, role string, doID uuid.UUID, req domain.PackCompleteRequest) (*domain.DeliveryOrder, error)
	ReleaseStockReceipt(ctx context.Context, tenantID, userID uuid.UUID, role string, receiptID uuid.UUID) (*domain.StockReceipt, error)
	GetWMSSettings(ctx context.Context, tenantID, userID uuid.UUID, role string) (*domain.WMSSettings, error)
	UpdateWMSSettings(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.UpdateWMSSettingsRequest) (*domain.WMSSettings, error)
	ListDefaultLocations(ctx context.Context, tenantID, userID uuid.UUID, role string, productID *uuid.UUID) ([]domain.ProductDefaultLocation, error)
	SetDefaultLocation(ctx context.Context, tenantID, userID uuid.UUID, role string, req uc.SetDefaultLocationRequest) error
	DeleteDefaultLocation(ctx context.Context, tenantID, userID uuid.UUID, role string, productID, warehouseID uuid.UUID) error
	TraceBatch(ctx context.Context, tenantID, userID uuid.UUID, role string, batchID uuid.UUID) (*domain.BatchTrace, error)
	TraceDocument(ctx context.Context, tenantID, userID uuid.UUID, role string, refType string, refID uuid.UUID) (*domain.DocumentTrace, error)
	ListAuditTrail(ctx context.Context, tenantID, userID uuid.UUID, role string, entityType string, entityID uuid.UUID) ([]domain.AuditTrailEntry, error)
}

type WMSHandler struct {
	uc WMSUsecase
}

func NewWMSHandler(u WMSUsecase) *WMSHandler {
	return &WMSHandler{uc: u}
}

func (h *WMSHandler) RegisterRoutes(r chi.Router) {
	r.Route("/wms", func(r chi.Router) {
		r.Get("/warehouses", h.ListWarehouses)
		r.Post("/warehouses", h.CreateWarehouse)
		r.Get("/warehouses/{id}", h.GetWarehouse)
		r.Get("/locations", h.ListLocations)
		r.Post("/locations", h.CreateLocation)
		r.Get("/barcodes/resolve", h.ResolveBarcode)
		r.Get("/transfers", h.ListTransfers)
		r.Post("/transfers", h.CreateTransfer)
		r.Get("/transfers/{id}", h.GetTransfer)
		r.Post("/transfers/{id}/submit", h.SubmitTransfer)
		r.Post("/transfers/{id}/approve", h.ApproveTransfer)
		r.Post("/transfers/{id}/reject", h.RejectTransfer)
		r.Post("/transfers/{id}/dispatch", h.DispatchTransfer)
		r.Post("/transfers/{id}/receive", h.ReceiveTransfer)
		r.Delete("/transfers/{id}", h.CancelTransfer)
		r.Get("/delivery-orders", h.ListDeliveryOrders)
		r.Post("/delivery-orders", h.CreateDeliveryOrder)
		r.Get("/delivery-orders/{id}", h.GetDeliveryOrder)
		r.Post("/delivery-orders/{id}/dispatch", h.DispatchDeliveryOrder)
		// Sprint 3 Outbound Picking & Pack Station
		r.Get("/delivery-orders/{id}/picking", h.GetPickingTask)
		r.Post("/delivery-orders/{id}/picking/start", h.StartPickingTask)
		r.Post("/delivery-orders/{id}/picking/items/{itemId}", h.RecordPickingItem)
		r.Post("/delivery-orders/{id}/picking/damaged", h.ReportPickingDamaged)
		r.Post("/delivery-orders/{id}/pack/scan", h.ScanPackStationItem)
		r.Post("/delivery-orders/{id}/pack/complete", h.CompletePackStation)

		// Stock Opnames
		r.Get("/opnames", h.ListStockOpnames)
		r.Post("/opnames", h.CreateStockOpname)
		r.Get("/opnames/{id}", h.GetStockOpname)
		r.Post("/opnames/{id}/items", h.AddOpnameItem)
		r.Post("/opnames/{id}/complete", h.CompleteStockOpname)

		// Stock Scraps
		r.Get("/scraps", h.ListStockScraps)
		r.Post("/scraps", h.CreateStockScrap)

		// Stock Receipts (Barang Masuk)
		r.Get("/receipts", h.ListStockReceipts)
		r.Post("/receipts", h.CreateStockReceipt)
		r.Get("/receipts/{id}", h.GetStockReceipt)
		r.Put("/receipts/{id}", h.UpdateStockReceipt)
		r.Post("/receipts/{id}/post", h.PostStockReceipt)
		r.Post("/receipts/{id}/cancel", h.CancelStockReceipt)
		r.Post("/receipts/{id}/release", h.ReleaseStockReceipt)

		// Putaway (sentry-wms §1.2 step 2)
		r.Get("/putaway/pending", h.GetPutawayPending)
		r.Post("/putaway/confirm", h.ConfirmPutaway)

		// QC Inbound & Karantina (Sprint 2, ADR-014 Invariant 3)
		r.Get("/receipts/{id}/qc", h.GetReceiptQC)
		r.Post("/receipts/{id}/qc", h.SubmitQCInspection)
		r.Get("/qc-inspections", h.ListQCInspections)
		r.Get("/quarantine", h.ListQuarantineStock)
		r.Post("/quarantine/release", h.ReleaseQuarantine)
		r.Post("/quarantine/scrap", h.ScrapQuarantine)

		// WMS Settings (PDF-06)
		r.Get("/settings", h.GetWMSSettings)
		r.Put("/settings", h.UpdateWMSSettings)

		// Product Default Locations (CR-03)
		r.Get("/default-locations", h.ListDefaultLocations)
		r.Post("/default-locations", h.SetDefaultLocation)
		r.Delete("/default-locations", h.DeleteDefaultLocation)

		// Traceability (KO-1c) & Audit
		r.Get("/trace/batch/{id}", h.TraceBatch)
		r.Get("/trace/document", h.TraceDocument)
		r.Get("/audit-trail", h.ListAuditTrail)

		// Stock movements & Ledger
		r.Get("/movements", h.ListStockMovements)
		r.Get("/stock", h.ListStockSummary)
		r.Get("/inventory", h.ListStockSummary)

		// Marketplace Sales Orders & SKU Mappings
		r.Route("/marketplace", func(r chi.Router) {
			r.Post("/import", h.ImportMarketplaceOrders)
			r.Get("/batches", h.ListMarketplaceBatches)
			r.Get("/orders", h.ListMarketplaceOrders)
			r.Get("/orders/{id}", h.GetMarketplaceOrder)
			r.Post("/sku-mappings", h.CreateSKUMapping)
			r.Get("/sku-mappings", h.ListSKUMappings)
		})
	})
}

func handleWMSError(w http.ResponseWriter, r *http.Request, err error) {
	var maxBytesErr *http.MaxBytesError
	var receiptValErr *domain.StockReceiptValidationError
	switch {
	case errors.As(err, &maxBytesErr):
		RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
	case errors.As(err, &receiptValErr):
		RespondError(w, r, http.StatusBadRequest, receiptValErr.Msg)
	case errors.Is(err, domain.ErrStockReceiptNotFound):
		RespondError(w, r, http.StatusNotFound, "Penerimaan barang tidak ditemukan")
	case errors.Is(err, domain.ErrStockReceiptNotDraft):
		RespondError(w, r, http.StatusConflict, "Penerimaan sudah dikonfirmasi atau dibatalkan dan tidak bisa diubah")
	case errors.Is(err, domain.ErrStockReceiptAlreadyCancelled):
		RespondError(w, r, http.StatusConflict, "Penerimaan sudah dibatalkan sebelumnya")
	case errors.Is(err, domain.ErrStockReceiptStockConsumed):
		RespondError(w, r, http.StatusUnprocessableEntity, "Penerimaan tidak bisa dibatalkan karena sebagian barang sudah terpakai, dipindahkan, atau terjual")
	case errors.Is(err, domain.ErrPutawayReasonRequired):
		RespondError(w, r, http.StatusBadRequest, "Alasan putaway wajib diisi jika lokasi tujuan berbeda dari rak default")
	case errors.Is(err, domain.ErrInvalidPutawayLocation):
		RespondError(w, r, http.StatusBadRequest, "Lokasi rak tujuan putaway tidak valid atau bukan rak internal")
	case errors.Is(err, domain.ErrReceiptNotOnHold):
		RespondError(w, r, http.StatusBadRequest, "Penerimaan barang tidak memiliki lot yang berstatus ON_HOLD")
	case errors.Is(err, domain.ErrBatchOnHold):
		RespondError(w, r, http.StatusConflict, "Batch/Lot berstatus ON_HOLD dan belum disetujui untuk rilis")
	case errors.Is(err, domain.ErrQCAlreadyInspected):
		RespondError(w, r, http.StatusConflict, "Penerimaan ini sudah diinspeksi QC")
	case errors.Is(err, domain.ErrQCReceiptNotPosted):
		RespondError(w, r, http.StatusConflict, "Hanya penerimaan berstatus POSTED yang bisa diinspeksi QC")
	case errors.Is(err, domain.ErrQCSamplingFailed):
		RespondError(w, r, http.StatusUnprocessableEntity, "Sampel gagal: wajib ulangi dengan inspeksi FULL")
	case errors.Is(err, domain.ErrQCBAKDriverRequired):
		RespondError(w, r, http.StatusBadRequest, "Barang rusak wajib dibuatkan BAK: isi nama sopir dan konfirmasi tanda tangan sopir")
	case errors.Is(err, domain.ErrQCStagedQtyChanged):
		RespondError(w, r, http.StatusConflict, "Jumlah rusak melebihi stok yang masih di staging (sebagian mungkin sudah di-putaway)")
	case errors.Is(err, domain.ErrQuarantineQtyInvalid):
		RespondError(w, r, http.StatusConflict, "Jumlah melebihi stok karantina batch ini")
	case errors.Is(err, domain.ErrQuarantineStockBlocked):
		RespondError(w, r, http.StatusConflict, "Stok di area karantina tidak bisa diambil, dijual, atau dipindahkan. Gunakan menu QC & Karantina untuk rilis atau musnahkan.")
	case errors.Is(err, domain.ErrPackStationIncomplete):
		RespondError(w, r, http.StatusBadRequest, "Semua item harus dipindai 100% sebelum menyelesaikan pengemasan")
	case errors.Is(err, domain.ErrPackBarcodeMismatch):
		RespondError(w, r, http.StatusUnprocessableEntity, "Barcode tidak cocok dengan item pesanan ini atau item sudah selesai dikemas")
	case errors.Is(err, domain.ErrPackQtyExceeded):
		RespondError(w, r, http.StatusUnprocessableEntity, "Jumlah pemindaian melebihi sisa yang harus dikemas")
	case errors.Is(err, domain.ErrPickingTaskNotFound):
		RespondError(w, r, http.StatusNotFound, "Dokumen picking task tidak ditemukan")
	case errors.Is(err, domain.ErrScrapNotesRequired):
		RespondError(w, r, http.StatusBadRequest, "Catatan wajib diisi untuk memusnahkan stok karantina")
	case errors.Is(err, domain.ErrBatchRequired):
		RespondError(w, r, http.StatusBadRequest, "Batch ID wajib dicantumkan pada setiap mutasi barang")
	case errors.Is(err, domain.ErrConflict):
		RespondError(w, r, http.StatusConflict, "conflict")
	case errors.Is(err, domain.ErrWarehouseNotFound):
		RespondError(w, r, http.StatusNotFound, "warehouse not found")
	case errors.Is(err, domain.ErrSourceLocationRequired):
		RespondError(w, r, http.StatusUnprocessableEntity, "Item transfer belum memiliki lokasi rak asal. Lengkapi data rak sebelum pengiriman.")
	case errors.Is(err, domain.ErrLocationNotFound):
		RespondError(w, r, http.StatusNotFound, "Lokasi rak tidak ditemukan di gudang ini. Periksa rak asal/tujuan atau tambahkan rak di menu Warehouse.")
	case errors.Is(err, domain.ErrBarcodeNotFound):
		RespondError(w, r, http.StatusNotFound, "barcode not found")
	case errors.Is(err, domain.ErrTransferNotFound):
		RespondError(w, r, http.StatusNotFound, "stock transfer not found")
	case errors.Is(err, domain.ErrTransferNotDraft):
		RespondError(w, r, http.StatusConflict, "Hanya transfer berstatus DRAFT yang bisa dibatalkan. Transfer yang sudah diajukan atau diproses harus menunggu penolakan atau penerimaan.")
	case errors.Is(err, domain.ErrDeliveryOrderNotFound):
		RespondError(w, r, http.StatusNotFound, "delivery order not found")
	case errors.Is(err, domain.ErrOpnameNotFound):
		RespondError(w, r, http.StatusNotFound, "stock opname not found")
	case errors.Is(err, domain.ErrScrapNotFound):
		RespondError(w, r, http.StatusNotFound, "stock scrap not found")
	case errors.Is(err, domain.ErrDuplicateMarketplaceOrder):
		RespondError(w, r, http.StatusConflict, "duplicate marketplace order")
	case errors.Is(err, domain.ErrBatchNotFound):
		RespondError(w, r, http.StatusNotFound, "marketplace import batch not found")
	case errors.Is(err, domain.ErrMarketplaceOrderNotFound):
		RespondError(w, r, http.StatusNotFound, "marketplace order not found")
	case errors.Is(err, domain.ErrSKUMappingNotFound):
		RespondError(w, r, http.StatusNotFound, "sku mapping not found")
	case errors.Is(err, domain.ErrNotFound):
		RespondError(w, r, http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrUnauthorizedWarehouse):
		RespondError(w, r, http.StatusForbidden, "unauthorized warehouse access")
	case errors.Is(err, domain.ErrForbidden):
		RespondError(w, r, http.StatusForbidden, "forbidden")
	case errors.Is(err, domain.ErrInsufficientStock):
		RespondError(w, r, http.StatusUnprocessableEntity, "insufficient stock at location")
	case errors.Is(err, domain.ErrInvalidTransferStatus):
		RespondError(w, r, http.StatusBadRequest, "invalid transfer status transition")
	case errors.Is(err, domain.ErrSelfApprovalForbidden):
		RespondError(w, r, http.StatusForbidden, "requester cannot approve or reject their own transfer")
	case errors.Is(err, domain.ErrRejectionReasonRequired):
		RespondError(w, r, http.StatusBadRequest, "rejection reason is required")
	case errors.Is(err, domain.ErrInvalidOpnameStatus):
		RespondError(w, r, http.StatusBadRequest, "invalid stock opname status transition")
	case errors.Is(err, domain.ErrInvalidStatus):
		RespondError(w, r, http.StatusBadRequest, "invalid status for this operation")
	case errors.Is(err, domain.ErrInvalidInput):
		RespondError(w, r, http.StatusBadRequest, "invalid input")
	default:
		log.Error().Err(err).Str("path", r.URL.Path).Msg("wms handler internal error")
		RespondError(w, r, http.StatusInternalServerError, "internal server error")
	}
}

// -----------------------------------------------------------------------------
// Warehouse Handlers
// -----------------------------------------------------------------------------

// ListWarehouses handles GET /api/v1/wms/warehouses
func (h *WMSHandler) ListWarehouses(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	warehouses, err := h.uc.ListWarehouses(r.Context(), tenantID, userID, role)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if warehouses == nil {
		warehouses = []domain.Warehouse{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": warehouses})
}

// CreateWarehouse handles POST /api/v1/wms/warehouses
func (h *WMSHandler) CreateWarehouse(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	var req uc.CreateWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
			return
		}
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return
	}

	wh, err := h.uc.CreateWarehouse(r.Context(), tenantID, userID, role, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, wh)
}

// -----------------------------------------------------------------------------
// Location Handlers
// -----------------------------------------------------------------------------

// ListLocations handles GET /api/v1/wms/locations?warehouse_id=
func (h *WMSHandler) ListLocations(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var whIDPtr *uuid.UUID
	whIDStr := r.URL.Query().Get("warehouse_id")
	if whIDStr != "" {
		parsed, err := uuid.Parse(whIDStr)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, "invalid warehouse_id uuid")
			return
		}
		whIDPtr = &parsed
	}

	locations, err := h.uc.ListLocations(r.Context(), tenantID, userID, role, whIDPtr)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if locations == nil {
		locations = []domain.WarehouseLocation{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": locations})
}

// CreateLocation handles POST /api/v1/wms/locations
func (h *WMSHandler) CreateLocation(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	var req uc.CreateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
			return
		}
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return
	}

	loc, err := h.uc.CreateLocation(r.Context(), tenantID, userID, role, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, loc)
}

// -----------------------------------------------------------------------------
// Barcode Resolution Handlers
// -----------------------------------------------------------------------------

// ResolveBarcode handles GET /api/v1/wms/barcodes/resolve?code=
func (h *WMSHandler) ResolveBarcode(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		RespondError(w, r, http.StatusBadRequest, "missing code query parameter")
		return
	}

	res, err := h.uc.ResolveBarcode(r.Context(), tenantID, code)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, res)
}

// -----------------------------------------------------------------------------
// Stock Transfer Handlers
// -----------------------------------------------------------------------------

// CreateTransfer handles POST /api/v1/wms/transfers
func (h *WMSHandler) CreateTransfer(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	var req uc.CreateTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
			return
		}
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return
	}

	transfer, err := h.uc.CreateTransfer(r.Context(), tenantID, userID, role, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, transfer)
}

// SubmitTransfer handles POST /api/v1/wms/transfers/{id}/submit
// Moves a DRAFT transfer into PENDING_APPROVAL so a supervisor/manager can review it.
func (h *WMSHandler) SubmitTransfer(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	transferID, ok := parseUUIDParam(r, "id")
	if !ok {
		RespondError(w, r, http.StatusBadRequest, "invalid transfer id")
		return
	}

	transfer, err := h.uc.SubmitTransfer(r.Context(), tenantID, userID, role, transferID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, transfer)
}

// ApproveTransfer handles POST /api/v1/wms/transfers/{id}/approve
// Only admin/owner/regional_manager roles (and never the original requester)
// may approve a PENDING_APPROVAL transfer.
func (h *WMSHandler) ApproveTransfer(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	transferID, ok := parseUUIDParam(r, "id")
	if !ok {
		RespondError(w, r, http.StatusBadRequest, "invalid transfer id")
		return
	}

	transfer, err := h.uc.ApproveTransfer(r.Context(), tenantID, userID, role, transferID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, transfer)
}

// RejectTransferRequest is the request body for POST /transfers/{id}/reject.
type RejectTransferRequest struct {
	Reason string `json:"reason"`
}

// RejectTransfer handles POST /api/v1/wms/transfers/{id}/reject
// Requires a non-empty rejection reason. Same approver rules as ApproveTransfer.
func (h *WMSHandler) RejectTransfer(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	transferID, ok := parseUUIDParam(r, "id")
	if !ok {
		RespondError(w, r, http.StatusBadRequest, "invalid transfer id")
		return
	}

	var req RejectTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
			return
		}
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return
	}

	transfer, err := h.uc.RejectTransfer(r.Context(), tenantID, userID, role, transferID, req.Reason)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, transfer)
}

// CancelTransfer handles DELETE /api/v1/wms/transfers/{id}
// Discards a DRAFT transfer. The record is kept with status CANCELLED so the
// history stays auditable instead of vanishing from the list.
func (h *WMSHandler) CancelTransfer(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	transferID, ok := parseUUIDParam(r, "id")
	if !ok {
		RespondError(w, r, http.StatusBadRequest, "invalid transfer id")
		return
	}

	transfer, err := h.uc.CancelTransfer(r.Context(), tenantID, userID, role, transferID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, transfer)
}

// DispatchTransfer handles POST /api/v1/wms/transfers/{id}/dispatch
func (h *WMSHandler) DispatchTransfer(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	transferID, ok := parseUUIDParam(r, "id")
	if !ok {
		RespondError(w, r, http.StatusBadRequest, "invalid transfer id")
		return
	}

	transfer, err := h.uc.DispatchTransfer(r.Context(), tenantID, userID, role, transferID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, transfer)
}

// ReceiveTransfer handles POST /api/v1/wms/transfers/{id}/receive
func (h *WMSHandler) ReceiveTransfer(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	transferID, ok := parseUUIDParam(r, "id")
	if !ok {
		RespondError(w, r, http.StatusBadRequest, "invalid transfer id")
		return
	}

	transfer, err := h.uc.ReceiveTransfer(r.Context(), tenantID, userID, role, transferID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, transfer)
}

// -----------------------------------------------------------------------------
// Delivery Order Handlers (Surat Jalan)
// -----------------------------------------------------------------------------

// ListDeliveryOrders handles GET /api/v1/wms/delivery-orders?warehouse_id=
func (h *WMSHandler) ListDeliveryOrders(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var whIDPtr *uuid.UUID
	whIDStr := r.URL.Query().Get("warehouse_id")
	if whIDStr != "" {
		parsed, err := uuid.Parse(whIDStr)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, "invalid warehouse_id uuid")
			return
		}
		whIDPtr = &parsed
	}

	orders, err := h.uc.ListDeliveryOrders(r.Context(), tenantID, userID, role, whIDPtr)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if orders == nil {
		orders = []domain.DeliveryOrder{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": orders})
}

// CreateDeliveryOrder handles POST /api/v1/wms/delivery-orders
func (h *WMSHandler) CreateDeliveryOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	var req uc.CreateDeliveryOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
			return
		}
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return
	}

	do, err := h.uc.CreateDeliveryOrder(r.Context(), tenantID, userID, role, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, do)
}

// GetWarehouse handles GET /api/v1/wms/warehouses/{id}
func (h *WMSHandler) GetWarehouse(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	whID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid warehouse id uuid")
		return
	}

	wh, err := h.uc.GetWarehouse(r.Context(), tenantID, userID, role, whID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, wh)
}

// ListTransfers handles GET /api/v1/wms/transfers?warehouse_id=
func (h *WMSHandler) ListTransfers(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var whIDPtr *uuid.UUID
	whIDStr := r.URL.Query().Get("warehouse_id")
	if whIDStr != "" {
		parsed, err := uuid.Parse(whIDStr)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, "invalid warehouse_id uuid")
			return
		}
		whIDPtr = &parsed
	}

	transfers, err := h.uc.ListTransfers(r.Context(), tenantID, userID, role, whIDPtr)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if transfers == nil {
		transfers = []domain.StockTransfer{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": transfers})
}

// GetTransfer handles GET /api/v1/wms/transfers/{id}
func (h *WMSHandler) GetTransfer(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	transferID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid transfer id uuid")
		return
	}

	transfer, items, err := h.uc.GetTransfer(r.Context(), tenantID, userID, role, transferID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"transfer": transfer,
		"items":    items,
	})
}

// GetDeliveryOrder handles GET /api/v1/wms/delivery-orders/{id}
func (h *WMSHandler) GetDeliveryOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	doID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid delivery order id uuid")
		return
	}

	do, items, err := h.uc.GetDeliveryOrder(r.Context(), tenantID, userID, role, doID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"delivery_order": do,
		"items":          items,
	})
}

// DispatchDeliveryOrder handles POST /api/v1/wms/delivery-orders/{id}/dispatch
func (h *WMSHandler) DispatchDeliveryOrder(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	doID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid delivery order id uuid")
		return
	}

	do, err := h.uc.DispatchDeliveryOrder(r.Context(), tenantID, userID, role, doID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, do)
}

// -----------------------------------------------------------------------------
// Stock Opname Handlers
// -----------------------------------------------------------------------------

// ListStockOpnames handles GET /api/v1/wms/opnames?warehouse_id=
func (h *WMSHandler) ListStockOpnames(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var whIDPtr *uuid.UUID
	whIDStr := r.URL.Query().Get("warehouse_id")
	if whIDStr != "" {
		parsed, err := uuid.Parse(whIDStr)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, "invalid warehouse_id uuid")
			return
		}
		whIDPtr = &parsed
	}

	opnames, err := h.uc.ListStockOpnames(r.Context(), tenantID, userID, role, whIDPtr)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if opnames == nil {
		opnames = []domain.StockOpname{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": opnames})
}

// CreateStockOpname handles POST /api/v1/wms/opnames
func (h *WMSHandler) CreateStockOpname(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var req uc.CreateStockOpnameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handleWMSError(w, r, err)
		return
	}

	op, err := h.uc.CreateStockOpname(r.Context(), tenantID, userID, role, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, op)
}

// GetStockOpname handles GET /api/v1/wms/opnames/{id}
func (h *WMSHandler) GetStockOpname(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	opnameID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid opname id uuid")
		return
	}

	op, items, err := h.uc.GetStockOpname(r.Context(), tenantID, userID, role, opnameID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if items == nil {
		items = []domain.StockOpnameItem{}
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"opname": op,
		"items":  items,
	})
}

// AddOpnameItem handles POST /api/v1/wms/opnames/{id}/items
func (h *WMSHandler) AddOpnameItem(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	opnameID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid opname id uuid")
		return
	}

	var req uc.AddOpnameItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handleWMSError(w, r, err)
		return
	}

	item, err := h.uc.AddOpnameItem(r.Context(), tenantID, userID, role, opnameID, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

// CompleteStockOpname handles POST /api/v1/wms/opnames/{id}/complete
func (h *WMSHandler) CompleteStockOpname(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	opnameID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid opname id uuid")
		return
	}

	op, err := h.uc.CompleteStockOpname(r.Context(), tenantID, userID, role, opnameID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, op)
}

// -----------------------------------------------------------------------------
// Stock Scrap Handlers
// -----------------------------------------------------------------------------

// ListStockScraps handles GET /api/v1/wms/scraps?warehouse_id=
func (h *WMSHandler) ListStockScraps(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var whIDPtr *uuid.UUID
	whIDStr := r.URL.Query().Get("warehouse_id")
	if whIDStr != "" {
		parsed, err := uuid.Parse(whIDStr)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, "invalid warehouse_id uuid")
			return
		}
		whIDPtr = &parsed
	}

	scraps, err := h.uc.ListStockScraps(r.Context(), tenantID, userID, role, whIDPtr)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if scraps == nil {
		scraps = []domain.StockScrap{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": scraps})
}

// CreateStockScrap handles POST /api/v1/wms/scraps
func (h *WMSHandler) CreateStockScrap(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var req uc.CreateStockScrapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handleWMSError(w, r, err)
		return
	}

	scrap, err := h.uc.CreateStockScrap(r.Context(), tenantID, userID, role, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, scrap)
}

// -----------------------------------------------------------------------------
// Marketplace Sales Orders & SKU Mappings
// -----------------------------------------------------------------------------

// ImportMarketplaceRequest represents the payload for importing orders via JSON or raw CSV string.
type ImportMarketplaceRequest struct {
	WarehouseID uuid.UUID                 `json:"warehouse_id"`
	Channel     domain.MarketplaceChannel `json:"channel"`
	FileName    string                    `json:"file_name"`
	CSVData     string                    `json:"csv_data,omitempty"`
	Orders      []uc.ImportOrderRequest   `json:"orders,omitempty"`
}

func parseMarketplaceCSV(r io.Reader, channel domain.MarketplaceChannel) ([]uc.ImportOrderRequest, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	if len(records) < 2 {
		return []uc.ImportOrderRequest{}, nil
	}

	headerMap := make(map[string]int)
	for i, h := range records[0] {
		norm := strings.TrimPrefix(strings.TrimSpace(h), "\ufeff")
		norm = strings.ToLower(norm)
		norm = strings.ReplaceAll(norm, "_", " ")
		headerMap[norm] = i
	}

	findCol := func(names ...string) int {
		for _, name := range names {
			if idx, ok := headerMap[strings.ToLower(name)]; ok {
				return idx
			}
		}
		return -1
	}

	orderIDIdx := findCol("nomor pesanan", "no. pesanan", "no pesanan", "order id", "order sn", "order_id", "external order id", "external_order_id")
	skuIdx := findCol("nomor referensi sku", "no. referensi sku", "sku induk", "sku", "seller sku", "item sku", "product sku", "external sku", "external_sku")
	nameIdx := findCol("nama produk", "nama barang", "product name", "item name", "product")
	qtyIdx := findCol("jumlah", "quantity", "qty", "jumlah produk")
	priceIdx := findCol("harga awal", "harga satuan", "unit price", "deal price", "harga", "price", "unit_price")
	subtotalIdx := findCol("total harga produk", "subtotal", "total price", "jumlah harga")
	totalAmtIdx := findCol("total pembayaran", "total amount", "grand total", "total pesanan", "total")
	shippingFeeIdx := findCol("ongkos kirim dibayar pembeli", "ongkir", "shipping fee", "biaya pengiriman", "shipping_fee")
	mktFeeIdx := findCol("biaya layanan", "biaya transaksi", "marketplace fee", "service fee", "marketplace_fee")
	custNameIdx := findCol("nama pembeli", "customer name", "username (pembeli)", "nama penerima", "customer")
	custPhoneIdx := findCol("nomor telepon pembeli", "no. telepon", "no telepon", "phone number", "phone", "telepon")
	addrIdx := findCol("alamat pengiriman", "shipping address", "alamat penerima", "alamat", "address")
	courierIdx := findCol("opsi pengiriman", "kurir", "shipping option", "jasa kirim", "courier")
	trackingIdx := findCol("no. resi", "no resi", "tracking number", "nomor pelacakan", "tracking", "airway bill")

	cleanDecimal := func(raw string) decimal.Decimal {
		raw = strings.TrimSpace(raw)
		raw = strings.ReplaceAll(raw, "Rp", "")
		raw = strings.ReplaceAll(raw, "rp", "")
		raw = strings.ReplaceAll(raw, "IDR", "")
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return decimal.Zero
		}
		// If contains thousands dot and decimal comma (e.g. 50.000,00)
		if strings.Contains(raw, ".") && strings.Contains(raw, ",") {
			raw = strings.ReplaceAll(raw, ".", "")
			raw = strings.ReplaceAll(raw, ",", ".")
		} else if strings.Count(raw, ".") > 1 {
			// Indonesian thousands without decimal (e.g. 1.250.000)
			raw = strings.ReplaceAll(raw, ".", "")
		} else if strings.Contains(raw, ",") {
			raw = strings.ReplaceAll(raw, ",", ".")
		}
		d, err := decimal.NewFromString(raw)
		if err == nil {
			return d
		}
		return decimal.Zero
	}

	ordersMap := make(map[string]*uc.ImportOrderRequest)
	orderKeys := make([]string, 0)

	sanitizeText := func(val string) string {
		val = strings.TrimSpace(val)
		if len(val) > 0 && (val[0] == '=' || val[0] == '+' || val[0] == '@') {
			return "'" + val
		}
		return val
	}

	for rowIdx, row := range records[1:] {
		getVal := func(idx int) string {
			if idx >= 0 && idx < len(row) {
				return sanitizeText(row[idx])
			}
			return ""
		}

		// Check if row is entirely blank
		isBlank := true
		for _, col := range row {
			if strings.TrimSpace(col) != "" {
				isBlank = false
				break
			}
		}
		if isBlank {
			continue
		}

		orderID := getVal(orderIDIdx)
		sku := getVal(skuIdx)
		name := getVal(nameIdx)
		if orderID == "" && sku == "" && name == "" {
			continue
		}
		if orderID == "" {
			orderID = fmt.Sprintf("CSV-ROW-%d", rowIdx+1)
		}

		reqOrder, exists := ordersMap[orderID]
		if !exists {
			var custNamePtr, custPhonePtr, addrPtr, courierPtr, trackingPtr *string
			if cn := getVal(custNameIdx); cn != "" {
				custNamePtr = &cn
			}
			if cp := getVal(custPhoneIdx); cp != "" {
				custPhonePtr = &cp
			}
			if ad := getVal(addrIdx); ad != "" {
				addrPtr = &ad
			}
			if cr := getVal(courierIdx); cr != "" {
				courierPtr = &cr
			}
			if tr := getVal(trackingIdx); tr != "" {
				trackingPtr = &tr
			}

			totAmt := cleanDecimal(getVal(totalAmtIdx))
			shipFee := cleanDecimal(getVal(shippingFeeIdx))
			mktFee := cleanDecimal(getVal(mktFeeIdx))

			reqOrder = &uc.ImportOrderRequest{
				ExternalOrderID: orderID,
				OrderDate:       time.Now().UTC(),
				CustomerName:    custNamePtr,
				CustomerPhone:   custPhonePtr,
				ShippingAddress: addrPtr,
				Courier:         courierPtr,
				TrackingNumber:  trackingPtr,
				TotalAmount:     totAmt,
				ShippingFee:     shipFee,
				MarketplaceFee:  mktFee,
				NetAmount:       totAmt.Add(shipFee).Sub(mktFee),
				Items:           make([]uc.ImportOrderItemRequest, 0),
			}
			ordersMap[orderID] = reqOrder
			orderKeys = append(orderKeys, orderID)
		}

		itemName := name
		if itemName == "" {
			itemName = sku
		}
		qty := cleanDecimal(getVal(qtyIdx))
		if qty.LessThanOrEqual(decimal.Zero) {
			qty = decimal.NewFromInt(1)
		}
		unitPrice := cleanDecimal(getVal(priceIdx))
		subtotal := cleanDecimal(getVal(subtotalIdx))
		if subtotal.IsZero() && !unitPrice.IsZero() {
			subtotal = qty.Mul(unitPrice)
		}

		reqOrder.Items = append(reqOrder.Items, uc.ImportOrderItemRequest{
			ExternalSKU: sku,
			ItemName:    itemName,
			Quantity:    qty,
			UnitPrice:   unitPrice,
			Subtotal:    subtotal,
		})
	}

	result := make([]uc.ImportOrderRequest, 0, len(orderKeys))
	for _, k := range orderKeys {
		result = append(result, *ordersMap[k])
	}
	return result, nil
}

// ImportMarketplaceOrders handles POST /api/v1/wms/marketplace/import
func (h *WMSHandler) ImportMarketplaceOrders(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	// Enforce 5MB limit
	r.Body = http.MaxBytesReader(w, r.Body, 5*1024*1024)

	contentType := r.Header.Get("Content-Type")

	var importReq uc.ImportMarketplaceOrdersRequest

	switch {
	case strings.Contains(contentType, "multipart/form-data"):
		if err := r.ParseMultipartForm(5 * 1024 * 1024); err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
				return
			}
			RespondError(w, r, http.StatusBadRequest, "invalid multipart form")
			return
		}
		whIDStr := r.FormValue("warehouse_id")
		whID, err := uuid.Parse(whIDStr)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, "invalid warehouse_id uuid")
			return
		}
		channel := domain.MarketplaceChannel(strings.ToUpper(strings.TrimSpace(r.FormValue("channel"))))
		file, header, err := r.FormFile("file")
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, "missing file in form data")
			return
		}
		defer file.Close()

		orders, err := parseMarketplaceCSV(file, channel)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, fmt.Sprintf("failed to parse csv: %v", err))
			return
		}
		importReq = uc.ImportMarketplaceOrdersRequest{
			WarehouseID: whID,
			Channel:     channel,
			FileName:    header.Filename,
			Orders:      orders,
		}

	case strings.Contains(contentType, "text/csv"):
		whIDStr := r.URL.Query().Get("warehouse_id")
		whID, err := uuid.Parse(whIDStr)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, "invalid warehouse_id uuid in query")
			return
		}
		channel := domain.MarketplaceChannel(strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("channel"))))
		fileName := r.URL.Query().Get("file_name")
		if fileName == "" {
			fileName = "import.csv"
		}
		orders, err := parseMarketplaceCSV(r.Body, channel)
		if err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
				return
			}
			RespondError(w, r, http.StatusBadRequest, fmt.Sprintf("failed to parse csv: %v", err))
			return
		}
		importReq = uc.ImportMarketplaceOrdersRequest{
			WarehouseID: whID,
			Channel:     channel,
			FileName:    fileName,
			Orders:      orders,
		}

	default: // application/json or fallback
		var jsonReq ImportMarketplaceRequest
		if err := json.NewDecoder(r.Body).Decode(&jsonReq); err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
				return
			}
			RespondError(w, r, http.StatusBadRequest, "invalid json payload")
			return
		}

		if jsonReq.WarehouseID == uuid.Nil {
			RespondError(w, r, http.StatusBadRequest, "warehouse_id is required")
			return
		}

		if len(jsonReq.Orders) == 0 && jsonReq.CSVData != "" {
			orders, err := parseMarketplaceCSV(strings.NewReader(jsonReq.CSVData), jsonReq.Channel)
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, fmt.Sprintf("failed to parse csv_data: %v", err))
				return
			}
			jsonReq.Orders = orders
		}

		fileName := jsonReq.FileName
		if fileName == "" {
			fileName = "marketplace_import.json"
		}

		importReq = uc.ImportMarketplaceOrdersRequest{
			WarehouseID: jsonReq.WarehouseID,
			Channel:     jsonReq.Channel,
			FileName:    fileName,
			Orders:      jsonReq.Orders,
		}
	}

	resp, err := h.uc.ImportMarketplaceOrders(r.Context(), tenantID, userID, role, importReq)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{"data": resp})
}

// ListMarketplaceBatches handles GET /api/v1/wms/marketplace/batches
func (h *WMSHandler) ListMarketplaceBatches(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var whID *uuid.UUID
	if s := r.URL.Query().Get("warehouse_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, "invalid warehouse_id uuid")
			return
		}
		whID = &id
	}

	batches, err := h.uc.ListMarketplaceBatches(r.Context(), tenantID, userID, role, whID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if batches == nil {
		batches = []domain.MarketplaceImportBatch{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": batches})
}

// ListMarketplaceOrders handles GET /api/v1/wms/marketplace/orders
func (h *WMSHandler) ListMarketplaceOrders(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var whID *uuid.UUID
	if s := r.URL.Query().Get("warehouse_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, "invalid warehouse_id uuid")
			return
		}
		whID = &id
	}

	var batchID *uuid.UUID
	if s := r.URL.Query().Get("batch_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, "invalid batch_id uuid")
			return
		}
		batchID = &id
	}

	var status *domain.MarketplaceOrderStatus
	if s := r.URL.Query().Get("status"); s != "" {
		st := domain.MarketplaceOrderStatus(strings.ToUpper(strings.TrimSpace(s)))
		status = &st
	}

	orders, err := h.uc.ListMarketplaceOrders(r.Context(), tenantID, userID, role, whID, batchID, status)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if orders == nil {
		orders = []domain.MarketplaceOrder{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": orders})
}

// GetMarketplaceOrder handles GET /api/v1/wms/marketplace/orders/{id}
func (h *WMSHandler) GetMarketplaceOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, "invalid order id uuid")
		return
	}

	order, err := h.uc.GetMarketplaceOrder(r.Context(), tenantID, userID, role, id)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": order})
}

// CreateSKUMapping handles POST /api/v1/wms/marketplace/sku-mappings
func (h *WMSHandler) CreateSKUMapping(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 5*1024*1024)
	var req uc.CreateSKUMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			RespondError(w, r, http.StatusRequestEntityTooLarge, "request entity too large")
			return
		}
		RespondError(w, r, http.StatusBadRequest, "invalid json payload")
		return
	}

	mapping, err := h.uc.ResolveSKUMapping(r.Context(), tenantID, userID, role, req)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	respondJSON(w, http.StatusCreated, map[string]any{"data": mapping})
}

// ListSKUMappings handles GET /api/v1/wms/marketplace/sku-mappings
func (h *WMSHandler) ListSKUMappings(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}

	channelName := r.URL.Query().Get("channel_name")
	if channelName == "" {
		channelName = r.URL.Query().Get("channel")
	}

	mappings, err := h.uc.ListSKUMappings(r.Context(), tenantID, channelName)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if mappings == nil {
		mappings = []domain.ProductSKUMapping{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": mappings})
}

// ListStockMovements handles GET /api/v1/wms/movements
func (h *WMSHandler) ListStockMovements(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	var productID, locationID *uuid.UUID
	if pid := r.URL.Query().Get("product_id"); pid != "" {
		if parsed, err := uuid.Parse(pid); err == nil {
			productID = &parsed
		}
	}
	if lid := r.URL.Query().Get("location_id"); lid != "" {
		if parsed, err := uuid.Parse(lid); err == nil {
			locationID = &parsed
		}
	}

	movements, err := h.uc.ListStockMovements(r.Context(), tenantID, userID, role, productID, locationID, limit)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if movements == nil {
		movements = []domain.StockMovement{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": movements})
}

// ListStockSummary handles GET /api/v1/wms/stock and /api/v1/wms/inventory
func (h *WMSHandler) ListStockSummary(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		RespondError(w, r, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())
	role := appMiddleware.GetRole(r.Context())

	var warehouseID *uuid.UUID
	if wid := r.URL.Query().Get("warehouse_id"); wid != "" {
		if parsed, err := uuid.Parse(wid); err == nil {
			warehouseID = &parsed
		}
	}

	stock, err := h.uc.ListStockSummary(r.Context(), tenantID, userID, role, warehouseID)
	if err != nil {
		handleWMSError(w, r, err)
		return
	}
	if stock == nil {
		stock = []domain.StockSummary{}
	}
	respondJSON(w, http.StatusOK, map[string]any{"data": stock})
}
