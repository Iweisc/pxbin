package translate

import "encoding/json"

// ---------------------------------------------------------------------------
// Anthropic API types
// ---------------------------------------------------------------------------

// AnthropicRequest represents a native Anthropic /v1/messages request.
type AnthropicRequest struct {
	Model         string             `json:"model"`
	MaxTokens     int                `json:"max_tokens"`
	Messages      []AnthropicMessage `json:"messages"`
	System        json.RawMessage    `json:"system,omitempty"`
	Tools         []AnthropicTool    `json:"tools,omitempty"`
	ToolChoice    json.RawMessage    `json:"tool_choice,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
	Temperature   *float64           `json:"temperature,omitempty"`
	TopP          *float64           `json:"top_p,omitempty"`
	TopK          *int               `json:"top_k,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
	Thinking      *ThinkingConfig    `json:"thinking,omitempty"`
	Metadata      *Metadata          `json:"metadata,omitempty"`
}

// ThinkingConfig controls extended thinking behaviour.
type ThinkingConfig struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

// Metadata carries optional request metadata.
type Metadata struct {
	UserID string `json:"user_id,omitempty"`
}

// AnthropicMessage represents a single message in a conversation.
// Content can be a plain string or an array of ContentBlock.
type AnthropicMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// ContentAsString returns the content as a plain string if it is one.
func (m *AnthropicMessage) ContentAsString() (string, bool) {
	var s string
	if err := json.Unmarshal(m.Content, &s); err == nil {
		return s, true
	}
	return "", false
}

// ContentAsBlocks parses the content as an array of ContentBlock.
func (m *AnthropicMessage) ContentAsBlocks() ([]ContentBlock, error) {
	var blocks []ContentBlock
	if err := json.Unmarshal(m.Content, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

// ContentBlock is a discriminated union for Anthropic content blocks.
// The Type field determines which subset of fields is populated.
type ContentBlock struct {
	Type string `json:"type"`

	// TextBlock fields
	Text string `json:"text,omitempty"`

	// ImageBlock fields
	Source *ImageSource `json:"source,omitempty"`

	// ToolUseBlock fields
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// ToolResultBlock fields
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"` // string or []ContentBlock
	IsError   bool            `json:"is_error,omitempty"`

	// ThinkingBlock fields
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`

	// Cache control
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// ImageSource describes the source of an image in a content block.
type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
	URL       string `json:"url,omitempty"`
}

// CacheControl carries cache control hints for prompt caching.
type CacheControl struct {
	Type string `json:"type"`
}

// SystemBlock is a structured system prompt block.
type SystemBlock struct {
	Type         string        `json:"type"`
	Text         string        `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// AnthropicTool describes a tool available to the model.
type AnthropicTool struct {
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	InputSchema  json.RawMessage `json:"input_schema"`
	Type         string          `json:"type,omitempty"`
	CacheControl *CacheControl   `json:"cache_control,omitempty"`
}

// ToolChoiceObj specifies how the model should pick tools.
type ToolChoiceObj struct {
	Type                   string `json:"type"`
	Name                   string `json:"name,omitempty"`
	DisableParallelToolUse bool   `json:"disable_parallel_tool_use,omitempty"`
}

// AnthropicResponse is the non-streaming response from the Anthropic API.
type AnthropicResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Model        string         `json:"model"`
	Content      []ContentBlock `json:"content"`
	StopReason   *string        `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence"`
	Usage        AnthropicUsage `json:"usage"`
}

// AnthropicUsage contains token usage information.
type AnthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// ---------------------------------------------------------------------------
// Anthropic streaming event types
// ---------------------------------------------------------------------------

// MessageStartEvent is emitted at the beginning of a streamed response.
type MessageStartEvent struct {
	Type    string            `json:"type"`
	Message AnthropicResponse `json:"message"`
}

// ContentBlockStartEvent signals the start of a new content block.
type ContentBlockStartEvent struct {
	Type         string       `json:"type"`
	Index        int          `json:"index"`
	ContentBlock ContentBlock `json:"content_block"`
}

// ContentBlockDeltaEvent carries an incremental update to a content block.
type ContentBlockDeltaEvent struct {
	Type  string     `json:"type"`
	Index int        `json:"index"`
	Delta DeltaBlock `json:"delta"`
}

// DeltaBlock is the payload inside a ContentBlockDeltaEvent.
type DeltaBlock struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	Thinking    string `json:"thinking,omitempty"`
	Signature   string `json:"signature,omitempty"`
}

// ContentBlockStopEvent signals the end of a content block.
type ContentBlockStopEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

// MessageDeltaEvent carries final metadata for the message.
type MessageDeltaEvent struct {
	Type  string             `json:"type"`
	Delta MessageDelta       `json:"delta"`
	Usage *MessageDeltaUsage `json:"usage"`
}

// MessageDelta contains the stop reason and stop sequence.
type MessageDelta struct {
	StopReason   *string `json:"stop_reason"`
	StopSequence *string `json:"stop_sequence"`
}

// MessageDeltaUsage contains token counts at the end of streaming.
type MessageDeltaUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// MessageStopEvent signals the end of a streamed message.
type MessageStopEvent struct {
	Type string `json:"type"`
}

// PingEvent is a keep-alive event in the stream.
type PingEvent struct {
	Type string `json:"type"`
}

// AnthropicErrorResponse wraps an error from the Anthropic API.
type AnthropicErrorResponse struct {
	Type  string         `json:"type"`
	Error AnthropicError `json:"error"`
}

// AnthropicError describes an individual Anthropic API error.
type AnthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// ---------------------------------------------------------------------------
// OpenAI API types
// ---------------------------------------------------------------------------

// OpenAIRequest represents an OpenAI /v1/chat/completions request.
type OpenAIRequest struct {
	Model               string          `json:"model"`
	Messages            []OpenAIMessage `json:"messages"`
	Tools               []OpenAITool    `json:"tools,omitempty"`
	ToolChoice          interface{}     `json:"tool_choice,omitempty"`
	MaxTokens           *int            `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int            `json:"max_completion_tokens,omitempty"`
	Temperature         *float64        `json:"temperature,omitempty"`
	TopP                *float64        `json:"top_p,omitempty"`
	Stop                interface{}     `json:"stop,omitempty"`
	Stream              bool            `json:"stream,omitempty"`
	StreamOptions       *StreamOptions  `json:"stream_options,omitempty"`
	User                string          `json:"user,omitempty"`
	ReasoningEffort     string          `json:"reasoning_effort,omitempty"`
}

// StreamOptions controls streaming behaviour for OpenAI requests.
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// OpenAIMessage represents a single message in an OpenAI conversation.
type OpenAIMessage struct {
	Role       string           `json:"role"`
	Content    interface{}      `json:"content,omitempty"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	Name       string           `json:"name,omitempty"`
}

// OpenAIContentPart is a multimodal content part (text or image).
type OpenAIContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL references an image by URL for OpenAI vision requests.
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// OpenAIToolCall represents a tool call made by the model.
type OpenAIToolCall struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Function OpenAIFunction `json:"function"`
}

// OpenAIFunction is the function name and arguments in a tool call.
type OpenAIFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OpenAITool describes a tool available to the OpenAI model.
type OpenAITool struct {
	Type     string            `json:"type"`
	Function OpenAIFunctionDef `json:"function"`
}

// OpenAIFunctionDef is the definition of a function tool.
type OpenAIFunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// OpenAIResponse is the non-streaming response from the OpenAI API.
type OpenAIResponse struct {
	ID                string         `json:"id"`
	Object            string         `json:"object"`
	Created           int64          `json:"created"`
	Model             string         `json:"model"`
	Choices           []OpenAIChoice `json:"choices"`
	Usage             *OpenAIUsage   `json:"usage,omitempty"`
	SystemFingerprint string         `json:"system_fingerprint,omitempty"`
}

// OpenAIChoice is a single completion choice.
type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason *string       `json:"finish_reason"`
}

// OpenAIUsage contains token usage information for an OpenAI response.
type OpenAIUsage struct {
	PromptTokens        int                       `json:"prompt_tokens"`
	CompletionTokens    int                       `json:"completion_tokens"`
	TotalTokens         int                       `json:"total_tokens"`
	PromptTokensDetails *OpenAIPromptTokensDetails `json:"prompt_tokens_details,omitempty"`
}

type OpenAIPromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
}

// ---------------------------------------------------------------------------
// OpenAI streaming types
// ---------------------------------------------------------------------------

// OpenAIStreamChunk is a single chunk in a streamed OpenAI response.
type OpenAIStreamChunk struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []OpenAIStreamChoice `json:"choices"`
	Usage   *OpenAIUsage         `json:"usage,omitempty"`
}

// OpenAIStreamChoice is a choice within a stream chunk.
type OpenAIStreamChoice struct {
	Index        int               `json:"index"`
	Delta        OpenAIStreamDelta `json:"delta"`
	FinishReason *string           `json:"finish_reason"`
}

// OpenAIStreamDelta carries incremental content in a stream chunk.
type OpenAIStreamDelta struct {
	Role             string                 `json:"role,omitempty"`
	Content          *string                `json:"content,omitempty"`
	ReasoningContent *string                `json:"reasoning_content,omitempty"`
	ToolCalls        []OpenAIStreamToolCall `json:"tool_calls,omitempty"`
}

// OpenAIStreamToolCall is a tool call delta within a stream chunk.
type OpenAIStreamToolCall struct {
	Index    int                  `json:"index"`
	ID       string               `json:"id,omitempty"`
	Type     string               `json:"type,omitempty"`
	Function *OpenAIStreamFunction `json:"function,omitempty"`
}

// OpenAIStreamFunction carries incremental function call data in a stream.
type OpenAIStreamFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// OpenAIErrorResponse wraps an error from the OpenAI API.
type OpenAIErrorResponse struct {
	Error OpenAIError `json:"error"`
}

// OpenAIError describes an individual OpenAI API error.
type OpenAIError struct {
	Message string  `json:"message"`
	Type    string  `json:"type"`
	Param   *string `json:"param"`
	Code    *string `json:"code"`
}

// OpenAIToolChoiceFunction specifies a particular function for tool_choice.
type OpenAIToolChoiceFunction struct {
	Type     string                   `json:"type"`
	Function OpenAIToolChoiceFuncName `json:"function"`
}

// OpenAIToolChoiceFuncName names the function in a tool choice object.
type OpenAIToolChoiceFuncName struct {
	Name string `json:"name"`
}
