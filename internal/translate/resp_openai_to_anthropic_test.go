package translate

import (
	"encoding/json"
	"strings"
	"testing"
)

func strPtr(s string) *string { return &s }

func TestOpenAIResponseToAnthropic_SimpleText(t *testing.T) {
	resp := &OpenAIResponse{
		ID:    "chatcmpl-123",
		Model: "gpt-4",
		Choices: []OpenAIChoice{
			{
				Index:        0,
				Message:      OpenAIMessage{Role: "assistant", Content: "Hello, world!"},
				FinishReason: strPtr("stop"),
			},
		},
		Usage: &OpenAIUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}

	result, err := OpenAIResponseToAnthropic(resp, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(result.ID, "msg_") {
		t.Errorf("ID should start with msg_, got %s", result.ID)
	}
	if len(result.ID) != 4+24 { // "msg_" + 24 hex chars
		t.Errorf("ID length should be 28, got %d", len(result.ID))
	}
	if result.Type != "message" {
		t.Errorf("Type = %q, want %q", result.Type, "message")
	}
	if result.Role != "assistant" {
		t.Errorf("Role = %q, want %q", result.Role, "assistant")
	}
	if result.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Model = %q, want %q", result.Model, "claude-sonnet-4-20250514")
	}
	if result.StopSequence != nil {
		t.Errorf("StopSequence should be nil, got %v", result.StopSequence)
	}
	if result.StopReason == nil || *result.StopReason != "end_turn" {
		t.Errorf("StopReason = %v, want %q", result.StopReason, "end_turn")
	}
	if len(result.Content) != 1 {
		t.Fatalf("Content length = %d, want 1", len(result.Content))
	}
	if result.Content[0].Type != "text" {
		t.Errorf("Content[0].Type = %q, want %q", result.Content[0].Type, "text")
	}
	if result.Content[0].Text != "Hello, world!" {
		t.Errorf("Content[0].Text = %q, want %q", result.Content[0].Text, "Hello, world!")
	}
}

func TestOpenAIResponseToAnthropic_SingleToolCall(t *testing.T) {
	resp := &OpenAIResponse{
		Choices: []OpenAIChoice{
			{
				Message: OpenAIMessage{
					Role: "assistant",
					ToolCalls: []OpenAIToolCall{
						{
							ID:   "call_abc123",
							Type: "function",
							Function: OpenAIFunction{
								Name:      "get_weather",
								Arguments: `{"location":"NYC"}`,
							},
						},
					},
				},
				FinishReason: strPtr("tool_calls"),
			},
		},
		Usage: &OpenAIUsage{PromptTokens: 20, CompletionTokens: 10},
	}

	result, err := OpenAIResponseToAnthropic(resp, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Content) != 1 {
		t.Fatalf("Content length = %d, want 1", len(result.Content))
	}

	block := result.Content[0]
	if block.Type != "tool_use" {
		t.Errorf("Type = %q, want %q", block.Type, "tool_use")
	}
	if block.ID != "call_abc123" {
		t.Errorf("ID = %q, want %q", block.ID, "call_abc123")
	}
	if block.Name != "get_weather" {
		t.Errorf("Name = %q, want %q", block.Name, "get_weather")
	}

	var input map[string]interface{}
	if err := json.Unmarshal(block.Input, &input); err != nil {
		t.Fatalf("failed to unmarshal input: %v", err)
	}
	if input["location"] != "NYC" {
		t.Errorf("input[location] = %v, want NYC", input["location"])
	}

	if result.StopReason == nil || *result.StopReason != "tool_use" {
		t.Errorf("StopReason = %v, want %q", result.StopReason, "tool_use")
	}
}

func TestOpenAIResponseToAnthropic_MultipleToolCalls(t *testing.T) {
	resp := &OpenAIResponse{
		Choices: []OpenAIChoice{
			{
				Message: OpenAIMessage{
					Role: "assistant",
					ToolCalls: []OpenAIToolCall{
						{
							ID:       "call_1",
							Type:     "function",
							Function: OpenAIFunction{Name: "search", Arguments: `{"q":"go"}`},
						},
						{
							ID:       "call_2",
							Type:     "function",
							Function: OpenAIFunction{Name: "fetch", Arguments: `{"url":"https://example.com"}`},
						},
					},
				},
				FinishReason: strPtr("tool_calls"),
			},
		},
	}

	result, err := OpenAIResponseToAnthropic(resp, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Content) != 2 {
		t.Fatalf("Content length = %d, want 2", len(result.Content))
	}

	if result.Content[0].Name != "search" {
		t.Errorf("Content[0].Name = %q, want %q", result.Content[0].Name, "search")
	}
	if result.Content[1].Name != "fetch" {
		t.Errorf("Content[1].Name = %q, want %q", result.Content[1].Name, "fetch")
	}
}

func TestOpenAIResponseToAnthropic_MixedTextAndToolCalls(t *testing.T) {
	resp := &OpenAIResponse{
		Choices: []OpenAIChoice{
			{
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: "Let me look that up.",
					ToolCalls: []OpenAIToolCall{
						{
							ID:       "call_mix1",
							Type:     "function",
							Function: OpenAIFunction{Name: "lookup", Arguments: `{"key":"val"}`},
						},
					},
				},
				FinishReason: strPtr("tool_calls"),
			},
		},
	}

	result, err := OpenAIResponseToAnthropic(resp, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Content) != 2 {
		t.Fatalf("Content length = %d, want 2", len(result.Content))
	}

	// Text block comes first.
	if result.Content[0].Type != "text" {
		t.Errorf("Content[0].Type = %q, want %q", result.Content[0].Type, "text")
	}
	if result.Content[0].Text != "Let me look that up." {
		t.Errorf("Content[0].Text = %q, want %q", result.Content[0].Text, "Let me look that up.")
	}

	// Then tool_use block.
	if result.Content[1].Type != "tool_use" {
		t.Errorf("Content[1].Type = %q, want %q", result.Content[1].Type, "tool_use")
	}
	if result.Content[1].Name != "lookup" {
		t.Errorf("Content[1].Name = %q, want %q", result.Content[1].Name, "lookup")
	}
}

func TestOpenAIResponseToAnthropic_FinishReasonMappings(t *testing.T) {
	tests := []struct {
		name         string
		finishReason *string
		wantStop     string
	}{
		{"stop", strPtr("stop"), "end_turn"},
		{"tool_calls", strPtr("tool_calls"), "tool_use"},
		{"length", strPtr("length"), "max_tokens"},
		{"content_filter", strPtr("content_filter"), "end_turn"},
		{"nil", nil, "end_turn"},
		{"empty", strPtr(""), "end_turn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &OpenAIResponse{
				Choices: []OpenAIChoice{
					{
						Message:      OpenAIMessage{Role: "assistant", Content: "text"},
						FinishReason: tt.finishReason,
					},
				},
			}

			result, err := OpenAIResponseToAnthropic(resp, "claude-sonnet-4-20250514")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.StopReason == nil {
				t.Fatalf("StopReason is nil, want %q", tt.wantStop)
			}
			if *result.StopReason != tt.wantStop {
				t.Errorf("StopReason = %q, want %q", *result.StopReason, tt.wantStop)
			}
		})
	}
}

func TestOpenAIResponseToAnthropic_UsageTranslation(t *testing.T) {
	resp := &OpenAIResponse{
		Choices: []OpenAIChoice{
			{
				Message:      OpenAIMessage{Role: "assistant", Content: "hi"},
				FinishReason: strPtr("stop"),
			},
		},
		Usage: &OpenAIUsage{PromptTokens: 42, CompletionTokens: 17, TotalTokens: 59},
	}

	result, err := OpenAIResponseToAnthropic(resp, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Usage.InputTokens != 42 {
		t.Errorf("InputTokens = %d, want 42", result.Usage.InputTokens)
	}
	if result.Usage.OutputTokens != 17 {
		t.Errorf("OutputTokens = %d, want 17", result.Usage.OutputTokens)
	}
}

func TestOpenAIResponseToAnthropic_NilUsage(t *testing.T) {
	resp := &OpenAIResponse{
		Choices: []OpenAIChoice{
			{
				Message:      OpenAIMessage{Role: "assistant", Content: "hi"},
				FinishReason: strPtr("stop"),
			},
		},
		Usage: nil,
	}

	result, err := OpenAIResponseToAnthropic(resp, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Usage.InputTokens != 0 {
		t.Errorf("InputTokens = %d, want 0", result.Usage.InputTokens)
	}
	if result.Usage.OutputTokens != 0 {
		t.Errorf("OutputTokens = %d, want 0", result.Usage.OutputTokens)
	}
}

func TestOpenAIResponseToAnthropic_EmptyChoices(t *testing.T) {
	resp := &OpenAIResponse{
		Choices: []OpenAIChoice{},
	}

	_, err := OpenAIResponseToAnthropic(resp, "claude-sonnet-4-20250514")
	if err == nil {
		t.Fatal("expected error for empty choices, got nil")
	}
	if !strings.Contains(err.Error(), "no choices") {
		t.Errorf("error = %q, want it to mention 'no choices'", err.Error())
	}
}

func TestOpenAIResponseToAnthropic_InvalidToolCallJSON(t *testing.T) {
	resp := &OpenAIResponse{
		Choices: []OpenAIChoice{
			{
				Message: OpenAIMessage{
					Role: "assistant",
					ToolCalls: []OpenAIToolCall{
						{
							ID:       "call_bad",
							Type:     "function",
							Function: OpenAIFunction{Name: "broken", Arguments: `{invalid json`},
						},
					},
				},
				FinishReason: strPtr("tool_calls"),
			},
		},
	}

	result, err := OpenAIResponseToAnthropic(resp, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("should not error on bad tool call JSON, got: %v", err)
	}

	if len(result.Content) != 1 {
		t.Fatalf("Content length = %d, want 1", len(result.Content))
	}

	var input map[string]interface{}
	if err := json.Unmarshal(result.Content[0].Input, &input); err != nil {
		t.Fatalf("failed to unmarshal input: %v", err)
	}
	if len(input) != 0 {
		t.Errorf("input should be empty object, got %v", input)
	}
}

func TestOpenAIResponseToAnthropic_NullContentWithToolCalls(t *testing.T) {
	resp := &OpenAIResponse{
		Choices: []OpenAIChoice{
			{
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: nil,
					ToolCalls: []OpenAIToolCall{
						{
							ID:       "call_nil",
							Type:     "function",
							Function: OpenAIFunction{Name: "do_thing", Arguments: `{"a":1}`},
						},
					},
				},
				FinishReason: strPtr("tool_calls"),
			},
		},
	}

	result, err := OpenAIResponseToAnthropic(resp, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Content) != 1 {
		t.Fatalf("Content length = %d, want 1", len(result.Content))
	}
	if result.Content[0].Type != "tool_use" {
		t.Errorf("Content[0].Type = %q, want %q", result.Content[0].Type, "tool_use")
	}
}

func TestOpenAIResponseToAnthropic_EmptyContentString(t *testing.T) {
	resp := &OpenAIResponse{
		Choices: []OpenAIChoice{
			{
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: "",
					ToolCalls: []OpenAIToolCall{
						{
							ID:       "call_empty",
							Type:     "function",
							Function: OpenAIFunction{Name: "act", Arguments: `{}`},
						},
					},
				},
				FinishReason: strPtr("tool_calls"),
			},
		},
	}

	result, err := OpenAIResponseToAnthropic(resp, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty string content should be skipped; only tool_use block.
	if len(result.Content) != 1 {
		t.Fatalf("Content length = %d, want 1", len(result.Content))
	}
	if result.Content[0].Type != "tool_use" {
		t.Errorf("Content[0].Type = %q, want %q", result.Content[0].Type, "tool_use")
	}
}

func TestOpenAIResponseToAnthropic_EmptyToolCallID(t *testing.T) {
	resp := &OpenAIResponse{
		Choices: []OpenAIChoice{
			{
				Message: OpenAIMessage{
					Role: "assistant",
					ToolCalls: []OpenAIToolCall{
						{
							ID:       "",
							Type:     "function",
							Function: OpenAIFunction{Name: "test_fn", Arguments: `{}`},
						},
					},
				},
				FinishReason: strPtr("tool_calls"),
			},
		},
	}

	result, err := OpenAIResponseToAnthropic(resp, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Content) != 1 {
		t.Fatalf("Content length = %d, want 1", len(result.Content))
	}

	id := result.Content[0].ID
	if !strings.HasPrefix(id, "toolu_") {
		t.Errorf("generated ID should start with toolu_, got %q", id)
	}
	if len(id) != 6+24 { // "toolu_" + 24 hex chars
		t.Errorf("generated ID length = %d, want 30", len(id))
	}
}

func TestOpenAIResponseToAnthropic_MultipleChoicesUsesFirst(t *testing.T) {
	resp := &OpenAIResponse{
		Choices: []OpenAIChoice{
			{
				Index:        0,
				Message:      OpenAIMessage{Role: "assistant", Content: "first"},
				FinishReason: strPtr("stop"),
			},
			{
				Index:        1,
				Message:      OpenAIMessage{Role: "assistant", Content: "second"},
				FinishReason: strPtr("stop"),
			},
		},
	}

	result, err := OpenAIResponseToAnthropic(resp, "claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Content) != 1 {
		t.Fatalf("Content length = %d, want 1", len(result.Content))
	}
	if result.Content[0].Text != "first" {
		t.Errorf("Content[0].Text = %q, want %q", result.Content[0].Text, "first")
	}
}
