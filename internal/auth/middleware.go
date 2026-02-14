package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/sertdev/pxbin/internal/store"
)

type contextKey int

const (
	ctxKeyLLMKeyID contextKey = iota
	ctxKeyLLMKey
	ctxKeyManagementKeyID
	ctxKeyManagementKey
)

func GetKeyIDFromContext(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(ctxKeyLLMKeyID).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

func GetKeyFromContext(ctx context.Context) *store.LLMAPIKey {
	if k, ok := ctx.Value(ctxKeyLLMKey).(*store.LLMAPIKey); ok {
		return k
	}
	return nil
}

func GetManagementKeyIDFromContext(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(ctxKeyManagementKeyID).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

func GetManagementKeyFromContext(ctx context.Context) *store.ManagementAPIKey {
	if k, ok := ctx.Value(ctxKeyManagementKey).(*store.ManagementAPIKey); ok {
		return k
	}
	return nil
}

func LLMAuthMiddleware(s *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := extractAPIKey(r)
			if key == "" {
				writeAuthError(w, r, http.StatusUnauthorized, "Missing API key")
				return
			}

			hash := HashKey(key)
			record, err := s.GetLLMKeyByHash(r.Context(), hash)
			if err != nil {
				writeAuthError(w, r, http.StatusInternalServerError, "Internal server error")
				return
			}
			if record == nil {
				writeAuthError(w, r, http.StatusUnauthorized, "Invalid API key")
				return
			}
			if !record.IsActive {
				writeAuthError(w, r, http.StatusForbidden, "API key is deactivated")
				return
			}

			go func() {
				_ = s.UpdateLLMKeyLastUsed(context.Background(), record.ID)
			}()

			ctx := context.WithValue(r.Context(), ctxKeyLLMKeyID, record.ID)
			ctx = context.WithValue(ctx, ctxKeyLLMKey, record)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ManagementAuthMiddleware(s *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := extractAPIKey(r)
			if key == "" {
				writeJSONError(w, http.StatusUnauthorized, "Missing API key")
				return
			}

			hash := HashKey(key)
			record, err := s.GetManagementKeyByHash(r.Context(), hash)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "Internal server error")
				return
			}
			if record == nil {
				writeJSONError(w, http.StatusUnauthorized, "Invalid API key")
				return
			}
			if !record.IsActive {
				writeJSONError(w, http.StatusForbidden, "API key is deactivated")
				return
			}

			ctx := context.WithValue(r.Context(), ctxKeyManagementKeyID, record.ID)
			ctx = context.WithValue(ctx, ctxKeyManagementKey, record)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractAPIKey(r *http.Request) string {
	if key := r.Header.Get("x-api-key"); key != "" {
		return key
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func writeAuthError(w http.ResponseWriter, r *http.Request, status int, message string) {
	if strings.HasPrefix(r.URL.Path, "/v1/messages") {
		writeAnthropicError(w, status, message)
	} else {
		writeOpenAIError(w, status, message)
	}
}

func writeAnthropicError(w http.ResponseWriter, status int, message string) {
	errType := "authentication_error"
	if status == http.StatusForbidden {
		errType = "permission_error"
	} else if status == http.StatusInternalServerError {
		errType = "api_error"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"type": "error",
		"error": map[string]any{
			"type":    errType,
			"message": message,
		},
	})
}

func writeOpenAIError(w http.ResponseWriter, status int, message string) {
	errType := "invalid_api_key"
	if status == http.StatusForbidden {
		errType = "access_denied"
	} else if status == http.StatusInternalServerError {
		errType = "server_error"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    errType,
			"code":    errType,
		},
	})
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": message,
	})
}
