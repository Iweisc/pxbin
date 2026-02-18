package proxy

import (
	"io"
	"strings"
	"testing"
)

func TestReadModelAndBuildBodyReaderProbeHit(t *testing.T) {
	payload := `{"model":"gpt-5.3-codex","input":"hello","stream":false}`

	model, bodyReader, err := readModelAndBuildBodyReader(strings.NewReader(payload), modelProbeLimitBytes)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if model != "gpt-5.3-codex" {
		t.Fatalf("expected model gpt-5.3-codex, got %q", model)
	}

	rebuilt, err := io.ReadAll(bodyReader)
	if err != nil {
		t.Fatalf("expected body replay read success, got %v", err)
	}
	if string(rebuilt) != payload {
		t.Fatalf("expected rebuilt body to equal original payload")
	}
}

func TestReadModelAndBuildBodyReaderFallback(t *testing.T) {
	payload := `{"input":"` + strings.Repeat("a", 256) + `","model":"gpt-5.3-codex","stream":false}`

	model, bodyReader, err := readModelAndBuildBodyReader(strings.NewReader(payload), 32)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if model != "gpt-5.3-codex" {
		t.Fatalf("expected model gpt-5.3-codex, got %q", model)
	}

	rebuilt, err := io.ReadAll(bodyReader)
	if err != nil {
		t.Fatalf("expected body replay read success, got %v", err)
	}
	if string(rebuilt) != payload {
		t.Fatalf("expected rebuilt body to equal original payload")
	}
}

func TestReadModelAndBuildBodyReaderMissingModel(t *testing.T) {
	payload := `{"input":"hello"}`

	_, _, err := readModelAndBuildBodyReader(strings.NewReader(payload), modelProbeLimitBytes)
	if err == nil {
		t.Fatalf("expected error for missing model field")
	}
}
