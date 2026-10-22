# Milestone 04B.4A: SafeSearch DNS Policies Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement persistent, provider-specific SafeSearch DNS enforcement (`internal/safesearch`) using canonical CNAME targets, versioned SQLite persistence, atomic rewrite reconciliation, cache invalidation, and React settings UI with truthful runtime status.

**Architecture:**
- `internal/safesearch/provider.go`: Data-driven provider definitions (Google, Bing, DuckDuckGo, YouTube, Yandex) with verified canonical targets, documentation URLs, and implementation status.
- `internal/safesearch/registry.go`: Policy registry validating providers and generating deterministic CNAME rewrite rules for enabled, supported providers.
- `internal/storage/migrations.go`: Migration v2 introducing `safesearch_policies` table.
- `internal/storage/safesearch.go`: Persistence repository for SafeSearch enable/disable toggles and timestamps.
- `internal/storage/reconcile.go`: Injects SafeSearch CNAME rewrites into the active `Rewriter` during reconciliation, respecting user-defined rewrite precedence.
- `internal/dnsengine/handler.go`: Exposes thread-safe atomic rewriter hot-swapping and cache invalidation.
- `app.go`: Wails IPC methods (`GetSafeSearchSettings`, `ToggleSafeSearch`) with validation, persistence, atomic hot-reload, and cache flush.
- `frontend/src/features/settings/SettingsView.tsx`: SafeSearch card controls with provider status badges, documentation links, canonical targets, and DNS-only scope disclaimers.

**Tech Stack:** Go 1.25+, SQLite (`modernc.org/sqlite`), React 19, TypeScript, Tailwind CSS, Lucide icons, Wails v2.

---

## Global Constraints

1. Zero CGO requirement: pure Go SQLite portability on Windows.
2. Verified canonical targets only:
   - Google: `forcesafesearch.google.com` (mapping `www.google.com` and regional Google domains).
   - Bing: `strict.bing.com` (mapping `www.bing.com` and `edgeservices.bing.com`).
   - DuckDuckGo: `safe.duckduckgo.com` (mapping `duckduckgo.com` and `www.duckduckgo.com`).
   - YouTube: `restrict.youtube.com` (mapping YouTube domains).
   - Yandex: Marked unsupported for CNAME rewrite mechanism (requires dedicated resolver IP `77.88.8.7` per authoritative documentation).
3. Deterministic precedence: User custom rewrites > SafeSearch CNAME rewrites > Custom Allow/Block rules > Subscriptions > Cache & Upstream.
4. Normal DNS cache must never bypass an activated SafeSearch policy; cache is flushed on policy change.
5. No Windows network adapter, registry, firewall, or driver modifications.
6. DNS-only scope: SafeSearch redirects search VIPs via DNS; it does NOT inspect or decrypt HTTPS traffic.

---

### Task 1: SafeSearch Provider Registry (`internal/safesearch`)

**Files:**
- Create: `internal/safesearch/provider.go`
- Create: `internal/safesearch/registry.go`
- Create: `internal/safesearch/registry_test.go`

**Interfaces:**
- `Provider` struct: `ID`, `Name`, `Domains`, `Target`, `DocURL`, `Status`, `Warning`.
- `Registry` interface: `GetProviders() []Provider`, `GetProvider(id string) (Provider, bool)`, `GenerateRewriteRules(enabledIDs []string) []rewrites.Rule`.

- [ ] **Step 1: Write unit tests in `registry_test.go`**
  - Verify default registry contains Google, Bing, DuckDuckGo, YouTube, and Yandex.
  - Verify Google target is `forcesafesearch.google.com`.
  - Verify Bing target is `strict.bing.com` and covers `edgeservices.bing.com`.
  - Verify DuckDuckGo target is `safe.duckduckgo.com`.
  - Verify YouTube target is `restrict.youtube.com`.
  - Verify Yandex is marked `unsupported` and generates zero CNAME rewrite rules even if requested.
  - Verify CNAME rules generated have `Type: "CNAME"`, `TTL: 60`, `Enabled: true`.

- [ ] **Step 2: Implement `provider.go` and `registry.go`**
- [ ] **Step 3: Run `go test -v ./internal/safesearch` to verify passing tests**

---

### Task 2: SQLite Migration v2 & Storage Repository (`internal/storage`)

**Files:**
- Modify: `internal/storage/migrations.go`
- Create: `internal/storage/safesearch.go`
- Create: `internal/storage/safesearch_test.go`

**Interfaces:**
- `Storage.GetSafeSearchPolicies(ctx context.Context) (map[string]bool, error)`
- `Storage.SetSafeSearchPolicy(ctx context.Context, providerID string, enabled bool) error`
- `Storage.GetSafeSearchPolicyRecords(ctx context.Context) ([]SafeSearchPolicyRecord, error)`

- [ ] **Step 1: Write storage tests in `safesearch_test.go`**
  - Test migration v2 applies cleanly.
  - Test setting policy to enabled/disabled and verifying updated timestamp.
  - Test querying policies for empty table returns defaults or empty map without errors.
- [ ] **Step 2: Implement migration v2 and storage methods**
- [ ] **Step 3: Run `go test -v ./internal/storage` to verify passing tests**

---

### Task 3: Active State Reconciliation & Rewrite Precedence (`internal/storage`, `internal/dnsengine`)

**Files:**
- Modify: `internal/storage/reconcile.go`
- Modify: `internal/storage/reconcile_test.go`
- Modify: `internal/dnsengine/handler.go`
- Create: `internal/dnsengine/safesearch_integration_test.go`

**Operations:**
- In `internal/dnsengine/handler.go`:
  - Make `rewriter` atomic (`atomic.Pointer[rewrites.Rewriter]`).
  - Add `SetRewriter(rw *rewrites.Rewriter)`.
- In `internal/storage/reconcile.go`:
  - Load enabled SafeSearch policies from SQLite.
  - Generate CNAME rules from `safesearch.DefaultRegistry()`.
  - Load user-defined rewrites.
  - Merge rules: User-defined rewrite for a domain takes precedence over SafeSearch rewrite.
  - Compile combined rules into active `Rewriter`.
- In `safesearch_integration_test.go`:
  - Test that `www.google.com` query returns CNAME `forcesafesearch.google.com.` when enabled.
  - Test that user-defined rewrite for `www.google.com` (e.g. to `192.168.1.50`) overrides SafeSearch CNAME.
  - Test that SafeSearch CNAME rewrite takes precedence over blocklists.
  - Test that A and AAAA queries resolve properly with CNAME synthesis.
  - Test CNAME loop protection if target points back to domain.

- [ ] **Step 1: Write integration tests in `safesearch_integration_test.go`**
- [ ] **Step 2: Implement atomic rewriter in `handler.go` and reconciliation logic in `reconcile.go`**
- [ ] **Step 3: Run `go test -v ./internal/dnsengine ./internal/storage` to verify passing tests**

---

### Task 4: App Wails IPC Bindings & Cache Invalidation (`app.go`, `internal/models`)

**Files:**
- Modify: `internal/models/models.go`
- Modify: `app.go`
- Modify: `app_test.go`

**Operations:**
- In `internal/models/models.go`:
  - Define `SafeSearchProviderSetting` struct with JSON tags for frontend binding.
- In `app.go`:
  - Add `GetSafeSearchSettings() []models.SafeSearchProviderSetting`.
  - Add `ToggleSafeSearch(providerID string, enabled bool) error`.
  - In `ToggleSafeSearch`, reject enabling unsupported providers, update SQLite, reconcile engines atomically, and flush the DNS cache.
  - Wire DNS handler reference or cache flush helper into `App`.

- [ ] **Step 1: Write unit tests in `app_test.go` for `GetSafeSearchSettings` and `ToggleSafeSearch`**
- [ ] **Step 2: Implement models and methods in `app.go`**
- [ ] **Step 3: Run `go test -v .` to verify passing tests**

---

### Task 5: Frontend SafeSearch UI in Settings (`frontend/src`)

**Files:**
- Modify: `frontend/src/features/settings/SettingsView.tsx`
- Modify: `frontend/src/lib/ipc.ts`
- Modify: `frontend/src/App.tsx`

**Operations:**
- In `frontend/src/lib/ipc.ts`:
  - Add `getSafeSearchSettings` and `toggleSafeSearch` with fallback mock data.
- In `SettingsView.tsx`:
  - Add SafeSearch policy section with provider cards.
  - Display status badges (`ACTIVE`, `DISABLED`, `UNSUPPORTED`).
  - Display canonical targets, domain coverage, doc link button, and toggle.
  - Display non-invasive scope disclaimer regarding DNS-only enforcement.
- In `App.tsx`:
  - Wire `loadData` and toggle handlers.

- [ ] **Step 1: Update `ipc.ts` with typed methods and mock data**
- [ ] **Step 2: Implement SafeSearch UI in `SettingsView.tsx`**
- [ ] **Step 3: Run `npm --prefix frontend run build` to verify clean build**

---

### Task 6: Documentation & Complete Verification

**Files:**
- Modify: `docs/architecture.md`
- Modify: `docs/roadmap.md`
- Modify: `docs/threat-model.md`
- Modify: `README.md`

**Operations:**
- Update docs marking Milestone 04B.4A completed with verified provider targets, precedence rules, and limitations.
- Run complete verification:
  - `go test -count=1 ./...`
  - `go vet ./...`
  - `gofmt -s -l .`
  - `npm --prefix frontend run build`
  - `wails build`
