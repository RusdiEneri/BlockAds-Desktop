package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

// Common storage errors.
var (
	ErrNotFound        = errors.New("record not found")
	ErrConflict        = errors.New("record already exists or conflicts")
	ErrInvalidInput    = errors.New("invalid storage input")
	ErrCorruptSnapshot = errors.New("filter snapshot failed checksum integrity check")
)

// Storage wraps the SQLite database and coordinates migrations and repositories.
type Storage struct {
	mu   sync.RWMutex
	db   *sql.DB
	path string
}

// Open initializes and configures a SQLite database at the specified path.
func Open(path string) (*Storage, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("database path cannot be empty")
	}

	// For file-based SQLite databases, ensure directory exists
	if path != ":memory:" && !strings.HasPrefix(path, "file::memory:") {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Single connection model for local desktop app: eliminates SQLite lock contention
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Apply core pragmas
	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
	}

	if path != ":memory:" && !strings.HasPrefix(path, "file::memory:") {
		pragmas = append(pragmas,
			"PRAGMA journal_mode = WAL;",
			"PRAGMA synchronous = NORMAL;",
		)
	}

	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma %q: %w", p, err)
		}
	}

	s := &Storage{
		db:   db,
		path: path,
	}

	return s, nil
}

// DB returns the underlying sql.DB instance for testing or advanced queries.
func (s *Storage) DB() *sql.DB {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db
}

// Path returns the filesystem path of the database.
func (s *Storage) Path() string {
	return s.path
}

// Close gracefully closes the SQLite database connection.
func (s *Storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

// InTransaction executes the given function within a database transaction.
// If fn returns an error, the transaction is rolled back; otherwise it is committed.
func (s *Storage) InTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error {
	s.mu.RLock()
	db := s.db
	s.mu.RUnlock()

	if db == nil {
		return errors.New("database is closed")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
