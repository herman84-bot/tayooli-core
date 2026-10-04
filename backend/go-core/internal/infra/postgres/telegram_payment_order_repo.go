package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

type TelegramPaymentOrderRepo struct {
	db *sql.DB
}

func NewTelegramPaymentOrderRepo(db *sql.DB) *TelegramPaymentOrderRepo {
	return &TelegramPaymentOrderRepo{db: db}
}

func (r *TelegramPaymentOrderRepo) CreateOrder(ctx context.Context, orderCode, tenantID, userEmail, plan string, amount int, period string) error {
	const q = `INSERT INTO telegram_payment_orders (order_code, tenant_id, user_email, plan, amount, period, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending') ON CONFLICT (order_code) DO NOTHING`
	_, err := r.db.ExecContext(ctx, q, orderCode, tenantID, userEmail, plan, amount, period)
	return err
}

func (r *TelegramPaymentOrderRepo) GetOrderByCode(ctx context.Context, orderCode string) (*domain.TelegramPaymentOrder, error) {
	const q = `SELECT id, order_code, tenant_id::text, user_email, plan, amount, period, status FROM telegram_payment_orders WHERE order_code = $1`
	var id uuid.UUID
	var code, tenantStr, email, planName, periodVal, status string
	var amount int
	err := r.db.QueryRowContext(ctx, q, orderCode).Scan(&id, &code, &tenantStr, &email, &planName, &amount, &periodVal, &status)
	if err != nil {
		return nil, err
	}
	tenantUUID, _ := uuid.Parse(tenantStr)
	return &domain.TelegramPaymentOrder{
		ID:        id,
		OrderCode: code,
		TenantID:  tenantUUID,
		UserEmail: email,
		Plan:      planName,
		Amount:    amount,
		Period:    periodVal,
		Status:    status,
	}, nil
}

func (r *TelegramPaymentOrderRepo) GetPendingOrderByCode(ctx context.Context, orderCode string) (*domain.TelegramPaymentOrder, error) {
	const q = `SELECT id, order_code, tenant_id::text, user_email, plan, amount, period, status FROM telegram_payment_orders WHERE order_code = $1 AND status = 'pending'`
	var id uuid.UUID
	var code, tenantStr, email, planName, periodVal, status string
	var amount int
	err := r.db.QueryRowContext(ctx, q, orderCode).Scan(&id, &code, &tenantStr, &email, &planName, &amount, &periodVal, &status)
	if err != nil {
		return nil, err
	}
	tenantUUID, _ := uuid.Parse(tenantStr)
	return &domain.TelegramPaymentOrder{
		ID:        id,
		OrderCode: code,
		TenantID:  tenantUUID,
		UserEmail: email,
		Plan:      planName,
		Amount:    amount,
		Period:    periodVal,
		Status:    status,
	}, nil
}

func (r *TelegramPaymentOrderRepo) UpdateStatus(ctx context.Context, orderCode, status string) error {
	const q = `UPDATE telegram_payment_orders SET status = $1, updated_at = NOW() WHERE order_code = $2`
	_, err := r.db.ExecContext(ctx, q, status, orderCode)
	return err
}
