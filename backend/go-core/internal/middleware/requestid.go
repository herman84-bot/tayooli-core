package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

const RequestIDKey contextKey = "request_id"

// RequestID middleware reads X-Request-ID from the incoming request header.
// If the header is absent, a new UUID is generated. The ID is stored in the
// request context and echoed back in the response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), RequestIDKey, reqID)
		w.Header().Set("X-Request-ID", reqID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID extracts the request ID from context. Returns empty string if absent.
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(RequestIDKey).(string); ok {
		return v
	}
	return ""
}
