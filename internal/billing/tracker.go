package billing

import (
	"context"
	"sync"
	"time"

	"github.com/sertdev/pxbin/internal/store"
)

type ModelPricing struct {
	InputCostPerMillion  float64
	OutputCostPerMillion float64
}

type Tracker struct {
	pricing map[string]*ModelPricing
	store   *store.Store
	mu      sync.RWMutex
	done    chan struct{}
}

func NewTracker(s *store.Store) *Tracker {
	t := &Tracker{
		pricing: make(map[string]*ModelPricing),
		store:   s,
		done:    make(chan struct{}),
	}
	// Load hardcoded defaults
	t.loadDefaults()
	// Try to load from DB (non-fatal if it fails)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = t.RefreshPricing(ctx)

	// Start periodic refresh
	go t.refreshLoop()
	return t
}

func (t *Tracker) CalculateCost(model string, inputTokens, outputTokens int) float64 {
	t.mu.RLock()
	p, ok := t.pricing[model]
	t.mu.RUnlock()
	if !ok {
		return 0
	}
	inputCost := float64(inputTokens) / 1_000_000 * p.InputCostPerMillion
	outputCost := float64(outputTokens) / 1_000_000 * p.OutputCostPerMillion
	return inputCost + outputCost
}

func (t *Tracker) RefreshPricing(ctx context.Context) error {
	models, err := t.store.ListModels(ctx)
	if err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, m := range models {
		t.pricing[m.Name] = &ModelPricing{
			InputCostPerMillion:  m.InputCostPerMillion,
			OutputCostPerMillion: m.OutputCostPerMillion,
		}
	}
	return nil
}

func (t *Tracker) refreshLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = t.RefreshPricing(ctx)
			cancel()
		case <-t.done:
			return
		}
	}
}

func (t *Tracker) Close() {
	close(t.done)
}

func (t *Tracker) loadDefaults() {
	for name, p := range DefaultPricing {
		t.pricing[name] = p
	}
}
