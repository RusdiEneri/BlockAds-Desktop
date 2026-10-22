package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveDatabasePath determines the active SQLite database path and whether the application is running in portable mode.
// In portable mode, all persistent databases, snapshots, and logs are kept inside a local 'data/' directory next to the executable.
// In installed mode, files reside in '%LOCALAPPDATA%\BlockAds Desktop' to isolate user data from binaries and upgrades.
func ResolveDatabasePath(explicitDBPath, executablePath string) (string, bool, error) {
	// 1. Explicit override (e.g. from BLOCKADS_DB_PATH or test runner)
	if strings.TrimSpace(explicitDBPath) != "" {
		return strings.TrimSpace(explicitDBPath), false, nil
	}

	exePath := executablePath
	if exePath == "" {
		var err error
		exePath, err = os.Executable()
		if err != nil {
			exePath = "."
		}
	}

	exeDir, err := filepath.Abs(filepath.Dir(exePath))
	if err != nil {
		exeDir = filepath.Dir(exePath)
	}

	// 2. Portable mode detection
	isPortable := false
	if os.Getenv("BLOCKADS_PORTABLE") == "1" {
		isPortable = true
	} else if _, err := os.Stat(filepath.Join(exeDir, "portable.marker")); err == nil {
		isPortable = true
	} else if _, err := os.Stat(filepath.Join(exeDir, "portable.dat")); err == nil {
		isPortable = true
	} else if fi, err := os.Stat(filepath.Join(exeDir, "data")); err == nil && fi.IsDir() {
		// If data/ directory already exists alongside executable
		isPortable = true
	}

	if isPortable {
		dataDir := filepath.Join(exeDir, "data")
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return "", true, fmt.Errorf("portable data directory %q is not writable: %w", dataDir, err)
		}

		// Verify writability with a test file
		probeFile := filepath.Join(dataDir, ".write_probe")
		if err := os.WriteFile(probeFile, []byte("ok"), 0644); err != nil {
			return "", true, fmt.Errorf("portable data directory %q is not writable: %w", dataDir, err)
		}
		_ = os.Remove(probeFile)

		return filepath.Join(dataDir, "blockads.db"), true, nil
	}

	// 3. Installed mode (%LOCALAPPDATA%\BlockAds Desktop)
	var baseDir string
	if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
		baseDir = localApp
	} else if appData := os.Getenv("APPDATA"); appData != "" {
		baseDir = appData
	} else {
		configDir, err := os.UserConfigDir()
		if err == nil && configDir != "" {
			baseDir = configDir
		} else {
			baseDir = "."
		}
	}

	dataDir := filepath.Join(baseDir, "BlockAds Desktop")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", false, fmt.Errorf("failed to create application data directory %q: %w", dataDir, err)
	}

	return filepath.Join(dataDir, "blockads.db"), false, nil
}

// GetDataDirFromDBPath returns the enclosing directory for a database path.
func GetDataDirFromDBPath(dbPath string) string {
	if dbPath == ":memory:" || strings.HasPrefix(dbPath, "file::memory:") {
		return ":memory:"
	}
	abs, err := filepath.Abs(filepath.Dir(dbPath))
	if err != nil {
		return filepath.Dir(dbPath)
	}
	return abs
}
