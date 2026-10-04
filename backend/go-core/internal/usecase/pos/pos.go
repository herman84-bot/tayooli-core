package pos

import (
	"context"
	"fmt"
	"math/rand"
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

func generatePOSNumber(prefix string) string {
	now := time.Now().UTC()
	return fmt.Sprintf("%s-%s-%04d", prefix, now.Format("20060102"), rand.Intn(9000)+1000)
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

	// 1. Validate items and stock
	for _, item := range req.Items {
		if item.ProductID == uuid.Nil || item.Quantity.LessThanOrEqual(decimal.Zero) {
			return nil, domain.ErrInvalidInput
		}

		p, err := u.productRepo.GetByID(ctx, tenantID, item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("produk tidak ditemukan: %w", err)
		}

		// Check stock in inventory
		if u.inventoryRepo != nil {
			invs, err := u.inventoryRepo.GetByProductID(ctx, tenantID, item.ProductID)
			if err == nil && len(invs) > 0 {
				var totalStock float64
				for _, inv := range invs {
					totalStock += inv.Quantity
				}
				stockDec := decimal.NewFromFloat(totalStock)
				if stockDec.LessThan(item.Quantity) {
					return nil, fmt.Errorf("%w: SKU %s stok tersedia %.2f, diminta %s",
						domain.ErrInsufficientStock, p.SKU, totalStock, item.Quantity.String())
				}

				// Deduct from the first inventory row with available stock
				reqQtyFloat, _ := item.Quantity.Float64()
				for _, inv := range invs {
					if inv.Quantity <= 0 {
						continue
					}
					deduct := reqQtyFloat
					if inv.Quantity < deduct {
						deduct = inv.Quantity
					}
					_ = u.inventoryRepo.UpdateQuantity(ctx, tenantID, inv.ID, -deduct)
					reqQtyFloat -= deduct
					if reqQtyFloat <= 0 {
						break
					}
				}
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

		// Record in immutable stock movement ledger if WMS repo is available
		if u.wmsRepo != nil {
			var locID uuid.UUID
			// Find a default internal location
			locs, err := u.wmsRepo.ListLocations(ctx, tenantID, req.WarehouseID)
			if err == nil && len(locs) > 0 {
				locID = locs[0].ID
			} else {
				locID = uuid.New()
			}

			mov := &domain.StockMovement{
				ID:               uuid.New(),
				TenantID:         tenantID,
				MovementNumber:   generatePOSNumber("MV-POS"),
				ProductID:        item.ProductID,
				SourceLocationID: locID,
				DestLocationID:   locID,
				Quantity:         item.Quantity,
				UnitCost:         itemPrice,
				Status:           domain.StockMovementStatusDone,
				ReferenceType:    "POS_SALE",
				ReferenceID:      orderID,
				ExecutedBy:       &userID,
				CreatedAt:        time.Now().UTC(),
			}
			_ = u.wmsRepo.CreateStockMovement(ctx, mov)
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

	// 3. Create Sales Order
	var salesOrderID *uuid.UUID
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
		if err := u.salesOrderRepo.Create(ctx, so); err == nil {
			salesOrderID = &so.ID
		}
	}

	// 4. Create Sales Invoice
	var salesInvoiceID *uuid.UUID
	if u.salesInvRepo != nil && salesOrderID != nil {
		si := &domain.SalesInvoice{
			ID:            uuid.New(),
			TenantID:      tenantID,
			SalesOrderID:  *salesOrderID,
			InvoiceNumber: generatePOSNumber("INV-POS"),
			Amount:        grandTotal,
			Status:        "PAID",
			DueDate:       time.Now().UTC(),
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}
		if err := u.salesInvRepo.Create(ctx, si); err == nil {
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

	if err := u.posRepo.CreateOrder(ctx, posOrder, lineItems); err != nil {
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
