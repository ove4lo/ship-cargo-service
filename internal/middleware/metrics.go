package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ove4lo/ship-cargo-service/internal/metrics"
)

// statusWriter wraps a standard http.ResponseWriter to capture the returned HTTP status code.
type statusWriter struct {
	http.ResponseWriter
	code int
}

// WriteHeader stores the status code and forwards it to the underlying ResponseWriter.
func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

// Metrics intercepts HTTP requests to record execution durations and total request counts 
// segmented by method, path, and status code for Prometheus monitoring.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}

		next.ServeHTTP(sw, r)

		metrics.HTTPRequests.WithLabelValues(r.Method, r.Pattern, strconv.Itoa(sw.code)).Inc()
		metrics.HTTPDuration.WithLabelValues(r.Method, r.Pattern).Observe(time.Since(start).Seconds())
	})
}