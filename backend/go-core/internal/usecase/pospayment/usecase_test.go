package pospayment_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/midtrans"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/pospayment"
)

// ── fakes ────────────────────────────────────────────────────────────────────

type fakeRepo struct {
	configs      map[string]*domain.TenantPaymentConfig
	transactions map[string]*domain.PaymentTransaction
	// ignoreTenantScope mimics a repo that leaks rows across tenants, so the
	// usecase's own tenant check can be proven to hold on its own.
	ignoreTenantScope bool
	failCreateTx      error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		configs:      map[string]*domain.TenantPaymentConfig{},
		transactions: map[string]*domain.PaymentTransaction{},
	}
}

func (f *fakeRepo) GetConfig(ctx context.Context, tenantID uuid.UUID, provider string) (*domain.TenantPaymentConfig, error) {
	cfg, ok := f.configs[tenantID.String()+"|"+provider]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return cfg, nil
}

func (f *fakeRepo) UpsertConfig(ctx context.Context, cfg *domain.TenantPaymentConfig) error {
	f.configs[cfg.TenantID.String()+"|"+cfg.Provider] = cfg
	return nil
}

func (f *fakeRepo) CreateTransaction(ctx context.Context, tx *domain.PaymentTransaction) error {
	if f.failCreateTx != nil {
		return f.failCreateTx
	}
	cp := *tx
	f.transactions[tx.OrderID] = &cp
	return nil
}

func (f *fakeRepo) GetTransactionByOrderID(ctx context.Context, tenantID uuid.UUID, orderID string) (*domain.PaymentTransaction, error) {
	tx, ok := f.transactions[orderID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	// Real repo relies on RLS; the fake mirrors that unless told otherwise.
	if !f.ignoreTenantScope && tx.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	return tx, nil
}

func (f *fakeRepo) UpdateTransactionStatus(ctx context.Context, tenantID uuid.UUID, orderID, status string, completedAt *time.Time) error {
	tx, ok := f.transactions[orderID]
	if !ok {
		return domain.ErrNotFound
	}
	if !f.ignoreTenantScope && tx.TenantID != tenantID {
		return domain.ErrNotFound
	}
	tx.Status = status
	tx.CompletedAt = completedAt
	return nil
}

func (f *fakeRepo) ConsumeTransaction(ctx context.Context, tenantID uuid.UUID, orderID string) error {
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

type fakeClient struct {
	chargeResp    *midtrans.CreateChargeQRISResponse
	chargeErr     error
	statusResp    *midtrans.TransactionStatus
	statusErr     error
	lastChargeReq midtrans.CreateChargeQRISRequest
	statusCalls   int
}

func (c *fakeClient) CreateChargeQRIS(ctx context.Context, req midtrans.CreateChargeQRISRequest) (*midtrans.CreateChargeQRISResponse, error) {
	c.lastChargeReq = req
	if c.chargeErr != nil {
		return nil, c.chargeErr
	}
	if c.chargeResp != nil {
		return c.chargeResp, nil
	}
	return &midtrans.CreateChargeQRISResponse{
		OrderID:   req.OrderID,
		QRString:  "00020101021226...",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}, nil
}

func (c *fakeClient) VerifyTransactionStatus(ctx context.Context, orderID string) (*midtrans.TransactionStatus, error) {
	c.statusCalls++
	if c.statusErr != nil {
		return nil, c.statusErr
	}
	if c.statusResp != nil {
		return c.statusResp, nil
	}
	return &midtrans.TransactionStatus{Status: "pending"}, nil
}

func withGateway(t *testing.T, repo *fakeRepo, tenantID uuid.UUID, serverKey string, isActive bool) {
	t.Helper()
	key := serverKey
	cfg := &domain.TenantPaymentConfig{
		TenantID: tenantID,
		Provider: pospayment.Provider,
		ServerKey: func() *string {
			if key == "" {
				return nil
			}
			return &key
		}(),
		IsActive: isActive,
	}
	if err := repo.UpsertConfig(context.Background(), cfg); err != nil {
		t.Fatalf("seed config: %v", err)
	}
}

func newUsecase(repo *fakeRepo, client *fakeClient) *pospayment.Usecase {
	factory := func(serverKey, clientKey string, isProduction bool) pospayment.Client {
		return client
	}
	return pospayment.New(repo, factory, "https://tayooli.my.id")
}

// ── CreateCharge ─────────────────────────────────────────────────────────────

func TestCreateCharge_DemoModeWhenTenantHasNoCredentials(t *testing.T) {
	repo := newFakeRepo()
	client := &fakeClient{}
	uc := newUsecase(repo, client)
	tenantID := uuid.New()

	charge, err := uc.CreateCharge(context.Background(), tenantID, decimal.NewFromInt(50000), "qris")
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	if !charge.Demo {
		t.Error("expected demo mode when the tenant has no gateway config")
	}
	if charge.QRString != "" {
		t.Errorf("demo charge must not carry a gateway QR string, got %q", charge.QRString)
	}
	if !strings.Contains(charge.PaymentLink, "/payments/demo/") {
		t.Errorf("demo charge should link to the demo checkout page, got %q", charge.PaymentLink)
	}
	if client.lastChargeReq.OrderID != "" {
		t.Error("demo mode must not call the gateway")
	}

	// The order id must carry the tenant so a public webhook can resolve it.
	gotTenant, err := midtrans.TenantFromOrderID(charge.OrderID)
	if err != nil {
		t.Fatalf("TenantFromOrderID(%q): %v", charge.OrderID, err)
	}
	if gotTenant != tenantID {
		t.Errorf("order id tenant = %s, want %s", gotTenant, tenantID)
	}

	tx := repo.transactions[charge.OrderID]
	if tx == nil {
		t.Fatal("transaction must be persisted before the gateway call")
	}
	if !tx.IsDemo || tx.Status != "pending" {
		t.Errorf("persisted tx = demo:%v status:%s, want demo:true status:pending", tx.IsDemo, tx.Status)
	}
}

func TestCreateCharge_RealModeUsesGatewayAndStoresIntent(t *testing.T) {
	repo := newFakeRepo()
	client := &fakeClient{}
	uc := newUsecase(repo, client)
	tenantID := uuid.New()
	withGateway(t, repo, tenantID, "SB-Mid-server-abc", true)

	charge, err := uc.CreateCharge(context.Background(), tenantID, decimal.NewFromInt(75000), "QRIS")
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	if charge.Demo {
		t.Error("tenant with credentials must not fall back to demo mode")
	}
	if charge.QRString == "" {
		t.Error("real charge must return the gateway QR string")
	}
	if client.lastChargeReq.OrderID != charge.OrderID {
		t.Errorf("gateway order id = %q, want %q", client.lastChargeReq.OrderID, charge.OrderID)
	}
	if !client.lastChargeReq.Amount.Equal(decimal.NewFromInt(75000)) {
		t.Errorf("gateway amount = %s, want 75000", client.lastChargeReq.Amount)
	}
	if tx := repo.transactions[charge.OrderID]; tx == nil || tx.IsDemo {
		t.Error("real charge must be persisted with is_demo=false")
	}
}

func TestCreateCharge_InactiveConfigFallsBackToDemo(t *testing.T) {
	repo := newFakeRepo()
	uc := newUsecase(repo, &fakeClient{})
	tenantID := uuid.New()
	withGateway(t, repo, tenantID, "SB-Mid-server-abc", false) // is_active=false

	charge, err := uc.CreateCharge(context.Background(), tenantID, decimal.NewFromInt(1000), "QRIS")
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	if !charge.Demo {
		t.Error("a disabled gateway config must not be used for live charges")
	}
}

func TestCreateCharge_RejectsBadInput(t *testing.T) {
	repo := newFakeRepo()
	uc := newUsecase(repo, &fakeClient{})
	tenantID := uuid.New()

	cases := []struct {
		name   string
		amount decimal.Decimal
		method string
	}{
		{"zero amount", decimal.Zero, "QRIS"},
		{"negative amount", decimal.NewFromInt(-5), "QRIS"},
		{"unsupported method", decimal.NewFromInt(1000), "VA"},
		{"empty method", decimal.NewFromInt(1000), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uc.CreateCharge(context.Background(), tenantID, tc.amount, tc.method)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("err = %v, want ErrInvalidInput", err)
			}
		})
	}
	if len(repo.transactions) != 0 {
		t.Error("rejected input must not create a transaction row")
	}
}

func TestCreateCharge_GatewayFailureMarksIntentFailed(t *testing.T) {
	repo := newFakeRepo()
	client := &fakeClient{chargeErr: errors.New("midtrans 500")}
	uc := newUsecase(repo, client)
	tenantID := uuid.New()
	withGateway(t, repo, tenantID, "SB-Mid-server-abc", true)

	_, err := uc.CreateCharge(context.Background(), tenantID, decimal.NewFromInt(9000), "QRIS")
	if err == nil {
		t.Fatal("expected the gateway error to surface")
	}
	if len(repo.transactions) != 1 {
		t.Fatalf("intent should still be recorded, got %d rows", len(repo.transactions))
	}
	for _, tx := range repo.transactions {
		if tx.Status != "failed" {
			t.Errorf("failed charge status = %q, want failed (never payable)", tx.Status)
		}
	}
}

// ── SimulateDemo ─────────────────────────────────────────────────────────────

func TestSimulateDemo_CompletesThenIsIdempotent(t *testing.T) {
	repo := newFakeRepo()
	uc := newUsecase(repo, &fakeClient{})
	tenantID := uuid.New()

	charge, err := uc.CreateCharge(context.Background(), tenantID, decimal.NewFromInt(20000), "QRIS")
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	if err := uc.SimulateDemo(context.Background(), tenantID, charge.OrderID); err != nil {
		t.Fatalf("SimulateDemo: %v", err)
	}
	if got := repo.transactions[charge.OrderID].Status; got != "completed" {
		t.Fatalf("status = %q, want completed", got)
	}
	// A repeated callback/page refresh must not error.
	if err := uc.SimulateDemo(context.Background(), tenantID, charge.OrderID); err != nil {
		t.Errorf("second simulate should be a no-op, got %v", err)
	}
}

func TestSimulateDemo_RefusedWhenRealGatewayConfigured(t *testing.T) {
	repo := newFakeRepo()
	uc := newUsecase(repo, &fakeClient{})
	tenantID := uuid.New()

	// Create the intent first (no credentials yet → demo row)...
	charge, err := uc.CreateCharge(context.Background(), tenantID, decimal.NewFromInt(15000), "QRIS")
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	// ...then the tenant adds real credentials.
	withGateway(t, repo, tenantID, "SB-Mid-server-abc", true)

	err = uc.SimulateDemo(context.Background(), tenantID, charge.OrderID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("err = %v, want ErrForbidden so demo can never settle a live tenant", err)
	}
	if got := repo.transactions[charge.OrderID].Status; got != "pending" {
		t.Errorf("status = %q, want pending (untouched)", got)
	}
}

func TestSimulateDemo_UnknownOrderIsNotFound(t *testing.T) {
	uc := newUsecase(newFakeRepo(), &fakeClient{})
	err := uc.SimulateDemo(context.Background(), uuid.New(), "Tdoesnotexist")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestSimulateDemo_CrossTenantIsNotFound(t *testing.T) {
	repo := newFakeRepo()
	uc := newUsecase(repo, &fakeClient{})
	owner := uuid.New()

	charge, err := uc.CreateCharge(context.Background(), owner, decimal.NewFromInt(11000), "QRIS")
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	// A repo that leaks rows across tenants must still not let another tenant
	// settle this payment: the usecase re-checks ownership itself.
	repo.ignoreTenantScope = true
	err = uc.SimulateDemo(context.Background(), uuid.New(), charge.OrderID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// ── Status ───────────────────────────────────────────────────────────────────

func TestStatus_CompletesPendingRealTransactionViaGateway(t *testing.T) {
	repo := newFakeRepo()
	client := &fakeClient{}
	uc := newUsecase(repo, client)
	tenantID := uuid.New()
	withGateway(t, repo, tenantID, "SB-Mid-server-abc", true)

	charge, err := uc.CreateCharge(context.Background(), tenantID, decimal.NewFromInt(30000), "QRIS")
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	// Webhook never arrives; the gateway says it is settled.
	client.statusResp = &midtrans.TransactionStatus{Status: "settlement", Amount: 30000}

	res, err := uc.Status(context.Background(), tenantID, charge.OrderID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if res.Status != "completed" {
		t.Errorf("status = %q, want completed", res.Status)
	}
	if res.CompletedAt == nil {
		t.Error("completed_at should be set")
	}
	if got := repo.transactions[charge.OrderID].Status; got != "completed" {
		t.Errorf("persisted status = %q, want completed", got)
	}
}

func TestStatus_IgnoresSettlementWithWrongAmount(t *testing.T) {
	repo := newFakeRepo()
	client := &fakeClient{}
	uc := newUsecase(repo, client)
	tenantID := uuid.New()
	withGateway(t, repo, tenantID, "SB-Mid-server-abc", true)

	charge, err := uc.CreateCharge(context.Background(), tenantID, decimal.NewFromInt(40000), "QRIS")
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	// Attacker-controlled or mismatched amount must never settle the sale.
	client.statusResp = &midtrans.TransactionStatus{Status: "settlement", Amount: 1}

	res, err := uc.Status(context.Background(), tenantID, charge.OrderID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if res.Status != "pending" {
		t.Errorf("status = %q, want pending on amount mismatch", res.Status)
	}
}

func TestStatus_MapsExpireAndDeny(t *testing.T) {
	for _, tc := range []struct {
		gateway  string
		expected string
	}{
		{"expire", "expired"},
		{"deny", "failed"},
		{"cancel", "failed"},
		{"failure", "failed"},
	} {
		t.Run(tc.gateway, func(t *testing.T) {
			repo := newFakeRepo()
			client := &fakeClient{}
			uc := newUsecase(repo, client)
			tenantID := uuid.New()
			withGateway(t, repo, tenantID, "SB-Mid-server-abc", true)

			charge, err := uc.CreateCharge(context.Background(), tenantID, decimal.NewFromInt(12000), "QRIS")
			if err != nil {
				t.Fatalf("CreateCharge: %v", err)
			}
			client.statusResp = &midtrans.TransactionStatus{Status: tc.gateway, Amount: 12000}

			res, err := uc.Status(context.Background(), tenantID, charge.OrderID)
			if err != nil {
				t.Fatalf("Status: %v", err)
			}
			if res.Status != tc.expected {
				t.Errorf("status = %q, want %q", res.Status, tc.expected)
			}
		})
	}
}

func TestStatus_DemoPaymentIsNotPolledFromGateway(t *testing.T) {
	repo := newFakeRepo()
	client := &fakeClient{}
	uc := newUsecase(repo, client)
	tenantID := uuid.New()

	charge, err := uc.CreateCharge(context.Background(), tenantID, decimal.NewFromInt(8000), "QRIS")
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	res, err := uc.Status(context.Background(), tenantID, charge.OrderID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if res.Status != "pending" || !res.Demo {
		t.Errorf("res = %+v, want pending demo", res)
	}
	if client.statusCalls != 0 {
		t.Error("demo payments must never be verified against the gateway")
	}
}

func TestStatus_TransientGatewayErrorKeepsStoredStatus(t *testing.T) {
	repo := newFakeRepo()
	client := &fakeClient{statusErr: errors.New("timeout")}
	uc := newUsecase(repo, client)
	tenantID := uuid.New()
	withGateway(t, repo, tenantID, "SB-Mid-server-abc", true)

	charge, err := uc.CreateCharge(context.Background(), tenantID, decimal.NewFromInt(6000), "QRIS")
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	res, err := uc.Status(context.Background(), tenantID, charge.OrderID)
	if err != nil {
		t.Fatalf("Status must not fail the cashier's poll: %v", err)
	}
	if res.Status != "pending" {
		t.Errorf("status = %q, want pending", res.Status)
	}
}

// ── Claim / Release ──────────────────────────────────────────────────────────

// settle creates a payment and marks it completed, returning the order id.
func settle(t *testing.T, uc *pospayment.Usecase, repo *fakeRepo, tenantID uuid.UUID, amount int64) string {
	t.Helper()
	charge, err := uc.CreateCharge(context.Background(), tenantID, decimal.NewFromInt(amount), "QRIS")
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	if err := uc.SimulateDemo(context.Background(), tenantID, charge.OrderID); err != nil {
		t.Fatalf("SimulateDemo: %v", err)
	}
	return charge.OrderID
}

func TestClaim_ConsumesSettledPaymentExactlyOnce(t *testing.T) {
	repo := newFakeRepo()
	uc := newUsecase(repo, &fakeClient{})
	tenantID := uuid.New()
	orderID := settle(t, uc, repo, tenantID, 25000)

	if err := uc.Claim(context.Background(), tenantID, orderID, decimal.NewFromInt(25000)); err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if got := repo.transactions[orderID].Status; got != "consumed" {
		t.Errorf("status = %q, want consumed", got)
	}
	// Replaying the same payment must not mint a second sale.
	err := uc.Claim(context.Background(), tenantID, orderID, decimal.NewFromInt(25000))
	if !errors.Is(err, pospayment.ErrPaymentNotSettled) {
		t.Errorf("second claim err = %v, want ErrPaymentNotSettled", err)
	}
}

func TestClaim_RejectsUnsettledPayment(t *testing.T) {
	repo := newFakeRepo()
	uc := newUsecase(repo, &fakeClient{})
	tenantID := uuid.New()

	charge, err := uc.CreateCharge(context.Background(), tenantID, decimal.NewFromInt(25000), "QRIS")
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	err = uc.Claim(context.Background(), tenantID, charge.OrderID, decimal.NewFromInt(25000))
	if !errors.Is(err, pospayment.ErrPaymentNotSettled) {
		t.Errorf("err = %v, want ErrPaymentNotSettled", err)
	}
}

func TestClaim_RejectsAmountMismatch(t *testing.T) {
	repo := newFakeRepo()
	uc := newUsecase(repo, &fakeClient{})
	tenantID := uuid.New()
	orderID := settle(t, uc, repo, tenantID, 25000)

	err := uc.Claim(context.Background(), tenantID, orderID, decimal.NewFromInt(1000))
	if !errors.Is(err, pospayment.ErrAmountMismatch) {
		t.Errorf("err = %v, want ErrAmountMismatch", err)
	}
	if got := repo.transactions[orderID].Status; got != "completed" {
		t.Errorf("status = %q, want completed (a mismatch must not consume)", got)
	}
}

func TestClaim_UnknownOrCrossTenantIsNotSettled(t *testing.T) {
	repo := newFakeRepo()
	uc := newUsecase(repo, &fakeClient{})
	tenantID := uuid.New()
	orderID := settle(t, uc, repo, tenantID, 5000)

	if err := uc.Claim(context.Background(), uuid.New(), orderID, decimal.NewFromInt(5000)); !errors.Is(err, pospayment.ErrPaymentNotSettled) {
		t.Errorf("cross-tenant err = %v, want ErrPaymentNotSettled", err)
	}
	if err := uc.Claim(context.Background(), tenantID, "Tunknown", decimal.NewFromInt(5000)); !errors.Is(err, pospayment.ErrPaymentNotSettled) {
		t.Errorf("unknown err = %v, want ErrPaymentNotSettled", err)
	}
}

func TestRelease_RestoresPaymentAfterFailedCheckout(t *testing.T) {
	repo := newFakeRepo()
	uc := newUsecase(repo, &fakeClient{})
	tenantID := uuid.New()
	orderID := settle(t, uc, repo, tenantID, 33000)

	if err := uc.Claim(context.Background(), tenantID, orderID, decimal.NewFromInt(33000)); err != nil {
		t.Fatalf("claim: %v", err)
	}
	// Checkout failed (e.g. stock ran out): the payment must be usable again.
	if err := uc.Release(context.Background(), tenantID, orderID); err != nil {
		t.Fatalf("Release: %v", err)
	}
	tx := repo.transactions[orderID]
	if tx.Status != "completed" {
		t.Fatalf("status = %q, want completed", tx.Status)
	}
	if tx.CompletedAt == nil {
		t.Error("completed_at must be preserved so the audit trail survives")
	}
	if err := uc.Claim(context.Background(), tenantID, orderID, decimal.NewFromInt(33000)); err != nil {
		t.Errorf("re-claim after release: %v", err)
	}
}

// ── RequiresVerifiedPayment ──────────────────────────────────────────────────

func TestRequiresVerifiedPayment(t *testing.T) {
	repo := newFakeRepo()
	uc := newUsecase(repo, &fakeClient{})
	tenantID := uuid.New()

	if uc.RequiresVerifiedPayment(context.Background(), tenantID) {
		t.Error("no config → demo mode → false")
	}
	withGateway(t, repo, tenantID, "SB-Mid-server-abc", true)
	if !uc.RequiresVerifiedPayment(context.Background(), tenantID) {
		t.Error("active server key → true")
	}
	withGateway(t, repo, tenantID, "", true) // blank key
	if uc.RequiresVerifiedPayment(context.Background(), tenantID) {
		t.Error("blank server key must not count as configured")
	}
}
