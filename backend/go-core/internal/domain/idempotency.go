package domain

import (
	"context"

	"github.com/google/uuid"
)

// IdempotencyRecord is one client-supplied Idempotency-Key for a write
// endpoint. While Completed is false the original request is still running
// (or crashed before finishing); once true, StatusCode/ResponseBody hold the
// response to replay for retries.
type IdempotencyRecord struct {
	Key          string
	RequestHash  string
	Completed    bool
	StatusCode   int
	ResponseBody []byte
}

// IdempotencyStore guards a write so a retried request (lost response, double
// tap, reconnect after power loss) runs at most once per (tenant, key).
type IdempotencyStore interface {
	// Reserve atomically claims the key. fresh=true means the caller owns it
	// and must later Complete or Release. fresh=false returns the existing
	// record unchanged.
	Reserve(ctx context.Context, tenantID uuid.UUID, key, requestHash string) (existing *IdempotencyRecord, fresh bool, err error)
	Complete(ctx context.Context, tenantID uuid.UUID, key string, statusCode int, body []byte) error
	Release(ctx context.Context, tenantID uuid.UUID, key string) error
}
