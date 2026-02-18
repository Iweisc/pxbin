package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metric collectors for pxbin.
type Metrics struct {
	Registry            *prometheus.Registry
	RequestsTotal       *prometheus.CounterVec
	RequestDuration     *prometheus.HistogramVec
	OverheadUS          prometheus.Histogram
	ActiveStreams        prometheus.Gauge
	DroppedLogsTotal    prometheus.Counter
	CircuitBreakerState *prometheus.GaugeVec
	RateLimitedTotal    prometheus.Counter
}

// New creates and registers a new Metrics instance using a dedicated registry.
func New() *Metrics {
	reg := prometheus.NewRegistry()

	m := &Metrics{
		Registry: reg,

		RequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_requests_total",
			Help: "Total number of proxied requests.",
		}, []string{"method", "path", "status_code", "format"}),

		RequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "proxy_request_duration_seconds",
			Help:    "Duration of proxied requests in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),

		OverheadUS: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "proxy_overhead_microseconds",
			Help:    "Proxy processing overhead in microseconds.",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		}),

		ActiveStreams: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "proxy_active_streams",
			Help: "Number of currently active streaming connections.",
		}),

		DroppedLogsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "proxy_dropped_logs_total",
			Help: "Total number of dropped log entries due to full buffer.",
		}),

		CircuitBreakerState: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "proxy_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open).",
		}, []string{"upstream"}),

		RateLimitedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "proxy_rate_limited_total",
			Help: "Total number of rate-limited requests.",
		}),
	}

	reg.MustRegister(
		m.RequestsTotal,
		m.RequestDuration,
		m.OverheadUS,
		m.ActiveStreams,
		m.DroppedLogsTotal,
		m.CircuitBreakerState,
		m.RateLimitedTotal,
	)

	return m
}

// Handler returns the Prometheus HTTP handler for the /metrics endpoint
// using the metrics instance's dedicated registry.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}
