# BlockAds Desktop — Architecture Specification

## 1. System Overview

BlockAds Desktop is a Windows-first network privacy application inspired by the functional clarity of AdGuard Home. It intercepts local OS DNS queries and resolves permitted queries over encrypted DNS-over-HTTPS (DoH), filtering advertisement, telemetry, tracking, and malware domains at the DNS protocol layer.

The architecture is divided into two decoupled components:
1. **BlockAds Service**: A lightweight, headless Windows Service (or background daemon) managing the DNS engine, upstream DoH, filter matcher, cache, and privileged network adapter settings.
2. **BlockAds Desktop**: An unprivileged Wails v2 desktop UI with a native Windows system tray icon, communicating with the service via secure, local Named Pipes.

```
                      ┌────────────────────────────────────────┐
                      │        Windows Applications            │
                      │  (Chrome, Firefox, Discord, Games)     │
                      └──────────────────┬─────────────────────┘
                                         │ Standard DNS (Port 5353 / 53 UDP/TCP)
                                         ▼
                      ┌────────────────────────────────────────────────────────┐
                      │         BLOCKADS SERVICE (Headless Background)         │
                      │  - Runs at boot (Session 0 or standalone background)    │
                      │  - Zero WebView2 / Zero GUI dependencies (< 25 MB RAM) │
                      │                                                        │
                      │  ┌──────────────┐    ┌──────────────┐    ┌──────────┐  │
                      │  │ In-Mem Cache │    │ Custom Rules │    │   DNS    │  │
                      │  │ (LRU/TTL, M4)│    │ (Allow/Block)│    │ Rewrites │  │
                      │  └──────┬───────┘    └──────┬───────┘    └────┬─────┘  │
                      │         │                   │                 │        │
                      │  ┌──────▼───────────────────▼─────────────────▼─────┐  │
                      │  │          Reversed-Label Trie Matcher             │  │
                      │  │       (Label Boundary Checking, Precedence)      │  │
                      │  └──────┬───────────────────┬───────────────────────┘  │
                      └─────────┼───────────────────┼──────────────────────────┘
           BLOCKED Domains      │                   │ PERMITTED Domains
         ┌──────────────────────┘                   └──────────────────────┐
         ▼                                                                 ▼
┌──────────────────┐                                             ┌──────────────────┐
│ Synthesized      │                                             │ Encrypted DoH    │
│ Sinkhole Answer  │                                             │ Upstream Client  │
│ A:    0.0.0.0    │                                             │ (Bootstrap DNS,  │
│ AAAA: ::         │                                             │  TLS 1.2+, M03)  │
│ Other: NODATA    │                                             │ (No User-Query   │
│ AD flag: false   │                                             │  Plaintext Fall) │
└──────────────────┘                                             └────────┬─────────┘
                                                                          │ HTTPS (Port 443)
                                                                          ▼
                                                         ┌──────────────────────────────────┐
                                                         │ https://dns.caliph.dev/dns-query │
                                                         └──────────────────────────────────┘

       ▲                                                                ▲
       │                                                                │
       │ Windows Named Pipe: \\.\pipe\BlockAdsDesktopPipe (DACL-Secured)│
       │                                                                │
       ▼                                                                ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                         BLOCKADS DESKTOP (Client UI)                         │
│  - Unprivileged interactive user session                                     │
│  - Wails v2 + React 19 + TypeScript + Vite                                   │
│  - Zero-CGO Win32 System Tray (Shell_NotifyIconW, TaskbarCreated recovery)   │
│  - Close-to-tray capability without stopping DNS service                     │
│  - Displays verified runtime protection state (reconnects seamlessly)        │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Component Boundaries & Responsibilities

### A. BlockAds Service (`blockads-service.exe` / `--service`)
- **Lifecycle**: Runs independently of user login as a Windows Service (`golang.org/x/sys/windows/svc`) or headless process.
- **Resource Footprint**: Minimal memory (< 25 MB RAM idle), zero WebView2 runtime, near 0% CPU.
- **Responsibilities**:
  1. **DNS Server (`internal/dnsengine/server.go` — Implemented in Milestone 02)**: Binds loopback listeners (`127.0.0.1:5353` / `53`) on both UDP and TCP.
  2. **Protocol Handler (`internal/dnsengine/handler.go` — Implemented in Milestone 02)**: Enforces single question, Opcode, ClassINET, and sinkhole generation without fabricated records.
  3. **Domain Matcher (`internal/filters/matcher.go` — Implemented in Milestone 02)**: Evaluates queries against reversed-label trie with strict label boundaries and atomic hot-reload.
  4. **DNS Cache & Failure Cache (`internal/cache/` — Implemented in Milestone 04A & 04A.1)**:
     - **DNS Response Cache (`DNSCache`)**:
       - Bounded in-memory LRU cache with deterministic eviction at `MaxEntries`.
       - Explicit cache key (`Key{Name, Qtype, Qclass, DO, CD}`) isolating queries by canonical lowercase name, record type, class, EDNS0 DNSSEC OK flag, and Checking Disabled bit.
       - Question casing preservation: Cache hits preserve the current request's original 0x20 mixed-casing and transaction ID.
       - EDNS/OPT isolation (RFC 6891): Never replays cached OPT records; attaches client-specific OPT metadata based on current request parameters. Bypasses unpartitioned caching for EDNS Client Subnet (ECS) queries.
       - RFC 1035/2181 zero-TTL compliance: Positive responses with TTL=0 are strictly uncacheable and cannot be extended by `MinTTL`.
       - RFC 2308 negative caching: Strictly requires valid SOA metadata in Authority section; name-level NXDOMAIN caching allows satisfying subsequent queries for any QTYPE at that name.
       - Deep-copy isolation (`msg.Copy()`) preventing shared in-place mutation of cached messages.
       - Rule precedence invariant: Filter rules are checked before returning cached responses, preventing stale allowed answers from bypassing newly activated block rules.
     - **Resolution Failure Cache (`FailureCache`, RFC 9520)**:
       - Separate bounded LRU cache for upstream SERVFAIL, timeout, and transport errors.
       - Bounded short TTL (1s - 300s, default 5s) preventing query storms while allowing automatic recovery.
       - Reports structured `cached resolution failure (RFC 9520)` diagnostics with 0ms upstream latency.
     - Thread-safe statistics reporting (`Hits`, `Misses`, `Expirations`, `Evictions`, `Size`, `Capacity`).
  5. **DNS Rewrites Subsystem (`internal/rewrites/` — Implemented in Milestone 04B.1)**:
     - Custom local DNS mapping engine supporting exact and wildcard domain patterns (e.g. `nas.home.arpa A 192.168.18.11`, `*.lab.home.arpa A 10.10.10.10`).
     - CNAME alias resolution with chained target lookups and loop detection (visited tracking + 8-hop depth cap returning SERVFAIL on circular chains).
     - RFC 1034 NODATA synthesis when a domain exists in local rewrites but the requested query type is unconfigured, preventing local private names from leaking to public upstream resolvers.
     - Evaluated in `ServeDNS` prior to blocklists and prior to the DNS response cache, ensuring immediate local resolution without stale cache interference.
  6. **SafeSearch Policies Subsystem (internal/safesearch/ — Implemented in Milestone 04B.4A)**:
     - Extensible provider registry with verified canonical CNAME targets (Google -> orcesafesearch.google.com, Bing -> strict.bing.com, DuckDuckGo -> safe.duckduckgo.com, YouTube -> 
estrict.youtube.com).
     - Yandex marked unsupported for CNAME rewrite mechanism (requires dedicated resolver IP 77.88.8.7).
     - Persistent policy storage via SQLite Migration v2 (safesearch_policies table).
     - Reconciled into active in-memory Rewriter with strict precedence: explicit user rewrites > SafeSearch rules > filter lists > cache.
     - Integrated cache flushing: toggling provider policies automatically invalidates affected DNS cache entries.
  7. **Safe Browsing Threat Feeds (internal/safebrowsing/ — Planned in Milestone 04B.4B)**: Blocks malware, phishing, and scam domains.
  7. **Upstream DoH (`internal/upstream/` — Implemented in Milestone 03)**:
     - Resolves permitted user queries exclusively over DNS-over-HTTPS (RFC 8484).
     - **Bootstrap DNS vs. Plaintext Fallback**:
       - *Bootstrap DNS*: Strictly used to resolve the DoH endpoint hostname (e.g. `dns.caliph.dev`) when initializing or recovering connections. A plaintext bootstrap query is acknowledged as plaintext DNS; therefore, bootstrap resolvers are user-configurable (defaulting to standard public resolvers like `1.1.1.1` and `8.8.8.8`) or can be bypassed if the user configures an IP literal DoH endpoint.
       - *TLS SNI & Verification*: Regardless of which bootstrap IP is used to open the TCP connection, `tls.Config.ServerName` is strictly set to the configured DoH hostname, and standard x509 system CA certificate validation is enforced.
       - *Endpoint IP Rotation & Recovery*: Resolved DoH server IPs are cached. If connections fail, the client refreshes the bootstrap resolution to handle upstream IP rotation.
       - *Zero User-Query Plaintext Fallback*: General user browsing queries are **never** forwarded to unencrypted DNS. If DoH resolution fails or times out, the handler returns `SERVFAIL`.
     - **TLS Negotiation**: Uses standard secure TLS negotiation (`MinVersion: tls.VersionTLS12`, negotiating TLS 1.3 whenever supported by client and server) with system root CA certificates and strict hostname verification, avoiding artificial TLS 1.3-only restrictions that break compatibility.
  7. **Privileged System Configuration (`internal/platform/` — Planned in Milestone 05)**: Configures Windows network adapters, stores configuration snapshots, and provides recovery fail-safes.
  8. **IPC Server (Planned in Milestone 06)**: Serves telemetry, statistics, and configuration commands to the desktop UI.

### B. BlockAds Desktop UI (`BlockAds-Desktop.exe`)
- **Lifecycle**: Starts in the interactive user desktop session. Can be launched on login or on-demand.
- **Technology**: Wails v2 (Go backend bridge + WebView2 + React 19/TypeScript).
- **Decoupling**: Can be opened, closed, or restarted at any time without stopping or degrading background DNS filtering.
- **Protection State Invariant**: Never displays "Protected" as a cosmetic assumption; verifies active IPC connection, loopback socket binding, and upstream DoH health before reporting protection status.

### C. Secure Inter-Process Communication (IPC — Planned in Milestone 06)
- **Transport**: Windows Named Pipe (`\\.\pipe\BlockAdsDesktopPipe`).
- **Security & Authorization**:
  - Protected with a Windows Security Descriptor / Discretionary Access Control List (DACL).
  - Permits read/write access strictly to the local Administrator group and the active interactive user session token (`SID`). Unrelated local processes cannot tamper with DNS settings.
- **Reconnection Semantics**:
  - The desktop client continuously monitors pipe health.
  - If the service restarts or the client launches before the service is ready, the client enters a reconnecting state with exponential backoff, updating the UI accordingly.

### D. Native Windows System Tray (`internal/tray/` — Planned in Milestone 07)
- **Zero-CGO Pure-Go Implementation**:
  - Standard Go systray packages (`getlantern/systray`, `energye/systray`) require CGO and a MinGW C compiler, and can cause runtime thread deadlocks with Wails event loops.
  - BlockAds Desktop will use native Win32 API calls (`Shell_NotifyIconW`, `CreatePopupMenu`, `TrackPopupMenuEx`) via `golang.org/x/sys/windows`.
  - Runs in a dedicated locked OS thread (`runtime.LockOSThread()`) to drive a hidden window message pump (`GetMessage`, `TranslateMessage`, `DispatchMessage`).
- **Explorer Restart Recovery**:
  - Registers the native Windows message `RegisterWindowMessage("TaskbarCreated")`.
  - When Windows Explorer crashes or restarts, the tray window receives this message and automatically recreates the notification icon, avoiding "ghost" or disappeared tray icons.
- **Tray Menu Actions**:
  - **Open Dashboard**: Restores and focuses the main application window (`runtime.WindowShow`).
  - **Pause / Resume Protection**: Temporarily pauses filtering (switching engine to pass-through) without altering network adapter DNS.
  - **Recent Queries**: Quick dropdown displaying the last 5 blocked/resolved domains.
  - **Settings**: Directly navigates to the configuration view.
  - **Exit UI**: Terminates the Wails UI process and releases WebView2 memory, while leaving the background DNS service running undisturbed.
  - **Stop Protection & Quit**: Restores Windows adapter DNS settings and shuts down the background service.

---

## 3. AdGuard Home-Inspired Feature Subsystems (Planned for Milestone 04)

### A. Filter List Subscriptions & Atomic Synchronization (`internal/filtermanager/` — Implemented in Milestone 04B.3)
- **Safe Downloader (`client.go`)**:
  - Enforces HTTPS by default; validates syntax and target hosts before any network I/O.
  - Bounded redirect policy (maximum 5 hops) with strict HTTPS-to-HTTP downgrade rejection (`ErrRedirectDowngrade`).
  - SSRF protection against loopback (127.0.0.0/8, ::1), private RFC1918 (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16), link-local unicast/multicast, and cloud metadata (169.254.169.254) addresses. Destination IPs are verified pre-dial and at socket connect time via `net.Dialer.Control`.
  - Response size limit strictly enforced via `io.LimitReader` (25MB maximum bound).
- **RFC 9110 HTTP Conditional Updates (`updater.go`)**:
  - Stores and sends `If-None-Match` (ETag) and `If-Modified-Since` (Last-Modified) validators.
  - **HTTP 304 Handling**: Verifies that a valid snapshot exists in SQLite and satisfies SHA-256 integrity. If valid, retains content, checksum, and active rules, updating validator metadata without re-parsing or rebuilding the matcher. If no valid snapshot exists or integrity check fails, automatically retries once via unconditional GET to obtain a fresh representation.
  - **HTTP 200 Handling**: Verifies body is not an HTML error page (`ErrHTMLResponse`), parses rules using `filters.ParseFilterList`, validates rule counts, checks whether checksum is unchanged (updating metadata without rebuilding matcher), atomically persists snapshot to SQLite, reconciles active state, and hot-reloads the active matcher lock-free via `atomic.Pointer[filters.Matcher]`.
  - Download or validation failures preserve the previously accepted snapshot and active matcher without erasing working security rules.
- **Background Scheduler & Lifecycle (`scheduler.go`)**:
  - Explicit, context-managed lifecycle (`Start`, `Stop`, `IsRunning`) without hidden background goroutines.
  - Configurable interval (default 24h) with pseudo-random jitter (±10%) to prevent simultaneous requests across subscriptions.
  - Bounded download concurrency controlled by a semaphore channel (max 2 concurrent downloads).
  - In-flight deduplication per subscription ID (`ErrAlreadyUpdating`) preventing concurrent request stampedes.
  - Bounded exponential retry for transient network errors (retries once after 200ms backoff); permanent validation errors (`ErrHTMLResponse`, `ErrEmptyFilterList`, `ErrSSRFBlocked`) are never retried.
- **Provenance & Merging**:
  - Enabled subscriptions and custom rules merge deterministically through `storage.ReconcileActiveState`. Disabling one list removes only rules contributed by that list; domains shared across multiple lists remain blocked as long as at least one list remains active. Custom allow rules always take precedence over blocklist directives.

### B. DNS Rewrites
- Configurable domain mappings taking precedence over upstream resolution:
  - **A / AAAA Rewrites**: Direct mapping of hostnames to fixed IP addresses (e.g. `nas.home.local` $\to$ `192.168.1.50`).
  - **CNAME Rewrites**: Alias redirection (e.g. `service.local` $\to$ `target.domain.com`).
- Enforces strict syntax validation before insertion into the rewrite table.

### C. SafeSearch & Safe Browsing
- **SafeSearch Domain Mapping Design (Milestone 04B.4A)**:
  - Avoids hardcoded static VIP IP addresses, which break dual-stack IPv6, regional routing, or when providers retire IPs.
  - Uses a provider-specific canonical domain mapping table (`internal/safesearch`):
    - **Google Search**: maps `www.google.com` and regional Google domains to canonical target `forcesafesearch.google.com.`
    - **Microsoft Bing**: maps `www.bing.com` and `edgeservices.bing.com` (Edge sidebar search) to `strict.bing.com.`
    - **DuckDuckGo**: maps `duckduckgo.com` and `www.duckduckgo.com` to `safe.duckduckgo.com.`
    - **YouTube**: maps `www.youtube.com`, `m.youtube.com`, and API hostnames to `restrict.youtube.com.`
    - **Yandex**: Truthfully marked unsupported for CNAME rewriting. Official Yandex Family Search requires routing through dedicated Yandex Family DNS resolvers (`77.88.8.7` / `77.88.8.3`); CNAME rewriting to `family.yandex.ru` is not officially supported and is rejected.
  - The engine returns canonical CNAMEs dynamically, ensuring dual-stack IPv4/IPv6 compatibility and regional routing.
  - User-configured explicit rewrites take precedence over SafeSearch defaults.
  - Response cache invalidation on policy changes guarantees no cache bypass.
  - Includes dedicated unit and integration tests per provider mapping.
- **Safe Browsing Threat Feeds (Milestone 04B.4B Planned)**: Curated threat feeds blocking malware, phishing, and command-and-control (C2) domains with distinct diagnostic classification.

### D. In-Memory DNS Cache & Metrics
- Thread-safe LRU cache storing resolved DNS messages.
- **TTL Governance**: Clamps minimum TTL (preventing overly aggressive upstream querying) and caps maximum TTL (ensuring timely DNS updates).
- **Metrics**: Real-time tracking of cache hits, cache misses, hit ratio, and upstream round-trip latency.

### E. Persistent Storage & Query History (`internal/storage/` — Implemented in Milestone 04B.2)
- **Zero-CGO SQLite Engine**:
  - Implemented using `modernc.org/sqlite v1.36.0` (BSD-3-Clause license, pure Go, zero CGO requirement).
  - Guarantees 100% reproducible builds and frictionless Windows native binary compilation without MinGW or external toolchains.
  - Concurrency model: Configured with `PRAGMA journal_mode = WAL`, `PRAGMA synchronous = NORMAL`, `PRAGMA foreign_keys = ON`, `PRAGMA busy_timeout = 5000`. Connection pooling enforces `db.SetMaxOpenConns(1)`, completely eliminating SQLite database lock contention across concurrent goroutines on Windows.
- **Automated Schema Migrations**:
  - Self-contained migration engine using `schema_migrations` tracking table for idempotent forward migrations.
  - Version 1 schema persists:
    - Application settings (`app_settings`): upstream DoH URL, bootstrap DNS, cache sizing, auto-start, adapter integration state.
    - DNS rewrites (`dns_rewrites`): domain, record type (A, AAAA, CNAME), target, TTL, enabled state.
    - Custom rules (`custom_rules`): domain, action (ALLOW/BLOCK), enabled state, notes.
    - Filter subscriptions (`filter_subscriptions`): stable ID, name, source URL, format, ETag, Last-Modified, attempt/success timestamps, status, checksum, rule count.
    - Filter snapshots (`filter_snapshots`): atomic BLOB/text content storage linked directly to subscription metadata with SHA-256 integrity verification, 25MB safety bounds, and deduplication.
    - Query events (`query_events`): structured resolution log with compound indexes on `timestamp DESC`, `domain`, and `status`.
- **Non-Blocking Asynchronous Query Logger (`internal/storage/async_logger.go`)**:
  - Bounded memory queue (`chan QueryEvent`, default capacity 4096).
  - Handoff from DNS handler via non-blocking `select`: measured benchmark overhead is **13.68 ns/op with 0 allocations and 0 ms added disk latency**.
  - Background worker batches events into SQLite transactions every 100 entries or 200ms, achieving ~62,500 events/sec batch write throughput.
  - Graceful shutdown drains and flushes pending queue events before database closure.
  - Real-time diagnostic telemetry: `QueueDepth`, `DroppedCount`, `TotalEnqueued`, `TotalWritten`, and `WriteFailures`.
- **Startup Reconciliation (`internal/storage/reconcile.go`)**:
  - Automatically reconstructs active `rewrites.Rewriter` and `filters.Matcher` in deterministic order at startup.
  - Verifies snapshot SHA-256 checksums before loading; guarantees active working in-memory matchers are never replaced with corrupt or partial indexes.

---

## 4. Measurable Performance Targets & Test Conditions

To ensure empirical verification rather than anecdotal claims, performance targets are separated into distinct test conditions:

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
