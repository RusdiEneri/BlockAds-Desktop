package storage_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"blockads-desktop/internal/storage"
)

func TestStorage_FilterSubscriptions_Lifecycle(t *testing.T) {
	store, ctx := setupTestStorage(t)

	sub := storage.FilterSubscription{
		ID:          "filter-oisd",
		Name:        "OISD Basic",
		SourceURL:   "https://basic.oisd.nl/",
		Enabled:     true,
		Format:      "hosts",
		Description: "OISD blocklist",
		Category:    "Privacy",
	}

	// 1. Save subscription
	if err := store.SaveFilterSubscription(ctx, sub); err != nil {
		t.Fatalf("failed to save filter subscription: %v", err)
	}

	// 2. Fetch all
	subs, err := store.GetFilterSubscriptions(ctx)
	if err != nil {
		t.Fatalf("failed to get subscriptions: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(subs))
	}
	if subs[0].ID != "filter-oisd" || subs[0].Name != "OISD Basic" {
		t.Errorf("unexpected subscription: %+v", subs[0])
	}

	// 3. Toggle subscription
	if err := store.ToggleFilterSubscription(ctx, "filter-oisd", false); err != nil {
		t.Fatalf("failed to toggle subscription: %v", err)
	}
	subs, _ = store.GetFilterSubscriptions(ctx)
	if subs[0].Enabled {
		t.Errorf("expected subscription to be disabled")
	}

	// 4. Delete subscription
	if err := store.DeleteFilterSubscription(ctx, "filter-oisd"); err != nil {
		t.Fatalf("failed to delete subscription: %v", err)
	}
	subs, _ = store.GetFilterSubscriptions(ctx)
	if len(subs) != 0 {
		t.Errorf("expected 0 subscriptions, got %d", len(subs))
	}
}

func TestStorage_FilterSnapshot_SaveVerifyAndDeduplicate(t *testing.T) {
	store, ctx := setupTestStorage(t)

	sub := storage.FilterSubscription{
		ID:        "sub-adaway",
		Name:      "AdAway Hosts",
		SourceURL: "https://adaway.org/hosts.txt",
		Enabled:   true,
	}
	if err := store.SaveFilterSubscription(ctx, sub); err != nil {
		t.Fatalf("failed to save subscription: %v", err)
	}

	sampleContent := "127.0.0.1 adservice.google.com\n127.0.0.1 telemetry.dropbox.com\n"
	hasher := sha256.New()
	hasher.Write([]byte(sampleContent))
	expectedHash := hex.EncodeToString(hasher.Sum(nil))

	meta := storage.SnapshotMeta{
		ETag:         `"v1.0"`,
		LastModified: "Wed, 21 Oct 2026 07:28:00 GMT",
	}

	// 1. Save valid snapshot
	checksum, err := store.SaveFilterSnapshot(ctx, "sub-adaway", sampleContent, 2, meta)
	if err != nil {
		t.Fatalf("failed to save filter snapshot: %v", err)
	}
	if checksum != expectedHash {
		t.Errorf("expected checksum %s, got %s", expectedHash, checksum)
	}

	// 2. Retrieve and verify content integrity
	content, retrievedChecksum, err := store.GetFilterSnapshot(ctx, "sub-adaway")
	if err != nil {
		t.Fatalf("failed to get filter snapshot: %v", err)
	}
	if content != sampleContent {
		t.Errorf("content mismatch")
	}
	if retrievedChecksum != expectedHash {
		t.Errorf("retrieved checksum mismatch")
	}

	// 3. Save identical content again (deduplication check)
	checksum2, err := store.SaveFilterSnapshot(ctx, "sub-adaway", sampleContent, 2, meta)
	if err != nil {
		t.Fatalf("failed to save duplicate snapshot: %v", err)
	}
	if checksum2 != expectedHash {
		t.Errorf("duplicate save should return same checksum")
	}

	// 4. Overwrite subscription with new content
	newContent := "127.0.0.1 tracking.vendor.com\n"
	newChecksum, err := store.SaveFilterSnapshot(ctx, "sub-adaway", newContent, 1, storage.SnapshotMeta{ETag: `"v2.0"`})
	if err != nil {
		t.Fatalf("failed to save updated snapshot: %v", err)
	}
	if newChecksum == expectedHash {
		t.Errorf("expected different checksum for new content")
	}

	// Verify active content is newContent
	activeContent, activeChecksum, err := store.GetFilterSnapshot(ctx, "sub-adaway")
	if err != nil {
		t.Fatalf("failed to get active snapshot: %v", err)
	}
	if activeContent != newContent || activeChecksum != newChecksum {
		t.Errorf("active snapshot was not updated")
	}
}

func TestStorage_FilterSnapshot_SizeLimitAndCorruptionDetection(t *testing.T) {
	store, ctx := setupTestStorage(t)

	sub := storage.FilterSubscription{
		ID:        "sub-limit",
		Name:      "Size Limit Test",
		SourceURL: "https://example.com/hosts",
		Enabled:   true,
	}
	if err := store.SaveFilterSubscription(ctx, sub); err != nil {
		t.Fatalf("failed to save subscription: %v", err)
	}

	// 1. Exceed size limit (> 25MB)
	oversized := strings.Repeat("A", storage.MaxSnapshotBytes+10)
	_, err := store.SaveFilterSnapshot(ctx, "sub-limit", oversized, 0, storage.SnapshotMeta{})
	if err == nil {
		t.Errorf("expected error saving oversized snapshot, got nil")
	}

	// 2. Corrupt snapshot detection: artificially tamper with checksum in database
	validContent := "127.0.0.1 normal-domain.local\n"
	_, err = store.SaveFilterSnapshot(ctx, "sub-limit", validContent, 1, storage.SnapshotMeta{})
	if err != nil {
		t.Fatalf("failed to save valid snapshot: %v", err)
	}

	// Tamper with the content directly in the database
	_, err = store.DB().ExecContext(ctx, "UPDATE filter_snapshots SET content = 'tampered content' WHERE subscription_id = 'sub-limit';")
	if err != nil {
		t.Fatalf("failed to tamper with snapshot: %v", err)
	}

	// Retrieving snapshot must detect checksum mismatch
	_, _, err = store.GetFilterSnapshot(ctx, "sub-limit")
	if !errors.Is(err, storage.ErrCorruptSnapshot) {
		t.Errorf("expected ErrCorruptSnapshot, got %v", err)
	}
}

func TestStorage_FilterSnapshot_RecordFilterFetchNotModified(t *testing.T) {
	store, ctx := setupTestStorage(t)

	sub := storage.FilterSubscription{
		ID:        "sub-304",
		Name:      "304 Test",
		SourceURL: "https://example.com/hosts",
		Enabled:   true,
	}
	if err := store.SaveFilterSubscription(ctx, sub); err != nil {
		t.Fatalf("failed to save subscription: %v", err)
	}

	content := "127.0.0.1 test304.local\n"
	_, err := store.SaveFilterSnapshot(ctx, "sub-304", content, 1, storage.SnapshotMeta{
		ETag: `"v1"`,
	})
	if err != nil {
		t.Fatalf("failed to save initial snapshot: %v", err)
	}

	// Update metadata via RecordFilterFetchNotModified with new ETag/LastModified
	newMeta := storage.SnapshotMeta{
		ETag:         `"v1-notmodified"`,
		LastModified: "Thu, 22 Oct 2026 12:00:00 GMT",
	}
	if err := store.RecordFilterFetchNotModified(ctx, "sub-304", newMeta); err != nil {
		t.Fatalf("failed to record 304 not modified: %v", err)
	}

	// Read subscription back
	subUpdated, err := store.GetFilterSubscription(ctx, "sub-304")
	if err != nil {
		t.Fatalf("failed to get subscription: %v", err)
	}
	if subUpdated.ETag != `"v1-notmodified"` || subUpdated.LastModified != "Thu, 22 Oct 2026 12:00:00 GMT" {
		t.Errorf("expected updated metadata, got etag=%s, lastModified=%s", subUpdated.ETag, subUpdated.LastModified)
	}
	if subUpdated.LastStatus != "ok" {
		t.Errorf("expected status ok, got %s", subUpdated.LastStatus)
	}

	// Verify snapshot content remained completely intact
	retrievedContent, _, err := store.GetFilterSnapshot(ctx, "sub-304")
	if err != nil {
		t.Fatalf("failed to get snapshot after 304: %v", err)
	}
	if retrievedContent != content {
		t.Errorf("snapshot content was altered unexpectedly")
	}
}

func TestStorage_FilterSubscription_DuplicatePrevention(t *testing.T) {
	store, ctx := setupTestStorage(t)

	sub1 := storage.FilterSubscription{
		ID:        "sub-adaway-1",
		Name:      "AdAway Main",
		SourceURL: "https://adaway.org/hosts.txt",
		Enabled:   true,
	}
	if err := store.SaveFilterSubscription(ctx, sub1); err != nil {
		t.Fatalf("failed to save first subscription: %v", err)
	}

	// 1. Same URL with different scheme/casing/fragment should conflict
	sub2 := storage.FilterSubscription{
		ID:        "sub-adaway-2",
		Name:      "AdAway Duplicate",
		SourceURL: "HTTPS://AdAway.ORG:443/hosts.txt#fragment",
		Enabled:   true,
	}
	err := store.SaveFilterSubscription(ctx, sub2)
	if err == nil {
		t.Fatalf("expected duplicate conflict error, got nil")
	}
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("expected ErrConflict, got: %v", err)
	}

	// 2. Fetch by URL should find existing subscription
	found, err := store.GetFilterSubscriptionByURL(ctx, "https://adaway.org/hosts.txt#something")
	if err != nil {
		t.Fatalf("expected to find subscription by URL, got err: %v", err)
	}
	if found.ID != "sub-adaway-1" {
		t.Errorf("expected found subscription ID 'sub-adaway-1', got %s", found.ID)
	}

	// 3. Updating the SAME subscription should succeed
	sub1.Description = "Updated description"
	if err := store.SaveFilterSubscription(ctx, sub1); err != nil {
		t.Fatalf("updating same subscription should succeed, got: %v", err)
	}
}
