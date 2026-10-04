package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being served",
		},
	)
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// MetricsMiddleware instruments HTTP handlers with Prometheus counters and histograms.
// It uses the chi route pattern (not the raw path) to avoid high-cardinality labels.
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()

		// Use the chi route pattern for stable label values (/api/v1/invoices/{id})
		routePath := r.URL.Path
		if chiCtx := chi.RouteContext(r.Context()); chiCtx != nil && chiCtx.RoutePattern() != "" {
			routePath = chiCtx.RoutePattern()
		}

		httpRequestsTotal.WithLabelValues(r.Method, routePath, strconv.Itoa(rw.statusCode)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, routePath).Observe(duration)
	})
}

// MetricsHandler returns an http.Handler that serves Prometheus metrics at /metrics.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
