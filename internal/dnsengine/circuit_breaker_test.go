package dnsengine_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"

	"blockads-desktop/internal/cache"
	"blockads-desktop/internal/dnsengine"
)

func TestCircuitBreaker_Lifecycle(t *testing.T) {
	mockClock := cache.NewMockClock(time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC))
	cb := dnsengine.NewCircuitBreaker(dnsengine.CircuitBreakerConfig{
		FailureThreshold: 3,
		BaseBackoff:      2 * time.Second,
		MaxBackoff:       10 * time.Second,
		Clock:            mockClock,
	})

	if cb.State() != dnsengine.CircuitClosed {
		t.Fatalf("expected initial state CLOSED")
	}

	// 1. Failures below threshold do not open circuit
	cb.RecordFailure()
	cb.RecordFailure()
	if !cb.Allow() {
		t.Fatalf("expected Allow=true before threshold reached")
	}
	if cb.State() != dnsengine.CircuitClosed {
		t.Fatalf("expected state still CLOSED after 2 failures")
	}

	// 2. 3rd failure trips circuit to OPEN
	cb.RecordFailure()
	if cb.State() != dnsengine.CircuitOpen {
		t.Fatalf("expected state OPEN after 3 failures")
	}
	if cb.Allow() {
		t.Fatalf("expected Allow=false while OPEN")
	}

	// 3. Advance clock by 1s (< 2s backoff) -> still OPEN
	mockClock.Advance(1 * time.Second)
	if cb.Allow() {
		t.Fatalf("expected Allow=false before backoff elapsed")
	}

	// 4. Advance clock by 1.5s (total 2.5s > 2s backoff) -> transitions to HALF_OPEN on Allow()
	mockClock.Advance(1500 * time.Millisecond)
	if !cb.Allow() {
		t.Fatalf("expected probe query to be allowed when entering HALF_OPEN")
	}
	if cb.State() != dnsengine.CircuitHalfOpen {
		t.Fatalf("expected state HALF_OPEN")
	}

	// Subsequent query while probe is in flight is rejected
	if cb.Allow() {
		t.Fatalf("expected second query in HALF_OPEN to be rejected while probe is in flight")
	}

	// 5. Probe succeeds -> resets to CLOSED
	cb.RecordSuccess()
	if cb.State() != dnsengine.CircuitClosed {
		t.Fatalf("expected state CLOSED after probe success")
	}
	if !cb.Allow() {
		t.Fatalf("expected Allow=true after reset to CLOSED")
	}
}

func TestCircuitBreaker_ExponentialBackoff(t *testing.T) {
	mockClock := cache.NewMockClock(time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC))
	cb := dnsengine.NewCircuitBreaker(dnsengine.CircuitBreakerConfig{
		FailureThreshold: 2,
		BaseBackoff:      2 * time.Second,
		MaxBackoff:       8 * time.Second,
		Clock:            mockClock,
	})

	// Trip circuit
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != dnsengine.CircuitOpen {
		t.Fatalf("expected OPEN")
	}

	// Advance 2s -> HALF_OPEN probe
	mockClock.Advance(2 * time.Second)
	cb.Allow()

	// Probe fails! Backoff should double from 2s to 4s
	cb.RecordFailure()
	if cb.State() != dnsengine.CircuitOpen {
		t.Fatalf("expected OPEN after failed probe")
	}

	// Advance 3s (< 4s backoff) -> still OPEN
	mockClock.Advance(3 * time.Second)
	if cb.Allow() {
		t.Fatalf("expected still OPEN during 4s backoff")
	}

	// Advance 2s (total 5s > 4s) -> enters HALF_OPEN
	mockClock.Advance(2 * time.Second)
	if !cb.Allow() {
		t.Fatalf("expected Allow=true after 4s backoff")
	}
}

func TestHandler_UpstreamCircuitBreaker_BlocksQueryStormAcrossDifferentDomains(t *testing.T) {
	var resolveCalls atomic.Int32
	mockClock := cache.NewMockClock(time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC))

	fakeResolver := dnsengine.NewFakeResolver(func(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
		resolveCalls.Add(1)
		return nil, errors.New("upstream connection refused")
	})

	cb := dnsengine.NewCircuitBreaker(dnsengine.CircuitBreakerConfig{
		FailureThreshold: 3,
		BaseBackoff:      5 * time.Second,
		Clock:            mockClock,
	})

	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Resolver:       fakeResolver,
		CircuitBreaker: cb,
	})

	// Query 100 DIFFERENT domains while upstream is dead
	for i := 0; i < 100; i++ {
		req := new(dns.Msg)
		req.SetQuestion(fmt.Sprintf("unique-domain-%d.org.", i), dns.TypeA)
		w := newMockResponseWriter("udp", "127.0.0.1:54321")
		handler.ServeDNS(w, req)
		if w.writtenMsg.Rcode != dns.RcodeServerFailure {
			t.Fatalf("expected SERVFAIL, got %d", w.writtenMsg.Rcode)
		}
	}

	// Only 3 queries should have reached the resolver; remaining 97 were short-circuited!
	if resolveCalls.Load() != 3 {
		t.Fatalf("expected exactly 3 resolver calls before circuit tripped, got %d", resolveCalls.Load())
	}
}
