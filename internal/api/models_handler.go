package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sertdev/pxbin/internal/billing"
	"github.com/sertdev/pxbin/internal/pricing"
	"github.com/sertdev/pxbin/internal/store"
)

type modelsHandler struct {
	store   *store.Store
	billing *billing.Tracker
}

func (h *modelsHandler) List(w http.ResponseWriter, r *http.Request) {
	models, err := h.store.ListModels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to list models")
		return
	}
	writeData(w, models)
}

func (h *modelsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req store.ModelCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Name is required")
		return
	}

	model, err := h.store.CreateModel(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to create model")
		return
	}

	writeJSON(w, http.StatusCreated, response{Data: model})
}

func (h *modelsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid ID format")
		return
	}

	var updates store.ModelUpdate
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	if err := h.store.UpdateModel(r.Context(), id, &updates); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to update model")
		return
	}

	writeJSON(w, http.StatusOK, response{Data: map[string]string{"status": "updated"}})
}

func (h *modelsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid ID format")
		return
	}

	if err := h.store.DeleteModel(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to delete model")
		return
	}

	writeJSON(w, http.StatusOK, response{Data: map[string]string{"status": "deleted"}})
}

func (h *modelsHandler) BulkDelete(w http.ResponseWriter, r *http.Request) {
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

	deleted, err := h.store.DeleteModels(r.Context(), ids)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to delete models")
		return
	}

	_ = h.billing.RefreshPricing(r.Context())

	writeJSON(w, http.StatusOK, response{Data: map[string]any{"deleted": deleted}})
}

type discoverRequest struct {
	UpstreamID string `json:"upstream_id"`
}

type discoveredModel struct {
	ID      string `json:"id"`
	OwnedBy string `json:"owned_by"`
}

func (h *modelsHandler) Discover(w http.ResponseWriter, r *http.Request) {
	var req discoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.UpstreamID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "upstream_id is required")
		return
	}

	upstreamID, err := uuid.Parse(req.UpstreamID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid upstream_id format")
		return
	}

	upstream, err := h.store.GetUpstream(r.Context(), upstreamID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to fetch upstream")
		return
	}
	if upstream == nil {
		writeError(w, http.StatusNotFound, "not_found", "Upstream not found")
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	upstreamReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, upstream.BaseURL+"/v1/models", nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid upstream base_url")
		return
	}
	if upstream.APIKeyEncrypted != "" {
		upstreamReq.Header.Set("Authorization", "Bearer "+upstream.APIKeyEncrypted)
	}

	resp, err := client.Do(upstreamReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream_error", fmt.Sprintf("Failed to reach upstream: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		writeError(w, http.StatusBadGateway, "upstream_error", fmt.Sprintf("Upstream returned %d: %s", resp.StatusCode, string(body)))
		return
	}

	var upstreamResp struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&upstreamResp); err != nil {
		writeError(w, http.StatusBadGateway, "upstream_error", "Failed to parse upstream response")
		return
	}

	models := make([]discoveredModel, len(upstreamResp.Data))
	for i, m := range upstreamResp.Data {
		models[i] = discoveredModel{ID: m.ID, OwnedBy: m.OwnedBy}
	}

	writeData(w, models)
}

type importRequest struct {
	UpstreamID string `json:"upstream_id"`
	Models     []struct {
		Name     string `json:"name"`
		Provider string `json:"provider"`
	} `json:"models"`
}

func (h *modelsHandler) Import(w http.ResponseWriter, r *http.Request) {
	var req importRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.UpstreamID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "upstream_id is required")
		return
	}
	if len(req.Models) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "At least one model is required")
		return
	}

	upstreamID, err := uuid.Parse(req.UpstreamID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid upstream_id format")
		return
	}

	upstream, err := h.store.GetUpstream(r.Context(), upstreamID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to fetch upstream")
		return
	}
	if upstream == nil {
		writeError(w, http.StatusNotFound, "not_found", "Upstream not found")
		return
	}

	// Fetch pricing data from LiteLLM
	pricingData, err := pricing.FetchLiteLLMPricing(r.Context())
	if err != nil {
		// Non-fatal: log and continue with zero pricing
		pricingData = make(map[string]*pricing.ModelPricing)
	}

	created := 0
	skipped := 0
	for _, m := range req.Models {
		existing, err := h.store.GetModelByName(r.Context(), m.Name)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "Failed to check existing model")
			return
		}
		if existing != nil {
			skipped++
			continue
		}

		// Look up pricing from LiteLLM
		inputCost := 0.0
		outputCost := 0.0
		if p, ok := pricingData[m.Name]; ok {
			inputCost = p.InputCostPerMillion
			outputCost = p.OutputCostPerMillion
		}

		_, err = h.store.CreateModel(r.Context(), &store.ModelCreate{
			Name:                 m.Name,
			Provider:             m.Provider,
			UpstreamID:           &upstreamID,
			InputCostPerMillion:  inputCost,
			OutputCostPerMillion: outputCost,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", fmt.Sprintf("Failed to create model %s", m.Name))
			return
		}
		created++
	}

	// Refresh billing tracker immediately
	_ = h.billing.RefreshPricing(r.Context())

	writeJSON(w, http.StatusCreated, response{Data: map[string]any{
		"upstream":       upstream,
		"models_created": created,
		"models_skipped": skipped,
	}})
}

func (h *modelsHandler) SyncPricing(w http.ResponseWriter, r *http.Request) {
	pricingData, err := pricing.FetchLiteLLMPricing(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream_error", fmt.Sprintf("Failed to fetch pricing: %v", err))
		return
	}

	models, err := h.store.ListModels(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to list models")
		return
	}

	updated := 0
	notFound := 0
	for _, model := range models {
		if p, ok := pricingData[model.Name]; ok {
			inputCost := p.InputCostPerMillion
			outputCost := p.OutputCostPerMillion
			err := h.store.UpdateModel(r.Context(), model.ID, &store.ModelUpdate{
				InputCostPerMillion:  &inputCost,
				OutputCostPerMillion: &outputCost,
			})
			if err != nil {
				writeError(w, http.StatusInternalServerError, "server_error", fmt.Sprintf("Failed to update model %s", model.Name))
				return
			}
			updated++
		} else {
			notFound++
		}
	}

	// Refresh billing tracker immediately so new requests get correct pricing
	_ = h.billing.RefreshPricing(r.Context())

	writeJSON(w, http.StatusOK, response{Data: map[string]any{
		"models_updated":  updated,
		"models_not_found": notFound,
		"total_models":    len(models),
	}})
}
