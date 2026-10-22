package dnsengine

import (
	"sync"
	"time"

	"blockads-desktop/internal/cache"
)

// CircuitState represents the current health state of an upstream resolver.
type CircuitState string

const (
	CircuitClosed   CircuitState = "CLOSED"    // Healthy: requests pass through
	CircuitOpen     CircuitState = "OPEN"      // Down: requests fail fast without network I/O
	CircuitHalfOpen CircuitState = "HALF_OPEN" // Probing: single request allowed to test recovery
)

const (
	DefaultFailureThreshold = 3
	DefaultBaseBackoff      = 2 * time.Second
	MinBackoff              = 1 * time.Second   // RFC 9520 minimum
	MaxBackoff              = 300 * time.Second // RFC 9520 maximum (5 minutes)
)

// CircuitBreakerConfig configures an upstream circuit breaker.
type CircuitBreakerConfig struct {
	FailureThreshold int
	BaseBackoff      time.Duration
	MaxBackoff       time.Duration
	Clock            cache.Clock
}

// CircuitBreaker protects upstream resolvers from query storms during outages (RFC 9520).
type CircuitBreaker struct {
	mu               sync.Mutex
	state            CircuitState
	consecutiveFails int
	failureThreshold int
	currentBackoff   time.Duration
	baseBackoff      time.Duration
	maxBackoff       time.Duration
	openUntil        time.Time
	clock            cache.Clock
}

// NewCircuitBreaker creates a new upstream circuit breaker.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	thresh := cfg.FailureThreshold
	if thresh <= 0 {
		thresh = DefaultFailureThreshold
	}

	base := cfg.BaseBackoff
	if base < MinBackoff {
		base = DefaultBaseBackoff
	}

	maxB := cfg.MaxBackoff
	if maxB <= 0 || maxB > MaxBackoff {
		maxB = MaxBackoff
	}

	clk := cfg.Clock
	if clk == nil {
		clk = cache.RealClock{}
	}

	return &CircuitBreaker{
		state:            CircuitClosed,
		failureThreshold: thresh,
		baseBackoff:      base,
		maxBackoff:       maxB,
		currentBackoff:   base,
		clock:            clk,
	}
}

// Allow checks if a request is allowed to reach the upstream resolver.
// Returns true if allowed (CLOSED or probing HALF_OPEN), or false if blocked (OPEN).
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := cb.clock.Now()

	switch cb.state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		if now.After(cb.openUntil) || now.Equal(cb.openUntil) {
			// Backoff expired: enter HALF_OPEN and permit single probe
			cb.state = CircuitHalfOpen
			return true
		}
		return false

	case CircuitHalfOpen:
		// A probe is already in flight; reject other queries until probe completes
		return false
	}

	return true
}

// RecordSuccess notifies the circuit breaker that an upstream resolution succeeded.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitClosed
	cb.consecutiveFails = 0
	cb.currentBackoff = cb.baseBackoff
}

// RecordFailure notifies the circuit breaker that an upstream resolution failed.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveFails++

	if cb.state == CircuitHalfOpen {
		// Probe failed: backoff exponentially up to maxBackoff
		cb.currentBackoff *= 2
		if cb.currentBackoff > cb.maxBackoff {
			cb.currentBackoff = cb.maxBackoff
		}
		cb.state = CircuitOpen
		cb.openUntil = cb.clock.Now().Add(cb.currentBackoff)
		return
	}

	if cb.consecutiveFails >= cb.failureThreshold {
		cb.state = CircuitOpen
		cb.openUntil = cb.clock.Now().Add(cb.currentBackoff)
	}
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Reset forcibly resets the circuit breaker to CLOSED.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitClosed
	cb.consecutiveFails = 0
	cb.currentBackoff = cb.baseBackoff
}
