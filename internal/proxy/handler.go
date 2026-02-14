package proxy

import (
	"encoding/json"
	"net/http"

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

func writeAnthropicError(w http.ResponseWriter, statusCode int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(translate.AnthropicErrorResponse{
		Type: "error",
		Error: translate.AnthropicError{
			Type:    errType,
			Message: message,
		},
	})
}

func writeOpenAIError(w http.ResponseWriter, statusCode int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(translate.OpenAIErrorResponse{
		Error: translate.OpenAIError{
			Message: message,
			Type:    errType,
		},
	})
}
