package auth

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sertdev/pxbin/internal/store"
)

// LastUsedTracker batches UpdateLLMKeyLastUsed calls instead of spawning
// unbounded goroutines. It deduplicates touches per flush interval.
type LastUsedTracker struct {
	mu      sync.Mutex
	pending map[uuid.UUID]struct{}
	store   *store.Store
	done    chan struct{}
	wg      sync.WaitGroup
}

// NewLastUsedTracker creates a tracker that flushes every 30 seconds.
func NewLastUsedTracker(s *store.Store) *LastUsedTracker {
	t := &LastUsedTracker{
		pending: make(map[uuid.UUID]struct{}),
		store:   s,
		done:    make(chan struct{}),
	}
	t.wg.Add(1)
	go t.worker()
	return t
}

// Touch marks a key as recently used. Non-blocking.
func (t *LastUsedTracker) Touch(id uuid.UUID) {
	t.mu.Lock()
	t.pending[id] = struct{}{}
	t.mu.Unlock()
}

// Close flushes remaining updates and stops the worker.
func (t *LastUsedTracker) Close() {
	close(t.done)
	t.wg.Wait()
}

func (t *LastUsedTracker) worker() {
	defer t.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.flush()
		case <-t.done:
			t.flush()
			return
		}
	}
}

func (t *LastUsedTracker) flush() {
	t.mu.Lock()
	if len(t.pending) == 0 {
		t.mu.Unlock()
		return
	}
	ids := make([]uuid.UUID, 0, len(t.pending))
	for id := range t.pending {
		ids = append(ids, id)
	}
	t.pending = make(map[uuid.UUID]struct{})
	t.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := t.store.BatchUpdateLLMKeyLastUsed(ctx, ids); err != nil {
		log.Printf("last-used tracker: batch update failed: %v", err)
	}
}
