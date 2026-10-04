package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/accounting/domain"
)

type journalEntryRepository struct {
	db *sql.DB
}

func NewJournalEntryRepository(db *sql.DB) domain.JournalEntryRepository {
	return &journalEntryRepository{db: db}
}

func (r *journalEntryRepository) Create(ctx context.Context, entry *domain.JournalEntry) error {
	query := `
		INSERT INTO journal_entries (id, tenant_id, account_id, reference_id, transaction_date, description, debit_amount, credit_amount)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	_, err := r.db.ExecContext(ctx, query, entry.ID, entry.TenantID, entry.AccountID, entry.ReferenceID, entry.TransactionDate, entry.Description, entry.DebitAmount, entry.CreditAmount)
	return err
}

func (r *journalEntryRepository) RecordJournalEntries(ctx context.Context, tenantID uuid.UUID, entries []*domain.JournalEntry) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Set RLS
	_, err = tx.ExecContext(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID.String())
	if err != nil {
		return err
	}

	insertQuery := `
		INSERT INTO journal_entries (id, tenant_id, account_id, reference_id, transaction_date, description, debit_amount, credit_amount)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_id, reference_id, account_id) DO NOTHING
	`
	updateQuery := `
		UPDATE accounts
		SET balance = balance + 
			CASE 
				WHEN type IN ('Asset', 'Expense') THEN $1 - $2
				ELSE $2 - $1
			END,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND tenant_id = $4
	`

	for _, entry := range entries {
		if entry.ID == uuid.Nil {
			entry.ID = uuid.New()
		}
		res, err := tx.ExecContext(ctx, insertQuery, entry.ID, entry.TenantID, entry.AccountID, entry.ReferenceID, entry.TransactionDate, entry.Description, entry.DebitAmount, entry.CreditAmount)
		if err != nil {
			return err
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		
		// Idempotency: If no rows were inserted (due to duplicate reference_id), skip balance update.
		if rowsAffected == 0 {
			continue
		}

		_, err = tx.ExecContext(ctx, updateQuery, entry.DebitAmount, entry.CreditAmount, entry.AccountID, entry.TenantID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *journalEntryRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.JournalEntry, error) {
	query := `SELECT id, tenant_id, account_id, reference_id, transaction_date, description, debit_amount, credit_amount, created_at FROM journal_entries WHERE tenant_id = $1`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*domain.JournalEntry
	for rows.Next() {
		var e domain.JournalEntry
		if err := rows.Scan(&e.ID, &e.TenantID, &e.AccountID, &e.ReferenceID, &e.TransactionDate, &e.Description, &e.DebitAmount, &e.CreditAmount, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, &e)
	}
	return entries, nil
}
