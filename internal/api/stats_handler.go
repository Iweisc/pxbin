package api

import (
	"net/http"

	"github.com/sertdev/pxbin/internal/store"
)

type statsHandler struct {
	store *store.Store
}

func (h *statsHandler) Overview(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	stats, err := h.store.GetOverviewStats(r.Context(), period)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to get overview stats")
		return
	}
	writeData(w, stats)
}

func (h *statsHandler) ByKey(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}
	page := queryInt(r, "page", 1)
	perPage := queryInt(r, "per_page", 50)

	stats, total, err := h.store.GetStatsByKey(r.Context(), period, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to get key stats")
		return
	}
	writeDataPaginated(w, stats, total, page, perPage)
}

func (h *statsHandler) ByModel(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	stats, err := h.store.GetStatsByModel(r.Context(), period)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to get model stats")
		return
	}
	writeData(w, stats)
}

func (h *statsHandler) TimeSeries(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "1h"
	}

	stats, err := h.store.GetTimeSeries(r.Context(), period, interval)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to get time series")
		return
	}
	writeData(w, stats)
}

func (h *statsHandler) Latency(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	stats, err := h.store.GetLatencyPercentiles(r.Context(), period)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to get latency stats")
		return
	}
	writeData(w, stats)
}
