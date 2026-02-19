package translate

import "encoding/json"

// ---------------------------------------------------------------------------
// OpenAI Responses API types (/v1/responses)
// ---------------------------------------------------------------------------

// ResponsesAPIRequest represents an OpenAI Responses API request.
type ResponsesAPIRequest struct {
	Model           string          `json:"model"`
	Input           json.RawMessage `json:"input"`
	Instructions    string          `json:"instructions,omitempty"`
	MaxOutputTokens *int            `json:"max_output_tokens,omitempty"`
	Temperature     *float64        `json:"temperature,omitempty"`
	TopP            *float64        `json:"top_p,omitempty"`
	Tools           json.RawMessage `json:"tools,omitempty"`
	ToolChoice      json.RawMessage `json:"tool_choice,omitempty"`
	Stream          bool            `json:"stream,omitempty"`
}

// ResponsesAPIResponse is a non-streaming Responses API response.
type ResponsesAPIResponse struct {
	ID     string                `json:"id"`
	Object string                `json:"object"`
	Model  string                `json:"model"`
	Status string                `json:"status"`
	Output []ResponsesOutputItem `json:"output"`
	Usage  ResponsesUsage        `json:"usage"`
}

// ResponsesOutputItem is a discriminated union for output items.
// Type is "message" or "function_call".
type ResponsesOutputItem struct {
	Type string `json:"type"`

	// message fields
	ID      string                 `json:"id,omitempty"`
	Role    string                 `json:"role,omitempty"`
	Content []ResponsesContentPart `json:"content,omitempty"`
	Status  string                 `json:"status,omitempty"`

	// function_call fields
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// ResponsesContentPart is a content part within a message output item.
type ResponsesContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ResponsesUsage contains token usage for a Responses API response.
type ResponsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// ---------------------------------------------------------------------------
// Responses API input item types (for parsing the input array)
// ---------------------------------------------------------------------------

// ResponsesInputItem represents a single item in the input array.
type ResponsesInputItem struct {
	Type    string          `json:"type,omitempty"`
	Role    string          `json:"role,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`

	// function_call fields
	ID        string `json:"id,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`

	// function_call_output fields
	Output string `json:"output,omitempty"`
}

// ResponsesToolDef represents a tool definition in the Responses API format.
// Unlike Chat Completions (which nests under "function"), Responses API tools
// have name/description/parameters at the top level.
type ResponsesToolDef struct {
	Type        string          `json:"type"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// ResponsesInputContentPart is a content part within an input item.
type ResponsesInputContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ---------------------------------------------------------------------------
// Responses API streaming event types
// ---------------------------------------------------------------------------

// ResponsesStreamEvent is the wrapper for all Responses API SSE events.
type ResponsesStreamEvent struct {
	Type     string      `json:"type"`
	Response interface{} `json:"response,omitempty"`

	// For output_item events
	OutputIndex int                  `json:"output_index,omitempty"`
	Item        *ResponsesOutputItem `json:"item,omitempty"`

	// For content_part events
	ContentIndex int                    `json:"content_index,omitempty"`
	Part         *ResponsesContentPart  `json:"part,omitempty"`

	// For delta events
	Delta string `json:"delta,omitempty"`
}
