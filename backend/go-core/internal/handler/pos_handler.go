package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/pos"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/pospayment"
)

type POSHandler struct {
	uc *uc.Usecase
	// payments may be nil (handlers then answer 503 for payment endpoints and
	// skip the non-cash gate) — wired in cmd/api.
	payments *pospayment.Usecase
}

func NewPOSHandler(u *uc.Usecase) *POSHandler {
	return &POSHandler{uc: u}
}

// NewPOSHandlerWithPayments builds the handler with the gateway-aware flow.
func NewPOSHandlerWithPayments(u *uc.Usecase, p *pospayment.Usecase) *POSHandler {
	return &POSHandler{uc: u, payments: p}
}

// nonCashAmount sums the parts of a sale that are not paid in cash, i.e. the
// amount a gateway must have collected before a receipt may be issued.
func nonCashAmount(payments []uc.CheckoutPaymentRequest) decimal.Decimal {
	var total decimal.Decimal
	for _, p := range payments {
		if !strings.EqualFold(strings.TrimSpace(p.Method), "CASH") {
			total = total.Add(p.Amount)
		}
	}
	return total
}

// Checkout creates a POS sale.
//
// Money safety: a non-cash sale must name a settled gateway payment that has
// not been redeemed yet, and for tenants with a gateway account a non-cash sale
// is rejected without one — there is no manual "paid" confirmation that can be
// clicked for a gateway payment. Cash sales are untouched.
func (h *POSHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, _ := appMiddleware.GetUserID(r.Context())

	var req uc.CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	claimed := ""
	if h.payments != nil {
		if req.PaymentOrderID != "" {
			// Reserve the payment first: it can back at most one sale.
			if err := h.payments.Claim(r.Context(), tenantID, req.PaymentOrderID, nonCashAmount(req.Payments)); err != nil {
				switch {
				case errors.Is(err, pospayment.ErrAmountMismatch):
					http.Error(w, "nominal pembayaran tidak sesuai dengan keranjang", http.StatusBadRequest)
					return
				case errors.Is(err, pospayment.ErrPaymentNotSettled), errors.Is(err, domain.ErrNotFound):
					http.Error(w, "pembayaran belum lunas atau sudah dipakai", http.StatusConflict)
					return
				default:
					http.Error(w, "gagal memverifikasi pembayaran", http.StatusInternalServerError)
					return
				}
			}
			claimed = req.PaymentOrderID
		} else if nonCashAmount(req.Payments).GreaterThan(decimal.Zero) && h.payments.RequiresVerifiedPayment(r.Context(), tenantID) {
			// Tenant runs a real gateway account: a gateway sale may only be
			// written after settlement.
			http.Error(w, "pembayaran non-tunai wajib diverifikasi gateway", http.StatusConflict)
			return
		}
	}

	res, err := h.uc.Checkout(r.Context(), tenantID, userID, req)
	if err != nil {
		if claimed != "" {
			// Give the payment back so the cashier can retry the same QR
			// (e.g. after fixing a stock shortage) instead of losing it.
			_ = h.payments.Release(r.Context(), tenantID, claimed)
		}
		if errors.Is(err, domain.ErrInsufficientStock) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

type createPOSPaymentRequest struct {
	Amount decimal.Decimal `json:"amount"`
	Method string          `json:"method"`
}

// CreatePayment starts a QRIS payment intent for the current cart total.
// Responds 503 when the payment usecase is not wired.
func (h *POSHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.payments == nil {
		respondError(w, r, http.StatusServiceUnavailable, "payment service unavailable")
		return
	}

	var req createPOSPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	charge, err := h.payments.CreateCharge(r.Context(), tenantID, req.Amount, req.Method)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, http.StatusCreated, charge)
}

// PaymentStatus is the cashier's poll target. It is read-only: a pending real
// payment is re-checked against the gateway so a missed webhook cannot strand
// a paid sale.
func (h *POSHandler) PaymentStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.payments == nil {
		respondError(w, r, http.StatusServiceUnavailable, "payment service unavailable")
		return
	}
	orderID := strings.TrimSpace(chi.URLParam(r, "orderId"))
	if orderID == "" {
		http.Error(w, "missing order id", http.StatusBadRequest)
		return
	}

	res, err := h.payments.Status(r.Context(), tenantID, orderID)
	if errors.Is(err, domain.ErrNotFound) {
		http.Error(w, "payment not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to read payment status", http.StatusInternalServerError)
		return
	}
	respondJSON(w, http.StatusOK, res)
}

// SimulatePayment settles a *demo* payment so the flow can be exercised
// without a merchant account. It refuses tenants that have real credentials,
// so it can never fake a live payment.
func (h *POSHandler) SimulatePayment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.payments == nil {
		respondError(w, r, http.StatusServiceUnavailable, "payment service unavailable")
		return
	}
	orderID := strings.TrimSpace(chi.URLParam(r, "orderId"))
	if orderID == "" {
		http.Error(w, "missing order id", http.StatusBadRequest)
		return
	}

	if err := h.payments.SimulateDemo(r.Context(), tenantID, orderID); err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			http.Error(w, "payment not found", http.StatusNotFound)
		case errors.Is(err, domain.ErrForbidden):
			http.Error(w, err.Error(), http.StatusForbidden)
		case errors.Is(err, domain.ErrInvalidInput):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "failed to simulate payment", http.StatusInternalServerError)
		}
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "completed"})
}

func (h *POSHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	orders, err := h.uc.ListOrders(r.Context(), tenantID, limit)
	if err != nil {
		http.Error(w, "failed to list pos orders", http.StatusInternalServerError)
		return
	}
	if orders == nil {
		orders = []domain.POSOrder{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"data": orders,
	})
}

func (h *POSHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appMiddleware.GetTenantID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid order id", http.StatusBadRequest)
		return
	}

	order, err := h.uc.GetOrderByID(r.Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "order not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get order", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}
