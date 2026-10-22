# Milestone 01: Desktop Foundation Implementation & Verification Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a verified, production-ready Milestone 01 for BlockAds Desktop featuring a Go backend contract, a restrained neo-brutalist React desktop UI, full custom rule CRUD (add, edit, delete, toggle), sample telemetry, diagnostics, and zero system-altering side effects.

**Architecture:** Wails v2 desktop shell connecting a Go backend state manager to a React 19 + TypeScript + Tailwind CSS UI. Frontend uses an IPC bridge (`frontend/src/lib/ipc.ts`) that transparently calls Wails Go methods when in desktop mode and provides realistic mock state when tested in standalone browser/DevTools mode.

**Tech Stack:** Go 1.27, Wails v2.17.0, React 19, TypeScript 5.6, Tailwind CSS 4, Vite 7, Lucide React.

**Spec:** User request & `DESIGN.md` specification.

## Global Constraints

- Platform: Windows 10/11 x64.
- Desktop framework: Wails v2 stable (do not migrate to Flutter, Electron, or Wails v3).
- Zero system mutations in Milestone 01: no adapter changes, no registry edits, no drivers, no background services, no CA certificates.
- Neo-brutalist aesthetic adhering to `DESIGN.md` (warm ivory `#F5F1E8`, crisp borders `#121212`, accent `#B38B6D`, semantic tags).
- No generic AI slop: no em dashes (`—`), no fake production metrics presented as live telemetry, no dead interactive elements.

## Review Focus

- Custom rule editing: Ensure domain validation, action toggle (`BLOCK`/`ALLOW`), and notes editing works correctly.
- Standalone browser preview: Ensure the UI does not crash with `Cannot read properties of undefined (reading 'main')` when previewed in browser DevTools.
- Network safety: Verify no actual DNS or system network adapter mutations occur.
- Keyboard and modal accessibility: Ensure modals close on `Escape` and focus is trapped appropriately.
- Build artifact integrity: Ensure `BlockAds-Desktop.exe` builds cleanly via `wails build`.

---

### Task 1: Complete Go Backend Rule Editing & Toggling

**Files:**
- Modify: `app.go`
- Test: `app_test.go`

**Interfaces:**
- Produces:
  - `App.UpdateCustomRule(id string, action string, notes string, enabled bool) error`
  - `App.ToggleCustomRule(id string, enabled bool) error`

- [ ] **Step 1: Write unit tests in `app_test.go` for updating and toggling custom rules**
- [ ] **Step 2: Run `go test -v ./...` to verify test fails**
- [ ] **Step 3: Implement `UpdateCustomRule` and `ToggleCustomRule` in `app.go`**
- [ ] **Step 4: Run `go test -v ./...` to verify all tests pass**

---

### Task 2: Update Wails Bindings & Build Robust Frontend IPC Adapter

**Files:**
- Create: `frontend/src/lib/ipc.ts`
- Modify: `frontend/src/App.tsx`
- Auto-generate: `frontend/wailsjs/` via Wails

**Interfaces:**
- Produces: `frontend/src/lib/ipc.ts` exporting typed async methods matching Go `App` methods with browser fallback.

- [ ] **Step 1: Run `wails build` or binding generation to update `wailsjs`**
- [ ] **Step 2: Create `frontend/src/lib/ipc.ts` with browser fallback support**
- [ ] **Step 3: Wire `frontend/src/App.tsx` to use the unified IPC layer**
- [ ] **Step 4: Run `npm run build` in `frontend/` to verify type checking**

---

### Task 3: Implement Custom Rule Edit/Toggle UI & Antislop Refinement

**Files:**
- Modify: `frontend/src/features/rules/CustomRulesView.tsx`
- Modify: `main.go`

**Interfaces:**
- Consumes: `onUpdateRule` and `onToggleRule` callbacks in `CustomRulesView`
- Produces: Interactive Edit Rule modal, toggle button, and sanitized copy (replacing em dashes with standard hyphens/colons)

- [ ] **Step 1: Update `CustomRulesView.tsx` with Edit modal and toggle button**
- [ ] **Step 2: Replace em dash in `main.go` and `CustomRulesView.tsx`**
- [ ] **Step 3: Run `npm run build` in `frontend/` to confirm zero lint or type errors**

---

### Task 4: Runtime Verification, Chrome DevTools QA, and Final Build

**Files:**
- Execute: `go test -v ./...`
- Execute: `wails build`
- Browser inspection: Launch Vite dev server and inspect with browser / DevTools

- [ ] **Step 1: Execute Go tests (`go test -v ./...`) and record output**
- [ ] **Step 2: Execute Wails production build (`wails build`) and record output**
- [ ] **Step 3: Start frontend dev server, inspect rendered DOM and interactions in browser**
- [ ] **Step 4: Confirm zero console errors and full navigation flow**
- [ ] **Step 5: Prepare final report with empirical evidence**
