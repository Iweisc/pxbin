package ratelimit

import (
	"sync"
	"sync/atomic"
	"time"
)

// bucket is a lock-free token bucket for a single key.
type bucket struct {
	tokens     atomic.Int64
	lastRefill atomic.Int64 // unix nanoseconds
	lastSeen   atomic.Int64 // unix nanoseconds for cleanup
}

// Limiter provides per-key token-bucket rate limiting using sync.Map for
// lock-free reads on the hot path.
type Limiter struct {
	rps     float64
	burst   int
	buckets sync.Map // map[string]*bucket
	done    chan struct{}
	wg      sync.WaitGroup
}

// NewLimiter creates a rate limiter. rps is the refill rate (tokens per
// second), burst is the maximum token count.
func NewLimiter(rps float64, burst int) *Limiter {
	l := &Limiter{
		rps:   rps,
		burst: burst,
		done:  make(chan struct{}),
	}
	l.wg.Add(1)
	go l.cleanup()
	return l
}

// Allow returns true if the request for key is allowed.
func (l *Limiter) Allow(key string) bool {
	now := time.Now().UnixNano()

	val, loaded := l.buckets.Load(key)
	if !loaded {
		b := &bucket{}
		b.tokens.Store(int64(l.burst) - 1) // consume one token
		b.lastRefill.Store(now)
		b.lastSeen.Store(now)
		val, loaded = l.buckets.LoadOrStore(key, b)
		if !loaded {
			return true
		}
	}

	b := val.(*bucket)
	b.lastSeen.Store(now)

	// Refill tokens based on elapsed time.
	for {
		oldRefill := b.lastRefill.Load()
		elapsed := float64(now-oldRefill) / float64(time.Second)
		if elapsed <= 0 {
			break
		}
		newTokens := int64(elapsed * l.rps)
		if newTokens <= 0 {
			break
		}
		if b.lastRefill.CompareAndSwap(oldRefill, now) {
			// Use CAS loop to add tokens atomically, avoiding overwriting
			// concurrent decrements from the consume path.
			for {
				oldTokens := b.tokens.Load()
				desired := oldTokens + newTokens
				if desired > int64(l.burst) {
					desired = int64(l.burst)
				}
				if b.tokens.CompareAndSwap(oldTokens, desired) {
					break
				}
			}
			break
		}
	}

	// Try to consume a token via CAS loop.
	for {
		current := b.tokens.Load()
		if current <= 0 {
			return false
		}
		if b.tokens.CompareAndSwap(current, current-1) {
			return true
		}
	}
}

// Close stops the cleanup goroutine.
func (l *Limiter) Close() {
	close(l.done)
	l.wg.Wait()
}

// cleanup evicts stale buckets every 60 seconds.
func (l *Limiter) cleanup() {
	defer l.wg.Done()
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	const staleThreshold = 5 * time.Minute

	for {
		select {
		case <-ticker.C:
			cutoff := time.Now().Add(-staleThreshold).UnixNano()
			l.buckets.Range(func(key, val any) bool {
				b := val.(*bucket)
				if b.lastSeen.Load() < cutoff {
					l.buckets.Delete(key)
				}
				return true
			})
		case <-l.done:
			return
		}
	}
}
