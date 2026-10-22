# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] - 2026-10-23

### Added
- **Curated Built-in Filter Catalog**: Pre-tested and verified filter list catalog (AdAway Default, StevenBlack Unified, OISD Basic, URLhaus Malicious URLs, Dan Pollock Hosts, Peter Lowe Ad Server List) with category filtering and instant subscription controls.
- **Manual Filter Addition with Metadata Extraction**: Support adding arbitrary filter-list URLs with real-time preview, metadata comment parsing (`! Title:`, `! Description:`, `! Homepage:`), format detection, and custom display name editing.
- **Strict Duplicate URL Prevention**: Canonical URL normalization (lowercase scheme/host, port 443 stripping, fragment removal, root slash normalization) preventing duplicate entries across catalog, manual entries, and SQLite storage.
- **Application Startup Splash Screen**: Neo-brutalist branded splash screen reporting truthful initialization stages (Database, DNS Engine, Resolver, Scheduler) with graceful error handling and retry actions.
- **Optional "Start with Windows" Setting**: Non-privileged, user-controlled startup persistence via `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, supporting quoted paths and detecting stale portable paths.
- **Dual Execution Modes (Portable & Installed)**:
  - **Portable Mode**: Automatic detection via `portable.marker` or `data/` directory adjacent to executable; self-contained SQLite storage, write-probe verification, and zero registry footprint.
  - **Installed Mode**: Predictable per-user data isolation in `%LOCALAPPDATA%\BlockAds Desktop`.
- **Packaging & Release Infrastructure**:
  - Windows NSIS installer configuration with registry cleanup on uninstall and preservation of user database.
  - Standalone portable package with dedicated `README-PORTABLE.txt`.
  - GitHub Actions CI workflow (`.github/workflows/ci.yml`) running Go tests, vet, formatting checks, frontend build, and Wails build on Windows runners.
  - Tag-triggered Release publishing workflow (`.github/workflows/release-windows.yml`) building raw executables, NSIS setup installers, portable ZIP bundles, and SHA-256 manifests.

### Changed
- Standardized filter subscription persistence and atomic matcher compiling.
- Enhanced settings view with runtime data isolation diagnostics and startup toggle.

---

## [0.4.0] - 2026-10-22

### Added
- **Provider-Specific SafeSearch Policies**: DNS rewrite enforcement for Google, Bing, DuckDuckGo, YouTube, and Pixiv without breaking other domains.
- **Filter-List Manager & Scheduler**: Background cron scheduler for automated filter updates, ETag and Last-Modified conditional HTTP requests, atomic matcher swapping, and SSRF/redirect defenses.
- **SQLite Persistence Foundation**: Zero-CGO database (`modernc.org/sqlite`) with WAL mode, schema migrations, and asynchronous batch query logging.
- **DNS Rewrites Engine**: Custom allowlist/blocklist rules and domain rewrites supporting wildcard matching and exact IP/NXDOMAIN responses.
- **DNS Cache & Reliability**: Bounded LRU response cache with TTL decrementing, negative caching (RFC 2308), upstream circuit breaker with half-open backoff, and EDNS option isolation (ECS and DNS Cookies).
- **DNS-over-HTTPS (DoH) Resolver**: RFC 8484 compliant encrypted DNS client using default Caliph DoH (`https://dns.caliph.dev/dns-query`) with bootstrap fallback, TLS SNI, and system CA verification.
- **Go DNS Engine & Matcher**: High-performance loopback DNS proxy (127.0.0.1:5335 UDP/TCP) utilizing reversed-label trie for sub-millisecond rule matching across hundreds of thousands of domains.
- **Wails v2 Desktop Foundation**: Modern Neo-Brutalist dashboard built with React 19, TypeScript, Lucide icons, and Tailwind CSS.
