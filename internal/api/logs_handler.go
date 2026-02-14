package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sertdev/pxbin/internal/store"
)

type logsHandler struct {
	store *store.Store
}

func (h *logsHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := store.LogFilter{
		Page:    queryInt(r, "page", 1),
		PerPage: queryInt(r, "per_page", 50),
	}

	if v := q.Get("key_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid key_id format")
			return
		}
		filter.KeyID = &id
	}
	if v := q.Get("model"); v != "" {
		filter.Model = &v
	}
	if v := q.Get("status_code"); v != "" {
		code, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid status_code")
			return
		}
		filter.StatusCode = &code
	}
	if v := q.Get("input_format"); v != "" {
		filter.InputFormat = &v
	}
	if v := q.Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid 'from' timestamp, use RFC3339")
			return
		}
		filter.DateFrom = &t
	}
	if v := q.Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid 'to' timestamp, use RFC3339")
			return
		}
		filter.DateTo = &t
	}

	logs, total, err := h.store.ListLogs(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to list logs")
		return
	}

	writeDataPaginated(w, logs, total, filter.Page, filter.PerPage)
}

func (h *logsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid ID format")
		return
	}

	log, err := h.store.GetLog(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to get log")
		return
	}
	if log == nil {
		writeError(w, http.StatusNotFound, "not_found", "Log not found")
		return
	}

	writeData(w, log)
}
