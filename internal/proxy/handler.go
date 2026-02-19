package proxy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	json "github.com/bytedance/sonic"

	"github.com/sertdev/pxbin/internal/billing"
	"github.com/sertdev/pxbin/internal/logging"
	"github.com/sertdev/pxbin/internal/store"
	"github.com/sertdev/pxbin/internal/translate"
)

// Handler contains the shared dependencies for the Anthropic and OpenAI proxy
// endpoints.
type Handler struct {
	clients    *ClientCache
	modelCache *ModelCache
	store      *store.Store
	logger     *logging.AsyncLogger
	billing    *billing.Tracker
}

// NewHandler creates a Handler wired up to a client cache, model cache, store,
// logger and billing tracker.
func NewHandler(clients *ClientCache, modelCache *ModelCache, s *store.Store, logger *logging.AsyncLogger, billing *billing.Tracker) *Handler {
	return &Handler{
		clients:    clients,
		modelCache: modelCache,
		store:      s,
		logger:     logger,
		billing:    billing,
	}
}

// maxRequestBodySize is the maximum allowed request body size (32 MB).
// Claude Code payloads with large system prompts and tools can reach ~1 MB;
// 32 MB provides generous headroom while preventing OOM from malicious input.
const maxRequestBodySize = 32 << 20 // 32 MB

// readBody reads the full request body. When Content-Length is known it
// pre-allocates a single buffer of the exact size, avoiding the repeated
// grow-and-copy cycles that io.ReadAll performs (which start at 512 bytes
// and double each time — ~20 allocs + 2x body size in wasted copies for
// a 500KB Claude Code payload). The body is capped at maxRequestBodySize.
func readBody(r *http.Request) ([]byte, error) {
	if r.ContentLength > maxRequestBodySize {
		return nil, fmt.Errorf("request body too large: %d bytes exceeds %d byte limit", r.ContentLength, maxRequestBodySize)
	}
	limited := io.LimitReader(r.Body, maxRequestBodySize+1)
	if r.ContentLength > 0 {
		buf := make([]byte, r.ContentLength)
		_, err := io.ReadFull(limited, buf)
		return buf, err
	}
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxRequestBodySize {
		return nil, fmt.Errorf("request body too large: exceeds %d byte limit", maxRequestBodySize)
	}
	return data, nil
}

var errModelNotFound = errors.New("missing model field")

const modelProbeLimitBytes = 8 * 1024

var modelJSONField = []byte(`"model"`)

// readModelAndBuildBodyReader reads just enough of a JSON body to extract the
// "model" field, then returns a reader that replays consumed bytes plus the
// remaining unread stream. This lets handlers dispatch upstream without first
// buffering the entire request body.
func readModelAndBuildBodyReader(body io.Reader, probeLimit int) (string, io.Reader, error) {
	if probeLimit <= 0 {
		probeLimit = modelProbeLimitBytes
	}

	prefix := make([]byte, 0, minInt(1024, probeLimit))
	chunk := make([]byte, 1024)

	for len(prefix) < probeLimit {
		toRead := minInt(len(chunk), probeLimit-len(prefix))
		n, err := body.Read(chunk[:toRead])
		if n > 0 {
			prefix = append(prefix, chunk[:n]...)
			if model, ok := extractJSONStringFieldValue(prefix, modelJSONField); ok {
				return model, io.MultiReader(bytes.NewReader(prefix), body), nil
			}
		}
		if err == io.EOF {
			model, modelErr := extractModelWithJSONGet(prefix)
			if modelErr != nil {
				return "", nil, modelErr
			}
			return model, bytes.NewReader(prefix), nil
		}
		if err != nil {
			return "", nil, err
		}
	}

	rest, err := io.ReadAll(body)
	if err != nil {
		return "", nil, err
	}
	full := append(prefix, rest...)
	model, err := extractModelWithJSONGet(full)
	if err != nil {
		return "", nil, err
	}
	return model, bytes.NewReader(full), nil
}

func extractModelWithJSONGet(body []byte) (string, error) {
	modelNode, err := json.Get(body, "model")
	if err != nil {
		return "", errModelNotFound
	}
	model, err := modelNode.String()
	if err != nil || model == "" {
		return "", errModelNotFound
	}
	return model, nil
}

// extractJSONStringFieldValue scans a JSON payload prefix for a string field
// and returns its value without needing a full parse.
func extractJSONStringFieldValue(body []byte, field []byte) (string, bool) {
	pos := 0
	for {
		rel := bytes.Index(body[pos:], field)
		if rel < 0 {
			return "", false
		}
		keyStart := pos + rel

		if !likelyJSONKeyBoundary(body, keyStart) {
			pos = keyStart + 1
			continue
		}

		i := keyStart + len(field)
		for i < len(body) && isJSONWhitespace(body[i]) {
			i++
		}
		if i >= len(body) {
			return "", false
		}
		if body[i] != ':' {
			pos = keyStart + 1
			continue
		}
		i++
		for i < len(body) && isJSONWhitespace(body[i]) {
			i++
		}
		if i >= len(body) {
			return "", false
		}
		if body[i] != '"' {
			pos = keyStart + 1
			continue
		}

		i++
		valueStart := i
		sawEscape := false
		for i < len(body) {
			switch body[i] {
			case '\\':
				sawEscape = true
				i++
				if i >= len(body) {
					return "", false
				}
			case '"':
				if !sawEscape {
					return string(body[valueStart:i]), true
				}
				raw := make([]byte, i-valueStart+2)
				raw[0] = '"'
				copy(raw[1:], body[valueStart:i])
				raw[len(raw)-1] = '"'
				decoded, err := strconv.Unquote(string(raw))
				if err != nil {
					return "", false
				}
				return decoded, true
			}
			i++
		}
		return "", false
	}
}

func likelyJSONKeyBoundary(body []byte, keyStart int) bool {
	if keyStart == 0 {
		return true
	}
	i := keyStart - 1
	for i >= 0 && isJSONWhitespace(body[i]) {
		i--
	}
	if i < 0 {
		return true
	}
	return body[i] == '{' || body[i] == ','
}

func isJSONWhitespace(b byte) bool {
	return b == ' ' || b == '\n' || b == '\r' || b == '\t'
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func writeAnthropicError(w http.ResponseWriter, statusCode int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	b, _ := json.Marshal(translate.AnthropicErrorResponse{
		Type: "error",
		Error: translate.AnthropicError{
			Type:    errType,
			Message: message,
		},
	})
	w.Write(b)
}

func writeOpenAIError(w http.ResponseWriter, statusCode int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	b, _ := json.Marshal(translate.OpenAIErrorResponse{
		Error: translate.OpenAIError{
			Message: message,
			Type:    errType,
		},
	})
	w.Write(b)
}
