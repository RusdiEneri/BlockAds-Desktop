# BlockAds Desktop — Product Completion, Packaging & Public Release Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a production-grade, secure, polished desktop DNS filtering application with a built-in filter catalog, URL filter adder with metadata extraction and deduplication, startup splash screen, optional Windows startup, portable mode, NSIS installer, GitHub Actions workflows, security audit, and a public GitHub release under the authenticated developer account.

**Architecture:** Layered Go + Wails v2 + React/TypeScript desktop architecture. The Go core handles DNS filtering, SQLite persistence, and Windows system integration behind clean OS abstraction interfaces. The React frontend follows the established NeoBrutalist design system.

**Tech Stack:** Go 1.23+, Wails v2.17+, React 19, TypeScript, Tailwind CSS, modernc.org/sqlite (Zero CGO), GitHub Actions, NSIS.

**Spec:** Final Mission prompt and Milestone 01-04B.4A documentation.

## Global Constraints
- Pure Go / Zero CGO requirement (`CGO_ENABLED=0`) across all packages.
- No network adapter modification, Windows Service installation, or certificate interception in this milestone.
- Do not claim system-wide DNS protection is active until Milestone 05 is completed.
- Treat remote filter list metadata as untrusted plain text (sanitize, bound length, no HTML injection).
- Preserve user database and settings during upgrades and uninstall.
- Authenticate to GitHub only using existing credentials in Windows Credential Manager (`RusdiEneri`).

---

### Task 1: Filter Catalog, Metadata Extraction & Backend Deduplication
- **Files:**
  - Create: `internal/filtermanager/catalog.go`, `internal/filtermanager/catalog_test.go`
  - Modify: `internal/filtermanager/client.go`, `internal/filtermanager/updater.go`
  - Modify: `internal/storage/filters.go`, `internal/storage/filters_test.go`
  - Modify: `internal/models/models.go`
  - Modify: `app.go`, `app_test.go`
- **Interfaces:**
  - `NormalizeURL(rawURL string) (string, error)`
  - `ExtractFilterMetadata(reader io.Reader) (title, desc, homepage string, ruleCount int, format string)`
  - `PreviewFilterURL(ctx context.Context, url string) (*models.FilterPreviewResult, error)`
  - `GetCatalogFilters() []models.CatalogFilter`
  - `SubscribeCatalogFilter(id string) (*models.FilterList, error)`
  - `AddFilterSubscriptionWithMetadata(name, url, format, category, desc string) (*models.FilterList, error)`

- [ ] **Step 1: Write tests for URL normalization and duplicate detection**
- [ ] **Step 2: Implement URL normalization, metadata comment parser, and catalog registry**
- [ ] **Step 3: Add database unique constraint / duplicate detection in storage**
- [ ] **Step 4: Expose IPC methods in `app.go` with full unit tests**
- [ ] **Step 5: Verify tests pass**

---

### Task 2: Portable Mode & Isolated Data Directory Resolution
- **Files:**
  - Create: `internal/storage/paths.go`, `internal/storage/paths_test.go`
  - Modify: `app.go`
  - Modify: `internal/models/models.go`
- **Interfaces:**
  - `ResolveDataDirectory(explicitPath string) (dataDir string, isPortable bool, err error)`
  - `AppInfo.DataDir` and `AppInfo.IsPortable`

- [ ] **Step 1: Write unit tests for portable marker detection, writability check, and installed mode path resolution**
- [ ] **Step 2: Implement portable detection (`portable.marker` or `data/` next to exe) and `%LOCALAPPDATA%` fallback**
- [ ] **Step 3: Wire into `NewApp()` and update `AppInfo` DTO**
- [ ] **Step 4: Verify tests pass**

---

### Task 3: Optional Windows Startup ("Start with Windows")
- **Files:**
  - Create: `internal/startup/startup.go`
  - Create: `internal/startup/startup_windows.go`
  - Create: `internal/startup/startup_other.go`
  - Create: `internal/startup/startup_test.go`
  - Modify: `app.go`, `internal/models/models.go`, `internal/storage/settings.go`
- **Interfaces:**
  - `type StartupManager interface { IsEnabled() (bool, error); SetEnabled(enable bool) error; GetExecutablePath() (string, error) }`
  - IPC: `GetStartupStatus() (*models.StartupStatus, error)`
  - IPC: `SetStartWithWindows(enabled bool) error`

- [ ] **Step 1: Write tests for startup registry key management and path quoting**
- [ ] **Step 2: Implement Windows CurrentVersion\Run registry manipulation (non-admin, per-user HKCU)**
- [ ] **Step 3: Implement clean non-Windows stub for cross-compilation**
- [ ] **Step 4: Expose IPC methods and wire into Settings**
- [ ] **Step 5: Verify tests pass**

---

### Task 4: Frontend Filter Catalog, Add Flow & Duplicate Navigation
- **Files:**
  - Create: `frontend/src/features/filters/FilterCatalogModal.tsx`
  - Create: `frontend/src/features/filters/AddFilterModal.tsx`
  - Modify: `frontend/src/features/filters/FilterListsView.tsx`
  - Modify: `frontend/src/lib/ipc.ts`
  - Modify: `frontend/wailsjs/go/main/App.d.ts`, `App.js`
- **Interfaces:**
  - Filter catalog browsing, search by keyword, category filter, subscribe button.
  - Manual URL add with real-time metadata extraction preview, editable title, and duplicate collision warnings.

- [ ] **Step 1: Update Wails JS bindings and IPC bridge with browser-preview fallbacks**
- [ ] **Step 2: Implement Catalog Modal with verified list metadata and Add/Subscribed state**
- [ ] **Step 3: Implement Add Filter Modal with preview fetch, metadata extraction, and duplicate detection**
- [ ] **Step 4: Update FilterListsView with integrated catalog and manual add flows**
- [ ] **Step 5: Test frontend build with `npm --prefix frontend run build`**

---

### Task 5: Application Splash Screen & Startup Experience
- **Files:**
  - Create: `frontend/src/components/SplashScreen.tsx`
  - Modify: `frontend/src/App.tsx`
  - Modify: `frontend/src/features/settings/SettingsView.tsx`
- **Interfaces:**
  - NeoBrutalist splash overlay with real initialization steps, version display, reduced-motion support, and error retry card.
  - Settings UI updated with "Start with Windows" toggle and Portable Mode / Data Directory diagnostics.

- [ ] **Step 1: Implement `SplashScreen.tsx` matching NeoBrutalist tokens and real initialization stages**
- [ ] **Step 2: Integrate into `App.tsx` with smooth transition and error boundary**
- [ ] **Step 3: Update `SettingsView.tsx` with "Start with Windows" toggle and Data Directory card**
- [ ] **Step 4: Verify frontend build passes**

---

### Task 6: NSIS Installer Configuration & Packaging
- **Files:**
  - Modify: `wails.json`
  - Modify: `build/windows/installer/project.nsi`
  - Create: `build/windows/installer/README-PORTABLE.txt`
  - Create: `build/windows/installer/portable.marker`
- **Interfaces:**
  - `wails build` and `wails build -nsis`
  - Portable zip packaging structure

- [ ] **Step 1: Configure `wails.json` with product metadata, version, company name, copyright**
- [ ] **Step 2: Update `project.nsi` to unregister HKCU Run key on uninstall without deleting `%LOCALAPPDATA%` database**
- [ ] **Step 3: Create portable package template files**
- [ ] **Step 4: Test Wails build**

---

### Task 7: GitHub Actions CI & Windows Release Workflows
- **Files:**
  - Create: `.github/workflows/ci.yml`
  - Create: `.github/workflows/release-windows.yml`
  - Create: `.github/dependabot.yml`
  - Create: `.github/ISSUE_TEMPLATE/bug_report.yml`
  - Create: `.github/ISSUE_TEMPLATE/feature_request.yml`
  - Create: `.github/PULL_REQUEST_TEMPLATE.md`

- [ ] **Step 1: Implement CI workflow with Windows runner, Go tests, vet, fmt, frontend build, Wails build**
- [ ] **Step 2: Implement Release workflow with tag trigger `v*`, NSIS installer, portable zip, raw exe, SHA-256 manifest**
- [ ] **Step 3: Implement issue templates, PR template, and dependabot config**

---

### Task 8: Security Audit, Documentation & Repo Hygiene
- **Files:**
  - Create: `LICENSE` (MIT)
  - Create: `CHANGELOG.md`
  - Create: `CONTRIBUTING.md`
  - Create: `SECURITY.md`
  - Modify: `README.md`
  - Modify: `docs/architecture.md`, `docs/roadmap.md`, `docs/threat-model.md`
  - Modify: `.gitignore`

- [ ] **Step 1: Scan workspace for secrets, personal tokens, and hardcoded local absolute paths**
- [ ] **Step 2: Update documentation with truthful status, limitations, portable guide, and build instructions**
- [ ] **Step 3: Review `.gitignore` to ensure databases, binary artifacts, and cache are excluded**

---

### Task 9: Full Verification, Git Commit & GitHub Public Release
- **Actions:**
  - Run all tests: `go test -count=1 ./...`, `go vet ./...`, `gofmt -s -l .`, `npm --prefix frontend run build`, `wails build`
  - Review Git staging
  - Create initial commit
  - Create public GitHub repository `RusdiEneri/BlockAds-Desktop` and push
  - Verify remote URL and commit SHA
