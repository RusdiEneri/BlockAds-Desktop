# Contributing to BlockAds Desktop

Thank you for your interest in contributing to BlockAds Desktop! We welcome pull requests, bug reports, and suggestions.

## 1. Development Prerequisites

To build and run BlockAds Desktop from source, ensure you have:

- **Go**: 1.24 or newer (tested with Go 1.25).
- **Node.js & npm**: Node.js 20+ LTS and npm 10+.
- **Wails CLI v2**:
  ```powershell
  go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0
  ```
- **WebView2 Runtime**: Included by default in Windows 11 and current Windows 10 releases.
- **Git**: Installed and configured.

## 2. Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/RusdiEneri/BlockAds-Desktop.git
   cd BlockAds-Desktop
   ```

2. Install frontend dependencies:
   ```powershell
   cd frontend
   npm ci
   cd ..
   ```

3. Run in development mode (hot reloading for frontend + backend):
   ```powershell
   wails dev
   ```

## 3. Architecture & Principles

- **Zero CGO**: The application is built with `CGO_ENABLED=0` using `modernc.org/sqlite` to guarantee reproducible Windows cross-compilation without GCC/MinGW dependencies.
- **Minimal Surgery & High Cohesion**: The Go DNS core (`internal/dnsengine`, `internal/cache`, `internal/filtermanager`) remains decoupled from the Wails frontend presentation layer.
- **Anti-Slop & Quality**: Avoid generic AI slop, placeholder UI, and unnecessary third-party dependencies.
- **Truth in Reporting**: The application never falsely reports active system-wide protection when network adapters have not been configured.

## 4. Verification & Testing

Before submitting a Pull Request, verify that all checks pass:

```powershell
# Run all Go unit & integration tests
go test -count=1 ./...

# Run Go static analysis
go vet ./...

# Check code formatting
gofmt -s -l .

# Validate frontend TypeScript compilation & build
npm --prefix frontend run build

# Verify Wails desktop binary compilation
wails build
```

## 5. Submitting Changes

1. Create a feature branch from `master` (`git checkout -b feat/my-feature`).
2. Write unit tests for new logic and bug fixes.
3. Commit using conventional commit format (e.g., `feat: ...`, `fix: ...`, `docs: ...`).
4. Push to your fork and submit a Pull Request describing your changes and verification results.
