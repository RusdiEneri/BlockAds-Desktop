package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"blockads-desktop/internal/models"
)

var defaultSettings = models.AppSettings{
	UpstreamDoHURL:      "https://dns.caliph.dev/dns-query",
	BootstrapDNS:        "1.1.1.1",
	CacheEnabled:        true,
	CacheSize:           4096,
	AutoStart:           false,
	SystemDNSIntegrated: false,
}

// GetSettings retrieves the persistent application settings.
// If no settings record exists yet, default settings are returned.
func (s *Storage) GetSettings(ctx context.Context) (models.AppSettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT upstream_doh_url, bootstrap_dns, cache_enabled, cache_size, auto_start, system_dns_integrated
		FROM app_settings
		WHERE id = 1;
	`

	var set models.AppSettings
	var cacheEnabled, autoStart, systemDNS int

	err := s.db.QueryRowContext(ctx, query).Scan(
		&set.UpstreamDoHURL,
		&set.BootstrapDNS,
		&cacheEnabled,
		&set.CacheSize,
		&autoStart,
		&systemDNS,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return defaultSettings, nil
	}
	if err != nil {
		return defaultSettings, fmt.Errorf("failed to query app_settings: %w", err)
	}

	set.CacheEnabled = cacheEnabled == 1
	set.AutoStart = autoStart == 1
	set.SystemDNSIntegrated = systemDNS == 1

	return set, nil
}

// UpdateSettings validates and updates the application settings in SQLite.
func (s *Storage) UpdateSettings(ctx context.Context, set models.AppSettings) error {
	// Validate values
	if err := models.ValidateUpstreamDoHURL(set.UpstreamDoHURL); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	if set.CacheSize <= 0 {
		return fmt.Errorf("%w: cache size must be greater than 0", ErrInvalidInput)
	}
	if strings.TrimSpace(set.BootstrapDNS) == "" {
		set.BootstrapDNS = "1.1.1.1"
	}

	cacheEnabled := 0
	if set.CacheEnabled {
		cacheEnabled = 1
	}
	autoStart := 0
	if set.AutoStart {
		autoStart = 1
	}
	sysDNS := 0
	if set.SystemDNSIntegrated {
		sysDNS = 1
	}

	query := `
		INSERT INTO app_settings (id, upstream_doh_url, bootstrap_dns, cache_enabled, cache_size, auto_start, system_dns_integrated, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			upstream_doh_url = excluded.upstream_doh_url,
			bootstrap_dns = excluded.bootstrap_dns,
			cache_enabled = excluded.cache_enabled,
			cache_size = excluded.cache_size,
			auto_start = excluded.auto_start,
			system_dns_integrated = excluded.system_dns_integrated,
			updated_at = excluded.updated_at;
	`

	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, query,
		strings.TrimSpace(set.UpstreamDoHURL),
		strings.TrimSpace(set.BootstrapDNS),
		cacheEnabled,
		set.CacheSize,
		autoStart,
		sysDNS,
		time.Now().UTC().Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to save app_settings: %w", err)
	}

	return nil
}
