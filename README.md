# BlockAds Desktop 🛡️

**BlockAds Desktop** is an open-source, privacy-first Windows DNS filtering application engineered with a high-performance **Go** backend and a modern neo-brutalist **React + TypeScript + Wails v2** desktop interface.

Inspired by mobile ad blockers like [blockads-android](https://github.com/pass-with-high-score/blockads-android) and the functional clarity of AdGuard Home, BlockAds Desktop provides local DNS-level ad, tracker, and malware blocking on Windows without complex driver installations or invasive HTTPS root certificates.

---

## 🎯 Architecture & Technologies

- **Desktop Runtime**: [Wails v2](https://wails.io) (Go + Microsoft Edge WebView2)
- **Engine Core**: Go 1.24+ / 1.25 (`github.com/miekg/dns`, reversed-label trie matcher, dual UDP/TCP server)
- **Frontend**: React 19, TypeScript, Vite, Tailwind CSS with Neo-brutalist design tokens
- **Persistence**: Pure Go SQLite (`modernc.org/sqlite`, zero CGO, WAL mode)
- **Design System**: Modern neo-brutalism documented in [`DESIGN.md`](./DESIGN.md)
- **Target OS**: Windows 10 / Windows 11 (64-bit amd64)

---

## 🚀 Current Implementation Status

### 1. Curated Filter Catalog & Manual URL Subscriptions
- **Built-in Catalog**: Curated selection of verified filter lists (AdAway, StevenBlack Unified, OISD Basic, URLhaus, Dan Pollock, Peter Lowe) categorized by protection type with one-click subscription.
- **Manual Filter Addition**: Add any remote filter-list URL with live metadata extraction (`! Title:`, `! Description:`, `! Homepage:`), format validation, and customizable display name.
- **Canonical URL Normalization & Duplicate Prevention**: Lowercase scheme/host, port 443 stripping, fragment removal, and root slash normalization enforced at both backend and SQLite storage layers.
- **Atomic Matcher Hot-Reloading**: Candidate filter rules are validated and snapshots persisted before atomically swapping the active matcher via lock-free atomic pointers with zero dropped queries.
- **Hardened Downloader**: Bounded redirects (max 5), SSRF protection blocking private/loopback/link-local destinations, 25MB response limits, and rejection of HTML error pages.

### 2. Provider-Specific SafeSearch Policies
- **AdGuard-Compatible DNS Enforcement**: Enforces SafeSearch for Google, Bing, DuckDuckGo, YouTube, and Pixiv via canonical CNAME rewrites.
- **Cache Invalidation & Precedence**: Policy changes immediately invalidate cached answers, ensuring strict rule precedence: Explicit Rewrites > SafeSearch Policies > Custom Rules > Subscriptions > Cache & Upstream.

### 3. High-Performance Go DNS Core & Encrypted Upstream
- **Loopback Proxy**: RFC-compliant UDP and TCP DNS server strictly bound to `127.0.0.1:5335`.
- **Encrypted DNS-over-HTTPS (DoH)**: RFC 8484 compliant DoH client targeting upstream resolvers (defaulting to `https://dns.caliph.dev/dns-query`) with bootstrap fallback and system CA verification.
- **Zero Plaintext Fallback**: Bounded timeouts and TLS errors return `SERVFAIL` rather than falling back to unencrypted plaintext DNS.
- **Bounded LRU DNS Cache**: Preserves client transaction IDs and question casing; supports negative caching (RFC 2308) and circuit breaker exponential backoff (RFC 9520) shielding upstream DoH outages.
- **DNS Rewrites Engine**: Local wildcard and exact domain mapping with CNAME recursion loop detection.

### 4. Dual Execution Modes: Portable & Installed
- **Portable Mode**: Automatic detection when `portable.marker` or a `data/` directory exists adjacent to `BlockAds-Desktop.exe`. SQLite database, filter snapshots, and configuration are stored exclusively within the local `data/` folder with write-permission validation and zero registry pollution.
- **Installed Mode**: Default per-user data isolation under `%LOCALAPPDATA%\BlockAds Desktop`.
- **Optional "Start with Windows"**: Non-privileged, user-controlled startup toggle via `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`. Quoted executable paths with automatic stale portable path detection.

### 5. Packaging & Release Automation
- **Windows NSIS Installer**: Professional setup installer with Start Menu and Desktop shortcuts, WebView2 bootstrapper support, and clean uninstaller that purges registry startup keys while preserving user databases.
- **GitHub Actions Workflows**:
  - `ci.yml`: Continuous integration on Windows runners validating Go formatting, `go vet`, unit tests, frontend build, and Wails binary compilation.
  - `release-windows.yml`: Automated tag-triggered release workflow publishing standalone executables, NSIS setup installers, portable ZIP packages, and SHA-256 checksum manifests.

> [!IMPORTANT]
> **Architectural Boundary Disclaimer**:
> BlockAds Desktop currently operates on loopback (`127.0.0.1:5335`). System-wide Windows network adapter redirection and background Windows Service integration are scheduled for Milestones 05-07. The application never falsely claims system-wide protection is active until those integrations are in place.

---

## 🛠️ Prerequisites

- **Operating System**: Windows 10 or Windows 11 (64-bit amd64)
- **Go**: 1.24 or later
- **Node.js**: 20+ LTS and `npm`
- **Microsoft Edge WebView2**: Pre-installed on Windows 11 and current Windows 10
- **Wails CLI v2**:
  ```powershell
  go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0
  ```

---

## 💻 Development & Building

### Run All Unit & Integration Tests
```powershell
go test -v -count=1 ./...
```

### Static Analysis & Formatting Check
```powershell
go vet ./...
gofmt -s -l .
```

### Build Frontend
```powershell
cd frontend
npm ci
npm run build
cd ..
```

### Run in Development Mode (Hot-Reload)
```powershell
wails dev
```

### Build Production Desktop Binary
```powershell
wails build
```
The compiled executable will be located at `build/bin/BlockAds-Desktop.exe`.

### Build NSIS Setup Installer (Requires NSIS on PATH)
```powershell
wails build -nsis
```

---

## 📦 Distribution Packages

Pre-built binaries and packages are published on [GitHub Releases](https://github.com/RusdiEneri/BlockAds-Desktop/releases):

1. **Standalone Executable** (`BlockAds-Desktop-vX.Y.Z-windows-amd64.exe`): Single self-contained desktop binary.
2. **Portable Edition** (`BlockAds-Desktop-vX.Y.Z-windows-amd64-portable.zip`): Includes `portable.marker` and `README-PORTABLE.txt`. Stores all data in a local `data/` directory.
3. **Windows Setup Installer** (`BlockAds-Desktop-vX.Y.Z-windows-amd64-setup.exe`): NSIS installer for per-user installation with shortcuts and uninstaller.
4. **Checksum Manifest** (`SHA256SUMS.txt`): Cryptographic verification for all release assets.

---

## 📚 Documentation

- [`DESIGN.md`](./DESIGN.md) — Neo-brutalist visual design system, tokens, and components.
- [`docs/architecture.md`](./docs/architecture.md) — Architecture specification, DNS pipeline, and component separation.
- [`docs/roadmap.md`](./docs/roadmap.md) — Milestone roadmap from foundation to service split and system tray.
- [`docs/threat-model.md`](./docs/threat-model.md) — Threat model, security boundaries, and SSRF/DoH defenses.
- [`CONTRIBUTING.md`](./CONTRIBUTING.md) — Contribution guidelines, coding standards, and PR workflows.
- [`SECURITY.md`](./SECURITY.md) — Vulnerability reporting policy and security disclosures.
- [`CHANGELOG.md`](./CHANGELOG.md) — Version history and release notes.

---

## ⚖️ License

BlockAds Desktop is licensed under the [MIT License](./LICENSE).
Copyright (c) 2026 Nuruddin Rusydi Ilham and BlockAds Desktop Contributors.
