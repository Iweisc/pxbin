package resilience

import (
	"testing"
	"time"
)

func TestCircuitBreakerClosedToOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerOpts{Threshold: 3, Timeout: 100 * time.Millisecond})

	// 3 consecutive failures should open the circuit.
	for i := 0; i < 3; i++ {
		done, err := cb.Allow()
		if err != nil {
			t.Fatalf("Allow on attempt %d: %v", i, err)
		}
		done(false)
	}

	if cb.State() != StateOpen {
		t.Fatalf("expected StateOpen, got %v", cb.State())
	}

	// Should reject.
	_, err := cb.Allow()
	if err != ErrCircuitOpen {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreakerOpenToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerOpts{Threshold: 2, Timeout: 50 * time.Millisecond})

	// Open the circuit.
	for i := 0; i < 2; i++ {
		done, _ := cb.Allow()
		done(false)
	}
	if cb.State() != StateOpen {
		t.Fatal("expected open")
	}

	// Wait for timeout.
	time.Sleep(60 * time.Millisecond)

	// Should transition to HalfOpen.
	if cb.State() != StateHalfOpen {
		t.Fatalf("expected StateHalfOpen, got %v", cb.State())
	}
}

func TestCircuitBreakerHalfOpenSuccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerOpts{Threshold: 2, Timeout: 50 * time.Millisecond, HalfOpenMaxRequests: 1})

	// Open the circuit.
	for i := 0; i < 2; i++ {
		done, _ := cb.Allow()
		done(false)
	}

	time.Sleep(60 * time.Millisecond)

	// HalfOpen — allow one test request.
	done, err := cb.Allow()
	if err != nil {
		t.Fatalf("Allow in HalfOpen: %v", err)
	}

	// Success should close the circuit.
	done(true)
	if cb.State() != StateClosed {
		t.Fatalf("expected StateClosed after success in HalfOpen, got %v", cb.State())
	}
}

func TestCircuitBreakerHalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerOpts{Threshold: 2, Timeout: 50 * time.Millisecond, HalfOpenMaxRequests: 1})

	// Open the circuit.
	for i := 0; i < 2; i++ {
		done, _ := cb.Allow()
		done(false)
	}

	time.Sleep(60 * time.Millisecond)

	// HalfOpen test request fails → back to Open.
	done, err := cb.Allow()
	if err != nil {
		t.Fatalf("Allow in HalfOpen: %v", err)
	}
	done(false)

	if cb.State() != StateOpen {
		t.Fatalf("expected StateOpen after failure in HalfOpen, got %v", cb.State())
	}
}

func TestCircuitBreakerSuccessResets(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerOpts{Threshold: 3})

	// 2 failures, then a success.
	for i := 0; i < 2; i++ {
		done, _ := cb.Allow()
		done(false)
	}
	done, _ := cb.Allow()
	done(true)

	// 2 more failures — should NOT open (counter was reset).
	for i := 0; i < 2; i++ {
		d, _ := cb.Allow()
		d(false)
	}

	if cb.State() != StateClosed {
		t.Fatal("circuit should still be closed after reset + 2 failures")
	}
}
