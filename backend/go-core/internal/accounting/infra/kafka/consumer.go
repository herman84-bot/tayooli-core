package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/accounting/usecase"
	globalDomain "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

type AccountingConsumer struct {
	reader  *kafka.Reader
	usecase *usecase.JournalEntryUseCase
}

func NewAccountingConsumer(brokers []string, uc *usecase.JournalEntryUseCase) *AccountingConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		GroupID:        "accounting-integration-group",
		GroupTopics:    []string{"payment_order.paid", "sales_invoice.created"},
		MinBytes:       10e3,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})

	return &AccountingConsumer{
		reader:  reader,
		usecase: uc,
	}
}

func (c *AccountingConsumer) Run(ctx context.Context) {
	log.Info().Msg("AccountingConsumer starting")
	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Info().Msg("AccountingConsumer context cancelled, exiting loop")
				return
			}
			log.Error().Err(err).Msg("AccountingConsumer: failed to fetch message")
			continue
		}

		err = c.processMessage(ctx, m)
		if err != nil {
			log.Error().Err(err).Str("topic", m.Topic).Msg("AccountingConsumer: failed to process message")
			// Depending on error, we might not commit to allow retry.
			// If it's a poison pill (unparseable json, etc.), we should probably commit to avoid blocking.
			// Let's assume schema mismatch is poison pill, DB constraint is retryable.
			// For simplicity, we just won't commit if there's an error so it will be retried.
			// However, in production, we should differentiate.
			if strings.Contains(err.Error(), "poison") {
				log.Error().Msg("Poison pill detected, committing offset to skip")
				c.reader.CommitMessages(ctx, m)
			}
			continue
		}

		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Error().Err(err).Msg("AccountingConsumer: failed to commit message")
		}
	}
}

func (c *AccountingConsumer) processMessage(ctx context.Context, m kafka.Message) error {
	switch m.Topic {
	case "payment_order.paid":
		var event globalDomain.PaymentOrderPaidEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			return fmt.Errorf("poison: failed to unmarshal payment_order.paid: %v", err)
		}

		tenantID, err := uuid.Parse(event.TenantID)
		if err != nil {
			return fmt.Errorf("poison: invalid tenant_id: %v", err)
		}

		amount, err := strconv.ParseFloat(event.Amount, 64)
		if err != nil {
			return fmt.Errorf("poison: invalid amount: %v", err)
		}

		var txnDate time.Time
		if event.PaidAt != "" {
			txnDate, err = time.Parse(time.RFC3339, event.PaidAt)
			if err != nil {
				txnDate = time.Now()
			}
		} else {
			txnDate = time.Now()
		}

		return c.usecase.RecordPurchasePayment(ctx, tenantID, event.PaymentOrderID, event.PaymentOrderID, amount, txnDate)

	case "sales_invoice.created":
		var event globalDomain.SalesInvoiceCreatedEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			return fmt.Errorf("poison: failed to unmarshal sales_invoice.created: %v", err)
		}

		tenantID, err := uuid.Parse(event.TenantID)
		if err != nil {
			return fmt.Errorf("poison: invalid tenant_id: %v", err)
		}

		amount, err := strconv.ParseFloat(event.Amount, 64)
		if err != nil {
			return fmt.Errorf("poison: invalid amount: %v", err)
		}

		txnDate := time.Now()
		if event.Timestamp != "" {
			parsed, err := time.Parse(time.RFC3339, event.Timestamp)
			if err == nil {
				txnDate = parsed
			}
		}

		return c.usecase.RecordSalesRevenue(ctx, tenantID, event.SalesInvoiceID, event.InvoiceNumber, amount, txnDate)

	default:
		return fmt.Errorf("poison: unknown topic %s", m.Topic)
	}
}

func (c *AccountingConsumer) Close() error {
	return c.reader.Close()
}
