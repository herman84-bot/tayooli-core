package invoice

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// InferenceTriggerer abstracts the AI usecase to avoid a circular import with
// the ai package.  Only TriggerInference is needed — the invoice facade fires
// it as a best-effort side-effect after OCR processing.
type InferenceTriggerer interface {
	TriggerInference(ctx context.Context, invoiceID, tenantID, amount, vendorID, extractedText string) (*InferenceResponse, error)
}

// InferenceResponse mirrors the AI ingest response without importing the ai package.
type InferenceResponse struct {
	JobID  string
	Status string
}

// ApprovalAmountThreshold is the minimum invoice amount that triggers the
// automatic creation of an approval request. Denominations are the invoice's
// own currency unit; the reference workflow is IDR (100,000,000).
var ApprovalAmountThreshold = decimal.NewFromInt(100_000_000)

// ApprovalRequester creates an approval request for a newly created invoice.
// It is injected as a function to avoid importing the approval usecase
// package (which would create a dependency cycle through domain types).
type ApprovalRequester func(ctx context.Context, tenantID, invoiceID uuid.UUID, requestedBy uuid.UUID) error

// Facade is a convenience wrapper that satisfies the handler's invoiceUsecase
// interface by delegating to the individual use-case structs.
// repo is stored directly on the Facade so that auxiliary methods such as
// HandleOCRResult can access it without requiring a dedicated sub-struct.
type Facade struct {
	repo        domain.InvoiceRepository
	list        *ListUsecase
	create      *CreateUsecase
	approve     *ApproveUsecase
	reject      *RejectUsecase
	autoApprove *AutoApproveUsecase
	requeue     *RequeueUsecase
	aiTrigger          InferenceTriggerer // optional — nil means AI disabled
	approvalRequester  ApprovalRequester  // optional — nil disables auto-approval-request creation
}

// FacadeOption is a functional option for NewFacade.
type FacadeOption func(*Facade)

// WithAITrigger injects an AI inference triggerer into the Facade.
func WithAITrigger(t InferenceTriggerer) FacadeOption {
	return func(f *Facade) { f.aiTrigger = t }
}

// WithApprovalRequester injects the approval-request creator used to
// auto-create approval requests for high-value invoices.
func WithApprovalRequester(fn ApprovalRequester) FacadeOption {
	return func(f *Facade) { f.approvalRequester = fn }
}

func NewFacade(repo domain.InvoiceRepository, publisher EventPublisher, opts ...FacadeOption) *Facade {
	approve := NewApproveUsecase(repo, publisher)
	f := &Facade{
		repo:        repo,
		list:        NewListUsecase(repo),
		create:      NewCreateUsecase(repo, publisher),
		approve:     approve,
		reject:      NewRejectUsecase(repo, publisher),
		autoApprove: NewAutoApproveUsecase(repo, approve),
		requeue:     NewRequeueUsecase(repo, publisher),
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *Facade) ListInvoices(ctx context.Context, tenantID uuid.UUID) ([]domain.Invoice, error) {
	return f.list.Execute(ctx, tenantID)
}

// ListInvoicesPaged returns a paginated invoice list.
func (f *Facade) ListInvoicesPaged(ctx context.Context, tenantID uuid.UUID, page, perPage int) (*domain.InvoiceListPage, error) {
	return f.list.ExecutePaged(ctx, tenantID, page, perPage)
}

func (f *Facade) GetInvoice(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	return f.repo.GetByID(ctx, id, tenantID)
}

func (f *Facade) CreateInvoice(ctx context.Context, params domain.CreateInvoiceParams) (*domain.Invoice, error) {
	inv, err := f.create.Execute(ctx, params)
	if err != nil {
		return nil, err
	}

	// Auto-create a pending approval request for high-value invoices.
	// Best-effort: a failure here must NOT roll back the invoice write or fail
	// the HTTP request — it is only logged.
	if f.approvalRequester != nil && params.CreatedBy != nil && inv.Amount.GreaterThanOrEqual(ApprovalAmountThreshold) {
		if err := f.approvalRequester(ctx, inv.TenantID, inv.ID, *params.CreatedBy); err != nil {
			log.Error().Err(err).Str("invoice_id", inv.ID.String()).Msg("failed to auto-create approval request")
		}
	}

	return inv, nil
}

func (f *Facade) ApproveInvoice(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	return f.approve.Execute(ctx, id, tenantID)
}

func (f *Facade) RejectInvoice(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	return f.reject.Execute(ctx, id, tenantID)
}

func (f *Facade) AutoApproveInvoice(ctx context.Context, id, tenantID uuid.UUID) (*AutoApproveResult, error) {
	return f.autoApprove.Execute(ctx, AutoApproveInput{InvoiceID: id, TenantID: tenantID})
}

func (f *Facade) RequeueInvoice(ctx context.Context, id, tenantID uuid.UUID) (*domain.Invoice, error) {
	return f.requeue.Execute(ctx, id, tenantID)
}
