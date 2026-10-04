package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrJournalNotBalanced is returned when total debits do not equal total credits.
var ErrJournalNotBalanced = errors.New("journal entry is not balanced: total debits must equal total credits")

type JournalEntry struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	AccountID      uuid.UUID `json:"account_id"`
	ReferenceID    string    `json:"reference_id"`
	TransactionDate time.Time `json:"transaction_date"`
	Description    string    `json:"description"`
	DebitAmount    float64   `json:"debit_amount"`
	CreditAmount   float64   `json:"credit_amount"`
	CreatedAt      time.Time `json:"created_at"`
}

type JournalEntryRepository interface {
	Create(ctx context.Context, entry *JournalEntry) error
	RecordJournalEntries(ctx context.Context, tenantID uuid.UUID, entries []*JournalEntry) error
	List(ctx context.Context, tenantID uuid.UUID) ([]*JournalEntry, error)
}
