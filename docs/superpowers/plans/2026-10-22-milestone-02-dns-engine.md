# Milestone 02: Core DNS Engine & Filtering Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a standalone, fully-tested, loopback-bound Go DNS filtering engine with hosts/domain filter parsing, reversed-label trie domain matching, allow/block precedence, DNS sinkholing, resolver abstraction, and UDP/TCP lifecycle management.

**Architecture:** Pure Go packages `internal/filters` (parser, reversed-label trie matcher, precedence rules) and `internal/dnsengine` (DNS protocol handler, sinkhole responder, resolver interface, query observer, UDP/TCP server). Decoupled from Wails desktop UI.

**Tech Stack:** Go 1.27, `github.com/miekg/dns v1.1.73` (BSD 3-Clause).

**Spec:** Milestone 02 requirements prompt.

## Global Constraints

- Loopback only by default (`127.0.0.1:5353`), never bind `0.0.0.0` or LAN by default.
- No Windows network adapter configuration, no registry edits, no services, no drivers.
- Precedence: Most specific rule wins; if equal specificity, ALLOW wins over BLOCK.
- Zero silent plaintext DNS fallback: if no resolver is configured or upstream fails, return `SERVFAIL`.
- Sinkhole policy: `A` -> `0.0.0.0`, `AAAA` -> `::`, other types -> `NOERROR` with empty Answer. No fabricated CNAME/MX/HTTPS. No false DNSSEC AD flags.
- Zero panic on malformed DNS input.

## Review Focus

- Reversed-label boundary check: `example.com` must match `sub.example.com`, but NEVER match `example.com.evil.org`.
- Precedence tie-breaker: ALLOW must override BLOCK when matching identical domain specificity.
- Line-numbered parser diagnostics: malformed lines and unsupported adblock syntax (e.g. `||`, `##`) must produce structured warnings.
- Server port reuse & graceful shutdown: UDP and TCP must bind cleanly and terminate without socket/goroutine leaks.
- Upstream resolver isolation: allowed queries must strictly invoke the `Resolver` interface, returning `SERVFAIL` when unconfigured.

---

### Task 1: Filter List Parser (`internal/filters/parser.go`)

**Files:**
- Create: `internal/filters/parser.go`
- Test: `internal/filters/parser_test.go`

**Interfaces:**
- `ParseRule(line string) (*Rule, error)`
- `ParseFilterList(r io.Reader) (*ParseResult, error)`
- `ParseResult{ Rules: []Rule, AcceptedCount: int, DuplicateCount: int, SkippedCount: int, Warnings: []ParseWarning }`

- [x] **Step 1: Write failing parser tests in `internal/filters/parser_test.go`**
  - Test single & multi-domain hosts lines (`0.0.0.0 a.com b.com`, `127.0.0.1 c.org`, `::1 d.net`).
  - Test plain domain lists (`# comment`, `ads.example.com`).
  - Test CRLF and LF line endings.
  - Test whitespace trimming and trailing root dot (`ads.example.com.`).
  - Test case normalization (`ADS.Example.Com` -> `ads.example.com`).
  - Test deduplication.
  - Test skipping localhost (`localhost`, `127.0.0.1`, `::1`, `0.0.0.0`).
  - Test warnings with line numbers for malformed domains, URLs (`https://...`), and cosmetic rules (`##...`, `||...`).
  - Test empty input and large inputs.
- [x] **Step 2: Run `go test ./internal/filters` to verify parser tests fail**
- [x] **Step 3: Implement `internal/filters/parser.go`**
- [x] **Step 4: Run `go test -v ./internal/filters` to verify parser tests pass**

---

### Task 2: Reversed-Label Trie Matcher (`internal/filters/trie.go`, `matcher.go`)

**Files:**
- Create: `internal/filters/trie.go`
- Create: `internal/filters/matcher.go`
- Test: `internal/filters/matcher_test.go`

**Interfaces:**
- `Action: ActionBlock | ActionAllow | ActionPass`
- `Rule: { Domain string, Action Action, Enabled bool }`
- `MatchResult: { Action Action, MatchedDomain string, RuleSource string }`
- `Matcher: NewMatcher() *Matcher`, `Matcher.AddRule(Rule)`, `Matcher.BuildFromRules([]Rule)`, `Matcher.Match(domain string) MatchResult`

- [x] **Step 1: Write failing matcher tests in `internal/filters/matcher_test.go`**
  - Test exact domain match.
  - Test subdomain match (`sub.example.com` matches `example.com`).
  - Test non-matching substring suffix (`example.com.evil.org` does NOT match `example.com`).
  - Test precedence: most specific wins (`block example.com`, `allow safe.example.com` -> `safe.example.com` is allowed).
  - Test precedence: tie-breaker (`block example.com`, `allow example.com` -> allowed).
  - Test inactive rules are ignored.
  - Test case normalization & trailing dots.
  - Test atomic replacement of matcher index.
- [x] **Step 2: Run `go test ./internal/filters` to verify matcher tests fail**
- [x] **Step 3: Implement `trie.go` and `matcher.go`**
- [x] **Step 4: Run `go test -v ./internal/filters` to verify matcher tests pass**

---

### Task 3: DNS Protocol Handler & Sinkholing (`internal/dnsengine/handler.go`, `response.go`, `resolver.go`)

**Files:**
- Create: `internal/dnsengine/resolver.go`
- Create: `internal/dnsengine/response.go`
- Create: `internal/dnsengine/handler.go`
- Test: `internal/dnsengine/handler_test.go`

**Interfaces:**
- `Resolver interface { Resolve(ctx context.Context, req *dns.Msg) (*dns.Msg, error) }`
- `QueryDecision { Timestamp, Domain, QueryType, Decision, MatchedRule, Reason, UpstreamLatency, Rcode, ClientIP, Protocol }`
- `Handler: NewHandler(matcher *filters.Matcher, resolver Resolver, observer QueryObserver) *Handler`
- `Handler.ServeDNS(w dns.ResponseWriter, r *dns.Msg)`

- [x] **Step 1: Write failing DNS handler tests in `internal/dnsengine/handler_test.go`**
  - Test blocked `A` query returns `0.0.0.0` with documented TTL (60s).
  - Test blocked `AAAA` query returns `::` with documented TTL (60s).
  - Test blocked `HTTPS`/`TXT`/`MX` query returns `NOERROR` with empty Answer section.
  - Test allowed query calls `Resolver` and returns upstream response.
  - Test allowed query with `nil` Resolver returns `SERVFAIL`.
  - Test resolver timeout/error returns `SERVFAIL`.
  - Test request ID and question are preserved in response.
  - Test AD (authenticated data) flag is NOT set.
  - Test malformed messages and 0-question queries return `RcodeFormatError` without panic.
  - Test query observer receives correct decision metadata.
- [x] **Step 2: Run `go test ./internal/dnsengine` to verify handler tests fail**
- [x] **Step 3: Implement `resolver.go`, `response.go`, and `handler.go`**
- [x] **Step 4: Run `go test -v ./internal/dnsengine` to verify handler tests pass**

---

### Task 4: UDP/TCP Server Lifecycle (`internal/dnsengine/server.go`)

**Files:**
- Create: `internal/dnsengine/server.go`
- Test: `internal/dnsengine/server_test.go`

**Interfaces:**
- `ServerConfig { ListenAddr string, Port int, Handler dns.Handler }`
- `Server: NewServer(cfg ServerConfig) *Server`
- `Server.Start() error`
- `Server.Shutdown(ctx context.Context) error`
- `Server.Addr() string`

- [x] **Step 1: Write failing server lifecycle and integration tests in `internal/dnsengine/server_test.go`**
  - Test server binds on dynamic loopback port (`127.0.0.1:0`).
  - Test real UDP DNS query against loopback server using `dns.Client`.
  - Test real TCP DNS query against loopback server using `dns.Client`.
  - Test duplicate start error.
  - Test graceful shutdown releases ports cleanly.
- [x] **Step 2: Run `go test ./internal/dnsengine` to verify server tests fail**
- [x] **Step 3: Implement `server.go` managing concurrent UDP and TCP listeners**
- [x] **Step 4: Run `go test -v ./internal/dnsengine` to verify server tests pass**

---

### Task 5: End-to-End Verification & Documentation Updates

**Files:**
- Modify: `docs/architecture.md`
- Modify: `docs/roadmap.md`
- Modify: `docs/threat-model.md`
- Modify: `README.md`

- [x] **Step 1: Run full test suite with race detection (`go test -v ./...`)**
- [x] **Step 2: Run `go vet ./...`**
- [x] **Step 3: Run frontend production build (`npm run build`) to ensure no regression**
- [x] **Step 4: Run Wails production build (`wails build`)**
- [x] **Step 5: Update documentation files (`architecture.md`, `roadmap.md`, `threat-model.md`, `README.md`)**
