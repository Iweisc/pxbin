package proxy

import (
	"bufio"
	"bytes"
	"context"
	stdjson "encoding/json"
	"fmt"
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

// upstreamInfo contains the resolved upstream client and metadata.
type upstreamInfo struct {
	client *UpstreamClient
	format string
	id     uuid.UUID
}

// resolveUpstream looks up the model's linked upstream from the DB. If found,
// it returns a cached UpstreamClient and the upstream's format. If the model
// has no linked upstream, it returns an error — all upstreams must be
// configured via the management API.
func (h *Handler) resolveUpstream(ctx context.Context, modelName string) (*upstreamInfo, error) {
	mw, err := h.modelCache.GetModelWithUpstream(ctx, modelName)
	if err != nil {
		return nil, fmt.Errorf("resolve upstream: %w", err)
	}
	if mw == nil {
		return nil, fmt.Errorf("no upstream configured for model %q", modelName)
	}
	client := h.clients.Get(*mw.UpstreamID, mw.UpstreamBaseURL, mw.UpstreamAPIKey)
	return &upstreamInfo{
		client: client,
		format: mw.UpstreamFormat,
		id:     *mw.UpstreamID,
	}, nil
}

// HandleAnthropic proxies Anthropic /v1/messages requests. Depending on the
// upstream format, it either passes through natively or translates to OpenAI.
func (h *Handler) HandleAnthropic(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	keyID := auth.GetKeyIDFromContext(r.Context())

	// Read the request body. Pre-allocates when Content-Length is known.
	body, err := readBody(r)
	if err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	defer r.Body.Close()

	// Lazy-extract only model and stream — avoids full parse of large payloads
	// (100KB+ system prompts, tools, conversation history).
	model, stream, err := extractModelAndStream(body)
	if err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON in request body")
		return
	}

	// Resolve which upstream to use based on the model.
	upstream, err := h.resolveUpstream(r.Context(), model)
	if err != nil {
		writeAnthropicError(w, http.StatusInternalServerError, "api_error", "Failed to resolve upstream")
		return
	}

	if upstream.format == "openai" {
		// Translation path — full parse required.
		var anthropicReq translate.AnthropicRequest
		if err := json.Unmarshal(body, &anthropicReq); err != nil {
			writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON in request body")
			return
		}
		h.handleAnthropicToOpenAI(w, r, upstream, body, &anthropicReq, keyID, start)
	} else {
		// Native passthrough — no full parse needed.
		h.handleAnthropicNative(w, r, upstream, body, model, stream, keyID, start)
	}
}

// extractModelAndStream uses sonic's lazy parser to pull out just "model" and
// "stream" from the request JSON without deserializing the full body.
func extractModelAndStream(body []byte) (string, bool, error) {
	modelNode, err := json.Get(body, "model")
	if err != nil {
		return "", false, err
	}
	model, err := modelNode.String()
	if err != nil {
		return "", false, err
	}
	streamNode, _ := json.Get(body, "stream")
	stream, _ := streamNode.Bool()
	return model, stream, nil
}

// handleAnthropicNative passes the request through to an Anthropic-format
// upstream using x-api-key auth.
func (h *Handler) handleAnthropicNative(w http.ResponseWriter, r *http.Request, upstream *upstreamInfo, body []byte, model string, stream bool, keyID uuid.UUID, start time.Time) {
	upstreamID := &upstream.id
	extraHeaders := http.Header{
		"X-Api-Key":         {upstream.client.apiKey},
		"Anthropic-Version": {"2023-06-01"},
	}
	// Strip unsupported fields (e.g. cache_control.scope) that some
	// upstreams reject. Cheap no-op when the field isn't present.
	body = sanitizeAnthropicBody(body)
	// Strip empty text content blocks that some clients (e.g. Claude Code)
	// include. Anthropic's API rejects text blocks with empty/whitespace text.
	body = stripEmptyTextBlocks(body)
	// Strip thinking blocks from conversation history. Thinking blocks
	// contain cryptographic signatures that are only valid from the
	// originating API — blocks synthesized during protocol translation
	// have no valid signature and cause upstream validation errors.
	// Anthropic re-derives thinking from context, so stripping is safe.
	body = stripThinkingBlocks(body)
	overheadUS := int(time.Since(start).Microseconds())
	upstreamResp, err := upstream.client.DoRaw(r.Context(), "POST", "/v1/messages", bytes.NewReader(body), extraHeaders)
	if err != nil {
		latency := time.Since(start)
		h.logger.Log(&logging.LogEntry{
			KeyID:        keyID,
			Timestamp:    start,
			Method:       r.Method,
			Path:         r.URL.Path,
			Model:        model,
			InputFormat:  "anthropic",
			UpstreamID:   upstreamID,
			StatusCode:   http.StatusBadGateway,
			LatencyMS:    int(latency.Milliseconds()),
			OverheadUS:   overheadUS,
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
			Model:        model,
			InputFormat:  "anthropic",
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

	// Streaming response — passthrough SSE and capture usage.
	if stream {
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
		cost := h.billing.CalculateCost(model, result.InputTokens, result.OutputTokens)
		h.logger.Log(&logging.LogEntry{
			KeyID:               keyID,
			Timestamp:           start,
			Method:              r.Method,
			Path:                r.URL.Path,
			Model:               model,
			InputFormat:         "anthropic",
			UpstreamID:          upstreamID,
			StatusCode:          http.StatusOK,
			LatencyMS:           int(latency.Milliseconds()),
			OverheadUS:          overheadUS,
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
		cost := h.billing.CalculateCost(model, inputTokens, outputTokens)
		h.logger.Log(&logging.LogEntry{
			KeyID:               keyID,
			Timestamp:           start,
			Method:              r.Method,
			Path:                r.URL.Path,
			Model:               model,
			InputFormat:         "anthropic",
			UpstreamID:          upstreamID,
			StatusCode:          http.StatusOK,
			LatencyMS:           int(latency.Milliseconds()),
			OverheadUS:          overheadUS,
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
func (h *Handler) handleAnthropicToOpenAI(w http.ResponseWriter, r *http.Request, upstream *upstreamInfo, body []byte, anthropicReq *translate.AnthropicRequest, keyID uuid.UUID, start time.Time) {
	upstreamID := &upstream.id
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

	overheadUS := int(time.Since(start).Microseconds())
	upstreamResp, err := upstream.client.Do(r.Context(), "POST", "/v1/chat/completions", bytes.NewReader(openaiBody), nil)
	if err != nil {
		latency := time.Since(start)
		h.logger.Log(&logging.LogEntry{
			KeyID:        keyID,
			Timestamp:    start,
			Method:       r.Method,
			Path:         r.URL.Path,
			Model:        anthropicReq.Model,
			InputFormat:  "anthropic",
			UpstreamID:   upstreamID,
			StatusCode:   http.StatusBadGateway,
			LatencyMS:    int(latency.Milliseconds()),
			OverheadUS:   overheadUS,
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
			UpstreamID:   upstreamID,
			StatusCode:   upstreamResp.StatusCode,
			LatencyMS:    int(latency.Milliseconds()),
			OverheadUS:   overheadUS,
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
		cacheCreationTokens := 0
		cacheReadTokens := 0
		if result != nil {
			inputTokens = result.InputTokens
			outputTokens = result.OutputTokens
			cacheCreationTokens = result.CacheCreationTokens
			cacheReadTokens = result.CacheReadTokens
		}
		cost := h.billing.CalculateCost(anthropicReq.Model, inputTokens, outputTokens)
		h.logger.Log(&logging.LogEntry{
			KeyID:               keyID,
			Timestamp:           start,
			Method:              r.Method,
			Path:                r.URL.Path,
			Model:               anthropicReq.Model,
			InputFormat:         "anthropic",
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
	cacheReadTokens := 0
	if oaiResp.Usage != nil {
		inputTokens = oaiResp.Usage.PromptTokens
		outputTokens = oaiResp.Usage.CompletionTokens
		if oaiResp.Usage.PromptTokensDetails != nil {
			cacheReadTokens = oaiResp.Usage.PromptTokensDetails.CachedTokens
			inputTokens, cacheReadTokens = normalizeOpenAIInputAndCache(inputTokens, cacheReadTokens)
		}
	}

	latency := time.Since(start)
	cost := h.billing.CalculateCost(anthropicReq.Model, inputTokens, outputTokens)
	h.logger.Log(&logging.LogEntry{
		KeyID:           keyID,
		Timestamp:       start,
		Method:          r.Method,
		Path:            r.URL.Path,
		Model:           anthropicReq.Model,
		InputFormat:     "anthropic",
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
	b, _ := json.Marshal(anthropicResp)
	w.Write(b)
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
		if _, err := w.Write(line); err != nil {
			log.Printf("anthropic stream write error: %v", err)
			break
		}
		if _, err := w.Write(newline); err != nil {
			log.Printf("anthropic stream write error: %v", err)
			break
		}

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

	if err := scanner.Err(); err != nil {
		log.Printf("anthropic stream read error: %v", err)
	}

	flusher.Flush()
	return usage
}

// sanitizeAnthropicBody strips fields from cache_control objects that some
// upstreams don't support (e.g. the "scope" field). Returns the body unchanged
// when no "scope" is present — the bytes.Contains check makes this a no-op
// for the vast majority of requests.
func sanitizeAnthropicBody(body []byte) []byte {
	if !bytes.Contains(body, []byte(`"scope"`)) {
		return body
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return body
	}
	stripCacheControlScope(raw)
	cleaned, err := json.Marshal(raw)
	if err != nil {
		return body
	}
	return cleaned
}

// stripCacheControlScope recursively removes the "scope" key from any
// cache_control object found in the JSON tree.
func stripCacheControlScope(v interface{}) {
	switch val := v.(type) {
	case map[string]interface{}:
		if cc, ok := val["cache_control"]; ok {
			if ccMap, ok := cc.(map[string]interface{}); ok {
				delete(ccMap, "scope")
			}
		}
		for _, child := range val {
			stripCacheControlScope(child)
		}
	case []interface{}:
		for _, item := range val {
			stripCacheControlScope(item)
		}
	}
}

// stripEmptyTextBlocks removes text content blocks with empty or whitespace-only
// text from messages. Some clients (e.g. Claude Code) send empty text blocks
// that Anthropic's API rejects with a ValidationException. Returns the body
// unchanged when no "text" blocks are present — the bytes.Contains check makes
// this a no-op for the vast majority of requests.
func stripEmptyTextBlocks(body []byte) []byte {
	if !bytes.Contains(body, []byte(`"text"`)) {
		return body
	}

	var raw map[string]stdjson.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return body
	}
	messagesRaw, ok := raw["messages"]
	if !ok {
		return body
	}

	var messages []stdjson.RawMessage
	if err := json.Unmarshal(messagesRaw, &messages); err != nil {
		return body
	}

	modified := false
	for i, msgRaw := range messages {
		var msg struct {
			Content stdjson.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(msgRaw, &msg); err != nil {
			continue
		}
		// Content may be a string (no blocks to filter) or an array.
		if len(msg.Content) == 0 || msg.Content[0] != '[' {
			continue
		}

		var blocks []stdjson.RawMessage
		if err := json.Unmarshal(msg.Content, &blocks); err != nil {
			continue
		}

		filtered := make([]stdjson.RawMessage, 0, len(blocks))
		for _, blockRaw := range blocks {
			var peek struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if json.Unmarshal(blockRaw, &peek) == nil &&
				peek.Type == "text" && strings.TrimSpace(peek.Text) == "" {
				modified = true
				continue
			}
			filtered = append(filtered, blockRaw)
		}

		if len(filtered) == len(blocks) {
			continue
		}

		// Re-assemble the message with filtered content.
		var msgMap map[string]stdjson.RawMessage
		if err := json.Unmarshal(msgRaw, &msgMap); err != nil {
			continue
		}
		newContent, err := json.Marshal(filtered)
		if err != nil {
			continue
		}
		msgMap["content"] = stdjson.RawMessage(newContent)
		rebuilt, err := json.Marshal(msgMap)
		if err != nil {
			continue
		}
		messages[i] = stdjson.RawMessage(rebuilt)
	}

	if !modified {
		return body
	}

	newMessages, err := json.Marshal(messages)
	if err != nil {
		return body
	}
	raw["messages"] = stdjson.RawMessage(newMessages)
	cleaned, err := json.Marshal(raw)
	if err != nil {
		return body
	}
	return cleaned
}

// stripThinkingBlocks removes thinking and redacted_thinking content blocks
// from assistant messages in an Anthropic request body. Thinking blocks carry
// cryptographic signatures issued by the originating API; blocks synthesized
// during protocol translation (e.g. from OpenAI reasoning_content) have no
// valid signature and are rejected by upstream Anthropic APIs. Stripping is
// safe — the API re-derives thinking from context when blocks are absent.
func stripThinkingBlocks(body []byte) []byte {
	if !bytes.Contains(body, []byte(`"thinking"`)) {
		return body
	}

	var raw map[string]stdjson.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return body
	}
	messagesRaw, ok := raw["messages"]
	if !ok {
		return body
	}

	var messages []stdjson.RawMessage
	if err := json.Unmarshal(messagesRaw, &messages); err != nil {
		return body
	}

	modified := false
	for i, msgRaw := range messages {
		var msg struct {
			Role    string              `json:"role"`
			Content stdjson.RawMessage  `json:"content"`
		}
		if err := json.Unmarshal(msgRaw, &msg); err != nil || msg.Role != "assistant" {
			continue
		}

		// Content may be a string (no thinking blocks possible) or an array.
		if len(msg.Content) == 0 || msg.Content[0] != '[' {
			continue
		}

		var blocks []stdjson.RawMessage
		if err := json.Unmarshal(msg.Content, &blocks); err != nil {
			continue
		}

		filtered := make([]stdjson.RawMessage, 0, len(blocks))
		for _, blockRaw := range blocks {
			var peek struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(blockRaw, &peek) == nil &&
				(peek.Type == "thinking" || peek.Type == "redacted_thinking") {
				modified = true
				continue
			}
			filtered = append(filtered, blockRaw)
		}

		if len(filtered) == len(blocks) {
			continue
		}

		// Re-assemble the message with filtered content, preserving all
		// other fields (tool calls, etc.) by patching into the raw map.
		var msgMap map[string]stdjson.RawMessage
		if err := json.Unmarshal(msgRaw, &msgMap); err != nil {
			continue
		}
		newContent, err := json.Marshal(filtered)
		if err != nil {
			continue
		}
		msgMap["content"] = stdjson.RawMessage(newContent)
		rebuilt, err := json.Marshal(msgMap)
		if err != nil {
			continue
		}
		messages[i] = stdjson.RawMessage(rebuilt)
	}

	if !modified {
		return body
	}

	newMessages, err := json.Marshal(messages)
	if err != nil {
		return body
	}
	raw["messages"] = stdjson.RawMessage(newMessages)
	cleaned, err := json.Marshal(raw)
	if err != nil {
		return body
	}
	return cleaned
}
