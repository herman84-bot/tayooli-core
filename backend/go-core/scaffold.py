import os
import re

base_dir = r"D:\Erp-Like-PAPER-ID\backend\go-core"
domain_dir = os.path.join(base_dir, "internal", "domain")
infra_dir = os.path.join(base_dir, "internal", "infra", "postgres")
usecase_dir = os.path.join(base_dir, "internal", "usecase")
handler_dir = os.path.join(base_dir, "internal", "handler")

os.makedirs(domain_dir, exist_ok=True)
os.makedirs(infra_dir, exist_ok=True)
os.makedirs(usecase_dir, exist_ok=True)
os.makedirs(handler_dir, exist_ok=True)
os.makedirs(os.path.join(usecase_dir, "customer"), exist_ok=True)
os.makedirs(os.path.join(usecase_dir, "sales_order"), exist_ok=True)
os.makedirs(os.path.join(usecase_dir, "sales_invoice"), exist_ok=True)

# 1. Domain
customer_domain = """package domain

import (
	"time"

	"github.com/google/uuid"
)

type Customer struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone,omitempty"`
	Address   string    `json:"address,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CustomerRepository interface {
	Create(ctx context.Context, c *Customer) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Customer, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]Customer, error)
}
"""

sales_order_domain = """package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SalesOrder struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	CustomerID  uuid.UUID       `json:"customer_id"`
	OrderNumber string          `json:"order_number"`
	TotalAmount decimal.Decimal `json:"total_amount"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type SalesOrderRepository interface {
	Create(ctx context.Context, so *SalesOrder) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*SalesOrder, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]SalesOrder, error)
}
"""

sales_invoice_domain = """package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SalesInvoice struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	SalesOrderID uuid.UUID       `json:"sales_order_id"`
	InvoiceNumber string         `json:"invoice_number"`
	Amount       decimal.Decimal `json:"amount"`
	Status       string          `json:"status"`
	DueDate      time.Time       `json:"due_date"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type SalesInvoiceRepository interface {
	Create(ctx context.Context, si *SalesInvoice) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*SalesInvoice, error)
	List(ctx context.Context, tenantID uuid.UUID) ([]SalesInvoice, error)
}
"""

with open(os.path.join(domain_dir, "customer.go"), "w") as f:
    f.write(customer_domain.replace("context.Context", "context.Context\n").replace("import (", "import (\n\t\"context\""))
with open(os.path.join(domain_dir, "sales_order.go"), "w") as f:
    f.write(sales_order_domain.replace("import (", "import (\n\t\"context\""))
with open(os.path.join(domain_dir, "sales_invoice.go"), "w") as f:
    f.write(sales_invoice_domain.replace("import (", "import (\n\t\"context\""))

# 2. Infra
customer_infra = """package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type CustomerRepo struct {
	db *sql.DB
}

func NewCustomerRepo(db *sql.DB) *CustomerRepo {
	return &CustomerRepo{db: db}
}

func (r *CustomerRepo) Create(ctx context.Context, c *domain.Customer) error {
	query := `INSERT INTO customers (id, tenant_id, name, email, phone, address, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query, c.ID, c.TenantID, c.Name, c.Email, c.Phone, c.Address, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *CustomerRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Customer, error) {
	query := `SELECT id, tenant_id, name, email, phone, address, created_at, updated_at
			  FROM customers WHERE tenant_id = $1 AND id = $2`
	row := r.db.QueryRowContext(ctx, query, tenantID, id)

	var c domain.Customer
	err := row.Scan(&c.ID, &c.TenantID, &c.Name, &c.Email, &c.Phone, &c.Address, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *CustomerRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Customer, error) {
	query := `SELECT id, tenant_id, name, email, phone, address, created_at, updated_at
			  FROM customers WHERE tenant_id = $1`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []domain.Customer
	for rows.Next() {
		var c domain.Customer
		if err := rows.Scan(&c.ID, &c.TenantID, &c.Name, &c.Email, &c.Phone, &c.Address, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		customers = append(customers, c)
	}
	return customers, nil
}
"""

sales_order_infra = """package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type SalesOrderRepo struct {
	db *sql.DB
}

func NewSalesOrderRepo(db *sql.DB) *SalesOrderRepo {
	return &SalesOrderRepo{db: db}
}

func (r *SalesOrderRepo) Create(ctx context.Context, so *domain.SalesOrder) error {
	query := `INSERT INTO sales_orders (id, tenant_id, customer_id, order_number, total_amount, status, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query, so.ID, so.TenantID, so.CustomerID, so.OrderNumber, so.TotalAmount, so.Status, so.CreatedAt, so.UpdatedAt)
	return err
}

func (r *SalesOrderRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SalesOrder, error) {
	query := `SELECT id, tenant_id, customer_id, order_number, total_amount, status, created_at, updated_at
			  FROM sales_orders WHERE tenant_id = $1 AND id = $2`
	row := r.db.QueryRowContext(ctx, query, tenantID, id)

	var so domain.SalesOrder
	err := row.Scan(&so.ID, &so.TenantID, &so.CustomerID, &so.OrderNumber, &so.TotalAmount, &so.Status, &so.CreatedAt, &so.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &so, nil
}

func (r *SalesOrderRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SalesOrder, error) {
	query := `SELECT id, tenant_id, customer_id, order_number, total_amount, status, created_at, updated_at
			  FROM sales_orders WHERE tenant_id = $1`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []domain.SalesOrder
	for rows.Next() {
		var so domain.SalesOrder
		if err := rows.Scan(&so.ID, &so.TenantID, &so.CustomerID, &so.OrderNumber, &so.TotalAmount, &so.Status, &so.CreatedAt, &so.UpdatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, so)
	}
	return orders, nil
}
"""

sales_invoice_infra = """package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type SalesInvoiceRepo struct {
	db *sql.DB
}

func NewSalesInvoiceRepo(db *sql.DB) *SalesInvoiceRepo {
	return &SalesInvoiceRepo{db: db}
}

func (r *SalesInvoiceRepo) Create(ctx context.Context, si *domain.SalesInvoice) error {
	query := `INSERT INTO sales_invoices (id, tenant_id, sales_order_id, invoice_number, amount, status, due_date, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.ExecContext(ctx, query, si.ID, si.TenantID, si.SalesOrderID, si.InvoiceNumber, si.Amount, si.Status, si.DueDate, si.CreatedAt, si.UpdatedAt)
	return err
}

func (r *SalesInvoiceRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SalesInvoice, error) {
	query := `SELECT id, tenant_id, sales_order_id, invoice_number, amount, status, due_date, created_at, updated_at
			  FROM sales_invoices WHERE tenant_id = $1 AND id = $2`
	row := r.db.QueryRowContext(ctx, query, tenantID, id)

	var si domain.SalesInvoice
	err := row.Scan(&si.ID, &si.TenantID, &si.SalesOrderID, &si.InvoiceNumber, &si.Amount, &si.Status, &si.DueDate, &si.CreatedAt, &si.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &si, nil
}

func (r *SalesInvoiceRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SalesInvoice, error) {
	query := `SELECT id, tenant_id, sales_order_id, invoice_number, amount, status, due_date, created_at, updated_at
			  FROM sales_invoices WHERE tenant_id = $1`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []domain.SalesInvoice
	for rows.Next() {
		var si domain.SalesInvoice
		if err := rows.Scan(&si.ID, &si.TenantID, &si.SalesOrderID, &si.InvoiceNumber, &si.Amount, &si.Status, &si.DueDate, &si.CreatedAt, &si.UpdatedAt); err != nil {
			return nil, err
		}
		invoices = append(invoices, si)
	}
	return invoices, nil
}
"""

with open(os.path.join(infra_dir, "customer_repo.go"), "w") as f:
    f.write(customer_infra)
with open(os.path.join(infra_dir, "sales_order_repo.go"), "w") as f:
    f.write(sales_order_infra)
with open(os.path.join(infra_dir, "sales_invoice_repo.go"), "w") as f:
    f.write(sales_invoice_infra)

# 3. Usecase
customer_uc = """package customer

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type Usecase struct {
	repo domain.CustomerRepository
}

func New(repo domain.CustomerRepository) *Usecase {
	return &Usecase{repo: repo}
}

type CreateRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
}

func (u *Usecase) Create(ctx context.Context, tenantID uuid.UUID, req CreateRequest) (*domain.Customer, error) {
	c := &domain.Customer{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		Address:   req.Address,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := u.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *Usecase) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Customer, error) {
	return u.repo.GetByID(ctx, tenantID, id)
}

func (u *Usecase) List(ctx context.Context, tenantID uuid.UUID) ([]domain.Customer, error) {
	return u.repo.List(ctx, tenantID)
}
"""

sales_order_uc = """package sales_order

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type Usecase struct {
	repo domain.SalesOrderRepository
}

func New(repo domain.SalesOrderRepository) *Usecase {
	return &Usecase{repo: repo}
}

type CreateRequest struct {
	CustomerID  uuid.UUID       `json:"customer_id"`
	OrderNumber string          `json:"order_number"`
	TotalAmount decimal.Decimal `json:"total_amount"`
}

func (u *Usecase) Create(ctx context.Context, tenantID uuid.UUID, req CreateRequest) (*domain.SalesOrder, error) {
	so := &domain.SalesOrder{
		ID:          uuid.New(),
		TenantID:    tenantID,
		CustomerID:  req.CustomerID,
		OrderNumber: req.OrderNumber,
		TotalAmount: req.TotalAmount,
		Status:      "PENDING",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := u.repo.Create(ctx, so); err != nil {
		return nil, err
	}
	return so, nil
}

func (u *Usecase) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SalesOrder, error) {
	return u.repo.GetByID(ctx, tenantID, id)
}

func (u *Usecase) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SalesOrder, error) {
	return u.repo.List(ctx, tenantID)
}
"""

sales_invoice_uc = """package sales_invoice

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type Usecase struct {
	repo domain.SalesInvoiceRepository
}

func New(repo domain.SalesInvoiceRepository) *Usecase {
	return &Usecase{repo: repo}
}

type CreateRequest struct {
	SalesOrderID  uuid.UUID       `json:"sales_order_id"`
	InvoiceNumber string          `json:"invoice_number"`
	Amount        decimal.Decimal `json:"amount"`
	DueDate       time.Time       `json:"due_date"`
}

func (u *Usecase) Create(ctx context.Context, tenantID uuid.UUID, req CreateRequest) (*domain.SalesInvoice, error) {
	si := &domain.SalesInvoice{
		ID:           uuid.New(),
		TenantID:     tenantID,
		SalesOrderID: req.SalesOrderID,
		InvoiceNumber: req.InvoiceNumber,
		Amount:       req.Amount,
		Status:       "UNPAID",
		DueDate:      req.DueDate,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := u.repo.Create(ctx, si); err != nil {
		return nil, err
	}
	return si, nil
}

func (u *Usecase) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SalesInvoice, error) {
	return u.repo.GetByID(ctx, tenantID, id)
}

func (u *Usecase) List(ctx context.Context, tenantID uuid.UUID) ([]domain.SalesInvoice, error) {
	return u.repo.List(ctx, tenantID)
}
"""

with open(os.path.join(usecase_dir, "customer", "customer.go"), "w") as f:
    f.write(customer_uc)
with open(os.path.join(usecase_dir, "sales_order", "sales_order.go"), "w") as f:
    f.write(sales_order_uc)
with open(os.path.join(usecase_dir, "sales_invoice", "sales_invoice.go"), "w") as f:
    f.write(sales_invoice_uc)

# 4. Handler
customer_h = """package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/customer"
)

type CustomerHandler struct {
	uc *uc.Usecase
}

func NewCustomerHandler(u *uc.Usecase) *CustomerHandler {
	return &CustomerHandler{uc: u}
}

func (h *CustomerHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := middleware.GetTenantID(r.Context())
	var req uc.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := h.uc.Create(r.Context(), tenantID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(res)
}

func (h *CustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := middleware.GetTenantID(r.Context())
	res, err := h.uc.List(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(res)
}

func (h *CustomerHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := middleware.GetTenantID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	res, err := h.uc.GetByID(r.Context(), tenantID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(res)
}
"""

sales_order_h = """package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/sales_order"
)

type SalesOrderHandler struct {
	uc *uc.Usecase
}

func NewSalesOrderHandler(u *uc.Usecase) *SalesOrderHandler {
	return &SalesOrderHandler{uc: u}
}

func (h *SalesOrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := middleware.GetTenantID(r.Context())
	var req uc.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := h.uc.Create(r.Context(), tenantID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(res)
}

func (h *SalesOrderHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := middleware.GetTenantID(r.Context())
	res, err := h.uc.List(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(res)
}

func (h *SalesOrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := middleware.GetTenantID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	res, err := h.uc.GetByID(r.Context(), tenantID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(res)
}
"""

sales_invoice_h = """package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	uc "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/sales_invoice"
)

type SalesInvoiceHandler struct {
	uc *uc.Usecase
}

func NewSalesInvoiceHandler(u *uc.Usecase) *SalesInvoiceHandler {
	return &SalesInvoiceHandler{uc: u}
}

func (h *SalesInvoiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := middleware.GetTenantID(r.Context())
	var req uc.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := h.uc.Create(r.Context(), tenantID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(res)
}

func (h *SalesInvoiceHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := middleware.GetTenantID(r.Context())
	res, err := h.uc.List(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(res)
}

func (h *SalesInvoiceHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := middleware.GetTenantID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	res, err := h.uc.GetByID(r.Context(), tenantID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(res)
}
"""

with open(os.path.join(handler_dir, "customer_handler.go"), "w") as f:
    f.write(customer_h)
with open(os.path.join(handler_dir, "sales_order_handler.go"), "w") as f:
    f.write(sales_order_h)
with open(os.path.join(handler_dir, "sales_invoice_handler.go"), "w") as f:
    f.write(sales_invoice_h)

print("Scaffolding complete.")
