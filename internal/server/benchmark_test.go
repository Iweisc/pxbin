package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sertdev/pxbin/internal/config"
	"github.com/sertdev/pxbin/internal/metrics"
	"github.com/sertdev/pxbin/internal/ratelimit"
)

type benchProxyHandler struct{}

func (b *benchProxyHandler) HandleAnthropic(w http.ResponseWriter, r *http.Request)       { w.WriteHeader(200) }
func (b *benchProxyHandler) HandleOpenAI(w http.ResponseWriter, r *http.Request)           { w.WriteHeader(200) }
func (b *benchProxyHandler) HandleOpenAIResponses(w http.ResponseWriter, r *http.Request)  { w.WriteHeader(200) }

func BenchmarkSecurityHeadersMiddleware(b *testing.B) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkRateLimiterAllow(b *testing.B) {
	l := ratelimit.NewLimiter(1_000_000, 1_000_000) // very high limit to not deny
	defer l.Close()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		l.Allow("bench-key")
	}
}

func BenchmarkMetricsMiddleware(b *testing.B) {
	m := metrics.New()
	handler := metrics.Middleware(m)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("POST", "/v1/messages", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkFullMiddlewareChain(b *testing.B) {
	cfg := &config.Config{CORSOrigins: []string{"*"}}
	m := metrics.New()
	limiter := ratelimit.NewLimiter(1_000_000, 1_000_000)
	defer limiter.Close()

	opts := &Opts{
		RateLimiter:       limiter,
		MetricsMiddleware: metrics.Middleware(m),
	}

	router := New(cfg, &benchProxyHandler{}, func(next http.Handler) http.Handler { return next }, chi.NewRouter(), nil, nil, opts)
	req := httptest.NewRequest("POST", "/v1/messages", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
	}
}
