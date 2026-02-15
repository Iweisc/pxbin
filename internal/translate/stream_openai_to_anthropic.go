package translate

import (
	"bufio"
	"context"
	"encoding/json"

	"fmt"
	"github.com/bytedance/sonic"
	"io"
	"net/http"
	"strings"
)

// toolCallState tracks the in-progress state of a single tool call within a
// streaming translation. OpenAI delivers tool calls incrementally across many
// chunks: the first chunk carries the ID and function name while subsequent
// chunks append to the arguments string.
type toolCallState struct {
	anthropicIndex int
	id             string
	name           string
	argsBuffer     strings.Builder
}

// streamState holds all mutable state for the OpenAI-to-Anthropic SSE
// translation state machine.
type streamState struct {
	messageStartSent  bool
	currentBlockIndex int
	currentBlockType  string // "" | "text" | "tool_use"
	toolCalls         map[int]*toolCallState
	finishReason      *string
	usage             *OpenAIUsage
	messageID         string
	model             string
}

// StreamResult contains usage information captured during streaming translation.
type StreamResult struct {
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
}

// TranslateOpenAIStreamToAnthropic reads an OpenAI streaming response from
// upstreamBody and writes Anthropic-format SSE events to w in real time.
//
// The caller MUST set these response headers before calling this function:
//
//	Content-Type: text/event-stream
//	Cache-Control: no-cache
//	Connection: keep-alive
func TranslateOpenAIStreamToAnthropic(
	ctx context.Context,
	upstreamBody io.ReadCloser,
	w http.ResponseWriter,
	flusher http.Flusher,
	model string,
) (*StreamResult, error) {
	defer upstreamBody.Close()

	// Close the upstream body when context is cancelled to unblock the
	// scanner if it is waiting on a read.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			upstreamBody.Close()
		case <-done:
		}
	}()

	state := &streamState{
		currentBlockIndex: -1,
		toolCalls:         make(map[int]*toolCallState),
		model:             model,
	}

	scanner := bufio.NewScanner(upstreamBody)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return streamResultFromState(state), err
		}

		line := scanner.Text()

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "data: [DONE]") {
			break
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		payload := line[6:]

		var chunk OpenAIStreamChunk
		if err := sonic.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}

		if err := processChunk(w, flusher, state, &chunk); err != nil {
			return streamResultFromState(state), err
		}
	}

	// If context was cancelled, return the context error rather than
	// a potentially misleading scanner/io error.
	if err := ctx.Err(); err != nil {
		return streamResultFromState(state), err
	}

	if err := scanner.Err(); err != nil {
		_ = finalizeStream(w, flusher, state)
		return streamResultFromState(state), fmt.Errorf("reading upstream SSE stream: %w", err)
	}

	return streamResultFromState(state), finalizeStream(w, flusher, state)
}

func streamResultFromState(state *streamState) *StreamResult {
	r := &StreamResult{}
	if state.usage != nil {
		r.InputTokens, r.OutputTokens, r.CacheReadTokens = normalizeOpenAIUsage(state.usage)
	}
	return r
}

// processChunk handles a single parsed OpenAI stream chunk.
func processChunk(w http.ResponseWriter, flusher http.Flusher, state *streamState, chunk *OpenAIStreamChunk) error {
	// Step 1: Emit message_start on the very first chunk.
	if !state.messageStartSent {
		state.messageID = generateMessageID()
		if err := emitMessageStart(w, flusher, state); err != nil {
			return err
		}
		state.messageStartSent = true

		if err := writeSSE(w, flusher, "ping", PingEvent{Type: "ping"}); err != nil {
			return err
		}
	}

	// Usage-only chunk (choices empty, usage set).
	if len(chunk.Choices) == 0 {
		if chunk.Usage != nil {
			state.usage = chunk.Usage
		}
		return nil
	}

	choice := chunk.Choices[0]

	// Step 3: Reasoning/thinking content delta.
	if choice.Delta.ReasoningContent != nil && *choice.Delta.ReasoningContent != "" {
		if err := handleThinkingDelta(w, flusher, state, *choice.Delta.ReasoningContent); err != nil {
			return err
		}
	}

	// Step 4: Content delta.
	if choice.Delta.Content != nil && *choice.Delta.Content != "" {
		if err := handleContentDelta(w, flusher, state, *choice.Delta.Content); err != nil {
			return err
		}
	}

	// Step 5: Tool call deltas.
	if choice.Delta.ToolCalls != nil {
		for _, tc := range choice.Delta.ToolCalls {
			if err := handleToolCallDelta(w, flusher, state, tc); err != nil {
				return err
			}
		}
	}

	// Step 6: Finish reason.
	if choice.FinishReason != nil {
		state.finishReason = choice.FinishReason
	}

	// Store usage if present alongside choices.
	if chunk.Usage != nil {
		state.usage = chunk.Usage
	}

	return nil
}

// emitMessageStart writes the initial message_start event.
func emitMessageStart(w http.ResponseWriter, flusher http.Flusher, state *streamState) error {
	evt := MessageStartEvent{
		Type: "message_start",
		Message: AnthropicResponse{
			ID:           state.messageID,
			Type:         "message",
			Role:         "assistant",
			Model:        state.model,
			Content:      []ContentBlock{},
			StopReason:   nil,
			StopSequence: nil,
			Usage: AnthropicUsage{
				InputTokens:  0,
				OutputTokens: 0,
			},
		},
	}
	return writeSSE(w, flusher, "message_start", evt)
}

// handleContentDelta processes a text content delta.
func handleContentDelta(w http.ResponseWriter, flusher http.Flusher, state *streamState, text string) error {
	if state.currentBlockType != "text" {
		if err := closeCurrentBlock(w, flusher, state); err != nil {
			return err
		}
		state.currentBlockIndex++
		if err := writeSSE(w, flusher, "content_block_start", ContentBlockStartEvent{
			Type:  "content_block_start",
			Index: state.currentBlockIndex,
			ContentBlock: ContentBlock{
				Type: "text",
				Text: "",
			},
		}); err != nil {
			return err
		}
		state.currentBlockType = "text"
	}

	return writeSSE(w, flusher, "content_block_delta", ContentBlockDeltaEvent{
		Type:  "content_block_delta",
		Index: state.currentBlockIndex,
		Delta: DeltaBlock{
			Type: "text_delta",
			Text: text,
		},
	})
}

// handleThinkingDelta processes a reasoning_content delta from the upstream
// and emits it as an Anthropic thinking content block.
func handleThinkingDelta(w http.ResponseWriter, flusher http.Flusher, state *streamState, text string) error {
	if state.currentBlockType != "thinking" {
		if err := closeCurrentBlock(w, flusher, state); err != nil {
			return err
		}
		state.currentBlockIndex++
		if err := writeSSE(w, flusher, "content_block_start", ContentBlockStartEvent{
			Type:  "content_block_start",
			Index: state.currentBlockIndex,
			ContentBlock: ContentBlock{
				Type:     "thinking",
				Thinking: "",
			},
		}); err != nil {
			return err
		}
		state.currentBlockType = "thinking"
	}

	return writeSSE(w, flusher, "content_block_delta", ContentBlockDeltaEvent{
		Type:  "content_block_delta",
		Index: state.currentBlockIndex,
		Delta: DeltaBlock{
			Type:     "thinking_delta",
			Thinking: text,
		},
	})
}

// handleToolCallDelta processes a single tool call delta from a chunk.
func handleToolCallDelta(w http.ResponseWriter, flusher http.Flusher, state *streamState, tc OpenAIStreamToolCall) error {
	tcIdx := tc.Index

	// New tool call starting (has ID).
	if tc.ID != "" {
		if err := closeCurrentBlock(w, flusher, state); err != nil {
			return err
		}
		state.currentBlockIndex++

		name := ""
		if tc.Function != nil {
			name = tc.Function.Name
		}

		tcs := &toolCallState{
			anthropicIndex: state.currentBlockIndex,
			id:             tc.ID,
			name:           name,
		}
		state.toolCalls[tcIdx] = tcs

		if err := writeSSE(w, flusher, "content_block_start", ContentBlockStartEvent{
			Type:  "content_block_start",
			Index: state.currentBlockIndex,
			ContentBlock: ContentBlock{
				Type:  "tool_use",
				ID:    tc.ID,
				Name:  name,
				Input: json.RawMessage(`{}`),
			},
		}); err != nil {
			return err
		}
		state.currentBlockType = "tool_use"
	}

	// Argument delta.
	if tc.Function != nil && tc.Function.Arguments != "" {
		tcs := state.toolCalls[tcIdx]
		if tcs != nil {
			tcs.argsBuffer.WriteString(tc.Function.Arguments)
			if err := writeSSE(w, flusher, "content_block_delta", ContentBlockDeltaEvent{
				Type:  "content_block_delta",
				Index: tcs.anthropicIndex,
				Delta: DeltaBlock{
					Type:        "input_json_delta",
					PartialJSON: tc.Function.Arguments,
				},
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

// closeCurrentBlock emits a content_block_stop for the current block if one
// is open.
func closeCurrentBlock(w http.ResponseWriter, flusher http.Flusher, state *streamState) error {
	if state.currentBlockIndex >= 0 {
		return writeSSE(w, flusher, "content_block_stop", ContentBlockStopEvent{
			Type:  "content_block_stop",
			Index: state.currentBlockIndex,
		})
	}
	return nil
}

// finalizeStream closes any open content block and emits message_delta +
// message_stop to finish the Anthropic stream.
func finalizeStream(w http.ResponseWriter, flusher http.Flusher, state *streamState) error {
	if !state.messageStartSent {
		return nil
	}

	if err := closeCurrentBlock(w, flusher, state); err != nil {
		return err
	}

	stopReason := mapFinishReason(state.finishReason)

	inputTokens := 0
	outputTokens := 0
	if state.usage != nil {
		inputTokens, outputTokens, _ = normalizeOpenAIUsage(state.usage)
	}

	if err := writeSSE(w, flusher, "message_delta", MessageDeltaEvent{
		Type: "message_delta",
		Delta: MessageDelta{
			StopReason:   &stopReason,
			StopSequence: nil,
		},
		Usage: &MessageDeltaUsage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		},
	}); err != nil {
		return err
	}

	return writeSSE(w, flusher, "message_stop", MessageStopEvent{Type: "message_stop"})
}

// writeSSE marshals data as JSON and writes a properly formatted SSE event,
// then flushes the response writer.
func writeSSE(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) error {
	jsonBytes, err := sonic.Marshal(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, jsonBytes)
	if err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
