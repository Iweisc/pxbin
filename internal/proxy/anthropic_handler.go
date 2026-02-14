package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sertdev/pxbin/internal/auth"
	"github.com/sertdev/pxbin/internal/logging"
	"github.com/sertdev/pxbin/internal/translate"
)

// resolveUpstream looks up the model's linked upstream from the DB. If found,
// it returns a cached UpstreamClient and the upstream's format. If the model
// has no linked upstream, it returns an error — all upstreams must be
// configured via the management API.
func (h *Handler) resolveUpstream(ctx context.Context, modelName string) (*UpstreamClient, string, error) {
	mw, err := h.modelCache.GetModelWithUpstream(ctx, modelName)
	if err != nil {
		return nil, "", fmt.Errorf("resolve upstream: %w", err)
	}
	if mw == nil {
		return nil, "", fmt.Errorf("no upstream configured for model %q", modelName)
	}
	client := h.clients.Get(*mw.UpstreamID, mw.UpstreamBaseURL, mw.UpstreamAPIKey)
	return client, mw.UpstreamFormat, nil
}

// HandleAnthropic proxies Anthropic /v1/messages requests. Depending on the
// upstream format, it either passes through natively or translates to OpenAI.
func (h *Handler) HandleAnthropic(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	keyID := auth.GetKeyIDFromContext(r.Context())

	// Read and parse the Anthropic request body.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var anthropicReq translate.AnthropicRequest
	if err := json.Unmarshal(body, &anthropicReq); err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON in request body")
		return
	}

	// Resolve which upstream to use based on the model.
	client, format, err := h.resolveUpstream(r.Context(), anthropicReq.Model)
	if err != nil {
		writeAnthropicError(w, http.StatusInternalServerError, "api_error", "Failed to resolve upstream")
		return
	}

	if format == "openai" {
		h.handleAnthropicToOpenAI(w, r, client, body, &anthropicReq, keyID, start)
	} else {
		// "anthropic" or "" (fallback) → native passthrough
		h.handleAnthropicNative(w, r, client, body, &anthropicReq, keyID, start)
	}
}

// handleAnthropicNative passes the request through to an Anthropic-format
// upstream using x-api-key auth.
func (h *Handler) handleAnthropicNative(w http.ResponseWriter, r *http.Request, client *UpstreamClient, body []byte, anthropicReq *translate.AnthropicRequest, keyID uuid.UUID, start time.Time) {
	extraHeaders := http.Header{
		"X-Api-Key":         {client.apiKey},
		"Anthropic-Version": {"2023-06-01"},
	}
	upstreamResp, err := client.DoRaw(r.Context(), "POST", "/v1/messages", bytes.NewReader(body), extraHeaders)
	if err != nil {
		latency := time.Since(start)
		h.logger.Log(&logging.LogEntry{
			KeyID:        keyID,
			Timestamp:    start,
			Method:       r.Method,
			Path:         r.URL.Path,
			Model:        anthropicReq.Model,
			InputFormat:  "anthropic",
			StatusCode:   http.StatusBadGateway,
			LatencyMS:    int(latency.Milliseconds()),
			ErrorMessage: "upstream connection error: " + err.Error(),
		})
		writeAnthropicError(w, http.StatusBadGateway, "api_error", "Failed to connect to upstream")
		return
	}
	defer upstreamResp.Body.Close()

	// Handle upstream errors — pass through as-is.
	if upstreamResp.StatusCode >= 400 {
		upstreamBody, _ := io.ReadAll(upstreamResp.Body)

		latency := time.Since(start)
		h.logger.Log(&logging.LogEntry{
			KeyID:        keyID,
			Timestamp:    start,
			Method:       r.Method,
			Path:         r.URL.Path,
			Model:        anthropicReq.Model,
			InputFormat:  "anthropic",
			StatusCode:   upstreamResp.StatusCode,
			LatencyMS:    int(latency.Milliseconds()),
			ErrorMessage: string(upstreamBody),
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(upstreamResp.StatusCode)
		w.Write(upstreamBody)
		return
	}

	// Streaming response — passthrough SSE and capture usage.
	if anthropicReq.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeAnthropicError(w, http.StatusInternalServerError, "api_error", "Streaming not supported")
			return
		}

		result := passthroughAnthropicStream(upstreamResp.Body, w, flusher)

		latency := time.Since(start)
		cost := h.billing.CalculateCost(anthropicReq.Model, result.InputTokens, result.OutputTokens)
		h.logger.Log(&logging.LogEntry{
			KeyID:               keyID,
			Timestamp:           start,
			Method:              r.Method,
			Path:                r.URL.Path,
			Model:               anthropicReq.Model,
			InputFormat:         "anthropic",
			StatusCode:          http.StatusOK,
			LatencyMS:           int(latency.Milliseconds()),
			InputTokens:         result.InputTokens,
			OutputTokens:        result.OutputTokens,
			CacheCreationTokens: result.CacheCreationTokens,
			CacheReadTokens:     result.CacheReadTokens,
			Cost:                cost,
		})
		return
	}

	// Non-streaming response — passthrough and capture usage.
	upstreamBody, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "api_error", "Failed to read upstream response")
		return
	}

	var anthropicResp translate.AnthropicResponse
	if err := json.Unmarshal(upstreamBody, &anthropicResp); err == nil {
		inputTokens := anthropicResp.Usage.InputTokens
		outputTokens := anthropicResp.Usage.OutputTokens
		cacheCreation := anthropicResp.Usage.CacheCreationInputTokens
		cacheRead := anthropicResp.Usage.CacheReadInputTokens

		latency := time.Since(start)
		cost := h.billing.CalculateCost(anthropicReq.Model, inputTokens, outputTokens)
		h.logger.Log(&logging.LogEntry{
			KeyID:               keyID,
			Timestamp:           start,
			Method:              r.Method,
			Path:                r.URL.Path,
			Model:               anthropicReq.Model,
			InputFormat:         "anthropic",
			StatusCode:          http.StatusOK,
			LatencyMS:           int(latency.Milliseconds()),
			InputTokens:         inputTokens,
			OutputTokens:        outputTokens,
			CacheCreationTokens: cacheCreation,
			CacheReadTokens:     cacheRead,
			Cost:                cost,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(upstreamBody)
}

// handleAnthropicToOpenAI translates an Anthropic request to OpenAI format,
// sends it to the upstream, and translates the response back.
func (h *Handler) handleAnthropicToOpenAI(w http.ResponseWriter, r *http.Request, client *UpstreamClient, body []byte, anthropicReq *translate.AnthropicRequest, keyID uuid.UUID, start time.Time) {
	openaiReq, err := translate.AnthropicRequestToOpenAI(anthropicReq)
	if err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "Failed to translate request: "+err.Error())
		return
	}

	openaiBody, err := json.Marshal(openaiReq)
	if err != nil {
		writeAnthropicError(w, http.StatusInternalServerError, "api_error", "Failed to encode translated request")
		return
	}

	upstreamResp, err := client.Do(r.Context(), "POST", "/v1/chat/completions", bytes.NewReader(openaiBody), nil)
	if err != nil {
		latency := time.Since(start)
		h.logger.Log(&logging.LogEntry{
			KeyID:        keyID,
			Timestamp:    start,
			Method:       r.Method,
			Path:         r.URL.Path,
			Model:        anthropicReq.Model,
			InputFormat:  "anthropic",
			StatusCode:   http.StatusBadGateway,
			LatencyMS:    int(latency.Milliseconds()),
			ErrorMessage: "upstream connection error: " + err.Error(),
		})
		writeAnthropicError(w, http.StatusBadGateway, "api_error", "Failed to connect to upstream")
		return
	}
	defer upstreamResp.Body.Close()

	// Handle upstream errors.
	if upstreamResp.StatusCode >= 400 {
		upstreamBody, _ := io.ReadAll(upstreamResp.Body)
		latency := time.Since(start)
		h.logger.Log(&logging.LogEntry{
			KeyID:        keyID,
			Timestamp:    start,
			Method:       r.Method,
			Path:         r.URL.Path,
			Model:        anthropicReq.Model,
			InputFormat:  "anthropic",
			StatusCode:   upstreamResp.StatusCode,
			LatencyMS:    int(latency.Milliseconds()),
			ErrorMessage: string(upstreamBody),
		})
		writeAnthropicError(w, upstreamResp.StatusCode, "api_error", "Upstream error: "+string(upstreamBody))
		return
	}

	// Streaming translation: OpenAI SSE → Anthropic SSE.
	if anthropicReq.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeAnthropicError(w, http.StatusInternalServerError, "api_error", "Streaming not supported")
			return
		}

		result, _ := translate.TranslateOpenAIStreamToAnthropic(r.Context(), upstreamResp.Body, w, flusher, anthropicReq.Model)

		latency := time.Since(start)
		inputTokens := 0
		outputTokens := 0
		if result != nil {
			inputTokens = result.InputTokens
			outputTokens = result.OutputTokens
		}
		cost := h.billing.CalculateCost(anthropicReq.Model, inputTokens, outputTokens)
		h.logger.Log(&logging.LogEntry{
			KeyID:        keyID,
			Timestamp:    start,
			Method:       r.Method,
			Path:         r.URL.Path,
			Model:        anthropicReq.Model,
			InputFormat:  "anthropic",
			StatusCode:   http.StatusOK,
			LatencyMS:    int(latency.Milliseconds()),
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			Cost:         cost,
		})
		return
	}

	// Non-streaming translation: read OpenAI response and translate.
	upstreamBody, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "api_error", "Failed to read upstream response")
		return
	}

	var oaiResp translate.OpenAIResponse
	if err := json.Unmarshal(upstreamBody, &oaiResp); err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "api_error", "Failed to parse upstream response")
		return
	}

	anthropicResp, err := translate.OpenAIResponseToAnthropic(&oaiResp, anthropicReq.Model)
	if err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "api_error", "Failed to translate upstream response")
		return
	}
	inputTokens := 0
	outputTokens := 0
	if oaiResp.Usage != nil {
		inputTokens = oaiResp.Usage.PromptTokens
		outputTokens = oaiResp.Usage.CompletionTokens
	}

	latency := time.Since(start)
	cost := h.billing.CalculateCost(anthropicReq.Model, inputTokens, outputTokens)
	h.logger.Log(&logging.LogEntry{
		KeyID:        keyID,
		Timestamp:    start,
		Method:       r.Method,
		Path:         r.URL.Path,
		Model:        anthropicReq.Model,
		InputFormat:  "anthropic",
		StatusCode:   http.StatusOK,
		LatencyMS:    int(latency.Milliseconds()),
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Cost:         cost,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(anthropicResp)
}

// streamUsage holds usage info captured from an Anthropic SSE stream.
type streamUsage struct {
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
}

var newline = []byte("\n")

// passthroughAnthropicStream forwards Anthropic SSE events to the client
// while extracting usage information from message_start and message_delta events.
func passthroughAnthropicStream(upstream io.Reader, w http.ResponseWriter, flusher http.Flusher) streamUsage {
	var usage streamUsage

	scanner := bufio.NewScanner(upstream)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()

		// Pass every line through as-is.
		w.Write(line)
		w.Write(newline)

		// Flush on empty lines (SSE event boundary).
		if len(line) == 0 {
			flusher.Flush()
			continue
		}

		// Only inspect data lines that might contain usage info.
		// Cheap byte prefix check avoids JSON parsing on most lines.
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := line[6:]

		// Only parse the two event types that carry usage — skip content_block_delta,
		// content_block_start, content_block_stop, ping, etc.
		if bytes.Contains(data, []byte(`"message_start"`)) {
			var msgStart translate.MessageStartEvent
			if json.Unmarshal(data, &msgStart) == nil && msgStart.Type == "message_start" {
				usage.InputTokens = msgStart.Message.Usage.InputTokens
				usage.CacheCreationTokens = msgStart.Message.Usage.CacheCreationInputTokens
				usage.CacheReadTokens = msgStart.Message.Usage.CacheReadInputTokens
			}
		} else if bytes.Contains(data, []byte(`"message_delta"`)) {
			var msgDelta struct {
				Type  string `json:"type"`
				Usage *struct {
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			}
			if json.Unmarshal(data, &msgDelta) == nil && msgDelta.Type == "message_delta" && msgDelta.Usage != nil {
				usage.OutputTokens = msgDelta.Usage.OutputTokens
			}
		}
	}

	flusher.Flush()
	return usage
}
