package translate

import (
	crypto_rand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/bytedance/sonic"
)

// generateMessageID returns a unique Anthropic-style message ID: "msg_" + 24 hex chars.
func generateMessageID() string {
	b := make([]byte, 12)
	crypto_rand.Read(b)
	return "msg_" + hex.EncodeToString(b)
}

// generateToolUseID returns a unique tool use ID: "toolu_" + 24 hex chars.
func generateToolUseID() string {
	b := make([]byte, 12)
	crypto_rand.Read(b)
	return "toolu_" + hex.EncodeToString(b)
}

// mapFinishReason converts an OpenAI finish_reason to an Anthropic stop_reason.
func mapFinishReason(reason *string) string {
	if reason == nil || *reason == "" {
		return "end_turn"
	}
	switch *reason {
	case "stop":
		return "end_turn"
	case "tool_calls":
		return "tool_use"
	case "length":
		return "max_tokens"
	case "content_filter":
		return "end_turn"
	default:
		return "end_turn"
	}
}

// OpenAIResponseToAnthropic translates a non-streaming OpenAI chat completion
// response into an Anthropic Messages API response.
func OpenAIResponseToAnthropic(resp *OpenAIResponse, model string) (*AnthropicResponse, error) {
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai response has no choices")
	}

	choice := resp.Choices[0]
	msg := choice.Message

	var content []ContentBlock

	// Extract text content. Content is interface{} — could be string or nil.
	if msg.Content != nil {
		if s, ok := msg.Content.(string); ok && s != "" {
			content = append(content, ContentBlock{
				Type: "text",
				Text: s,
			})
		}
	}

	// Extract tool calls.
	for _, tc := range msg.ToolCalls {
		id := tc.ID
		if id == "" {
			id = generateToolUseID()
		}

		var input json.RawMessage
		if err := sonic.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
			input = json.RawMessage(`{}`)
		}

		content = append(content, ContentBlock{
			Type:  "tool_use",
			ID:    id,
			Name:  tc.Function.Name,
			Input: input,
		})
	}

	stopReason := mapFinishReason(choice.FinishReason)

	var usage AnthropicUsage
	if resp.Usage != nil {
		usage = AnthropicUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		}
	}

	return &AnthropicResponse{
		ID:           generateMessageID(),
		Type:         "message",
		Role:         "assistant",
		Model:        model,
		Content:      content,
		StopReason:   &stopReason,
		StopSequence: nil,
		Usage:        usage,
	}, nil
}
