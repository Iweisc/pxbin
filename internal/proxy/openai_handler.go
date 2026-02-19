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

	"github.com/google/uuid"
	"github.com/sertdev/pxbin/internal/auth"
	"github.com/sertdev/pxbin/internal/logging"
	"github.com/sertdev/pxbin/internal/translate"
)

type openAIResponsesStreamResult struct {
	Model              string
	InputTokens        int
	OutputTokens       int
	CacheReadTokens    int
	HasModel           bool
	HasInputTokens     bool
	HasOutputTokens    bool
	HasCacheReadTokens bool
}

// HandleOpenAIResponses translates OpenAI Responses API requests (/v1/responses)
// into Chat Completions requests, forwards them to the upstream, and translates
// the response back to Responses API format.
func (h *Handler) HandleOpenAIResponses(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	keyID := auth.GetKeyIDFromContext(r.Context())

	body, err := readBody(r)
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var responsesReq translate.ResponsesAPIRequest
	if err := json.Unmarshal(body, &responsesReq); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON in request body")
		return
	}

	model := responsesReq.Model

	upstream, err := h.resolveUpstream(r.Context(), model)
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, "server_error", "Failed to resolve upstream")
		return
	}
	upstreamID := &upstream.id

	if upstream.format == "anthropic" {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "Model is linked to an Anthropic-format upstream; use the Anthropic endpoint instead")
		return
	}

	// Translate Responses API → Chat Completions.
	chatReq, err := translate.ResponsesRequestToChatCompletions(&responsesReq)
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "Failed to translate request: "+err.Error())
		return
	}

	chatBody, err := json.Marshal(chatReq)
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, "server_error", "Failed to encode translated request")
		return
	}

	overheadUS := int(time.Since(start).Microseconds())
	upstreamResp, err := upstream.client.Do(r.Context(), "POST", "/v1/chat/completions", bytes.NewReader(chatBody), nil)
	if err != nil {
		latency := time.Since(start)
		h.logger.Log(&logging.LogEntry{
			KeyID:        keyID,
			Timestamp:    start,
			Method:       r.Method,
			Path:         r.URL.Path,
			Model:        model,
			InputFormat:  "openai",
			UpstreamID:   upstreamID,
			StatusCode:   http.StatusBadGateway,
			LatencyMS:    int(latency.Milliseconds()),
			OverheadUS:   overheadUS,
			ErrorMessage: "upstream connection error: " + err.Error(),
		})
		writeOpenAIError(w, http.StatusBadGateway, "server_error", "Failed to connect to upstream")
		return
	}
	defer upstreamResp.Body.Close()

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
			UpstreamID:   upstreamID,
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

	// Streaming: translate Chat Completions SSE → Responses API SSE.
	if responsesReq.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeOpenAIError(w, http.StatusInternalServerError, "server_error", "Streaming not supported")
			return
		}

		result, _ := translate.TranslateChatStreamToResponses(r.Context(), upstreamResp.Body, w, flusher, model)

		latency := time.Since(start)
		var inputTokens, outputTokens, cacheReadTokens int
		if result != nil {
			inputTokens = result.InputTokens
			outputTokens = result.OutputTokens
			cacheReadTokens = result.CacheReadTokens
		}
		cost := h.billing.CalculateCost(model, inputTokens, outputTokens)
		h.logger.Log(&logging.LogEntry{
			KeyID:           keyID,
			Timestamp:       start,
			Method:          r.Method,
			Path:            r.URL.Path,
			Model:           model,
			InputFormat:     "openai",
			UpstreamID:      upstreamID,
			StatusCode:      http.StatusOK,
			LatencyMS:       int(latency.Milliseconds()),
			OverheadUS:      overheadUS,
			InputTokens:     inputTokens,
			OutputTokens:    outputTokens,
			CacheReadTokens: cacheReadTokens,
			Cost:            cost,
		})
		return
	}

	// Non-streaming: translate Chat Completions response → Responses API.
	upstreamBody, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, "server_error", "Failed to read upstream response")
		return
	}

	var chatResp translate.OpenAIResponse
	if err := json.Unmarshal(upstreamBody, &chatResp); err != nil {
		writeOpenAIError(w, http.StatusBadGateway, "server_error", "Failed to parse upstream response")
		return
	}

	responsesResp := translate.ChatCompletionsToResponsesAPI(&chatResp, model)

	var inputTokens, outputTokens, cacheReadTokens int
	if chatResp.Usage != nil {
		inputTokens = chatResp.Usage.PromptTokens
		outputTokens = chatResp.Usage.CompletionTokens
		if chatResp.Usage.PromptTokensDetails != nil {
			cacheReadTokens = chatResp.Usage.PromptTokensDetails.CachedTokens
			inputTokens, cacheReadTokens = normalizeOpenAIInputAndCache(inputTokens, cacheReadTokens)
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
		UpstreamID:      upstreamID,
		StatusCode:      http.StatusOK,
		LatencyMS:       int(latency.Milliseconds()),
		OverheadUS:      overheadUS,
		InputTokens:     inputTokens,
		OutputTokens:    outputTokens,
		CacheReadTokens: cacheReadTokens,
		Cost:            cost,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	b, _ := json.Marshal(responsesResp)
	w.Write(b)
}

// passthroughOpenAIChatStream forwards OpenAI Chat Completions SSE events to
// the client while extracting usage information for logging/billing.
func passthroughOpenAIChatStream(upstream io.Reader, w http.ResponseWriter, flusher http.Flusher, fallbackModel string) openAIResponsesStreamResult {
	result := openAIResponsesStreamResult{Model: fallbackModel}

	scanner := bufio.NewScanner(upstream)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()

		if _, err := w.Write(line); err != nil {
			log.Printf("openai chat stream write error: %v", err)
			break
		}
		if _, err := w.Write([]byte("\n")); err != nil {
			log.Printf("openai chat stream write error: %v", err)
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

		// Extract usage from the stream chunk. Chat Completions uses
		// prompt_tokens / completion_tokens instead of input_tokens / output_tokens.
		var chunk translate.OpenAIStreamChunk
		if json.Unmarshal(payload, &chunk) != nil {
			continue
		}
		if chunk.Model != "" {
			result.Model = chunk.Model
			result.HasModel = true
		}
		if chunk.Usage != nil {
			result.InputTokens = chunk.Usage.PromptTokens
			result.HasInputTokens = true
			result.OutputTokens = chunk.Usage.CompletionTokens
			result.HasOutputTokens = true
			if chunk.Usage.PromptTokensDetails != nil {
				result.CacheReadTokens = chunk.Usage.PromptTokensDetails.CachedTokens
				result.HasCacheReadTokens = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("openai chat stream read error: %v", err)
	}

	flusher.Flush()
	return result
}

// HandleOpenAI proxies OpenAI-format requests to the upstream. If the
// upstream's format is "openai" the request passes through unchanged;
// "anthropic" upstreams are currently unsupported and return an error.
func (h *Handler) HandleOpenAI(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	keyID := auth.GetKeyIDFromContext(r.Context())

	defer r.Body.Close()

	limitedBody := io.LimitReader(r.Body, maxRequestBodySize+1)
	model, upstreamReqBody, err := readModelAndBuildBodyReader(limitedBody, modelProbeLimitBytes)
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "Invalid request body: missing or invalid model")
		return
	}

	// Resolve upstream based on model.
	upstream, err := h.resolveUpstream(r.Context(), model)
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, "server_error", "Failed to resolve upstream")
		return
	}
	upstreamID := &upstream.id

	if upstream.format == "anthropic" {
		// Translation path: OpenAI → Anthropic — full parse required.
		body, readErr := io.ReadAll(upstreamReqBody)
		if readErr != nil {
			writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
			return
		}
		var openaiReq translate.OpenAIRequest
		if err := json.Unmarshal(body, &openaiReq); err != nil {
			writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON in request body")
			return
		}
		h.handleOpenAIToAnthropic(w, r, upstream, &openaiReq, keyID, start)
		return
	}

	// Forward the request body to the upstream unchanged.
	overheadUS := int(time.Since(start).Microseconds())
	upstreamResp, err := upstream.client.Do(r.Context(), r.Method, "/v1/chat/completions", upstreamReqBody, nil)
	if err != nil {
		latency := time.Since(start)
		h.logger.Log(&logging.LogEntry{
			KeyID:        keyID,
			Timestamp:    start,
			Method:       r.Method,
			Path:         r.URL.Path,
			Model:        model,
			InputFormat:  "openai",
			UpstreamID:  upstreamID,
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
			UpstreamID:  upstreamID,
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

		streamResult := passthroughOpenAIChatStream(upstreamResp.Body, w, flusher, model)
		if streamResult.Model != "" {
			model = streamResult.Model
		}
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
			UpstreamID:      upstreamID,
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
		UpstreamID:      upstreamID,
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

// handleOpenAIToAnthropic translates an OpenAI request to Anthropic format,
// sends it to the upstream, and translates the response back.
func (h *Handler) handleOpenAIToAnthropic(w http.ResponseWriter, r *http.Request, upstream *upstreamInfo, openaiReq *translate.OpenAIRequest, keyID uuid.UUID, start time.Time) {
	upstreamID := &upstream.id
	anthropicReq, err := translate.OpenAIRequestToAnthropic(openaiReq)
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "Failed to translate request: "+err.Error())
		return
	}

	anthropicBody, err := json.Marshal(anthropicReq)
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, "server_error", "Failed to encode translated request")
		return
	}

	extraHeaders := http.Header{
		"X-Api-Key":         {upstream.client.apiKey},
		"Anthropic-Version": {"2023-06-01"},
	}

	overheadUS := int(time.Since(start).Microseconds())
	upstreamResp, err := upstream.client.DoRaw(r.Context(), "POST", "/v1/messages", bytes.NewReader(anthropicBody), extraHeaders)
	if err != nil {
		latency := time.Since(start)
		h.logger.Log(&logging.LogEntry{
			KeyID:        keyID,
			Timestamp:    start,
			Method:       r.Method,
			Path:         r.URL.Path,
			Model:        openaiReq.Model,
			InputFormat:  "openai",
			UpstreamID:  upstreamID,
			StatusCode:   http.StatusBadGateway,
			LatencyMS:    int(latency.Milliseconds()),
			OverheadUS:   overheadUS,
			ErrorMessage: "upstream connection error: " + err.Error(),
		})
		writeOpenAIError(w, http.StatusBadGateway, "server_error", "Failed to connect to upstream")
		return
	}
	defer upstreamResp.Body.Close()

	// Handle upstream errors: translate Anthropic error to OpenAI format.
	if upstreamResp.StatusCode >= 400 {
		upstreamBody, _ := io.ReadAll(upstreamResp.Body)
		latency := time.Since(start)
		h.logger.Log(&logging.LogEntry{
			KeyID:        keyID,
			Timestamp:    start,
			Method:       r.Method,
			Path:         r.URL.Path,
			Model:        openaiReq.Model,
			InputFormat:  "openai",
			UpstreamID:  upstreamID,
			StatusCode:   upstreamResp.StatusCode,
			LatencyMS:    int(latency.Milliseconds()),
			OverheadUS:   overheadUS,
			ErrorMessage: string(upstreamBody),
		})
		oaiErr := translate.TranslateAnthropicErrorToOpenAI(upstreamResp.StatusCode, upstreamBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(upstreamResp.StatusCode)
		w.Write(oaiErr)
		return
	}

	// Streaming translation: Anthropic SSE → OpenAI SSE.
	if openaiReq.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeOpenAIError(w, http.StatusInternalServerError, "server_error", "Streaming not supported")
			return
		}

		result, _ := translate.TranslateAnthropicStreamToOpenAI(r.Context(), upstreamResp.Body, w, flusher, openaiReq.Model)

		latency := time.Since(start)
		var inputTokens, outputTokens, cacheCreationTokens, cacheReadTokens int
		if result != nil {
			inputTokens = result.InputTokens
			outputTokens = result.OutputTokens
			cacheCreationTokens = result.CacheCreationTokens
			cacheReadTokens = result.CacheReadTokens
		}
		cost := h.billing.CalculateCost(openaiReq.Model, inputTokens, outputTokens)
		h.logger.Log(&logging.LogEntry{
			KeyID:               keyID,
			Timestamp:           start,
			Method:              r.Method,
			Path:                r.URL.Path,
			Model:               openaiReq.Model,
			InputFormat:         "openai",
			UpstreamID:          upstreamID,
			StatusCode:          http.StatusOK,
			LatencyMS:           int(latency.Milliseconds()),
			OverheadUS:          overheadUS,
			InputTokens:         inputTokens,
			OutputTokens:        outputTokens,
			CacheCreationTokens: cacheCreationTokens,
			CacheReadTokens:     cacheReadTokens,
			Cost:                cost,
		})
		return
	}

	// Non-streaming translation: read Anthropic response and translate.
	upstreamBody, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, "server_error", "Failed to read upstream response")
		return
	}

	var anthropicResp translate.AnthropicResponse
	if err := json.Unmarshal(upstreamBody, &anthropicResp); err != nil {
		writeOpenAIError(w, http.StatusBadGateway, "server_error", "Failed to parse upstream response")
		return
	}

	oaiResp := translate.AnthropicResponseToOpenAI(&anthropicResp)
	inputTokens := anthropicResp.Usage.InputTokens
	outputTokens := anthropicResp.Usage.OutputTokens
	cacheReadTokens := anthropicResp.Usage.CacheReadInputTokens

	latency := time.Since(start)
	cost := h.billing.CalculateCost(openaiReq.Model, inputTokens, outputTokens)
	h.logger.Log(&logging.LogEntry{
		KeyID:           keyID,
		Timestamp:       start,
		Method:          r.Method,
		Path:            r.URL.Path,
		Model:           openaiReq.Model,
		InputFormat:     "openai",
		UpstreamID:      upstreamID,
		StatusCode:      http.StatusOK,
		LatencyMS:       int(latency.Milliseconds()),
		OverheadUS:      overheadUS,
		InputTokens:     inputTokens,
		OutputTokens:    outputTokens,
		CacheReadTokens: cacheReadTokens,
		Cost:            cost,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	b, _ := json.Marshal(oaiResp)
	w.Write(b)
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
