package cache

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

const (
	DefaultFailureTTL        = 5 * time.Second
	MinFailureTTL            = 1 * time.Second
	MaxFailureTTL            = 300 * time.Second
	DefaultFailureMaxEntries = 1000
)

// FailureCategory classifies DNS resolution failures per RFC 9520.
type FailureCategory string

const (
	FailureCategorySERVFAIL  FailureCategory = "SERVFAIL"
	FailureCategoryTimeout   FailureCategory = "TIMEOUT"
	FailureCategoryTransport FailureCategory = "TRANSPORT"
)

// FailureEntry represents a cached resolution failure record.
type FailureEntry struct {
	Category  FailureCategory `json:"category"`
	Reason    string          `json:"reason"`
	StoredAt  time.Time       `json:"storedAt"`
	ExpiresAt time.Time       `json:"expiresAt"`
}

// FailureStats tracks statistics for the RFC 9520 resolution failure cache.
type FailureStats struct {
	Hits        uint64 `json:"hits"`
	Misses      uint64 `json:"misses"`
	Stores      uint64 `json:"stores"`
	Expirations uint64 `json:"expirations"`
	Evictions   uint64 `json:"evictions"`
	Size        int    `json:"size"`
	Capacity    int    `json:"capacity"`
}

type failureNode struct {
	key   Key
	entry FailureEntry
	prev  *failureNode
	next  *failureNode
}

// FailureCache is a bounded LRU cache for DNS resolution failures (RFC 9520).
// It prevents upstream query storms when authoritative servers or DoH endpoints fail.
type FailureCache struct {
	ttl        time.Duration
	maxEntries int
	clock      Clock

	mu    sync.Mutex
	table map[Key]*failureNode
	head  *failureNode
	tail  *failureNode

	hits        atomic.Uint64
	misses      atomic.Uint64
	stores      atomic.Uint64
	expirations atomic.Uint64
	evictions   atomic.Uint64
}

// NewFailureCache initializes a new FailureCache with bounded TTL and capacity.
func NewFailureCache(ttl time.Duration, maxEntries int, clock Clock) *FailureCache {
	if ttl < MinFailureTTL {
		ttl = DefaultFailureTTL
	}
	if ttl > MaxFailureTTL {
		ttl = MaxFailureTTL
	}
	if maxEntries <= 0 {
		maxEntries = DefaultFailureMaxEntries
	}
	if clock == nil {
		clock = RealClock{}
	}

	head := &failureNode{}
	tail := &failureNode{}
	head.next = tail
	tail.prev = head

	return &FailureCache{
		ttl:        ttl,
		maxEntries: maxEntries,
		clock:      clock,
		table:      make(map[Key]*failureNode, maxEntries),
		head:       head,
		tail:       tail,
	}
}

// Get checks if a resolution failure is currently cached for the given request.
func (fc *FailureCache) Get(req *dns.Msg) (FailureEntry, bool) {
	if req == nil || !IsCacheableQuery(req) {
		fc.misses.Add(1)
		return FailureEntry{}, false
	}

	key, ok := NewKey(req)
	if !ok {
		fc.misses.Add(1)
		return FailureEntry{}, false
	}

	fc.mu.Lock()
	defer fc.mu.Unlock()

	node, exists := fc.table[key]
	if !exists {
		fc.misses.Add(1)
		return FailureEntry{}, false
	}

	now := fc.clock.Now()
	if !now.Before(node.entry.ExpiresAt) {
		// Expired failure entry
		fc.removeNode(node)
		delete(fc.table, key)
		fc.expirations.Add(1)
		fc.misses.Add(1)
		return FailureEntry{}, false
	}

	fc.moveToHead(node)
	fc.hits.Add(1)
	return node.entry, true
}

// Set stores a resolution failure entry with the configured failure TTL.
func (fc *FailureCache) Set(req *dns.Msg, cat FailureCategory, reason string) bool {
	if req == nil || !IsCacheableQuery(req) {
		return false
	}

	key, ok := NewKey(req)
	if !ok {
		return false
	}

	now := fc.clock.Now()
	expiresAt := now.Add(fc.ttl)

	entry := FailureEntry{
		Category:  cat,
		Reason:    reason,
		StoredAt:  now,
		ExpiresAt: expiresAt,
	}

	fc.mu.Lock()
	defer fc.mu.Unlock()

	if node, exists := fc.table[key]; exists {
		node.entry = entry
		fc.moveToHead(node)
		fc.stores.Add(1)
		return true
	}

	if len(fc.table) >= fc.maxEntries {
		oldest := fc.tail.prev
		if oldest != fc.head {
			fc.removeNode(oldest)
			delete(fc.table, oldest.key)
			fc.evictions.Add(1)
		}
	}

	newNode := &failureNode{
		key:   key,
		entry: entry,
	}
	fc.addToHead(newNode)
	fc.table[key] = newNode
	fc.stores.Add(1)
	return true
}

// Flush clears all cached failure entries.
func (fc *FailureCache) Flush() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.table = make(map[Key]*failureNode, fc.maxEntries)
	fc.head.next = fc.tail
	fc.tail.prev = fc.head
}

// Stats returns a snapshot of failure cache counters.
func (fc *FailureCache) Stats() FailureStats {
	fc.mu.Lock()
	size := len(fc.table)
	fc.mu.Unlock()

	return FailureStats{
		Hits:        fc.hits.Load(),
		Misses:      fc.misses.Load(),
		Stores:      fc.stores.Load(),
		Expirations: fc.expirations.Load(),
		Evictions:   fc.evictions.Load(),
		Size:        size,
		Capacity:    fc.maxEntries,
	}
}

func (fc *FailureCache) addToHead(node *failureNode) {
	node.next = fc.head.next
	node.prev = fc.head
	fc.head.next.prev = node
	fc.head.next = node
}

func (fc *FailureCache) removeNode(node *failureNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
	node.prev = nil
	node.next = nil
}

func (fc *FailureCache) moveToHead(node *failureNode) {
	fc.removeNode(node)
	fc.addToHead(node)
}
