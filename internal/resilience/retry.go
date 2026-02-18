package resilience

import (
	"context"
	"math/rand/v2"
	"net"
	"time"
)

// RetryOpts configures retry behavior.
type RetryOpts struct {
	MaxAttempts int           // total attempts including first try (default 3)
	BaseDelay   time.Duration // initial delay between retries (default 100ms)
	MaxDelay    time.Duration // maximum delay cap (default 2s)
	Jitter      bool          // add ±25% random jitter (default true)
}

func (o *RetryOpts) withDefaults() RetryOpts {
	out := *o
	if out.MaxAttempts <= 0 {
		out.MaxAttempts = 3
	}
	if out.BaseDelay <= 0 {
		out.BaseDelay = 100 * time.Millisecond
	}
	if out.MaxDelay <= 0 {
		out.MaxDelay = 2 * time.Second
	}
	return out
}

// Do retries fn with exponential backoff. Only connection-level errors
// (net.Error) are retried — HTTP status errors should NOT be wrapped in
// retryable errors.
func Do(ctx context.Context, opts RetryOpts, fn func() error) error {
	opts = opts.withDefaults()

	var lastErr error
	for attempt := 0; attempt < opts.MaxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if !IsRetryable(lastErr) {
			return lastErr
		}

		if attempt == opts.MaxAttempts-1 {
			break
		}

		delay := opts.BaseDelay * (1 << uint(attempt))
		if delay > opts.MaxDelay {
			delay = opts.MaxDelay
		}

		if opts.Jitter {
			// ±25% jitter
			jitter := float64(delay) * 0.25
			delta := (rand.Float64()*2 - 1) * jitter
			delay = time.Duration(float64(delay) + delta)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return lastErr
}

// IsRetryable returns true only for transient network errors.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	netErr, ok := err.(net.Error)
	if !ok {
		return false
	}
	return netErr.Timeout()
}
