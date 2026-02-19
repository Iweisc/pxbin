package proxy

import (
	"testing"
)

func TestNormalizeOpenAIInputAndCache(t *testing.T) {
	input, cache := normalizeOpenAIInputAndCache(1000, 980)
	if input != 20 || cache != 980 {
		t.Fatalf("expected normalized tokens (20,980), got (%d,%d)", input, cache)
	}

	input, cache = normalizeOpenAIInputAndCache(5, 9)
	if input != 0 || cache != 5 {
		t.Fatalf("expected clamped tokens (0,5), got (%d,%d)", input, cache)
	}
}
