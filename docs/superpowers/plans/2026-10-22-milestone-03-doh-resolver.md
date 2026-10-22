# Milestone 03: Production DNS-over-HTTPS (DoH) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a production-grade RFC 8484 DNS-over-HTTPS (DoH) resolver package in `internal/upstream` that satisfies `dnsengine.Resolver`, integrates with `dnsengine.Handler`, enforces strict TLS validation, separates bootstrap endpoint dialing from hostname verification, and handles timeouts, cancellation, and explicit SERVFAIL failures.

**Architecture:** A pure Go package `internal/upstream` containing `DoHResolver` with a pooled HTTP client and transport. Endpoint dialing separates the physical IP destination (static override or bootstrap DNS) from the logical HTTPS hostname, strictly preserving TLS SNI and certificate validation. Bounded request sizes, zero plaintext user-query fallback, and structured upstream diagnostics.

**Tech Stack:** Go 1.25+, `github.com/miekg/dns v1.1.73`, standard library `net/http`, `crypto/tls`, `crypto/x509`.

**Spec:** Milestone 03 requirements prompt and RFC 8484.

## Global Constraints

- Default upstream: `https://dns.caliph.dev/dns-query`.
- Media type: `application/dns-message` (POST method).
- Normalizes outbound DNS ID to 0 for RFC 8484 HTTP caching compatibility, restoring original client ID on return.
- Never set `InsecureSkipVerify: true`. System root CAs used by default.
- TLS negotiation: Minimum TLS 1.2 (`tls.VersionTLS12`), negotiating TLS 1.3 when supported.
- Bootstrap DNS is strictly scoped to resolving the DoH server's own hostname; user queries are never sent via plaintext fallback.
- Bootstrap endpoint dialing overrides connection target without altering TLS SNI or HTTP Host authority.
- Upstream failures, timeouts, and malformed responses return explicit errors that `dnsengine.Handler` converts to `SERVFAIL`.
- No Windows network adapter configuration, no registry edits, no services, no drivers.

## Review Focus

- Hostname mismatch / untrusted CA returns an explicit error without falling back to plaintext.
- Context cancellation and timeouts properly abort in-flight HTTP requests and drain/close response bodies.
- Oversized response bodies (> 64KB) and non-200 HTTP codes (400, 429, 500, 503) are safely rejected.
- Client transaction ID correlation: outbound ID normalized to 0 in wire format, original ID restored in client response.
- Bootstrap dialing preserves logical SNI: connection is routed to bootstrap IP, but TLS handshake verifies certificate against DoH hostname.

---

### Task 1: TLS Configuration & Endpoint Dialing (`internal/upstream/tls.go`, `bootstrap.go`)

**Files:**
- Create: `internal/upstream/tls.go`
- Create: `internal/upstream/bootstrap.go`
- Test: `internal/upstream/bootstrap_test.go`

**Interfaces:**
- `NewTLSConfig(serverName string, rootCAs *x509.CertPool) *tls.Config`
- `EndpointDialer { Hostname string, OverrideIP string, BootstrapDNS string, Timeout time.Duration }`
- `EndpointDialer.DialContext(ctx context.Context, network, addr string) (net.Conn, error)`
- `EndpointDialer.ResolveBootstrap(ctx context.Context) (string, error)`

- [x] **Step 1: Write failing tests in `internal/upstream/bootstrap_test.go`**
  - Test TLS config enforces TLS 1.2 minimum and sets `ServerName`.
  - Test `EndpointDialer` routes connection to `OverrideIP` while leaving `addr` parameter logically intact.
  - Test `EndpointDialer` rejects non-TCP networks.
  - Test bootstrap DNS lookup queries external resolver and caches IP.
- [x] **Step 2: Run `go test ./internal/upstream` to verify failure**
- [x] **Step 3: Implement `tls.go` and `bootstrap.go`**
- [x] **Step 4: Run `go test -v ./internal/upstream` to verify pass**

---

### Task 2: RFC 8484 DoH Transport & Resolver (`internal/upstream/doh.go`, `diagnostics.go`)

**Files:**
- Create: `internal/upstream/diagnostics.go`
- Create: `internal/upstream/doh.go`
- Test: `internal/upstream/doh_test.go`

**Interfaces:**
- `Config { EndpointURL string, OverrideIP string, BootstrapDNS string, Timeout time.Duration, RootCAs *x509.CertPool, Transport http.RoundTripper }`
- `DoHResolver: NewDoHResolver(cfg Config) (*DoHResolver, error)`
- `DoHResolver.Resolve(ctx context.Context, req *dns.Msg) (*dns.Msg, error)`
- `DoHResolver.Diagnostics() Diagnostics`

- [x] **Step 1: Write failing tests in `internal/upstream/doh_test.go` with local `httptest.Server`**
  - Test valid A, AAAA, and HTTPS queries returning wire-format DNS responses.
  - Test DNS Rcode propagation (NXDOMAIN, SERVFAIL returned as valid DNS response message).
  - Test ID normalization to 0 and restoration of original client transaction ID.
  - Test HTTP status error (400, 429, 500, 503) returns explicit error.
  - Test unexpected content-type (e.g. `text/html`) returns explicit error.
  - Test empty or malformed DNS wire format body returns explicit error.
  - Test oversized response body (> 65536 bytes) returns error.
  - Test redirect (301/302) is rejected by `CheckRedirect`.
  - Test context cancellation and timeout aborts request cleanly.
  - Test TLS certificate validation fails when server presents untrusted/mismatched certificate.
  - Test concurrent queries succeed on shared reusable transport.
- [x] **Step 2: Run `go test ./internal/upstream` to verify failure**
- [x] **Step 3: Implement `diagnostics.go` and `doh.go`**
- [x] **Step 4: Run `go test -v ./internal/upstream` to verify pass**

---

### Task 3: DNS Engine Handler Integration (`internal/dnsengine/doh_integration_test.go`)

**Files:**
- Create: `internal/dnsengine/doh_integration_test.go`
- Modify: `internal/dnsengine/handler.go` (if minor refinement needed)

- [x] **Step 1: Write failing integration tests in `internal/dnsengine/doh_integration_test.go`**
  - Test `dnsengine.Handler` with `upstream.DoHResolver` resolving permitted query through local HTTPS test server.
  - Test `dnsengine.Handler` with blocked query returns sinkhole `0.0.0.0` without invoking DoH resolver.
  - Test `dnsengine.Handler` with failing DoH resolver returns `SERVFAIL` without plaintext fallback.
- [x] **Step 2: Run `go test -v ./internal/dnsengine` to verify integration tests pass**

---

### Task 4: End-to-End Verification & Documentation Updates

**Files:**
- Modify: `docs/architecture.md`
- Modify: `docs/roadmap.md`
- Modify: `docs/threat-model.md`
- Modify: `README.md`

- [x] **Step 1: Run full test suite (`go test -count=1 ./...`)**
- [x] **Step 2: Run `go vet ./...`**
- [x] **Step 3: Run `gofmt -s -l .`**
- [x] **Step 4: Run frontend build (`npm run build`)**
- [x] **Step 5: Run Wails production build (`wails build`)**
- [x] **Step 6: Update documentation files to mark Milestone 03 completed**
