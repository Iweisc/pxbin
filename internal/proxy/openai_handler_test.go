package proxy

import (
	"net/http/httptest"
	"strings"
	"testing"
)

type recorderFlusher struct {
	*httptest.ResponseRecorder
}

func (r *recorderFlusher) Flush() {
	r.ResponseRecorder.Flush()
}

func TestExtractOpenAIResponsesUsageTopLevel(t *testing.T) {
	payload := []byte(`{
		"model": "gpt-5.3-codex",
		"usage": {
			"input_tokens": 128,
			"output_tokens": 64,
			"input_tokens_details": {
				"cached_tokens": 96
			}
		}
	}`)

	got := extractOpenAIResponsesUsage(payload)

	if !got.HasModel || got.Model != "gpt-5.3-codex" {
		t.Fatalf("expected model gpt-5.3-codex, got %+v", got)
	}
	if !got.HasInputTokens || got.InputTokens != 128 {
		t.Fatalf("expected input_tokens=128, got %+v", got)
	}
	if !got.HasOutputTokens || got.OutputTokens != 64 {
		t.Fatalf("expected output_tokens=64, got %+v", got)
	}
	if !got.HasCachedTokens || got.CachedTokens != 96 {
		t.Fatalf("expected cached_tokens=96, got %+v", got)
	}
}

func TestExtractOpenAIResponsesUsageNestedCompletedEvent(t *testing.T) {
	payload := []byte(`{
		"type": "response.completed",
		"response": {
			"model": "gpt-5.3-codex",
			"usage": {
				"input_tokens": 222,
				"output_tokens": 111,
				"input_tokens_details": {
					"cached_tokens": 200
				}
			}
		}
	}`)

	got := extractOpenAIResponsesUsage(payload)

	if !got.HasModel || got.Model != "gpt-5.3-codex" {
		t.Fatalf("expected model gpt-5.3-codex, got %+v", got)
	}
	if !got.HasInputTokens || got.InputTokens != 222 {
		t.Fatalf("expected input_tokens=222, got %+v", got)
	}
	if !got.HasOutputTokens || got.OutputTokens != 111 {
		t.Fatalf("expected output_tokens=111, got %+v", got)
	}
	if !got.HasCachedTokens || got.CachedTokens != 200 {
		t.Fatalf("expected cached_tokens=200, got %+v", got)
	}
}

func TestExtractOpenAIResponsesUsageChatFallbackFields(t *testing.T) {
	payload := []byte(`{
		"usage": {
			"prompt_tokens": 19,
			"completion_tokens": 7,
			"prompt_tokens_details": {
				"cached_tokens": 17
			}
		}
	}`)

	got := extractOpenAIResponsesUsage(payload)

	if !got.HasInputTokens || got.InputTokens != 19 {
		t.Fatalf("expected prompt_tokens fallback for input, got %+v", got)
	}
	if !got.HasOutputTokens || got.OutputTokens != 7 {
		t.Fatalf("expected completion_tokens fallback for output, got %+v", got)
	}
	if !got.HasCachedTokens || got.CachedTokens != 17 {
		t.Fatalf("expected cached_tokens fallback from prompt_tokens_details, got %+v", got)
	}
}

func TestPassthroughOpenAIResponsesStreamCapturesUsage(t *testing.T) {
	upstream := strings.NewReader(strings.Join([]string{
		"event: response.created",
		`data: {"type":"response.created","response":{"model":"gpt-5.3-codex"}}`,
		"",
		"event: response.output_text.delta",
		`data: {"type":"response.output_text.delta","delta":"Hello"}`,
		"",
		"event: response.completed",
		`data: {"type":"response.completed","response":{"model":"gpt-5.3-codex","usage":{"input_tokens":345,"output_tokens":210,"input_tokens_details":{"cached_tokens":300}}}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n"))

	rec := httptest.NewRecorder()
	flusher := &recorderFlusher{ResponseRecorder: rec}

	result := passthroughOpenAIResponsesStream(upstream, rec, flusher, "fallback-model")

	if result.Model != "gpt-5.3-codex" {
		t.Fatalf("expected model gpt-5.3-codex, got %q", result.Model)
	}
	if result.InputTokens != 345 {
		t.Fatalf("expected input_tokens=345, got %d", result.InputTokens)
	}
	if result.OutputTokens != 210 {
		t.Fatalf("expected output_tokens=210, got %d", result.OutputTokens)
	}
	if !result.HasCacheReadTokens || result.CacheReadTokens != 300 {
		t.Fatalf("expected cached_tokens=300, got %+v", result)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"response.completed"`) {
		t.Fatalf("expected stream to include completed event, got: %s", body)
	}
}

func TestNormalizeOpenAIInputAndCache(t *testing.T) {
	input, cache := normalizeOpenAIInputAndCache(1000, 980)
	if input != 20 || cache != 980 {
		t.Fatalf("expected normalized tokens (20,980), got (%d,%d)", input, cache)
	}

	input, cache = normalizeOpenAIInputAndCache(5, 9)
	if input != 0 || cache != 5 {
		t.Fatalf("expected clamped tokens (0,5), got (%d,%d)", input, cache)
	}
}

func TestResponsesUpstreamPath(t *testing.T) {
	if got := responsesUpstreamPath("/v1/responses"); got != "/v1/responses" {
		t.Fatalf("expected /v1/responses, got %q", got)
	}
	if got := responsesUpstreamPath("/v1/responses/compact"); got != "/v1/responses/compact" {
		t.Fatalf("expected /v1/responses/compact passthrough, got %q", got)
	}
	if got := responsesUpstreamPath("/v1/responses/something/else"); got != "/v1/responses/something/else" {
		t.Fatalf("expected nested responses path passthrough, got %q", got)
	}
	if got := responsesUpstreamPath("/v1/chat/completions"); got != "/v1/responses" {
		t.Fatalf("expected fallback /v1/responses for non-responses path, got %q", got)
	}
}
