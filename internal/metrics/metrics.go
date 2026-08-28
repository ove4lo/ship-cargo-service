package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// BookingsTotal tracks the cumulative number of processed cargo bookings partitioned by status.
	BookingsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bookings_total",
			Help: "Total number of bookings by status",
		},
		[]string{"status"},
	)

	// BookingDuration measures the latency distribution of internal booking domain operations.
	BookingDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "booking_duration_seconds",
			Help:    "Time spent processing a booking request",
			Buckets: prometheus.DefBuckets,
		},
	)

	// LockConflicts tracks the absolute frequency of requests rejected due to distributed lock contention.
	LockConflicts = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "lock_conflicts_total",
			Help: "Number of times a booking was rejected due lock contention",
		},
	)

	// HTTPRequests counts inbound HTTP requests fully resolved by the API layer, split by method, path, and code.
	HTTPRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests by method, path and status code",
		},
		[]string{"method", "path", "code"},
	)

	// HTTPDuration metrics the overall lifecycle duration of HTTP responses handled by the API.
	HTTPDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration by method and path",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

// Register registers all defined system metrics into the default Prometheus registry.
func Register() {
	prometheus.MustRegister(
		BookingsTotal,
		BookingDuration,
		LockConflicts,
		HTTPRequests,
		HTTPDuration,
	)
}