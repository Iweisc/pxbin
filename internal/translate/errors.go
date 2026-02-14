package translate

import json "github.com/bytedance/sonic"

// TranslateOpenAIErrorToAnthropic converts an OpenAI error response body and
// HTTP status code into an Anthropic-format error response body and status code.
func TranslateOpenAIErrorToAnthropic(statusCode int, body []byte) ([]byte, int) {
	var oaiErr OpenAIErrorResponse
	if err := json.Unmarshal(body, &oaiErr); err != nil {
		return marshalAnthropicError(mapStatusToErrorType(statusCode), string(body)), mapStatusCode(statusCode)
	}

	errType := mapErrorType(statusCode)
	result, _ := json.Marshal(AnthropicErrorResponse{
		Type: "error",
		Error: AnthropicError{
			Type:    errType,
			Message: oaiErr.Error.Message,
		},
	})
	return result, mapStatusCode(statusCode)
}

// marshalAnthropicError builds a serialised Anthropic error response.
func marshalAnthropicError(errType, message string) []byte {
	result, _ := json.Marshal(AnthropicErrorResponse{
		Type: "error",
		Error: AnthropicError{
			Type:    errType,
			Message: message,
		},
	})
	return result
}

// mapErrorType maps an HTTP status code to an Anthropic error type string.
func mapErrorType(statusCode int) string {
	switch statusCode {
	case 400:
		return "invalid_request_error"
	case 401:
		return "authentication_error"
	case 403:
		return "permission_error"
	case 404:
		return "not_found_error"
	case 429:
		return "rate_limit_error"
	case 500, 502, 503:
		return "api_error"
	default:
		return "api_error"
	}
}

// mapStatusToErrorType is an alias used when we cannot parse the upstream body.
func mapStatusToErrorType(statusCode int) string {
	return mapErrorType(statusCode)
}

// mapStatusCode translates an upstream HTTP status code to an appropriate
// Anthropic proxy status code.
func mapStatusCode(upstream int) int {
	switch {
	case upstream >= 400 && upstream < 500:
		return upstream
	case upstream >= 500:
		return 502
	default:
		return upstream
	}
}
