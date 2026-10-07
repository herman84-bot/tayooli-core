// Package pospayment orchestrates non-cash (QRIS) payments for the POS.
//
// Multi-tenant model (ADR-008, Model B / BYO): each tenant uses their own
// gateway account, so funds settle to the tenant's bank and Tayooli never
// holds money. Credentials live in tenant_payment_configs; when a tenant has
// no usable credentials the flow runs in demo mode so the product can be
// exercised without a merchant account.
//
// Money safety: a POS sale is only allowed against a *settled* payment record
// (status completed) that has not been used before, and one payment can back
// at most one sale (Claim consumes it atomically).
package pospayment

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/midtrans"
)

// Provider is the gateway used for V1 (see docs/adr/008).
const Provider = "midtrans"

// posInvoiceRef labels POS-originated payment intents. The table was designed
// for invoice payments (invoice_id NOT NULL); POS has no invoice at charge
// time, so it carries this sentinel and is identified by its order_id.
const posInvoiceRef = "POS"

const (
	// qrisExpiryMinutes is how long a real QRIS charge stays payable.
	qrisExpiryMinutes = 5
	// demoExpiry is the nominal lifetime shown for demo charges.
	demoExpiry = 15 * time.Minute
)

// Sentinel errors the HTTP layer maps to status codes.
var (
	// ErrAmountMismatch means the settled payment amount differs from the
	// amount the checkout claims to collect.
	ErrAmountMismatch = errors.New("payment amount does not match")
	// ErrPaymentNotSettled means no settled payment backs this checkout
	// (unknown order, still pending, or already used for another sale).
	ErrPaymentNotSettled = errors.New("payment is not settled")
)

// StatusChecker re-verifies a transaction against the gateway (source of truth).
type StatusChecker interface {
	VerifyTransactionStatus(ctx context.Context, orderID string) (*midtrans.TransactionStatus, error)
}

// QrisCharger creates a dynamic QRIS charge.
type QrisCharger interface {
	CreateChargeQRIS(ctx context.Context, req midtrans.CreateChargeQRISRequest) (*midtrans.CreateChargeQRISResponse, error)
}

// Client bundles the gateway operations this package needs.
type Client interface {
	StatusChecker
	QrisCharger
}

// ClientFactory builds a client from a tenant's own credentials.
type ClientFactory func(serverKey, clientKey string, isProduction bool) Client

// DefaultClientFactory uses the real Midtrans SDK.
func DefaultClientFactory(serverKey, clientKey string, isProduction bool) Client {
	return midtrans.NewClient(serverKey, clientKey, isProduction)
}

// Usecase implements POS payment intents.
type Usecase struct {
	repo    domain.PaymentGatewayRepository
	factory ClientFactory
	appURL  string
	now     func() time.Time
}

// New builds the usecase. factory may be nil (real Midtrans client is used).
func New(repo domain.PaymentGatewayRepository, factory ClientFactory, appURL string) *Usecase {
	if factory == nil {
		factory = DefaultClientFactory
	}
	return &Usecase{
		repo:    repo,
		factory: factory,
		appURL:  strings.TrimRight(appURL, "/"),
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// Charge describes a payment intent handed to the cashier UI.
type Charge struct {
	OrderID     string          `json:"order_id"`
	Provider    string          `json:"provider"`
	Method      string          `json:"method"`
	Amount      decimal.Decimal `json:"amount"`
	Status      string          `json:"status"`
	Demo        bool            `json:"demo"`
	QRString    string          `json:"qr_string,omitempty"`
	PaymentLink string          `json:"payment_link,omitempty"`
	ExpiresAt   time.Time       `json:"expires_at"`
}

// StatusResult is the polling payload for the cashier UI.
type StatusResult struct {
	OrderID     string          `json:"order_id"`
	Status      string          `json:"status"`
	Amount      decimal.Decimal `json:"amount"`
	Method      string          `json:"method"`
	Demo        bool            `json:"demo"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
}

// tenantGateway holds the resolved per-tenant gateway credentials.
type tenantGateway struct {
	serverKey string
	clientKey string
	isProd    bool
}

// usable reports whether real gateway calls are possible.
func (g tenantGateway) usable() bool { return g.serverKey != "" }

// loadGateway resolves the tenant's Midtrans credentials. A missing config is
// not an error: it simply means demo mode.
func (u *Usecase) loadGateway(ctx context.Context, tenantID uuid.UUID) (tenantGateway, error) {
	cfg, err := u.repo.GetConfig(ctx, tenantID, Provider)
	if errors.Is(err, domain.ErrNotFound) {
		return tenantGateway{}, nil
	}
	if err != nil {
		return tenantGateway{}, err
	}
	if cfg == nil || !cfg.IsActive {
		return tenantGateway{}, nil
	}
	g := tenantGateway{isProd: cfg.IsProduction}
	if cfg.ServerKey != nil {
		g.serverKey = strings.TrimSpace(*cfg.ServerKey)
	}
	if cfg.ClientKey != nil {
		g.clientKey = strings.TrimSpace(*cfg.ClientKey)
	}
	return g, nil
}

// RequiresVerifiedPayment reports whether the tenant has usable gateway
// credentials. When true, non-cash checkout must carry a settled payment so a
// cashier can never mark a gateway sale paid by hand.
func (u *Usecase) RequiresVerifiedPayment(ctx context.Context, tenantID uuid.UUID) bool {
	g, err := u.loadGateway(ctx, tenantID)
	return err == nil && g.usable()
}

// CreateCharge creates a QRIS payment intent for the given amount.
//
// Real credentials → dynamic QRIS from the gateway. No credentials → demo
// intent pointing at the local demo checkout page. The transaction row is
// written before the gateway call so a webhook can always be matched.
func (u *Usecase) CreateCharge(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal, method string) (*Charge, error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method != "QRIS" {
		return nil, fmt.Errorf("%w: metode %s belum didukung (gunakan QRIS)", domain.ErrInvalidInput, method)
	}
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("%w: nominal harus lebih dari 0", domain.ErrInvalidInput)
	}

	gw, err := u.loadGateway(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("gagal memuat konfigurasi gateway: %w", err)
	}
	demo := !gw.usable()

	orderID := midtrans.BuildOrderID(tenantID)
	charge := &Charge{
		OrderID:  orderID,
		Provider: Provider,
		Method:   method,
		Amount:   amount,
		Status:   "pending",
		Demo:     demo,
	}

	// Persist first: an unrecognised order id would make a later webhook
	// un-matchable ("unknown transaction") and money invisible.
	tx := &domain.PaymentTransaction{
		TenantID:  tenantID,
		Provider:  Provider,
		InvoiceID: posInvoiceRef,
		OrderID:   orderID,
		Amount:    amount,
		Method:    method,
		Status:    "pending",
		IsDemo:    demo,
	}

	if demo {
		charge.PaymentLink = fmt.Sprintf(
			"%s/payments/demo/%s?provider=%s&amount=%d&method=qris",
			u.appURL, url.PathEscape(orderID), Provider, amount.IntPart(),
		)
		tx.PaymentLink = &charge.PaymentLink
		charge.ExpiresAt = u.now().Add(demoExpiry)
	} else {
		charge.ExpiresAt = u.now().Add(time.Duration(qrisExpiryMinutes) * time.Minute)
	}

	if err := u.repo.CreateTransaction(ctx, tx); err != nil {
		return nil, fmt.Errorf("gagal menyimpan transaksi pembayaran: %w", err)
	}

	if demo {
		return charge, nil
	}

	res, err := u.factory(gw.serverKey, gw.clientKey, gw.isProd).CreateChargeQRIS(ctx, midtrans.CreateChargeQRISRequest{
		OrderID:         orderID,
		Amount:          amount,
		ExpiryMinutes:   qrisExpiryMinutes,
		ItemDescription: "Penjualan POS",
	})
	if err != nil {
		// Mark the intent failed so it never looks payable.
		_ = u.repo.UpdateTransactionStatus(ctx, tenantID, orderID, "failed", nil)
		return nil, fmt.Errorf("gagal membuat QRIS: %w", err)
	}

	charge.QRString = res.QRString
	if !res.ExpiresAt.IsZero() {
		charge.ExpiresAt = res.ExpiresAt
	}
	return charge, nil
}

// Status returns the stored status, re-verifying a pending real transaction
// against the gateway so a missed webhook cannot strand a paid sale.
func (u *Usecase) Status(ctx context.Context, tenantID uuid.UUID, orderID string) (*StatusResult, error) {
	tx, err := u.repo.GetTransactionByOrderID(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	if tx.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}

	res := &StatusResult{
		OrderID:     tx.OrderID,
		Status:      tx.Status,
		Amount:      tx.Amount,
		Method:      tx.Method,
		Demo:        tx.IsDemo,
		CompletedAt: tx.CompletedAt,
	}

	// Only a pending, non-demo transaction can be confirmed by the gateway.
	if tx.Status != "pending" || tx.IsDemo {
		return res, nil
	}

	gw, err := u.loadGateway(ctx, tenantID)
	if err != nil || !gw.usable() {
		return res, nil
	}
	st, err := u.factory(gw.serverKey, gw.clientKey, gw.isProd).VerifyTransactionStatus(ctx, orderID)
	if err != nil {
		// Transient gateway failure: keep serving the stored status so the
		// cashier UI can keep polling; the webhook remains the backstop.
		return res, nil
	}
	if st.Amount != tx.Amount.IntPart() {
		return res, nil
	}

	switch {
	case midtrans.IsPaidStatus(st.Status):
		now := u.now()
		if err := u.repo.UpdateTransactionStatus(ctx, tenantID, orderID, "completed", &now); err == nil {
			res.Status = "completed"
			res.CompletedAt = &now
		}
	case st.Status == "expire":
		if err := u.repo.UpdateTransactionStatus(ctx, tenantID, orderID, "expired", nil); err == nil {
			res.Status = "expired"
		}
	case st.Status == "deny" || st.Status == "cancel" || st.Status == "failure":
		if err := u.repo.UpdateTransactionStatus(ctx, tenantID, orderID, "failed", nil); err == nil {
			res.Status = "failed"
		}
	}
	return res, nil
}

// SimulateDemo marks a demo payment as settled so the full flow can be
// exercised without a merchant account. It is refused for tenants that have
// real credentials, so it can never fake a production payment.
func (u *Usecase) SimulateDemo(ctx context.Context, tenantID uuid.UUID, orderID string) error {
	tx, err := u.repo.GetTransactionByOrderID(ctx, tenantID, orderID)
	if err != nil {
		return err
	}
	if tx.TenantID != tenantID {
		return domain.ErrNotFound
	}
	if !tx.IsDemo {
		return fmt.Errorf("%w: simulasi hanya tersedia saat gateway belum dikonfigurasi", domain.ErrForbidden)
	}
	if gw, err := u.loadGateway(ctx, tenantID); err == nil && gw.usable() {
		return fmt.Errorf("%w: gateway sudah dikonfigurasi, simulasikan lewat gateway", domain.ErrForbidden)
	}
	switch tx.Status {
	case "completed", "consumed":
		return nil // idempotent
	case "pending":
	default:
		return fmt.Errorf("%w: transaksi berstatus %s tidak dapat disimulasikan", domain.ErrInvalidInput, tx.Status)
	}
	now := u.now()
	if err := u.repo.UpdateTransactionStatus(ctx, tenantID, orderID, "completed", &now); err != nil {
		return err
	}
	return nil
}

// Claim atomically consumes a settled payment for a checkout. It returns
// ErrAmountMismatch when the amounts differ and ErrPaymentNotSettled when the
// payment is unsettled or was already used, so one payment backs one sale.
func (u *Usecase) Claim(ctx context.Context, tenantID uuid.UUID, orderID string, amount decimal.Decimal) error {
	tx, err := u.repo.GetTransactionByOrderID(ctx, tenantID, orderID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrPaymentNotSettled
		}
		return err
	}
	if tx.TenantID != tenantID {
		return ErrPaymentNotSettled
	}
	if !tx.Amount.Equal(amount) {
		return ErrAmountMismatch
	}
	if err := u.repo.ConsumeTransaction(ctx, tenantID, orderID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrPaymentNotSettled
		}
		return err
	}
	return nil
}

// Release returns a claimed payment to settled. The POS handler calls it when
// checkout fails after the claim (e.g. stock ran out) so the cashier can retry
// the same payment instead of losing it.
func (u *Usecase) Release(ctx context.Context, tenantID uuid.UUID, orderID string) error {
	tx, err := u.repo.GetTransactionByOrderID(ctx, tenantID, orderID)
	if err != nil {
		return err
	}
	completedAt := tx.CompletedAt
	if completedAt == nil {
		now := u.now()
		completedAt = &now
	}
	return u.repo.UpdateTransactionStatus(ctx, tenantID, orderID, "completed", completedAt)
}
