package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/accounting/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/accounting/usecase"
)

type AccountHandler struct {
	useCase *usecase.AccountUseCase
}

func NewAccountHandler(uc *usecase.AccountUseCase) *AccountHandler {
	return &AccountHandler{useCase: uc}
}

func (h *AccountHandler) RegisterRoutes(r chi.Router) {
	r.Route("/accounting", func(r chi.Router) {
		r.Post("/accounts", h.CreateAccount)
		r.Get("/accounts", h.ListAccounts)
	})
}

func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var a domain.Account
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Assuming tenantID is passed via context from auth middleware
	tenantIDRaw := r.Context().Value("tenant_id")
	if tenantIDStr, ok := tenantIDRaw.(string); ok {
		a.TenantID = uuid.MustParse(tenantIDStr)
	}

	if err := h.useCase.CreateAccount(r.Context(), &a); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(a)
}

func (h *AccountHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	tenantIDRaw := r.Context().Value("tenant_id")
	tenantIDStr, _ := tenantIDRaw.(string)
	tenantID := uuid.MustParse(tenantIDStr)

	accounts, err := h.useCase.ListAccounts(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(accounts)
}
