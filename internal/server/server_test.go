package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sertdev/pxbin/internal/config"
)

type stubProxyHandler struct {
	responsesCalls int
	lastPath       string
}

func (s *stubProxyHandler) HandleAnthropic(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *stubProxyHandler) HandleOpenAI(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *stubProxyHandler) HandleOpenAIResponses(w http.ResponseWriter, r *http.Request) {
	s.responsesCalls++
	s.lastPath = r.URL.Path
	w.WriteHeader(http.StatusNoContent)
}

func TestResponsesCompactRoute(t *testing.T) {
	cfg := &config.Config{CORSOrigins: []string{"*"}}
	proxy := &stubProxyHandler{}
	router := New(
		cfg,
		proxy,
		func(next http.Handler) http.Handler { return next },
		chi.NewRouter(),
		nil,
		nil,
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses/compact", strings.NewReader(`{"model":"gpt-5.3-codex"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d body=%q", http.StatusNoContent, rec.Code, rec.Body.String())
	}
	if proxy.responsesCalls != 1 {
		t.Fatalf("expected HandleOpenAIResponses to be called once, got %d", proxy.responsesCalls)
	}
	if proxy.lastPath != "/v1/responses/compact" {
		t.Fatalf("expected path /v1/responses/compact, got %q", proxy.lastPath)
	}
}
