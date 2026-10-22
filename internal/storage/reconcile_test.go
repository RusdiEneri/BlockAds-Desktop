package storage_test

import (
	"testing"

	"github.com/miekg/dns"

	"blockads-desktop/internal/filters"
	"blockads-desktop/internal/models"
	"blockads-desktop/internal/rewrites"
	"blockads-desktop/internal/storage"
)

func TestReconcileActiveState_RestoreAfterRestart(t *testing.T) {
	store, ctx := setupTestStorage(t)

	// 1. Populate rewrites
	err := store.SaveRewrite(ctx, rewrites.Rule{
		Domain:  "nas.home.arpa",
		Type:    "A",
		Target:  "192.168.18.11",
		TTL:     60,
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("failed to save rewrite: %v", err)
	}

	// 2. Populate custom allow rule
	err = store.SaveCustomRule(ctx, models.CustomRule{
		ID:      "cr-1",
		Domain:  "safe-exception.local",
		Action:  "ALLOW",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("failed to save custom rule: %v", err)
	}

	// 3. Populate filter subscription with valid snapshot
	err = store.SaveFilterSubscription(ctx, storage.FilterSubscription{
		ID:        "sub-test",
		Name:      "Test Filter",
		SourceURL: "https://example.com/hosts.txt",
		Enabled:   true,
	})
	if err != nil {
		t.Fatalf("failed to save subscription: %v", err)
	}

	snapshotContent := "0.0.0.0 bad-tracker.com\n0.0.0.0 safe-exception.local\n"
	_, err = store.SaveFilterSnapshot(ctx, "sub-test", snapshotContent, 2, storage.SnapshotMeta{})
	if err != nil {
		t.Fatalf("failed to save snapshot: %v", err)
	}

	// 4. Reconcile active state
	active, err := storage.ReconcileActiveState(ctx, store)
	if err != nil {
		t.Fatalf("failed to reconcile active state: %v", err)
	}
	if active.Rewriter == nil || active.Matcher == nil {
		t.Fatalf("expected non-nil Rewriter and Matcher in active state")
	}

	// 5. Verify Rewriter works
	req := new(dns.Msg)
	req.SetQuestion("nas.home.arpa.", dns.TypeA)
	rewriteResp := active.Rewriter.Rewrite(req)
	if rewriteResp == nil || len(rewriteResp.Answer) != 1 {
		t.Fatalf("expected rewrite match for nas.home.arpa")
	}

	// 6. Verify Matcher works (custom allow precedes filter block)
	matchBlocked := active.Matcher.Match("bad-tracker.com")
	if matchBlocked.Action != filters.ActionBlock {
		t.Errorf("expected bad-tracker.com to be BLOCKED, got %v", matchBlocked.Action)
	}

	matchAllowed := active.Matcher.Match("safe-exception.local")
	if matchAllowed.Action != filters.ActionAllow {
		t.Errorf("expected safe-exception.local to be ALLOWED due to custom rule precedence, got %v", matchAllowed.Action)
	}
}

func TestReconcileActiveState_SafeSearchAndPrecedence(t *testing.T) {
	store, ctx := setupTestStorage(t)

	// 1. Enable Google and Bing SafeSearch
	if err := store.SetSafeSearchPolicy(ctx, "google", true); err != nil {
		t.Fatalf("failed to set google safesearch: %v", err)
	}
	if err := store.SetSafeSearchPolicy(ctx, "bing", true); err != nil {
		t.Fatalf("failed to set bing safesearch: %v", err)
	}

	// 2. Add an explicit user rewrite for www.google.com -> 192.168.1.50
	err := store.SaveRewrite(ctx, rewrites.Rule{
		Domain:  "www.google.com",
		Type:    "A",
		Target:  "192.168.1.50",
		TTL:     60,
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("failed to save custom rewrite: %v", err)
	}

	// 3. Reconcile
	active, err := storage.ReconcileActiveState(ctx, store)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	// 4. Verify www.google.com returns user's A record (192.168.1.50), NOT CNAME
	reqGoogle := new(dns.Msg)
	reqGoogle.SetQuestion("www.google.com.", dns.TypeA)
	respGoogle := active.Rewriter.Rewrite(reqGoogle)
	if respGoogle == nil || len(respGoogle.Answer) != 1 {
		t.Fatalf("expected rewrite response for www.google.com")
	}
	aRec, ok := respGoogle.Answer[0].(*dns.A)
	if !ok || aRec.A.String() != "192.168.1.50" {
		t.Fatalf("expected user rewrite 192.168.1.50 to override SafeSearch, got %v", respGoogle.Answer[0])
	}

	// 5. Verify edgeservices.bing.com returns SafeSearch CNAME strict.bing.com.
	reqBing := new(dns.Msg)
	reqBing.SetQuestion("edgeservices.bing.com.", dns.TypeA)
	respBing := active.Rewriter.Rewrite(reqBing)
	if respBing == nil || len(respBing.Answer) != 1 {
		t.Fatalf("expected SafeSearch CNAME for edgeservices.bing.com")
	}
	cnameRec, ok := respBing.Answer[0].(*dns.CNAME)
	if !ok || cnameRec.Target != "strict.bing.com." {
		t.Fatalf("expected CNAME strict.bing.com., got %v", respBing.Answer[0])
	}
}
