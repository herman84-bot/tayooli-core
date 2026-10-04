package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/accounting/domain"
	"github.com/shopspring/decimal"
)

type JournalEntryUseCase struct {
	repo        domain.JournalEntryRepository
	accountRepo domain.AccountRepository
}

func NewJournalEntryUseCase(repo domain.JournalEntryRepository, accountRepo domain.AccountRepository) *JournalEntryUseCase {
	return &JournalEntryUseCase{repo: repo, accountRepo: accountRepo}
}

func (u *JournalEntryUseCase) CreateJournalEntry(ctx context.Context, entry *domain.JournalEntry) error {
	// Server-side balance validation using decimal for precision
	totalDebit := decimal.NewFromFloat(entry.DebitAmount)
	totalCredit := decimal.NewFromFloat(entry.CreditAmount)
	if !totalDebit.Equal(totalCredit) {
		return domain.ErrJournalNotBalanced
	}

	return u.repo.Create(ctx, entry)
}

func (u *JournalEntryUseCase) ListJournalEntries(ctx context.Context, tenantID uuid.UUID) ([]*domain.JournalEntry, error) {
	return u.repo.List(ctx, tenantID)
}

func (u *JournalEntryUseCase) RecordPurchasePayment(ctx context.Context, tenantID uuid.UUID, paymentID, referenceNumber string, amount float64, transactionDate time.Time) error {
	// 2.1 Use Case 1: Purchase Invoice Payment
	// Debit: Accounts Payable (Code: "2001")
	// Credit: Cash / Bank (Code: "1001")
	apAcc, err := u.accountRepo.GetByCode(ctx, "2001", tenantID)
	if err != nil {
		return err
	}
	cashAcc, err := u.accountRepo.GetByCode(ctx, "1001", tenantID)
	if err != nil {
		return err
	}

	debitEntry := &domain.JournalEntry{
		TenantID:        tenantID,
		AccountID:       apAcc.ID,
		ReferenceID:     paymentID,
		TransactionDate: transactionDate,
		Description:     "Purchase Payment - " + referenceNumber,
		DebitAmount:     amount,
		CreditAmount:    0,
	}

	creditEntry := &domain.JournalEntry{
		TenantID:        tenantID,
		AccountID:       cashAcc.ID,
		ReferenceID:     paymentID,
		TransactionDate: transactionDate,
		Description:     "Purchase Payment - " + referenceNumber,
		DebitAmount:     0,
		CreditAmount:    amount,
	}

	return u.repo.RecordJournalEntries(ctx, tenantID, []*domain.JournalEntry{debitEntry, creditEntry})
}

func (u *JournalEntryUseCase) RecordSalesRevenue(ctx context.Context, tenantID uuid.UUID, invoiceID, invoiceNumber string, amount float64, transactionDate time.Time) error {
	// 2.2 Use Case 2: Sales Invoice Creation
	// Debit: Accounts Receivable (Code: "1101")
	// Credit: Sales Revenue (Code: "4001")
	arAcc, err := u.accountRepo.GetByCode(ctx, "1101", tenantID)
	if err != nil {
		return err
	}
	revAcc, err := u.accountRepo.GetByCode(ctx, "4001", tenantID)
	if err != nil {
		return err
	}

	debitEntry := &domain.JournalEntry{
		TenantID:        tenantID,
		AccountID:       arAcc.ID,
		ReferenceID:     invoiceID,
		TransactionDate: transactionDate,
		Description:     "Sales Invoice - " + invoiceNumber,
		DebitAmount:     amount,
		CreditAmount:    0,
	}

	creditEntry := &domain.JournalEntry{
		TenantID:        tenantID,
		AccountID:       revAcc.ID,
		ReferenceID:     invoiceID,
		TransactionDate: transactionDate,
		Description:     "Sales Invoice - " + invoiceNumber,
		DebitAmount:     0,
		CreditAmount:    amount,
	}

	return u.repo.RecordJournalEntries(ctx, tenantID, []*domain.JournalEntry{debitEntry, creditEntry})
}
