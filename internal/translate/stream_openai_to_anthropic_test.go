package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- helpers ----------------------------------------------------------------

// mockFlusher wraps httptest.ResponseRecorder and satisfies http.Flusher.
type mockFlusher struct {
	*httptest.ResponseRecorder
}

func (m *mockFlusher) Flush() {
	m.ResponseRecorder.Flush()
}

// sseLines builds a raw SSE byte stream from a list of OpenAI stream chunks.
// Each chunk is wrapped in "data: ...\n\n". A final "data: [DONE]\n\n" is
// appended.
func sseLines(chunks ...OpenAIStreamChunk) io.ReadCloser {
	var buf bytes.Buffer
	for _, c := range chunks {
		b, _ := json.Marshal(c)
		buf.WriteString("data: ")
		buf.Write(b)
		buf.WriteString("\n\n")
	}
	buf.WriteString("data: [DONE]\n\n")
	return io.NopCloser(&buf)
}

// sseRaw builds a raw SSE stream from raw string lines.
func sseRaw(lines ...string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(strings.Join(lines, "\n")))
}

// ptr returns a pointer to v.
func ptr[T any](v T) *T { return &v }

// parseSSEEvents splits a recorded response body into (eventType, data) pairs.
func parseSSEEvents(body string) []sseEvent {
	var events []sseEvent
	lines := strings.Split(body, "\n")
	var eventType, data string
	for _, line := range lines {
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data = strings.TrimPrefix(line, "data: ")
		} else if line == "" && eventType != "" {
			events = append(events, sseEvent{Type: eventType, Data: data})
			eventType = ""
			data = ""
		}
	}
	return events
}

type sseEvent struct {
	Type string
	Data string
}

func runStream(t *testing.T, body io.ReadCloser) ([]sseEvent, *StreamResult, error) {
	t.Helper()
	rec := httptest.NewRecorder()
	flusher := &mockFlusher{rec}
	result, err := TranslateOpenAIStreamToAnthropic(context.Background(), body, rec, flusher, "claude-opus-4-6")
	events := parseSSEEvents(rec.Body.String())
	return events, result, err
}

// --- tests ------------------------------------------------------------------

func TestSimpleTextStream(t *testing.T) {
	body := sseLines(
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{Role: "assistant"},
			}},
		},
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{Content: ptr("Hello")},
			}},
		},
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{Content: ptr(" world")},
			}},
		},
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index:        0,
				Delta:        OpenAIStreamDelta{},
				FinishReason: ptr("stop"),
			}},
		},
	)

	events, _, err := runStream(t, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEventTypes(t, events, []string{
		"message_start",
		"ping",
		"content_block_start",
		"content_block_delta",
		"content_block_delta",
		"content_block_stop",
		"message_delta",
		"message_stop",
	})

	// Verify message_start has correct structure.
	var msgStart MessageStartEvent
	mustUnmarshal(t, events[0].Data, &msgStart)
	if msgStart.Message.Role != "assistant" {
		t.Errorf("expected role assistant, got %q", msgStart.Message.Role)
	}
	if msgStart.Message.Model != "claude-opus-4-6" {
		t.Errorf("expected model claude-opus-4-6, got %q", msgStart.Message.Model)
	}
	if !strings.HasPrefix(msgStart.Message.ID, "msg_") {
		t.Errorf("expected message ID starting with msg_, got %q", msgStart.Message.ID)
	}

	// Verify text deltas.
	var delta1 ContentBlockDeltaEvent
	mustUnmarshal(t, events[3].Data, &delta1)
	if delta1.Delta.Text != "Hello" {
		t.Errorf("expected 'Hello', got %q", delta1.Delta.Text)
	}

	var delta2 ContentBlockDeltaEvent
	mustUnmarshal(t, events[4].Data, &delta2)
	if delta2.Delta.Text != " world" {
		t.Errorf("expected ' world', got %q", delta2.Delta.Text)
	}

	// Verify stop reason.
	var msgDelta MessageDeltaEvent
	mustUnmarshal(t, events[6].Data, &msgDelta)
	if msgDelta.Delta.StopReason == nil || *msgDelta.Delta.StopReason != "end_turn" {
		t.Errorf("expected stop_reason 'end_turn', got %v", msgDelta.Delta.StopReason)
	}
}

func TestToolCallStream(t *testing.T) {
	body := sseLines(
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{Role: "assistant"},
			}},
		},
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{
					ToolCalls: []OpenAIStreamToolCall{{
						Index: 0,
						ID:    "call_abc123",
						Type:  "function",
						Function: &OpenAIStreamFunction{
							Name:      "get_weather",
							Arguments: "",
						},
					}},
				},
			}},
		},
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{
					ToolCalls: []OpenAIStreamToolCall{{
						Index: 0,
						Function: &OpenAIStreamFunction{
							Arguments: `{"loc`,
						},
					}},
				},
			}},
		},
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{
					ToolCalls: []OpenAIStreamToolCall{{
						Index: 0,
						Function: &OpenAIStreamFunction{
							Arguments: `ation":"NYC"}`,
						},
					}},
				},
			}},
		},
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index:        0,
				Delta:        OpenAIStreamDelta{},
				FinishReason: ptr("tool_calls"),
			}},
		},
	)

	events, _, err := runStream(t, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEventTypes(t, events, []string{
		"message_start",
		"ping",
		"content_block_start",
		"content_block_delta",
		"content_block_delta",
		"content_block_stop",
		"message_delta",
		"message_stop",
	})

	// Verify tool_use content_block_start.
	var blockStart ContentBlockStartEvent
	mustUnmarshal(t, events[2].Data, &blockStart)
	if blockStart.ContentBlock.Type != "tool_use" {
		t.Errorf("expected tool_use, got %q", blockStart.ContentBlock.Type)
	}
	if blockStart.ContentBlock.ID != "call_abc123" {
		t.Errorf("expected call_abc123, got %q", blockStart.ContentBlock.ID)
	}
	if blockStart.ContentBlock.Name != "get_weather" {
		t.Errorf("expected get_weather, got %q", blockStart.ContentBlock.Name)
	}

	// Verify input_json_delta events.
	var d1 ContentBlockDeltaEvent
	mustUnmarshal(t, events[3].Data, &d1)
	if d1.Delta.Type != "input_json_delta" {
		t.Errorf("expected input_json_delta, got %q", d1.Delta.Type)
	}
	if d1.Delta.PartialJSON != `{"loc` {
		t.Errorf("unexpected partial json: %q", d1.Delta.PartialJSON)
	}

	// Verify stop reason is tool_use.
	var msgDelta MessageDeltaEvent
	mustUnmarshal(t, events[6].Data, &msgDelta)
	if msgDelta.Delta.StopReason == nil || *msgDelta.Delta.StopReason != "tool_use" {
		t.Errorf("expected stop_reason 'tool_use', got %v", msgDelta.Delta.StopReason)
	}
}

func TestTextThenToolCall(t *testing.T) {
	body := sseLines(
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{Role: "assistant"},
			}},
		},
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{Content: ptr("Let me check.")},
			}},
		},
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{
					ToolCalls: []OpenAIStreamToolCall{{
						Index: 0,
						ID:    "call_xyz",
						Type:  "function",
						Function: &OpenAIStreamFunction{
							Name:      "search",
							Arguments: `{"q":"test"}`,
						},
					}},
				},
			}},
		},
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index:        0,
				Delta:        OpenAIStreamDelta{},
				FinishReason: ptr("tool_calls"),
			}},
		},
	)

	events, _, err := runStream(t, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have two content blocks: text then tool_use.
	assertEventTypes(t, events, []string{
		"message_start",
		"ping",
		"content_block_start",  // text
		"content_block_delta",  // text delta
		"content_block_stop",   // close text
		"content_block_start",  // tool_use
		"content_block_delta",  // arguments
		"content_block_stop",   // close tool_use
		"message_delta",
		"message_stop",
	})

	// Verify indices.
	var textStart ContentBlockStartEvent
	mustUnmarshal(t, events[2].Data, &textStart)
	if textStart.Index != 0 {
		t.Errorf("expected text block index 0, got %d", textStart.Index)
	}

	var toolStart ContentBlockStartEvent
	mustUnmarshal(t, events[5].Data, &toolStart)
	if toolStart.Index != 1 {
		t.Errorf("expected tool block index 1, got %d", toolStart.Index)
	}
}

func TestMultipleToolCalls(t *testing.T) {
	body := sseLines(
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{Role: "assistant"},
			}},
		},
		// Two tool calls starting in the same chunk.
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{
					ToolCalls: []OpenAIStreamToolCall{
						{
							Index: 0,
							ID:    "call_1",
							Type:  "function",
							Function: &OpenAIStreamFunction{
								Name:      "func_a",
								Arguments: `{"a":1}`,
							},
						},
					},
				},
			}},
		},
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{
					ToolCalls: []OpenAIStreamToolCall{
						{
							Index: 1,
							ID:    "call_2",
							Type:  "function",
							Function: &OpenAIStreamFunction{
								Name:      "func_b",
								Arguments: `{"b":2}`,
							},
						},
					},
				},
			}},
		},
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index:        0,
				Delta:        OpenAIStreamDelta{},
				FinishReason: ptr("tool_calls"),
			}},
		},
	)

	events, _, err := runStream(t, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Two tool_use blocks.
	assertEventTypes(t, events, []string{
		"message_start",
		"ping",
		"content_block_start",  // tool 1
		"content_block_delta",  // tool 1 args
		"content_block_stop",   // close tool 1
		"content_block_start",  // tool 2
		"content_block_delta",  // tool 2 args
		"content_block_stop",   // close tool 2
		"message_delta",
		"message_stop",
	})

	var block1 ContentBlockStartEvent
	mustUnmarshal(t, events[2].Data, &block1)
	if block1.ContentBlock.Name != "func_a" {
		t.Errorf("expected func_a, got %q", block1.ContentBlock.Name)
	}
	if block1.Index != 0 {
		t.Errorf("expected index 0, got %d", block1.Index)
	}

	var block2 ContentBlockStartEvent
	mustUnmarshal(t, events[5].Data, &block2)
	if block2.ContentBlock.Name != "func_b" {
		t.Errorf("expected func_b, got %q", block2.ContentBlock.Name)
	}
	if block2.Index != 1 {
		t.Errorf("expected index 1, got %d", block2.Index)
	}
}

func TestUsageInFinalChunk(t *testing.T) {
	body := sseLines(
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{Content: ptr("Hi")},
			}},
		},
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index:        0,
				Delta:        OpenAIStreamDelta{},
				FinishReason: ptr("stop"),
			}},
		},
		// Usage-only chunk (empty choices).
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{},
			Usage: &OpenAIUsage{
				PromptTokens:     10,
				CompletionTokens: 42,
				TotalTokens:      52,
			},
		},
	)

	events, result, err := runStream(t, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find message_delta and check usage.
	var msgDelta MessageDeltaEvent
	for _, e := range events {
		if e.Type == "message_delta" {
			mustUnmarshal(t, e.Data, &msgDelta)
			break
		}
	}
	if msgDelta.Usage == nil {
		t.Fatal("expected usage in message_delta")
	}
	if msgDelta.Usage.OutputTokens != 42 {
		t.Errorf("expected 42 output tokens, got %d", msgDelta.Usage.OutputTokens)
	}

	// Verify StreamResult captures usage.
	if result == nil {
		t.Fatal("expected non-nil StreamResult")
	}
	if result.InputTokens != 10 {
		t.Errorf("expected 10 input tokens in result, got %d", result.InputTokens)
	}
	if result.OutputTokens != 42 {
		t.Errorf("expected 42 output tokens in result, got %d", result.OutputTokens)
	}
}

func TestEmptyContentDeltaSkipped(t *testing.T) {
	body := sseLines(
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{Content: ptr("")},
			}},
		},
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{Content: ptr("Real content")},
			}},
		},
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index:        0,
				Delta:        OpenAIStreamDelta{},
				FinishReason: ptr("stop"),
			}},
		},
	)

	events, _, err := runStream(t, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only one content_block_start (empty string should not trigger one).
	starts := 0
	for _, e := range events {
		if e.Type == "content_block_start" {
			starts++
		}
	}
	if starts != 1 {
		t.Errorf("expected 1 content_block_start, got %d", starts)
	}
}

func TestFinishReasonMapping(t *testing.T) {
	tests := []struct {
		oai      string
		expected string
	}{
		{"stop", "end_turn"},
		{"tool_calls", "tool_use"},
		{"length", "max_tokens"},
		{"content_filter", "end_turn"},
		{"unknown_reason", "end_turn"},
	}

	for _, tt := range tests {
		t.Run(tt.oai, func(t *testing.T) {
			body := sseLines(
				OpenAIStreamChunk{
					Choices: []OpenAIStreamChoice{{
						Index: 0,
						Delta: OpenAIStreamDelta{Content: ptr("x")},
					}},
				},
				OpenAIStreamChunk{
					Choices: []OpenAIStreamChoice{{
						Index:        0,
						Delta:        OpenAIStreamDelta{},
						FinishReason: ptr(tt.oai),
					}},
				},
			)

			events, _, err := runStream(t, body)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, e := range events {
				if e.Type == "message_delta" {
					var md MessageDeltaEvent
					mustUnmarshal(t, e.Data, &md)
					if md.Delta.StopReason == nil || *md.Delta.StopReason != tt.expected {
						t.Errorf("expected stop_reason %q, got %v", tt.expected, md.Delta.StopReason)
					}
					return
				}
			}
			t.Error("no message_delta found")
		})
	}
}

func TestDataDoneHandling(t *testing.T) {
	// Ensure data: [DONE] properly terminates the stream.
	raw := sseRaw(
		`data: {"choices":[{"index":0,"delta":{"content":"hi"}}]}`,
		"",
		"data: [DONE]",
		"",
	)

	events, _, err := runStream(t, raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must end with message_stop.
	last := events[len(events)-1]
	if last.Type != "message_stop" {
		t.Errorf("expected last event to be message_stop, got %q", last.Type)
	}
}

func TestFirstChunkRoleOnly(t *testing.T) {
	// First chunk has role but no content -> no content block started yet.
	body := sseLines(
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{Role: "assistant"},
			}},
		},
		OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index:        0,
				Delta:        OpenAIStreamDelta{},
				FinishReason: ptr("stop"),
			}},
		},
	)

	events, _, err := runStream(t, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have message_start, ping, message_delta, message_stop
	// but NO content_block_start (no content was produced).
	for _, e := range events {
		if e.Type == "content_block_start" {
			t.Error("unexpected content_block_start for role-only stream")
		}
	}

	assertEventTypes(t, events, []string{
		"message_start",
		"ping",
		"message_delta",
		"message_stop",
	})
}

func TestContextCancellation(t *testing.T) {
	// Create a stream that will block.
	pr, pw := io.Pipe()

	// Write one chunk.
	go func() {
		chunk := OpenAIStreamChunk{
			Choices: []OpenAIStreamChoice{{
				Index: 0,
				Delta: OpenAIStreamDelta{Content: ptr("hello")},
			}},
		}
		b, _ := json.Marshal(chunk)
		pw.Write([]byte("data: "))
		pw.Write(b)
		pw.Write([]byte("\n\n"))
		// Don't close - let it hang to simulate slow upstream.
		time.Sleep(5 * time.Second)
		pw.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after a short delay.
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	rec := httptest.NewRecorder()
	flusher := &mockFlusher{rec}
	_, err := TranslateOpenAIStreamToAnthropic(ctx, pr, rec, flusher, "claude-opus-4-6")
	if err == nil {
		t.Error("expected error from context cancellation")
	}
}

func TestSkipsNonDataLines(t *testing.T) {
	// Some providers send "event:" lines before "data:" lines.
	raw := sseRaw(
		"event: message",
		`data: {"choices":[{"index":0,"delta":{"content":"ok"}}]}`,
		"",
		": comment line",
		`data: {"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		"",
		"data: [DONE]",
		"",
	)

	events, _, err := runStream(t, raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still produce valid output.
	found := false
	for _, e := range events {
		if e.Type == "content_block_delta" {
			var d ContentBlockDeltaEvent
			mustUnmarshal(t, e.Data, &d)
			if d.Delta.Text == "ok" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected to find 'ok' in content_block_delta")
	}
}

func TestMalformedChunkSkipped(t *testing.T) {
	raw := sseRaw(
		`data: {"choices":[{"index":0,"delta":{"content":"before"}}]}`,
		"",
		`data: {invalid json`,
		"",
		`data: {"choices":[{"index":0,"delta":{"content":"after"}}]}`,
		"",
		"data: [DONE]",
		"",
	)

	events, _, err := runStream(t, raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	deltas := 0
	for _, e := range events {
		if e.Type == "content_block_delta" {
			deltas++
		}
	}
	if deltas != 2 {
		t.Errorf("expected 2 content_block_delta events (skipping malformed), got %d", deltas)
	}
}

// --- error translation tests ------------------------------------------------

func TestTranslateOpenAIErrorToAnthropic(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		body           string
		wantType       string
		wantStatus     int
		wantContains   string
	}{
		{
			name:       "400 bad request",
			statusCode: 400,
			body:       `{"error":{"message":"Invalid model","type":"invalid_request_error","param":null,"code":null}}`,
			wantType:   "invalid_request_error",
			wantStatus: 400,
			wantContains: "Invalid model",
		},
		{
			name:       "401 auth error",
			statusCode: 401,
			body:       `{"error":{"message":"Invalid API key","type":"invalid_api_key","param":null,"code":"invalid_api_key"}}`,
			wantType:   "authentication_error",
			wantStatus: 401,
			wantContains: "Invalid API key",
		},
		{
			name:       "429 rate limit",
			statusCode: 429,
			body:       `{"error":{"message":"Rate limit exceeded","type":"rate_limit_error","param":null,"code":null}}`,
			wantType:   "rate_limit_error",
			wantStatus: 429,
			wantContains: "Rate limit exceeded",
		},
		{
			name:       "500 server error",
			statusCode: 500,
			body:       `{"error":{"message":"Internal error","type":"server_error","param":null,"code":null}}`,
			wantType:   "api_error",
			wantStatus: 502,
			wantContains: "Internal error",
		},
		{
			name:       "unparseable body",
			statusCode: 502,
			body:       `not json at all`,
			wantType:   "api_error",
			wantStatus: 502,
			wantContains: "not json at all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, status := TranslateOpenAIErrorToAnthropic(tt.statusCode, []byte(tt.body))
			if status != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, status)
			}

			var resp AnthropicErrorResponse
			if err := json.Unmarshal(result, &resp); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}
			if resp.Type != "error" {
				t.Errorf("expected type 'error', got %q", resp.Type)
			}
			if resp.Error.Type != tt.wantType {
				t.Errorf("expected error type %q, got %q", tt.wantType, resp.Error.Type)
			}
			if !strings.Contains(resp.Error.Message, tt.wantContains) {
				t.Errorf("expected message containing %q, got %q", tt.wantContains, resp.Error.Message)
			}
		})
	}
}

// --- assertion helpers ------------------------------------------------------

func assertEventTypes(t *testing.T, events []sseEvent, expected []string) {
	t.Helper()
	if len(events) != len(expected) {
		types := make([]string, len(events))
		for i, e := range events {
			types[i] = e.Type
		}
		t.Fatalf("expected %d events %v, got %d events %v", len(expected), expected, len(events), types)
	}
	for i, e := range events {
		if e.Type != expected[i] {
			t.Errorf("event[%d]: expected type %q, got %q", i, expected[i], e.Type)
		}
	}
}

func mustUnmarshal(t *testing.T, data string, v interface{}) {
	t.Helper()
	if err := json.Unmarshal([]byte(data), v); err != nil {
		t.Fatalf("failed to unmarshal %q: %v", data, err)
	}
}

// Verify that the recorder implements http.Flusher via our wrapper.
var _ http.Flusher = (*mockFlusher)(nil)
