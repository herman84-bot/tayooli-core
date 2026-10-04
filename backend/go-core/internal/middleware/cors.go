package middleware

import (
	"net/http"
	"os"
)

// CORSEnv returns CORS middleware that reads the allowed origin from the
// FRONTEND_ORIGIN environment variable once at creation time and caches it.
// Falls back to http://localhost:3000 when the variable is unset or empty.
//
// Unlike the existing CORS() in tenant.go (which accepts explicit origins),
// this is a zero-configuration wrapper for the common single-origin case.
func CORSEnv() func(http.Handler) http.Handler {
	allowedOrigin := os.Getenv("FRONTEND_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:3000"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
