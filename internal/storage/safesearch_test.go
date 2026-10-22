package storage_test

import (
	"testing"
	"time"
)

func TestStorage_SafeSearch_MigrationAndCRUD(t *testing.T) {
	store, ctx := setupTestStorage(t)

	// 1. Initial query: should be empty without error
	policies, err := store.GetSafeSearchPolicies(ctx)
	if err != nil {
		t.Fatalf("unexpected error getting initial safesearch policies: %v", err)
	}
	if len(policies) != 0 {
		t.Fatalf("expected 0 initial policies, got %d", len(policies))
	}

	// 2. Enable google
	if err := store.SetSafeSearchPolicy(ctx, "google", true); err != nil {
		t.Fatalf("failed to set google safesearch: %v", err)
	}

	// 3. Enable bing
	if err := store.SetSafeSearchPolicy(ctx, "bing", true); err != nil {
		t.Fatalf("failed to set bing safesearch: %v", err)
	}

	// 4. Verify policies map
	policies, err = store.GetSafeSearchPolicies(ctx)
	if err != nil {
		t.Fatalf("failed to get safesearch policies: %v", err)
	}
	if !policies["google"] {
		t.Errorf("expected google to be enabled")
	}
	if !policies["bing"] {
		t.Errorf("expected bing to be enabled")
	}
	if policies["duckduckgo"] {
		t.Errorf("duckduckgo should not be enabled yet")
	}

	// 5. Verify records
	records, err := store.GetSafeSearchPolicyRecords(ctx)
	if err != nil {
		t.Fatalf("failed to get policy records: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 policy records, got %d", len(records))
	}

	// 6. Disable google
	time.Sleep(10 * time.Millisecond)
	if err := store.SetSafeSearchPolicy(ctx, "google", false); err != nil {
		t.Fatalf("failed to disable google: %v", err)
	}

	policies, err = store.GetSafeSearchPolicies(ctx)
	if err != nil {
		t.Fatalf("failed to re-get policies: %v", err)
	}
	if policies["google"] {
		t.Errorf("expected google to be disabled")
	}
	if !policies["bing"] {
		t.Errorf("bing should still be enabled")
	}

	// 7. Input validation: empty provider ID rejected
	if err := store.SetSafeSearchPolicy(ctx, "", true); err == nil {
		t.Errorf("expected error when setting policy with empty provider ID")
	}
}
