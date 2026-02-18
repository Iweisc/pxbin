package proxy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sertdev/pxbin/internal/config"
	"github.com/sertdev/pxbin/internal/ratelimit"
	"github.com/sertdev/pxbin/internal/server"
)

type mockProxyHandler struct{}

func (m *mockProxyHandler) HandleAnthropic(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"id":"msg_123","type":"message","role":"assistant","content":[{"type":"text","text":"Hello"}],"model":"claude-3-opus","usage":{"input_tokens":10,"output_tokens":5}}`))
}

func (m *mockProxyHandler) HandleOpenAI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"id":"chatcmpl-123","object":"chat.completion","choices":[{"message":{"role":"assistant","content":"Hi"}}]}`))
}

func (m *mockProxyHandler) HandleOpenAIResponses(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"id":"resp_123"}`))
}

func newTestRouter(opts *server.Opts) *chi.Mux {
	cfg := &config.Config{CORSOrigins: []string{"*"}}
	noAuth := func(next http.Handler) http.Handler { return next }
	return server.New(cfg, &mockProxyHandler{}, noAuth, chi.NewRouter(), nil, nil, opts)
}

func TestSecurityHeadersPresent(t *testing.T) {
	router := newTestRouter(nil)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":       "DENY",
		"X-XSS-Protection":      "0",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Permissions-Policy":     "camera=(), microphone=(), geolocation=()",
	}

	for h, want := range headers {
		if got := rec.Header().Get(h); got != want {
			t.Errorf("header %s: got %q, want %q", h, got, want)
		}
	}
}

func TestHealthEndpoint(t *testing.T) {
	router := newTestRouter(nil)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if body != `{"status":"ok"}` {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestRateLimitReturns429(t *testing.T) {
	limiter := ratelimit.NewLimiter(1, 1) // 1 rps, burst of 1
	defer limiter.Close()

	router := newTestRouter(&server.Opts{RateLimiter: limiter})

	// First request should succeed.
	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Second request should be rate-limited.
	req = httptest.NewRequest("POST", "/v1/messages", strings.NewReader(`{}`))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d; body=%s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "rate_limit") {
		t.Fatalf("expected rate limit error in body, got: %s", body)
	}
}

func TestAnthropicProxyRoundTrip(t *testing.T) {
	router := newTestRouter(nil)

	body := `{"model":"claude-3-opus","max_tokens":100,"messages":[{"role":"user","content":"Hi"}]}`
	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", rec.Code, rec.Body.String())
	}

	respBody, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(respBody), "msg_123") {
		t.Fatalf("unexpected response body: %s", respBody)
	}
}

func TestOpenAIProxyRoundTrip(t *testing.T) {
	router := newTestRouter(nil)

	body := `{"model":"gpt-4","messages":[{"role":"user","content":"Hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", rec.Code, rec.Body.String())
	}

	respBody, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(respBody), "chatcmpl-123") {
		t.Fatalf("unexpected response body: %s", respBody)
	}
}
