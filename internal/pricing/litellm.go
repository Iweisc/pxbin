package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const LiteLLMPricingURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"

type LiteLLMModel struct {
	InputCostPerToken  float64 `json:"input_cost_per_token"`
	OutputCostPerToken float64 `json:"output_cost_per_token"`
	Mode               string  `json:"mode"`
	// Fields we don't need can be omitted or left as json.RawMessage
}

type ModelPricing struct {
	InputCostPerMillion  float64
	OutputCostPerMillion float64
}

// FetchLiteLLMPricing fetches the model pricing from LiteLLM's GitHub repo.
func FetchLiteLLMPricing(ctx context.Context) (map[string]*ModelPricing, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, LiteLLMPricingURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch pricing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var raw map[string]LiteLLMModel
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode pricing JSON: %w", err)
	}

	pricing := make(map[string]*ModelPricing)
	for modelName, model := range raw {
		// Skip sample_spec and non-chat models
		if modelName == "sample_spec" || (model.Mode != "" && model.Mode != "chat") {
			continue
		}
		// Skip models with zero pricing
		if model.InputCostPerToken == 0 && model.OutputCostPerToken == 0 {
			continue
		}
		pricing[modelName] = &ModelPricing{
			InputCostPerMillion:  model.InputCostPerToken * 1_000_000,
			OutputCostPerMillion: model.OutputCostPerToken * 1_000_000,
		}
	}

	return pricing, nil
}
