package translate

import (
	crypto_rand "crypto/rand"
	"encoding/hex"
)

// generateResponseID returns a unique Responses API response ID: "resp_" + 24 hex chars.
func generateResponseID() string {
	b := make([]byte, 12)
	crypto_rand.Read(b)
	return "resp_" + hex.EncodeToString(b)
}

// ChatCompletionsToResponsesAPI translates a Chat Completions response into a
// Responses API response.
func ChatCompletionsToResponsesAPI(resp *OpenAIResponse, model string) *ResponsesAPIResponse {
	out := &ResponsesAPIResponse{
		ID:     generateResponseID(),
		Object: "response",
		Model:  model,
		Status: "completed",
	}

	if resp.Model != "" {
		out.Model = resp.Model
	}

	if len(resp.Choices) > 0 {
		msg := resp.Choices[0].Message
		out.Output = translateChatMessageToOutputItems(msg)
	}

	if resp.Usage != nil {
		out.Usage = ResponsesUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		}
	}

	return out
}

// translateChatMessageToOutputItems converts a Chat Completions message into
// Responses API output items.
func translateChatMessageToOutputItems(msg OpenAIMessage) []ResponsesOutputItem {
	var items []ResponsesOutputItem

	// Text content → message output item.
	if text := messageContentAsString(msg.Content); text != "" {
		items = append(items, ResponsesOutputItem{
			Type:   "message",
			ID:     generateMessageID(),
			Role:   "assistant",
			Status: "completed",
			Content: []ResponsesContentPart{{
				Type: "output_text",
				Text: text,
			}},
		})
	}

	// Tool calls → function_call output items.
	for _, tc := range msg.ToolCalls {
		items = append(items, ResponsesOutputItem{
			Type:      "function_call",
			ID:        tc.ID,
			CallID:    tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
			Status:    "completed",
		})
	}

	return items
}

// messageContentAsString extracts a string from an OpenAIMessage Content field,
// which may be a string or []OpenAIContentPart.
func messageContentAsString(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case nil:
		return ""
	}
	return ""
}
