package translate

import (
	"encoding/json"
	"testing"
)

// helper to create json.RawMessage from any value.
func mustJSON(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func float64Ptr(f float64) *float64 { return &f }

func TestAnthropicRequestToOpenAI(t *testing.T) {
	tests := []struct {
		name    string
		input   AnthropicRequest
		check   func(t *testing.T, out *OpenAIRequest)
		wantErr bool
	}{
		{
			name: "simple text user+assistant",
			input: AnthropicRequest{
				Model:     "claude-3-opus-20240229",
				MaxTokens: 1024,
				Messages: []AnthropicMessage{
					{Role: "user", Content: mustJSON("Hello")},
					{Role: "assistant", Content: mustJSON("Hi there!")},
				},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if out.Model != "claude-3-opus-20240229" {
					t.Errorf("model = %q, want %q", out.Model, "claude-3-opus-20240229")
				}
				if *out.MaxTokens != 1024 {
					t.Errorf("max_tokens = %d, want 1024", *out.MaxTokens)
				}
				if len(out.Messages) != 2 {
					t.Fatalf("got %d messages, want 2", len(out.Messages))
				}
				if out.Messages[0].Role != "user" || out.Messages[0].Content != "Hello" {
					t.Errorf("msg[0] = %+v", out.Messages[0])
				}
				if out.Messages[1].Role != "assistant" || out.Messages[1].Content != "Hi there!" {
					t.Errorf("msg[1] = %+v", out.Messages[1])
				}
			},
		},
		{
			name: "system as string",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 100,
				System:    mustJSON("You are a helpful assistant."),
				Messages: []AnthropicMessage{
					{Role: "user", Content: mustJSON("Hi")},
				},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if len(out.Messages) != 2 {
					t.Fatalf("got %d messages, want 2", len(out.Messages))
				}
				if out.Messages[0].Role != "system" {
					t.Errorf("msg[0].role = %q, want system", out.Messages[0].Role)
				}
				if out.Messages[0].Content != "You are a helpful assistant." {
					t.Errorf("system content = %v", out.Messages[0].Content)
				}
			},
		},
		{
			name: "system as array of blocks",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 100,
				System: mustJSON([]SystemBlock{
					{Type: "text", Text: "You are helpful.", CacheControl: &CacheControl{Type: "ephemeral"}},
					{Type: "text", Text: "Be concise."},
				}),
				Messages: []AnthropicMessage{
					{Role: "user", Content: mustJSON("Hi")},
				},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if len(out.Messages) != 2 {
					t.Fatalf("got %d messages, want 2", len(out.Messages))
				}
				if out.Messages[0].Content != "You are helpful.\n\nBe concise." {
					t.Errorf("system content = %v", out.Messages[0].Content)
				}
			},
		},
		{
			name: "image base64 in user message",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 100,
				Messages: []AnthropicMessage{
					{
						Role: "user",
						Content: mustJSON([]ContentBlock{
							{Type: "text", Text: "What is this?"},
							{Type: "image", Source: &ImageSource{
								Type:      "base64",
								MediaType: "image/png",
								Data:      "iVBORw0KGgo=",
							}},
						}),
					},
				},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if len(out.Messages) != 1 {
					t.Fatalf("got %d messages, want 1", len(out.Messages))
				}
				parts, ok := out.Messages[0].Content.([]OpenAIContentPart)
				if !ok {
					t.Fatalf("content is not []OpenAIContentPart, got %T", out.Messages[0].Content)
				}
				if len(parts) != 2 {
					t.Fatalf("got %d parts, want 2", len(parts))
				}
				if parts[0].Type != "text" || parts[0].Text != "What is this?" {
					t.Errorf("parts[0] = %+v", parts[0])
				}
				if parts[1].Type != "image_url" || parts[1].ImageURL == nil {
					t.Fatalf("parts[1] = %+v", parts[1])
				}
				expected := "data:image/png;base64,iVBORw0KGgo="
				if parts[1].ImageURL.URL != expected {
					t.Errorf("image url = %q, want %q", parts[1].ImageURL.URL, expected)
				}
			},
		},
		{
			name: "image URL in user message",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 100,
				Messages: []AnthropicMessage{
					{
						Role: "user",
						Content: mustJSON([]ContentBlock{
							{Type: "image", Source: &ImageSource{
								Type: "url",
								URL:  "https://example.com/img.png",
							}},
						}),
					},
				},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				parts := out.Messages[0].Content.([]OpenAIContentPart)
				if parts[0].ImageURL.URL != "https://example.com/img.png" {
					t.Errorf("image url = %q", parts[0].ImageURL.URL)
				}
			},
		},
		{
			name: "single tool_use and tool_result round trip",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 1024,
				Messages: []AnthropicMessage{
					{Role: "user", Content: mustJSON("What is the weather?")},
					{
						Role: "assistant",
						Content: mustJSON([]ContentBlock{
							{Type: "tool_use", ID: "call_1", Name: "get_weather", Input: mustJSON(map[string]string{"city": "NYC"})},
						}),
					},
					{
						Role: "user",
						Content: mustJSON([]ContentBlock{
							{Type: "tool_result", ToolUseID: "call_1", Content: mustJSON("72F and sunny")},
						}),
					},
				},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if len(out.Messages) != 3 {
					t.Fatalf("got %d messages, want 3", len(out.Messages))
				}
				// assistant message with tool_calls
				aMsg := out.Messages[1]
				if aMsg.Role != "assistant" {
					t.Errorf("msg[1].role = %q", aMsg.Role)
				}
				if len(aMsg.ToolCalls) != 1 {
					t.Fatalf("tool_calls count = %d", len(aMsg.ToolCalls))
				}
				tc := aMsg.ToolCalls[0]
				if tc.ID != "call_1" || tc.Type != "function" || tc.Function.Name != "get_weather" {
					t.Errorf("tool call = %+v", tc)
				}
				if tc.Function.Arguments != `{"city":"NYC"}` {
					t.Errorf("arguments = %q", tc.Function.Arguments)
				}
				// tool message
				tMsg := out.Messages[2]
				if tMsg.Role != "tool" || tMsg.ToolCallID != "call_1" {
					t.Errorf("msg[2] = %+v", tMsg)
				}
				if tMsg.Content != "72F and sunny" {
					t.Errorf("tool content = %v", tMsg.Content)
				}
			},
		},
		{
			name: "multiple parallel tool calls",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 1024,
				Messages: []AnthropicMessage{
					{Role: "user", Content: mustJSON("Check weather and time")},
					{
						Role: "assistant",
						Content: mustJSON([]ContentBlock{
							{Type: "tool_use", ID: "call_a", Name: "get_weather", Input: mustJSON(map[string]string{"city": "NYC"})},
							{Type: "tool_use", ID: "call_b", Name: "get_time", Input: mustJSON(map[string]string{"tz": "EST"})},
						}),
					},
					{
						Role: "user",
						Content: mustJSON([]ContentBlock{
							{Type: "tool_result", ToolUseID: "call_a", Content: mustJSON("72F")},
							{Type: "tool_result", ToolUseID: "call_b", Content: mustJSON("3pm")},
						}),
					},
				},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				// user, assistant, tool(a), tool(b)
				if len(out.Messages) != 4 {
					t.Fatalf("got %d messages, want 4", len(out.Messages))
				}
				aMsg := out.Messages[1]
				if len(aMsg.ToolCalls) != 2 {
					t.Fatalf("tool_calls = %d", len(aMsg.ToolCalls))
				}
				if out.Messages[2].Role != "tool" || out.Messages[2].ToolCallID != "call_a" {
					t.Errorf("msg[2] = %+v", out.Messages[2])
				}
				if out.Messages[3].Role != "tool" || out.Messages[3].ToolCallID != "call_b" {
					t.Errorf("msg[3] = %+v", out.Messages[3])
				}
			},
		},
		{
			name: "mixed text and tool_use in assistant message",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 1024,
				Messages: []AnthropicMessage{
					{Role: "user", Content: mustJSON("Help")},
					{
						Role: "assistant",
						Content: mustJSON([]ContentBlock{
							{Type: "text", Text: "Let me check. "},
							{Type: "tool_use", ID: "call_x", Name: "search", Input: mustJSON(map[string]string{"q": "foo"})},
						}),
					},
				},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				aMsg := out.Messages[1]
				if aMsg.Content != "Let me check. " {
					t.Errorf("content = %v", aMsg.Content)
				}
				if len(aMsg.ToolCalls) != 1 {
					t.Fatalf("tool_calls = %d", len(aMsg.ToolCalls))
				}
			},
		},
		{
			name: "thinking blocks stripped",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 1024,
				Messages: []AnthropicMessage{
					{Role: "user", Content: mustJSON("Think hard")},
					{
						Role: "assistant",
						Content: mustJSON([]ContentBlock{
							{Type: "thinking", Thinking: "hmm...", Signature: "sig123"},
							{Type: "text", Text: "Here is my answer."},
						}),
					},
				},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				aMsg := out.Messages[1]
				if aMsg.Content != "Here is my answer." {
					t.Errorf("content = %v", aMsg.Content)
				}
				if len(aMsg.ToolCalls) != 0 {
					t.Errorf("unexpected tool_calls")
				}
			},
		},
		{
			name: "tool choice string auto",
			input: AnthropicRequest{
				Model:      "claude-3-sonnet",
				MaxTokens:  100,
				ToolChoice: mustJSON("auto"),
				Messages:   []AnthropicMessage{{Role: "user", Content: mustJSON("Hi")}},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if out.ToolChoice != "auto" {
					t.Errorf("tool_choice = %v", out.ToolChoice)
				}
			},
		},
		{
			name: "tool choice string any → required",
			input: AnthropicRequest{
				Model:      "claude-3-sonnet",
				MaxTokens:  100,
				ToolChoice: mustJSON("any"),
				Messages:   []AnthropicMessage{{Role: "user", Content: mustJSON("Hi")}},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if out.ToolChoice != "required" {
					t.Errorf("tool_choice = %v", out.ToolChoice)
				}
			},
		},
		{
			name: "tool choice string none",
			input: AnthropicRequest{
				Model:      "claude-3-sonnet",
				MaxTokens:  100,
				ToolChoice: mustJSON("none"),
				Messages:   []AnthropicMessage{{Role: "user", Content: mustJSON("Hi")}},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if out.ToolChoice != "none" {
					t.Errorf("tool_choice = %v", out.ToolChoice)
				}
			},
		},
		{
			name: "tool choice object auto",
			input: AnthropicRequest{
				Model:      "claude-3-sonnet",
				MaxTokens:  100,
				ToolChoice: mustJSON(ToolChoiceObj{Type: "auto"}),
				Messages:   []AnthropicMessage{{Role: "user", Content: mustJSON("Hi")}},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if out.ToolChoice != "auto" {
					t.Errorf("tool_choice = %v", out.ToolChoice)
				}
			},
		},
		{
			name: "tool choice object any → required",
			input: AnthropicRequest{
				Model:      "claude-3-sonnet",
				MaxTokens:  100,
				ToolChoice: mustJSON(ToolChoiceObj{Type: "any"}),
				Messages:   []AnthropicMessage{{Role: "user", Content: mustJSON("Hi")}},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if out.ToolChoice != "required" {
					t.Errorf("tool_choice = %v", out.ToolChoice)
				}
			},
		},
		{
			name: "tool choice object tool with name",
			input: AnthropicRequest{
				Model:      "claude-3-sonnet",
				MaxTokens:  100,
				ToolChoice: mustJSON(ToolChoiceObj{Type: "tool", Name: "my_func"}),
				Messages:   []AnthropicMessage{{Role: "user", Content: mustJSON("Hi")}},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				tc, ok := out.ToolChoice.(OpenAIToolChoiceFunction)
				if !ok {
					t.Fatalf("tool_choice type = %T", out.ToolChoice)
				}
				if tc.Type != "function" || tc.Function.Name != "my_func" {
					t.Errorf("tool_choice = %+v", tc)
				}
			},
		},
		{
			name: "streaming flag",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 100,
				Stream:    true,
				Messages:  []AnthropicMessage{{Role: "user", Content: mustJSON("Hi")}},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if !out.Stream {
					t.Error("stream should be true")
				}
				if out.StreamOptions == nil || !out.StreamOptions.IncludeUsage {
					t.Error("stream_options.include_usage should be true")
				}
			},
		},
		{
			name: "no streaming by default",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 100,
				Messages:  []AnthropicMessage{{Role: "user", Content: mustJSON("Hi")}},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if out.Stream {
					t.Error("stream should be false")
				}
				if out.StreamOptions != nil {
					t.Error("stream_options should be nil")
				}
			},
		},
		{
			name: "metadata user_id → user",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 100,
				Metadata:  &Metadata{UserID: "user-123"},
				Messages:  []AnthropicMessage{{Role: "user", Content: mustJSON("Hi")}},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if out.User != "user-123" {
					t.Errorf("user = %q", out.User)
				}
			},
		},
		{
			name: "temperature and top_p passthrough",
			input: AnthropicRequest{
				Model:       "claude-3-sonnet",
				MaxTokens:   100,
				Temperature: float64Ptr(0.7),
				TopP:        float64Ptr(0.9),
				Messages:    []AnthropicMessage{{Role: "user", Content: mustJSON("Hi")}},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if out.Temperature == nil || *out.Temperature != 0.7 {
					t.Errorf("temperature = %v", out.Temperature)
				}
				if out.TopP == nil || *out.TopP != 0.9 {
					t.Errorf("top_p = %v", out.TopP)
				}
			},
		},
		{
			name: "stop_sequences → stop",
			input: AnthropicRequest{
				Model:         "claude-3-sonnet",
				MaxTokens:     100,
				StopSequences: []string{"STOP", "END"},
				Messages:      []AnthropicMessage{{Role: "user", Content: mustJSON("Hi")}},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				stops, ok := out.Stop.([]string)
				if !ok {
					t.Fatalf("stop type = %T", out.Stop)
				}
				if len(stops) != 2 || stops[0] != "STOP" || stops[1] != "END" {
					t.Errorf("stop = %v", stops)
				}
			},
		},
		{
			name: "tools translation with skip non-custom",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 100,
				Tools: []AnthropicTool{
					{Name: "search", Description: "Search the web", InputSchema: mustJSON(map[string]interface{}{"type": "object", "properties": map[string]interface{}{"q": map[string]string{"type": "string"}}})},
					{Name: "web_search_builtin", Type: "web_search"},
					{Name: "custom_tool", Type: "custom", Description: "A custom tool", InputSchema: mustJSON(map[string]string{"type": "object"})},
				},
				Messages: []AnthropicMessage{{Role: "user", Content: mustJSON("Hi")}},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if len(out.Tools) != 2 {
					t.Fatalf("got %d tools, want 2", len(out.Tools))
				}
				if out.Tools[0].Function.Name != "search" {
					t.Errorf("tool[0].name = %q", out.Tools[0].Function.Name)
				}
				if out.Tools[0].Type != "function" {
					t.Errorf("tool[0].type = %q", out.Tools[0].Type)
				}
				if out.Tools[1].Function.Name != "custom_tool" {
					t.Errorf("tool[1].name = %q", out.Tools[1].Function.Name)
				}
			},
		},
		{
			name: "tool_result with is_error true",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 100,
				Messages: []AnthropicMessage{
					{Role: "user", Content: mustJSON("Run")},
					{
						Role: "assistant",
						Content: mustJSON([]ContentBlock{
							{Type: "tool_use", ID: "call_err", Name: "run", Input: mustJSON(map[string]string{})},
						}),
					},
					{
						Role: "user",
						Content: mustJSON([]ContentBlock{
							{Type: "tool_result", ToolUseID: "call_err", Content: mustJSON("Something went wrong"), IsError: true},
						}),
					},
				},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				// tool message should still contain the error content
				toolMsg := out.Messages[2]
				if toolMsg.Role != "tool" {
					t.Errorf("msg[2].role = %q", toolMsg.Role)
				}
				if toolMsg.Content != "Something went wrong" {
					t.Errorf("tool content = %v", toolMsg.Content)
				}
			},
		},
		{
			name: "tool_result with array content (text blocks)",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 100,
				Messages: []AnthropicMessage{
					{Role: "user", Content: mustJSON("Go")},
					{
						Role: "assistant",
						Content: mustJSON([]ContentBlock{
							{Type: "tool_use", ID: "call_arr", Name: "multi", Input: mustJSON(map[string]string{})},
						}),
					},
					{
						Role: "user",
						Content: mustJSON([]ContentBlock{
							{
								Type:      "tool_result",
								ToolUseID: "call_arr",
								Content: mustJSON([]ContentBlock{
									{Type: "text", Text: "Part 1. "},
									{Type: "text", Text: "Part 2."},
								}),
							},
						}),
					},
				},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				toolMsg := out.Messages[2]
				if toolMsg.Content != "Part 1. Part 2." {
					t.Errorf("tool content = %v", toolMsg.Content)
				}
			},
		},
		{
			name: "empty tool_use input {} → arguments {}",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 100,
				Messages: []AnthropicMessage{
					{Role: "user", Content: mustJSON("Do it")},
					{
						Role: "assistant",
						Content: mustJSON([]ContentBlock{
							{Type: "tool_use", ID: "call_empty", Name: "noop", Input: mustJSON(map[string]interface{}{})},
						}),
					},
				},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				tc := out.Messages[1].ToolCalls[0]
				if tc.Function.Arguments != "{}" {
					t.Errorf("arguments = %q, want %q", tc.Function.Arguments, "{}")
				}
			},
		},
		{
			name: "mixed text+tool_result in user message emits tool first then user",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 100,
				Messages: []AnthropicMessage{
					{Role: "user", Content: mustJSON("Start")},
					{
						Role: "assistant",
						Content: mustJSON([]ContentBlock{
							{Type: "tool_use", ID: "call_m", Name: "fn", Input: mustJSON(map[string]string{})},
						}),
					},
					{
						Role: "user",
						Content: mustJSON([]ContentBlock{
							{Type: "tool_result", ToolUseID: "call_m", Content: mustJSON("result")},
							{Type: "text", Text: "Thanks, now continue."},
						}),
					},
				},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				// user, assistant, tool, user
				if len(out.Messages) != 4 {
					t.Fatalf("got %d messages, want 4", len(out.Messages))
				}
				if out.Messages[2].Role != "tool" {
					t.Errorf("msg[2].role = %q, want tool", out.Messages[2].Role)
				}
				if out.Messages[3].Role != "user" {
					t.Errorf("msg[3].role = %q, want user", out.Messages[3].Role)
				}
				parts, ok := out.Messages[3].Content.([]OpenAIContentPart)
				if !ok {
					t.Fatalf("msg[3].content type = %T", out.Messages[3].Content)
				}
				if len(parts) != 1 || parts[0].Text != "Thanks, now continue." {
					t.Errorf("msg[3].content = %+v", parts)
				}
			},
		},
		{
			name: "model passthrough",
			input: AnthropicRequest{
				Model:     "claude-4-opus-20250514",
				MaxTokens: 100,
				Messages:  []AnthropicMessage{{Role: "user", Content: mustJSON("Hi")}},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if out.Model != "claude-4-opus-20250514" {
					t.Errorf("model = %q", out.Model)
				}
			},
		},
		{
			name: "top_k omitted",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 100,
				TopK:      intPtr(40),
				Messages:  []AnthropicMessage{{Role: "user", Content: mustJSON("Hi")}},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				// OpenAI request should not have top_k — verify via JSON output
				b, _ := json.Marshal(out)
				var raw map[string]interface{}
				json.Unmarshal(b, &raw)
				if _, ok := raw["top_k"]; ok {
					t.Error("top_k should not be present in OpenAI request")
				}
			},
		},
		{
			name: "no tool choice when nil",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 100,
				Messages:  []AnthropicMessage{{Role: "user", Content: mustJSON("Hi")}},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if out.ToolChoice != nil {
					t.Errorf("tool_choice = %v, want nil", out.ToolChoice)
				}
			},
		},
		{
			name: "multi-turn conversation",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 100,
				Messages: []AnthropicMessage{
					{Role: "user", Content: mustJSON("Hello")},
					{Role: "assistant", Content: mustJSON("Hi!")},
					{Role: "user", Content: mustJSON("How are you?")},
					{Role: "assistant", Content: mustJSON("I'm doing well!")},
					{Role: "user", Content: mustJSON("Great")},
				},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				if len(out.Messages) != 5 {
					t.Fatalf("got %d messages, want 5", len(out.Messages))
				}
				for i, expected := range []string{"user", "assistant", "user", "assistant", "user"} {
					if out.Messages[i].Role != expected {
						t.Errorf("msg[%d].role = %q, want %q", i, out.Messages[i].Role, expected)
					}
				}
			},
		},
		{
			name: "tool_result with nil/empty content",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 100,
				Messages: []AnthropicMessage{
					{Role: "user", Content: mustJSON("Go")},
					{
						Role: "assistant",
						Content: mustJSON([]ContentBlock{
							{Type: "tool_use", ID: "call_nil", Name: "noop", Input: mustJSON(map[string]interface{}{})},
						}),
					},
					{
						Role: "user",
						Content: mustJSON([]ContentBlock{
							{Type: "tool_result", ToolUseID: "call_nil"},
						}),
					},
				},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				toolMsg := out.Messages[2]
				if toolMsg.Content != "" {
					t.Errorf("tool content = %v, want empty string", toolMsg.Content)
				}
			},
		},
		{
			name: "consecutive tool_result blocks become separate tool messages",
			input: AnthropicRequest{
				Model:     "claude-3-sonnet",
				MaxTokens: 100,
				Messages: []AnthropicMessage{
					{Role: "user", Content: mustJSON("Go")},
					{
						Role: "assistant",
						Content: mustJSON([]ContentBlock{
							{Type: "tool_use", ID: "c1", Name: "fn1", Input: mustJSON(map[string]string{})},
							{Type: "tool_use", ID: "c2", Name: "fn2", Input: mustJSON(map[string]string{})},
							{Type: "tool_use", ID: "c3", Name: "fn3", Input: mustJSON(map[string]string{})},
						}),
					},
					{
						Role: "user",
						Content: mustJSON([]ContentBlock{
							{Type: "tool_result", ToolUseID: "c1", Content: mustJSON("r1")},
							{Type: "tool_result", ToolUseID: "c2", Content: mustJSON("r2")},
							{Type: "tool_result", ToolUseID: "c3", Content: mustJSON("r3")},
						}),
					},
				},
			},
			check: func(t *testing.T, out *OpenAIRequest) {
				// user, assistant, tool, tool, tool = 5
				if len(out.Messages) != 5 {
					t.Fatalf("got %d messages, want 5", len(out.Messages))
				}
				for i := 2; i <= 4; i++ {
					if out.Messages[i].Role != "tool" {
						t.Errorf("msg[%d].role = %q, want tool", i, out.Messages[i].Role)
					}
				}
				if out.Messages[2].ToolCallID != "c1" {
					t.Errorf("msg[2].tool_call_id = %q", out.Messages[2].ToolCallID)
				}
				if out.Messages[3].ToolCallID != "c2" {
					t.Errorf("msg[3].tool_call_id = %q", out.Messages[3].ToolCallID)
				}
				if out.Messages[4].ToolCallID != "c3" {
					t.Errorf("msg[4].tool_call_id = %q", out.Messages[4].ToolCallID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := AnthropicRequestToOpenAI(&tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tt.wantErr)
			}
			if err == nil && tt.check != nil {
				tt.check(t, out)
			}
		})
	}
}

func intPtr(i int) *int { return &i }
