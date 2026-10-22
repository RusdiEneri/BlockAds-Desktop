package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"blockads-desktop/internal/models"
)

// GetCustomRules retrieves all custom allow/block rules ordered by creation time descending.
func (s *Storage) GetCustomRules(ctx context.Context) ([]models.CustomRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, domain, action, enabled, created_at, notes
		FROM custom_rules
		ORDER BY created_at DESC;
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query custom_rules: %w", err)
	}
	defer rows.Close()

	var rules []models.CustomRule
	for rows.Next() {
		var r models.CustomRule
		var enabledInt int
		if err := rows.Scan(&r.ID, &r.Domain, &r.Action, &enabledInt, &r.CreatedAt, &r.Notes); err != nil {
			return nil, fmt.Errorf("failed to scan custom_rule row: %w", err)
		}
		r.Enabled = enabledInt == 1
		rules = append(rules, r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rules, nil
}

// SaveCustomRule validates and stores a custom rule.
func (s *Storage) SaveCustomRule(ctx context.Context, rule models.CustomRule) error {
	domain := strings.ToLower(strings.TrimSpace(rule.Domain))
	domain = strings.TrimSuffix(domain, ".")
	if err := models.ValidateDomain(domain); err != nil {
		return fmt.Errorf("%w: invalid domain: %v", ErrInvalidInput, err)
	}

	action := strings.ToUpper(strings.TrimSpace(rule.Action))
	if action != "BLOCK" && action != "ALLOW" {
		return fmt.Errorf("%w: action must be BLOCK or ALLOW", ErrInvalidInput)
	}

	id := strings.TrimSpace(rule.ID)
	if id == "" {
		id = fmt.Sprintf("rule-%d", time.Now().UnixNano())
	}

	createdAt := strings.TrimSpace(rule.CreatedAt)
	if createdAt == "" {
		createdAt = time.Now().UTC().Format(time.RFC3339)
	}

	enabledInt := 0
	if rule.Enabled {
		enabledInt = 1
	}

	query := `
		INSERT INTO custom_rules (id, domain, action, enabled, created_at, notes)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(domain) DO UPDATE SET
			action = excluded.action,
			enabled = excluded.enabled,
			notes = excluded.notes;
	`

	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, query, id, domain, action, enabledInt, createdAt, strings.TrimSpace(rule.Notes))
	if err != nil {
		return fmt.Errorf("failed to save custom_rule: %w", err)
	}

	return nil
}

// UpdateCustomRule updates the action, notes, and enabled state of an existing custom rule.
func (s *Storage) UpdateCustomRule(ctx context.Context, id string, action string, notes string, enabled bool) error {
	normalizedAction := strings.ToUpper(strings.TrimSpace(action))
	if normalizedAction != "BLOCK" && normalizedAction != "ALLOW" {
		return fmt.Errorf("%w: action must be BLOCK or ALLOW", ErrInvalidInput)
	}

	enabledInt := 0
	if enabled {
		enabledInt = 1
	}

	query := `
		UPDATE custom_rules
		SET action = ?, notes = ?, enabled = ?
		WHERE id = ?;
	`

	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.ExecContext(ctx, query, normalizedAction, strings.TrimSpace(notes), enabledInt, id)
	if err != nil {
		return fmt.Errorf("failed to update custom_rule: %w", err)
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

// ToggleCustomRule updates the enabled status of an existing custom rule.
func (s *Storage) ToggleCustomRule(ctx context.Context, id string, enabled bool) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}

	query := `UPDATE custom_rules SET enabled = ? WHERE id = ?;`

	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.ExecContext(ctx, query, enabledInt, id)
	if err != nil {
		return fmt.Errorf("failed to toggle custom_rule: %w", err)
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

// DeleteCustomRule deletes a custom rule by its unique ID.
func (s *Storage) DeleteCustomRule(ctx context.Context, id string) error {
	query := `DELETE FROM custom_rules WHERE id = ?;`

	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete custom_rule: %w", err)
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
