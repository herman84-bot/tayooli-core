package invoice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// HandleOCRResult persists the OCR result received from the Python AI worker via
// the invoice.ocr_completed Kafka topic.
//
// tenantID and invoiceID are passed explicitly — the consumer already parsed them
// from the Kafka message payload, so we do not re-extract them from context here.
//
// After the OCR result is saved, anomaly detection is triggered asynchronously
// via a goroutine.  AI failure is best-effort and never blocks the OCR flow.
func (f *Facade) HandleOCRResult(ctx context.Context, invoiceID, tenantID uuid.UUID, score float64, extractedText, status string) error {
	if err := f.repo.UpdateOCRResult(ctx, tenantID, invoiceID, score, extractedText, status); err != nil {
		return fmt.Errorf("invoiceFacade.HandleOCRResult: %w", err)
	}

	// Best-effort async AI trigger — failures are logged, never propagated.
	if f.aiTrigger != nil {
		go func() {
			aiCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			// Fetch the invoice to get the amount for the AI payload.
			inv, err := f.repo.GetByID(aiCtx, invoiceID, tenantID)
			if err != nil {
				log.Error().Err(err).
					Str("invoice_id", invoiceID.String()).
					Msg("ai trigger: failed to fetch invoice for anomaly detection")
				return
			}
			if _, infErr := f.aiTrigger.TriggerInference(
				aiCtx,
				invoiceID.String(),
				tenantID.String(),
				inv.Amount.String(),
				inv.VendorID,
				extractedText,
			); infErr != nil {
				log.Error().Err(infErr).
					Str("invoice_id", invoiceID.String()).
					Msg("ai trigger: anomaly detection failed (best-effort)")
			}
		}()
	}

	return nil
}
