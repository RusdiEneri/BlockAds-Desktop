# Milestone 04B.3: Filter-List Manager & Scheduler Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a reliable filter-list subscription manager and background scheduler (`internal/filtermanager`) with HTTP conditional requests (ETag / Last-Modified under RFC 9110), safe URL validation (HTTPS enforcement and SSRF protection), 304 Not Modified preservation, atomic candidate matcher rebuild and hot-reload, bounded download concurrency, and integration into `App` and frontend.

**Architecture:**
- `internal/filtermanager/client.go`: Safe HTTP downloader enforcing HTTPS, rejecting HTTPS-to-HTTP downgrade redirects, blocking loopback/private/link-local SSRF destinations by default, bounded to 25MB response limit.
- `internal/filtermanager/updater.go`: Orchestrator for conditional HTTP fetches, 304 snapshot verification/metadata update, 200 body parsing and HTML rejection, atomic SQLite snapshot transaction, and candidate matcher building.
- `internal/filtermanager/scheduler.go`: Lifecycle-managed background worker with configurable interval (default 24h), jitter, bounded concurrency (semaphore), and in-flight deduplication per subscription.
- `internal/dnsengine/handler.go`: Add atomic `SetMatcher(*filters.Matcher)` and `Matcher() *filters.Matcher` for zero-downtime hot-reloads.
- `app.go`: Wire filter manager and scheduler into application lifecycle and IPC endpoints.
- `frontend`: Expose refresh actions, add/remove subscriptions, and display verified update timestamps, status, and rule counts.

**Tech Stack:** Go 1.25+, standard library `net/http`, `net`, `crypto/sha256`, `internal/storage`, `internal/filters`.

---

## Global Constraints

1. Zero CGO requirement: preserve pure Go Windows portability.
2. Existing DNS engine, cache, DoH resolver, rewrites, circuit breaker, and SQLite storage must continue functioning without regressions.
3. Safe HTTP updates: HTTPS only, maximum 5 redirects, no downgrade to HTTP, SSRF protection against private networks.
4. RFC 9110 compliance: `If-None-Match` and `If-Modified-Since` conditional headers; on 304 verify existing snapshot and do not re-parse empty bodies.
5. Snapshot safety: a malformed, unreachable, or 4xx/5xx response must NEVER erase previously accepted snapshots or disable active filtering.
6. Atomic hot-reload: candidate matchers are fully built in memory before atomic swap into `Handler` with zero dropped queries.
7. Explicit scheduler lifecycle: scheduler does NOT start during unit tests or unexpected database init.
8. No Windows network adapter alterations.

---

### Task 1: Handler Atomic Matcher Support & Storage 304 Helpers (`internal/dnsengine/handler.go`, `internal/storage/filters.go`)

**Files:**
- Modify: `internal/dnsengine/handler.go`
- Modify: `internal/dnsengine/handler_test.go`
- Modify: `internal/storage/filters.go`
- Modify: `internal/storage/filters_test.go`

**Operations:**
- In `internal/dnsengine/handler.go`:
  - Add `matcher atomic.Pointer[filters.Matcher]`.
  - Add `Matcher() *filters.Matcher` and `SetMatcher(m *filters.Matcher)`.
- In `internal/storage/filters.go`:
  - Add `RecordFilterFetchNotModified(ctx context.Context, subID string, meta SnapshotMeta) error`.
- Tests:
  - Verify `SetMatcher` updates active filtering under concurrent DNS queries without races or panics.
  - Verify `RecordFilterFetchNotModified` updates attempt/success timestamps and ETag/LastModified without modifying snapshot.

- [ ] **Step 1: Write tests for atomic `SetMatcher` and `RecordFilterFetchNotModified`**
- [ ] **Step 2: Implement changes in `handler.go` and `filters.go`**
- [ ] **Step 3: Run `go test -v ./internal/dnsengine ./internal/storage` to verify**

---

### Task 2: Safe Downloader & SSRF Protection (`internal/filtermanager/client.go`)

**Files:**
- Create: `internal/filtermanager/client.go`
- Create: `internal/filtermanager/client_test.go`

**Operations:**
- `DownloaderConfig struct { Timeout time.Duration, MaxBodyBytes int64, AllowPrivateNetworks bool }`
- URL validation: HTTPS required, reject schemes like `http`, `ftp`, `file`.
- SSRF IP validation: block loopback (`127.0.0.0/8`, `::1`), private networks (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`), link-local (`169.254.0.0/16`), multicast, unless `AllowPrivateNetworks` is explicitly enabled.
- Redirect policy: max 5 redirects, reject any redirect that downgrades to HTTP or targets private IPs.
- Bounded reader: cap response body to `MaxSnapshotBytes` (25MB).

- [ ] **Step 1: Write unit tests in `client_test.go` for scheme validation, redirect downgrades, and SSRF blocking**
- [ ] **Step 2: Implement `internal/filtermanager/client.go`**
- [ ] **Step 3: Run `go test -v ./internal/filtermanager` to verify**

---

### Task 3: Filter Updater with Conditional HTTP (RFC 9110) & Atomic Rebuild (`internal/filtermanager/updater.go`)

**Files:**
- Create: `internal/filtermanager/updater.go`
- Create: `internal/filtermanager/updater_test.go`

**Operations:**
- `FilterUpdater struct` wrapping `*storage.Storage`, `Downloader`, and `MatcherReloader`.
- `UpdateSubscription(ctx context.Context, subID string) (*UpdateResult, error)`:
  - Fetch subscription metadata from SQLite.
  - Set `If-None-Match: <etag>` and `If-Modified-Since: <last-modified>`.
  - Handle HTTP 304:
    - Check if snapshot exists and SHA-256 is valid.
    - If valid, update metadata via `RecordFilterFetchNotModified` and return 304 status without re-parsing.
    - If snapshot missing or corrupt, retry once unconditionally without conditional headers.
  - Handle HTTP 200:
    - Verify body size <= 25MB.
    - Compute SHA-256.
    - Check if content looks like HTML error page (`<html`, `<!DOCTYPE`) -> reject with error.
    - Parse rules using `filters.ParseFilterList(body, format)`.
    - Reject if accepted rule count == 0.
    - If SHA-256 matches existing snapshot, update metadata only.
    - If new content, persist snapshot + metadata in single transaction via `storage.SaveFilterSnapshot`.
    - Reconcile active state via `storage.ReconcileActiveState` and reload candidate matcher into engine.
  - On error:
    - Call `storage.RecordFilterFetchError(ctx, subID, err.Error())`.
    - Active matcher and previous valid snapshot remain intact.

- [ ] **Step 1: Write integration tests in `updater_test.go` using `httptest.Server` (200, 304, 304-retry, 500 error, HTML rejection, duplicate checksum)**
- [ ] **Step 2: Implement `internal/filtermanager/updater.go`**
- [ ] **Step 3: Run `go test -v ./internal/filtermanager` to verify updater**

---

### Task 4: Filter Scheduler & Concurrency Control (`internal/filtermanager/scheduler.go`)

**Files:**
- Create: `internal/filtermanager/scheduler.go`
- Create: `internal/filtermanager/scheduler_test.go`

**Operations:**
- `SchedulerConfig struct { Storage *storage.Storage, Updater *FilterUpdater, Interval time.Duration, MaxConcurrency int }`
- `Scheduler struct` with start/stop lifecycle context, ticker, semaphore for bounded concurrency, in-flight mutex tracking active subscription IDs.
- `Start(ctx context.Context)` and `Stop()`.
- `RefreshSubscription(ctx context.Context, id string) (*UpdateResult, error)` (manual trigger).
- `RefreshAll(ctx context.Context) ([]UpdateResult, error)` (batch manual trigger).
- Jitter on scheduled timer to avoid burst load.

- [ ] **Step 1: Write tests in `scheduler_test.go` for deduplication, bounded concurrency, start/stop lifecycle**
- [ ] **Step 2: Implement `internal/filtermanager/scheduler.go`**
- [ ] **Step 3: Run `go test -v ./internal/filtermanager` to verify scheduler**

---

### Task 5: App Integration & IPC Endpoints (`app.go`, `app_test.go`)

**Files:**
- Modify: `app.go`
- Modify: `app_test.go`

**Operations:**
- Integrate `filtermanager.Scheduler` and `filtermanager.FilterUpdater` into `App`.
- Expose methods:
  - `RefreshFilterList(id string) error`
  - `RefreshAllFilterLists() error`
  - `AddFilterSubscription(name string, url string, format string) (*models.FilterList, error)`
  - `DeleteFilterSubscription(id string) error`
- Ensure `App.Close()` stops scheduler and cleans up.

- [ ] **Step 1: Update `app.go` and add unit tests in `app_test.go`**
- [ ] **Step 2: Run `go test -count=1 ./...` to verify all packages**

---

### Task 6: Frontend Filter Lists View Enhancement (`frontend/src/features/filters/FilterListsView.tsx`)

**Files:**
- Modify: `frontend/src/features/filters/FilterListsView.tsx`
- Modify: `frontend/src/lib/ipc.ts`
- Modify: `frontend/src/App.tsx`

**Operations:**
- Display real status indicators (`ok`, `failed`, `pending`, `304`).
- Add manual "Refresh" button per card and "Refresh All" button.
- Show last attempt, last success, and error tooltip/banner.
- Add "Add Filter List" dialog/input for custom HTTPS feeds.
- Update scope banner: filter lists are backed by persistent SQLite and conditional HTTP caching.

- [ ] **Step 1: Update frontend components and IPC bindings**
- [ ] **Step 2: Run `npm --prefix frontend run build`**

---

### Task 7: Documentation & Acceptance Verification

**Files:**
- Update: `docs/architecture.md`
- Update: `docs/roadmap.md`
- Update: `docs/threat-model.md`
- Update: `README.md`

- [ ] **Step 1: Update documentation reflecting filter manager, scheduler, conditional HTTP, and SSRF controls**
- [ ] **Step 2: Run full verification suite (`go test`, `go vet`, `gofmt`, `npm run build`, `wails build`)**
- [ ] **Step 3: Review Git diff and produce final report**
