package translate

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
)

// responsesStreamState holds mutable state for the Chat Completions → Responses
// API SSE translation state machine.
type responsesStreamState struct {
	responseID string
	model      string

	// Track whether initial events have been emitted.
	headerSent bool

	// Current output item tracking.
	messageItemEmitted bool
	messageItemIndex   int
	contentPartIndex   int
	textAccum          strings.Builder

	// Tool call tracking: openaiIndex → state.
	toolCalls       map[int]*responsesToolCallState
	nextOutputIndex int

	// Final state.
	finishReason *string
	usage        *OpenAIUsage
}

type responsesToolCallState struct {
	outputIndex int
	id          string
	name        string
	argsAccum   strings.Builder
}

// TranslateChatStreamToResponses reads Chat Completions SSE from upstreamBody
// and writes Responses API SSE events to w.
//
// The caller MUST set response headers before calling:
//
//	Content-Type: text/event-stream
//	Cache-Control: no-cache
//	Connection: keep-alive
func TranslateChatStreamToResponses(
	ctx context.Context,
	upstreamBody io.ReadCloser,
	w http.ResponseWriter,
	flusher http.Flusher,
	model string,
) (*StreamResult, error) {
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

	state := &responsesStreamState{
		responseID:       generateResponseID(),
		model:            model,
		messageItemIndex: -1,
		toolCalls:        make(map[int]*responsesToolCallState),
	}

	scanner := bufio.NewScanner(upstreamBody)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return responsesStreamResultFromState(state), err
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

		var chunk OpenAIStreamChunk
		if err := sonic.Unmarshal([]byte(line[6:]), &chunk); err != nil {
			continue
		}

		if err := processResponsesChunk(w, flusher, state, &chunk); err != nil {
			return responsesStreamResultFromState(state), err
		}
	}

	if err := ctx.Err(); err != nil {
		return responsesStreamResultFromState(state), err
	}
	if err := scanner.Err(); err != nil {
		_ = finalizeResponsesStream(w, flusher, state)
		return responsesStreamResultFromState(state), fmt.Errorf("reading upstream SSE: %w", err)
	}

	return responsesStreamResultFromState(state), finalizeResponsesStream(w, flusher, state)
}

func responsesStreamResultFromState(state *responsesStreamState) *StreamResult {
	r := &StreamResult{}
	if state.usage != nil {
		r.InputTokens, r.OutputTokens, r.CacheReadTokens = normalizeOpenAIUsage(state.usage)
	}
	return r
}

// processResponsesChunk handles a single Chat Completions stream chunk and
// emits the corresponding Responses API SSE events.
func processResponsesChunk(w http.ResponseWriter, flusher http.Flusher, state *responsesStreamState, chunk *OpenAIStreamChunk) error {
	// Emit initial response events on first chunk.
	if !state.headerSent {
		if chunk.Model != "" {
			state.model = chunk.Model
		}
		if err := emitResponsesHeader(w, flusher, state); err != nil {
			return err
		}
		state.headerSent = true
	}

	// Usage-only chunk.
	if len(chunk.Choices) == 0 {
		if chunk.Usage != nil {
			state.usage = chunk.Usage
		}
		return nil
	}

	choice := chunk.Choices[0]

	// Content delta → output_text.delta
	if choice.Delta.Content != nil && *choice.Delta.Content != "" {
		if err := handleResponsesContentDelta(w, flusher, state, *choice.Delta.Content); err != nil {
			return err
		}
	}

	// Tool call deltas.
	for _, tc := range choice.Delta.ToolCalls {
		if err := handleResponsesToolCallDelta(w, flusher, state, tc); err != nil {
			return err
		}
	}

	if choice.FinishReason != nil {
		state.finishReason = choice.FinishReason
	}
	if chunk.Usage != nil {
		state.usage = chunk.Usage
	}

	return nil
}

// emitResponsesHeader emits the initial Responses API events:
// response.created and response.in_progress.
func emitResponsesHeader(w http.ResponseWriter, flusher http.Flusher, state *responsesStreamState) error {
	resp := buildPartialResponse(state, "in_progress", nil)

	if err := writeResponsesSSE(w, flusher, "response.created", map[string]interface{}{
		"type":     "response.created",
		"response": resp,
	}); err != nil {
		return err
	}
	return writeResponsesSSE(w, flusher, "response.in_progress", map[string]interface{}{
		"type":     "response.in_progress",
		"response": resp,
	})
}

// handleResponsesContentDelta processes text content and emits the appropriate
// Responses API events (adding message/content_part on first delta).
func handleResponsesContentDelta(w http.ResponseWriter, flusher http.Flusher, state *responsesStreamState, text string) error {
	// Emit message output item + content part on first text delta.
	if !state.messageItemEmitted {
		state.messageItemIndex = state.nextOutputIndex
		state.nextOutputIndex++
		state.contentPartIndex = 0

		msgItem := ResponsesOutputItem{
			Type:    "message",
			ID:      generateMessageID(),
			Role:    "assistant",
			Status:  "in_progress",
			Content: []ResponsesContentPart{},
		}
		if err := writeResponsesSSE(w, flusher, "response.output_item.added", map[string]interface{}{
			"type":         "response.output_item.added",
			"output_index": state.messageItemIndex,
			"item":         msgItem,
		}); err != nil {
			return err
		}

		part := ResponsesContentPart{Type: "output_text", Text: ""}
		if err := writeResponsesSSE(w, flusher, "response.content_part.added", map[string]interface{}{
			"type":          "response.content_part.added",
			"output_index":  state.messageItemIndex,
			"content_index": state.contentPartIndex,
			"part":          part,
		}); err != nil {
			return err
		}

		state.messageItemEmitted = true
	}

	state.textAccum.WriteString(text)

	return writeResponsesSSE(w, flusher, "response.output_text.delta", map[string]interface{}{
		"type":          "response.output_text.delta",
		"output_index":  state.messageItemIndex,
		"content_index": state.contentPartIndex,
		"delta":         text,
	})
}

// handleResponsesToolCallDelta processes a tool call delta from Chat Completions.
func handleResponsesToolCallDelta(w http.ResponseWriter, flusher http.Flusher, state *responsesStreamState, tc OpenAIStreamToolCall) error {
	tcIdx := tc.Index

	// New tool call — emit function_call output item.
	if tc.ID != "" {
		// Close the message content part/item if still open.
		if state.messageItemEmitted {
			if err := closeResponsesMessageItem(w, flusher, state); err != nil {
				return err
			}
		}

		outputIdx := state.nextOutputIndex
		state.nextOutputIndex++

		name := ""
		if tc.Function != nil {
			name = tc.Function.Name
		}

		tcs := &responsesToolCallState{
			outputIndex: outputIdx,
			id:          tc.ID,
			name:        name,
		}
		state.toolCalls[tcIdx] = tcs

		item := ResponsesOutputItem{
			Type:   "function_call",
			ID:     tc.ID,
			CallID: tc.ID,
			Name:   name,
			Status: "in_progress",
		}
		if err := writeResponsesSSE(w, flusher, "response.output_item.added", map[string]interface{}{
			"type":         "response.output_item.added",
			"output_index": outputIdx,
			"item":         item,
		}); err != nil {
			return err
		}
	}

	// Arguments delta.
	if tc.Function != nil && tc.Function.Arguments != "" {
		tcs := state.toolCalls[tcIdx]
		if tcs != nil {
			tcs.argsAccum.WriteString(tc.Function.Arguments)
			if err := writeResponsesSSE(w, flusher, "response.function_call_arguments.delta", map[string]interface{}{
				"type":         "response.function_call_arguments.delta",
				"output_index": tcs.outputIndex,
				"delta":        tc.Function.Arguments,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

// closeResponsesMessageItem emits the done events for the text content part
// and message item. Called when transitioning to tool calls or finalizing.
func closeResponsesMessageItem(w http.ResponseWriter, flusher http.Flusher, state *responsesStreamState) error {
	if !state.messageItemEmitted {
		return nil
	}

	// output_text.done
	if err := writeResponsesSSE(w, flusher, "response.output_text.done", map[string]interface{}{
		"type":          "response.output_text.done",
		"output_index":  state.messageItemIndex,
		"content_index": state.contentPartIndex,
		"text":          state.textAccum.String(),
	}); err != nil {
		return err
	}

	// content_part.done
	if err := writeResponsesSSE(w, flusher, "response.content_part.done", map[string]interface{}{
		"type":          "response.content_part.done",
		"output_index":  state.messageItemIndex,
		"content_index": state.contentPartIndex,
		"part":          ResponsesContentPart{Type: "output_text", Text: state.textAccum.String()},
	}); err != nil {
		return err
	}

	// output_item.done
	if err := writeResponsesSSE(w, flusher, "response.output_item.done", map[string]interface{}{
		"type":         "response.output_item.done",
		"output_index": state.messageItemIndex,
		"item": ResponsesOutputItem{
			Type:   "message",
			ID:     generateMessageID(),
			Role:   "assistant",
			Status: "completed",
			Content: []ResponsesContentPart{{
				Type: "output_text",
				Text: state.textAccum.String(),
			}},
		},
	}); err != nil {
		return err
	}

	// Mark as closed so we don't emit again.
	state.messageItemEmitted = false
	return nil
}

// finalizeResponsesStream emits closing events for all open items and the
// response.completed event.
func finalizeResponsesStream(w http.ResponseWriter, flusher http.Flusher, state *responsesStreamState) error {
	if !state.headerSent {
		return nil
	}

	// Close open message item.
	if state.messageItemEmitted {
		if err := closeResponsesMessageItem(w, flusher, state); err != nil {
			return err
		}
	}

	// Close open tool calls.
	for _, tcs := range state.toolCalls {
		// function_call_arguments.done
		if err := writeResponsesSSE(w, flusher, "response.function_call_arguments.done", map[string]interface{}{
			"type":         "response.function_call_arguments.done",
			"output_index": tcs.outputIndex,
			"arguments":    tcs.argsAccum.String(),
		}); err != nil {
			return err
		}
		// output_item.done
		if err := writeResponsesSSE(w, flusher, "response.output_item.done", map[string]interface{}{
			"type":         "response.output_item.done",
			"output_index": tcs.outputIndex,
			"item": ResponsesOutputItem{
				Type:      "function_call",
				ID:        tcs.id,
				CallID:    tcs.id,
				Name:      tcs.name,
				Arguments: tcs.argsAccum.String(),
				Status:    "completed",
			},
		}); err != nil {
			return err
		}
	}

	// Build final output items for the completed response.
	var output []ResponsesOutputItem
	if state.textAccum.Len() > 0 {
		output = append(output, ResponsesOutputItem{
			Type:   "message",
			ID:     generateMessageID(),
			Role:   "assistant",
			Status: "completed",
			Content: []ResponsesContentPart{{
				Type: "output_text",
				Text: state.textAccum.String(),
			}},
		})
	}
	for _, tcs := range state.toolCalls {
		output = append(output, ResponsesOutputItem{
			Type:      "function_call",
			ID:        tcs.id,
			CallID:    tcs.id,
			Name:      tcs.name,
			Arguments: tcs.argsAccum.String(),
			Status:    "completed",
		})
	}

	var usage *ResponsesUsage
	if state.usage != nil {
		inputTokens, outputTokens, _ := normalizeOpenAIUsage(state.usage)
		usage = &ResponsesUsage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  inputTokens + outputTokens,
		}
	}

	resp := buildPartialResponse(state, "completed", output)
	if usage != nil {
		resp["usage"] = usage
	}

	return writeResponsesSSE(w, flusher, "response.completed", map[string]interface{}{
		"type":     "response.completed",
		"response": resp,
	})
}

// buildPartialResponse constructs the response object embedded in SSE events.
func buildPartialResponse(state *responsesStreamState, status string, output []ResponsesOutputItem) map[string]interface{} {
	resp := map[string]interface{}{
		"id":     state.responseID,
		"object": "response",
		"model":  state.model,
		"status": status,
	}
	if output != nil {
		resp["output"] = output
	} else {
		resp["output"] = []interface{}{}
	}
	return resp
}

// writeResponsesSSE writes a Responses API SSE event. Unlike Chat Completions
// SSE (data-only), the Responses API uses event: type lines.
func writeResponsesSSE(w http.ResponseWriter, flusher http.Flusher, eventType string, data interface{}) error {
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
