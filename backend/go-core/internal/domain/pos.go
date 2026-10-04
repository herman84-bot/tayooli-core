package domain

import (
	"context"
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
	Items          []POSOrderItem  `json:"items,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type POSRepository interface {
	CreateOrder(ctx context.Context, order *POSOrder, items []POSOrderItem) error
	GetOrderByID(ctx context.Context, tenantID, id uuid.UUID) (*POSOrder, error)
	ListOrders(ctx context.Context, tenantID uuid.UUID, limit int) ([]POSOrder, error)
}
