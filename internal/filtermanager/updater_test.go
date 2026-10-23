package filtermanager_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"blockads-desktop/internal/filtermanager"
	"blockads-desktop/internal/filters"
	"blockads-desktop/internal/models"
	"blockads-desktop/internal/storage"
)

type mockReloader struct {
	mu      sync.Mutex
	matcher *filters.Matcher
}

func (m *mockReloader) SetMatcher(matcher *filters.Matcher) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.matcher = matcher
}

func (m *mockReloader) Matcher() *filters.Matcher {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.matcher
}

func setupTestStore(t *testing.T) *storage.Storage {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test_updater.db")
	store, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test storage: %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}

func TestUpdater_FullDownloadAndActivation(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	reloader := &mockReloader{matcher: filters.NewMatcher()}

	hostsContent := "0.0.0.0 bad-ads.com\n127.0.0.1 tracker.analytics.net\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", "\"v1.0\"")
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2026 07:28:00 GMT")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(hostsContent))
	}))
	defer server.Close()

	// Create subscription
	sub := storage.FilterSubscription{
		ID:        "sub-test-1",
		Name:      "Test Blocklist",
		SourceURL: server.URL,
		Enabled:   true,
		Format:    "hosts",
	}
	if err := store.SaveFilterSubscription(ctx, sub); err != nil {
		t.Fatalf("failed to save subscription: %v", err)
	}

	dl := filtermanager.NewDownloader(filtermanager.DownloaderConfig{
		AllowPrivateNetworks: true,
	})
	updater := filtermanager.NewFilterUpdater(store, dl, reloader)

	// Run refresh
	res, err := updater.RefreshSubscription(ctx, sub.ID)
	if err != nil {
		t.Fatalf("RefreshSubscription failed: %v", err)
	}

	if res.Status != "updated" || res.StatusCode != 200 {
		t.Errorf("expected status 'updated' (200), got %s (%d)", res.Status, res.StatusCode)
	}
	if res.RuleCount != 2 {
		t.Errorf("expected 2 rules, got %d", res.RuleCount)
	}

	// Verify persistence
	updatedSub, err := store.GetFilterSubscription(ctx, sub.ID)
	if err != nil {
		t.Fatalf("failed to get updated subscription: %v", err)
	}
	if updatedSub.RuleCount != 2 {
		t.Errorf("expected stored rule count 2, got %d", updatedSub.RuleCount)
	}
	if updatedSub.LastSuccessAt == "" {
		t.Errorf("expected LastSuccessAt to be set")
	}
	if updatedSub.ETag != "\"v1.0\"" {
		t.Errorf("expected ETag '\"v1.0\"', got %s", updatedSub.ETag)
	}

	// Verify active matcher received the rules
	activeMatcher := reloader.Matcher()
	if activeMatcher == nil {
		t.Fatalf("expected non-nil active matcher")
	}
	match := activeMatcher.Match("bad-ads.com")
	if match.Action != filters.ActionBlock {
		t.Errorf("expected bad-ads.com to be BLOCKED, got %s", match.Action)
	}
	matchSub := activeMatcher.Match("sub.tracker.analytics.net")
	if matchSub.Action != filters.ActionBlock {
		t.Errorf("expected sub.tracker.analytics.net to be BLOCKED, got %s", matchSub.Action)
	}
}

func TestUpdater_ConditionalETagAndLastModified_304NotModified(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	reloader := &mockReloader{matcher: filters.NewMatcher()}

	requestCount := 0
	var receivedETag, receivedLastMod string
	hostsContent := "0.0.0.0 ads.sample.org\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		receivedETag = r.Header.Get("If-None-Match")
		receivedLastMod = r.Header.Get("If-Modified-Since")

		if requestCount == 1 {
			w.Header().Set("ETag", "\"etag-abc\"")
			w.Header().Set("Last-Modified", "Thu, 22 Oct 2026 12:00:00 GMT")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(hostsContent))
			return
		}

		// Second request sends validators
		if r.Header.Get("If-None-Match") == "\"etag-abc\"" {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(hostsContent))
	}))
	defer server.Close()

	sub := storage.FilterSubscription{
		ID:        "sub-conditional-1",
		Name:      "Conditional List",
		SourceURL: server.URL,
		Enabled:   true,
		Format:    "hosts",
	}
	if err := store.SaveFilterSubscription(ctx, sub); err != nil {
		t.Fatalf("failed to save sub: %v", err)
	}

	dl := filtermanager.NewDownloader(filtermanager.DownloaderConfig{
		AllowPrivateNetworks: true,
	})
	updater := filtermanager.NewFilterUpdater(store, dl, reloader)

	// First fetch: HTTP 200
	res1, err := updater.RefreshSubscription(ctx, sub.ID)
	if err != nil || res1.Status != "updated" {
		t.Fatalf("first fetch failed: res=%v, err=%v", res1, err)
	}

	// Second fetch: HTTP 304
	res2, err := updater.RefreshSubscription(ctx, sub.ID)
	if err != nil {
		t.Fatalf("second fetch failed: %v", err)
	}
	if res2.Status != "not_modified" || res2.StatusCode != 304 {
		t.Errorf("expected 'not_modified' (304), got %s (%d)", res2.Status, res2.StatusCode)
	}
	if receivedETag != "\"etag-abc\"" {
		t.Errorf("expected If-None-Match '\"etag-abc\"', got %q", receivedETag)
	}
	if receivedLastMod != "Thu, 22 Oct 2026 12:00:00 GMT" {
		t.Errorf("expected If-Modified-Since 'Thu, 22 Oct 2026 12:00:00 GMT', got %q", receivedLastMod)
	}

	// Verify matcher still works
	match := reloader.Matcher().Match("ads.sample.org")
	if match.Action != filters.ActionBlock {
		t.Errorf("expected ads.sample.org to remain blocked after 304")
	}
}

func TestUpdater_304WithoutValidSnapshot_RecoversUnconditionally(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	reloader := &mockReloader{matcher: filters.NewMatcher()}

	unconditionalReceived := false
	hostsContent := "0.0.0.0 recovered-domain.com\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inm := r.Header.Get("If-None-Match")
		if inm != "" {
			// Solicits 304 when ETag is present
			w.WriteHeader(http.StatusNotModified)
			return
		}
		// Unconditional request received
		unconditionalReceived = true
		w.Header().Set("ETag", "\"etag-recovered\"")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(hostsContent))
	}))
	defer server.Close()

	// Artificially save a subscription with ETag but WITHOUT any snapshot saved
	sub := storage.FilterSubscription{
		ID:              "sub-unsolicited",
		Name:            "Unsolicited 304 Sub",
		SourceURL:       server.URL,
		Enabled:         true,
		Format:          "hosts",
		ETag:            "\"etag-phantom\"",
		ContentChecksum: "phantom-checksum",
	}
	if err := store.SaveFilterSubscription(ctx, sub); err != nil {
		t.Fatalf("failed to save subscription: %v", err)
	}

	dl := filtermanager.NewDownloader(filtermanager.DownloaderConfig{
		AllowPrivateNetworks: true,
	})
	updater := filtermanager.NewFilterUpdater(store, dl, reloader)

	res, err := updater.RefreshSubscription(ctx, sub.ID)
	if err != nil {
		t.Fatalf("expected successful recovery after 304, got err: %v", err)
	}
	if !unconditionalReceived {
		t.Errorf("expected unconditional request to be triggered after 304 with no valid snapshot")
	}
	if res.Status != "updated" || res.StatusCode != 200 {
		t.Errorf("expected 'updated' (200), got %s (%d)", res.Status, res.StatusCode)
	}
	match := reloader.Matcher().Match("recovered-domain.com")
	if match.Action != filters.ActionBlock {
		t.Errorf("expected recovered-domain.com to be blocked")
	}
}

func TestUpdater_FailurePreservesWorkingState(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	reloader := &mockReloader{matcher: filters.NewMatcher()}

	// 1. Prime a working snapshot
	hostsContent := "0.0.0.0 preserved-domain.org\n"
	hasher := sha256.New()
	hasher.Write([]byte(hostsContent))
	chksum := hex.EncodeToString(hasher.Sum(nil))

	sub := storage.FilterSubscription{
		ID:              "sub-failing-1",
		Name:            "Failing Server List",
		SourceURL:       "http://127.0.0.1:9999/will-fail", // initially unreachable
		Enabled:         true,
		Format:          "hosts",
		ContentChecksum: chksum,
		RuleCount:       1,
	}
	_ = store.SaveFilterSubscription(ctx, sub)
	_, _ = store.SaveFilterSnapshot(ctx, sub.ID, hostsContent, 1, storage.SnapshotMeta{})
	// Load into matcher
	active, _ := storage.ReconcileActiveState(ctx, store)
	reloader.SetMatcher(active.Matcher)

	if match := reloader.Matcher().Match("preserved-domain.org"); match.Action != filters.ActionBlock {
		t.Fatalf("failed to prime active matcher")
	}

	// 2. Refresh against server returning 500 Internal Server Error
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	sub.SourceURL = failServer.URL
	_ = store.SaveFilterSubscription(ctx, sub)

	dl := filtermanager.NewDownloader(filtermanager.DownloaderConfig{
		AllowPrivateNetworks: true,
	})
	updater := filtermanager.NewFilterUpdater(store, dl, reloader)

	res, err := updater.RefreshSubscription(ctx, sub.ID)
	if err == nil {
		t.Errorf("expected error from 500 response, got nil")
	}
	if res.Status != "error" {
		t.Errorf("expected status 'error', got %s", res.Status)
	}

	// 3. Verify prior snapshot was NOT wiped
	snap, snapChecksum, err := store.GetFilterSnapshot(ctx, sub.ID)
	if err != nil || snap != hostsContent || snapChecksum != chksum {
		t.Errorf("stored snapshot was corrupted or erased after fetch error: %v", err)
	}

	// 4. Verify active matcher STILL blocks preserved-domain.org
	if match := reloader.Matcher().Match("preserved-domain.org"); match.Action != filters.ActionBlock {
		t.Errorf("active matcher lost working rules after failed refresh")
	}
}

func TestUpdater_HTMLResponseAndEmptyRulesRejection(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	reloader := &mockReloader{matcher: filters.NewMatcher()}

	tests := []struct {
		name        string
		body        string
		expectedErr error
	}{
		{
			name:        "html_doctype_page",
			body:        "<!DOCTYPE html>\n<html><head><title>Cloudflare Block</title></head><body>Error</body></html>",
			expectedErr: filtermanager.ErrHTMLResponse,
		},
		{
			name:        "empty_content",
			body:        "# Just comments\n# Nothing here\n",
			expectedErr: filtermanager.ErrEmptyFilterList,
		},
	}

	dl := filtermanager.NewDownloader(filtermanager.DownloaderConfig{
		AllowPrivateNetworks: true,
	})
	updater := filtermanager.NewFilterUpdater(store, dl, reloader)

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			subID := fmt.Sprintf("sub-reject-%d", i)
			sub := storage.FilterSubscription{
				ID:        subID,
				Name:      tc.name,
				SourceURL: server.URL,
				Enabled:   true,
				Format:    "auto",
			}
			_ = store.SaveFilterSubscription(ctx, sub)

			res, err := updater.RefreshSubscription(ctx, subID)
			if err == nil || res.Status != "error" {
				t.Errorf("expected rejection error %v, got res=%v, err=%v", tc.expectedErr, res, err)
			}

			// Confirm error recorded in DB
			storedSub, _ := store.GetFilterSubscription(ctx, subID)
			if storedSub.LastError == "" {
				t.Errorf("expected LastError to be recorded in DB")
			}
		})
	}
}

func TestUpdater_InflightDeduplication(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	reloader := &mockReloader{matcher: filters.NewMatcher()}

	started := make(chan struct{})
	gate := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-gate
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("0.0.0.0 delayed.com\n"))
	}))
	defer server.Close()

	sub := storage.FilterSubscription{
		ID:        "sub-inflight-1",
		Name:      "Inflight Sub",
		SourceURL: server.URL,
		Enabled:   true,
		Format:    "hosts",
	}
	_ = store.SaveFilterSubscription(ctx, sub)

	dl := filtermanager.NewDownloader(filtermanager.DownloaderConfig{
		AllowPrivateNetworks: true,
	})
	updater := filtermanager.NewFilterUpdater(store, dl, reloader)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = updater.RefreshSubscription(ctx, sub.ID)
	}()

	<-started // First request is running

	// Try concurrent request for same sub ID
	_, err2 := updater.RefreshSubscription(ctx, sub.ID)
	if err2 == nil || err2 != filtermanager.ErrAlreadyUpdating {
		t.Errorf("expected ErrAlreadyUpdating for concurrent request, got %v", err2)
	}

	close(gate)
	wg.Wait()
}

func TestUpdater_CustomRulePrecedenceAndDisablingProvenance(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	reloader := &mockReloader{matcher: filters.NewMatcher()}

	// 1. Sub1 blocks ad-domain.com and tracker-shared.com
	// 2. Sub2 blocks analytics.com and tracker-shared.com
	// 3. Custom rule ALLOWS ad-domain.com
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ad-domain.com\ntracker-shared.com\n"))
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("analytics.com\ntracker-shared.com\n"))
	}))
	defer server2.Close()

	sub1 := storage.FilterSubscription{
		ID:        "sub-src-1",
		Name:      "Source 1",
		SourceURL: server1.URL,
		Enabled:   true,
		Format:    "plain",
	}
	sub2 := storage.FilterSubscription{
		ID:        "sub-src-2",
		Name:      "Source 2",
		SourceURL: server2.URL,
		Enabled:   true,
		Format:    "plain",
	}
	_ = store.SaveFilterSubscription(ctx, sub1)
	_ = store.SaveFilterSubscription(ctx, sub2)

	// Add custom allow rule for ad-domain.com
	_ = store.SaveCustomRule(ctx, models.CustomRule{
		Domain:  "ad-domain.com",
		Action:  "ALLOW",
		Enabled: true,
	})

	dl := filtermanager.NewDownloader(filtermanager.DownloaderConfig{
		AllowPrivateNetworks: true,
	})
	updater := filtermanager.NewFilterUpdater(store, dl, reloader)

	// Fetch both
	if _, err := updater.RefreshSubscription(ctx, sub1.ID); err != nil {
		t.Fatalf("sub1 refresh failed: %v", err)
	}
	if _, err := updater.RefreshSubscription(ctx, sub2.ID); err != nil {
		t.Fatalf("sub2 refresh failed: %v", err)
	}

	matcher := reloader.Matcher()

	// Check custom rule precedence: ALLOW wins over subscription BLOCK
	if m := matcher.Match("ad-domain.com"); m.Action != filters.ActionAllow {
		t.Errorf("expected ad-domain.com to be ALLOW, got %s", m.Action)
	}

	// Check shared domain is BLOCKED
	if m := matcher.Match("tracker-shared.com"); m.Action != filters.ActionBlock {
		t.Errorf("expected tracker-shared.com to be BLOCK, got %s", m.Action)
	}

	// Now disable Sub1
	if err := store.ToggleFilterSubscription(ctx, sub1.ID, false); err != nil {
		t.Fatalf("failed to disable sub1: %v", err)
	}
	if err := updater.ReconcileMatcher(ctx); err != nil {
		t.Fatalf("reconcile matcher failed: %v", err)
	}

	matcherAfter := reloader.Matcher()
	// tracker-shared.com MUST STILL BE BLOCKED because Sub2 is still enabled!
	if m := matcherAfter.Match("tracker-shared.com"); m.Action != filters.ActionBlock {
		t.Errorf("tracker-shared.com was unblocked when Sub1 was disabled, but Sub2 still blocks it!")
	}
	// analytics.com is still blocked
	if m := matcherAfter.Match("analytics.com"); m.Action != filters.ActionBlock {
		t.Errorf("analytics.com should still be blocked")
	}
}

func TestUpdater_AdblockSubscriptionUpdate(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(t)
	reloader := &mockReloader{matcher: filters.NewMatcher()}

	adblockContent := `[Adblock Plus 2.0]
! Title: Test Adblock List
||ads-adblock.com^
||sub.tracker-adblock.org^
@@||safe-whitelisted.com^
##.cosmetic-rule
||path.rule.com/banner.gif^
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(adblockContent))
	}))
	defer server.Close()

	sub := storage.FilterSubscription{
		ID:        "sub-test-adblock",
		Name:      "Test Adblock",
		SourceURL: server.URL,
		Format:    "adblock",
		Enabled:   true,
	}
	if err := store.SaveFilterSubscription(ctx, sub); err != nil {
		t.Fatalf("failed to add subscription: %v", err)
	}

	dl := filtermanager.NewDownloader(filtermanager.DownloaderConfig{
		AllowPrivateNetworks: true,
	})
	updater := filtermanager.NewFilterUpdater(store, dl, reloader)

	res, err := updater.RefreshSubscription(ctx, sub.ID)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	if res.Status != "updated" {
		t.Errorf("expected status 'updated', got %q", res.Status)
	}
	if res.RuleCount != 3 {
		t.Errorf("expected 3 accepted rules, got %d", res.RuleCount)
	}

	// Verify in DB
	updatedSub, err := store.GetFilterSubscription(ctx, sub.ID)
	if err != nil {
		t.Fatalf("failed to fetch updated sub: %v", err)
	}
	if updatedSub.RuleCount != 3 {
		t.Errorf("expected DB rule_count 3, got %d", updatedSub.RuleCount)
	}
	if updatedSub.LastStatus != "ok" {
		t.Errorf("expected DB last_status 'ok', got %q", updatedSub.LastStatus)
	}
	if updatedSub.LastError != "" {
		t.Errorf("expected DB last_error empty, got %q", updatedSub.LastError)
	}

	// Verify matcher
	matcher := reloader.Matcher()
	if m := matcher.Match("ads-adblock.com"); m.Action != filters.ActionBlock {
		t.Errorf("expected ads-adblock.com to be ActionBlock, got %s", m.Action)
	}
	if m := matcher.Match("sub.tracker-adblock.org"); m.Action != filters.ActionBlock {
		t.Errorf("expected sub.tracker-adblock.org to be ActionBlock, got %s", m.Action)
	}
	if m := matcher.Match("safe-whitelisted.com"); m.Action != filters.ActionAllow {
		t.Errorf("expected safe-whitelisted.com to be ActionAllow, got %s", m.Action)
	}
}
