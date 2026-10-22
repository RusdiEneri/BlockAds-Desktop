package storage_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"blockads-desktop/internal/storage"
)

func TestStorage_OpenAndPragmas(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open storage: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Verify journal_mode
	var journalMode string
	err = store.DB().QueryRowContext(ctx, "PRAGMA journal_mode;").Scan(&journalMode)
	if err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("expected journal_mode wal, got %s", journalMode)
	}

	// Verify foreign_keys
	var foreignKeys int
	err = store.DB().QueryRowContext(ctx, "PRAGMA foreign_keys;").Scan(&foreignKeys)
	if err != nil {
		t.Fatalf("failed to query foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Errorf("expected foreign_keys 1, got %d", foreignKeys)
	}

	// Verify busy_timeout
	var busyTimeout int
	err = store.DB().QueryRowContext(ctx, "PRAGMA busy_timeout;").Scan(&busyTimeout)
	if err != nil {
		t.Fatalf("failed to query busy_timeout: %v", err)
	}
	if busyTimeout < 5000 {
		t.Errorf("expected busy_timeout >= 5000, got %d", busyTimeout)
	}
}

func TestStorage_Migrations_Idempotent(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "migrations.db")

	store, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open storage: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Run migrations first time
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("first migration failed: %v", err)
	}

	// Verify schema_migrations table has version 1
	var version int
	err = store.DB().QueryRowContext(ctx, "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;").Scan(&version)
	if err != nil {
		t.Fatalf("failed to query schema_migrations: %v", err)
	}
	if version != 2 {
		t.Errorf("expected schema version 2, got %d", version)
	}

	// Run migrations second time (idempotency check)
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("second migration failed: %v", err)
	}
}

func TestStorage_ReopenExisting(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "reopen.db")

	// 1. First open & migrate
	store1, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open store1: %v", err)
	}
	ctx := context.Background()
	if err := store1.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate store1: %v", err)
	}
	if err := store1.Close(); err != nil {
		t.Fatalf("failed to close store1: %v", err)
	}

	// 2. Reopen existing file
	store2, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen store2: %v", err)
	}
	defer store2.Close()

	// Verify schema is present
	var version int
	err = store2.DB().QueryRowContext(ctx, "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;").Scan(&version)
	if err != nil {
		t.Fatalf("failed to query schema_migrations on reopened db: %v", err)
	}
	if version != 2 {
		t.Errorf("expected schema version 2 on reopen, got %d", version)
	}
}

func TestStorage_InvalidPath(t *testing.T) {
	// 1. Empty path
	_, err := storage.Open("")
	if err == nil {
		t.Errorf("expected error opening empty path, got nil")
	}

	// 2. Directory path blocked by existing regular file
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "regular_file.txt")
	if err := os.WriteFile(filePath, []byte("blocking file"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	invalidSubPath := filepath.Join(filePath, "child_dir", "test.db")
	_, err = storage.Open(invalidSubPath)
	if err == nil {
		t.Errorf("expected error opening path inside regular file, got nil")
	}
}
