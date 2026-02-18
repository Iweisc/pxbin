package translate

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
)

// OpenAIRequestToAnthropic translates an OpenAI /v1/chat/completions request
// into an Anthropic /v1/messages request.
func OpenAIRequestToAnthropic(req *OpenAIRequest) (*AnthropicRequest, error) {
	out := &AnthropicRequest{
		Model: req.Model,
	}

	// --- System prompt: extract system role messages ---
	var systemParts []string
	var nonSystemMsgs []OpenAIMessage
	for _, msg := range req.Messages {
		if msg.Role == "system" || msg.Role == "developer" {
			text := extractOpenAIMessageText(msg)
			if text != "" {
				systemParts = append(systemParts, text)
			}
		} else {
			nonSystemMsgs = append(nonSystemMsgs, msg)
		}
	}
	if len(systemParts) > 0 {
		systemStr := strings.Join(systemParts, "\n\n")
		raw, _ := sonic.Marshal(systemStr)
		out.System = json.RawMessage(raw)
	}

	// --- Messages ---
	// OpenAI has separate "tool" role messages while Anthropic puts tool_result
	// blocks inside user messages. Consecutive tool messages are grouped into a
	// single user message.
	for i := 0; i < len(nonSystemMsgs); i++ {
		msg := nonSystemMsgs[i]
		switch msg.Role {
		case "user":
			anthropicMsg, err := translateOpenAIUserToAnthropic(msg)
			if err != nil {
				return nil, fmt.Errorf("translating user message %d: %w", i, err)
			}
			out.Messages = append(out.Messages, anthropicMsg)
		case "assistant":
			anthropicMsg, err := translateOpenAIAssistantToAnthropic(msg)
			if err != nil {
				return nil, fmt.Errorf("translating assistant message %d: %w", i, err)
			}
			out.Messages = append(out.Messages, anthropicMsg)
		case "tool":
			// Collect consecutive tool messages into tool_result blocks.
			var toolResults []ContentBlock
			for i < len(nonSystemMsgs) && nonSystemMsgs[i].Role == "tool" {
				toolMsg := nonSystemMsgs[i]
				content := extractOpenAIMessageText(toolMsg)
				raw, _ := sonic.Marshal(content)
				toolResults = append(toolResults, ContentBlock{
					Type:      "tool_result",
					ToolUseID: toolMsg.ToolCallID,
					Content:   json.RawMessage(raw),
				})
				i++
			}
			i-- // outer loop will increment
			blocks, _ := sonic.Marshal(toolResults)
			out.Messages = append(out.Messages, AnthropicMessage{
				Role:    "user",
				Content: json.RawMessage(blocks),
			})
		default:
			return nil, fmt.Errorf("unsupported message role: %q", msg.Role)
		}
	}

	// Post-process: merge consecutive user messages (Anthropic requires
	// alternating user/assistant roles).
	out.Messages = mergeConsecutiveUserMessages(out.Messages)

	// --- Tools ---
	for _, t := range req.Tools {
		if t.Type != "function" {
			continue
		}
		out.Tools = append(out.Tools, AnthropicTool{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			InputSchema: t.Function.Parameters,
		})
	}

	// --- Tool choice ---
	if req.ToolChoice != nil {
		tc, err := translateOpenAIToolChoiceToAnthropic(req.ToolChoice)
		if err != nil {
			return nil, fmt.Errorf("translating tool_choice: %w", err)
		}
		out.ToolChoice = tc
	}

	// --- Scalars ---
	if req.MaxTokens != nil {
		out.MaxTokens = *req.MaxTokens
	} else if req.MaxCompletionTokens != nil {
		out.MaxTokens = *req.MaxCompletionTokens
	} else {
		out.MaxTokens = 8192 // Anthropic requires max_tokens
	}

	out.Temperature = req.Temperature
	out.TopP = req.TopP

	if req.Stop != nil {
		switch v := req.Stop.(type) {
		case string:
			if v != "" {
				out.StopSequences = []string{v}
			}
		case []interface{}:
			for _, s := range v {
				if str, ok := s.(string); ok {
					out.StopSequences = append(out.StopSequences, str)
				}
			}
		}
	}

	// --- Streaming ---
	out.Stream = req.Stream

	// --- Thinking / Reasoning ---
	if req.ReasoningEffort != "" {
		budgetTokens := 10000
		switch req.ReasoningEffort {
		case "low":
			budgetTokens = 5000
		case "medium":
			budgetTokens = 10000
		case "high":
			budgetTokens = 20000
		}
		out.Thinking = &ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: budgetTokens,
		}
	}

	// --- Metadata ---
	if req.User != "" {
		out.Metadata = &Metadata{UserID: req.User}
	}

	return out, nil
}

// extractOpenAIMessageText extracts text content from an OpenAI message.
// Content can be a string or an array of content parts.
func extractOpenAIMessageText(msg OpenAIMessage) string {
	if msg.Content == nil {
		return ""
	}
	switch v := msg.Content.(type) {
	case string:
		return v
	case []interface{}:
		var parts []string
		for _, part := range v {
			if m, ok := part.(map[string]interface{}); ok {
				if t, _ := m["type"].(string); t == "text" {
					if text, ok := m["text"].(string); ok {
						parts = append(parts, text)
					}
				}
			}
		}
		return strings.Join(parts, "")
	}
	return ""
}

func translateOpenAIUserToAnthropic(msg OpenAIMessage) (AnthropicMessage, error) {
	// Simple string content.
	if s, ok := msg.Content.(string); ok {
		raw, _ := sonic.Marshal(s)
		return AnthropicMessage{Role: "user", Content: json.RawMessage(raw)}, nil
	}

	// Array of content parts.
	if contentParts, ok := msg.Content.([]interface{}); ok {
		var blocks []ContentBlock
		for _, part := range contentParts {
			m, ok := part.(map[string]interface{})
			if !ok {
				continue
			}
			partType, _ := m["type"].(string)
			switch partType {
			case "text":
				text, _ := m["text"].(string)
				blocks = append(blocks, ContentBlock{Type: "text", Text: text})
			case "image_url":
				if imgURL, ok := m["image_url"].(map[string]interface{}); ok {
					url, _ := imgURL["url"].(string)
					if strings.HasPrefix(url, "data:") {
						// Parse data URI: data:media/type;base64,DATA
						uriParts := strings.SplitN(url, ";base64,", 2)
						if len(uriParts) == 2 {
							mediaType := strings.TrimPrefix(uriParts[0], "data:")
							blocks = append(blocks, ContentBlock{
								Type: "image",
								Source: &ImageSource{
									Type:      "base64",
									MediaType: mediaType,
									Data:      uriParts[1],
								},
							})
						}
					} else {
						blocks = append(blocks, ContentBlock{
							Type: "image",
							Source: &ImageSource{
								Type: "url",
								URL:  url,
							},
						})
					}
				}
			}
		}
		raw, _ := sonic.Marshal(blocks)
		return AnthropicMessage{Role: "user", Content: json.RawMessage(raw)}, nil
	}

	// Fallback: empty string content.
	return AnthropicMessage{Role: "user", Content: json.RawMessage(`""`)}, nil
}

func translateOpenAIAssistantToAnthropic(msg OpenAIMessage) (AnthropicMessage, error) {
	var blocks []ContentBlock

	// Text content.
	text := extractOpenAIMessageText(msg)
	if text != "" {
		blocks = append(blocks, ContentBlock{Type: "text", Text: text})
	}

	// Tool calls.
	for _, tc := range msg.ToolCalls {
		var input json.RawMessage
		if tc.Function.Arguments != "" {
			if err := sonic.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
				input = json.RawMessage(`{}`)
			}
		} else {
			input = json.RawMessage(`{}`)
		}
		blocks = append(blocks, ContentBlock{
			Type:  "tool_use",
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: input,
		})
	}

	if len(blocks) == 0 {
		raw, _ := sonic.Marshal("")
		return AnthropicMessage{Role: "assistant", Content: json.RawMessage(raw)}, nil
	}

	raw, _ := sonic.Marshal(blocks)
	return AnthropicMessage{Role: "assistant", Content: json.RawMessage(raw)}, nil
}

func translateOpenAIToolChoiceToAnthropic(tc interface{}) (json.RawMessage, error) {
	switch v := tc.(type) {
	case string:
		switch v {
		case "auto":
			raw, _ := sonic.Marshal(ToolChoiceObj{Type: "auto"})
			return json.RawMessage(raw), nil
		case "required":
			raw, _ := sonic.Marshal(ToolChoiceObj{Type: "any"})
			return json.RawMessage(raw), nil
		case "none":
			return nil, nil
		default:
			raw, _ := sonic.Marshal(v)
			return json.RawMessage(raw), nil
		}
	case map[string]interface{}:
		if t, _ := v["type"].(string); t == "function" {
			if fn, ok := v["function"].(map[string]interface{}); ok {
				name, _ := fn["name"].(string)
				raw, _ := sonic.Marshal(ToolChoiceObj{Type: "tool", Name: name})
				return json.RawMessage(raw), nil
			}
		}
		return nil, nil
	}
	return nil, nil
}

// ensureBlocksContent normalises Anthropic message content into an array of
// ContentBlock, regardless of whether it was stored as a plain string.
func ensureBlocksContent(content json.RawMessage) []ContentBlock {
	var s string
	if err := sonic.Unmarshal(content, &s); err == nil {
		return []ContentBlock{{Type: "text", Text: s}}
	}
	var blocks []ContentBlock
	if err := sonic.Unmarshal(content, &blocks); err == nil {
		return blocks
	}
	return nil
}

// mergeConsecutiveUserMessages combines adjacent user messages into a single
// message. Anthropic requires strictly alternating user/assistant roles.
func mergeConsecutiveUserMessages(msgs []AnthropicMessage) []AnthropicMessage {
	if len(msgs) <= 1 {
		return msgs
	}

	var result []AnthropicMessage
	for _, msg := range msgs {
		if len(result) > 0 && msg.Role == "user" && result[len(result)-1].Role == "user" {
			prev := &result[len(result)-1]
			prevBlocks := ensureBlocksContent(prev.Content)
			currBlocks := ensureBlocksContent(msg.Content)
			merged := append(prevBlocks, currBlocks...)
			raw, _ := sonic.Marshal(merged)
			prev.Content = json.RawMessage(raw)
		} else {
			result = append(result, msg)
		}
	}
	return result
}
