package resilience

import (
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned when the circuit breaker is in the Open state.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// State represents the circuit breaker state.
type State int

const (
	StateClosed   State = iota // normal operation
	StateOpen                  // failing, rejecting requests
	StateHalfOpen              // testing with limited requests
)

// CircuitBreakerOpts configures the circuit breaker behavior.
type CircuitBreakerOpts struct {
	Threshold          int           // consecutive failures before opening (default 5)
	Timeout            time.Duration // time in Open before transitioning to HalfOpen (default 30s)
	HalfOpenMaxRequests int          // max test requests in HalfOpen (default 1)
}

func (o *CircuitBreakerOpts) withDefaults() CircuitBreakerOpts {
	out := *o
	if out.Threshold <= 0 {
		out.Threshold = 5
	}
	if out.Timeout <= 0 {
		out.Timeout = 30 * time.Second
	}
	if out.HalfOpenMaxRequests <= 0 {
		out.HalfOpenMaxRequests = 1
	}
	return out
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	mu              sync.Mutex
	state           State
	failures        int
	halfOpenCount   int
	lastFailureTime time.Time
	opts            CircuitBreakerOpts
}

// NewCircuitBreaker creates a new circuit breaker with the given options.
func NewCircuitBreaker(opts CircuitBreakerOpts) *CircuitBreaker {
	opts = opts.withDefaults()
	return &CircuitBreaker{
		state: StateClosed,
		opts:  opts,
	}
}

// State returns the current circuit breaker state.
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.currentState()
}

// currentState returns the state, transitioning Open→HalfOpen if timeout has
// elapsed. Must be called with mu held.
func (cb *CircuitBreaker) currentState() State {
	if cb.state == StateOpen && time.Since(cb.lastFailureTime) >= cb.opts.Timeout {
		cb.state = StateHalfOpen
		cb.halfOpenCount = 0
	}
	return cb.state
}

// Allow checks if a request is allowed. If allowed, it returns a done
// function that the caller must invoke with the result (success=true/false).
// Returns ErrCircuitOpen if the circuit is open.
func (cb *CircuitBreaker) Allow() (done func(success bool), err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.currentState()

	switch state {
	case StateClosed:
		// Allow all requests.
	case StateOpen:
		return nil, ErrCircuitOpen
	case StateHalfOpen:
		if cb.halfOpenCount >= cb.opts.HalfOpenMaxRequests {
			return nil, ErrCircuitOpen
		}
		cb.halfOpenCount++
	}

	return func(success bool) {
		cb.mu.Lock()
		defer cb.mu.Unlock()

		if success {
			cb.failures = 0
			if cb.state == StateHalfOpen {
				cb.state = StateClosed
			}
		} else {
			cb.failures++
			cb.lastFailureTime = time.Now()
			if cb.state == StateHalfOpen || cb.failures >= cb.opts.Threshold {
				cb.state = StateOpen
				cb.halfOpenCount = 0
			}
		}
	}, nil
}
