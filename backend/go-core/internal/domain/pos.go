package domain

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type POSOrderItem struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	POSOrderID  uuid.UUID       `json:"pos_order_id"`
	ProductID   uuid.UUID       `json:"product_id"`
	ProductName string          `json:"product_name"`
	SKU         string          `json:"sku"`
	Quantity    decimal.Decimal `json:"quantity"`
	Price       decimal.Decimal `json:"price"`
	Discount    decimal.Decimal `json:"discount"`
	Subtotal    decimal.Decimal `json:"subtotal"`
	CreatedAt   time.Time       `json:"created_at"`
}

type POSOrder struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	OrderNumber    string          `json:"order_number"`
	CustomerID     *uuid.UUID      `json:"customer_id,omitempty"`
	CustomerName   string          `json:"customer_name,omitempty"`
	WarehouseID    *uuid.UUID      `json:"warehouse_id,omitempty"`
	WarehouseName  string          `json:"warehouse_name,omitempty"`
	Subtotal       decimal.Decimal `json:"subtotal"`
	TaxAmount      decimal.Decimal `json:"tax_amount"`
	DiscountAmount decimal.Decimal `json:"discount_amount"`
	TotalAmount    decimal.Decimal `json:"total_amount"`
	PaymentMethod  string          `json:"payment_method"`
	PaymentAmount  decimal.Decimal `json:"payment_amount"`
	ChangeAmount   decimal.Decimal `json:"change_amount"`
	SaleMode       string          `json:"sale_mode"`
	Status         string          `json:"status"`
	SalesOrderID   *uuid.UUID      `json:"sales_order_id,omitempty"`
	SalesInvoiceID *uuid.UUID      `json:"sales_invoice_id,omitempty"`
	CashierID      *uuid.UUID      `json:"cashier_id,omitempty"`
	// CashierName is read-only, joined from users (full_name or email).
	CashierName    string          `json:"cashier_name,omitempty"`
	Items          []POSOrderItem  `json:"items,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// POSInventoryDeduction removes quantity from the legacy `inventory` rows of a
// product (oldest rows first). Products without inventory rows are skipped,
// matching the previous checkout behaviour.
type POSInventoryDeduction struct {
	ProductID uuid.UUID
	SKU       string
	Quantity  decimal.Decimal
}

// POSStockDeduction removes quantity from the WMS ledger of one warehouse with
// FEFO batch allocation (WMSRepo.DeductWarehouseStock semantics). Movement is
// the template for the stock_movements rows written for it.
type POSStockDeduction struct {
	WarehouseID uuid.UUID
	ProductID   uuid.UUID
	SKU         string
	Quantity    decimal.Decimal
	Movement    StockMovement
}

// POSSale is everything one POS checkout writes. POSRepository.CreateSale must
// persist it all in ONE database transaction: either the order, its items,
// every stock deduction and the optional sales order / invoice are committed,
// or nothing is (no stock leaves the shelf for a sale that was not recorded).
type POSSale struct {
	Order               *POSOrder
	Items               []POSOrderItem
	InventoryDeductions []POSInventoryDeduction
	StockDeductions     []POSStockDeduction
	SalesOrder          *SalesOrder   // optional
	SalesInvoice        *SalesInvoice // optional, requires SalesOrder
}

// LockOrder sorts deductions so every concurrent checkout takes row/advisory
// locks in the same order (product, then warehouse). Without it two carts with
// the same products in opposite order could deadlock inside one transaction.
func (s *POSSale) LockOrder() {
	sort.SliceStable(s.InventoryDeductions, func(i, j int) bool {
		return s.InventoryDeductions[i].ProductID.String() < s.InventoryDeductions[j].ProductID.String()
	})
	sort.SliceStable(s.StockDeductions, func(i, j int) bool {
		a, b := s.StockDeductions[i], s.StockDeductions[j]
		if a.ProductID != b.ProductID {
			return a.ProductID.String() < b.ProductID.String()
		}
		return a.WarehouseID.String() < b.WarehouseID.String()
	})
}

// LegacyInsufficientStockError is the cashier-facing error when legacy
// inventory rows cannot cover a line (message kept from the previous checkout).
func LegacyInsufficientStockError(sku string, available, requested decimal.Decimal) error {
	return fmt.Errorf("%w: SKU %s stok tersedia %s, diminta %s",
		ErrInsufficientStock, sku, available.StringFixed(2), requested.String())
}

// WrapPOSStockError adds the SKU to a WMS deduction error for the cashier.
func WrapPOSStockError(sku string, err error) error {
	switch {
	case errors.Is(err, ErrExpiredStockBlocked):
		return fmt.Errorf("%w (SKU %s)", ErrExpiredStockBlocked, sku)
	case errors.Is(err, ErrInsufficientStock):
		return fmt.Errorf("%w: stok gudang tidak mencukupi untuk SKU %s", ErrInsufficientStock, sku)
	default:
		return fmt.Errorf("pos: deduct warehouse stock SKU %s: %w", sku, err)
	}
}

type POSRepository interface {
	// CreateSale persists a whole checkout atomically (see POSSale).
	CreateSale(ctx context.Context, sale *POSSale) error
	CreateOrder(ctx context.Context, order *POSOrder, items []POSOrderItem) error
	GetOrderByID(ctx context.Context, tenantID, id uuid.UUID) (*POSOrder, error)
	ListOrders(ctx context.Context, tenantID uuid.UUID, limit int) ([]POSOrder, error)
}
