package translate

import "time"

// mapAnthropicStopReason converts an Anthropic stop_reason to an OpenAI finish_reason.
func mapAnthropicStopReason(reason *string) *string {
	if reason == nil {
		return nil
	}
	var result string
	switch *reason {
	case "end_turn":
		result = "stop"
	case "tool_use":
		result = "tool_calls"
	case "max_tokens":
		result = "length"
	case "stop_sequence":
		result = "stop"
	default:
		result = "stop"
	}
	return &result
}

// AnthropicResponseToOpenAI translates a non-streaming Anthropic Messages API
// response into an OpenAI chat completion response.
func AnthropicResponseToOpenAI(resp *AnthropicResponse) *OpenAIResponse {
	msg := OpenAIMessage{Role: "assistant"}

	var textParts []string
	var toolCalls []OpenAIToolCall

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			textParts = append(textParts, block.Text)
		case "tool_use":
			args := "{}"
			if len(block.Input) > 0 {
				args = string(block.Input)
			}
			id := block.ID
			if id == "" {
				id = generateToolUseID()
			}
			toolCalls = append(toolCalls, OpenAIToolCall{
				ID:   id,
				Type: "function",
				Function: OpenAIFunction{
					Name:      block.Name,
					Arguments: args,
				},
			})
		}
	}

	if len(textParts) > 0 {
		joined := textParts[0]
		for _, p := range textParts[1:] {
			joined += p
		}
		msg.Content = joined
	}
	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}

	finishReason := mapAnthropicStopReason(resp.StopReason)

	// Convert Anthropic usage to OpenAI usage. Anthropic tracks input_tokens
	// excluding cache reads; OpenAI prompt_tokens includes them.
	totalInput := resp.Usage.InputTokens + resp.Usage.CacheReadInputTokens
	usage := &OpenAIUsage{
		PromptTokens:     totalInput,
		CompletionTokens: resp.Usage.OutputTokens,
		TotalTokens:      totalInput + resp.Usage.OutputTokens,
	}
	if resp.Usage.CacheReadInputTokens > 0 {
		usage.PromptTokensDetails = &OpenAIPromptTokensDetails{
			CachedTokens: resp.Usage.CacheReadInputTokens,
		}
	}

	return &OpenAIResponse{
		ID:      "chatcmpl-" + resp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   resp.Model,
		Choices: []OpenAIChoice{
			{
				Index:        0,
				Message:      msg,
				FinishReason: finishReason,
			},
		},
		Usage: usage,
	}
}
