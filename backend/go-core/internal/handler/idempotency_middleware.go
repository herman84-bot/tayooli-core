package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	appMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
)

// maxIdempotentBody bounds the request body read into memory for hashing
// and replay caching (1 MB).
const maxIdempotentBody = 1 << 20

// requestFingerprint binds the idempotency reservation to the specific
// HTTP method, path, and request body so a key cannot be reused across
// different resources or payloads.
func requestFingerprint(method, path string, body []byte) string {
	h := sha256.New()
	h.Write([]byte(method))
	h.Write([]byte(" "))
	h.Write([]byte(path))
	h.Write([]byte("\n"))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

// Idempotent wraps a write handler with Idempotency-Key support backed by
// domain.IdempotencyStore (Postgres table idempotency_keys).
//
// Semantics:
//   - Without an Idempotency-Key header: passes through untouched.
//   - Key in-flight: returns 409 Conflict with Retry-After: 2.
//   - Key already completed: replays status and body with Idempotent-Replayed: true.
//   - Key reused for a different endpoint or payload: returns 422 Unprocessable Entity.
//   - Handler failed (non-2xx): releases the key immediately so the client can retry.
func Idempotent(store domain.IdempotencyStore) func(http.Handler) http.Handler {
	if store == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}
			if len(key) > maxIdempotencyKeyLen {
				http.Error(w, "Idempotency-Key terlalu panjang", http.StatusBadRequest)
				return
			}
			tenantID, ok := appMiddleware.GetTenantID(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			lr := io.LimitReader(r.Body, int64(maxIdempotentBody)+1)
			body, err := io.ReadAll(lr)
			if err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			if len(body) > maxIdempotentBody {
				http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
				return
			}

			hash := requestFingerprint(r.Method, r.URL.Path, body)
			existing, fresh, err := store.Reserve(r.Context(), tenantID, key, hash)
			if err != nil {
				if errors.Is(err, domain.ErrConflict) {
					w.Header().Set("Retry-After", "2")
					http.Error(w, "permintaan dengan kunci ini sedang diproses, coba lagi", http.StatusConflict)
					return
				}
				respondInternalError(w, r, "idempotency reserve", err)
				return
			}

			if !fresh {
				switch {
				case existing.RequestHash != hash:
					http.Error(w, "Idempotency-Key sudah dipakai untuk permintaan lain", http.StatusUnprocessableEntity)
				case !existing.Completed:
					w.Header().Set("Retry-After", "2")
					http.Error(w, "permintaan dengan kunci ini sedang diproses, coba lagi", http.StatusConflict)
				default:
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("Idempotent-Replayed", "true")
					w.WriteHeader(existing.StatusCode)
					_, _ = w.Write(existing.ResponseBody)
				}
				return
			}

			// Caller owns the key. Run downstream into a recorder so the response
			// can be saved atomically before or during client transmission.
			r.Body = io.NopCloser(bytes.NewReader(body))
			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, r)

			bg := context.WithoutCancel(r.Context())
			if rec.Code >= 200 && rec.Code < 300 {
				if err := store.Complete(bg, tenantID, key, rec.Code, rec.Body.Bytes()); err != nil {
					slog.Error("idempotency complete failed", "err", err, "tenant_id", tenantID, "key", key)
				}
			} else {
				if err := store.Release(bg, tenantID, key); err != nil {
					slog.Error("idempotency release failed", "err", err, "tenant_id", tenantID, "key", key)
				}
			}

			for k, v := range rec.Header() {
				w.Header()[k] = v
			}
			w.WriteHeader(rec.Code)
			_, _ = w.Write(rec.Body.Bytes())
		})
	}
}
