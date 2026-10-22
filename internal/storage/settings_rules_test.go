package storage_test

import (
	"context"
	"path/filepath"
	"testing"

	"blockads-desktop/internal/models"
	"blockads-desktop/internal/rewrites"
	"blockads-desktop/internal/storage"
)

func setupTestStorage(t *testing.T) (*storage.Storage, context.Context) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open storage: %v", err)
	}

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		store.Close()
		t.Fatalf("failed to migrate storage: %v", err)
	}

	t.Cleanup(func() {
		store.Close()
	})

	return store, ctx
}

func TestStorage_Settings_DefaultAndLifecycle(t *testing.T) {
	store, ctx := setupTestStorage(t)

	// 1. Initial settings when fresh should be defaults
	settings, err := store.GetSettings(ctx)
	if err != nil {
		t.Fatalf("failed to get settings: %v", err)
	}
	if settings.UpstreamDoHURL == "" {
		t.Errorf("expected non-empty default upstream URL")
	}

	// 2. Update settings with valid data
	updated := settings
	updated.UpstreamDoHURL = "https://1.1.1.1/dns-query"
	updated.CacheSize = 8192
	updated.AutoStart = true

	if err := store.UpdateSettings(ctx, updated); err != nil {
		t.Fatalf("failed to update settings: %v", err)
	}

	// 3. Read back and verify
	readBack, err := store.GetSettings(ctx)
	if err != nil {
		t.Fatalf("failed to read back settings: %v", err)
	}
	if readBack.UpstreamDoHURL != "https://1.1.1.1/dns-query" {
		t.Errorf("expected updated DoH URL, got %s", readBack.UpstreamDoHURL)
	}
	if readBack.CacheSize != 8192 {
		t.Errorf("expected cache size 8192, got %d", readBack.CacheSize)
	}
	if !readBack.AutoStart {
		t.Errorf("expected autoStart true")
	}

	// 4. Update with invalid DoH URL must fail and NOT alter DB
	invalidSettings := updated
	invalidSettings.UpstreamDoHURL = "http://insecure.example.com"
	if err := store.UpdateSettings(ctx, invalidSettings); err == nil {
		t.Errorf("expected error updating with invalid DoH URL, got nil")
	}

	// Verify DB was preserved
	check, _ := store.GetSettings(ctx)
	if check.UpstreamDoHURL != "https://1.1.1.1/dns-query" {
		t.Errorf("settings were mutated despite validation failure")
	}
}

func TestStorage_DNSRewrites_CRUD(t *testing.T) {
	store, ctx := setupTestStorage(t)

	rule1 := rewrites.Rule{
		Domain:  "nas.home.arpa",
		Type:    "A",
		Target:  "192.168.18.11",
		TTL:     60,
		Enabled: true,
	}

	// 1. Save rewrite
	if err := store.SaveRewrite(ctx, rule1); err != nil {
		t.Fatalf("failed to save rewrite: %v", err)
	}

	// 2. Get rewrites
	rules, err := store.GetRewrites(ctx)
	if err != nil {
		t.Fatalf("failed to get rewrites: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rewrite, got %d", len(rules))
	}
	if rules[0].Domain != "nas.home.arpa" || rules[0].Target != "192.168.18.11" {
		t.Errorf("unexpected rewrite rule: %+v", rules[0])
	}

	// 3. Update rewrite target and toggle
	rule1Updated := rule1
	rule1Updated.Target = "192.168.18.12"
	if err := store.SaveRewrite(ctx, rule1Updated); err != nil {
		t.Fatalf("failed to update rewrite: %v", err)
	}

	if err := store.ToggleRewrite(ctx, "nas.home.arpa", "A", false); err != nil {
		t.Fatalf("failed to toggle rewrite: %v", err)
	}

	rules, _ = store.GetRewrites(ctx)
	if len(rules) != 1 || rules[0].Target != "192.168.18.12" || rules[0].Enabled {
		t.Errorf("expected target 192.168.18.12 and enabled false, got: %+v", rules[0])
	}

	// 4. Delete rewrite
	if err := store.DeleteRewrite(ctx, "nas.home.arpa", "A"); err != nil {
		t.Fatalf("failed to delete rewrite: %v", err)
	}

	rules, _ = store.GetRewrites(ctx)
	if len(rules) != 0 {
		t.Errorf("expected 0 rewrites after deletion, got %d", len(rules))
	}
}

func TestStorage_CustomRules_CRUD(t *testing.T) {
	store, ctx := setupTestStorage(t)

	rule := models.CustomRule{
		ID:      "rule-100",
		Domain:  "tracker.example.com",
		Action:  "BLOCK",
		Enabled: true,
		Notes:   "Block tracking domain",
	}

	// 1. Save custom rule
	if err := store.SaveCustomRule(ctx, rule); err != nil {
		t.Fatalf("failed to save custom rule: %v", err)
	}

	// 2. Fetch all
	rules, err := store.GetCustomRules(ctx)
	if err != nil {
		t.Fatalf("failed to get custom rules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 custom rule, got %d", len(rules))
	}
	if rules[0].Domain != "tracker.example.com" || rules[0].Action != "BLOCK" {
		t.Errorf("unexpected custom rule: %+v", rules[0])
	}

	// 3. Update custom rule
	if err := store.UpdateCustomRule(ctx, "rule-100", "ALLOW", "Allow for testing", false); err != nil {
		t.Fatalf("failed to update custom rule: %v", err)
	}

	rules, _ = store.GetCustomRules(ctx)
	if rules[0].Action != "ALLOW" || rules[0].Enabled || rules[0].Notes != "Allow for testing" {
		t.Errorf("unexpected updated rule: %+v", rules[0])
	}

	// 4. Toggle
	if err := store.ToggleCustomRule(ctx, "rule-100", true); err != nil {
		t.Fatalf("failed to toggle rule: %v", err)
	}
	rules, _ = store.GetCustomRules(ctx)
	if !rules[0].Enabled {
		t.Errorf("expected rule enabled true after toggle")
	}

	// 5. Delete
	if err := store.DeleteCustomRule(ctx, "rule-100"); err != nil {
		t.Fatalf("failed to delete custom rule: %v", err)
	}
	rules, _ = store.GetCustomRules(ctx)
	if len(rules) != 0 {
		t.Errorf("expected 0 rules after deletion, got %d", len(rules))
	}
}
