package api

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

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

func (h *upstreamsHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UpstreamID string `json:"upstream_id"`
		BaseURL    string `json:"base_url"`
		APIKey     string `json:"api_key"`
		Format     string `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	var baseURL, apiKey, format string

	if req.UpstreamID != "" {
		id, err := uuid.Parse(req.UpstreamID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid upstream_id format")
			return
		}
		upstream, err := h.store.GetUpstream(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", "Failed to fetch upstream")
			return
		}
		if upstream == nil {
			writeError(w, http.StatusNotFound, "not_found", "Upstream not found")
			return
		}
		baseURL = upstream.BaseURL
		apiKey = upstream.APIKeyEncrypted // already decrypted by store
		format = upstream.Format
	} else if req.BaseURL != "" && req.APIKey != "" {
		baseURL = req.BaseURL
		apiKey = req.APIKey
		format = req.Format
		if format == "" {
			format = "openai"
		}
	} else {
		writeError(w, http.StatusBadRequest, "invalid_request", "Provide upstream_id or base_url + api_key")
		return
	}

	baseURL = strings.TrimRight(baseURL, "/")
	start := time.Now()

	result := healthCheckResult{Healthy: false}

	if format == "anthropic" {
		h.healthCheckAnthropic(baseURL, apiKey, &result)
	} else {
		h.healthCheckOpenAI(baseURL, apiKey, &result)
	}

	result.LatencyMs = time.Since(start).Milliseconds()
	writeData(w, result)
}

type healthCheckResult struct {
	Healthy     bool    `json:"healthy"`
	ModelsFound int     `json:"models_found"`
	TestedModel string  `json:"tested_model"`
	LatencyMs   int64   `json:"latency_ms"`
	Error       *string `json:"error"`
}

func (h *upstreamsHandler) healthCheckOpenAI(baseURL, apiKey string, result *healthCheckResult) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Step 1: List models
	modelsReq, _ := http.NewRequest("GET", baseURL+"/v1/models", nil)
	modelsReq.Header.Set("Authorization", "Bearer "+apiKey)

	modelsResp, err := client.Do(modelsReq)
	if err != nil {
		errMsg := fmt.Sprintf("Connection failed: %s", err.Error())
		result.Error = &errMsg
		return
	}
	defer modelsResp.Body.Close()

	if modelsResp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Models endpoint returned %d", modelsResp.StatusCode)
		result.Error = &errMsg
		return
	}

	var modelsBody struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(modelsResp.Body).Decode(&modelsBody); err != nil {
		errMsg := fmt.Sprintf("Failed to parse models response: %s", err.Error())
		result.Error = &errMsg
		return
	}

	if len(modelsBody.Data) == 0 {
		errMsg := "No models returned by upstream"
		result.Error = &errMsg
		return
	}

	result.ModelsFound = len(modelsBody.Data)
	model := modelsBody.Data[rand.IntN(len(modelsBody.Data))].ID
	result.TestedModel = model

	// Step 2: Chat completion
	client.Timeout = 15 * time.Second
	completionPayload := fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"hi"}],"max_tokens":1}`, model)
	completionReq, _ := http.NewRequest("POST", baseURL+"/v1/chat/completions", strings.NewReader(completionPayload))
	completionReq.Header.Set("Authorization", "Bearer "+apiKey)
	completionReq.Header.Set("Content-Type", "application/json")

	completionResp, err := client.Do(completionReq)
	if err != nil {
		errMsg := fmt.Sprintf("Completion request failed: %s", err.Error())
		result.Error = &errMsg
		return
	}
	defer completionResp.Body.Close()

	if completionResp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Completion returned %d", completionResp.StatusCode)
		result.Error = &errMsg
		return
	}

	result.Healthy = true
}

func (h *upstreamsHandler) healthCheckAnthropic(baseURL, apiKey string, result *healthCheckResult) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Step 1: List models
	modelsReq, _ := http.NewRequest("GET", baseURL+"/v1/models", nil)
	modelsReq.Header.Set("x-api-key", apiKey)
	modelsReq.Header.Set("anthropic-version", "2023-06-01")

	modelsResp, err := client.Do(modelsReq)
	if err != nil {
		errMsg := fmt.Sprintf("Connection failed: %s", err.Error())
		result.Error = &errMsg
		return
	}
	defer modelsResp.Body.Close()

	if modelsResp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Models endpoint returned %d", modelsResp.StatusCode)
		result.Error = &errMsg
		return
	}

	var modelsBody struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(modelsResp.Body).Decode(&modelsBody); err != nil {
		errMsg := fmt.Sprintf("Failed to parse models response: %s", err.Error())
		result.Error = &errMsg
		return
	}

	if len(modelsBody.Data) == 0 {
		errMsg := "No models returned by upstream"
		result.Error = &errMsg
		return
	}

	result.ModelsFound = len(modelsBody.Data)
	model := modelsBody.Data[rand.IntN(len(modelsBody.Data))].ID
	result.TestedModel = model

	// Step 2: Messages API
	client.Timeout = 15 * time.Second
	messagesPayload := fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"hi"}],"max_tokens":1}`, model)
	messagesReq, _ := http.NewRequest("POST", baseURL+"/v1/messages", strings.NewReader(messagesPayload))
	messagesReq.Header.Set("x-api-key", apiKey)
	messagesReq.Header.Set("anthropic-version", "2023-06-01")
	messagesReq.Header.Set("Content-Type", "application/json")

	messagesResp, err := client.Do(messagesReq)
	if err != nil {
		errMsg := fmt.Sprintf("Messages request failed: %s", err.Error())
		result.Error = &errMsg
		return
	}
	defer messagesResp.Body.Close()

	if messagesResp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Messages endpoint returned %d", messagesResp.StatusCode)
		result.Error = &errMsg
		return
	}

	result.Healthy = true
}
