package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/lib/pq"
)

type POSRepo struct {
	db *sql.DB
}

func NewPOSRepo(db *sql.DB) *POSRepo {
	return &POSRepo{db: db}
}

const createPOSOrderSQL = `
INSERT INTO pos_orders (
    id, tenant_id, order_number, customer_id, warehouse_id,
    subtotal, tax_amount, discount_amount, total_amount,
    payment_method, payment_amount, change_amount,
    sale_mode, status, sales_order_id, sales_invoice_id, cashier_id,
    created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9,
    $10, $11, $12,
    $13, $14, $15, $16, $17,
    $18, $19
)`

const createPOSOrderItemSQL = `
INSERT INTO pos_order_items (
    id, tenant_id, pos_order_id, product_id, product_name, sku,
    quantity, price, discount, subtotal, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

func (r *POSRepo) CreateOrder(ctx context.Context, order *domain.POSOrder, items []domain.POSOrderItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("POSRepo.CreateOrder: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, order.TenantID); err != nil {
		return fmt.Errorf("POSRepo.CreateOrder: set tenant: %w", err)
	}

	_, err = tx.ExecContext(ctx, createPOSOrderSQL,
		order.ID, order.TenantID, order.OrderNumber,
		ptrToNullUUID(order.CustomerID), ptrToNullUUID(order.WarehouseID),
		order.Subtotal, order.TaxAmount, order.DiscountAmount, order.TotalAmount,
		order.PaymentMethod, order.PaymentAmount, order.ChangeAmount,
		order.SaleMode, order.Status,
		ptrToNullUUID(order.SalesOrderID), ptrToNullUUID(order.SalesInvoiceID), ptrToNullUUID(order.CashierID),
		order.CreatedAt, order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("POSRepo.CreateOrder: exec order: %w", err)
	}

	for _, item := range items {
		_, err = tx.ExecContext(ctx, createPOSOrderItemSQL,
			item.ID, item.TenantID, item.POSOrderID, item.ProductID, item.ProductName, item.SKU,
			item.Quantity, item.Price, item.Discount, item.Subtotal, item.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("POSRepo.CreateOrder: exec item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("POSRepo.CreateOrder: commit: %w", err)
	}
	return nil
}

func (r *POSRepo) GetOrderByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.POSOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("POSRepo.GetOrderByID: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("POSRepo.GetOrderByID: set tenant: %w", err)
	}

	query := `
SELECT 
    o.id, o.tenant_id, o.order_number, o.customer_id, COALESCE(c.name, 'Pelanggan Umum'),
    o.warehouse_id, COALESCE(w.name, 'Gudang Utama'),
    o.subtotal, o.tax_amount, o.discount_amount, o.total_amount,
    o.payment_method, o.payment_amount, o.change_amount,
    o.sale_mode, o.status, o.sales_order_id, o.sales_invoice_id, o.cashier_id,
    o.created_at, o.updated_at
FROM pos_orders o
LEFT JOIN customers c ON c.id = o.customer_id AND c.tenant_id = o.tenant_id
LEFT JOIN warehouses w ON w.id = o.warehouse_id AND w.tenant_id = o.tenant_id
WHERE o.id = $1 AND o.tenant_id = $2`

	var o domain.POSOrder
	var custID, whID, soID, siID, cashierID sql.NullString
	if err := tx.QueryRowContext(ctx, query, id, tenantID).Scan(
		&o.ID, &o.TenantID, &o.OrderNumber, &custID, &o.CustomerName,
		&whID, &o.WarehouseName,
		&o.Subtotal, &o.TaxAmount, &o.DiscountAmount, &o.TotalAmount,
		&o.PaymentMethod, &o.PaymentAmount, &o.ChangeAmount,
		&o.SaleMode, &o.Status, &soID, &siID, &cashierID,
		&o.CreatedAt, &o.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("POSRepo.GetOrderByID: scan order: %w", err)
	}
	o.CustomerID = nullUUIDToPtr(custID)
	o.WarehouseID = nullUUIDToPtr(whID)
	o.SalesOrderID = nullUUIDToPtr(soID)
	o.SalesInvoiceID = nullUUIDToPtr(siID)
	o.CashierID = nullUUIDToPtr(cashierID)

	itemsQuery := `
SELECT id, tenant_id, pos_order_id, product_id, product_name, sku, quantity, price, discount, subtotal, created_at
FROM pos_order_items
WHERE pos_order_id = $1 AND tenant_id = $2
ORDER BY created_at ASC`

	rows, err := tx.QueryContext(ctx, itemsQuery, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("POSRepo.GetOrderByID: query items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var it domain.POSOrderItem
		if err := rows.Scan(
			&it.ID, &it.TenantID, &it.POSOrderID, &it.ProductID, &it.ProductName, &it.SKU,
			&it.Quantity, &it.Price, &it.Discount, &it.Subtotal, &it.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("POSRepo.GetOrderByID: scan item: %w", err)
		}
		o.Items = append(o.Items, it)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("POSRepo.GetOrderByID: commit: %w", err)
	}
	return &o, nil
}

func (r *POSRepo) ListOrders(ctx context.Context, tenantID uuid.UUID, limit int) ([]domain.POSOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("POSRepo.ListOrders: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("POSRepo.ListOrders: set tenant: %w", err)
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
SELECT 
    o.id, o.tenant_id, o.order_number, o.customer_id, COALESCE(c.name, 'Pelanggan Umum'),
    o.warehouse_id, COALESCE(w.name, 'Gudang Utama'),
    o.subtotal, o.tax_amount, o.discount_amount, o.total_amount,
    o.payment_method, o.payment_amount, o.change_amount,
    o.sale_mode, o.status, o.sales_order_id, o.sales_invoice_id, o.cashier_id,
    o.created_at, o.updated_at
FROM pos_orders o
LEFT JOIN customers c ON c.id = o.customer_id AND c.tenant_id = o.tenant_id
LEFT JOIN warehouses w ON w.id = o.warehouse_id AND w.tenant_id = o.tenant_id
WHERE o.tenant_id = $1
ORDER BY o.created_at DESC
LIMIT $2`

	rows, err := tx.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("POSRepo.ListOrders: query: %w", err)
	}
	defer rows.Close()

	var orders []domain.POSOrder
	for rows.Next() {
		var o domain.POSOrder
		var custID, whID, soID, siID, cashierID sql.NullString
		if err := rows.Scan(
			&o.ID, &o.TenantID, &o.OrderNumber, &custID, &o.CustomerName,
			&whID, &o.WarehouseName,
			&o.Subtotal, &o.TaxAmount, &o.DiscountAmount, &o.TotalAmount,
			&o.PaymentMethod, &o.PaymentAmount, &o.ChangeAmount,
			&o.SaleMode, &o.Status, &soID, &siID, &cashierID,
			&o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("POSRepo.ListOrders: scan: %w", err)
		}
		o.CustomerID = nullUUIDToPtr(custID)
		o.WarehouseID = nullUUIDToPtr(whID)
		o.SalesOrderID = nullUUIDToPtr(soID)
		o.SalesInvoiceID = nullUUIDToPtr(siID)
		o.CashierID = nullUUIDToPtr(cashierID)
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("POSRepo.ListOrders: rows: %w", err)
	}

	// Bulk-load line items for every order in this page so history cards and
	// receipt reprints always have the full item breakdown (not just totals).
	if len(orders) > 0 {
		idStrings := make([]string, len(orders))
		idxByOrder := make(map[uuid.UUID]int, len(orders))
		for i, o := range orders {
			idStrings[i] = o.ID.String()
			idxByOrder[o.ID] = i
		}

		itemsQuery := `
SELECT id, tenant_id, pos_order_id, product_id, product_name, sku, quantity, price, discount, subtotal, created_at
FROM pos_order_items
WHERE tenant_id = $1 AND pos_order_id = ANY($2)
ORDER BY created_at ASC`

		itemRows, err := tx.QueryContext(ctx, itemsQuery, tenantID, pq.Array(idStrings))
		if err != nil {
			return nil, fmt.Errorf("POSRepo.ListOrders: query items: %w", err)
		}
		defer itemRows.Close()

		for itemRows.Next() {
			var it domain.POSOrderItem
			if err := itemRows.Scan(
				&it.ID, &it.TenantID, &it.POSOrderID, &it.ProductID, &it.ProductName, &it.SKU,
				&it.Quantity, &it.Price, &it.Discount, &it.Subtotal, &it.CreatedAt,
			); err != nil {
				return nil, fmt.Errorf("POSRepo.ListOrders: scan item: %w", err)
			}
			if idx, ok := idxByOrder[it.POSOrderID]; ok {
				orders[idx].Items = append(orders[idx].Items, it)
			}
		}
		if err := itemRows.Err(); err != nil {
			return nil, fmt.Errorf("POSRepo.ListOrders: item rows: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("POSRepo.ListOrders: commit: %w", err)
	}
	return orders, nil
}
