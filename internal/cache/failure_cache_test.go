package cache_test

import (
	"testing"
	"time"

	"blockads-desktop/internal/cache"

	"github.com/miekg/dns"
)

func TestFailureCache_SetGetAndExpiration(t *testing.T) {
	mockClock := cache.NewMockClock(time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC))
	fc := cache.NewFailureCache(5*time.Second, 100, mockClock)

	q := new(dns.Msg)
	q.SetQuestion("failing-upstream.com.", dns.TypeA)

	// Initial lookup -> Miss
	if _, ok := fc.Get(q); ok {
		t.Fatalf("expected miss on empty failure cache")
	}

	// Store SERVFAIL failure
	stored := fc.Set(q, cache.FailureCategorySERVFAIL, "upstream returned RCODE 2")
	if !stored {
		t.Fatalf("expected failure entry to be stored")
	}

	// Lookup -> Hit within TTL
	entry, ok := fc.Get(q)
	if !ok {
		t.Fatalf("expected failure cache hit")
	}
	if entry.Category != cache.FailureCategorySERVFAIL {
		t.Errorf("expected category SERVFAIL, got %s", entry.Category)
	}
	if entry.Reason != "upstream returned RCODE 2" {
		t.Errorf("expected reason 'upstream returned RCODE 2', got %q", entry.Reason)
	}

	// Advance clock by 3 seconds -> Still Hit
	mockClock.Advance(3 * time.Second)
	if _, ok := fc.Get(q); !ok {
		t.Fatalf("expected failure cache hit after 3s")
	}

	// Advance clock by 3 more seconds (total 6s > 5s TTL) -> Expired, Miss
	mockClock.Advance(3 * time.Second)
	if _, ok := fc.Get(q); ok {
		t.Fatalf("expected failure cache entry to expire after 5s TTL")
	}

	stats := fc.Stats()
	if stats.Hits != 2 || stats.Misses != 2 || stats.Expirations != 1 {
		t.Errorf("unexpected failure stats: %+v", stats)
	}
}

func TestFailureCache_LRUEviction(t *testing.T) {
	mockClock := cache.NewMockClock(time.Now())
	fc := cache.NewFailureCache(10*time.Second, 2, mockClock)

	mkQ := func(name string) *dns.Msg {
		q := new(dns.Msg)
		q.SetQuestion(name+".", dns.TypeA)
		return q
	}

	q1 := mkQ("fail1")
	q2 := mkQ("fail2")
	q3 := mkQ("fail3")

	fc.Set(q1, cache.FailureCategoryTimeout, "timeout")
	fc.Set(q2, cache.FailureCategoryTransport, "conn reset")

	// Access q1 (making q2 LRU)
	fc.Get(q1)

	// Add q3 -> q2 should be evicted
	fc.Set(q3, cache.FailureCategorySERVFAIL, "servfail")

	if _, ok := fc.Get(q2); ok {
		t.Fatalf("expected q2 to be evicted as LRU")
	}
	if _, ok := fc.Get(q1); !ok {
		t.Fatalf("expected q1 to remain in failure cache")
	}
	if _, ok := fc.Get(q3); !ok {
		t.Fatalf("expected q3 to be in failure cache")
	}

	if fc.Stats().Evictions != 1 {
		t.Errorf("expected 1 eviction, got %d", fc.Stats().Evictions)
	}
}

func TestFailureCache_TTLClamping(t *testing.T) {
	mockClock := cache.NewMockClock(time.Now())

	// Sub-minimum TTL should clamp to default (5s)
	fcShort := cache.NewFailureCache(100*time.Millisecond, 10, mockClock)
	q := new(dns.Msg)
	q.SetQuestion("clamp.example.com.", dns.TypeA)
	fcShort.Set(q, cache.FailureCategoryTimeout, "timeout")

	// Advance 2 seconds: should still be present because clamped to DefaultFailureTTL (5s)
	mockClock.Advance(2 * time.Second)
	if _, ok := fcShort.Get(q); !ok {
		t.Fatalf("expected failure cache entry to still be present (clamped to at least 1s/default)")
	}
}
