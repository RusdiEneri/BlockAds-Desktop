package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"blockads-desktop/internal/storage"
)

func TestResolveDataDirectory_ExplicitPath(t *testing.T) {
	tmpDir := t.TempDir()
	customDB := filepath.Join(tmpDir, "custom", "test.db")

	resolvedDB, isPortable, err := storage.ResolveDatabasePath(customDB, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolvedDB != customDB {
		t.Errorf("expected explicit db path %q, got %q", customDB, resolvedDB)
	}
	if isPortable {
		t.Errorf("explicit path should not be marked portable by default")
	}
}

func TestResolveDataDirectory_PortableMarker(t *testing.T) {
	tmpDir := t.TempDir()
	markerFile := filepath.Join(tmpDir, "portable.marker")
	if err := os.WriteFile(markerFile, []byte("portable"), 0644); err != nil {
		t.Fatalf("failed to create marker file: %v", err)
	}

	fakeExe := filepath.Join(tmpDir, "BlockAds-Desktop.exe")

	resolvedDB, isPortable, err := storage.ResolveDatabasePath("", fakeExe)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isPortable {
		t.Errorf("expected portable mode to be detected with portable.marker")
	}

	expectedDB := filepath.Join(tmpDir, "data", "blockads.db")
	if resolvedDB != expectedDB {
		t.Errorf("expected portable db %q, got %q", expectedDB, resolvedDB)
	}

	// Verify data directory was created
	dataDir := filepath.Dir(resolvedDB)
	if fi, err := os.Stat(dataDir); err != nil || !fi.IsDir() {
		t.Errorf("expected data directory %q to exist and be a directory", dataDir)
	}
}

func TestResolveDataDirectory_InstalledFallback(t *testing.T) {
	tmpDir := t.TempDir()
	fakeExe := filepath.Join(tmpDir, "Programs", "BlockAds Desktop", "BlockAds-Desktop.exe")

	// Set temporary LOCALAPPDATA
	origLocalApp := os.Getenv("LOCALAPPDATA")
	defer os.Setenv("LOCALAPPDATA", origLocalApp)

	fakeLocalApp := filepath.Join(tmpDir, "AppData", "Local")
	os.Setenv("LOCALAPPDATA", fakeLocalApp)

	resolvedDB, isPortable, err := storage.ResolveDatabasePath("", fakeExe)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isPortable {
		t.Errorf("expected installed mode, got portable")
	}

	expectedDB := filepath.Join(fakeLocalApp, "BlockAds Desktop", "blockads.db")
	if resolvedDB != expectedDB {
		t.Errorf("expected installed db %q, got %q", expectedDB, resolvedDB)
	}
}
