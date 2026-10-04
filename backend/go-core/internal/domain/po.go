package domain

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreatePOParams struct {
	TenantID uuid.UUID
	VendorID string
	PONumber string
	Amount   decimal.Decimal
	Qty      int
	Currency string
}

type CreateGRParams struct {
	TenantID       uuid.UUID
	POID           uuid.UUID
	VendorID       string
	ReceivedQty    int
	ReceivedAmount decimal.Decimal
	Currency       string
}

// PurchaseOrderListPage holds a paginated result set plus total count.
type PurchaseOrderListPage struct {
	Data    []PurchaseOrder `json:"data"`
	Total   int             `json:"total"`
	Page    int             `json:"page"`
	PerPage int             `json:"per_page"`
}

// GoodsReceiptListPage holds a paginated result set plus total count.
type GoodsReceiptListPage struct {
	Data    []GoodsReceipt `json:"data"`
	Total   int            `json:"total"`
	Page    int            `json:"page"`
	PerPage int            `json:"per_page"`
}

type PORepository interface {
	Create(ctx context.Context, params CreatePOParams) (*PurchaseOrder, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]PurchaseOrder, error)
	ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*PurchaseOrderListPage, error)
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*PurchaseOrder, error)
}

type GRRepository interface {
	Create(ctx context.Context, params CreateGRParams) (*GoodsReceipt, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]GoodsReceipt, error)
	ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*GoodsReceiptListPage, error)
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*GoodsReceipt, error)
}
