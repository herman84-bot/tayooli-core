package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/rs/zerolog/log"
	kafkago "github.com/segmentio/kafka-go"
)

// OCRResult is the payload published by the Python AI worker to
// the invoice.ocr_completed topic after running Tesseract OCR.
type OCRResult struct {
	InvoiceID       string  `json:"invoice_id"`
	TenantID        string  `json:"tenant_id"`
	ConfidenceScore float64 `json:"confidence_score"`
	ExtractedText   string  `json:"extracted_text"`
	Status          string  `json:"status"`
}

// OCRResultHandler is a function that processes a single OCR result.
type OCRResultHandler func(ctx context.Context, result OCRResult) error

// Consumer reads OCR result messages from Kafka and delegates them to a handler.
type Consumer struct {
	reader  *kafkago.Reader
	handler OCRResultHandler
}

// NewOCRConsumer creates a Consumer subscribed to the invoice.ocr_completed topic.
// brokers must be a non-empty slice of host:port strings.
func NewOCRConsumer(brokers []string, handler OCRResultHandler) *Consumer {
	r := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  brokers,
		Topic:    "invoice.ocr_completed",
		GroupID:  "go-core-ocr-handler",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	return &Consumer{reader: r, handler: handler}
}

// Run blocks, reading messages from Kafka and calling the handler for each one.
// It returns when ctx is cancelled (graceful shutdown) or a non-recoverable
// error occurs.
//
// Offset commit strategy: FetchMessage does NOT auto-commit.
//   - Handler success → commit.
//   - Handler returns domain.ErrInvalidInput (unrecoverable/poison-pill) → commit
//     and skip so the partition is never stalled by bad data.
//   - Handler returns any other error (transient: DB down, network) → do NOT
//     commit; Kafka redelivers. A consecutive-error counter drives an
//     exponential-ish backoff (capped at 10 s) to avoid a tight retry storm.
func (c *Consumer) Run(ctx context.Context) {
	consecutiveErrors := 0

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				// Context cancelled — normal shutdown path.
				return
			}
			log.Error().Err(err).Msg("kafka consumer: fetch error")
			continue
		}

		var result OCRResult
		if err := json.Unmarshal(msg.Value, &result); err != nil {
			log.Error().Err(err).
				Str("topic", msg.Topic).
				Int("partition", msg.Partition).
				Int64("offset", msg.Offset).
				Msg("kafka consumer: failed to unmarshal OCR result — committing to skip poison pill")
			// Commit malformed messages so they are not redelivered endlessly.
			if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
				log.Error().Err(commitErr).Msg("kafka consumer: failed to commit malformed message offset")
			}
			continue
		}

		if err := callHandler(c.handler, ctx, result); err != nil {
			log.Error().Err(err).
				Str("invoice_id", result.InvoiceID).
				Msg("kafka consumer: OCR result handler failed")

			// H-1: unrecoverable (bad data / validation failure) — commit to
			// skip the poison pill rather than stalling the partition.
			if errors.Is(err, domain.ErrInvalidInput) {
				log.Warn().
					Str("invoice_id", result.InvoiceID).
					Msg("kafka consumer: unrecoverable validation error — committing to skip poison pill")
				if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
					log.Error().Err(commitErr).Msg("kafka consumer: failed to commit poison-pill offset")
				}
				continue
			}

			// M-4: transient error (DB/network) — do NOT commit; Kafka redelivers.
			// Apply backoff to avoid hammering a recovering downstream service.
			consecutiveErrors++
			backoff := time.Duration(min(consecutiveErrors, 10)) * time.Second
			log.Warn().
				Dur("backoff", backoff).
				Int("consecutive_errors", consecutiveErrors).
				Msg("kafka consumer: transient error, backing off before next fetch")
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			continue
		}

		consecutiveErrors = 0 // reset on success

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Error().Err(err).
				Str("invoice_id", result.InvoiceID).
				Msg("kafka consumer: failed to commit offset after successful handler")
		}
	}
}

// Close releases the underlying Kafka reader.
func (c *Consumer) Close() error {
	return c.reader.Close()
}

// callHandler invokes fn inside a recover so that a panic in the handler
// (e.g. nil-pointer in match engine) is converted to an error and the
// consumer goroutine stays alive to process subsequent messages.
func callHandler(fn OCRResultHandler, ctx context.Context, result OCRResult) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("handler panic recovered: %v", r)
		}
	}()
	return fn(ctx, result)
}
