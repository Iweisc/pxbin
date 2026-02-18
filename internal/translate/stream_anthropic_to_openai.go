package translate

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
)

// AnthropicToOpenAIStreamResult contains usage information captured during
// Anthropic-to-OpenAI streaming translation.
type AnthropicToOpenAIStreamResult struct {
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
	Model               string
}

// TranslateAnthropicStreamToOpenAI reads an Anthropic SSE stream from
// upstreamBody and writes OpenAI-format SSE events to w in real time.
//
// The caller MUST set these response headers before calling:
//
//	Content-Type: text/event-stream
//	Cache-Control: no-cache
//	Connection: keep-alive
func TranslateAnthropicStreamToOpenAI(
	ctx context.Context,
	upstreamBody io.ReadCloser,
	w http.ResponseWriter,
	flusher http.Flusher,
	model string,
) (*AnthropicToOpenAIStreamResult, error) {
	defer upstreamBody.Close()

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			upstreamBody.Close()
		case <-done:
		}
	}()

	result := &AnthropicToOpenAIStreamResult{Model: model}
	chunkID := "chatcmpl-" + generateMessageID()
	created := time.Now().Unix()
	firstChunkSent := false
	toolCallIndex := -1
	currentEventType := ""

	scanner := bufio.NewScanner(upstreamBody)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		if bytes.HasPrefix(line, []byte("event: ")) {
			currentEventType = string(line[7:])
			continue
		}
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := line[6:]

		switch currentEventType {
		case "message_start":
			var evt MessageStartEvent
			if err := sonic.Unmarshal(data, &evt); err != nil {
				continue
			}
			if evt.Message.Model != "" {
				result.Model = evt.Message.Model
				model = evt.Message.Model
			}
			result.InputTokens = evt.Message.Usage.InputTokens
			result.CacheCreationTokens = evt.Message.Usage.CacheCreationInputTokens
			result.CacheReadTokens = evt.Message.Usage.CacheReadInputTokens

			if !firstChunkSent {
				writeOpenAIStreamChunk(w, flusher, chunkID, created, model, &OpenAIStreamChoice{
					Index: 0,
					Delta: OpenAIStreamDelta{Role: "assistant"},
				}, nil)
				firstChunkSent = true
			}

		case "content_block_start":
			var evt ContentBlockStartEvent
			if err := sonic.Unmarshal(data, &evt); err != nil {
				continue
			}
			if !firstChunkSent {
				writeOpenAIStreamChunk(w, flusher, chunkID, created, model, &OpenAIStreamChoice{
					Index: 0,
					Delta: OpenAIStreamDelta{Role: "assistant"},
				}, nil)
				firstChunkSent = true
			}
			if evt.ContentBlock.Type == "tool_use" {
				toolCallIndex++
				writeOpenAIStreamChunk(w, flusher, chunkID, created, model, &OpenAIStreamChoice{
					Index: 0,
					Delta: OpenAIStreamDelta{
						ToolCalls: []OpenAIStreamToolCall{
							{
								Index: toolCallIndex,
								ID:    evt.ContentBlock.ID,
								Type:  "function",
								Function: &OpenAIStreamFunction{
									Name:      evt.ContentBlock.Name,
									Arguments: "",
								},
							},
						},
					},
				}, nil)
			}

		case "content_block_delta":
			var evt ContentBlockDeltaEvent
			if err := sonic.Unmarshal(data, &evt); err != nil {
				continue
			}
			switch evt.Delta.Type {
			case "text_delta":
				text := evt.Delta.Text
				writeOpenAIStreamChunk(w, flusher, chunkID, created, model, &OpenAIStreamChoice{
					Index: 0,
					Delta: OpenAIStreamDelta{Content: &text},
				}, nil)
			case "input_json_delta":
				writeOpenAIStreamChunk(w, flusher, chunkID, created, model, &OpenAIStreamChoice{
					Index: 0,
					Delta: OpenAIStreamDelta{
						ToolCalls: []OpenAIStreamToolCall{
							{
								Index: toolCallIndex,
								Function: &OpenAIStreamFunction{
									Arguments: evt.Delta.PartialJSON,
								},
							},
						},
					},
				}, nil)
			case "thinking_delta":
				text := evt.Delta.Thinking
				writeOpenAIStreamChunk(w, flusher, chunkID, created, model, &OpenAIStreamChoice{
					Index: 0,
					Delta: OpenAIStreamDelta{ReasoningContent: &text},
				}, nil)
			}

		case "message_delta":
			var evt MessageDeltaEvent
			if err := sonic.Unmarshal(data, &evt); err != nil {
				continue
			}
			if evt.Usage != nil {
				result.OutputTokens = evt.Usage.OutputTokens
			}

			finishReason := mapAnthropicStopReason(evt.Delta.StopReason)

			totalInput := result.InputTokens + result.CacheReadTokens
			usage := &OpenAIUsage{
				PromptTokens:     totalInput,
				CompletionTokens: result.OutputTokens,
				TotalTokens:      totalInput + result.OutputTokens,
			}
			if result.CacheReadTokens > 0 {
				usage.PromptTokensDetails = &OpenAIPromptTokensDetails{
					CachedTokens: result.CacheReadTokens,
				}
			}

			writeOpenAIStreamChunk(w, flusher, chunkID, created, model, &OpenAIStreamChoice{
				Index:        0,
				Delta:        OpenAIStreamDelta{},
				FinishReason: finishReason,
			}, usage)

		case "message_stop":
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
		}

		currentEventType = ""
	}

	if err := ctx.Err(); err != nil {
		return result, err
	}
	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("reading upstream SSE stream: %w", err)
	}

	return result, nil
}

func writeOpenAIStreamChunk(w http.ResponseWriter, flusher http.Flusher, id string, created int64, model string, choice *OpenAIStreamChoice, usage *OpenAIUsage) {
	chunk := OpenAIStreamChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   model,
	}
	if choice != nil {
		chunk.Choices = []OpenAIStreamChoice{*choice}
	}
	chunk.Usage = usage

	data, err := sonic.Marshal(chunk)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}
