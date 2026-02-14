package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sertdev/pxbin/internal/auth"
	"github.com/sertdev/pxbin/internal/store"
)

// NewBootstrapHandler returns an http.HandlerFunc that creates API keys
// when authenticated with the bootstrap key. Returns nil if bootstrapKey
// is empty (disabled).
func NewBootstrapHandler(s *store.Store, bootstrapKey string) http.HandlerFunc {
	if bootstrapKey == "" {
		return nil
	}

	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("x-api-key")
		if key == "" {
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				key = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		if subtle.ConstantTimeCompare([]byte(key), []byte(bootstrapKey)) != 1 {
			writeError(w, http.StatusUnauthorized, "authentication_error", "Invalid bootstrap key")
			return
		}

		var req createKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
			return
		}

		switch req.Type {
		case "management":
			plaintext, hash, prefix := auth.GenerateManagementKey()
			perms := req.Permissions
			if len(perms) == 0 {
				perms = []string{"read", "write"}
			}
			record, err := s.CreateManagementKey(r.Context(), hash, prefix, req.Name, perms)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "server_error", "Failed to create key")
				return
			}
			writeJSON(w, http.StatusCreated, response{Data: createKeyResponse{
				Key:       plaintext,
				ID:        record.ID.String(),
				Name:      record.Name,
				CreatedAt: record.CreatedAt.Format("2006-01-02T15:04:05Z"),
			}})
		case "llm", "":
			plaintext, hash, prefix := auth.GenerateLLMKey()
			record, err := s.CreateLLMKey(r.Context(), hash, prefix, req.Name, req.RateLimit)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "server_error", "Failed to create key")
				return
			}
			writeJSON(w, http.StatusCreated, response{Data: createKeyResponse{
				Key:       plaintext,
				ID:        record.ID.String(),
				Name:      record.Name,
				CreatedAt: record.CreatedAt.Format("2006-01-02T15:04:05Z"),
			}})
		default:
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid key type, must be 'llm' or 'management'")
		}
	}
}
