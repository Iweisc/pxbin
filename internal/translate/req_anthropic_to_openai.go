package translate

import (
	"encoding/json"

	"github.com/bytedance/sonic"
	"fmt"
	"strings"
)

// AnthropicRequestToOpenAI translates a native Anthropic /v1/messages request
// into an OpenAI /v1/chat/completions request.
func AnthropicRequestToOpenAI(req *AnthropicRequest) (*OpenAIRequest, error) {
	out := &OpenAIRequest{
		Model: req.Model,
	}

	// --- System prompt ---
	if len(req.System) > 0 {
		sysMsg, err := translateSystem(req.System)
		if err != nil {
			return nil, fmt.Errorf("translating system prompt: %w", err)
		}
		if sysMsg != nil {
			out.Messages = append(out.Messages, *sysMsg)
		}
	}

	// --- Messages ---
	for i, msg := range req.Messages {
		translated, err := translateMessage(msg)
		if err != nil {
			return nil, fmt.Errorf("translating message %d: %w", i, err)
		}
		out.Messages = append(out.Messages, translated...)
	}

	// --- Tools ---
	for _, t := range req.Tools {
		if t.Type != "" && t.Type != "custom" {
			continue
		}
		out.Tools = append(out.Tools, OpenAITool{
			Type: "function",
			Function: OpenAIFunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	// --- Tool choice ---
	if len(req.ToolChoice) > 0 {
		tc, err := translateToolChoice(req.ToolChoice)
		if err != nil {
			return nil, fmt.Errorf("translating tool_choice: %w", err)
		}
		out.ToolChoice = tc
	}

	// --- Scalars ---
	if req.MaxTokens > 0 {
		mt := req.MaxTokens
		out.MaxTokens = &mt
	}
	if len(req.StopSequences) > 0 {
		out.Stop = req.StopSequences
	}
	out.Temperature = req.Temperature
	out.TopP = req.TopP
	// top_k has no OpenAI equivalent — omit

	// --- Streaming ---
	if req.Stream {
		out.Stream = true
		out.StreamOptions = &StreamOptions{IncludeUsage: true}
	}

	// --- Thinking / Extended reasoning ---
	if req.Thinking != nil && req.Thinking.Type == "enabled" {
		out.ReasoningEffort = "high"
		// When thinking is enabled, use max_completion_tokens instead of max_tokens
		// to account for both thinking + output tokens
		if req.Thinking.BudgetTokens > 0 {
			budget := req.Thinking.BudgetTokens
			if out.MaxTokens != nil {
				total := budget + *out.MaxTokens
				out.MaxCompletionTokens = &total
			} else {
				out.MaxCompletionTokens = &budget
			}
			out.MaxTokens = nil
		}
	}

	// --- Metadata ---
	if req.Metadata != nil && req.Metadata.UserID != "" {
		out.User = req.Metadata.UserID
	}

	return out, nil
}

// translateSystem parses the Anthropic system field (string or []SystemBlock)
// and returns an OpenAI system message.
func translateSystem(raw json.RawMessage) (*OpenAIMessage, error) {
	// Try as a plain string first.
	var s string
	if err := sonic.Unmarshal(raw, &s); err == nil {
		if s == "" {
			return nil, nil
		}
		return &OpenAIMessage{Role: "system", Content: s}, nil
	}

	// Try as an array of SystemBlock.
	var blocks []SystemBlock
	if err := sonic.Unmarshal(raw, &blocks); err != nil {
		return nil, fmt.Errorf("system is neither a string nor an array of blocks: %w", err)
	}
	if len(blocks) == 0 {
		return nil, nil
	}

	var parts []string
	for _, b := range blocks {
		if b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	if len(parts) == 0 {
		return nil, nil
	}
	return &OpenAIMessage{Role: "system", Content: strings.Join(parts, "\n\n")}, nil
}

// translateMessage converts a single Anthropic message into one or more OpenAI
// messages (tool_result blocks expand into separate tool messages).
func translateMessage(msg AnthropicMessage) ([]OpenAIMessage, error) {
	switch msg.Role {
	case "user":
		return translateUserMessage(msg)
	case "assistant":
		return translateAssistantMessage(msg)
	default:
		return nil, fmt.Errorf("unsupported message role: %q", msg.Role)
	}
}

func translateUserMessage(msg AnthropicMessage) ([]OpenAIMessage, error) {
	// Simple string content.
	if s, ok := msg.ContentAsString(); ok {
		return []OpenAIMessage{{Role: "user", Content: s}}, nil
	}

	blocks, err := msg.ContentAsBlocks()
	if err != nil {
		return nil, fmt.Errorf("parsing user content blocks: %w", err)
	}

	var toolMsgs []OpenAIMessage
	var contentParts []OpenAIContentPart

	for _, b := range blocks {
		switch b.Type {
		case "text":
			contentParts = append(contentParts, OpenAIContentPart{
				Type: "text",
				Text: b.Text,
			})
		case "image":
			part, err := translateImageBlock(b)
			if err != nil {
				return nil, err
			}
			contentParts = append(contentParts, part)
		case "tool_result":
			toolMsg, err := translateToolResult(b)
			if err != nil {
				return nil, err
			}
			toolMsgs = append(toolMsgs, toolMsg)
		}
	}

	var result []OpenAIMessage
	// Tool messages come first.
	result = append(result, toolMsgs...)

	// If there are remaining content parts, add a user message.
	if len(contentParts) > 0 {
		result = append(result, OpenAIMessage{
			Role:    "user",
			Content: contentParts,
		})
	}

	return result, nil
}

func translateImageBlock(b ContentBlock) (OpenAIContentPart, error) {
	if b.Source == nil {
		return OpenAIContentPart{}, fmt.Errorf("image block has no source")
	}
	var url string
	switch b.Source.Type {
	case "base64":
		url = fmt.Sprintf("data:%s;base64,%s", b.Source.MediaType, b.Source.Data)
	case "url":
		url = b.Source.URL
	default:
		return OpenAIContentPart{}, fmt.Errorf("unsupported image source type: %q", b.Source.Type)
	}
	return OpenAIContentPart{
		Type:     "image_url",
		ImageURL: &ImageURL{URL: url},
	}, nil
}

func translateToolResult(b ContentBlock) (OpenAIMessage, error) {
	content, err := toolResultContent(b.Content)
	if err != nil {
		return OpenAIMessage{}, fmt.Errorf("parsing tool_result content: %w", err)
	}
	return OpenAIMessage{
		Role:       "tool",
		ToolCallID: b.ToolUseID,
		Content:    content,
	}, nil
}

// toolResultContent parses tool_result content which can be a string or
// an array of content blocks (text blocks are concatenated).
func toolResultContent(raw json.RawMessage) (interface{}, error) {
	if len(raw) == 0 {
		return "", nil
	}

	// Try as string.
	var s string
	if err := sonic.Unmarshal(raw, &s); err == nil {
		return s, nil
	}

	// Try as array of ContentBlock.
	var blocks []ContentBlock
	if err := sonic.Unmarshal(raw, &blocks); err != nil {
		return nil, fmt.Errorf("tool_result content is neither string nor array: %w", err)
	}

	var texts []string
	for _, cb := range blocks {
		if cb.Type == "text" {
			texts = append(texts, cb.Text)
		}
	}
	return strings.Join(texts, ""), nil
}

func translateAssistantMessage(msg AnthropicMessage) ([]OpenAIMessage, error) {
	// Simple string content.
	if s, ok := msg.ContentAsString(); ok {
		return []OpenAIMessage{{Role: "assistant", Content: s}}, nil
	}

	blocks, err := msg.ContentAsBlocks()
	if err != nil {
		return nil, fmt.Errorf("parsing assistant content blocks: %w", err)
	}

	oMsg := OpenAIMessage{Role: "assistant"}

	var textParts []string
	var toolCalls []OpenAIToolCall

	for _, b := range blocks {
		switch b.Type {
		case "text":
			textParts = append(textParts, b.Text)
		case "tool_use":
			args, err := marshalToolInput(b.Input)
			if err != nil {
				return nil, fmt.Errorf("marshaling tool_use input: %w", err)
			}
			toolCalls = append(toolCalls, OpenAIToolCall{
				ID:   b.ID,
				Type: "function",
				Function: OpenAIFunction{
					Name:      b.Name,
					Arguments: args,
				},
			})
		case "thinking":
			// Skip — no OpenAI equivalent.
		}
	}

	if len(textParts) > 0 {
		oMsg.Content = strings.Join(textParts, "")
	}
	if len(toolCalls) > 0 {
		oMsg.ToolCalls = toolCalls
	}

	return []OpenAIMessage{oMsg}, nil
}

// marshalToolInput serializes tool_use input to a JSON string.
// An empty or nil input becomes "{}".
func marshalToolInput(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "{}", nil
	}
	// Re-marshal to get a compact canonical form.
	var v interface{}
	if err := sonic.Unmarshal(raw, &v); err != nil {
		return "", err
	}
	b, err := sonic.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// translateToolChoice converts Anthropic tool_choice to OpenAI tool_choice.
func translateToolChoice(raw json.RawMessage) (interface{}, error) {
	// Try as a plain string first.
	var s string
	if err := sonic.Unmarshal(raw, &s); err == nil {
		switch s {
		case "auto":
			return "auto", nil
		case "any":
			return "required", nil
		case "none":
			return "none", nil
		default:
			return s, nil
		}
	}

	// Try as an object.
	var obj ToolChoiceObj
	if err := sonic.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("tool_choice is neither string nor object: %w", err)
	}

	switch obj.Type {
	case "auto":
		return "auto", nil
	case "any":
		return "required", nil
	case "tool":
		return OpenAIToolChoiceFunction{
			Type:     "function",
			Function: OpenAIToolChoiceFuncName{Name: obj.Name},
		}, nil
	default:
		return obj.Type, nil
	}
}
