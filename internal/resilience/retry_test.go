package resilience

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

// mockNetError implements net.Error for testing.
type mockNetError struct {
	timeout   bool
	temporary bool
}

func (e *mockNetError) Error() string   { return "mock net error" }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }

var _ net.Error = (*mockNetError)(nil)

func TestRetrySuccessOnSecondAttempt(t *testing.T) {
	attempts := 0
	err := Do(context.Background(), RetryOpts{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		Jitter:      false,
	}, func() error {
		attempts++
		if attempts < 2 {
			return &mockNetError{timeout: true}
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestRetryMaxAttemptsExceeded(t *testing.T) {
	attempts := 0
	err := Do(context.Background(), RetryOpts{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		Jitter:      false,
	}, func() error {
		attempts++
		return &mockNetError{timeout: true}
	})

	if err == nil {
		t.Fatal("expected error after max attempts")
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryNonRetryableError(t *testing.T) {
	attempts := 0
	err := Do(context.Background(), RetryOpts{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
	}, func() error {
		attempts++
		return errors.New("not a net error")
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt for non-retryable error, got %d", attempts)
	}
}

func TestRetryContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	err := Do(ctx, RetryOpts{
		MaxAttempts: 10,
		BaseDelay:   50 * time.Millisecond,
		Jitter:      false,
	}, func() error {
		attempts++
		return &mockNetError{timeout: true}
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestIsRetryable(t *testing.T) {
	if IsRetryable(nil) {
		t.Error("nil should not be retryable")
	}
	if IsRetryable(errors.New("plain error")) {
		t.Error("plain error should not be retryable")
	}
	if !IsRetryable(&mockNetError{timeout: true}) {
		t.Error("timeout net error should be retryable")
	}
	if IsRetryable(&mockNetError{timeout: false}) {
		t.Error("non-timeout net error should not be retryable")
	}
}
