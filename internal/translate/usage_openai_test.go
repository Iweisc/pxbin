package translate

import "testing"

func TestNormalizeOpenAIUsage_NoUsage(t *testing.T) {
	in, out, cache := normalizeOpenAIUsage(nil)
	if in != 0 || out != 0 || cache != 0 {
		t.Fatalf("expected zero values, got (%d,%d,%d)", in, out, cache)
	}
}

func TestNormalizeOpenAIUsage_NoCacheDetails(t *testing.T) {
	in, out, cache := normalizeOpenAIUsage(&OpenAIUsage{
		PromptTokens:     120,
		CompletionTokens: 33,
	})
	if in != 120 || out != 33 || cache != 0 {
		t.Fatalf("expected (120,33,0), got (%d,%d,%d)", in, out, cache)
	}
}

func TestNormalizeOpenAIUsage_WithCachedTokens(t *testing.T) {
	in, out, cache := normalizeOpenAIUsage(&OpenAIUsage{
		PromptTokens:     120,
		CompletionTokens: 33,
		PromptTokensDetails: &OpenAIPromptTokensDetails{
			CachedTokens: 100,
		},
	})
	if in != 20 || out != 33 || cache != 100 {
		t.Fatalf("expected (20,33,100), got (%d,%d,%d)", in, out, cache)
	}
}

func TestNormalizeOpenAIUsage_CachedTokensClamped(t *testing.T) {
	in, out, cache := normalizeOpenAIUsage(&OpenAIUsage{
		PromptTokens:     8,
		CompletionTokens: 3,
		PromptTokensDetails: &OpenAIPromptTokensDetails{
			CachedTokens: 99,
		},
	})
	if in != 0 || out != 3 || cache != 8 {
		t.Fatalf("expected clamped (0,3,8), got (%d,%d,%d)", in, out, cache)
	}
}
