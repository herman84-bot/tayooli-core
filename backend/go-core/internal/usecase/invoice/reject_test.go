package invoice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/invoice"
)

// ---------------------------------------------------------------------------
// RejectUsecase tests
// ---------------------------------------------------------------------------

func TestRejectUsecase_Execute_PendingToRejected(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()
	pub := &mockPublisher{}

	repo := &mockInvoiceRepo{
		rejectFn: func(_ context.Context, id, tid uuid.UUID) (*domain.Invoice, error) {
			inv := fakeInvoice(tid)
			inv.ID = id
			inv.Status = domain.StatusRejected
			return inv, nil
		},
	}

	uc := invoice.NewRejectUsecase(repo, pub)
	inv, err := uc.Execute(context.Background(), invoiceID, tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Status != domain.StatusRejected {
		t.Errorf("expected status rejected, got %s", inv.Status)
	}
	if pub.publishedTopic != invoice.TopicInvoiceRejected {
		t.Errorf("expected topic %q, got %q", invoice.TopicInvoiceRejected, pub.publishedTopic)
	}
	if pub.publishedKey != inv.ID.String() {
		t.Errorf("expected kafka key %q (invoice UUID), got %q", inv.ID.String(), pub.publishedKey)
	}
}

func TestRejectUsecase_Execute_PendingReviewToRejected(t *testing.T) {
	tenantID := uuid.New()
	invoiceID := uuid.New()
	pub := &mockPublisher{}

	repo := &mockInvoiceRepo{
		rejectFn: func(_ context.Context, id, tid uuid.UUID) (*domain.Invoice, error) {
			inv := fakeInvoice(tid)
			inv.ID = id
			inv.Status = domain.StatusRejected
			return inv, nil
		},
	}

	uc := invoice.NewRejectUsecase(repo, pub)
	inv, err := uc.Execute(context.Background(), invoiceID, tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Status != domain.StatusRejected {
		t.Errorf("expected status rejected, got %s", inv.Status)
	}
	if pub.publishedTopic != invoice.TopicInvoiceRejected {
		t.Errorf("expected topic %q, got %q", invoice.TopicInvoiceRejected, pub.publishedTopic)
	}
}

func TestRejectUsecase_Execute_RepoReturnsErrNotFound(t *testing.T) {
	pub := &mockPublisher{}
	repo := &mockInvoiceRepo{
		rejectFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
			return nil, domain.ErrNotFound
		},
	}

	uc := invoice.NewRejectUsecase(repo, pub)
	_, err := uc.Execute(context.Background(), uuid.New(), uuid.New())

	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	// Event must NOT be published when repo fails.
	if pub.publishedTopic != "" {
		t.Errorf("expected no event published on repo error, got topic %q", pub.publishedTopic)
	}
}

func TestRejectUsecase_Execute_RepoReturnsErrConflict(t *testing.T) {
	pub := &mockPublisher{}
	repo := &mockInvoiceRepo{
		rejectFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
			return nil, domain.ErrConflict
		},
	}

	uc := invoice.NewRejectUsecase(repo, pub)
	_, err := uc.Execute(context.Background(), uuid.New(), uuid.New())

	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict (invoice already approved), got %v", err)
	}
	// Event must NOT be published when repo fails.
	if pub.publishedTopic != "" {
		t.Errorf("expected no event published on repo conflict, got topic %q", pub.publishedTopic)
	}
}

func TestRejectUsecase_Execute_PublishFailureIsNonFatal(t *testing.T) {
	tenantID := uuid.New()
	pub := &mockPublisher{err: errors.New("kafka down")}

	repo := &mockInvoiceRepo{
		rejectFn: func(_ context.Context, id, tid uuid.UUID) (*domain.Invoice, error) {
			inv := fakeInvoice(tid)
			inv.ID = id
			inv.Status = domain.StatusRejected
			return inv, nil
		},
	}

	uc := invoice.NewRejectUsecase(repo, pub)
	_, err := uc.Execute(context.Background(), uuid.New(), tenantID)
	if err != nil {
		t.Errorf("publish failure should be swallowed, got %v", err)
	}
}

func TestRejectUsecase_Execute_EnforcesTenantID(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	invoiceID := uuid.New()
	pub := &mockPublisher{}

	repo := &mockInvoiceRepo{
		rejectFn: func(_ context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
			// Simulate DB returning ErrNotFound when tenant doesn't own the invoice.
			if tenantID != tenantA {
				return nil, domain.ErrNotFound
			}
			inv := fakeInvoice(tenantID)
			inv.ID = id
			inv.Status = domain.StatusRejected
			return inv, nil
		},
	}

	uc := invoice.NewRejectUsecase(repo, pub)

	// Tenant B must not be able to reject tenant A's invoice.
	_, err := uc.Execute(context.Background(), invoiceID, tenantB)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound for cross-tenant reject, got %v", err)
	}
}
