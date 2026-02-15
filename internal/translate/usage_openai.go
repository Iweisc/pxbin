package translate

// normalizeOpenAIUsage converts OpenAI usage semantics into Anthropic-style
// accounting:
// - OpenAI prompt_tokens includes cached tokens
// - Anthropic input_tokens excludes cache reads and tracks them separately
func normalizeOpenAIUsage(usage *OpenAIUsage) (inputTokens, outputTokens, cacheReadTokens int) {
	if usage == nil {
		return 0, 0, 0
	}

	inputTokens = usage.PromptTokens
	outputTokens = usage.CompletionTokens

	if inputTokens < 0 {
		inputTokens = 0
	}
	if usage.PromptTokensDetails == nil {
		return inputTokens, outputTokens, 0
	}

	cacheReadTokens = usage.PromptTokensDetails.CachedTokens
	if cacheReadTokens < 0 {
		cacheReadTokens = 0
	}
	if cacheReadTokens > inputTokens {
		cacheReadTokens = inputTokens
	}

	inputTokens -= cacheReadTokens
	return inputTokens, outputTokens, cacheReadTokens
}
