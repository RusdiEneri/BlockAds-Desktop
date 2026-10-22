# BlockAds Desktop — Threat Model & Security Posture

## 1. System Threat Landscape

DNS filtering is an effective, non-invasive method for blocking ads, tracking, and known malicious domains. Operating a local DNS resolver and Windows Service presents specific architectural security risks that must be analyzed and defended against.

---

## 2. Security Boundaries & Risk Analysis

### A. Bootstrap DNS vs. Plaintext User-Query Fallback
- **Privacy & Security Implications of Bootstrap DNS**:
  - To connect to a DoH endpoint specified by hostname (such as `https://dns.caliph.dev/dns-query`), the client must obtain an initial IP address for `dns.caliph.dev`.
  - **A plaintext bootstrap query is still plaintext DNS**: Querying an external resolver over standard UDP/TCP port 53 sends an unencrypted DNS query across the network, revealing the identity of the user's chosen DoH provider to local network observers and ISPs.
- **Scope Limitation & Architectural Invariant**:
  - *Strictly Scoped*: Bootstrap DNS is used **exclusively** to resolve the DoH server's own hostname during initialization or endpoint IP recovery. It is **never** used to resolve arbitrary user queries (e.g., `google.com`, `internal.company.com`).
- **Mitigations & Controls (Milestone 03)**:
  1. **Configurable Bootstrap Resolvers**: Users can configure which bootstrap resolver IPs to trust (`models.AppSettings.BootstrapDNS`), defaulting to trusted public resolvers (e.g., Cloudflare `1.1.1.1`, Google `8.8.8.8`, Quad9 `9.9.9.9`).
  2. **Direct IP Literal Endpoints**: If the user configures a DoH endpoint with an IP address directly (or pins bootstrap IPs), bootstrap DNS queries over port 53 are completely bypassed.
  3. **Strict TLS SNI & Hostname Verification**: Even when connecting to a socket using a bootstrap IP address, `tls.Config.ServerName` is strictly set to the configured DoH hostname, and standard x509 system CA certificate validation is enforced.
  4. **Resistance to Bootstrap Poisoning**: If a local attacker spoofs the plaintext bootstrap DNS response, the connection will route to the attacker's IP. However, the subsequent TLS handshake requires a valid certificate for the DoH hostname issued by a trusted system root CA. Unless the attacker possesses a compromised root CA installed on the machine, the TLS handshake fails immediately and safely aborts.
  5. **Endpoint IP Rotation & Recovery**: The client caches resolved DoH server IPs to prevent continuous plaintext bootstrap queries. If connections to the cached IP fail (e.g., upstream CDN IP rotation), the client initiates a controlled bootstrap refresh to recover connectivity.

### B. Insecure Plaintext Fallback (Forbidden Silent Degradation)
- **Risk**: When an upstream DoH request times out or TLS fails, an engine might be tempted to fall back to standard UDP port 53 to maintain connectivity.
- **Engine Invariant**:
  - **Zero Silent Fallback**: The engine will *never* silently downgrade user queries to plaintext DNS or fall back to OS/ISP resolvers.
  - When the upstream resolver is unconfigured or returns an error, the engine immediately responds with `SERVFAIL` (`dns.RcodeServerFailure`) and logs the failure in query observability.

### C. Local DNS Amplification & Exposure Risks
- **Risk**: Exposing a DNS resolver on `0.0.0.0` or a LAN interface allows external actors on the local subnet to abuse the server as an open recursive resolver or DNS amplification vector.
- **Engine Invariant (Milestone 02 Implemented)**:
  - **Loopback Only by Default**: The DNS engine binds strictly to `127.0.0.1` (or loopback `::1`). Attempts to bind non-loopback addresses (e.g. `0.0.0.0`) are actively rejected during server initialization unless explicitly enabled by developer override.

### D. Subdomain Boundary & Parsing Vulnerabilities
- **Risk**: Naive string suffix matching (e.g. `strings.HasSuffix("example.com.evil.org", "example.com")`) results in catastrophic false matches or circumvention by attackers who append trusted domains to attacker-controlled origins.
- **Engine Invariant (Milestone 02 Implemented)**:
  - Domain matching is handled by a reversed-label trie (`internal/filters/trie.go`) with strict DNS label boundaries. Matching occurs label-by-label (`["com", "example"]`). A rule for `example.com` matches `sub.example.com` but will *never* match `example.com.evil.org`.
  - Filter list parsers strictly validate domain syntax, stripping trailing root dots, rejecting URLs, wildcards, and cosmetic adblock filters, reporting line-numbered diagnostics.

### E. Sinkhole Response Integrity & DNSSEC Posture
- **Risk**: Synthesizing artificial CNAME or MX records can break applications or misdirect traffic. Falsely setting the Authenticated Data (AD) flag tricks clients into believing spoofed records were validated by DNSSEC.
- **Engine Invariant (Milestone 02 Implemented)**:
  - Sinkhole answers are strictly limited to `0.0.0.0` for `A` queries and `::` for `AAAA` queries with a documented TTL (default 60s).
  - All other blocked query types (`HTTPS`, `SVCB`, `TXT`, `MX`, `CNAME`) return `NOERROR` with an empty Answer section (NODATA). No artificial records are ever fabricated.
  - The `AuthenticatedData` (AD) bit is explicitly set to `false` on all engine-generated responses.

### F. Atomic Filter Index Reloads
- **Risk**: Reloading or modifying rules while concurrent DNS queries are processed can cause race conditions, partial filtering, or panic crashes.
- **Engine Invariant (Milestone 02 Implemented)**:
  - Matcher indexes are completely constructed and validated in memory before being atomically installed via `atomic.Pointer[trieIndex]`. Readers access the trie lock-free without exposing partially loaded or corrupt states.

### G. Inter-Process Communication (IPC) & Named Pipe Access Control
- **Risk**: A Windows Named Pipe without strict security descriptors can allow unprivileged malware or other users on a multi-user machine to inject configuration changes, manipulate custom rules, or view sensitive DNS logs.
- **Mitigation (Planned for Milestone 06)**:
  - The Named Pipe (`\\.\pipe\BlockAdsDesktopPipe`) is configured with an explicit Discretionary Access Control List (DACL).
  - Access is restricted exclusively to the interactive user's SID and the local Administrators group.
  - Impersonation levels are restricted to `SecurityIdentification`, preventing the service from being tricked into elevating client privileges.

### H. DNS Rewrite Hijacking, Loops & Validation
- **Risk**: Malicious or malformed DNS rewrites could redirect sensitive domains to attacker IPs, create circular CNAME exhaustion loops, or leak private local hostnames upstream.
- **Engine Invariants & Mitigations (Milestone 04B.1 Implemented)**:
  - Strict type validation: `A` records must parse as IPv4 (`To4() != nil`); `AAAA` records must parse as valid IPv6; `CNAME` targets cannot be IP addresses or self-referential loops.
  - Circular CNAME loop defense: `Rewriter` maintains visited chain state and caps chaining depth to 8 hops, returning SERVFAIL immediately on circular references.
  - RFC 1034 private name leakage defense: If a local domain has an `A` rewrite but receives a `AAAA` query, `Rewriter` returns NOERROR/NODATA instead of passing the query to public upstream DoH resolvers.
  - Administrative boundary: DNS rewrites are configured locally and cannot be manipulated by remote adblock list subscriptions.

### I. SafeSearch Provider Mapping Integrity & DNS-Only Boundaries (Milestone 04B.4A Implemented)
- **Risks**:
  - Hardcoding static IP literals for search engine SafeSearch VIPs leads to IPv6 dual-stack failures, regional latency degradation, and service outages when providers migrate IPs.
  - Assuming non-standard or unverified CNAME targets (such as `family.yandex.ru`) can cause silent resolution failures.
  - Stale cached normal answers could bypass newly enabled SafeSearch policies.
  - Expecting DNS filtering to inspect HTTPS encrypted traffic leads to false security assumptions.
- **Engine Invariants & Security Controls**:
  1. *Canonical CNAME Aliases*: SafeSearch exclusively uses verified canonical targets: Google (`forcesafesearch.google.com`), Bing (`strict.bing.com` covering `edgeservices.bing.com` for Edge sidebar), DuckDuckGo (`safe.duckduckgo.com`), and YouTube (`restrict.youtube.com`). Zero static IP addresses are hardcoded.
  2. *Verified Status & Honest Limitations*: Providers that do not support standard CNAME rewriting (such as Yandex, which mandates dedicated filtering resolver IPs `77.88.8.7` / `77.88.8.3`) are marked `unsupported` in the registry and prevented from being enabled with misleading CNAMEs.
  3. *Deterministic Precedence*: User-configured rewrites take top precedence over SafeSearch rules, and SafeSearch rules take precedence over filter list block rules.
  4. *Automatic Cache Invalidation*: Activating or deactivating a SafeSearch policy immediately flushes the DNS response cache, preventing previously resolved normal search engine IPs from bypassing the policy.
  5. *DNS-Only Scope Limits*: DNS-layer SafeSearch redirects DNS hostname resolution; it does NOT decrypt, inspect, or intercept HTTPS web traffic, nor does it replace browser-level, operating system-level, or account-level parental controls.

### J. Privilege Escalation & UI/Service Separation
- **Risk**: Changing network adapter DNS settings requires Windows Administrator privileges. Running the entire desktop GUI as Administrator introduces a substantial attack surface.
- **Mitigation (Planned for Milestone 05 & 06)**:
  - The desktop UI runs as a standard unprivileged user in the interactive desktop session.
  - Privileged network configuration and DNS serving are isolated in the background Windows Service.
  - The UI interacts with the service solely through the DACL-restricted IPC interface.

### K. Query Log Privacy & Local SQLite Persistence (Milestone 04B.2 Implemented)
- **Risk**: DNS query logs record every domain accessed by users on the machine, creating a local privacy risk. Unchecked logging can cause disk exhaustion, memory leaks, or SQL injection.
- **Engine Invariants & Security Controls**:
  1. *Parameterized SQL*: 100% of SQLite database queries use parameterized placeholders (`?`). No concatenated raw SQL strings exist in the codebase.
  2. *Zero UI SQL Exposure*: Frontend components communicate through strongly-typed DTO models. No raw SQL execution or schema modification is exposed over Wails IPC.
  3. *Zero DNS Payload Logging*: Full DNS wire payloads and EDNS options are never stored; only high-level resolution metadata (domain, query type, status, upstream latency, rule reason, client IP, protocol) is persisted.
  4. *Filesystem Access Control*: The database file (`blockads.db`) is stored in `%APPDATA%\BlockAds-Desktop\`, inheriting Windows NTFS user ACLs. Other unprivileged user accounts on the machine cannot read the database.
  5. *Corrupt Snapshot & Tamper Resistance*: Remote filter snapshots stored in SQLite are verified against their SHA-256 content checksums on restoration. If tampered with or truncated on disk, `ErrCorruptSnapshot` is raised and the active in-memory matcher remains untouched.
  6. *Snapshot Size Bounds*: Snapshots are capped at 25MB (`MaxSnapshotBytes`), preventing memory exhaustion during list updates.
  7. *Asynchronous Non-Blocking Logging*: Logging uses a bounded in-memory channel. When saturated, excess events are safely dropped with an atomic diagnostic counter (`droppedCount`), preventing memory exhaustion or blocking the DNS response path.
  8. *Privacy Retention Pruning*: Historical logs are prunable by retention duration (`PruneQueryEvents`) or count limit (`PruneMaxQueryEvents`). Zero cloud analytics or telemetry are transmitted.

### L. DNS Cache Poisoning, Memory Exhaustion & Stale Bypass
- **Risks**:
  1. *Stale Rule Bypass*: A domain previously allowed and cached might be subsequently blocked by a dynamic rule update or filter list reload; serving from cache could bypass active block rules.
  2. *Cache Key Collisions & EDNS Mismatch*: Conflating distinct record types (e.g. A vs AAAA), classes, DNSSEC requests (DO / CD bits), or replaying cached OPT records to non-EDNS clients could cause protocol rejections or DNSSEC validation failures.
  3. *Cross-Subnet Cache Poisoning*: Caching EDNS Client Subnet (ECS) responses in an unpartitioned cache leaks regional routing across unrelated client subnets.
  4. *Denial of Service / Query Storms*: Upstream failures or broken authoritative zones could cause repeated client retries flooding upstream resolvers; alternatively, caching transient failures as normal answers could persistently lock clients out.
  5. *Shared Message Mutation*: Modifying a returned DNS message in memory could corrupt subsequent cache hits.
- **Engine Invariants & Mitigations (Milestone 04A & 04A.1 Implemented)**:
  1. *Precedence Checking*: Filter rules are strictly evaluated on every query prior to returning cached records. A domain matched by an active block rule is immediately sinkholed, rendering cached answers unable to bypass blocklists.
  2. *Explicit Cache Key & Flag Isolation*: Keys are uniquely computed across canonical lowercase FQDN, query type, query class, EDNS0 DO bit, and CD bit (`Key{Name, Qtype, Qclass, DO, CD}`).
  3. *EDNS Client Subnet (ECS) Bypass*: Queries containing ECS options bypass unpartitioned caching to prevent cross-subnet poisoning.
  4. *Response Integrity & Question Casing*: Cached responses preserve the incoming query's transaction ID and exact 0x20 mixed-case Question section. Stale OPT records are stripped and client-tailored EDNS metadata is generated dynamically.
  5. *Bounded Capacity & Deterministic LRU*: Maximum cache entries are strictly enforced via a doubly-linked LRU eviction list. Oldest entries are deterministically purged when reaching capacity.
  6. *Strict Negative Caching (RFC 2308)*: Negative responses (NXDOMAIN and NODATA) require valid SOA metadata; responses lacking SOA or having TTL=0 are never cached. NXDOMAIN responses without CNAME answers apply to the name itself for all record types.
  7. *Resolution Failure Caching (RFC 9520)*: Upstream failures (SERVFAIL, timeouts, transport errors) are cached in a separate bounded LRU cache with short configurable TTL (1s-300s, default 5s), mitigating query storms while enabling automatic recovery.
  8. *Deep Copy Isolation*: Stored messages and retrieved messages are deep-copied (`msg.Copy()`), ensuring concurrent handlers cannot mutate shared cached state.

### M. Filter List Fetching, SSRF & Malicious Content Ingestion (Milestone 04B.3 Implemented)
- **Risks**:
  1. *Server-Side Request Forgery (SSRF)*: Malicious subscription URLs could target internal loopback services (e.g. `http://127.0.0.1:8080`), local routers (`192.168.1.1`), link-local devices, or cloud metadata endpoints (`http://169.254.169.254/`).
  2. *Plaintext Downgrade & Insecure Transit*: Intermediaries on the network could inject or tamper with filter lists fetched over unencrypted HTTP.
  3. *Resource Exhaustion (Decompression Bombs & Oversized Feeds)*: Massive HTTP responses could trigger Out-Of-Memory (OOM) crashes.
  4. *Corrupt Feeds & Empty List Attacks*: A compromised or failing feed returning an empty file, 404/500 HTML page, or cloudflare challenge must not silently disable DNS filtering.
  5. *DNS Rebinding*: An attacker could return a public IP during initial URL parsing and resolve to a loopback/private IP at socket connect time.
- **Engine Invariants & Security Controls**:
  1. *HTTPS Enforced by Default*: All subscription URLs must use the `https://` scheme. Clear-text `http://` or other schemes (`file://`, `ftp://`) are rejected (`ErrInsecureScheme`).
  2. *Strict SSRF Defense & Socket-Level Dial Control*: URLs are validated prior to dialing, and `net.Dialer.Control` verifies every resolved IP address at the socket connection level. Requests to loopback (`127.0.0.0/8`, `::1`), RFC1918 private subnets (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`), link-local (`169.254.0.0/16`, `fe80::/10`), and multicast addresses are actively blocked (`ErrSSRFBlocked`).
  3. *Downgrade-Resistant Bounded Redirects*: Redirects are limited to 5 hops. Any redirect from an HTTPS origin to an HTTP destination is immediately aborted (`ErrRedirectDowngrade`), and every redirect target is independently re-validated against the SSRF policy.
  4. *Strict 25MB Response Bounds*: HTTP response bodies are read through an `io.LimitReader` capped at 25MB (`MaxSnapshotBytes`), preventing memory exhaustion.
  5. *HTML & Error-Page Ingestion Defense*: Downloaded content is inspected for HTML doctype, tags, and error pages (`ErrHTMLResponse`). HTML error responses are rejected and never parsed as filter lists.
  6. *Empty Rule-Set Protection*: Feeds producing zero valid rules are rejected (`ErrEmptyFilterList`), preventing broken or wiped feeds from silently disabling DNS protection.
  7. *Working State Preservation*: Network errors, timeouts, or validation failures preserve the previously accepted snapshot in SQLite and leave the active in-memory matcher untouched. A failing update can never erase working protection.
  8. *RFC 9110 Conditional Update Integrity*: HTTP 304 Not Modified responses require verification of the stored snapshot's SHA-256 checksum. If the snapshot is missing or corrupt, an unconditional recovery GET is automatically executed to re-acquire the list.

### N. URL Normalization, Metadata Sanitization & Duplicate Prevention (Milestone 04B.5 Implemented)
- **Risks**:
  1. *Duplicate Subscription Sprawl*: Minor variations in URL representation (e.g. `HTTPS://example.com/list.txt` vs `https://example.com:443/list.txt#frag`) could lead to multiple subscriptions downloading and caching identical data.
  2. *Remote Comment Injection / XSS*: Filter list comment headers (e.g. `! Title: <script>...`) could attempt to inject malicious HTML, script tags, or terminal control sequences into the UI or application state.
  3. *Credential Leakage*: Usernames or passwords embedded in URLs (e.g. `https://user:pass@host/`) could be recorded in persistent logs.
- **Engine Invariants & Security Controls**:
  1. *Canonical URL Normalization*: All URLs are strictly normalized via `models.NormalizeURL` (lowercased scheme and host, default HTTPS port 443 stripped, URI fragments removed, empty paths normalized to `/`).
  2. *Embedded Credential Rejection*: URLs containing userinfo/credentials (`user:pass@`) are rejected with `ErrInvalidURL`.
  3. *Plaintext Sanitization of Remote Metadata*: Extracted titles, descriptions, and homepages from comment headers (`! Title:`, `# Description:`) are sanitized, length-bounded (max 128-512 chars), stripped of control characters, and treated strictly as untrusted text strings. They are never evaluated as HTML, markdown scripts, or system commands.
  4. *Storage-Layer Duplicate Enforcement*: SQLite schema queries verify URL uniqueness before insertion; concurrent additions of identical normalized URLs are safely blocked with `ErrConflict`.

### O. Portable Execution Isolation & Windows Startup Registry Security (Milestone 04B.5 Implemented)
- **Risks**:
  1. *Cross-Environment Data Leakage*: Portable installations run from USB drives or shared systems must not write databases, logs, or cache files into the host machine's `%APPDATA%` or registry.
  2. *Privilege Escalation via Windows Startup*: Running desktop auto-start as Administrator or modifying machine-wide keys (`HKLM`) creates privilege escalation vectors for other users on shared machines.
  3. *Stale Startup Path Execution*: If a portable folder is moved or deleted, an obsolete startup entry could fail or attempt to launch an unintended binary.
- **Engine Invariants & Security Controls**:
  1. *Deterministic Path Resolution & Write Probing*: If `portable.marker` or a local `data/` folder exists, application data is strictly localized to the portable bundle directory. Before opening the database, the directory is probed for write permissions; if unwritable, startup halts with a clear error rather than silently falling back to host system directories.
  2. *HKCU Current-User Startup*: "Start with Windows" exclusively interacts with `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`. No machine-level (`HKLM`) registry modifications are performed, and no Administrator rights are requested.
  3. *Quoted Executable Paths & Stale Path Detection*: All registry values are properly quoted (e.g., `"C:\Path With Spaces\BlockAds-Desktop.exe"`). On startup, the registered path is checked against the current runtime executable path; if the portable directory was moved, the obsolete entry is detected and the UI offers synchronization.
  4. *Clean Uninstaller*: The NSIS uninstaller automatically purges the user's `HKCU` startup entry while preserving the user's local database and filter snapshots in `%LOCALAPPDATA%\BlockAds Desktop`.

