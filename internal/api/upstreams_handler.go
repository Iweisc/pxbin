package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sertdev/pxbin/internal/store"
)

type upstreamsHandler struct {
	store *store.Store
}

func (h *upstreamsHandler) List(w http.ResponseWriter, r *http.Request) {
	upstreams, err := h.store.ListUpstreams(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to list upstreams")
		return
	}
	writeData(w, upstreams)
}

func (h *upstreamsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req store.UpstreamCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.Name == "" || req.BaseURL == "" || req.APIKey == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Name, base_url, and api_key are required")
		return
	}
	if req.Format == "" {
		req.Format = "openai"
	}
	if req.Format != "openai" && req.Format != "anthropic" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Format must be 'openai' or 'anthropic'")
		return
	}

	upstream, err := h.store.CreateUpstream(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to create upstream")
		return
	}

	writeJSON(w, http.StatusCreated, response{Data: upstream})
}

func (h *upstreamsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid ID format")
		return
	}

	var updates store.UpstreamUpdate
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if err := h.store.UpdateUpstream(r.Context(), id, &updates); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to update upstream")
		return
	}

	writeJSON(w, http.StatusOK, response{Data: map[string]string{"status": "updated"}})
}

func (h *upstreamsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid ID format")
		return
	}

	if err := h.store.DeleteUpstream(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to delete upstream")
		return
	}

	writeJSON(w, http.StatusOK, response{Data: map[string]string{"status": "deleted"}})
}

func (h *upstreamsHandler) BulkDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "At least one ID is required")
		return
	}

	ids := make([]uuid.UUID, 0, len(req.IDs))
	for _, raw := range req.IDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Invalid ID: %s", raw))
			return
		}
		ids = append(ids, id)
	}

	deleted, err := h.store.DeleteUpstreams(r.Context(), ids)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to delete upstreams")
		return
	}

	writeJSON(w, http.StatusOK, response{Data: map[string]any{"deleted": deleted}})
}
