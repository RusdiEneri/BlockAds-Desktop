package storage

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SafeSearchPolicyRecord represents a persistent provider policy setting.
type SafeSearchPolicyRecord struct {
	ProviderID string `json:"providerId"`
	Enabled    bool   `json:"enabled"`
	UpdatedAt  string `json:"updatedAt"`
}

// GetSafeSearchPolicies retrieves a map of provider_id to enabled state.
func (s *Storage) GetSafeSearchPolicies(ctx context.Context) (map[string]bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT provider_id, enabled FROM safesearch_policies;`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query safesearch_policies: %w", err)
	}
	defer rows.Close()

	policies := make(map[string]bool)
	for rows.Next() {
		var id string
		var enabledInt int
		if err := rows.Scan(&id, &enabledInt); err != nil {
			return nil, fmt.Errorf("failed to scan safesearch policy: %w", err)
		}
		policies[id] = enabledInt == 1
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return policies, nil
}

// GetSafeSearchPolicyRecords retrieves all stored SafeSearch policy records with timestamps.
func (s *Storage) GetSafeSearchPolicyRecords(ctx context.Context) ([]SafeSearchPolicyRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT provider_id, enabled, updated_at FROM safesearch_policies ORDER BY provider_id ASC;`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query safesearch policy records: %w", err)
	}
	defer rows.Close()

	var records []SafeSearchPolicyRecord
	for rows.Next() {
		var r SafeSearchPolicyRecord
		var enabledInt int
		if err := rows.Scan(&r.ProviderID, &enabledInt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan safesearch policy record: %w", err)
		}
		r.Enabled = enabledInt == 1
		records = append(records, r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

// SetSafeSearchPolicy creates or updates the persistent enabled state of a SafeSearch provider policy.
func (s *Storage) SetSafeSearchPolicy(ctx context.Context, providerID string, enabled bool) error {
	cleanID := strings.ToLower(strings.TrimSpace(providerID))
	if cleanID == "" {
		return fmt.Errorf("%w: provider ID cannot be empty", ErrInvalidInput)
	}

	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)

	query := `
		INSERT INTO safesearch_policies (provider_id, enabled, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(provider_id) DO UPDATE SET
			enabled = excluded.enabled,
			updated_at = excluded.updated_at;
	`

	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, query, cleanID, enabledInt, now)
	if err != nil {
		return fmt.Errorf("failed to save safesearch policy for %s: %w", cleanID, err)
	}

	return nil
}
