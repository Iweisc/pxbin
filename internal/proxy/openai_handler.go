package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sertdev/pxbin/internal/auth"
	"github.com/sertdev/pxbin/internal/logging"
	"github.com/sertdev/pxbin/internal/translate"
)

// HandleOpenAI proxies OpenAI-format requests to the upstream. If the
// upstream's format is "openai" the request passes through unchanged;
// "anthropic" upstreams are currently unsupported and return an error.
func (h *Handler) HandleOpenAI(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	keyID := auth.GetKeyIDFromContext(r.Context())

	// Read the request body to extract the model name for routing.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var partial struct {
		Model string `json:"model"`
	}
	json.Unmarshal(body, &partial)

	// Resolve upstream based on model.
	client, format, err := h.resolveUpstream(r.Context(), partial.Model)
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, "server_error", "Failed to resolve upstream")
		return
	}

	if format == "anthropic" {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "Model is linked to an Anthropic-format upstream; use the Anthropic endpoint instead")
		return
	}

	// Forward the request body to the upstream unchanged.
	upstreamResp, err := client.Do(r.Context(), r.Method, "/v1/chat/completions", bytes.NewReader(body), nil)
	if err != nil {
		latency := time.Since(start)
		h.logger.Log(&logging.LogEntry{
			KeyID:        keyID,
			Timestamp:    start,
			Method:       r.Method,
			Path:         r.URL.Path,
			Model:        partial.Model,
			InputFormat:  "openai",
			StatusCode:   http.StatusBadGateway,
			LatencyMS:    int(latency.Milliseconds()),
			ErrorMessage: "upstream connection error: " + err.Error(),
		})
		writeOpenAIError(w, http.StatusBadGateway, "server_error", "Failed to connect to upstream")
		return
	}
	defer upstreamResp.Body.Close()

	// Copy relevant upstream response headers.
	for _, hdr := range []string{"Content-Type", "X-Request-Id"} {
		if v := upstreamResp.Header.Get(hdr); v != "" {
			w.Header().Set(hdr, v)
		}
	}

	// Handle upstream errors: pass through as-is.
	if upstreamResp.StatusCode >= 400 {
		upstreamBody, _ := io.ReadAll(upstreamResp.Body)
		latency := time.Since(start)

		h.logger.Log(&logging.LogEntry{
			KeyID:        keyID,
			Timestamp:    start,
			Method:       r.Method,
			Path:         r.URL.Path,
			Model:        partial.Model,
			InputFormat:  "openai",
			StatusCode:   upstreamResp.StatusCode,
			LatencyMS:    int(latency.Milliseconds()),
			ErrorMessage: string(upstreamBody),
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(upstreamResp.StatusCode)
		w.Write(upstreamBody)
		return
	}

	// Streaming passthrough.
	if strings.Contains(upstreamResp.Header.Get("Content-Type"), "text/event-stream") {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeOpenAIError(w, http.StatusInternalServerError, "server_error", "Streaming not supported")
			return
		}

		buf := make([]byte, 32*1024)
		for {
			n, readErr := upstreamResp.Body.Read(buf)
			if n > 0 {
				if _, writeErr := w.Write(buf[:n]); writeErr != nil {
					log.Printf("openai stream write error: %v", writeErr)
					break
				}
				flusher.Flush()
			}
			if readErr != nil {
				break
			}
		}

		latency := time.Since(start)
		h.logger.Log(&logging.LogEntry{
			KeyID:       keyID,
			Timestamp:   start,
			Method:      r.Method,
			Path:        r.URL.Path,
			Model:       partial.Model,
			InputFormat: "openai",
			StatusCode:  http.StatusOK,
			LatencyMS:   int(latency.Milliseconds()),
		})
		return
	}

	// Non-streaming passthrough: read body to extract usage for logging,
	// then write back to the client.
	upstreamBody, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, "server_error", "Failed to read upstream response")
		return
	}

	var oaiResp translate.OpenAIResponse
	var model string
	var inputTokens, outputTokens int
	if err := json.Unmarshal(upstreamBody, &oaiResp); err == nil {
		model = oaiResp.Model
		if oaiResp.Usage != nil {
			inputTokens = oaiResp.Usage.PromptTokens
			outputTokens = oaiResp.Usage.CompletionTokens
		}
	}

	latency := time.Since(start)
	cost := h.billing.CalculateCost(model, inputTokens, outputTokens)

	h.logger.Log(&logging.LogEntry{
		KeyID:        keyID,
		Timestamp:    start,
		Method:       r.Method,
		Path:         r.URL.Path,
		Model:        model,
		InputFormat:  "openai",
		StatusCode:   upstreamResp.StatusCode,
		LatencyMS:    int(latency.Milliseconds()),
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Cost:         cost,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(upstreamResp.StatusCode)
	w.Write(upstreamBody)
}
