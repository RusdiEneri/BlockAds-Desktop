package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"blockads-desktop/internal/models"
)

// MaxSnapshotBytes is the maximum allowed size for a single filter list snapshot (25MB).
const MaxSnapshotBytes = 25 * 1024 * 1024

// FilterSubscription represents metadata for an upstream filter list subscription.
type FilterSubscription struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	SourceURL       string `json:"sourceUrl"`
	Enabled         bool   `json:"enabled"`
	Format          string `json:"format"`
	ETag            string `json:"etag"`
	LastModified    string `json:"lastModified"`
	LastAttemptAt   string `json:"lastAttemptAt"`
	LastSuccessAt   string `json:"lastSuccessAt"`
	LastStatus      string `json:"lastStatus"`
	LastError       string `json:"lastError"`
	ContentChecksum string `json:"contentChecksum"`
	RuleCount       int    `json:"ruleCount"`
	Description     string `json:"description"`
	Category        string `json:"category"`
	UpdatedAt       string `json:"updatedAt"`
}

// SnapshotMeta holds HTTP caching headers associated with a filter snapshot.
type SnapshotMeta struct {
	ETag         string
	LastModified string
}

// GetFilterSubscriptions returns all filter list subscriptions.
func (s *Storage) GetFilterSubscriptions(ctx context.Context) ([]FilterSubscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, name, source_url, enabled, format, etag, last_modified,
		       last_attempt_at, last_success_at, last_status, last_error,
		       content_checksum, rule_count, description, category, updated_at
		FROM filter_subscriptions
		ORDER BY id ASC;
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query filter_subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []FilterSubscription
	for rows.Next() {
		var sub FilterSubscription
		var enabledInt int
		err := rows.Scan(
			&sub.ID, &sub.Name, &sub.SourceURL, &enabledInt, &sub.Format,
			&sub.ETag, &sub.LastModified, &sub.LastAttemptAt, &sub.LastSuccessAt,
			&sub.LastStatus, &sub.LastError, &sub.ContentChecksum, &sub.RuleCount,
			&sub.Description, &sub.Category, &sub.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan filter_subscription: %w", err)
		}
		sub.Enabled = enabledInt == 1
		subs = append(subs, sub)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subs, nil
}

// GetFilterSubscription retrieves a single filter subscription by ID.
func (s *Storage) GetFilterSubscription(ctx context.Context, id string) (*FilterSubscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, name, source_url, enabled, format, etag, last_modified,
		       last_attempt_at, last_success_at, last_status, last_error,
		       content_checksum, rule_count, description, category, updated_at
		FROM filter_subscriptions
		WHERE id = ?;
	`

	var sub FilterSubscription
	var enabledInt int
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&sub.ID, &sub.Name, &sub.SourceURL, &enabledInt, &sub.Format,
		&sub.ETag, &sub.LastModified, &sub.LastAttemptAt, &sub.LastSuccessAt,
		&sub.LastStatus, &sub.LastError, &sub.ContentChecksum, &sub.RuleCount,
		&sub.Description, &sub.Category, &sub.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query filter_subscription: %w", err)
	}

	sub.Enabled = enabledInt == 1
	return &sub, nil
}

// GetFilterSubscriptionByURL retrieves a subscription by comparing normalized source URLs.
func (s *Storage) GetFilterSubscriptionByURL(ctx context.Context, rawURL string) (*FilterSubscription, error) {
	normURL, err := models.NormalizeURL(rawURL)
	if err != nil {
		normURL = strings.TrimSpace(rawURL)
	}

	subs, err := s.GetFilterSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	for _, sub := range subs {
		existingNorm, err := models.NormalizeURL(sub.SourceURL)
		if err != nil {
			existingNorm = strings.TrimSpace(sub.SourceURL)
		}
		if strings.EqualFold(normURL, existingNorm) {
			return &sub, nil
		}
	}

	return nil, ErrNotFound
}

// SaveFilterSubscription creates or updates a filter subscription metadata record.
// It normalizes source URLs and prevents duplicate subscriptions with different IDs.
func (s *Storage) SaveFilterSubscription(ctx context.Context, sub FilterSubscription) error {
	id := strings.TrimSpace(sub.ID)
	if id == "" {
		return fmt.Errorf("%w: subscription ID cannot be empty", ErrInvalidInput)
	}
	name := strings.TrimSpace(sub.Name)
	if name == "" {
		return fmt.Errorf("%w: subscription name cannot be empty", ErrInvalidInput)
	}
	srcURL := strings.TrimSpace(sub.SourceURL)
	if srcURL == "" {
		return fmt.Errorf("%w: source URL cannot be empty", ErrInvalidInput)
	}

	normURL, err := models.NormalizeURL(srcURL)
	if err == nil {
		srcURL = normURL
	}

	format := strings.ToLower(strings.TrimSpace(sub.Format))
	if format == "" {
		format = "hosts"
	}

	enabledInt := 0
	if sub.Enabled {
		enabledInt = 1
	}

	now := time.Now().UTC().Format(time.RFC3339)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Enforce duplicate URL prevention across different subscription IDs
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, source_url FROM filter_subscriptions WHERE id != ?;`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var otherID, otherName, otherURL string
			if scanErr := rows.Scan(&otherID, &otherName, &otherURL); scanErr == nil {
				otherNorm, nErr := models.NormalizeURL(otherURL)
				if nErr != nil {
					otherNorm = strings.TrimSpace(otherURL)
				}
				if strings.EqualFold(srcURL, otherNorm) {
					return fmt.Errorf("%w: duplicate subscription URL matches %q (id: %s)", ErrConflict, otherName, otherID)
				}
			}
		}
	}

	query := `
		INSERT INTO filter_subscriptions (
			id, name, source_url, enabled, format, etag, last_modified,
			last_attempt_at, last_success_at, last_status, last_error,
			content_checksum, rule_count, description, category, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			source_url = excluded.source_url,
			enabled = excluded.enabled,
			format = excluded.format,
			description = excluded.description,
			category = excluded.category,
			updated_at = excluded.updated_at;
	`

	_, err = s.db.ExecContext(ctx, query,
		id, name, srcURL, enabledInt, format,
		sub.ETag, sub.LastModified, sub.LastAttemptAt, sub.LastSuccessAt,
		sub.LastStatus, sub.LastError, sub.ContentChecksum, sub.RuleCount,
		sub.Description, sub.Category, now,
	)

	if err != nil {
		return fmt.Errorf("failed to save filter_subscription: %w", err)
	}

	return nil
}

// ToggleFilterSubscription enables or disables a filter subscription.
func (s *Storage) ToggleFilterSubscription(ctx context.Context, id string, enabled bool) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}

	query := `UPDATE filter_subscriptions SET enabled = ?, updated_at = ? WHERE id = ?;`

	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.ExecContext(ctx, query, enabledInt, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to toggle filter_subscription: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteFilterSubscription removes a subscription and cascades to its snapshots.
func (s *Storage) DeleteFilterSubscription(ctx context.Context, id string) error {
	query := `DELETE FROM filter_subscriptions WHERE id = ?;`

	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete filter_subscription: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// SaveFilterSnapshot atomically stores validated filter content and updates subscription metadata.
// It verifies size limits, computes SHA-256, avoids storing duplicate snapshots, and preserves
// existing data on failure.
func (s *Storage) SaveFilterSnapshot(ctx context.Context, subID string, content string, ruleCount int, meta SnapshotMeta) (string, error) {
	if len(content) > MaxSnapshotBytes {
		return "", fmt.Errorf("%w: snapshot content size %d bytes exceeds maximum allowed (%d bytes)",
			ErrInvalidInput, len(content), MaxSnapshotBytes)
	}

	hasher := sha256.New()
	hasher.Write([]byte(content))
	checksum := hex.EncodeToString(hasher.Sum(nil))
	now := time.Now().UTC().Format(time.RFC3339)

	err := s.InTransaction(ctx, func(tx *sql.Tx) error {
		// 1. Verify subscription exists
		var exists int
		err := tx.QueryRowContext(ctx, `SELECT 1 FROM filter_subscriptions WHERE id = ?;`, subID).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("failed to check subscription: %w", err)
		}

		// 2. Insert snapshot if not already present (deduplication)
		_, err = tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO filter_snapshots (subscription_id, checksum, content, rule_count, size_bytes, created_at)
			VALUES (?, ?, ?, ?, ?, ?);
		`, subID, checksum, content, ruleCount, len(content), now)
		if err != nil {
			return fmt.Errorf("failed to insert filter_snapshot: %w", err)
		}

		// 3. Update subscription metadata atomically
		_, err = tx.ExecContext(ctx, `
			UPDATE filter_subscriptions
			SET content_checksum = ?,
			    rule_count = ?,
			    etag = ?,
			    last_modified = ?,
			    last_attempt_at = ?,
			    last_success_at = ?,
			    last_status = 'ok',
			    last_error = '',
			    updated_at = ?
			WHERE id = ?;
		`, checksum, ruleCount, meta.ETag, meta.LastModified, now, now, now, subID)
		if err != nil {
			return fmt.Errorf("failed to update subscription with new snapshot: %w", err)
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return checksum, nil
}

// GetFilterSnapshot retrieves the active snapshot content for a subscription and verifies its SHA-256 checksum.
func (s *Storage) GetFilterSnapshot(ctx context.Context, subID string) (string, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 1. Look up active content checksum
	var checksum string
	err := s.db.QueryRowContext(ctx, `
		SELECT content_checksum
		FROM filter_subscriptions
		WHERE id = ?;
	`, subID).Scan(&checksum)

	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to query subscription checksum: %w", err)
	}

	if strings.TrimSpace(checksum) == "" {
		return "", "", ErrNotFound
	}

	// 2. Fetch snapshot content
	var content string
	var storedChecksum string
	err = s.db.QueryRowContext(ctx, `
		SELECT content, checksum
		FROM filter_snapshots
		WHERE subscription_id = ? AND checksum = ?;
	`, subID, checksum).Scan(&content, &storedChecksum)

	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to query snapshot content: %w", err)
	}

	// 3. Verify SHA-256 integrity
	hasher := sha256.New()
	hasher.Write([]byte(content))
	actualChecksum := hex.EncodeToString(hasher.Sum(nil))

	if actualChecksum != storedChecksum {
		return "", "", fmt.Errorf("%w: stored %s, actual %s", ErrCorruptSnapshot, storedChecksum, actualChecksum)
	}

	return content, storedChecksum, nil
}

// RecordFilterFetchError updates subscription attempt timestamp and error without erasing active snapshot.
func (s *Storage) RecordFilterFetchError(ctx context.Context, subID string, errMsg string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	query := `
		UPDATE filter_subscriptions
		SET last_attempt_at = ?,
		    last_status = 'failed',
		    last_error = ?,
		    updated_at = ?
		WHERE id = ?;
	`

	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.ExecContext(ctx, query, now, strings.TrimSpace(errMsg), now, subID)
	if err != nil {
		return fmt.Errorf("failed to record filter fetch error: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// RecordFilterFetchNotModified updates timestamps and HTTP validators for a 304 response
// without altering the active snapshot content or rule count.
func (s *Storage) RecordFilterFetchNotModified(ctx context.Context, subID string, meta SnapshotMeta) error {
	now := time.Now().UTC().Format(time.RFC3339)

	query := `
		UPDATE filter_subscriptions
		SET last_attempt_at = ?,
		    last_success_at = ?,
		    last_status = 'ok',
		    last_error = '',
		    etag = CASE WHEN ? != '' THEN ? ELSE etag END,
		    last_modified = CASE WHEN ? != '' THEN ? ELSE last_modified END,
		    updated_at = ?
		WHERE id = ?;
	`

	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.ExecContext(ctx, query,
		now, now,
		meta.ETag, meta.ETag,
		meta.LastModified, meta.LastModified,
		now, subID,
	)
	if err != nil {
		return fmt.Errorf("failed to record 304 not modified: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}
