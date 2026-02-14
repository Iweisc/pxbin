package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sertdev/pxbin/internal/auth"
	"github.com/sertdev/pxbin/internal/store"
)

type keysHandler struct {
	store *store.Store
}

func (h *keysHandler) List(w http.ResponseWriter, r *http.Request) {
	keyType := r.URL.Query().Get("type")
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "per_page", 50)

	switch keyType {
	case "management":
		keys, total, err := h.store.ListManagementKeys(r.Context(), page, perPage)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "Failed to list keys")
			return
		}
		writeDataPaginated(w, keys, total, page, perPage)
	default:
		keys, total, err := h.store.ListLLMKeys(r.Context(), page, perPage)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "Failed to list keys")
			return
		}
		writeDataPaginated(w, keys, total, page, perPage)
	}
}

type createKeyRequest struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	RateLimit   *int     `json:"rate_limit"`
	Permissions []string `json:"permissions"`
}

type createKeyResponse struct {
	Key       string `json:"key"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

func (h *keysHandler) Create(w http.ResponseWriter, r *http.Request) {
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
			perms = []string{"read"}
		}
		record, err := h.store.CreateManagementKey(r.Context(), hash, prefix, req.Name, perms)
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
		record, err := h.store.CreateLLMKey(r.Context(), hash, prefix, req.Name, req.RateLimit)
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

func (h *keysHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid ID format")
		return
	}

	keyType := r.URL.Query().Get("type")

	switch keyType {
	case "management":
		var updates store.ManagementKeyUpdate
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
			return
		}
		if err := h.store.UpdateManagementKey(r.Context(), id, updates); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "Failed to update key")
			return
		}
	default:
		var updates store.LLMKeyUpdate
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
			return
		}
		if err := h.store.UpdateLLMKey(r.Context(), id, updates); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "Failed to update key")
			return
		}
	}

	writeJSON(w, http.StatusOK, response{Data: map[string]string{"status": "updated"}})
}

func (h *keysHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid ID format")
		return
	}

	keyType := r.URL.Query().Get("type")

	switch keyType {
	case "management":
		if err := h.store.DeactivateManagementKey(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "Failed to deactivate key")
			return
		}
	default:
		if err := h.store.DeactivateLLMKey(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "Failed to deactivate key")
			return
		}
	}

	writeJSON(w, http.StatusOK, response{Data: map[string]string{"status": "deactivated"}})
}
