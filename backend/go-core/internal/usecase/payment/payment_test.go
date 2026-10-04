package payment_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/payment"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockPaymentOrderRepo struct {
	createFn         func(ctx context.Context, params domain.CreatePaymentOrderParams) (*domain.PaymentOrder, error)
	listPagedFn      func(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.PaymentOrderListPage, error)
	getByIDFn        func(ctx context.Context, id, tenantID uuid.UUID) (*domain.PaymentOrder, error)
	updateStatusFn   func(ctx context.Context, id, tenantID uuid.UUID, from, to domain.PaymentOrderStatus, approvedBy *uuid.UUID) (*domain.PaymentOrder, error)
	markInvoicePaidFn func(ctx context.Context, paymentOrderID, tenantID uuid.UUID) error
}

func (m *mockPaymentOrderRepo) Create(ctx context.Context, params domain.CreatePaymentOrderParams) (*domain.PaymentOrder, error) {
	return m.createFn(ctx, params)
}
func (m *mockPaymentOrderRepo) ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.PaymentOrderListPage, error) {
	return m.listPagedFn(ctx, tenantID, page, perPage)
}
func (m *mockPaymentOrderRepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.PaymentOrder, error) {
	return m.getByIDFn(ctx, id, tenantID)
}
func (m *mockPaymentOrderRepo) UpdateStatus(ctx context.Context, id, tenantID uuid.UUID, from, to domain.PaymentOrderStatus, approvedBy *uuid.UUID) (*domain.PaymentOrder, error) {
	return m.updateStatusFn(ctx, id, tenantID, from, to, approvedBy)
}
func (m *mockPaymentOrderRepo) MarkInvoicePaid(ctx context.Context, paymentOrderID, tenantID uuid.UUID) error {
	return m.markInvoicePaidFn(ctx, paymentOrderID, tenantID)
}

type mockInvoiceRepo struct {
	getByIDFn func(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error)
}

func (m *mockInvoiceRepo) List(_ context.Context, _ uuid.UUID) ([]domain.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceRepo) ListPaged(_ context.Context, _ uuid.UUID, _, _ int) (*domain.InvoiceListPage, error) {
	return nil, nil
}
func (m *mockInvoiceRepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	return m.getByIDFn(ctx, id, tenantID)
}
func (m *mockInvoiceRepo) Create(_ context.Context, _ domain.CreateInvoiceParams) (*domain.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceRepo) Approve(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceRepo) Reject(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
	return nil, nil
}
func (m *mockInvoiceRepo) UpdateOCRResult(_ context.Context, _, _ uuid.UUID, _ float64, _, _ string) error {
	return nil
}

type mockPublisher struct {
	publishedTopic   string
	publishedKey     string
	publishedPayload any
	err              error
}

func (m *mockPublisher) Publish(_ context.Context, topic string, key string, payload any) error {
	m.publishedTopic = topic
	m.publishedKey = key
	m.publishedPayload = payload
	return m.err
}

type mockAuditLogRepo struct {
	entries []domain.AuditLogEntry
	err     error
}

func (m *mockAuditLogRepo) Create(_ context.Context, entry domain.AuditLogEntry) error {
	if m.err != nil {
		return m.err
	}
	m.entries = append(m.entries, entry)
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func fakePaymentOrder(tenantID uuid.UUID) *domain.PaymentOrder {
	return &domain.PaymentOrder{
		ID:            uuid.New(),
		TenantID:      tenantID,
		InvoiceID:     uuid.New(),
		Amount:        decimal.New(5_000_000, 0),
		Currency:      "IDR",
		PaymentMethod: "bank_transfer",
		Status:        domain.PaymentOrderStatusDraft,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func fakeApprovedInvoice(tenantID uuid.UUID) *domain.Invoice {
	return &domain.Invoice{
		ID:       uuid.New(),
		TenantID: tenantID,
		Status:   domain.StatusApproved,
		Amount:   decimal.New(5_000_000, 0),
		Currency: "IDR",
	}
}

// ---------------------------------------------------------------------------
// CreateUsecase tests
// ---------------------------------------------------------------------------

func TestCreateUsecase_Execute_OK(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{}

	inv := fakeApprovedInvoice(tenantID)
	inv.ID = invoiceID

	invRepo := &mockInvoiceRepo{
		getByIDFn: func(_ context.Context, id, tid uuid.UUID) (*domain.Invoice, error) {
			if id == invoiceID && tid == tenantID {
				return inv, nil
			}
			return nil, domain.ErrNotFound
		},
	}

	poRepo := &mockPaymentOrderRepo{
		createFn: func(_ context.Context, p domain.CreatePaymentOrderParams) (*domain.PaymentOrder, error) {
			po := fakePaymentOrder(p.TenantID)
			po.InvoiceID = p.InvoiceID
			po.Amount = p.Amount
			return po, nil
		},
	}

	uc := payment.NewCreateUsecase(poRepo, invRepo, pub, audit)
	po, err := uc.Execute(context.Background(), domain.CreatePaymentOrderParams{
		TenantID:      tenantID,
		InvoiceID:     invoiceID,
		Amount:        decimal.New(5_000_000, 0),
		Currency:      "IDR",
		PaymentMethod: "bank_transfer",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if po == nil {
		t.Fatal("expected payment order, got nil")
	}
	if pub.publishedTopic != payment.TopicPaymentOrderCreated {
		t.Errorf("expected topic %q, got %q", payment.TopicPaymentOrderCreated, pub.publishedTopic)
	}
	if pub.publishedKey != po.ID.String() {
		t.Errorf("expected kafka key %q, got %q", po.ID.String(), pub.publishedKey)
	}
	// Verify audit log was written.
	if len(audit.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(audit.entries))
	}
	if audit.entries[0].Action != "created" {
		t.Errorf("expected audit action 'created', got %q", audit.entries[0].Action)
	}
	if audit.entries[0].EntityType != "payment_order" {
		t.Errorf("expected entity_type 'payment_order', got %q", audit.entries[0].EntityType)
	}
}

func TestCreateUsecase_Execute_InvoiceNotApproved(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{}

	inv := fakeApprovedInvoice(tenantID)
	inv.ID = invoiceID
	inv.Status = domain.StatusPending // not approved

	invRepo := &mockInvoiceRepo{
		getByIDFn: func(_ context.Context, id, tid uuid.UUID) (*domain.Invoice, error) {
			return inv, nil
		},
	}
	poRepo := &mockPaymentOrderRepo{}

	uc := payment.NewCreateUsecase(poRepo, invRepo, pub, audit)
	_, err := uc.Execute(context.Background(), domain.CreatePaymentOrderParams{
		TenantID:  tenantID,
		InvoiceID: invoiceID,
		Amount:    decimal.New(100, 0),
		Currency:  "IDR",
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestCreateUsecase_Execute_InvoiceNotFound(t *testing.T) {
	tenantID := uuid.New()
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{}

	invRepo := &mockInvoiceRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
			return nil, domain.ErrNotFound
		},
	}
	poRepo := &mockPaymentOrderRepo{}

	uc := payment.NewCreateUsecase(poRepo, invRepo, pub, audit)
	_, err := uc.Execute(context.Background(), domain.CreatePaymentOrderParams{
		TenantID:  tenantID,
		InvoiceID: uuid.New(),
		Amount:    decimal.New(100, 0),
		Currency:  "IDR",
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateUsecase_Execute_InvalidInput(t *testing.T) {
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{}
	invRepo := &mockInvoiceRepo{}
	poRepo := &mockPaymentOrderRepo{}
	uc := payment.NewCreateUsecase(poRepo, invRepo, pub, audit)

	cases := []struct {
		name   string
		params domain.CreatePaymentOrderParams
	}{
		{"zero invoice_id", domain.CreatePaymentOrderParams{TenantID: uuid.New(), InvoiceID: uuid.Nil, Amount: decimal.New(100, 0), Currency: "IDR"}},
		{"zero amount", domain.CreatePaymentOrderParams{TenantID: uuid.New(), InvoiceID: uuid.New(), Amount: decimal.Zero, Currency: "IDR"}},
		{"empty currency", domain.CreatePaymentOrderParams{TenantID: uuid.New(), InvoiceID: uuid.New(), Amount: decimal.New(100, 0), Currency: ""}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uc.Execute(context.Background(), tc.params)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestCreateUsecase_Execute_ContinuesIfPublishFails(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()
	pub := &mockPublisher{err: errors.New("kafka down")}
	audit := &mockAuditLogRepo{}

	inv := fakeApprovedInvoice(tenantID)
	inv.ID = invoiceID

	invRepo := &mockInvoiceRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
			return inv, nil
		},
	}
	poRepo := &mockPaymentOrderRepo{
		createFn: func(_ context.Context, p domain.CreatePaymentOrderParams) (*domain.PaymentOrder, error) {
			return fakePaymentOrder(p.TenantID), nil
		},
	}

	uc := payment.NewCreateUsecase(poRepo, invRepo, pub, audit)
	_, err := uc.Execute(context.Background(), domain.CreatePaymentOrderParams{
		TenantID:  tenantID,
		InvoiceID: invoiceID,
		Amount:    decimal.New(100, 0),
		Currency:  "IDR",
	})
	if err != nil {
		t.Errorf("publish failure should be swallowed, got %v", err)
	}
}

func TestCreateUsecase_Execute_ContinuesIfAuditFails(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{err: errors.New("audit db down")}

	inv := fakeApprovedInvoice(tenantID)
	inv.ID = invoiceID

	invRepo := &mockInvoiceRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
			return inv, nil
		},
	}
	poRepo := &mockPaymentOrderRepo{
		createFn: func(_ context.Context, p domain.CreatePaymentOrderParams) (*domain.PaymentOrder, error) {
			return fakePaymentOrder(p.TenantID), nil
		},
	}

	uc := payment.NewCreateUsecase(poRepo, invRepo, pub, audit)
	_, err := uc.Execute(context.Background(), domain.CreatePaymentOrderParams{
		TenantID:  tenantID,
		InvoiceID: invoiceID,
		Amount:    decimal.New(100, 0),
		Currency:  "IDR",
	})
	if err != nil {
		t.Errorf("audit failure should be swallowed, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// ApproveUsecase tests
// ---------------------------------------------------------------------------

func TestApproveUsecase_Execute_OK(t *testing.T) {
	tenantID := uuid.New()
	poID := uuid.New()
	approvedBy := uuid.New()
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{}

	poRepo := &mockPaymentOrderRepo{
		updateStatusFn: func(_ context.Context, id, tid uuid.UUID, from, to domain.PaymentOrderStatus, ab *uuid.UUID) (*domain.PaymentOrder, error) {
			po := fakePaymentOrder(tid)
			po.ID = id
			po.Status = domain.PaymentOrderStatusApproved
			po.ApprovedBy = ab
			return po, nil
		},
	}

	uc := payment.NewApproveUsecase(poRepo, pub, audit)
	po, err := uc.Execute(context.Background(), poID, tenantID, approvedBy, "cfo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if po.Status != domain.PaymentOrderStatusApproved {
		t.Errorf("expected status approved, got %s", po.Status)
	}
	if pub.publishedTopic != payment.TopicPaymentOrderApproved {
		t.Errorf("expected topic %q, got %q", payment.TopicPaymentOrderApproved, pub.publishedTopic)
	}
	if pub.publishedKey != po.ID.String() {
		t.Errorf("expected kafka key %q, got %q", po.ID.String(), pub.publishedKey)
	}
	// Verify audit log.
	if len(audit.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(audit.entries))
	}
	if audit.entries[0].Action != "approved" {
		t.Errorf("expected audit action 'approved', got %q", audit.entries[0].Action)
	}
	if audit.entries[0].UserID == nil || *audit.entries[0].UserID != approvedBy {
		t.Errorf("expected audit user_id %s, got %v", approvedBy, audit.entries[0].UserID)
	}
}

func TestApproveUsecase_Execute_AdminRole(t *testing.T) {
	tenantID := uuid.New()
	poID := uuid.New()
	approvedBy := uuid.New()
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{}

	poRepo := &mockPaymentOrderRepo{
		updateStatusFn: func(_ context.Context, id, tid uuid.UUID, _, _ domain.PaymentOrderStatus, _ *uuid.UUID) (*domain.PaymentOrder, error) {
			po := fakePaymentOrder(tid)
			po.ID = id
			po.Status = domain.PaymentOrderStatusApproved
			return po, nil
		},
	}

	uc := payment.NewApproveUsecase(poRepo, pub, audit)
	po, err := uc.Execute(context.Background(), poID, tenantID, approvedBy, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if po.Status != domain.PaymentOrderStatusApproved {
		t.Errorf("expected status approved, got %s", po.Status)
	}
}

func TestApproveUsecase_Execute_ForbiddenRole(t *testing.T) {
	tenantID := uuid.New()
	poID := uuid.New()
	approvedBy := uuid.New()
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{}
	poRepo := &mockPaymentOrderRepo{}

	uc := payment.NewApproveUsecase(poRepo, pub, audit)
	_, err := uc.Execute(context.Background(), poID, tenantID, approvedBy, "accountant")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestApproveUsecase_Execute_ForbiddenEmptyRole(t *testing.T) {
	tenantID := uuid.New()
	poID := uuid.New()
	approvedBy := uuid.New()
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{}
	poRepo := &mockPaymentOrderRepo{}

	uc := payment.NewApproveUsecase(poRepo, pub, audit)
	_, err := uc.Execute(context.Background(), poID, tenantID, approvedBy, "")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden for empty role, got %v", err)
	}
}

func TestApproveUsecase_Execute_NotFound(t *testing.T) {
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{}
	poRepo := &mockPaymentOrderRepo{
		updateStatusFn: func(_ context.Context, _, _ uuid.UUID, _, _ domain.PaymentOrderStatus, _ *uuid.UUID) (*domain.PaymentOrder, error) {
			return nil, domain.ErrNotFound
		},
	}

	uc := payment.NewApproveUsecase(poRepo, pub, audit)
	_, err := uc.Execute(context.Background(), uuid.New(), uuid.New(), uuid.New(), "cfo")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestApproveUsecase_Execute_Conflict(t *testing.T) {
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{}
	poRepo := &mockPaymentOrderRepo{
		updateStatusFn: func(_ context.Context, _, _ uuid.UUID, _, _ domain.PaymentOrderStatus, _ *uuid.UUID) (*domain.PaymentOrder, error) {
			return nil, domain.ErrConflict
		},
	}

	uc := payment.NewApproveUsecase(poRepo, pub, audit)
	_, err := uc.Execute(context.Background(), uuid.New(), uuid.New(), uuid.New(), "cfo")
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestApproveUsecase_Execute_ContinuesIfAuditFails(t *testing.T) {
	tenantID := uuid.New()
	poID := uuid.New()
	approvedBy := uuid.New()
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{err: errors.New("audit db down")}

	poRepo := &mockPaymentOrderRepo{
		updateStatusFn: func(_ context.Context, id, tid uuid.UUID, _, _ domain.PaymentOrderStatus, _ *uuid.UUID) (*domain.PaymentOrder, error) {
			po := fakePaymentOrder(tid)
			po.ID = id
			po.Status = domain.PaymentOrderStatusApproved
			return po, nil
		},
	}

	uc := payment.NewApproveUsecase(poRepo, pub, audit)
	_, err := uc.Execute(context.Background(), poID, tenantID, approvedBy, "cfo")
	if err != nil {
		t.Errorf("audit failure should be swallowed, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// PayUsecase tests
// ---------------------------------------------------------------------------

func TestPayUsecase_Execute_OK(t *testing.T) {
	tenantID := uuid.New()
	poID := uuid.New()
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{}
	callCount := 0

	poRepo := &mockPaymentOrderRepo{
		getByIDFn: func(_ context.Context, id, tid uuid.UUID) (*domain.PaymentOrder, error) {
			callCount++
			po := fakePaymentOrder(tid)
			po.ID = id
			if callCount == 1 {
				// First call — return approved status
				po.Status = domain.PaymentOrderStatusApproved
			} else {
				// Second call (after MarkInvoicePaid) — return paid status
				po.Status = domain.PaymentOrderStatusPaid
				now := time.Now()
				po.PaidAt = &now
			}
			return po, nil
		},
		markInvoicePaidFn: func(_ context.Context, _, _ uuid.UUID) error {
			return nil
		},
	}

	uc := payment.NewPayUsecase(poRepo, pub, audit)
	po, err := uc.Execute(context.Background(), poID, tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if po.Status != domain.PaymentOrderStatusPaid {
		t.Errorf("expected status paid, got %s", po.Status)
	}
	if pub.publishedTopic != payment.TopicPaymentOrderPaid {
		t.Errorf("expected topic %q, got %q", payment.TopicPaymentOrderPaid, pub.publishedTopic)
	}
	// Verify audit log.
	if len(audit.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(audit.entries))
	}
	if audit.entries[0].Action != "paid" {
		t.Errorf("expected audit action 'paid', got %q", audit.entries[0].Action)
	}
}

func TestPayUsecase_Execute_Conflict_NotApproved(t *testing.T) {
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{}
	poRepo := &mockPaymentOrderRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentOrder, error) {
			po := fakePaymentOrder(uuid.New())
			po.Status = domain.PaymentOrderStatusDraft
			return po, nil
		},
	}

	uc := payment.NewPayUsecase(poRepo, pub, audit)
	_, err := uc.Execute(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestPayUsecase_Execute_NotFound(t *testing.T) {
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{}
	poRepo := &mockPaymentOrderRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentOrder, error) {
			return nil, domain.ErrNotFound
		},
	}

	uc := payment.NewPayUsecase(poRepo, pub, audit)
	_, err := uc.Execute(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// RejectUsecase tests
// ---------------------------------------------------------------------------

func TestRejectUsecase_Execute_OK(t *testing.T) {
	tenantID := uuid.New()
	poID := uuid.New()
	rejectedBy := uuid.New()
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{}

	poRepo := &mockPaymentOrderRepo{
		updateStatusFn: func(_ context.Context, id, tid uuid.UUID, from, to domain.PaymentOrderStatus, _ *uuid.UUID) (*domain.PaymentOrder, error) {
			po := fakePaymentOrder(tid)
			po.ID = id
			po.Status = domain.PaymentOrderStatusRejected
			return po, nil
		},
	}

	uc := payment.NewRejectUsecase(poRepo, pub, audit)
	po, err := uc.Execute(context.Background(), poID, tenantID, rejectedBy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if po.Status != domain.PaymentOrderStatusRejected {
		t.Errorf("expected status rejected, got %s", po.Status)
	}
	if pub.publishedTopic != payment.TopicPaymentOrderRejected {
		t.Errorf("expected topic %q, got %q", payment.TopicPaymentOrderRejected, pub.publishedTopic)
	}
	if pub.publishedKey != po.ID.String() {
		t.Errorf("expected kafka key %q, got %q", po.ID.String(), pub.publishedKey)
	}
	// Verify audit log.
	if len(audit.entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(audit.entries))
	}
	if audit.entries[0].Action != "rejected" {
		t.Errorf("expected audit action 'rejected', got %q", audit.entries[0].Action)
	}
	if audit.entries[0].UserID == nil || *audit.entries[0].UserID != rejectedBy {
		t.Errorf("expected audit user_id %s, got %v", rejectedBy, audit.entries[0].UserID)
	}
}

func TestRejectUsecase_Execute_NotFound(t *testing.T) {
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{}
	poRepo := &mockPaymentOrderRepo{
		updateStatusFn: func(_ context.Context, _, _ uuid.UUID, _, _ domain.PaymentOrderStatus, _ *uuid.UUID) (*domain.PaymentOrder, error) {
			return nil, domain.ErrNotFound
		},
	}

	uc := payment.NewRejectUsecase(poRepo, pub, audit)
	_, err := uc.Execute(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRejectUsecase_Execute_Conflict(t *testing.T) {
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{}
	poRepo := &mockPaymentOrderRepo{
		updateStatusFn: func(_ context.Context, _, _ uuid.UUID, _, _ domain.PaymentOrderStatus, _ *uuid.UUID) (*domain.PaymentOrder, error) {
			return nil, domain.ErrConflict
		},
	}

	uc := payment.NewRejectUsecase(poRepo, pub, audit)
	_, err := uc.Execute(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestRejectUsecase_Execute_ContinuesIfAuditFails(t *testing.T) {
	tenantID := uuid.New()
	poID := uuid.New()
	rejectedBy := uuid.New()
	pub := &mockPublisher{}
	audit := &mockAuditLogRepo{err: errors.New("audit db down")}

	poRepo := &mockPaymentOrderRepo{
		updateStatusFn: func(_ context.Context, id, tid uuid.UUID, _, _ domain.PaymentOrderStatus, _ *uuid.UUID) (*domain.PaymentOrder, error) {
			po := fakePaymentOrder(tid)
			po.ID = id
			po.Status = domain.PaymentOrderStatusRejected
			return po, nil
		},
	}

	uc := payment.NewRejectUsecase(poRepo, pub, audit)
	_, err := uc.Execute(context.Background(), poID, tenantID, rejectedBy)
	if err != nil {
		t.Errorf("audit failure should be swallowed, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// ListUsecase tests
// ---------------------------------------------------------------------------

func TestListUsecase_ExecutePaged_OK(t *testing.T) {
	tenantID := uuid.New()
	po := fakePaymentOrder(tenantID)

	poRepo := &mockPaymentOrderRepo{
		listPagedFn: func(_ context.Context, tid uuid.UUID, page, perPage int) (*domain.PaymentOrderListPage, error) {
			return &domain.PaymentOrderListPage{
				Data:    []domain.PaymentOrder{*po},
				Total:   1,
				Page:    page,
				PerPage: perPage,
			}, nil
		},
	}

	uc := payment.NewListUsecase(poRepo)
	result, err := uc.ExecutePaged(context.Background(), tenantID, 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Total)
	}
	if len(result.Data) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Data))
	}
}

func TestListUsecase_ExecutePaged_PropagatesError(t *testing.T) {
	poRepo := &mockPaymentOrderRepo{
		listPagedFn: func(_ context.Context, _ uuid.UUID, _, _ int) (*domain.PaymentOrderListPage, error) {
			return nil, errors.New("db error")
		},
	}

	uc := payment.NewListUsecase(poRepo)
	_, err := uc.ExecutePaged(context.Background(), uuid.New(), 1, 20)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Kafka event JSON serialization tests
// ---------------------------------------------------------------------------

func TestPaymentOrderCreatedEvent_JSONSerialization(t *testing.T) {
	poID := uuid.New()
	tenantID := uuid.New()
	invoiceID := uuid.New()

	event := domain.PaymentOrderCreatedEvent{
		PaymentOrderID: poID.String(),
		TenantID:       tenantID.String(),
		InvoiceID:      invoiceID.String(),
		Amount:         "5000000.00",
		Status:         "draft",
	}

	b, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	requiredKeys := []string{"payment_order_id", "tenant_id", "invoice_id", "amount", "status"}
	for _, key := range requiredKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON key %q missing from PaymentOrderCreatedEvent", key)
		}
	}

	if m["payment_order_id"] != poID.String() {
		t.Errorf("payment_order_id: got %v, want %v", m["payment_order_id"], poID.String())
	}

	// Amount must be JSON string.
	amtVal, ok := m["amount"].(string)
	if !ok {
		t.Errorf("amount: expected JSON string type, got %T", m["amount"])
	} else if amtVal != "5000000.00" {
		t.Errorf("amount: got %q, want %q", amtVal, "5000000.00")
	}

	// No camelCase keys.
	camelCaseKeys := []string{"paymentOrderId", "tenantId", "invoiceId"}
	for _, key := range camelCaseKeys {
		if _, ok := m[key]; ok {
			t.Errorf("camelCase key %q found — consumer expects snake_case only", key)
		}
	}
}

func TestPaymentOrderApprovedEvent_JSONSerialization(t *testing.T) {
	event := domain.PaymentOrderApprovedEvent{
		PaymentOrderID: uuid.New().String(),
		TenantID:       uuid.New().String(),
		InvoiceID:      uuid.New().String(),
		Amount:         "1000000.50",
		ApprovedBy:     uuid.New().String(),
	}

	b, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	requiredKeys := []string{"payment_order_id", "tenant_id", "invoice_id", "amount", "approved_by"}
	for _, key := range requiredKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON key %q missing from PaymentOrderApprovedEvent", key)
		}
	}
}

func TestPaymentOrderPaidEvent_JSONSerialization(t *testing.T) {
	event := domain.PaymentOrderPaidEvent{
		PaymentOrderID: uuid.New().String(),
		TenantID:       uuid.New().String(),
		InvoiceID:      uuid.New().String(),
		Amount:         "2000000.00",
		PaidAt:         "2026-07-11T10:00:00Z",
	}

	b, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	requiredKeys := []string{"payment_order_id", "tenant_id", "invoice_id", "amount", "paid_at"}
	for _, key := range requiredKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON key %q missing from PaymentOrderPaidEvent", key)
		}
	}
}

func TestPaymentOrderRejectedEvent_JSONSerialization(t *testing.T) {
	event := domain.PaymentOrderRejectedEvent{
		PaymentOrderID: uuid.New().String(),
		TenantID:       uuid.New().String(),
		InvoiceID:      uuid.New().String(),
		Amount:         "3000000.00",
	}

	b, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	requiredKeys := []string{"payment_order_id", "tenant_id", "invoice_id", "amount"}
	for _, key := range requiredKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON key %q missing from PaymentOrderRejectedEvent", key)
		}
	}
}
