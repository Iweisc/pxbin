package proxy

import (
	"bufio"
	"bytes"
	json "github.com/bytedance/sonic"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sertdev/pxbin/internal/auth"
	"github.com/sertdev/pxbin/internal/logging"
	"github.com/sertdev/pxbin/internal/translate"
)

type openAIResponsesUsage struct {
	Model           string
	InputTokens     int
	OutputTokens    int
	CachedTokens    int
	HasModel        bool
	HasInputTokens  bool
	HasOutputTokens bool
	HasCachedTokens bool
}

type openAIResponsesStreamResult struct {
	Model              string
	InputTokens        int
	OutputTokens       int
	CacheReadTokens    int
	HasInputTokens     bool
	HasOutputTokens    bool
	HasCacheReadTokens bool
}

// HandleOpenAIResponses proxies OpenAI Responses API requests (/v1/responses)
// to the upstream unchanged. Usage extraction uses the Responses API field
// names (input_tokens / output_tokens) instead of the Chat Completions names.
func (h *Handler) HandleOpenAIResponses(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	keyID := auth.GetKeyIDFromContext(r.Context())

	body, err := readBody(r)
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	defer r.Body.Close()

	modelNode, _ := json.Get(body, "model")
	model, _ := modelNode.String()

	client, format, err := h.resolveUpstream(r.Context(), model)
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, "server_error", "Failed to resolve upstream")
		return
	}

	if format == "anthropic" {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "Model is linked to an Anthropic-format upstream; use the Anthropic endpoint instead")
		return
	}

	upstreamPath := responsesUpstreamPath(r.URL.Path)
	overheadUS := int(time.Since(start).Microseconds())
	upstreamResp, err := client.Do(r.Context(), r.Method, upstreamPath, bytes.NewReader(body), nil)
	if err != nil {
		latency := time.Since(start)
		h.logger.Log(&logging.LogEntry{
			KeyID:        keyID,
			Timestamp:    start,
			Method:       r.Method,
			Path:         r.URL.Path,
			Model:        model,
			InputFormat:  "openai",
			StatusCode:   http.StatusBadGateway,
			LatencyMS:    int(latency.Milliseconds()),
			OverheadUS:   overheadUS,
			ErrorMessage: "upstream connection error: " + err.Error(),
		})
		writeOpenAIError(w, http.StatusBadGateway, "server_error", "Failed to connect to upstream")
		return
	}
	defer upstreamResp.Body.Close()

	for _, hdr := range []string{"Content-Type", "X-Request-Id"} {
		if v := upstreamResp.Header.Get(hdr); v != "" {
			w.Header().Set(hdr, v)
		}
	}

	if upstreamResp.StatusCode >= 400 {
		upstreamBody, _ := io.ReadAll(upstreamResp.Body)
		latency := time.Since(start)

		h.logger.Log(&logging.LogEntry{
			KeyID:        keyID,
			Timestamp:    start,
			Method:       r.Method,
			Path:         r.URL.Path,
			Model:        model,
			InputFormat:  "openai",
			StatusCode:   upstreamResp.StatusCode,
			LatencyMS:    int(latency.Milliseconds()),
			OverheadUS:   overheadUS,
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

		streamResult := passthroughOpenAIResponsesStream(upstreamResp.Body, w, flusher, model)
		model = streamResult.Model
		inputTokens := streamResult.InputTokens
		cacheReadTokens := streamResult.CacheReadTokens
		if streamResult.HasInputTokens && streamResult.HasCacheReadTokens {
			inputTokens, cacheReadTokens = normalizeOpenAIInputAndCache(inputTokens, cacheReadTokens)
		}

		latency := time.Since(start)
		cost := h.billing.CalculateCost(model, inputTokens, streamResult.OutputTokens)
		h.logger.Log(&logging.LogEntry{
			KeyID:           keyID,
			Timestamp:       start,
			Method:          r.Method,
			Path:            r.URL.Path,
			Model:           model,
			InputFormat:     "openai",
			StatusCode:      http.StatusOK,
			LatencyMS:       int(latency.Milliseconds()),
			OverheadUS:      overheadUS,
			InputTokens:     inputTokens,
			OutputTokens:    streamResult.OutputTokens,
			CacheReadTokens: cacheReadTokens,
			Cost:            cost,
		})
		return
	}

	// Non-streaming: extract usage with Responses API field names.
	upstreamBody, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, "server_error", "Failed to read upstream response")
		return
	}

	var inputTokens, outputTokens, cacheReadTokens int
	usage := extractOpenAIResponsesUsage(upstreamBody)
	if usage.HasModel && usage.Model != "" {
		model = usage.Model
	}
	if usage.HasInputTokens {
		inputTokens = usage.InputTokens
	}
	if usage.HasOutputTokens {
		outputTokens = usage.OutputTokens
	}
	if usage.HasCachedTokens {
		cacheReadTokens = usage.CachedTokens
	}
	if usage.HasInputTokens && usage.HasCachedTokens {
		inputTokens, cacheReadTokens = normalizeOpenAIInputAndCache(inputTokens, cacheReadTokens)
	}

	latency := time.Since(start)
	cost := h.billing.CalculateCost(model, inputTokens, outputTokens)

	h.logger.Log(&logging.LogEntry{
		KeyID:           keyID,
		Timestamp:       start,
		Method:          r.Method,
		Path:            r.URL.Path,
		Model:           model,
		InputFormat:     "openai",
		StatusCode:      upstreamResp.StatusCode,
		LatencyMS:       int(latency.Milliseconds()),
		OverheadUS:      overheadUS,
		InputTokens:     inputTokens,
		OutputTokens:    outputTokens,
		CacheReadTokens: cacheReadTokens,
		Cost:            cost,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(upstreamResp.StatusCode)
	w.Write(upstreamBody)
}

func responsesUpstreamPath(requestPath string) string {
	const basePath = "/v1/responses"
	if strings.HasPrefix(requestPath, basePath+"/") {
		return requestPath
	}
	return basePath
}

func passthroughOpenAIResponsesStream(upstream io.Reader, w http.ResponseWriter, flusher http.Flusher, fallbackModel string) openAIResponsesStreamResult {
	result := openAIResponsesStreamResult{Model: fallbackModel}

	scanner := bufio.NewScanner(upstream)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()

		if _, err := w.Write(line); err != nil {
			log.Printf("openai responses stream write error: %v", err)
			break
		}
		if _, err := w.Write([]byte("\n")); err != nil {
			log.Printf("openai responses stream write error: %v", err)
			break
		}

		if len(line) == 0 {
			flusher.Flush()
			continue
		}

		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		payload := bytes.TrimSpace(line[len("data:"):])
		if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) {
			continue
		}

		usage := extractOpenAIResponsesUsage(payload)
		if usage.HasModel && usage.Model != "" {
			result.Model = usage.Model
		}
		if usage.HasInputTokens {
			result.InputTokens = usage.InputTokens
			result.HasInputTokens = true
		}
		if usage.HasOutputTokens {
			result.OutputTokens = usage.OutputTokens
			result.HasOutputTokens = true
		}
		if usage.HasCachedTokens {
			result.CacheReadTokens = usage.CachedTokens
			result.HasCacheReadTokens = true
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("openai responses stream read error: %v", err)
	}

	flusher.Flush()
	return result
}

func extractOpenAIResponsesUsage(payload []byte) openAIResponsesUsage {
	var result openAIResponsesUsage

	if model, ok := getJSONFieldString(payload, []string{"model"}, []string{"response", "model"}); ok {
		result.Model = model
		result.HasModel = true
	}
	if inputTokens, ok := getJSONFieldInt(payload,
		[]string{"usage", "input_tokens"},
		[]string{"response", "usage", "input_tokens"},
		[]string{"usage", "prompt_tokens"},
		[]string{"response", "usage", "prompt_tokens"},
	); ok {
		result.InputTokens = inputTokens
		result.HasInputTokens = true
	}
	if outputTokens, ok := getJSONFieldInt(payload,
		[]string{"usage", "output_tokens"},
		[]string{"response", "usage", "output_tokens"},
		[]string{"usage", "completion_tokens"},
		[]string{"response", "usage", "completion_tokens"},
	); ok {
		result.OutputTokens = outputTokens
		result.HasOutputTokens = true
	}
	if cachedTokens, ok := getJSONFieldInt(payload,
		[]string{"usage", "input_tokens_details", "cached_tokens"},
		[]string{"response", "usage", "input_tokens_details", "cached_tokens"},
		[]string{"usage", "prompt_tokens_details", "cached_tokens"},
		[]string{"response", "usage", "prompt_tokens_details", "cached_tokens"},
	); ok {
		result.CachedTokens = cachedTokens
		result.HasCachedTokens = true
	}

	return result
}

func getJSONFieldInt(payload []byte, paths ...[]string) (int, bool) {
	for _, path := range paths {
		node, err := json.Get(payload, toJSONPath(path)...)
		if err != nil {
			continue
		}
		v, err := node.Int64()
		if err != nil {
			continue
		}
		return int(v), true
	}
	return 0, false
}

func getJSONFieldString(payload []byte, paths ...[]string) (string, bool) {
	for _, path := range paths {
		node, err := json.Get(payload, toJSONPath(path)...)
		if err != nil {
			continue
		}
		v, err := node.String()
		if err != nil || v == "" {
			continue
		}
		return v, true
	}
	return "", false
}

func toJSONPath(path []string) []interface{} {
	parts := make([]interface{}, len(path))
	for i, p := range path {
		parts[i] = p
	}
	return parts
}

// HandleOpenAI proxies OpenAI-format requests to the upstream. If the
// upstream's format is "openai" the request passes through unchanged;
// "anthropic" upstreams are currently unsupported and return an error.
func (h *Handler) HandleOpenAI(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	keyID := auth.GetKeyIDFromContext(r.Context())

	// Read the request body. Pre-allocates when Content-Length is known.
	body, err := readBody(r)
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	defer r.Body.Close()

	modelNode, _ := json.Get(body, "model")
	model, _ := modelNode.String()

	// Resolve upstream based on model.
	client, format, err := h.resolveUpstream(r.Context(), model)
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, "server_error", "Failed to resolve upstream")
		return
	}

	if format == "anthropic" {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "Model is linked to an Anthropic-format upstream; use the Anthropic endpoint instead")
		return
	}

	// Forward the request body to the upstream unchanged.
	overheadUS := int(time.Since(start).Microseconds())
	upstreamResp, err := client.Do(r.Context(), r.Method, "/v1/chat/completions", bytes.NewReader(body), nil)
	if err != nil {
		latency := time.Since(start)
		h.logger.Log(&logging.LogEntry{
			KeyID:        keyID,
			Timestamp:    start,
			Method:       r.Method,
			Path:         r.URL.Path,
			Model:        model,
			InputFormat:  "openai",
			StatusCode:   http.StatusBadGateway,
			LatencyMS:    int(latency.Milliseconds()),
			OverheadUS:   overheadUS,
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
			Model:        model,
			InputFormat:  "openai",
			StatusCode:   upstreamResp.StatusCode,
			LatencyMS:    int(latency.Milliseconds()),
			OverheadUS:   overheadUS,
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
			Model:       model,
			InputFormat: "openai",
			StatusCode:  http.StatusOK,
			LatencyMS:   int(latency.Milliseconds()),
			OverheadUS:  overheadUS,
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
	var inputTokens, outputTokens, cacheReadTokens int
	if err := json.Unmarshal(upstreamBody, &oaiResp); err == nil {
		if oaiResp.Model != "" {
			model = oaiResp.Model
		}
		if oaiResp.Usage != nil {
			inputTokens = oaiResp.Usage.PromptTokens
			outputTokens = oaiResp.Usage.CompletionTokens
			if oaiResp.Usage.PromptTokensDetails != nil {
				cacheReadTokens = oaiResp.Usage.PromptTokensDetails.CachedTokens
				inputTokens, cacheReadTokens = normalizeOpenAIInputAndCache(inputTokens, cacheReadTokens)
			}
		}
	}

	latency := time.Since(start)
	cost := h.billing.CalculateCost(model, inputTokens, outputTokens)

	h.logger.Log(&logging.LogEntry{
		KeyID:           keyID,
		Timestamp:       start,
		Method:          r.Method,
		Path:            r.URL.Path,
		Model:           model,
		InputFormat:     "openai",
		StatusCode:      upstreamResp.StatusCode,
		LatencyMS:       int(latency.Milliseconds()),
		OverheadUS:      overheadUS,
		InputTokens:     inputTokens,
		OutputTokens:    outputTokens,
		CacheReadTokens: cacheReadTokens,
		Cost:            cost,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(upstreamResp.StatusCode)
	w.Write(upstreamBody)
}

func normalizeOpenAIInputAndCache(totalInputTokens, cacheReadTokens int) (int, int) {
	if totalInputTokens < 0 {
		totalInputTokens = 0
	}
	if cacheReadTokens < 0 {
		cacheReadTokens = 0
	}
	if cacheReadTokens > totalInputTokens {
		cacheReadTokens = totalInputTokens
	}
	return totalInputTokens - cacheReadTokens, cacheReadTokens
}
