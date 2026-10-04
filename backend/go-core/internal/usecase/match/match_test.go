package match_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/match"
)

// --- mocks ---

type mockInvoiceFetcher struct {
	invoice *domain.Invoice
	err     error
}

func (m *mockInvoiceFetcher) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
	return m.invoice, m.err
}

type mockMatchRepo struct {
	po              *domain.PurchaseOrder
	gr              *domain.GoodsReceipt
	poErr           error
	grErr           error
	updatedResult   domain.MatchResult
	updatedStatus   domain.InvoiceStatus
	updateErr       error
}

func (m *mockMatchRepo) GetPOByID(_ context.Context, _, _ uuid.UUID) (*domain.PurchaseOrder, error) {
	return m.po, m.poErr
}
func (m *mockMatchRepo) GetGRByPOID(_ context.Context, _, _ uuid.UUID) (*domain.GoodsReceipt, error) {
	return m.gr, m.grErr
}
func (m *mockMatchRepo) UpdateInvoiceMatchResult(_ context.Context, _, _ uuid.UUID, _ *uuid.UUID, result domain.MatchResult, status domain.InvoiceStatus) error {
	m.updatedResult = result
	m.updatedStatus = status
	return m.updateErr
}

type mockPublisher struct {
	topic   string
	payload any
}

func (m *mockPublisher) Publish(_ context.Context, topic, _ string, payload any) error {
	m.topic = topic
	m.payload = payload
	return nil
}

// --- helpers ---

func poID() *uuid.UUID { id := uuid.New(); return &id }

func mustDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

func baseInvoice(poid *uuid.UUID) *domain.Invoice {
	return &domain.Invoice{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Amount:   mustDecimal("1000.00"),
		Status:   domain.StatusPending,
		POID:     poid,
	}
}

func basePO(qty int, amount decimal.Decimal) *domain.PurchaseOrder {
	return &domain.PurchaseOrder{
		ID:     uuid.New(),
		Qty:    qty,
		Amount: amount,
		Status: domain.POStatusOpen,
	}
}

func baseGR(qty int, amount decimal.Decimal) *domain.GoodsReceipt {
	return &domain.GoodsReceipt{
		ReceivedQty:    qty,
		ReceivedAmount: amount,
		Status:         domain.GRStatusAccepted,
	}
}

func tolerance2pct() decimal.Decimal { return mustDecimal("2.0") }

// --- tests ---

func TestMatch_Approved(t *testing.T) {
	pid := poID()
	inv := baseInvoice(pid)
	inv.Amount = mustDecimal("1010.00") // 1% over PO — within 2% tolerance

	repo := &mockMatchRepo{
		po: basePO(5, mustDecimal("1000.00")),
		gr: baseGR(5, mustDecimal("1000.00")),
	}
	pub := &mockPublisher{}

	uc := match.NewUsecase(&mockInvoiceFetcher{invoice: inv}, repo, pub, tolerance2pct())
	if err := uc.Execute(context.Background(), inv.ID, inv.TenantID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.updatedResult != domain.MatchResultMatched {
		t.Errorf("expected match_result=matched, got %s", repo.updatedResult)
	}
	if repo.updatedStatus != domain.StatusApproved {
		t.Errorf("expected status=approved, got %s", repo.updatedStatus)
	}
	if pub.topic != "invoice.approved" {
		t.Errorf("expected topic invoice.approved, got %s", pub.topic)
	}
}

func TestMatch_AmountMismatch(t *testing.T) {
	pid := poID()
	inv := baseInvoice(pid)
	inv.Amount = mustDecimal("1100.00") // 10% over — exceeds 2% tolerance

	repo := &mockMatchRepo{
		po: basePO(5, mustDecimal("1000.00")),
		gr: baseGR(5, mustDecimal("1000.00")),
	}
	pub := &mockPublisher{}

	uc := match.NewUsecase(&mockInvoiceFetcher{invoice: inv}, repo, pub, tolerance2pct())
	if err := uc.Execute(context.Background(), inv.ID, inv.TenantID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.updatedResult != domain.MatchResultAmountMismatch {
		t.Errorf("expected match_result=amount_mismatch, got %s", repo.updatedResult)
	}
	if repo.updatedStatus != domain.StatusPendingReview {
		t.Errorf("expected status=pending_review, got %s", repo.updatedStatus)
	}
	if pub.topic != "invoice.pending_review" {
		t.Errorf("expected topic invoice.pending_review, got %s", pub.topic)
	}
}

func TestMatch_QtyMismatch(t *testing.T) {
	pid := poID()
	inv := baseInvoice(pid)

	repo := &mockMatchRepo{
		po: basePO(5, mustDecimal("1000.00")),
		gr: baseGR(3, mustDecimal("1000.00")), // received 3 of 5
	}
	pub := &mockPublisher{}

	uc := match.NewUsecase(&mockInvoiceFetcher{invoice: inv}, repo, pub, tolerance2pct())
	if err := uc.Execute(context.Background(), inv.ID, inv.TenantID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.updatedResult != domain.MatchResultQtyMismatch {
		t.Errorf("expected match_result=qty_mismatch, got %s", repo.updatedResult)
	}
	if repo.updatedStatus != domain.StatusPendingReview {
		t.Errorf("expected status=pending_review, got %s", repo.updatedStatus)
	}
}

func TestMatch_NoPO(t *testing.T) {
	inv := baseInvoice(nil) // no PO linked
	repo := &mockMatchRepo{}
	pub := &mockPublisher{}

	uc := match.NewUsecase(&mockInvoiceFetcher{invoice: inv}, repo, pub, tolerance2pct())
	if err := uc.Execute(context.Background(), inv.ID, inv.TenantID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.updatedResult != domain.MatchResultNoPO {
		t.Errorf("expected match_result=no_po, got %s", repo.updatedResult)
	}
	if repo.updatedStatus != domain.StatusPendingReview {
		t.Errorf("expected status=pending_review, got %s", repo.updatedStatus)
	}
}

func TestMatch_GRNotFound(t *testing.T) {
	pid := poID()
	inv := baseInvoice(pid)

	repo := &mockMatchRepo{
		po:    basePO(5, mustDecimal("1000.00")),
		grErr: domain.ErrNotFound,
	}
	pub := &mockPublisher{}

	uc := match.NewUsecase(&mockInvoiceFetcher{invoice: inv}, repo, pub, tolerance2pct())
	if err := uc.Execute(context.Background(), inv.ID, inv.TenantID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.updatedResult != domain.MatchResultNoGR {
		t.Errorf("expected match_result=no_gr, got %s", repo.updatedResult)
	}
	if repo.updatedStatus != domain.StatusPendingReview {
		t.Errorf("expected status=pending_review, got %s", repo.updatedStatus)
	}
}

func TestMatch_ToleranceBoundary(t *testing.T) {
	// Exactly 2.0% diff → approved (boundary inclusive)
	pid := poID()
	inv := baseInvoice(pid)
	inv.Amount = mustDecimal("1020.00") // exactly 2% over 1000

	repo := &mockMatchRepo{
		po: basePO(5, mustDecimal("1000.00")),
		gr: baseGR(5, mustDecimal("1000.00")),
	}
	pub := &mockPublisher{}

	uc := match.NewUsecase(&mockInvoiceFetcher{invoice: inv}, repo, pub, tolerance2pct())
	if err := uc.Execute(context.Background(), inv.ID, inv.TenantID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.updatedResult != domain.MatchResultMatched {
		t.Errorf("boundary 2%% should be approved, got %s", repo.updatedResult)
	}
}

func TestMatch_ToleranceJustOver(t *testing.T) {
	// 2.001% over → escalate
	pid := poID()
	inv := baseInvoice(pid)
	inv.Amount = mustDecimal("1020.01")

	repo := &mockMatchRepo{
		po: basePO(5, mustDecimal("1000.00")),
		gr: baseGR(5, mustDecimal("1000.00")),
	}
	pub := &mockPublisher{}

	uc := match.NewUsecase(&mockInvoiceFetcher{invoice: inv}, repo, pub, tolerance2pct())
	if err := uc.Execute(context.Background(), inv.ID, inv.TenantID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.updatedResult != domain.MatchResultAmountMismatch {
		t.Errorf("2.0001%% over should be amount_mismatch, got %s", repo.updatedResult)
	}
}

func TestMatch_ZeroAmountPO(t *testing.T) {
	// PO with zero amount must escalate — division-by-zero guard in match.go
	// The guard fires after GR is fetched successfully, so GR qty must match to
	// reach that branch (the qty check comes after the zero-amount check).
	pid := poID()
	inv := baseInvoice(pid)
	inv.Amount = mustDecimal("0.00")

	repo := &mockMatchRepo{
		po: basePO(5, mustDecimal("0.00")), // zero-amount PO
		gr: baseGR(5, mustDecimal("0.00")), // qty matches — ensures guard is reached
	}
	pub := &mockPublisher{}

	uc := match.NewUsecase(&mockInvoiceFetcher{invoice: inv}, repo, pub, tolerance2pct())
	if err := uc.Execute(context.Background(), inv.ID, inv.TenantID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.updatedResult != domain.MatchResultAmountMismatch {
		t.Errorf("zero-amount PO should yield amount_mismatch, got %s", repo.updatedResult)
	}
	if repo.updatedStatus != domain.StatusPendingReview {
		t.Errorf("zero-amount PO should yield pending_review, got %s", repo.updatedStatus)
	}
	if pub.topic != "invoice.pending_review" {
		t.Errorf("expected topic invoice.pending_review, got %s", pub.topic)
	}
}
