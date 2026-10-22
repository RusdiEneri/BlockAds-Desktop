# BlockAds Desktop — Development Roadmap

This roadmap defines the engineering milestones for BlockAds Desktop, moving from initial desktop shell to a lightweight two-component Windows Service + Wails Desktop architecture inspired by the functional strengths of AdGuard Home.

---

## Milestone 01: Desktop Foundation & Contracts (Completed)
- [x] Wails v2 + React 19 + TypeScript + Vite project scaffolding.
- [x] Restrained neo-brutalist design system defined in `DESIGN.md`.
- [x] Implementation of 6 primary screens:
  - **Dashboard**: Protection status, upstream preview, sample telemetry metrics, recent activity.
  - **Query Log**: Domain search, status/type filters, detailed query records, clear sample data branding.
  - **Filter Lists**: Catalog cards, enabled toggles, metadata inspection, disclaimer notice.
  - **Custom Rules**: Domain validation, allow/block list tabs, add/delete workflows.
  - **Settings**: Upstream DoH URL validation, planned engine controls marked disabled with milestone badges.
  - **Diagnostics & About**: System runtime info, port 53 probe, DoH upstream probe, read-only adapter status.
- [x] Go backend contract with typed DTO models and unit tests.
- [x] Zero system mutations (no adapter alterations, no registry tampering, no drivers).

---

## Milestone 02: Core Go DNS Engine & Filtering (Completed)
- [x] Standalone DNS server listening on `127.0.0.1` (UDP/TCP) using `github.com/miekg/dns v1.1.73`.
- [x] Loopback-only binding enforcement by default; safe dynamic port binding for tests.
- [x] Filter list parser supporting `/etc/hosts` format and line-delimited domain lists with LF/CRLF support.
- [x] Deduplication, comment handling, whitespace/trailing dot normalization, and localhost/special IP skipping.
- [x] Line-numbered diagnostics and warnings for malformed entries, URLs, wildcards, and cosmetic filter syntax.
- [x] Reversed-label trie matcher with strict DNS label boundaries (`example.com` matches `sub.example.com` but never `example.com.evil.org`).
- [x] Deterministic precedence: Most specific rule wins; if equal specificity, ALLOW wins over BLOCK; inactive rules ignored.
- [x] Atomic index reload (`atomic.Pointer[trieIndex]`) guaranteeing corrupt or partially loaded indexes are never exposed.
- [x] DNS protocol validation (single question, Opcode, ClassINET) with zero panics on malformed messages.
- [x] Synthetic sinkhole responses (`0.0.0.0` for A, `::` for AAAA, NOERROR/NODATA for other types; no false DNSSEC AD flags).
- [x] Injected `Resolver` interface for allowed queries with zero plaintext fallback (returns `SERVFAIL` when unconfigured or on failure).
- [x] Structured `QueryDecision` observability event callback.
- [x] Test-driven development with unit, benchmark, and real loopback UDP/TCP integration tests.

---

## Milestone 03: Encrypted DoH Upstream & Resolver (Completed)
- [x] RFC 8484 DNS-over-HTTPS (DoH) client implementation (`internal/upstream`).
- [x] Default upstream: `https://dns.caliph.dev/dns-query` (verified with live connectivity test).
- [x] **Dedicated Bootstrap DNS Mechanism**:
  - Scoped strictly to resolving the DoH server's own endpoint hostname; plaintext bootstrap is acknowledged as plaintext DNS and never used for ordinary user queries.
  - Configurable bootstrap resolver IPs (via `models.AppSettings.BootstrapDNS`), defaulting to public resolvers (`1.1.1.1`, `8.8.8.8`).
  - Direct IP literal DoH endpoints supported without unnecessary bootstrap queries.
  - Resolved endpoint IP caching with automatic re-resolution recovery when CDN/upstream IPs rotate.
- [x] **Standard Secure TLS Negotiation**:
  - Minimum TLS 1.2 (`tls.VersionTLS12`) negotiating TLS 1.3 whenever supported by client and server.
  - Strict system root CA certificate verification (`x509.SystemCertPool`) and SNI preservation (`tls.Config.ServerName = host`).
  - Avoids artificial TLS 1.3-only restrictions that break compatibility with valid corporate or middlebox TLS 1.2 endpoints.
- [x] Context cancellation, request timeout management, and HTTP/2 connection pooling with multiplexing.
- [x] **Zero User-Query Plaintext Fallback**:
  - User browsing queries are never forwarded to unencrypted DNS. If DoH fails or times out, return SERVFAIL immediately.
- [x] Upstream latency benchmarking, round-trip time tracking, and structured diagnostics observability.
- [x] Handler integration: `dnsengine.Handler` directly consumes `upstream.DoHResolver` via `dnsengine.Resolver` interface.

---

## Milestone 04A: Bounded LRU DNS Response Cache (Completed)
- [x] **Thread-Safe In-Memory LRU Cache (`internal/cache`)**:
  - Deterministic LRU eviction when reaching `MaxEntries` limit.
  - Explicit cache key separating canonical lowercase domain name, Qtype, Qclass, and DNSSEC OK (`DO`) bit.
  - DNS TTL semantics with time-based TTL decrementing on cache hits using injectable clock (`Clock` interface, `RealClock`, `MockClock`).
  - RFC 1035 positive caching (A, AAAA, CNAME) and RFC 2308 negative caching (NXDOMAIN, NODATA) respecting SOA `min(TTL, MINIMUM)` limits and configurable negative TTL caps.
  - Non-caching of transient transport errors and SERVFAIL/REFUSED responses.
  - Deep-copy safety (`msg.Copy()`) preventing shared in-place mutations.
  - Rule precedence invariant: Filter rules are evaluated before returning cached answers, ensuring newly activated block rules take effect immediately without cache bypass.
  - Non-blocking resolution: Cache lock is never held during upstream network I/O or DoH calls.
  - Thread-safe statistics reporting (`Hits`, `Misses`, `Expirations`, `Evictions`, `Size`, `Capacity`).
  - High-performance benchmarked operations (< 0.2µs cache hits, sub-30ns misses).
- [x] **Handler Integration (`internal/dnsengine`)**:
  - Dynamic rule changes immediately override cached entries.
- [x] **Correctness Audit & Circuit Breaker (`internal/cache`, `internal/dnsengine` — Milestone 04A.1)**:
  - RFC 1035/2181 zero-TTL compliance: responses with TTL=0 are strictly uncacheable.
  - RFC 2308 negative caching: strictly requires valid SOA metadata; name-level NXDOMAIN caching.
  - EDNS/OPT isolation and 0x20 mixed-casing Question preservation on cache hits.
  - EDNS options audit: bypasses cache for queries containing ECS, DNS Cookies (RFC 7873), or unsupported options.
  - RFC 9520 Upstream Circuit Breaker (`internal/dnsengine/circuit_breaker.go`): detects consecutive upstream transport failures and trips an exponential backoff circuit breaker (1s - 300s), shielding against query storms across arbitrary domains during upstream DoH outages.

---

## Milestone 04B: DNS Rewrites, Persistent Storage & Scheduler
- [x] **DNS Rewrites Subsystem (`internal/rewrites` — Completed)**:
  - Exact and wildcard domain rewrites to custom IPv4/IPv6 addresses (e.g., `nas.home.arpa A 192.168.18.11`, `*.lab.home.arpa A 10.10.10.10`).
  - CNAME aliases and chained resolution (`memos.home.arpa CNAME nas.home.arpa`) with circular loop detection (visited tracking and 8-hop depth cap returning SERVFAIL).
  - RFC 1034 NODATA synthesis when domain exists in rewrites but requested query type does not, preventing local domain leakage upstream.
  - Strict precedence in `ServeDNS`: evaluated before blocklists and before DNS cache, guaranteeing immediate local resolution without stale cache interference.
- [x] **SQLite Persistence Foundation (`internal/storage` — Completed)**:
  - Pure Go SQLite integration (`modernc.org/sqlite v1.36.0`, BSD-3-Clause, zero CGO requirement) operating in WAL mode with normal synchronous mode.
  - Foreign keys enabled, busy timeout 5000ms, single connection pool (`MaxOpenConns(1)`) eliminating database lock contention on Windows.
  - Automated schema migrations engine (`schema_migrations` tracking table) with Version 1 schema for settings, rewrites, custom rules, subscriptions, snapshots, and query events.
  - Filter snapshot storage strategy: SQLite `filter_snapshots` table with atomic ACID transaction linkage to subscription metadata, SHA-256 integrity verification, 25MB size bound, and deduplication.
  - Non-blocking asynchronous query logger (`internal/storage/async_logger.go`): bounded channel queue (capacity 4096), batch inserts in single transactions (100 events or 200ms), dropped-event telemetry, ~13.7ns enqueue overhead on DNS response path with zero disk I/O.
  - Startup reconciliation (`storage.ReconcileActiveState`): restores persistent rewrites, custom rules, and filter snapshots in deterministic order, preserving working active matchers on failure.
- [x] **Filter List Manager & Scheduler (`internal/filtermanager` — Milestone 04B.3 Completed)**:
  - Persistent subscriptions backed by SQLite `filter_subscriptions` and `filter_snapshots` tables.
  - Safe downloader (`client.go`) enforcing HTTPS by default, bounded redirects with HTTPS-to-HTTP downgrade rejection, SSRF protection against loopback/private/link-local/multicast IP destinations (`net.Dialer.Control` IP validation), and strict 25MB response bounds.
  - RFC 9110 HTTP conditional requests with persistent `ETag` (`If-None-Match`) and `Last-Modified` (`If-Modified-Since`) validators.
  - HTTP 304 Not Modified validation preserving valid snapshots and active rules without redundant parsing or matcher rebuilding; automatic unconditional recovery retry if local snapshot is corrupt or missing.
  - HTTP 200 OK processing with SHA-256 integrity verification, HTML/error-page detection and rejection (`ErrHTMLResponse`), format validation (hosts and plain domain lists), and quality checks (`ErrEmptyFilterList`, `ErrExcessiveWarnings`).
  - Safe transactional hot-reload: candidate matcher prepared and verified, snapshot saved atomically in SQLite, and active matcher swapped lock-free via `atomic.Pointer[filters.Matcher]` in `dnsengine.Handler` with zero dropped queries.
  - In-flight deduplication preventing concurrent fetch stampedes on identical subscriptions.
  - Provenance & determinism: disabling a subscription removes only its contributed rules while retaining domains blocked by other active lists; custom allow/block rules maintain top precedence.
  - Context-managed background scheduler (`scheduler.go`) with configurable interval (default 24h), pseudo-random jitter (±10%), bounded download concurrency (semaphore cap of 2), bounded exponential retry for transient errors, and clean context shutdown.
  - Full frontend integration in `FilterListsView.tsx` with honest status indicators (IN MATCHER, PENDING SYNC, DISABLED), last attempt/success timestamps, rule counts, add subscription modal with SSRF validation, delete with cascade, and manual per-list/all-list refresh.
- [x] **SafeSearch DNS Policies (`internal/safesearch` — Milestone 04B.4A Completed)**:
  - Extensible, data-driven provider registry (`internal/safesearch/registry.go`):
    - **Google Search**: Canonical CNAME target `forcesafesearch.google.com` (mapping `www.google.com` and international Google regional search domains).
    - **Microsoft Bing**: Canonical strict target `strict.bing.com` (mapping `www.bing.com` and `edgeservices.bing.com` for Edge sidebar search).
    - **DuckDuckGo**: Canonical safe-search target `safe.duckduckgo.com` (mapping `duckduckgo.com` and `www.duckduckgo.com`).
    - **YouTube**: Strict Restricted Mode canonical target `restrict.youtube.com` (mapping `www.youtube.com`, `m.youtube.com`, `youtubei.googleapis.com`, `youtube.googleapis.com`, and `www.youtube-nocookie.com`).
    - **Yandex**: Truthfully marked unsupported for CNAME rewrite mechanism. Yandex Family Search requires dedicated upstream DNS resolvers (`77.88.8.7` / `77.88.8.3`); CNAME rewriting to `family.yandex.ru` is not officially supported and is rejected with clear architectural diagnostics.
  - **Versioned SQLite Persistence & Migration v2**: `safesearch_policies` table persisting enabled states and modification timestamps.
  - **Deterministic Precedence & Active State Reconciliation**: User-configured rewrites > SafeSearch CNAME rewrites > Custom Allow/Block rules > Filter Subscriptions > Normal Cache & Upstream.
  - **DNS Cache Invalidation**: Automatic response cache flushing upon policy changes prevents cached normal answers from bypassing activated SafeSearch policies.
  - **Thread-Safe Hot-Reload**: Lock-free rewriter swapping via `atomic.Pointer[rewrites.Rewriter]` in `dnsengine.Handler` with zero dropped queries.
  - **Truthful Frontend Settings UI**: Dedicated SafeSearch section in `SettingsView.tsx` with honest runtime status badges (`ACTIVE`, `DISABLED`, `UNSUPPORTED ON THIS MECHANISM`, `PENDING ACTIVATION`), domain coverage metadata, canonical target inspection, doc links, and DNS-only scope disclaimers.
- [x] **Product Completion, Packaging & Release Infrastructure (Milestone 04B.5 Completed)**:
  - **Curated Filter Catalog**: Pre-verified list definitions (AdAway, StevenBlack, OISD, URLhaus, Dan Pollock, Peter Lowe) with category classification and non-duplicative subscription flow.
  - **Manual Filter Addition**: Dynamic metadata extraction (`! Title:`, `! Description:`, `! Homepage:`), format auto-detection, HTML response rejection, and customizable display name.
  - **Canonical URL Normalization & Duplicate Prevention**: Backend & SQLite layer enforcement preventing duplicate filter subscriptions across catalog and manual additions.
  - **Truthful Startup Splash Screen**: Neo-brutalist splash view reporting real initialization stages (Database, DNS Engine, Resolver, Scheduler) with graceful error recovery.
  - **Optional "Start with Windows"**: Non-privileged startup toggle via `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, supporting quoted paths and detecting stale portable paths.
  - **Dual Execution Modes**: Portable mode (`portable.marker`, local `data/` folder, write probe) and installed mode (`%LOCALAPPDATA%\BlockAds Desktop`).
  - **Packaging & CI/CD Release Automation**: Windows NSIS installer configuration with startup cleanup and data preservation; GitHub Actions CI (`ci.yml`) and tag-triggered automated release publishing (`release-windows.yml`).
- [ ] **Milestone 04B.4B: Domain-Based Safe Browsing Threat Feeds**:
  - Dedicated threat intelligence feeds for blocking phishing, malware C2, and fraud domains with clear diagnostic categorization.

---

## Milestone 05: Windows Network Integration & Safe Recovery
- [ ] Network adapter DNS discovery, IPv4 and IPv6 snapshotting to local disk.
- [ ] Windows DNS configuration via PowerShell / Win32 `Set-DnsClientServerAddress` APIs.
- [ ] **Fail-Safe Watchdog & Rollback**:
  - If the engine crashes, terminates unexpectedly, or is killed, an independent watchdog helper immediately restores original DNS servers.
  - Standalone recovery command line utility: `BlockAds-Desktop.exe --restore-dns`.
- [ ] Network change listener (handling adapter transitions such as switching between Wi-Fi and Ethernet or connecting/disconnecting VPNs).
- [ ] Pre-flight port availability verification with clear diagnostic reporting.

---

## Milestone 06: Lightweight Service Architecture & IPC Decoupling
- [ ] **Two-Component Architectural Split**:
  - **Component 1: BlockAds Service (`blockads-service.exe` / `--service`)**:
    - Standalone Windows Service (`golang.org/x/sys/windows/svc`) running in session 0.
    - Manages DNS engine, DoH upstream client, cache, filter matcher, and privileged network settings.
    - Zero WebView2, zero GUI dependencies, minimal idle memory (< 25 MB RAM).
    - Starts automatically at boot; independent of user login state.
  - **Component 2: BlockAds Desktop UI**:
    - Unprivileged Wails v2 + React 19 application running in the user session.
    - Launches on user login only when enabled in settings.
    - Can be opened, closed, or restarted without interrupting active DNS protection.
- [ ] **Secure Inter-Process Communication (IPC)**:
  - Windows Named Pipe (`\\.\pipe\BlockAdsDesktopPipe`) protected by Discretionary Access Control Lists (DACLs).
  - Restricted to current interactive user and Administrator tokens.
  - Structured binary/JSON-RPC protocol for real-time telemetry, configuration sync, and query events.
  - Automatic reconnection handling with exponential backoff if the UI reconnects to a running service.
- [ ] **Truthful Protection Status**:
  - UI strictly displays "Protected" only when the IPC connection is active, loopback listeners are serving, and upstream DoH health probe succeeds.

---

## Milestone 07: Native Windows System Tray & Production Distribution
- [ ] **Zero-CGO Pure-Go System Tray**:
  - Pure Win32 implementation using `Shell_NotifyIconW`, `CreatePopupMenu`, and `TrackPopupMenuEx` via `golang.org/x/sys/windows`.
  - Zero CGO dependency (`CGO_ENABLED=0`), ensuring 100% Go toolchain build reproducibility without MinGW.
  - Dedicated message loop running in a locked OS thread (`runtime.LockOSThread()`).
  - **Explorer Restart Recovery**: Subscribes to the registered `"TaskbarCreated"` window message to automatically restore the tray icon whenever `explorer.exe` crashes or restarts.
- [ ] **Tray Context Menu Actions**:
  - *Open Dashboard*: Restores and brings the Wails window to front (`runtime.WindowShow`).
  - *Pause / Resume Protection*: Temporarily bypasses filtering without disconnecting DNS or breaking adapter settings.
  - *Recent Queries*: Submenu displaying recent blocked and resolved queries.
  - *Settings*: Directly navigates to the configuration screen.
  - *Exit UI*: Closes the desktop UI and frees WebView2 memory while leaving the background DNS service running.
  - *Stop Protection & Quit*: Safely restores Windows adapter DNS and shuts down the service.
- [ ] **Close-to-Tray Window Management**:
  - Intercepts window close events to minimize to system tray instead of terminating the app when enabled.
- [ ] **Production Distribution**:
  - NSIS installer packaging both service and desktop components.
  - Clean uninstallation with verified adapter DNS restoration.

---

## 📊 Performance & Governance Targets

Measurable performance benchmarks categorized by specific test conditions:

| Target Category | Measurable Target | Defined Test Condition |
| :--- | :--- | :--- |
| **Idle Service Memory** | < 25 MB Working Set | Headless Go background engine with zero active clients, 0 filter rules, empty cache, measured via Windows `ProcessMemoryCounters`. |
| **Loaded Filter Memory** | < 80 MB Working Set (100k rules)<br>< 150 MB (300k rules) | Headless Go engine with 100k and 300k parsed domains held in the reversed-label trie index. |
| **Idle Service CPU** | ~0.0% CPU | Background engine with zero incoming queries, verifying no polling loops or spinlocks. |
| **Populated Cache Memory** | < 35 MB Working Set | Engine holding 5,000 active cached DNS response messages in in-memory LRU. |
| **Cache Hit Latency** | < 0.2 ms per query | DNS queries matching populated in-memory cache entries under synthetic loopback query benchmark. |
| **Filter Match Throughput** | < 200 ns/op (10k rules)<br>< 500 ns/op (100k rules) | Direct in-memory lookup benchmark against reversed-label trie (`BenchmarkMatcher`). Verified: 127.3 ns/op at 10k rules. |
| **UI Active Memory** | ~70–100 MB Private Bytes | Wails v2 desktop shell with WebView2 rendering dashboard and query log. |
| **UI Minimized/Tray Memory** | Memory suspended / reclaimed | WebView2 execution paused when desktop window is hidden to the Windows system tray. |
| **Cold Startup to Ready** | < 50 ms (Service)<br>< 800 ms (Desktop UI) | Service: time from binary entrypoint to active socket listening on `127.0.0.1:5353`.<br>UI: time from launch to first meaningful paint. |
| **Query Log I/O Impact** | 0 ms added to DNS response | Ring-buffer handoff to background worker, verifying synchronous disk I/O is never on the query path. |
