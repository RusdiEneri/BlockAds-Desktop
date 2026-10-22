package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type migration struct {
	version int
	name    string
	up      func(ctx context.Context, tx *sql.Tx) error
}

var migrations = []migration{
	{
		version: 1,
		name:    "initial_schema",
		up:      migrationV1,
	},
	{
		version: 2,
		name:    "safesearch_policies",
		up:      migrationV2,
	},
}

// Migrate applies all pending schema migrations sequentially.
func (s *Storage) Migrate(ctx context.Context) error {
	return s.InTransaction(ctx, func(tx *sql.Tx) error {
		// 1. Ensure schema_migrations table exists
		_, err := tx.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version INTEGER PRIMARY KEY,
				name TEXT NOT NULL,
				applied_at TEXT NOT NULL
			);
		`)
		if err != nil {
			return fmt.Errorf("failed to create schema_migrations table: %w", err)
		}

		// 2. Query applied migration versions
		rows, err := tx.QueryContext(ctx, `SELECT version FROM schema_migrations;`)
		if err != nil {
			return fmt.Errorf("failed to query applied migrations: %w", err)
		}
		defer rows.Close()

		applied := make(map[int]bool)
		for rows.Next() {
			var v int
			if err := rows.Scan(&v); err != nil {
				return fmt.Errorf("failed to scan migration version: %w", err)
			}
			applied[v] = true
		}
		if err := rows.Err(); err != nil {
			return err
		}

		// 3. Apply pending migrations in order
		for _, m := range migrations {
			if applied[m.version] {
				continue
			}

			if err := m.up(ctx, tx); err != nil {
				return fmt.Errorf("failed to apply migration %d (%s): %w", m.version, m.name, err)
			}

			_, err = tx.ExecContext(ctx, `
				INSERT INTO schema_migrations (version, name, applied_at)
				VALUES (?, ?, ?);
			`, m.version, m.name, time.Now().UTC().Format(time.RFC3339))
			if err != nil {
				return fmt.Errorf("failed to record migration %d: %w", m.version, err)
			}
		}

		return nil
	})
}

func migrationV1(ctx context.Context, tx *sql.Tx) error {
	schema := `
	-- 1. Application Settings
	CREATE TABLE IF NOT EXISTS app_settings (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		upstream_doh_url TEXT NOT NULL,
		bootstrap_dns TEXT NOT NULL,
		cache_enabled INTEGER NOT NULL DEFAULT 1,
		cache_size INTEGER NOT NULL DEFAULT 4096,
		auto_start INTEGER NOT NULL DEFAULT 0,
		system_dns_integrated INTEGER NOT NULL DEFAULT 0,
		updated_at TEXT NOT NULL
	);

	-- 2. DNS Rewrites
	CREATE TABLE IF NOT EXISTS dns_rewrites (
		id TEXT PRIMARY KEY,
		domain TEXT NOT NULL,
		record_type TEXT NOT NULL,
		target TEXT NOT NULL,
		ttl INTEGER NOT NULL DEFAULT 60,
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		UNIQUE(domain, record_type)
	);

	-- 3. Custom Rules (Allowlist / Blocklist)
	CREATE TABLE IF NOT EXISTS custom_rules (
		id TEXT PRIMARY KEY,
		domain TEXT NOT NULL UNIQUE,
		action TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL,
		notes TEXT NOT NULL DEFAULT ''
	);

	-- 4. Filter Subscriptions Metadata
	CREATE TABLE IF NOT EXISTS filter_subscriptions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		source_url TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 1,
		format TEXT NOT NULL DEFAULT 'hosts',
		etag TEXT NOT NULL DEFAULT '',
		last_modified TEXT NOT NULL DEFAULT '',
		last_attempt_at TEXT NOT NULL DEFAULT '',
		last_success_at TEXT NOT NULL DEFAULT '',
		last_status TEXT NOT NULL DEFAULT 'pending',
		last_error TEXT NOT NULL DEFAULT '',
		content_checksum TEXT NOT NULL DEFAULT '',
		rule_count INTEGER NOT NULL DEFAULT 0,
		description TEXT NOT NULL DEFAULT '',
		category TEXT NOT NULL DEFAULT '',
		updated_at TEXT NOT NULL
	);

	-- 5. Filter Snapshots (Atomic BLOB/Text storage with Checksum Integrity)
	CREATE TABLE IF NOT EXISTS filter_snapshots (
		subscription_id TEXT NOT NULL,
		checksum TEXT NOT NULL,
		content TEXT NOT NULL,
		rule_count INTEGER NOT NULL DEFAULT 0,
		size_bytes INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		PRIMARY KEY (subscription_id, checksum),
		FOREIGN KEY (subscription_id) REFERENCES filter_subscriptions(id) ON DELETE CASCADE
	);

	-- 6. Query Events (Bounded Historical Log)
	CREATE TABLE IF NOT EXISTS query_events (
		id TEXT PRIMARY KEY,
		timestamp TEXT NOT NULL,
		domain TEXT NOT NULL,
		client_ip TEXT NOT NULL,
		query_type TEXT NOT NULL,
		status TEXT NOT NULL,
		upstream_latency_ms INTEGER NOT NULL DEFAULT 0,
		filter_rule TEXT NOT NULL DEFAULT '',
		reason TEXT NOT NULL DEFAULT '',
		protocol TEXT NOT NULL DEFAULT 'UDP',
		rcode INTEGER NOT NULL DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_query_events_timestamp ON query_events(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_query_events_domain ON query_events(domain);
	CREATE INDEX IF NOT EXISTS idx_query_events_status ON query_events(status);
	`

	_, err := tx.ExecContext(ctx, schema)
	return err
}

func migrationV2(ctx context.Context, tx *sql.Tx) error {
	schema := `
	CREATE TABLE IF NOT EXISTS safesearch_policies (
		provider_id TEXT PRIMARY KEY,
		enabled INTEGER NOT NULL DEFAULT 0,
		updated_at TEXT NOT NULL
	);
	`
	_, err := tx.ExecContext(ctx, schema)
	return err
}
