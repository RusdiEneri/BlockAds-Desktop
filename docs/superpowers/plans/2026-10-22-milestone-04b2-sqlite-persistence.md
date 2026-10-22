# Milestone 04B.2: SQLite Persistence Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a pure Go SQLite persistence layer (`internal/storage`) using `modernc.org/sqlite` without CGO. Persist application settings, DNS rewrites, custom rules, filter-list subscriptions, filter snapshots with SHA-256 integrity verification, and an asynchronous non-blocking query logger with batch transactions. Reconcile database state with in-memory DNS engine on startup without modifying network settings.

**Architecture:**
- `internal/storage`: SQLite database wrapper with WAL journal mode, busy timeout (5000ms), foreign keys, connection pooling (`SetMaxOpenConns(1)` to eliminate lock contention), schema migration system, and repository methods.
- `internal/storage/async_logger.go`: Bounded channel queue for DNS query events, worker loop batching inserts into transactions every 100 items or 200ms, non-blocking enqueue with dropped-event telemetry.
- Snapshot strategy: SQLite `filter_snapshots` table with atomic transaction linkage to subscription metadata, SHA-256 verification, 25MB bound, and zero orphaned files.
- Application integration: `app.go` connects to local SQLite database in app data directory, loads persistent settings, custom rules, rewrites, and enables query logging.

**Tech Stack:** Go 1.25+ / 1.27, `modernc.org/sqlite v1.36.0` (BSD-3-Clause, pure Go, zero CGO), standard library `database/sql`, `crypto/sha256`.

---

## Global Constraints

1. Zero CGO requirement: use `modernc.org/sqlite` to ensure pure Go reproducible builds on Windows.
2. Safe SQLite configuration: WAL mode, foreign keys enabled, busy timeout 5000ms, `MaxOpenConns(1)`.
3. Non-blocking query logging: zero disk I/O on DNS response path; bounded queue with dropped counter.
4. Filter snapshot integrity: verify SHA-256 checksum on restoration; keep existing working matcher if new snapshot fails.
5. Exact rule precedence: DNS rewrites evaluated before blocklists and before cache; blocklist evaluated before upstream DoH.
6. No raw SQL exposed to frontend: all IPC calls use validated Go models/DTOs.
7. No system-wide network adapter changes.

---

### Task 1: Storage Package Foundation & Migrations (`internal/storage/db.go`, `migrations.go`)

**Files:**
- Create: `internal/storage/db.go`
- Create: `internal/storage/migrations.go`
- Create: `internal/storage/db_test.go`

**Interfaces & Types:**
- `Storage struct { db *sql.DB, path string, mu sync.RWMutex }`
- `Open(path string) (*Storage, error)`
- `Close() error`
- `Migrate(ctx context.Context) error`
- Pragmas: `WAL`, `foreign_keys=ON`, `busy_timeout=5000`, `synchronous=NORMAL`.
- `schema_migrations` tracking table.

- [ ] **Step 1: Write tests in `internal/storage/db_test.go` for DB open, pragma verification, and migrations**
- [ ] **Step 2: Implement `db.go` and `migrations.go`**
- [ ] **Step 3: Run `go test -v ./internal/storage` to verify database setup and migrations**

---

### Task 2: Repositories for Settings, Custom Rules & DNS Rewrites (`internal/storage/settings.go`, `rewrites.go`, `rules.go`)

**Files:**
- Create: `internal/storage/settings.go`
- Create: `internal/storage/rewrites.go`
- Create: `internal/storage/rules.go`
- Create: `internal/storage/settings_rules_test.go`

**Operations:**
- `GetSettings(ctx) (models.AppSettings, error)`
- `UpdateSettings(ctx, models.AppSettings) error`
- `GetRewrites(ctx) ([]rewrites.Rule, error)`
- `SaveRewrite(ctx, rewrites.Rule) error`
- `DeleteRewrite(ctx, domain, recordType string) error`
- `GetCustomRules(ctx) ([]models.CustomRule, error)`
- `SaveCustomRule(ctx, models.CustomRule) error`
- `DeleteCustomRule(ctx, id string) error`
- `ToggleCustomRule(ctx, id string, enabled bool) error`

- [ ] **Step 1: Write tests in `internal/storage/settings_rules_test.go`**
- [ ] **Step 2: Implement settings, rewrites, and custom rules repository methods**
- [ ] **Step 3: Run `go test -v ./internal/storage` to verify persistence**

---

### Task 3: Filter Subscriptions & Snapshot Storage (`internal/storage/filters.go`)

**Files:**
- Create: `internal/storage/filters.go`
- Create: `internal/storage/filters_test.go`

**Operations:**
- Snapshot strategy: `filter_snapshots` table (`subscription_id`, `checksum`, `content`, `rule_count`, `size_bytes`, `created_at`).
- `GetFilterSubscriptions(ctx) ([]models.FilterList, error)`
- `SaveFilterSubscription(ctx, models.FilterList) error`
- `SaveFilterSnapshot(ctx, subID string, content string, ruleCount int) (checksum string, err error)`
- `GetFilterSnapshot(ctx, subID string) (content string, checksum string, err error)`
- SHA-256 verification and size bounds (max 25MB).

- [ ] **Step 1: Write tests for subscription metadata and snapshot checksum verification in `internal/storage/filters_test.go`**
- [ ] **Step 2: Implement `filters.go` with atomic snapshot transaction and checksum validation**
- [ ] **Step 3: Run `go test -v ./internal/storage` to verify snapshot integrity**

---

### Task 4: Non-Blocking Asynchronous Query Logger (`internal/storage/async_logger.go`, `query_log.go`)

**Files:**
- Create: `internal/storage/query_log.go`
- Create: `internal/storage/async_logger.go`
- Create: `internal/storage/async_logger_test.go`

**Operations:**
- `QueryEvent struct { ID, Timestamp, Domain, ClientIP, QueryType, Status, LatencyMs, FilterRule, Reason, Protocol, Rcode }`
- `AsyncLogger struct` with bounded channel, batch flushing worker, dropped count, queue stats.
- `Enqueue(event QueryEvent) bool` (non-blocking).
- `Close() error` (drains and flushes).
- `GetQueryEvents(ctx, filter QueryFilter) ([]models.QueryLogEntry, int, error)` (paginated).
- `PruneQueryEvents(ctx, olderThan time.Duration) (int64, error)`.

- [ ] **Step 1: Write tests for non-blocking enqueue, batching, saturation, and pagination**
- [ ] **Step 2: Implement `query_log.go` and `async_logger.go`**
- [ ] **Step 3: Benchmark enqueue latency and batch insert throughput**
- [ ] **Step 4: Run `go test -v ./internal/storage` to verify logger**

---

### Task 5: Integration with App State & Startup Reconciliation (`app.go`, `app_test.go`)

**Files:**
- Modify: `app.go`
- Modify: `app_test.go`
- Modify: `internal/dnsengine/handler.go` (if necessary to attach async logger observer)

**Operations:**
- Initialize `storage.Storage` in `App` using app data directory or in-memory for testing.
- Load settings, rewrites, custom rules, and filter lists from SQLite at startup.
- Wire `QueryObserver` in DNS handler to `AsyncLogger.Enqueue`.
- Connect UI methods (`AddCustomRule`, `UpdateUpstreamDoH`, `ToggleFilterList`) to SQLite persistence.

- [ ] **Step 1: Update `app.go` with storage lifecycle and persistent methods**
- [ ] **Step 2: Update `app_test.go` with storage-backed integration tests**
- [ ] **Step 3: Run `go test -count=1 ./...` across all packages**

---

### Task 6: Documentation & Acceptance Verification

**Files:**
- Update: `docs/architecture.md`
- Update: `docs/roadmap.md`
- Update: `docs/threat-model.md`
- Update: `README.md`

- [ ] **Step 1: Update documentation reflecting SQLite storage architecture, schema, and non-blocking logging**
- [ ] **Step 2: Run complete verification suite (`go test`, `go vet`, `gofmt`, `npm run build`, `wails build`)**
- [ ] **Step 3: Review Git diff and produce final report**
