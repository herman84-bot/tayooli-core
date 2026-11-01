package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

type POSRepo struct {
	db  *sql.DB
	wms *WMSRepo // shares the pool; used for FEFO stock deduction inside CreateSale
}

func NewPOSRepo(db *sql.DB) *POSRepo {
	return &POSRepo{db: db, wms: NewWMSRepo(db)}
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

// CreateSale writes a whole POS checkout in ONE transaction: legacy inventory
// deductions, WMS ledger deductions (FEFO, expired batches blocked), the
// optional sales order and invoice, and the POS order with its items. Any
// failure rolls everything back, so stock never leaves for an unrecorded sale.
// Locks are taken in POSSale.LockOrder() order to avoid deadlocks between
// concurrent carts.
func (r *POSRepo) CreateSale(ctx context.Context, sale *domain.POSSale) error {
	if sale == nil || sale.Order == nil {
		return domain.ErrInvalidInput
	}
	tenantID := sale.Order.TenantID
	sale.LockOrder()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("POSRepo.CreateSale: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("POSRepo.CreateSale: set tenant: %w", err)
	}

	// 1. Legacy inventory rows (locked FOR UPDATE, checked, then deducted).
	for _, d := range sale.InventoryDeductions {
		if err := deductLegacyInventoryTx(ctx, tx, tenantID, d); err != nil {
			return err
		}
	}

	// 2. WMS ledger with FEFO batch allocation.
	for _, d := range sale.StockDeductions {
		mov := d.Movement
		if err := r.wms.deductWarehouseStockTx(ctx, tx, tenantID, d.WarehouseID, d.ProductID, d.Quantity, &mov); err != nil {
			return domain.WrapPOSStockError(d.SKU, err)
		}
	}

	// 3. Sales order and invoice (accounting link of the sale).
	if so := sale.SalesOrder; so != nil {
		if _, err := tx.ExecContext(ctx, createSalesOrder, so.ID, so.TenantID, so.CustomerID, so.OrderNumber, so.TotalAmount, so.Status, so.CreatedAt, so.UpdatedAt); err != nil {
			return fmt.Errorf("POSRepo.CreateSale: sales order: %w", err)
		}
		if si := sale.SalesInvoice; si != nil {
			if _, err := tx.ExecContext(ctx, createSalesInvoice, si.ID, si.TenantID, si.SalesOrderID, si.InvoiceNumber, si.Amount, si.Status, si.DueDate, si.CreatedAt, si.UpdatedAt); err != nil {
				return fmt.Errorf("POSRepo.CreateSale: sales invoice: %w", err)
			}
		}
	}

	// 4. POS order and items.
	if err := insertPOSOrderTx(ctx, tx, sale.Order, sale.Items); err != nil {
		return fmt.Errorf("POSRepo.CreateSale: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("POSRepo.CreateSale: commit: %w", err)
	}
	return nil
}

// deductLegacyInventoryTx removes d.Quantity from the product's legacy
// inventory rows, oldest first. Products without rows are skipped (as before);
// if the rows cannot cover the quantity the whole sale is refused.
func deductLegacyInventoryTx(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, d domain.POSInventoryDeduction) error {
	rows, err := tx.QueryContext(ctx, `
SELECT id, quantity FROM inventory
WHERE tenant_id = $1 AND product_id = $2
ORDER BY created_at ASC, id ASC
FOR UPDATE`, tenantID, d.ProductID)
	if err != nil {
		return fmt.Errorf("POSRepo.CreateSale: lock inventory: %w", err)
	}
	type invRow struct {
		id  uuid.UUID
		qty decimal.Decimal
	}
	var list []invRow
	total := decimal.Zero
	for rows.Next() {
		var ir invRow
		if err := rows.Scan(&ir.id, &ir.qty); err != nil {
			rows.Close()
			return fmt.Errorf("POSRepo.CreateSale: scan inventory: %w", err)
		}
		list = append(list, ir)
		total = total.Add(ir.qty)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("POSRepo.CreateSale: inventory rows: %w", err)
	}
	rows.Close()

	if len(list) == 0 {
		return nil
	}
	if total.LessThan(d.Quantity) {
		return domain.LegacyInsufficientStockError(d.SKU, total, d.Quantity)
	}
	remaining := d.Quantity
	for _, ir := range list {
		if !remaining.IsPositive() {
			break
		}
		if !ir.qty.IsPositive() {
			continue
		}
		take := decimal.Min(ir.qty, remaining)
		if _, err := tx.ExecContext(ctx, updateInventoryQuantity, take.Neg(), ir.id, tenantID); err != nil {
			return fmt.Errorf("POSRepo.CreateSale: deduct inventory: %w", err)
		}
		remaining = remaining.Sub(take)
	}
	return nil
}

func (r *POSRepo) CreateOrder(ctx context.Context, order *domain.POSOrder, items []domain.POSOrderItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("POSRepo.CreateOrder: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := setTenantLocally(ctx, tx, order.TenantID); err != nil {
		return fmt.Errorf("POSRepo.CreateOrder: set tenant: %w", err)
	}
	if err := insertPOSOrderTx(ctx, tx, order, items); err != nil {
		return fmt.Errorf("POSRepo.CreateOrder: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("POSRepo.CreateOrder: commit: %w", err)
	}
	return nil
}

// insertPOSOrderTx inserts a POS order and its items inside the caller's transaction.
func insertPOSOrderTx(ctx context.Context, tx *sql.Tx, order *domain.POSOrder, items []domain.POSOrderItem) error {
	_, err := tx.ExecContext(ctx, createPOSOrderSQL,
		order.ID, order.TenantID, order.OrderNumber,
		ptrToNullUUID(order.CustomerID), ptrToNullUUID(order.WarehouseID),
		order.Subtotal, order.TaxAmount, order.DiscountAmount, order.TotalAmount,
		order.PaymentMethod, order.PaymentAmount, order.ChangeAmount,
		order.SaleMode, order.Status,
		ptrToNullUUID(order.SalesOrderID), ptrToNullUUID(order.SalesInvoiceID), ptrToNullUUID(order.CashierID),
		order.CreatedAt, order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("exec order: %w", err)
	}

	for _, item := range items {
		_, err = tx.ExecContext(ctx, createPOSOrderItemSQL,
			item.ID, item.TenantID, item.POSOrderID, item.ProductID, item.ProductName, item.SKU,
			item.Quantity, item.Price, item.Discount, item.Subtotal, item.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("exec item: %w", err)
		}
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
    COALESCE(NULLIF(u.full_name, ''), u.email, ''),
    o.created_at, o.updated_at
FROM pos_orders o
LEFT JOIN customers c ON c.id = o.customer_id AND c.tenant_id = o.tenant_id
LEFT JOIN warehouses w ON w.id = o.warehouse_id AND w.tenant_id = o.tenant_id
LEFT JOIN users u ON u.id = o.cashier_id AND u.tenant_id = o.tenant_id
WHERE o.id = $1 AND o.tenant_id = $2`

	var o domain.POSOrder
	var custID, whID, soID, siID, cashierID sql.NullString
	if err := tx.QueryRowContext(ctx, query, id, tenantID).Scan(
		&o.ID, &o.TenantID, &o.OrderNumber, &custID, &o.CustomerName,
		&whID, &o.WarehouseName,
		&o.Subtotal, &o.TaxAmount, &o.DiscountAmount, &o.TotalAmount,
		&o.PaymentMethod, &o.PaymentAmount, &o.ChangeAmount,
		&o.SaleMode, &o.Status, &soID, &siID, &cashierID, &o.CashierName,
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
    COALESCE(NULLIF(u.full_name, ''), u.email, ''),
    o.created_at, o.updated_at
FROM pos_orders o
LEFT JOIN customers c ON c.id = o.customer_id AND c.tenant_id = o.tenant_id
LEFT JOIN warehouses w ON w.id = o.warehouse_id AND w.tenant_id = o.tenant_id
LEFT JOIN users u ON u.id = o.cashier_id AND u.tenant_id = o.tenant_id
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
			&o.SaleMode, &o.Status, &soID, &siID, &cashierID, &o.CashierName,
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
