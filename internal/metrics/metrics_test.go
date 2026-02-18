package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestMiddlewareRecordsMetrics(t *testing.T) {
	m := New()

	handler := Middleware(m)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("GET", "/v1/messages", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Check counter was incremented.
	var metric dto.Metric
	counter := m.RequestsTotal.WithLabelValues("GET", "/v1/messages", "200", "")
	counter.(prometheus.Metric).Write(&metric)

	if metric.GetCounter().GetValue() != 1 {
		t.Fatalf("expected counter=1, got %v", metric.GetCounter().GetValue())
	}
}

func TestMiddlewareRecords500(t *testing.T) {
	m := New()

	handler := Middleware(m)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var metric dto.Metric
	counter := m.RequestsTotal.WithLabelValues("POST", "/v1/chat/completions", "500", "")
	counter.(prometheus.Metric).Write(&metric)

	if metric.GetCounter().GetValue() != 1 {
		t.Fatalf("expected counter=1, got %v", metric.GetCounter().GetValue())
	}
}
