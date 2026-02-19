package translate

import (
	"fmt"

	"github.com/bytedance/sonic"
)

// ResponsesRequestToChatCompletions translates an OpenAI Responses API request
// into a Chat Completions request suitable for /v1/chat/completions.
func ResponsesRequestToChatCompletions(req *ResponsesAPIRequest) (*OpenAIRequest, error) {
	out := &OpenAIRequest{
		Model: req.Model,
	}

	// --- Instructions → system message ---
	if req.Instructions != "" {
		out.Messages = append(out.Messages, OpenAIMessage{
			Role:    "system",
			Content: req.Instructions,
		})
	}

	// --- Input → messages ---
	msgs, err := translateResponsesInput(req.Input)
	if err != nil {
		return nil, fmt.Errorf("translating input: %w", err)
	}
	out.Messages = append(out.Messages, msgs...)

	// --- Scalars ---
	if req.MaxOutputTokens != nil {
		mt := *req.MaxOutputTokens
		out.MaxCompletionTokens = &mt
	}
	out.Temperature = req.Temperature
	out.TopP = req.TopP

	// --- Tools ---
	// Responses API tools have name/description/parameters at top level;
	// Chat Completions nests them under "function". Only function-type
	// tools are translatable; skip web_search, file_search, etc.
	if len(req.Tools) > 0 {
		var rTools []ResponsesToolDef
		if err := sonic.Unmarshal(req.Tools, &rTools); err == nil {
			for _, rt := range rTools {
				if rt.Type != "function" || rt.Name == "" {
					continue
				}
				out.Tools = append(out.Tools, OpenAITool{
					Type: "function",
					Function: OpenAIFunctionDef{
						Name:        rt.Name,
						Description: rt.Description,
						Parameters:  rt.Parameters,
					},
				})
			}
		}
	}
	if len(req.ToolChoice) > 0 {
		var tc interface{}
		if err := sonic.Unmarshal(req.ToolChoice, &tc); err == nil {
			out.ToolChoice = tc
		}
	}

	// --- Streaming ---
	if req.Stream {
		out.Stream = true
		out.StreamOptions = &StreamOptions{IncludeUsage: true}
	}

	return out, nil
}

// translateResponsesInput converts the Responses API input field into OpenAI
// messages. The input can be a plain string or an array of input items.
func translateResponsesInput(raw []byte) ([]OpenAIMessage, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	// Try as a plain string first.
	var s string
	if err := sonic.Unmarshal(raw, &s); err == nil {
		return []OpenAIMessage{{Role: "user", Content: s}}, nil
	}

	// Parse as array of input items.
	var items []ResponsesInputItem
	if err := sonic.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("input is neither a string nor an array: %w", err)
	}

	var msgs []OpenAIMessage
	for _, item := range items {
		switch item.Type {
		case "message", "":
			msg, err := translateResponsesInputMessage(item)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, msg)
		case "function_call":
			// In Chat Completions, tool_calls belong on the assistant message.
			// Merge into the preceding assistant message if one exists.
			tc := OpenAIToolCall{
				ID:   item.CallID,
				Type: "function",
				Function: OpenAIFunction{
					Name:      item.Name,
					Arguments: item.Arguments,
				},
			}
			if n := len(msgs); n > 0 && msgs[n-1].Role == "assistant" {
				msgs[n-1].ToolCalls = append(msgs[n-1].ToolCalls, tc)
			} else {
				msgs = append(msgs, OpenAIMessage{
					Role:      "assistant",
					ToolCalls: []OpenAIToolCall{tc},
				})
			}
		case "function_call_output":
			msgs = append(msgs, OpenAIMessage{
				Role:       "tool",
				ToolCallID: item.CallID,
				Content:    item.Output,
			})
		default:
			if item.Role != "" {
				msg, err := translateResponsesInputMessage(item)
				if err != nil {
					return nil, err
				}
				msgs = append(msgs, msg)
			}
		}
	}
	return msgs, nil
}

// translateResponsesInputMessage translates a single message-type input item.
func translateResponsesInputMessage(item ResponsesInputItem) (OpenAIMessage, error) {
	role := item.Role
	if role == "" {
		role = "user"
	}

	if len(item.Content) == 0 {
		return OpenAIMessage{Role: role}, nil
	}

	// Try as string content.
	var s string
	if err := sonic.Unmarshal(item.Content, &s); err == nil {
		return OpenAIMessage{Role: role, Content: s}, nil
	}

	// Try as array of content parts.
	var parts []ResponsesInputContentPart
	if err := sonic.Unmarshal(item.Content, &parts); err != nil {
		return OpenAIMessage{}, fmt.Errorf("message content is neither string nor array: %w", err)
	}

	// Build OpenAI content parts. Map all Responses API text part types
	// (input_text, output_text, text) to a plain OpenAI text part.
	var oaiParts []OpenAIContentPart
	for _, p := range parts {
		switch p.Type {
		case "input_text", "output_text", "text":
			oaiParts = append(oaiParts, OpenAIContentPart{Type: "text", Text: p.Text})
		}
	}

	if len(oaiParts) == 1 && oaiParts[0].Type == "text" {
		return OpenAIMessage{Role: role, Content: oaiParts[0].Text}, nil
	}

	return OpenAIMessage{Role: role, Content: oaiParts}, nil
}
