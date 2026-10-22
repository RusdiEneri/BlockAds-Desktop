package cache

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

// Config configures the bounded LRU DNS cache.
type Config struct {
	MaxEntries         int
	MinTTL             time.Duration
	MaxTTL             time.Duration
	DefaultNegativeTTL time.Duration // Deprecated in favor of strict RFC 2308 SOA metadata
	MaxNegativeTTL     time.Duration
	Clock              Clock
}

// Stats holds snapshot statistics of the DNS cache.
type Stats struct {
	Hits        uint64 `json:"hits"`
	Misses      uint64 `json:"misses"`
	Expirations uint64 `json:"expirations"`
	Evictions   uint64 `json:"evictions"`
	Size        int    `json:"size"`
	Capacity    int    `json:"capacity"`
}

type lruNode struct {
	key       Key
	msg       *dns.Msg
	storedAt  time.Time
	expiresAt time.Time
	prev      *lruNode
	next      *lruNode
}

// DNSCache is a thread-safe, bounded, in-memory LRU DNS cache.
type DNSCache struct {
	cfg   Config
	clock Clock

	mu    sync.Mutex
	table map[Key]*lruNode
	head  *lruNode // dummy head
	tail  *lruNode // dummy tail

	hits        atomic.Uint64
	misses      atomic.Uint64
	expirations atomic.Uint64
	evictions   atomic.Uint64
}

// NewDNSCache initializes a new DNSCache with given configuration.
func NewDNSCache(cfg Config) *DNSCache {
	if cfg.MaxEntries <= 0 {
		cfg.MaxEntries = DefaultMaxEntries
	}
	if cfg.Clock == nil {
		cfg.Clock = RealClock{}
	}
	if cfg.MinTTL <= 0 {
		cfg.MinTTL = DefaultMinTTL
	}
	if cfg.MaxTTL <= 0 {
		cfg.MaxTTL = DefaultMaxTTL
	}
	if cfg.MaxNegativeTTL <= 0 {
		cfg.MaxNegativeTTL = DefaultMaxNegativeTTL
	}

	head := &lruNode{}
	tail := &lruNode{}
	head.next = tail
	tail.prev = head

	return &DNSCache{
		cfg:   cfg,
		clock: cfg.Clock,
		table: make(map[Key]*lruNode, cfg.MaxEntries),
		head:  head,
		tail:  tail,
	}
}

// Get retrieves a cached DNS response for the given request message.
// If found and unexpired, a decremented deep copy is returned with matching ID,
// preserved Question casing, and client-tailored EDNS metadata.
// If expired, uncacheable, or missing, nil is returned.
func (c *DNSCache) Get(req *dns.Msg) *dns.Msg {
	if req == nil || !IsCacheableQuery(req) {
		c.misses.Add(1)
		return nil
	}

	key, ok := NewKey(req)
	if !ok {
		c.misses.Add(1)
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. Try exact key lookup (<Name, Qtype, Qclass, DO, CD>)
	node, exists := c.table[key]

	// 2. Fallback to name-level NXDOMAIN key per RFC 2308 Section 3
	// An NXDOMAIN response without CNAME answers applies to the whole name.
	if !exists {
		nameKey := NewNameKey(key.Name, key.Qclass, key.DO, key.CD)
		if n, nExists := c.table[nameKey]; nExists && n.msg.Rcode == dns.RcodeNameError {
			node = n
			exists = true
		}
	}

	if !exists {
		c.misses.Add(1)
		return nil
	}

	now := c.clock.Now()
	if !now.Before(node.expiresAt) {
		// Expired
		c.removeNode(node)
		delete(c.table, node.key)
		c.expirations.Add(1)
		c.misses.Add(1)
		return nil
	}

	// Move to MRU position (head)
	c.moveToHead(node)
	c.hits.Add(1)

	// Safe copy
	out := node.msg.Copy()
	out.Id = req.Id

	// Preserve the current request's original Question section, including 0x20 casing
	out.Question = req.Question

	// Decrement TTL
	elapsed := now.Sub(node.storedAt)
	decrementTTL(out, elapsed)

	// EDNS / OPT Record Normalization (RFC 6891 Section 6.1.1):
	// A response must not blindly replay upstream or previous query OPT records.
	// 1. Strip any existing OPT records from Extra
	cleanExtra := make([]dns.RR, 0, len(out.Extra))
	for _, rr := range out.Extra {
		if rr == nil || rr.Header().Rrtype != dns.TypeOPT {
			cleanExtra = append(cleanExtra, rr)
		}
	}
	out.Extra = cleanExtra

	// 2. If the current client sent EDNS, attach client-tailored OPT
	if reqOpt := req.IsEdns0(); reqOpt != nil {
		out.SetEdns0(reqOpt.UDPSize(), reqOpt.Do())
	}

	return out
}

// Set stores a DNS response in the cache if it meets caching criteria.
// Returns true if stored, or false if not cacheable.
func (c *DNSCache) Set(req *dns.Msg, resp *dns.Msg) bool {
	if req == nil || resp == nil || !IsCacheableQuery(req) {
		return false
	}

	dur, cacheable := determineTTL(resp, c.cfg)
	if !cacheable || dur <= 0 {
		return false
	}

	key, ok := NewKey(req)
	if !ok {
		return false
	}

	storedMsg := resp.Copy()
	now := c.clock.Now()
	expiresAt := now.Add(dur)

	// RFC 2308: In negative caching, clamp SOA record TTL to the negative TTL
	if resp.Rcode == dns.RcodeNameError || (resp.Rcode == dns.RcodeSuccess && len(resp.Answer) == 0) {
		durSec := uint32(dur.Seconds())
		for _, rr := range storedMsg.Ns {
			if soa, ok := rr.(*dns.SOA); ok {
				if soa.Hdr.Ttl > durSec {
					soa.Hdr.Ttl = durSec
				}
			}
		}
	}

	// RFC 2308 Section 3: If this is an NXDOMAIN without CNAME answers,
	// the non-existence applies to the name itself for all types.
	storeKey := key
	if resp.Rcode == dns.RcodeNameError && len(resp.Answer) == 0 {
		storeKey = NewNameKey(key.Name, key.Qclass, key.DO, key.CD)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if node, exists := c.table[storeKey]; exists {
		node.msg = storedMsg
		node.storedAt = now
		node.expiresAt = expiresAt
		c.moveToHead(node)
		return true
	}

	if len(c.table) >= c.cfg.MaxEntries {
		// Evict LRU (tail.prev)
		oldest := c.tail.prev
		if oldest != c.head {
			c.removeNode(oldest)
			delete(c.table, oldest.key)
			c.evictions.Add(1)
		}
	}

	newNode := &lruNode{
		key:       storeKey,
		msg:       storedMsg,
		storedAt:  now,
		expiresAt: expiresAt,
	}
	c.addToHead(newNode)
	c.table[storeKey] = newNode

	return true
}

// Flush removes all cached entries and resets current size.
func (c *DNSCache) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.table = make(map[Key]*lruNode, c.cfg.MaxEntries)
	c.head.next = c.tail
	c.tail.prev = c.head
}

// Stats returns a snapshot of cache metrics.
func (c *DNSCache) Stats() Stats {
	c.mu.Lock()
	size := len(c.table)
	c.mu.Unlock()

	return Stats{
		Hits:        c.hits.Load(),
		Misses:      c.misses.Load(),
		Expirations: c.expirations.Load(),
		Evictions:   c.evictions.Load(),
		Size:        size,
		Capacity:    c.cfg.MaxEntries,
	}
}

func (c *DNSCache) addToHead(node *lruNode) {
	node.next = c.head.next
	node.prev = c.head
	c.head.next.prev = node
	c.head.next = node
}

func (c *DNSCache) removeNode(node *lruNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
	node.prev = nil
	node.next = nil
}

func (c *DNSCache) moveToHead(node *lruNode) {
	c.removeNode(node)
	c.addToHead(node)
}
