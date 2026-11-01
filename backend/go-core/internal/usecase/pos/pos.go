package pos

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type CheckoutItemRequest struct {
	ProductID uuid.UUID       `json:"product_id"`
	Quantity  decimal.Decimal `json:"qty"`
	Price     decimal.Decimal `json:"price"`
	Discount  decimal.Decimal `json:"discount"`
}

type CheckoutPaymentRequest struct {
	Method string          `json:"method"` // CASH, QRIS, DEBIT
	Amount decimal.Decimal `json:"amount"`
}

type CheckoutRequest struct {
	Items       []CheckoutItemRequest    `json:"items"`
	Payments    []CheckoutPaymentRequest `json:"payments"`
	Tax         decimal.Decimal          `json:"tax"`
	Discount    decimal.Decimal          `json:"discount"`
	CustomerID  *uuid.UUID               `json:"customer_id,omitempty"`
	WarehouseID *uuid.UUID               `json:"warehouse_id,omitempty"`
	SaleMode    string                   `json:"sale_mode"` // DIRECT, KONSINYASI
	// PaymentOrderID links this sale to a settled gateway payment (see
	// usecase/pospayment). Empty for cash sales and for tenants without a
	// gateway account. The HTTP layer verifies and consumes it before the
	// sale is written.
	PaymentOrderID string `json:"payment_order_id,omitempty"`
}

type CheckoutResponse struct {
	OrderNumber    string            `json:"order_number"`
	Total          decimal.Decimal   `json:"total"`
	Subtotal       decimal.Decimal   `json:"subtotal"`
	Tax            decimal.Decimal   `json:"tax"`
	Discount       decimal.Decimal   `json:"discount"`
	PaymentMethod  string            `json:"payment_method"`
	PaidAmount     decimal.Decimal   `json:"paid_amount"`
	Change         decimal.Decimal   `json:"change"`
	SalesOrderID   *uuid.UUID        `json:"sales_order_id,omitempty"`
	SalesInvoiceID *uuid.UUID        `json:"sales_invoice_id,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	Items          []domain.POSOrderItem `json:"items"`
}

type Usecase struct {
	posRepo        domain.POSRepository
	productRepo    domain.ProductRepository
	inventoryRepo  domain.InventoryRepository
	wmsRepo        domain.WMSRepository
	customerRepo   domain.CustomerRepository
	salesOrderRepo domain.SalesOrderRepository
	salesInvRepo   domain.SalesInvoiceRepository
}

func New(
	posRepo domain.POSRepository,
	productRepo domain.ProductRepository,
	inventoryRepo domain.InventoryRepository,
	wmsRepo domain.WMSRepository,
	customerRepo domain.CustomerRepository,
	salesOrderRepo domain.SalesOrderRepository,
	salesInvRepo domain.SalesInvoiceRepository,
) *Usecase {
	return &Usecase{
		posRepo:        posRepo,
		productRepo:    productRepo,
		inventoryRepo:  inventoryRepo,
		wmsRepo:        wmsRepo,
		customerRepo:   customerRepo,
		salesOrderRepo: salesOrderRepo,
		salesInvRepo:   salesInvRepo,
	}
}

// posNumberAlphabet is Crockford base32 (no I, L, O, U) so a number read aloud
// or copied from a receipt is not mistaken for 1 or 0.
const posNumberAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// posNumberRandLen random characters give 50 bits per number. The previous
// 4-digit suffix allowed only 9,000 numbers per day, so busy tills hit the
// unique constraints on pos_orders/stock_movements and checkout failed with
// HTTP 500. At 10,000 sales a day the collision chance is now ~4e-8.
const posNumberRandLen = 10

// generatePOSNumber returns "<prefix>-YYYYMMDD-<10 random base32 chars>".
func generatePOSNumber(prefix string) string {
	now := time.Now().UTC()
	var buf [posNumberRandLen]byte
	_, _ = rand.Read(buf[:]) // crypto/rand.Read never fails (Go 1.24+)
	out := make([]byte, posNumberRandLen)
	for i, b := range buf {
		out[i] = posNumberAlphabet[b&31]
	}
	return fmt.Sprintf("%s-%s-%s", prefix, now.Format("20060102"), out)
}

func (u *Usecase) Checkout(ctx context.Context, tenantID, userID uuid.UUID, req CheckoutRequest) (*CheckoutResponse, error) {
	if len(req.Items) == 0 {
		return nil, domain.ErrInvalidInput
	}

	saleMode := req.SaleMode
	if saleMode == "" {
		saleMode = "DIRECT"
	}

	orderID := uuid.New()
	posOrderNumber := generatePOSNumber("POS")

	var subtotal decimal.Decimal
	var lineItems []domain.POSOrderItem
	// Nothing is written while the cart is validated: every write is collected
	// here and persisted atomically by posRepo.CreateSale at the end.
	sale := &domain.POSSale{}

	// Resolve the WMS warehouse and customer location once (read / idempotent).
	var whID, custLocID uuid.UUID
	if u.wmsRepo != nil {
		if req.WarehouseID != nil && *req.WarehouseID != uuid.Nil {
			whID = *req.WarehouseID
		} else {
			whs, err := u.wmsRepo.ListWarehouses(ctx, tenantID)
			if err == nil && len(whs) > 0 {
				whID = whs[0].ID
			}
		}
		if whID != uuid.Nil {
			custLoc, err := u.wmsRepo.GetOrCreateSystemLocation(ctx, tenantID, domain.LocationTypeCustomer)
			if err != nil {
				return nil, fmt.Errorf("pos: resolve customer location: %w", err)
			}
			custLocID = custLoc.ID
		}
	}

	// 1. Validate items and plan stock deductions
	for _, item := range req.Items {
		if item.ProductID == uuid.Nil || item.Quantity.LessThanOrEqual(decimal.Zero) {
			return nil, domain.ErrInvalidInput
		}

		p, err := u.productRepo.GetByID(ctx, tenantID, item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("produk tidak ditemukan: %w", err)
		}

		// Fast, read-only pre-check of legacy inventory. CreateSale re-checks
		// under row locks, so concurrent cashiers cannot oversell.
		if u.inventoryRepo != nil {
			invs, err := u.inventoryRepo.GetByProductID(ctx, tenantID, item.ProductID)
			if err == nil && len(invs) > 0 {
				total := decimal.Zero
				for _, inv := range invs {
					total = total.Add(decimal.NewFromFloat(inv.Quantity))
				}
				if total.LessThan(item.Quantity) {
					return nil, domain.LegacyInsufficientStockError(p.SKU, total, item.Quantity)
				}
				sale.InventoryDeductions = append(sale.InventoryDeductions, domain.POSInventoryDeduction{
					ProductID: item.ProductID, SKU: p.SKU, Quantity: item.Quantity,
				})
			}
		}

		itemPrice := item.Price
		if itemPrice.IsZero() {
			itemPrice = decimal.NewFromFloat(p.Price)
		}
		itemSubtotal := itemPrice.Mul(item.Quantity).Sub(item.Discount)
		if itemSubtotal.LessThan(decimal.Zero) {
			itemSubtotal = decimal.Zero
		}
		subtotal = subtotal.Add(itemSubtotal)

		lineItem := domain.POSOrderItem{
			ID:          uuid.New(),
			TenantID:    tenantID,
			POSOrderID:  orderID,
			ProductID:   item.ProductID,
			ProductName: p.Name,
			SKU:         p.SKU,
			Quantity:    item.Quantity,
			Price:       itemPrice,
			Discount:    item.Discount,
			Subtotal:    itemSubtotal,
			CreatedAt:   time.Now().UTC(),
		}
		lineItems = append(lineItems, lineItem)

		// Plan the immutable stock movement (FEFO per batch) for this line.
		if whID != uuid.Nil {
			sale.StockDeductions = append(sale.StockDeductions, domain.POSStockDeduction{
				WarehouseID: whID,
				ProductID:   item.ProductID,
				SKU:         p.SKU,
				Quantity:    item.Quantity,
				Movement: domain.StockMovement{
					ID:             uuid.New(),
					TenantID:       tenantID,
					MovementNumber: generatePOSNumber("MV-POS"),
					ProductID:      item.ProductID,
					DestLocationID: custLocID,
					Quantity:       item.Quantity,
					UnitCost:       itemPrice,
					Status:         domain.StockMovementStatusDone,
					ReferenceType:  domain.StockRefPOS,
					ReferenceID:    orderID,
					ExecutedBy:     &userID,
					CreatedAt:      time.Now().UTC(),
				},
			})
		}
	}

	grandTotal := subtotal.Add(req.Tax).Sub(req.Discount)
	if grandTotal.LessThan(decimal.Zero) {
		grandTotal = decimal.Zero
	}

	var paidAmount decimal.Decimal
	paymentMethod := "CASH"
	if len(req.Payments) > 0 {
		paymentMethod = req.Payments[0].Method
		for _, p := range req.Payments {
			paidAmount = paidAmount.Add(p.Amount)
		}
	} else {
		paidAmount = grandTotal
	}

	changeAmount := paidAmount.Sub(grandTotal)
	if changeAmount.LessThan(decimal.Zero) {
		changeAmount = decimal.Zero
	}

	// 2. Ensure customer exists (create Walk-In Customer if needed)
	var custID *uuid.UUID = req.CustomerID
	if custID == nil || *custID == uuid.Nil {
		if u.customerRepo != nil {
			// Find existing walk-in or create
			custs, err := u.customerRepo.List(ctx, tenantID)
			if err == nil && len(custs) > 0 {
				custID = &custs[0].ID
			} else {
				newCust := &domain.Customer{
					ID:        uuid.New(),
					TenantID:  tenantID,
					Name:      "Pelanggan Umum (POS)",
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				}
				if err := u.customerRepo.Create(ctx, newCust); err == nil {
					custID = &newCust.ID
				}
			}
		}
	}

	// 3. Plan Sales Order & Invoice (written in the same transaction as the sale)
	var salesOrderID, salesInvoiceID *uuid.UUID
	if u.salesOrderRepo != nil && custID != nil {
		so := &domain.SalesOrder{
			ID:          uuid.New(),
			TenantID:    tenantID,
			CustomerID:  *custID,
			OrderNumber: generatePOSNumber("SO-POS"),
			TotalAmount: grandTotal,
			Status:      "CONFIRMED",
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}
		sale.SalesOrder = so
		salesOrderID = &so.ID

		// 4. Sales Invoice
		if u.salesInvRepo != nil {
			si := &domain.SalesInvoice{
				ID:            uuid.New(),
				TenantID:      tenantID,
				SalesOrderID:  so.ID,
				InvoiceNumber: generatePOSNumber("INV-POS"),
				Amount:        grandTotal,
				Status:        "PAID",
				DueDate:       time.Now().UTC(),
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
			}
			sale.SalesInvoice = si
			salesInvoiceID = &si.ID
		}
	}

	// 5. Create POS Order & Items
	posOrder := &domain.POSOrder{
		ID:             orderID,
		TenantID:       tenantID,
		OrderNumber:    posOrderNumber,
		CustomerID:     custID,
		WarehouseID:    req.WarehouseID,
		Subtotal:       subtotal,
		TaxAmount:      req.Tax,
		DiscountAmount: req.Discount,
		TotalAmount:    grandTotal,
		PaymentMethod:  paymentMethod,
		PaymentAmount:  paidAmount,
		ChangeAmount:   changeAmount,
		SaleMode:       saleMode,
		Status:         "COMPLETED",
		SalesOrderID:   salesOrderID,
		SalesInvoiceID: salesInvoiceID,
		CashierID:      &userID,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	// 6. Persist everything atomically: stock deductions, sales order/invoice
	// and the POS order commit together or not at all.
	sale.Order = posOrder
	sale.Items = lineItems
	if err := u.posRepo.CreateSale(ctx, sale); err != nil {
		if errors.Is(err, domain.ErrInsufficientStock) || errors.Is(err, domain.ErrExpiredStockBlocked) {
			return nil, err
		}
		return nil, fmt.Errorf("gagal menyimpan pesanan POS: %w", err)
	}

	return &CheckoutResponse{
		OrderNumber:    posOrderNumber,
		Total:          grandTotal,
		Subtotal:       subtotal,
		Tax:            req.Tax,
		Discount:       req.Discount,
		PaymentMethod:  paymentMethod,
		PaidAmount:     paidAmount,
		Change:         changeAmount,
		SalesOrderID:   salesOrderID,
		SalesInvoiceID: salesInvoiceID,
		CreatedAt:      posOrder.CreatedAt,
		Items:          lineItems,
	}, nil
}

func (u *Usecase) ListOrders(ctx context.Context, tenantID uuid.UUID, limit int) ([]domain.POSOrder, error) {
	return u.posRepo.ListOrders(ctx, tenantID, limit)
}

func (u *Usecase) GetOrderByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.POSOrder, error) {
	return u.posRepo.GetOrderByID(ctx, tenantID, id)
}
