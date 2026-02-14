package proxy

import (
	"io"
	"net/http"

	json "github.com/bytedance/sonic"

	"github.com/sertdev/pxbin/internal/billing"
	"github.com/sertdev/pxbin/internal/logging"
	"github.com/sertdev/pxbin/internal/store"
	"github.com/sertdev/pxbin/internal/translate"
)

// Handler contains the shared dependencies for the Anthropic and OpenAI proxy
// endpoints.
type Handler struct {
	clients    *ClientCache
	modelCache *ModelCache
	store      *store.Store
	logger     *logging.AsyncLogger
	billing    *billing.Tracker
}

// NewHandler creates a Handler wired up to a client cache, model cache, store,
// logger and billing tracker.
func NewHandler(clients *ClientCache, modelCache *ModelCache, s *store.Store, logger *logging.AsyncLogger, billing *billing.Tracker) *Handler {
	return &Handler{
		clients:    clients,
		modelCache: modelCache,
		store:      s,
		logger:     logger,
		billing:    billing,
	}
}

// readBody reads the full request body. When Content-Length is known it
// pre-allocates a single buffer of the exact size, avoiding the repeated
// grow-and-copy cycles that io.ReadAll performs (which start at 512 bytes
// and double each time — ~20 allocs + 2x body size in wasted copies for
// a 500KB Claude Code payload).
func readBody(r *http.Request) ([]byte, error) {
	if r.ContentLength > 0 {
		buf := make([]byte, r.ContentLength)
		_, err := io.ReadFull(r.Body, buf)
		return buf, err
	}
	return io.ReadAll(r.Body)
}

func writeAnthropicError(w http.ResponseWriter, statusCode int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	b, _ := json.Marshal(translate.AnthropicErrorResponse{
		Type: "error",
		Error: translate.AnthropicError{
			Type:    errType,
			Message: message,
		},
	})
	w.Write(b)
}

func writeOpenAIError(w http.ResponseWriter, statusCode int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	b, _ := json.Marshal(translate.OpenAIErrorResponse{
		Error: translate.OpenAIError{
			Message: message,
			Type:    errType,
		},
	})
	w.Write(b)
}
