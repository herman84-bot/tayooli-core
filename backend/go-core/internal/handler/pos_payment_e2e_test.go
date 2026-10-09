package handler_test

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/midtrans"
	mw "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	posUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/pos"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/pospayment"
)

// This file is the end-to-end proof of the POS QRIS flow with demo data: real
// HTTP handlers + the real POS and payment usecases, with only the outermost
// boundaries (database rows, gateway HTTP) faked. It runs without Postgres,
// without Docker and without Midtrans credentials.

var (
	e2eTenant = uuid.MustParse("660aaa9b-442a-47a5-9e34-0a1f64eee2e0")
	e2eOther  = uuid.MustParse("660aaa9b-442a-47a5-9e34-0a1f64eee2e1")
	e2eUser   = uuid.MustParse("770bbb9b-552b-48b6-8f45-1b2c75eef2e1")
)

// ── request helpers ──────────────────────────────────────────────────────────

// tenantCtx injects what TenantMiddleware would put in the context.
func tenantCtx(r *http.Request, tenantID, userID uuid.UUID) *http.Request {
	ctx := context.WithValue(r.Context(), mw.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, mw.UserIDKey, userID)
	return r.WithContext(ctx)
}

// withRoute injects a chi URL parameter (what the router does for {orderId}).
func withRoute(r *http.Request, param, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(param, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// ── payment repo fake (mirrors PaymentGatewayRepository + RLS scoping) ───────

type e2ePaymentRepo struct {
	configs      map[string]*domain.TenantPaymentConfig
	transactions map[string]*domain.PaymentTransaction
	// ignoreTenantScope simulates a repo that leaks rows across tenants so the
	// handlers' own tenant checks are proven to hold on their own.
	ignoreTenantScope bool
}

func newE2EPaymentRepo() *e2ePaymentRepo {
	return &e2ePaymentRepo{
		configs:      map[string]*domain.TenantPaymentConfig{},
		transactions: map[string]*domain.PaymentTransaction{},
	}
}

func (f *e2ePaymentRepo) GetConfig(ctx context.Context, tenantID uuid.UUID, provider string) (*domain.TenantPaymentConfig, error) {
	cfg, ok := f.configs[tenantID.String()+"|"+provider]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return cfg, nil
}

func (f *e2ePaymentRepo) UpsertConfig(ctx context.Context, cfg *domain.TenantPaymentConfig) error {
	f.configs[cfg.TenantID.String()+"|"+cfg.Provider] = cfg
	return nil
}

func (f *e2ePaymentRepo) CreateTransaction(ctx context.Context, tx *domain.PaymentTransaction) error {
	cp := *tx
	f.transactions[tx.OrderID] = &cp
	return nil
}

func (f *e2ePaymentRepo) GetTransactionByOrderID(ctx context.Context, tenantID uuid.UUID, orderID string) (*domain.PaymentTransaction, error) {
	tx, ok := f.transactions[orderID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if !f.ignoreTenantScope && tx.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	return tx, nil
}

func (f *e2ePaymentRepo) UpdateTransactionStatus(ctx context.Context, tenantID uuid.UUID, orderID, status string, completedAt *time.Time) error {
	tx, ok := f.transactions[orderID]
	if !ok {
		return domain.ErrNotFound
	}
	if !f.ignoreTenantScope && tx.TenantID != tenantID {
		return domain.ErrNotFound
	}
	tx.Status = status
	if completedAt != nil {
		tx.CompletedAt = completedAt
	}
	return nil
}

func (f *e2ePaymentRepo) ConsumeTransaction(ctx context.Context, tenantID uuid.UUID, orderID string) error {
	tx, ok := f.transactions[orderID]
	if !ok {
		return domain.ErrNotFound
	}
	if !f.ignoreTenantScope && tx.TenantID != tenantID {
		return domain.ErrNotFound
	}
	if tx.Status != "completed" {
		return domain.ErrNotFound
	}
	tx.Status = "consumed"
	return nil
}

// ── gateway fake ─────────────────────────────────────────────────────────────

type e2eGateway struct {
	statusResp *midtrans.TransactionStatus
	statusErr  error
	chargeErr  error
	// chargeCalls/statusCalls let a test assert whether the network was touched.
	chargeCalls int
	statusCalls int
}

func (g *e2eGateway) CreateChargeQRIS(ctx context.Context, req midtrans.CreateChargeQRISRequest) (*midtrans.CreateChargeQRISResponse, error) {
	g.chargeCalls++
	if g.chargeErr != nil {
		return nil, g.chargeErr
	}
	return &midtrans.CreateChargeQRISResponse{
		OrderID:   req.OrderID,
		QRString:  "00020101021226670014ID.CO.QRIS.WWW0118" + req.OrderID,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}, nil
}

func (g *e2eGateway) VerifyTransactionStatus(ctx context.Context, orderID string) (*midtrans.TransactionStatus, error) {
	g.statusCalls++
	if g.statusErr != nil {
		return nil, g.statusErr
	}
	if g.statusResp != nil {
		return g.statusResp, nil
	}
	return &midtrans.TransactionStatus{Status: "pending"}, nil
}

// ── POS domain fakes (same shapes as internal/usecase/pos/pos_test.go) ───────

type e2ePOSRepo struct {
	orders []domain.POSOrder
	items  map[uuid.UUID][]domain.POSOrderItem
}

func (m *e2ePOSRepo) CreateOrder(ctx context.Context, order *domain.POSOrder, items []domain.POSOrderItem) error {
	m.orders = append(m.orders, *order)
	if m.items == nil {
		m.items = map[uuid.UUID][]domain.POSOrderItem{}
	}
	m.items[order.ID] = items
	return nil
}

func (m *e2ePOSRepo) GetOrderByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.POSOrder, error) {
	for _, o := range m.orders {
		if o.ID == id && o.TenantID == tenantID {
			cpy := o
			cpy.Items = m.items[id]
			return &cpy, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *e2ePOSRepo) ListOrders(ctx context.Context, tenantID uuid.UUID, limit int) ([]domain.POSOrder, error) {
	var list []domain.POSOrder
	for _, o := range m.orders {
		if o.TenantID == tenantID {
			list = append(list, o)
		}
	}
	return list, nil
}

type e2eProductRepo struct{ products map[uuid.UUID]*domain.Product }

func (m *e2eProductRepo) Create(ctx context.Context, p *domain.Product) error { return nil }
func (m *e2eProductRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error) {
	p, ok := m.products[id]
	if !ok || p.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	return p, nil
}
func (m *e2eProductRepo) GetBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*domain.Product, error) {
	return nil, domain.ErrNotFound
}
func (m *e2eProductRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Product, error) {
	return nil, nil
}
func (m *e2eProductRepo) ListByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]domain.Product, error) {
	return nil, nil
}
func (m *e2eProductRepo) Update(ctx context.Context, p *domain.Product) error          { return nil }
func (m *e2eProductRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error     { return nil }
func (m *e2eProductRepo) HasMovementsOrStock(ctx context.Context, tenantID, id uuid.UUID) (bool, error) {
	return false, nil
}
func (m *e2eProductRepo) ListInventoryFromWMS(ctx context.Context, tenantID uuid.UUID) ([]domain.InventoryItemWithProduct, error) {
	return nil, nil
}

type e2eInventoryRepo struct {
	stocks map[uuid.UUID]*domain.Inventory
	// deducted accumulates the total quantity removed, so a test can prove
	// stock only moved once even if checkout is retried.
	deducted float64
}

func (m *e2eInventoryRepo) Create(ctx context.Context, i *domain.Inventory) error { return nil }
func (m *e2eInventoryRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Inventory, error) {
	return nil, domain.ErrNotFound
}
func (m *e2eInventoryRepo) GetByProductID(ctx context.Context, tenantID, productID uuid.UUID) ([]domain.Inventory, error) {
	var list []domain.Inventory
	for _, inv := range m.stocks {
		if inv.TenantID == tenantID && inv.ProductID == productID {
			list = append(list, *inv)
		}
	}
	return list, nil
}
func (m *e2eInventoryRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Inventory, error) {
	return nil, nil
}
func (m *e2eInventoryRepo) UpdateQuantity(ctx context.Context, tenantID, id uuid.UUID, delta float64) error {
	if inv, ok := m.stocks[id]; ok {
		inv.Quantity += delta
		if delta < 0 {
			m.deducted += -delta
		}
	}
	return nil
}

type e2eSalesOrderRepo struct{ orders []domain.SalesOrder }

func (m *e2eSalesOrderRepo) Create(ctx context.Context, so *domain.SalesOrder) error {
	m.orders = append(m.orders, *so)
	return nil
}
func (m *e2eSalesOrderRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SalesOrder, error) {
	return nil, domain.ErrNotFound
}
func (m *e2eSalesOrderRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SalesOrder, error) {
	return m.orders, nil
}

type e2eSalesInvoiceRepo struct{ invoices []domain.SalesInvoice }

func (m *e2eSalesInvoiceRepo) Create(ctx context.Context, si *domain.SalesInvoice) error {
	m.invoices = append(m.invoices, *si)
	return nil
}
func (m *e2eSalesInvoiceRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SalesInvoice, error) {
	return nil, domain.ErrNotFound
}
func (m *e2eSalesInvoiceRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SalesInvoice, error) {
	return m.invoices, nil
}

type e2eCustomerRepo struct{ customers []domain.Customer }

func (m *e2eCustomerRepo) Create(ctx context.Context, c *domain.Customer) error {
	m.customers = append(m.customers, *c)
	return nil
}
func (m *e2eCustomerRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Customer, error) {
	return nil, domain.ErrNotFound
}
func (m *e2eCustomerRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Customer, error) {
	return m.customers, nil
}

// ── harness ──────────────────────────────────────────────────────────────────

type e2eEnv struct {
	handler  *handler.POSHandler
	payments *e2ePaymentRepo
	gateway  *e2eGateway
	inventory *e2eInventoryRepo
	posRepo  *e2ePOSRepo
	productID uuid.UUID
	tenantID uuid.UUID
	userID   uuid.UUID
}

// buildE2E wires the real handler + real usecases over faked boundaries.
func buildE2E(t *testing.T, withCredentials bool) *e2eEnv {
	t.Helper()

	payments := newE2EPaymentRepo()
	gateway := &e2eGateway{}
	paymentUC := pospayment.New(payments, func(serverKey, clientKey string, isProduction bool) pospayment.Client {
		return gateway
	}, "https://tayooli.my.id")

	productID := uuid.New()
	productRepo := &e2eProductRepo{products: map[uuid.UUID]*domain.Product{
		productID: {ID: productID, TenantID: e2eTenant, Name: "Demo SKU", SKU: "DEMO-001", Price: 10000},
	}}
	invID := uuid.New()
	inventory := &e2eInventoryRepo{stocks: map[uuid.UUID]*domain.Inventory{
		invID: {ID: invID, TenantID: e2eTenant, ProductID: productID, Quantity: 100},
	}}
	posRepo := &e2ePOSRepo{items: map[uuid.UUID][]domain.POSOrderItem{}}

	// wmsRepo is nil, exactly like the existing POS usecase test.
	posUsecase := posUC.New(posRepo, productRepo, inventory, nil, &e2eCustomerRepo{}, &e2eSalesOrderRepo{}, &e2eSalesInvoiceRepo{})
	h := handler.NewPOSHandlerWithPayments(posUsecase, paymentUC)

	if withCredentials {
		key := "SB-Mid-server-e2e-key"
		if err := payments.UpsertConfig(context.Background(), &domain.TenantPaymentConfig{
			TenantID: e2eTenant, Provider: pospayment.Provider, ServerKey: &key, IsActive: true,
		}); err != nil {
			t.Fatalf("seed config: %v", err)
		}
	}

	return &e2eEnv{
		handler: h, payments: payments, gateway: gateway, inventory: inventory,
		posRepo: posRepo, productID: productID, tenantID: e2eTenant, userID: e2eUser,
	}
}

func (e *e2eEnv) createQRIS(t *testing.T, amount int64) (orderID string, demo bool) {
	t.Helper()
	body := mustJSON(t, map[string]any{"amount": amount, "method": "QRIS"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pos/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = tenantCtx(req, e.tenantID, e.userID)
	rr := httptest.NewRecorder()
	e.handler.CreatePayment(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("CreatePayment: %d %s", rr.Code, rr.Body.String())
	}
	var charge struct {
		OrderID string `json:"order_id"`
		Demo    bool   `json:"demo"`
		QRStr   string `json:"qr_string"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &charge); err != nil {
		t.Fatalf("decode charge: %v (body=%s)", err, rr.Body.String())
	}
	if charge.OrderID == "" {
		t.Fatal("charge must carry an order_id")
	}
	if !charge.Demo && charge.QRStr == "" {
		t.Error("a real charge must carry a QR string")
	}
	return charge.OrderID, charge.Demo
}

func (e *e2eEnv) simulate(t *testing.T, orderID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pos/payments/"+orderID+"/simulate", nil)
	req = withRoute(tenantCtx(req, e.tenantID, e.userID), "orderId", orderID)
	rr := httptest.NewRecorder()
	e.handler.SimulatePayment(rr, req)
	return rr
}

func (e *e2eEnv) status(t *testing.T, orderID string) (string, bool, int) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pos/payments/"+orderID+"/status", nil)
	req = withRoute(tenantCtx(req, e.tenantID, e.userID), "orderId", orderID)
	rr := httptest.NewRecorder()
	e.handler.PaymentStatus(rr, req)
	if rr.Code != http.StatusOK {
		return "", false, rr.Code
	}
	var res struct {
		Status string `json:"status"`
		Demo   bool   `json:"demo"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode status: %v (body=%s)", err, rr.Body.String())
	}
	return res.Status, res.Demo, rr.Code
}

func (e *e2eEnv) checkout(t *testing.T, method string, amount int64, paymentOrderID string) *httptest.ResponseRecorder {
	t.Helper()
	payload := map[string]any{
		"items":     []any{map[string]any{"product_id": e.productID.String(), "qty": 1, "price": amount}},
		"payments":  []any{map[string]any{"method": method, "amount": amount}},
		"sale_mode": "DIRECT",
	}
	if paymentOrderID != "" {
		payload["payment_order_id"] = paymentOrderID
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(mustJSON(t, payload)))
	req.Header.Set("Content-Type", "application/json")
	req = tenantCtx(req, e.tenantID, e.userID)
	rr := httptest.NewRecorder()
	e.handler.Checkout(rr, req)
	return rr
}

// ── Tests ────────────────────────────────────────────────────────────────────

// The full demo journey: cart → QRIS intent → (customer pays) → settled →
// receipt issued → payment spent.
func TestPOSE2E_DemoScanToReceipt(t *testing.T) {
	e := buildE2E(t, false) // no credentials → demo mode
	const amount = int64(10000)

	orderID, demo := e.createQRIS(t, amount)
	if !demo {
		t.Fatal("tenant without credentials must get a demo charge")
	}
	if e.gateway.chargeCalls != 0 {
		t.Error("demo mode must not call the gateway")
	}

	// Nothing is paid yet: the status poll says pending and no receipt exists.
	if status, isDemo, code := e.status(t, orderID); code != http.StatusOK || status != "pending" || !isDemo {
		t.Fatalf("status = %q demo=%v code=%d, want pending demo 200", status, isDemo, code)
	}
	if rr := e.checkout(t, "QRIS", amount, orderID); rr.Code != http.StatusConflict {
		t.Fatalf("checkout before settlement = %d %s, want 409", rr.Code, rr.Body.String())
	}
	if len(e.posRepo.orders) != 0 {
		t.Fatal("an unsettled payment must not create a POS order")
	}
	if e.inventory.deducted != 0 {
		t.Fatalf("stock moved %v before settlement, want 0", e.inventory.deducted)
	}

	// The customer pays (simulated) and the same QR now funds the sale.
	if rr := e.simulate(t, orderID); rr.Code != http.StatusOK {
		t.Fatalf("simulate = %d %s", rr.Code, rr.Body.String())
	}
	rr := e.checkout(t, "QRIS", amount, orderID)
	if rr.Code != http.StatusCreated {
		t.Fatalf("checkout after settlement = %d %s", rr.Code, rr.Body.String())
	}
	var receipt struct {
		OrderNumber string `json:"order_number"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &receipt); err != nil {
		t.Fatalf("decode receipt: %v", err)
	}
	if receipt.OrderNumber == "" {
		t.Error("receipt must carry an order_number")
	}
	if e.payments.transactions[orderID].Status != "consumed" {
		t.Errorf("payment status = %q, want consumed", e.payments.transactions[orderID].Status)
	}
	if e.inventory.deducted != 1 {
		t.Errorf("stock deducted = %v, want 1", e.inventory.deducted)
	}
}

// One payment can never mint two sales, even past the gateway.
func TestPOSE2E_ReplayingASettledPaymentNeverPaysTwice(t *testing.T) {
	e := buildE2E(t, false)
	const amount = int64(25000)

	orderID, _ := e.createQRIS(t, amount)
	if rr := e.simulate(t, orderID); rr.Code != http.StatusOK {
		t.Fatalf("simulate = %d %s", rr.Code, rr.Body.String())
	}
	if rr := e.checkout(t, "QRIS", amount, orderID); rr.Code != http.StatusCreated {
		t.Fatalf("first checkout = %d %s", rr.Code, rr.Body.String())
	}
	// Second attempt with the very same order id.
	rr := e.checkout(t, "QRIS", amount, orderID)
	if rr.Code != http.StatusConflict {
		t.Fatalf("replay checkout = %d %s, want 409", rr.Code, rr.Body.String())
	}
	if len(e.posRepo.orders) != 1 {
		t.Errorf("POS orders = %d, want exactly 1", len(e.posRepo.orders))
	}
	if e.inventory.deducted != 1 {
		t.Errorf("stock deducted = %v, want 1 (never twice)", e.inventory.deducted)
	}
}

// A gateway tenant cannot hand-wave a QRIS sale: no settled payment → no receipt.
func TestPOSE2E_GatewayTenantCannotCheckoutQRISWithoutSettledPayment(t *testing.T) {
	e := buildE2E(t, true) // credentials configured

	rr := e.checkout(t, "QRIS", 10000, "")
	if rr.Code != http.StatusConflict {
		t.Fatalf("unverified QRIS checkout = %d %s, want 409", rr.Code, rr.Body.String())
	}
	if len(e.posRepo.orders) != 0 {
		t.Error("no POS order may be created without settlement")
	}
}

// Cash keeps working untouched, gateway or not.
func TestPOSE2E_CashIsUnaffected(t *testing.T) {
	for _, withCreds := range []bool{false, true} {
		e := buildE2E(t, withCreds)
		rr := e.checkout(t, "CASH", 10000, "")
		if rr.Code != http.StatusCreated {
			t.Fatalf("cash checkout (creds=%v) = %d %s", withCreds, rr.Code, rr.Body.String())
		}
		if e.inventory.deducted != 1 {
			t.Errorf("cash stock deducted = %v, want 1", e.inventory.deducted)
		}
	}
}

// Amount mismatch (cart ≠ what the gateway collected) is refused.
func TestPOSE2E_AmountMismatchIsRefused(t *testing.T) {
	e := buildE2E(t, false)
	orderID, _ := e.createQRIS(t, 15000)
	if rr := e.simulate(t, orderID); rr.Code != http.StatusOK {
		t.Fatalf("simulate = %d %s", rr.Code, rr.Body.String())
	}
	rr := e.checkout(t, "QRIS", 1, orderID) // cart claims 1 rupiah
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("mismatch checkout = %d %s, want 400", rr.Code, rr.Body.String())
	}
	if e.payments.transactions[orderID].Status != "completed" {
		t.Error("a mismatched claim must not consume the payment")
	}
}

// Demo simulation is refused the moment the tenant has real credentials, so it
// can never be used to fake a live payment.
func TestPOSE2E_SimulateRefusedForGatewayTenant(t *testing.T) {
	e := buildE2E(t, false)
	orderID, _ := e.createQRIS(t, 9000)

	// Tenant connects their own Midtrans account afterwards.
	key := "SB-Mid-server-live"
	if err := e.payments.UpsertConfig(context.Background(), &domain.TenantPaymentConfig{
		TenantID: e.tenantID, Provider: pospayment.Provider, ServerKey: &key, IsActive: true,
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	if rr := e.simulate(t, orderID); rr.Code != http.StatusForbidden {
		t.Fatalf("simulate for gateway tenant = %d %s, want 403", rr.Code, rr.Body.String())
	}
	if got := e.payments.transactions[orderID].Status; got != "pending" {
		t.Errorf("status = %q, want pending (demo must not settle)", got)
	}
}

// Another tenant cannot settle or spend this tenant's payment.
func TestPOSE2E_CrossTenantIsolation(t *testing.T) {
	e := buildE2E(t, false)
	orderID, _ := e.createQRIS(t, 11000)

	// Even if the store leaked the row, handlers must deny on ownership.
	e.payments.ignoreTenantScope = true
	defer func() { e.payments.ignoreTenantScope = false }()

	simReq := httptest.NewRequest(http.MethodPost, "/api/v1/pos/payments/"+orderID+"/simulate", nil)
	simReq = withRoute(tenantCtx(simReq, e2eOther, e2eUser), "orderId", orderID)
	simRR := httptest.NewRecorder()
	e.handler.SimulatePayment(simRR, simReq)
	if simRR.Code == http.StatusOK {
		t.Errorf("cross-tenant simulate succeeded: %d %s", simRR.Code, simRR.Body.String())
	}

	// And a checkout by the other tenant must not consume it either.
	body := mustJSON(t, map[string]any{
		"items":            []any{map[string]any{"product_id": e.productID.String(), "qty": 1, "price": 11000}},
		"payments":         []any{map[string]any{"method": "QRIS", "amount": 11000}},
		"payment_order_id": orderID,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(body))
	req = tenantCtx(req, e2eOther, e2eUser)
	rr := httptest.NewRecorder()
	e.handler.Checkout(rr, req)
	if rr.Code == http.StatusCreated {
		t.Errorf("cross-tenant checkout succeeded: %d %s", rr.Code, rr.Body.String())
	}
	if got := e.payments.transactions[orderID].Status; got != "pending" {
		t.Errorf("status = %q, want pending (untouched by the other tenant)", got)
	}
}

// A real (BYO) charge: the webhook is the source of truth, and the status poll
// re-verifies server-to-server so a missed webhook cannot strand a paid sale.
func TestPOSE2E_RealGatewayChargeSettlesViaWebhook(t *testing.T) {
	e := buildE2E(t, true)

	orderID, demo := e.createQRIS(t, 40000)
	if demo {
		t.Fatal("gateway tenant must receive a real charge")
	}
	if e.gateway.chargeCalls != 1 {
		t.Errorf("gateway charge calls = %d, want 1", e.gateway.chargeCalls)
	}
	if got := e.payments.transactions[orderID].Status; got != "pending" {
		t.Fatalf("status = %q, want pending", got)
	}

	// Midtrans notifies us (signed) that the QRIS was paid.
	webhook := handler.NewPaymentWebhookHandler(e.payments, func(serverKey, clientKey string, isProduction bool) handler.MidtransStatusChecker {
		return e.gateway
	})
	// The re-check must agree with the callback, so make the gateway settled.
	e.gateway.statusResp = &midtrans.TransactionStatus{Status: "settlement", Amount: 40000}

	payload := map[string]any{
		"order_id":           orderID,
		"transaction_status": "settlement",
		"status_code":        "200",
		"gross_amount":       "40000.00",
		"payment_type":       "qris",
		"signature_key":      fmt.Sprintf("%x", sha512.Sum512([]byte(orderID+"200"+"40000.00"+"SB-Mid-server-e2e-key"))),
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/midtrans", bytes.NewReader(mustJSON(t, payload)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	webhook.HandleMidtransWebhook(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("webhook = %d %s", rr.Code, rr.Body.String())
	}
	if got := e.payments.transactions[orderID].Status; got != "completed" {
		t.Fatalf("status after webhook = %q, want completed", got)
	}

	// Now the sale can be written and the payment is spent.
	if rr := e.checkout(t, "QRIS", 40000, orderID); rr.Code != http.StatusCreated {
		t.Fatalf("checkout = %d %s", rr.Code, rr.Body.String())
	}
	if got := e.payments.transactions[orderID].Status; got != "consumed" {
		t.Errorf("final status = %q, want consumed", got)
	}
}

// A forged/unsigned callback must never mark a payment paid.
func TestPOSE2E_UnsignedWebhookIsRejected(t *testing.T) {
	e := buildE2E(t, true)
	orderID, _ := e.createQRIS(t, 40000)

	webhook := handler.NewPaymentWebhookHandler(e.payments, func(serverKey, clientKey string, isProduction bool) handler.MidtransStatusChecker {
		return e.gateway
	})
	payload := map[string]any{
		"order_id":           orderID,
		"transaction_status": "settlement",
		"status_code":        "200",
		"gross_amount":       "40000.00",
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/midtrans", bytes.NewReader(mustJSON(t, payload)))
	req.Header.Set("Content-Type", "application/json")
	// No signature field at all.
	rr := httptest.NewRecorder()
	webhook.HandleMidtransWebhook(rr, req)
	if rr.Code == http.StatusOK {
		t.Fatalf("unsigned webhook accepted: %d %s", rr.Code, rr.Body.String())
	}
	if got := e.payments.transactions[orderID].Status; got != "pending" {
		t.Errorf("status = %q, want pending after a rejected webhook", got)
	}
}
