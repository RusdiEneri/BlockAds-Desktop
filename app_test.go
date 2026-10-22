package main

import (
	"testing"

	"blockads-desktop/internal/models"
	"blockads-desktop/internal/rewrites"
)

func TestAppInitialization(t *testing.T) {
	app := NewApp()
	if app == nil {
		t.Fatal("expected NewApp() to return non-nil App instance")
	}

	info := app.GetAppInfo()
	if info.Name != "BlockAds Desktop" {
		t.Errorf("expected app name 'BlockAds Desktop', got %q", info.Name)
	}
	if info.Version == "" {
		t.Error("expected non-empty version")
	}

	status := app.GetProtectionStatus()
	if !status.IsActive {
		t.Error("expected protection to be active by default")
	}
	if status.State != "ACTIVE" {
		t.Errorf("unexpected status state: %s", status.State)
	}

	// Test Toggle Protection Off
	pausedStatus, err := app.ToggleProtection(false)
	if err != nil {
		t.Fatalf("unexpected error toggling protection off: %v", err)
	}
	if pausedStatus.IsActive {
		t.Error("expected protection to be inactive when toggled off")
	}
	if pausedStatus.State != "PAUSED" {
		t.Errorf("expected PAUSED state, got %s", pausedStatus.State)
	}

	// Test Toggle Protection On
	resumedStatus, err := app.ToggleProtection(true)
	if err != nil {
		t.Fatalf("unexpected error toggling protection on: %v", err)
	}
	if !resumedStatus.IsActive || resumedStatus.State != "ACTIVE" {
		t.Errorf("expected active protection after resuming, got %+v", resumedStatus)
	}
}

func TestAppSimulateQuery(t *testing.T) {
	app := NewApp()
	entry, err := app.SimulateQuery("adservice.google.com", "A")
	if err != nil {
		t.Fatalf("failed to simulate query: %v", err)
	}
	if entry.Domain != "adservice.google.com" {
		t.Errorf("expected domain adservice.google.com, got %s", entry.Domain)
	}
	if entry.Status != "BLOCKED" {
		t.Errorf("expected adservice.google.com to be blocked, got %s", entry.Status)
	}
}

func TestAppQueryLogs(t *testing.T) {
	app := NewApp()
	logs := app.GetSampleQueryLogs()
	if len(logs) == 0 {
		t.Fatal("expected initial sample query logs, got empty slice")
	}

	hasBlocked := false
	hasAllowed := false
	for _, l := range logs {
		if l.Status == "BLOCKED" {
			hasBlocked = true
		}
		if l.Status == "ALLOWED" {
			hasAllowed = true
		}
	}

	if !hasBlocked || !hasAllowed {
		t.Errorf("expected sample logs to contain both BLOCKED and ALLOWED entries; hasBlocked=%v, hasAllowed=%v", hasBlocked, hasAllowed)
	}
}

func TestAppFilterListsToggle(t *testing.T) {
	app := NewApp()
	initialFilters := app.GetFilterLists()
	if len(initialFilters) == 0 {
		t.Fatal("expected filter lists")
	}

	targetID := initialFilters[0].ID
	initialState := initialFilters[0].Enabled

	updated := app.ToggleFilterList(targetID, !initialState)
	for _, f := range updated {
		if f.ID == targetID {
			if f.Enabled == initialState {
				t.Errorf("expected filter %s to toggle to %v", targetID, !initialState)
			}
			return
		}
	}

	t.Errorf("filter %s not found in updated list", targetID)
}

func TestAppCustomRulesCRUD(t *testing.T) {
	app := NewApp()

	// 1. Add valid rule
	newRule, err := app.AddCustomRule("test-ad-tracker.com", "BLOCK", "test note")
	if err != nil {
		t.Fatalf("failed to add valid custom rule: %v", err)
	}
	if newRule.Domain != "test-ad-tracker.com" || newRule.Action != "BLOCK" {
		t.Errorf("unexpected rule properties: %+v", newRule)
	}

	// 2. Reject duplicate rule
	_, dupErr := app.AddCustomRule("test-ad-tracker.com", "ALLOW", "duplicate test")
	if dupErr == nil {
		t.Error("expected duplicate rule to be rejected, but it succeeded")
	}

	// 3. Reject invalid domain
	_, invalidErr := app.AddCustomRule("https://invalid.domain", "BLOCK", "")
	if invalidErr == nil {
		t.Error("expected invalid domain to be rejected, but it succeeded")
	}

	// 4. Update rule
	if err := app.UpdateCustomRule(newRule.ID, "ALLOW", "updated note", false); err != nil {
		t.Fatalf("failed to update custom rule: %v", err)
	}
	rules := app.GetCustomRules()
	var updatedFound *models.CustomRule
	for _, r := range rules {
		if r.ID == newRule.ID {
			updatedFound = &r
			break
		}
	}
	if updatedFound == nil {
		t.Fatal("updated rule not found")
	}
	if updatedFound.Action != "ALLOW" || updatedFound.Notes != "updated note" || updatedFound.Enabled != false {
		t.Errorf("unexpected updated rule state: %+v", updatedFound)
	}

	// 5. Reject invalid action in update
	if err := app.UpdateCustomRule(newRule.ID, "INVALID_ACTION", "", true); err == nil {
		t.Error("expected invalid action in UpdateCustomRule to fail")
	}

	// 6. Update non-existent rule
	if err := app.UpdateCustomRule("ghost-id-999", "BLOCK", "", true); err == nil {
		t.Error("expected updating non-existent rule to fail")
	}

	// 7. Toggle rule enabled state
	if err := app.ToggleCustomRule(newRule.ID, true); err != nil {
		t.Fatalf("failed to toggle rule: %v", err)
	}
	rulesAfterToggle := app.GetCustomRules()
	for _, r := range rulesAfterToggle {
		if r.ID == newRule.ID && !r.Enabled {
			t.Errorf("expected rule to be enabled after toggle, got %v", r.Enabled)
		}
	}

	// 8. Delete rule
	delErr := app.DeleteCustomRule(newRule.ID)
	if delErr != nil {
		t.Fatalf("failed to delete custom rule: %v", delErr)
	}

	// 9. Delete non-existent rule
	delGhostErr := app.DeleteCustomRule("ghost-id-999")
	if delGhostErr == nil {
		t.Error("expected deleting nonexistent rule to fail")
	}
}

func TestAppUpdateUpstreamDoH(t *testing.T) {
	app := NewApp()

	validURL := "https://cloudflare-dns.com/dns-query"
	if err := app.UpdateUpstreamDoH(validURL); err != nil {
		t.Fatalf("expected valid upstream URL to succeed: %v", err)
	}

	settings := app.GetSettings()
	if settings.UpstreamDoHURL != validURL {
		t.Errorf("expected upstream DoH to be %q, got %q", validURL, settings.UpstreamDoHURL)
	}

	invalidURL := "http://insecure-dns.com"
	if err := app.UpdateUpstreamDoH(invalidURL); err == nil {
		t.Error("expected insecure HTTP URL to be rejected")
	}
}

func TestAppRunDiagnostics(t *testing.T) {
	app := NewApp()
	defer app.Close()
	diagnostics := app.RunDiagnostics()
	if len(diagnostics) < 4 {
		t.Fatalf("expected at least 4 diagnostic checks, got %d", len(diagnostics))
	}

	for _, d := range diagnostics {
		if d.Name == "" || d.Category == "" || d.Status == "" || d.Details == "" {
			t.Errorf("diagnostic check contains empty fields: %+v", d)
		}
	}
}

func TestAppDNSRewritesCRUD(t *testing.T) {
	app := NewApp()
	defer app.Close()

	// 1. Add rewrite rule
	rule := rewrites.Rule{
		Domain:  "service.homelab.arpa",
		Type:    "A",
		Target:  "192.168.1.100",
		TTL:     60,
		Enabled: true,
	}
	if err := app.SaveDNSRewrite(rule); err != nil {
		t.Fatalf("failed to save DNS rewrite: %v", err)
	}

	// 2. Fetch rewrites
	rules, err := app.GetDNSRewrites()
	if err != nil {
		t.Fatalf("failed to get rewrites: %v", err)
	}
	if len(rules) != 1 || rules[0].Domain != "service.homelab.arpa" {
		t.Fatalf("unexpected rewrites: %+v", rules)
	}

	// 3. Toggle rewrite
	if err := app.ToggleDNSRewrite("service.homelab.arpa", "A", false); err != nil {
		t.Fatalf("failed to toggle rewrite: %v", err)
	}
	rules, _ = app.GetDNSRewrites()
	if rules[0].Enabled {
		t.Errorf("expected rewrite rule to be disabled")
	}

	// 4. Delete rewrite
	if err := app.DeleteDNSRewrite("service.homelab.arpa", "A"); err != nil {
		t.Fatalf("failed to delete rewrite: %v", err)
	}
	rules, _ = app.GetDNSRewrites()
	if len(rules) != 0 {
		t.Errorf("expected 0 rewrites after deletion, got %d", len(rules))
	}
}

func TestAppGetQueryLogs(t *testing.T) {
	app := NewApp()
	defer app.Close()

	entries, total, err := app.GetQueryLogs("", "", 10, 0)
	if err != nil {
		t.Fatalf("failed to get query logs: %v", err)
	}
	if total == 0 || len(entries) == 0 {
		t.Fatalf("expected seeded sample query logs, got total=%d", total)
	}
}

func TestAppFilterSubscriptionManagement(t *testing.T) {
	app := NewApp()
	defer app.Close()

	// 1. Insecure HTTP must be rejected
	_, err := app.AddFilterSubscription("Insecure List", "http://insecure.example.com/hosts.txt", "hosts")
	if err == nil {
		t.Errorf("expected insecure HTTP URL to be rejected")
	}

	// 2. Loopback/Private IP must be rejected (SSRF prevention)
	_, err = app.AddFilterSubscription("SSRF List", "https://127.0.0.1/hosts.txt", "hosts")
	if err == nil {
		t.Errorf("expected loopback IP URL to be rejected")
	}

	// 3. Valid public HTTPS URL
	sub, err := app.AddFilterSubscription("Valid List", "https://example.com/hosts.txt", "hosts")
	if err != nil {
		t.Fatalf("failed to add valid subscription: %v", err)
	}
	if sub.Name != "Valid List" || sub.SourceURL != "https://example.com/hosts.txt" {
		t.Errorf("unexpected subscription fields: %+v", sub)
	}

	// 4. Verify listed in GetFilterLists
	lists := app.GetFilterLists()
	found := false
	for _, l := range lists {
		if l.ID == sub.ID {
			found = true
			if l.Format != "hosts" {
				t.Errorf("expected format 'hosts', got %s", l.Format)
			}
			break
		}
	}
	if !found {
		t.Errorf("added subscription %s not found in GetFilterLists", sub.ID)
	}

	// 5. Delete subscription
	if err := app.DeleteFilterSubscription(sub.ID); err != nil {
		t.Fatalf("failed to delete subscription: %v", err)
	}

	// 6. Verify removed
	listsAfter := app.GetFilterLists()
	for _, l := range listsAfter {
		if l.ID == sub.ID {
			t.Errorf("deleted subscription %s still present in GetFilterLists", sub.ID)
		}
	}
}

func TestAppSafeSearch(t *testing.T) {
	app := NewApp()

	// 1. Initial settings: verify providers listed
	settings := app.GetSafeSearchSettings()
	if len(settings) < 5 {
		t.Fatalf("expected at least 5 providers in SafeSearch settings, got %d", len(settings))
	}

	var googleSetting, yandexSetting *models.SafeSearchProviderSetting
	for i := range settings {
		if settings[i].ID == "google" {
			googleSetting = &settings[i]
		}
		if settings[i].ID == "yandex" {
			yandexSetting = &settings[i]
		}
	}

	if googleSetting == nil {
		t.Fatal("expected google setting in SafeSearch settings")
	}
	if yandexSetting == nil {
		t.Fatal("expected yandex setting in SafeSearch settings")
	}

	// Initially google is disabled
	if googleSetting.Enabled {
		t.Errorf("expected google to be disabled initially")
	}
	if googleSetting.Status != "DISABLED" {
		t.Errorf("expected google status DISABLED, got %s", googleSetting.Status)
	}

	// Yandex is unsupported
	if yandexSetting.Status != "UNSUPPORTED" {
		t.Errorf("expected yandex status UNSUPPORTED, got %s", yandexSetting.Status)
	}
	if yandexSetting.Warning == "" {
		t.Errorf("expected yandex warning diagnostic")
	}

	// 2. Reject enabling unsupported provider Yandex
	err := app.ToggleSafeSearch("yandex", true)
	if err == nil {
		t.Errorf("expected error enabling unsupported provider yandex, got nil")
	}

	// 3. Enable google
	if err := app.ToggleSafeSearch("google", true); err != nil {
		t.Fatalf("failed to enable google: %v", err)
	}

	updated := app.GetSafeSearchSettings()
	var updatedGoogle *models.SafeSearchProviderSetting
	for i := range updated {
		if updated[i].ID == "google" {
			updatedGoogle = &updated[i]
			break
		}
	}
	if updatedGoogle == nil || !updatedGoogle.Enabled {
		t.Fatalf("expected google to be enabled in updated settings")
	}
	if updatedGoogle.Status != "ACTIVE" {
		t.Errorf("expected google status to be ACTIVE, got %s", updatedGoogle.Status)
	}
	if updatedGoogle.Target != "forcesafesearch.google.com" {
		t.Errorf("expected target forcesafesearch.google.com, got %s", updatedGoogle.Target)
	}

	// 4. Disable google
	if err := app.ToggleSafeSearch("google", false); err != nil {
		t.Fatalf("failed to disable google: %v", err)
	}

	settingsAfterDisable := app.GetSafeSearchSettings()
	for _, s := range settingsAfterDisable {
		if s.ID == "google" {
			if s.Enabled || s.Status != "DISABLED" {
				t.Errorf("expected google to be DISABLED, got enabled=%v, status=%s", s.Enabled, s.Status)
			}
			break
		}
	}

	// 5. Reject unknown provider
	if err := app.ToggleSafeSearch("unknown-search", true); err == nil {
		t.Errorf("expected error for unknown provider, got nil")
	}
}

func TestApp_CatalogAndDuplicatePrevention(t *testing.T) {
	app := NewApp()

	// 1. Check catalog items
	catalog := app.GetCatalogFilters()
	if len(catalog) < 4 {
		t.Fatalf("expected at least 4 catalog items, got %d", len(catalog))
	}

	// AdAway is seeded by default in NewApp()
	var adaway *models.CatalogFilterItem
	for i := range catalog {
		if catalog[i].ID == "adaway" {
			adaway = &catalog[i]
			break
		}
	}
	if adaway == nil {
		t.Fatal("expected 'adaway' in catalog")
	}
	if !adaway.IsSubscribed {
		t.Errorf("expected AdAway to be marked as subscribed because of initial seed")
	}

	// 2. Reject duplicate manual addition
	_, err := app.AddFilterSubscription("Duplicate AdAway", "https://adaway.org/hosts.txt", "hosts")
	if err == nil {
		t.Fatal("expected duplicate subscription addition to fail, but it succeeded")
	}

	// 3. Preview URL detects already subscribed
	preview, err := app.PreviewFilterURL("https://adaway.org/hosts.txt#section")
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if !preview.IsAlreadySubscribed {
		t.Errorf("expected preview to detect existing subscription for AdAway")
	}
}

func TestApp_StartupIntegration(t *testing.T) {
	app := NewApp()

	status, err := app.GetStartupStatus()
	if err != nil {
		t.Fatalf("failed to get startup status: %v", err)
	}
	if !status.Supported {
		t.Log("startup not supported on this environment")
	}

	// Ensure setting to false works safely without modifying system
	updated, err := app.SetStartWithWindows(false)
	if err != nil {
		t.Fatalf("failed to set start with windows false: %v", err)
	}
	if updated.Enabled {
		t.Errorf("expected startup to be disabled")
	}
}
