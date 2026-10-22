package storage

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"blockads-desktop/internal/models"
	"blockads-desktop/internal/rewrites"
)

// GetRewrites retrieves all persistent DNS rewrites ordered by domain.
func (s *Storage) GetRewrites(ctx context.Context) ([]rewrites.Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT domain, record_type, target, ttl, enabled
		FROM dns_rewrites
		ORDER BY domain ASC, record_type ASC;
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query dns_rewrites: %w", err)
	}
	defer rows.Close()

	var rules []rewrites.Rule
	for rows.Next() {
		var r rewrites.Rule
		var enabledInt int
		if err := rows.Scan(&r.Domain, &r.Type, &r.Target, &r.TTL, &enabledInt); err != nil {
			return nil, fmt.Errorf("failed to scan dns_rewrite row: %w", err)
		}
		r.Enabled = enabledInt == 1
		rules = append(rules, r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rules, nil
}

// SaveRewrite validates and upserts a DNS rewrite rule.
func (s *Storage) SaveRewrite(ctx context.Context, rule rewrites.Rule) error {
	domain := strings.ToLower(strings.TrimSpace(rule.Domain))
	domain = strings.TrimSuffix(domain, ".")
	if domain == "" {
		return fmt.Errorf("%w: domain cannot be empty", ErrInvalidInput)
	}

	// Validate wildcard or exact domain
	if strings.HasPrefix(domain, "*.") {
		parent := strings.TrimPrefix(domain, "*.")
		if err := models.ValidateDomain(parent); err != nil {
			return fmt.Errorf("%w: invalid wildcard parent domain: %v", ErrInvalidInput, err)
		}
	} else {
		if err := models.ValidateDomain(domain); err != nil {
			return fmt.Errorf("%w: invalid domain: %v", ErrInvalidInput, err)
		}
	}

	qType := strings.ToUpper(strings.TrimSpace(rule.Type))
	target := strings.TrimSpace(rule.Target)

	// Validate target by type
	switch qType {
	case "A":
		ip := net.ParseIP(target)
		if ip == nil || ip.To4() == nil {
			return fmt.Errorf("%w: target %q is not a valid IPv4 address", ErrInvalidInput, target)
		}
	case "AAAA":
		ip := net.ParseIP(target)
		if ip == nil || ip.To4() != nil {
			return fmt.Errorf("%w: target %q is not a valid IPv6 address", ErrInvalidInput, target)
		}
	case "CNAME":
		targetLower := strings.ToLower(target)
		targetLower = strings.TrimSuffix(targetLower, ".")
		if ip := net.ParseIP(target); ip != nil {
			return fmt.Errorf("%w: CNAME target cannot be an IP address", ErrInvalidInput)
		}
		if err := models.ValidateDomain(targetLower); err != nil {
			return fmt.Errorf("%w: CNAME target must be a valid FQDN: %v", ErrInvalidInput, err)
		}
		if targetLower == domain {
			return fmt.Errorf("%w: direct circular CNAME loop (self-referential)", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: unsupported rewrite record type %q (must be A, AAAA, or CNAME)", ErrInvalidInput, qType)
	}

	ttl := rule.TTL
	if ttl == 0 {
		ttl = 60
	}

	enabledInt := 0
	if rule.Enabled {
		enabledInt = 1
	}

	id := fmt.Sprintf("rewrite-%s-%s", domain, qType)
	now := time.Now().UTC().Format(time.RFC3339)

	query := `
		INSERT INTO dns_rewrites (id, domain, record_type, target, ttl, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(domain, record_type) DO UPDATE SET
			target = excluded.target,
			ttl = excluded.ttl,
			enabled = excluded.enabled,
			updated_at = excluded.updated_at;
	`

	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, query, id, domain, qType, target, ttl, enabledInt, now, now)
	if err != nil {
		return fmt.Errorf("failed to save dns_rewrite: %w", err)
	}

	return nil
}

// DeleteRewrite deletes a DNS rewrite rule by domain and record type.
func (s *Storage) DeleteRewrite(ctx context.Context, domain string, recordType string) error {
	cleanDomain := strings.ToLower(strings.TrimSpace(domain))
	cleanDomain = strings.TrimSuffix(cleanDomain, ".")
	cleanType := strings.ToUpper(strings.TrimSpace(recordType))

	query := `DELETE FROM dns_rewrites WHERE domain = ? AND record_type = ?;`

	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.ExecContext(ctx, query, cleanDomain, cleanType)
	if err != nil {
		return fmt.Errorf("failed to delete dns_rewrite: %w", err)
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

// ToggleRewrite updates the enabled flag of a DNS rewrite rule.
func (s *Storage) ToggleRewrite(ctx context.Context, domain string, recordType string, enabled bool) error {
	cleanDomain := strings.ToLower(strings.TrimSpace(domain))
	cleanDomain = strings.TrimSuffix(cleanDomain, ".")
	cleanType := strings.ToUpper(strings.TrimSpace(recordType))

	enabledInt := 0
	if enabled {
		enabledInt = 1
	}

	query := `
		UPDATE dns_rewrites
		SET enabled = ?, updated_at = ?
		WHERE domain = ? AND record_type = ?;
	`

	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.ExecContext(ctx, query, enabledInt, time.Now().UTC().Format(time.RFC3339), cleanDomain, cleanType)
	if err != nil {
		return fmt.Errorf("failed to toggle dns_rewrite: %w", err)
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
