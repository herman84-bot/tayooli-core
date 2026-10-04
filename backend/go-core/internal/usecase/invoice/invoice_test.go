package invoice_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/invoice"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockInvoiceRepo struct {
	listFn       func(ctx context.Context, tenantID uuid.UUID) ([]domain.Invoice, error)
	listPagedFn  func(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.InvoiceListPage, error)
	getByIDFn    func(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error)
	createFn     func(ctx context.Context, params domain.CreateInvoiceParams) (*domain.Invoice, error)
	approveFn    func(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error)
	rejectFn     func(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error)
	updateOCRFn  func(ctx context.Context, tenantID, invoiceID uuid.UUID, score float64, extractedText, status string) error
}

func (m *mockInvoiceRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Invoice, error) {
	return m.listFn(ctx, tenantID)
}
func (m *mockInvoiceRepo) ListPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.InvoiceListPage, error) {
	if m.listPagedFn != nil {
		return m.listPagedFn(ctx, tenantID, page, perPage)
	}
	return &domain.InvoiceListPage{Data: []domain.Invoice{}, Total: 0, Page: page, PerPage: perPage}, nil
}
func (m *mockInvoiceRepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	return m.getByIDFn(ctx, id, tenantID)
}
func (m *mockInvoiceRepo) Create(ctx context.Context, params domain.CreateInvoiceParams) (*domain.Invoice, error) {
	return m.createFn(ctx, params)
}
func (m *mockInvoiceRepo) Approve(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	return m.approveFn(ctx, id, tenantID)
}
func (m *mockInvoiceRepo) Reject(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	if m.rejectFn != nil {
		return m.rejectFn(ctx, id, tenantID)
	}
	return nil, domain.ErrNotFound
}
func (m *mockInvoiceRepo) UpdateOCRResult(ctx context.Context, tenantID, invoiceID uuid.UUID, score float64, extractedText, status string) error {
	if m.updateOCRFn != nil {
		return m.updateOCRFn(ctx, tenantID, invoiceID, score, extractedText, status)
	}
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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func fakeInvoice(tenantID uuid.UUID) *domain.Invoice {
	return &domain.Invoice{
		ID:            uuid.New(),
		TenantID:      tenantID,
		VendorID:      "vendor-1",
		InvoiceNumber: "INV-001",
		Amount:        decimal.New(10_000, 0),
		Currency:      "IDR",
		Status:        domain.StatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// ---------------------------------------------------------------------------
// ListUsecase tests
// ---------------------------------------------------------------------------

func TestListUsecase_Execute_ReturnsOnlyTenantInvoices(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()

	invA := fakeInvoice(tenantA)
	invB := fakeInvoice(tenantB)

	repo := &mockInvoiceRepo{
		listFn: func(_ context.Context, tenantID uuid.UUID) ([]domain.Invoice, error) {
			// Simulate DB-level tenant isolation
			if tenantID == tenantA {
				return []domain.Invoice{*invA}, nil
			}
			return []domain.Invoice{*invB}, nil
		},
	}

	uc := invoice.NewListUsecase(repo)

	resultA, err := uc.Execute(context.Background(), tenantA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resultA) != 1 {
		t.Fatalf("expected 1 invoice, got %d", len(resultA))
	}
	if resultA[0].TenantID != tenantA {
		t.Errorf("tenant isolation broken: got tenant_id %v, want %v", resultA[0].TenantID, tenantA)
	}

	resultB, err := uc.Execute(context.Background(), tenantB)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resultB[0].TenantID != tenantB {
		t.Errorf("tenant isolation broken: got tenant_id %v, want %v", resultB[0].TenantID, tenantB)
	}
}

func TestListUsecase_Execute_PassesTenantIDToRepo(t *testing.T) {
	tenantID := uuid.New()
	var capturedTenantID uuid.UUID

	repo := &mockInvoiceRepo{
		listFn: func(_ context.Context, id uuid.UUID) ([]domain.Invoice, error) {
			capturedTenantID = id
			return nil, nil
		},
	}

	uc := invoice.NewListUsecase(repo)
	_, _ = uc.Execute(context.Background(), tenantID)

	if capturedTenantID != tenantID {
		t.Errorf("usecase did not forward tenantID: got %v, want %v", capturedTenantID, tenantID)
	}
}

func TestListUsecase_Execute_PropagatesRepoError(t *testing.T) {
	repo := &mockInvoiceRepo{
		listFn: func(_ context.Context, _ uuid.UUID) ([]domain.Invoice, error) {
			return nil, errors.New("db error")
		},
	}
	uc := invoice.NewListUsecase(repo)
	_, err := uc.Execute(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// CreateUsecase tests
// ---------------------------------------------------------------------------

func TestCreateUsecase_Execute_PublishesEvent(t *testing.T) {
	tenantID := uuid.New()
	pub := &mockPublisher{}

	repo := &mockInvoiceRepo{
		createFn: func(_ context.Context, p domain.CreateInvoiceParams) (*domain.Invoice, error) {
			return fakeInvoice(p.TenantID), nil
		},
	}

	uc := invoice.NewCreateUsecase(repo, pub)
	params := domain.CreateInvoiceParams{
		TenantID:      tenantID,
		VendorID:      "vendor-1",
		InvoiceNumber: "INV-001",
		Amount:        decimal.New(5000, 0),
		Currency:      "IDR",
	}

	inv, err := uc.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv == nil {
		t.Fatal("expected invoice, got nil")
	}
	if pub.publishedTopic != invoice.TopicInvoiceCreated {
		t.Errorf("expected topic %q, got %q", invoice.TopicInvoiceCreated, pub.publishedTopic)
	}
	// Key must be the invoice UUID, not a timestamp
	if pub.publishedKey != inv.ID.String() {
		t.Errorf("expected kafka key %q (invoice UUID), got %q", inv.ID.String(), pub.publishedKey)
	}
}

func TestCreateUsecase_Execute_RejectsInvalidInput(t *testing.T) {
	pub := &mockPublisher{}
	repo := &mockInvoiceRepo{}
	uc := invoice.NewCreateUsecase(repo, pub)

	cases := []domain.CreateInvoiceParams{
		{TenantID: uuid.New(), VendorID: "", InvoiceNumber: "INV-001", Amount: decimal.New(100, 0), Currency: "IDR"},
		{TenantID: uuid.New(), VendorID: "v1", InvoiceNumber: "", Amount: decimal.New(100, 0), Currency: "IDR"},
		{TenantID: uuid.New(), VendorID: "v1", InvoiceNumber: "INV-001", Amount: decimal.Zero, Currency: "IDR"},
		{TenantID: uuid.New(), VendorID: "v1", InvoiceNumber: "INV-001", Amount: decimal.New(100, 0), Currency: ""},
	}

	for _, p := range cases {
		_, err := uc.Execute(context.Background(), p)
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput for params %+v, got %v", p, err)
		}
	}
}

func TestCreateUsecase_Execute_ContinuesIfPublishFails(t *testing.T) {
	tenantID := uuid.New()
	pub := &mockPublisher{err: errors.New("kafka down")}

	repo := &mockInvoiceRepo{
		createFn: func(_ context.Context, p domain.CreateInvoiceParams) (*domain.Invoice, error) {
			return fakeInvoice(p.TenantID), nil
		},
	}
	uc := invoice.NewCreateUsecase(repo, pub)

	_, err := uc.Execute(context.Background(), domain.CreateInvoiceParams{
		TenantID:      tenantID,
		VendorID:      "v1",
		InvoiceNumber: "INV-002",
		Amount:        decimal.New(100, 0),
		Currency:      "IDR",
	})
	if err != nil {
		t.Errorf("publish failure should be swallowed, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// ApproveUsecase tests
// ---------------------------------------------------------------------------

func TestApproveUsecase_Execute_PublishesEvent(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()
	pub := &mockPublisher{}

	repo := &mockInvoiceRepo{
		approveFn: func(_ context.Context, id, tid uuid.UUID) (*domain.Invoice, error) {
			inv := fakeInvoice(tid)
			inv.ID = id
			inv.Status = domain.StatusApproved
			return inv, nil
		},
	}

	uc := invoice.NewApproveUsecase(repo, pub)
	inv, err := uc.Execute(context.Background(), invoiceID, tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Status != domain.StatusApproved {
		t.Errorf("expected status approved, got %s", inv.Status)
	}
	if pub.publishedTopic != invoice.TopicInvoiceApproved {
		t.Errorf("expected topic %q, got %q", invoice.TopicInvoiceApproved, pub.publishedTopic)
	}
	// Key must be the invoice UUID for partition ordering
	if pub.publishedKey != inv.ID.String() {
		t.Errorf("expected kafka key %q (invoice UUID), got %q", inv.ID.String(), pub.publishedKey)
	}
}

func TestApproveUsecase_Execute_EnforcesTenantID(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	invoiceID := uuid.New()
	pub := &mockPublisher{}

	repo := &mockInvoiceRepo{
		approveFn: func(_ context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
			// Simulate DB returning ErrNotFound when tenant doesn't own the invoice
			if tenantID != tenantA {
				return nil, domain.ErrNotFound
			}
			return fakeInvoice(tenantID), nil
		},
	}

	uc := invoice.NewApproveUsecase(repo, pub)

	// Tenant B should not be able to approve tenant A's invoice
	_, err := uc.Execute(context.Background(), invoiceID, tenantB)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound for cross-tenant approve, got %v", err)
	}
}

func TestApproveUsecase_Execute_PropagatesRepoError(t *testing.T) {
	pub := &mockPublisher{}
	repo := &mockInvoiceRepo{
		approveFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
			return nil, domain.ErrNotFound
		},
	}
	uc := invoice.NewApproveUsecase(repo, pub)
	_, err := uc.Execute(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// HandleOCRResult tests
// ---------------------------------------------------------------------------

func TestHandleOCRResult_Success(t *testing.T) {
	invoiceID := uuid.New()
	tenantID := uuid.New()

	repo := &mockInvoiceRepo{
		updateOCRFn: func(_ context.Context, _, _ uuid.UUID, _ float64, _, _ string) error {
			return nil
		},
	}
	pub := &mockPublisher{}
	f := invoice.NewFacade(repo, pub)

	err := f.HandleOCRResult(context.Background(), invoiceID, tenantID, 0.95, "Invoice text content", "ai_processed")
	if err != nil {
		t.Errorf("expected nil error on success, got %v", err)
	}
}

func TestHandleOCRResult_NotFound(t *testing.T) {
	invoiceID := uuid.New()
	tenantID := uuid.New()

	repo := &mockInvoiceRepo{
		updateOCRFn: func(_ context.Context, _, _ uuid.UUID, _ float64, _, _ string) error {
			return domain.ErrNotFound
		},
	}
	pub := &mockPublisher{}
	f := invoice.NewFacade(repo, pub)

	err := f.HandleOCRResult(context.Background(), invoiceID, tenantID, 0.0, "", "ai_failed")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected errors.Is(err, domain.ErrNotFound) to be true, got %v", err)
	}
}

// TestHandleOCRResult_WrongTenant verifies that HandleOCRResult forwards
// exactly the tenantID it was given to repo.UpdateOCRResult.
// This is the usecase-layer tenant isolation guarantee: the caller (the Kafka
// consumer) owns the tenantID extracted from the event payload; the usecase
// must not substitute or ignore it.
func TestHandleOCRResult_WrongTenant(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	invoiceID := uuid.New()

	var capturedTenantID uuid.UUID
	var capturedInvoiceID uuid.UUID

	repo := &mockInvoiceRepo{
		updateOCRFn: func(_ context.Context, tenantID, invID uuid.UUID, _ float64, _, _ string) error {
			capturedTenantID = tenantID
			capturedInvoiceID = invID
			// Simulate DB isolation: only tenant A's records exist.
			if tenantID != tenantA {
				return domain.ErrNotFound
			}
			return nil
		},
	}
	pub := &mockPublisher{}
	f := invoice.NewFacade(repo, pub)

	// --- Call as tenant A (owner) — must succeed ---
	err := f.HandleOCRResult(context.Background(), invoiceID, tenantA, 0.9, "text", "ai_processed")
	if err != nil {
		t.Fatalf("tenantA should succeed, got %v", err)
	}
	if capturedTenantID != tenantA {
		t.Errorf("HandleOCRResult forwarded wrong tenantID to repo: got %v, want %v", capturedTenantID, tenantA)
	}
	if capturedInvoiceID != invoiceID {
		t.Errorf("HandleOCRResult forwarded wrong invoiceID to repo: got %v, want %v", capturedInvoiceID, invoiceID)
	}

	// --- Call as tenant B (non-owner) — must surface ErrNotFound (no cross-tenant read) ---
	err = f.HandleOCRResult(context.Background(), invoiceID, tenantB, 0.9, "text", "ai_processed")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("tenantB accessing tenantA invoice: expected ErrNotFound, got %v", err)
	}
	if capturedTenantID != tenantB {
		t.Errorf("HandleOCRResult did not forward tenantB to repo: got %v, want %v", capturedTenantID, tenantB)
	}
}

// TestCreateUsecase_Execute_EventAmountIsDecimalString verifies that the
// InvoiceCreatedEvent published to Kafka carries Amount as a decimal string,
// not a float64. If Amount were serialized as a JSON number the Python worker
// could silently lose sub-cent precision (e.g. 12345.6789 → 12345.67).
func TestCreateUsecase_Execute_EventAmountIsDecimalString(t *testing.T) {
	tenantID := uuid.New()
	const precisionAmount = "12345.6789"
	parsedAmount, _ := decimal.NewFromString(precisionAmount)

	pub := &mockPublisher{}
	repo := &mockInvoiceRepo{
		createFn: func(_ context.Context, p domain.CreateInvoiceParams) (*domain.Invoice, error) {
			inv := fakeInvoice(p.TenantID)
			inv.Amount = p.Amount // echo the exact input amount
			return inv, nil
		},
	}

	uc := invoice.NewCreateUsecase(repo, pub)
	_, err := uc.Execute(context.Background(), domain.CreateInvoiceParams{
		TenantID:      tenantID,
		VendorID:      "vendor-1",
		InvoiceNumber: "INV-999",
		Amount:        parsedAmount,
		Currency:      "IDR",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	event, ok := pub.publishedPayload.(domain.InvoiceCreatedEvent)
	if !ok {
		t.Fatalf("publishedPayload is %T, want domain.InvoiceCreatedEvent", pub.publishedPayload)
	}

	// Amount field must be the exact decimal string — never a float representation.
	if event.Amount != precisionAmount {
		t.Errorf("event.Amount = %q, want %q — decimal precision not preserved in Kafka payload",
			event.Amount, precisionAmount)
	}

	// Confirm the JSON encoding of the event has amount as a JSON string type.
	b, _ := json.Marshal(event)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	amtVal, isStr := m["amount"].(string)
	if !isStr {
		t.Errorf("InvoiceCreatedEvent JSON amount is %T, want string — Python consumer will receive wrong type",
			m["amount"])
	} else if amtVal != precisionAmount {
		t.Errorf("InvoiceCreatedEvent JSON amount = %q, want %q", amtVal, precisionAmount)
	}
}

// ---------------------------------------------------------------------------
// InvoiceRejectedEvent JSON serialization test
// ---------------------------------------------------------------------------

// TestInvoiceRejectedEvent_JSONSerialization verifies the Kafka contract for
// the invoice.rejected topic. Python consumers read snake_case keys; any
// rename in this struct is a breaking change. Amount must be a JSON string to
// prevent float precision loss — the same invariant as InvoiceCreatedEvent.
func TestInvoiceRejectedEvent_JSONSerialization(t *testing.T) {
	invID := uuid.New()
	tenantID := uuid.New()
	const amountStr = "98765.4321"

	event := domain.InvoiceRejectedEvent{
		InvoiceID: invID.String(),
		TenantID:  tenantID.String(),
		Amount:    amountStr,
	}

	b, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal InvoiceRejectedEvent failed: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Mandatory snake_case keys.
	requiredKeys := []string{"invoice_id", "tenant_id", "amount"}
	for _, key := range requiredKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON key %q missing from InvoiceRejectedEvent — consumer contract broken", key)
		}
	}

	// Values round-trip correctly.
	if m["invoice_id"] != invID.String() {
		t.Errorf("invoice_id: got %v, want %v", m["invoice_id"], invID.String())
	}
	if m["tenant_id"] != tenantID.String() {
		t.Errorf("tenant_id: got %v, want %v", m["tenant_id"], tenantID.String())
	}

	// Amount MUST be a JSON string, not a JSON number.
	amtVal, ok := m["amount"].(string)
	if !ok {
		t.Errorf("amount: expected JSON string type, got %T — float serialization regression", m["amount"])
	} else if amtVal != amountStr {
		t.Errorf("amount: got %q, want %q", amtVal, amountStr)
	}

	// camelCase keys must NOT appear.
	camelCaseKeys := []string{"invoiceId", "tenantId"}
	for _, key := range camelCaseKeys {
		if _, ok := m[key]; ok {
			t.Errorf("camelCase key %q found — Python consumer expects snake_case only", key)
		}
	}
}

// ---------------------------------------------------------------------------
// InvoiceCreatedEvent JSON serialization test
// ---------------------------------------------------------------------------

// TestInvoiceCreatedEvent_JSONSerialization verifies the Kafka contract
// between the Go producer and the Python AI worker consumer.
// Python reads snake_case keys; any rename in this struct is a breaking change.
func TestInvoiceCreatedEvent_JSONSerialization(t *testing.T) {
	invID := uuid.New()
	tenantID := uuid.New()

	event := domain.InvoiceCreatedEvent{
		InvoiceID:        invID.String(),
		TenantID:         tenantID.String(),
		Amount:           "12345.6700",
		Status:           "pending",
		RawDocumentBytes: "dGVzdA==",
		MimeType:         "application/pdf",
	}

	b, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Mandatory snake_case keys that the Python consumer reads via payload.get(...)
	requiredKeys := []string{"invoice_id", "tenant_id", "amount", "status"}
	for _, key := range requiredKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("JSON key %q missing — Python consumer contract broken", key)
		}
	}

	// Verify values round-trip correctly
	if m["invoice_id"] != invID.String() {
		t.Errorf("invoice_id: got %v, want %v", m["invoice_id"], invID.String())
	}
	if m["tenant_id"] != tenantID.String() {
		t.Errorf("tenant_id: got %v, want %v — tenant_id MUST be present in Kafka payload", m["tenant_id"], tenantID.String())
	}
	if m["status"] != "pending" {
		t.Errorf("status: got %v, want pending", m["status"])
	}

	// Regression: amount MUST be a JSON string, not a JSON number.
	// If amount were float64 the Python consumer would receive 12345.67 (number),
	// losing sub-cent precision. As string it is "12345.6700".
	amountVal, ok := m["amount"].(string)
	if !ok {
		t.Errorf("amount: expected JSON string type, got %T — float serialization regression", m["amount"])
	} else if amountVal != "12345.6700" {
		t.Errorf("amount: got %q, want %q", amountVal, "12345.6700")
	}

	// Optional fields: raw_document_bytes present when non-empty
	if m["raw_document_bytes"] != "dGVzdA==" {
		t.Errorf("raw_document_bytes: got %v, want dGVzdA==", m["raw_document_bytes"])
	}

	// Camel-case keys must NOT appear — that would break the Python consumer
	camelCaseKeys := []string{"invoiceId", "tenantId", "rawDocumentBytes", "mimeType"}
	for _, key := range camelCaseKeys {
		if _, ok := m[key]; ok {
			t.Errorf("camelCase key %q found in JSON — Python consumer expects snake_case only", key)
		}
	}
}
