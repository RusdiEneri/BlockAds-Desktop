# Milestone 04A: DNS Cache Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a bounded, thread-safe, in-memory LRU DNS response cache (`internal/cache`) supporting positive and negative TTL caching, TTL decrementing, LRU eviction, deterministic testing with mock clocks, and integration into `dnsengine.Handler`.

**Architecture:** A standalone `internal/cache` package with `DNSCache`, an exact-question `Key` type, doubly-linked LRU list, RFC 2308 negative caching, and thread-safe stats. Integrated into `dnsengine.Handler` with rule-precedence re-evaluation on cache hits.

**Tech Stack:** Go 1.25+, `github.com/miekg/dns v1.1.73`, standard library `sync`, `time`.

**Spec:** Milestone 04A requirements prompt.

## Global Constraints

- Do not implement DNS rewrites, SafeSearch, Safe Browsing, SQLite persistence, Windows Service, system tray, or network adapter changes in this milestone.
- Cache key must safely differentiate query name (canonical lowercase without trailing dot), query type, query class, and DNSSEC OK (DO) flag.
- Safe message copying: stored and returned DNS messages must be deep-copied with `msg.Copy()` to prevent concurrent mutation.
- Decrement TTLs of Answer, Ns, and Extra records on cache hits based on elapsed time.
- Cache negative responses (NXDOMAIN and NODATA) using SOA minimum/TTL according to RFC 2308; clamp with configurable min/max bounds.
- Never cache SERVFAIL, REFUSED, or transport errors.
- Never hold cache locks while performing network I/O or resolving through upstream DoH.
- Block rules take precedence: cached allowed answers must never bypass newly activated block rules.
- Thread-safe statistics tracking: Hits, Misses, Expirations, Evictions, and Current Size.

## Review Focus

- Cache key separation: identical domain queried for A vs AAAA vs HTTPS vs CNAME must never return cross-type answers.
- TTL decrementing on cache hit: records must have updated, non-zero TTLs matching remaining cache lifetime.
- Negative caching: NXDOMAIN and NODATA responses cached with bounded SOA TTL; SERVFAIL strictly un-cached.
- Concurrency and copy safety: concurrent requests reading and writing to cache must never cause race conditions or corrupt wire buffers.
- Rule precedence on cache hit: an allowed domain cached in memory must immediately be blocked if the matcher adds a block rule.

---

### Task 1: Clock Abstraction & Cache Key Types (`internal/cache/clock.go`, `key.go`)

**Files:**
- Create: `internal/cache/clock.go`
- Create: `internal/cache/key.go`
- Test: `internal/cache/key_test.go`

**Interfaces:**
- `Clock interface { Now() time.Time }`
- `RealClock struct {}`, `MockClock struct { mu sync.Mutex, now time.Time }`
- `Key struct { Name string, Qtype uint16, Qclass uint16, DO bool }`
- `NewKey(req *dns.Msg) (Key, bool)`

- [ ] **Step 1: Write failing key and clock tests in `internal/cache/key_test.go`**
  - Test Key creation normalizes domain to lowercase and strips trailing dot.
  - Test Key separates by Qtype (A vs AAAA vs HTTPS).
  - Test Key separates by Qclass (INET vs CHAOS).
  - Test Key detects DNSSEC OK (DO) flag from EDNS0 OPT RR.
  - Test MockClock advances time deterministically.
- [ ] **Step 2: Run `go test ./internal/cache` to verify failure**
- [ ] **Step 3: Implement `clock.go` and `key.go`**
- [ ] **Step 4: Run `go test -v ./internal/cache` to verify pass**

---

### Task 2: Core LRU DNS Cache Engine (`internal/cache/cache.go`, `ttl.go`)

**Files:**
- Create: `internal/cache/ttl.go`
- Create: `internal/cache/cache.go`
- Test: `internal/cache/cache_test.go`

**Interfaces:**
- `Config struct { MaxEntries int, MinTTL time.Duration, MaxTTL time.Duration, DefaultNegativeTTL time.Duration, MaxNegativeTTL time.Duration, Clock Clock }`
- `Stats struct { Hits, Misses, Expirations, Evictions uint64, Entries, Capacity int }`
- `DNSCache: New(cfg Config) *DNSCache`
- `DNSCache.Get(req *dns.Msg) (*dns.Msg, bool)`
- `DNSCache.Set(req *dns.Msg, resp *dns.Msg)`
- `DNSCache.Flush()`
- `DNSCache.Stats() Stats`

- [ ] **Step 1: Write failing cache tests in `internal/cache/cache_test.go`**
  - Test cache hit and cache miss.
  - Test positive A and AAAA responses stored and retrieved with deep copy.
  - Test TTL decrementing on cache hit (e.g. 300s TTL after 100s returns 200s TTL).
  - Test TTL expiration (entry expired returns miss, increments expiration count).
  - Test negative caching for NXDOMAIN with SOA TTL.
  - Test negative caching for NODATA (NOERROR with 0 answers).
  - Test SERVFAIL and REFUSED are NOT cached.
  - Test LRU eviction when capacity is exceeded (least recently used evicted).
  - Test Cache key separation (A vs AAAA vs HTTPS).
  - Test `Flush()` clears all entries and resets count.
  - Test thread-safe concurrent lookups and insertions.
- [ ] **Step 2: Run `go test ./internal/cache` to verify failure**
- [ ] **Step 3: Implement `ttl.go` (TTL extraction & clamping) and `cache.go` (LRU doubly linked list + thread-safe map)**
- [ ] **Step 4: Run `go test -v ./internal/cache` to verify pass**
- [ ] **Step 5: Write benchmarks in `internal/cache/cache_bench_test.go` and run them**

---

### Task 3: Handler Integration & Precedence Safety (`internal/dnsengine/handler.go`)

**Files:**
- Modify: `internal/dnsengine/handler.go`
- Test: `internal/dnsengine/cache_integration_test.go`

**Interfaces:**
- Add `Cache *cache.DNSCache` to `HandlerConfig`.
- If `Cache != nil`, evaluate cache before calling resolver.
- Enforce: Rule matcher checked before returning cached response to prevent stale bypasses.

- [ ] **Step 1: Write failing handler cache integration tests in `internal/dnsengine/cache_integration_test.go`**
  - Test allowed query returns cache hit on second query without calling resolver.
  - Test query decision observer records `cache hit` and 0ms latency.
  - Test blocklist update immediately blocks a domain even if previously cached as allowed.
  - Test resolver failures return SERVFAIL and are not cached.
- [ ] **Step 2: Run `go test -v ./internal/dnsengine` to verify failure**
- [ ] **Step 3: Modify `internal/dnsengine/handler.go` to integrate `DNSCache`**
- [ ] **Step 4: Run `go test -v ./internal/dnsengine` to verify pass**

---

### Task 4: End-to-End Verification & Documentation Updates

**Files:**
- Modify: `docs/architecture.md`
- Modify: `docs/roadmap.md`
- Modify: `docs/threat-model.md`
- Modify: `README.md`

- [ ] **Step 1: Run full test suite (`go test -count=1 ./...`)**
- [ ] **Step 2: Run `go vet ./...`**
- [ ] **Step 3: Run `gofmt -s -l .`**
- [ ] **Step 4: Run benchmarks (`go test -run=^$ -bench=. ./internal/cache`)**
- [ ] **Step 5: Run frontend build (`npm run build`)**
- [ ] **Step 6: Run Wails production build (`wails build`)**
- [ ] **Step 7: Update documentation files to reflect Milestone 04A completion**
