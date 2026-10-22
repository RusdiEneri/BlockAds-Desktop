package filters_test

import (
	"sync"
	"testing"

	"blockads-desktop/internal/filters"
)

func TestMatcher_ExactAndSubdomainMatch(t *testing.T) {
	matcher := filters.NewMatcher()
	matcher.BuildFromRules([]filters.Rule{
		{
			Domain:  "example.com",
			Action:  filters.ActionBlock,
			Enabled: true,
			Source:  "test-blocklist",
		},
	})

	tests := []struct {
		query       string
		expectedAct filters.Action
		matchedDom  string
	}{
		{"example.com", filters.ActionBlock, "example.com"},
		{"sub.example.com", filters.ActionBlock, "example.com"},
		{"deep.sub.example.com", filters.ActionBlock, "example.com"},
		{"EXAMPLE.COM.", filters.ActionBlock, "example.com"},
		{"SUB.EXAMPLE.COM.", filters.ActionBlock, "example.com"},
		{"other.org", filters.ActionPass, ""},
		// Strict label boundary: must not match example.com.evil.org
		{"example.com.evil.org", filters.ActionPass, ""},
		{"notexample.com", filters.ActionPass, ""},
		{"example.com.net", filters.ActionPass, ""},
	}

	for _, tc := range tests {
		res := matcher.Match(tc.query)
		if res.Action != tc.expectedAct {
			t.Errorf("Match(%q) Action = %s, expected %s", tc.query, res.Action, tc.expectedAct)
		}
		if res.MatchedDomain != tc.matchedDom {
			t.Errorf("Match(%q) MatchedDomain = %q, expected %q", tc.query, res.MatchedDomain, tc.matchedDom)
		}
	}
}

func TestMatcher_PrecedenceSpecificWins(t *testing.T) {
	matcher := filters.NewMatcher()
	matcher.BuildFromRules([]filters.Rule{
		// Block parent domain
		{Domain: "example.com", Action: filters.ActionBlock, Enabled: true, Source: "block-parent"},
		// Allow specific subdomain
		{Domain: "safe.example.com", Action: filters.ActionAllow, Enabled: true, Source: "allow-sub"},
		// Allow another parent domain
		{Domain: "goodservice.org", Action: filters.ActionAllow, Enabled: true, Source: "allow-parent"},
		// Block specific subdomain of allowed parent
		{Domain: "ads.goodservice.org", Action: filters.ActionBlock, Enabled: true, Source: "block-sub"},
	})

	tests := []struct {
		query       string
		expectedAct filters.Action
		matchedDom  string
	}{
		// Block parent, but allow specific child
		{"example.com", filters.ActionBlock, "example.com"},
		{"ads.example.com", filters.ActionBlock, "example.com"},
		{"safe.example.com", filters.ActionAllow, "safe.example.com"},
		{"sub.safe.example.com", filters.ActionAllow, "safe.example.com"},

		// Allow parent, but block specific child
		{"goodservice.org", filters.ActionAllow, "goodservice.org"},
		{"app.goodservice.org", filters.ActionAllow, "goodservice.org"},
		{"ads.goodservice.org", filters.ActionBlock, "ads.goodservice.org"},
		{"tracker.ads.goodservice.org", filters.ActionBlock, "ads.goodservice.org"},
	}

	for _, tc := range tests {
		res := matcher.Match(tc.query)
		if res.Action != tc.expectedAct {
			t.Errorf("Match(%q) Action = %s, expected %s", tc.query, res.Action, tc.expectedAct)
		}
		if res.MatchedDomain != tc.matchedDom {
			t.Errorf("Match(%q) MatchedDomain = %q, expected %q", tc.query, res.MatchedDomain, tc.matchedDom)
		}
	}
}

func TestMatcher_PrecedenceTieBreakerAllowWins(t *testing.T) {
	matcher := filters.NewMatcher()
	// Two rules for exact same domain: one block, one allow
	matcher.BuildFromRules([]filters.Rule{
		{Domain: "conflict.example.com", Action: filters.ActionBlock, Enabled: true, Source: "block-list"},
		{Domain: "conflict.example.com", Action: filters.ActionAllow, Enabled: true, Source: "custom-allow"},
	})

	res := matcher.Match("conflict.example.com")
	if res.Action != filters.ActionAllow {
		t.Fatalf("expected ActionAllow on tie-breaker, got %s", res.Action)
	}
	if res.MatchedDomain != "conflict.example.com" {
		t.Fatalf("expected MatchedDomain conflict.example.com, got %s", res.MatchedDomain)
	}

	// Subdomain of conflicting domain should also inherit the winning allow verdict
	subRes := matcher.Match("sub.conflict.example.com")
	if subRes.Action != filters.ActionAllow {
		t.Fatalf("expected ActionAllow on subdomain of tie-breaker, got %s", subRes.Action)
	}
}

func TestMatcher_InactiveRulesIgnored(t *testing.T) {
	matcher := filters.NewMatcher()
	matcher.BuildFromRules([]filters.Rule{
		{Domain: "disabled.example.com", Action: filters.ActionBlock, Enabled: false, Source: "disabled-rule"},
	})

	res := matcher.Match("disabled.example.com")
	if res.Action != filters.ActionPass {
		t.Fatalf("expected ActionPass for disabled rule, got %s", res.Action)
	}
}

func TestMatcher_EmptyMatcher(t *testing.T) {
	matcher := filters.NewMatcher()
	res := matcher.Match("any.example.com")
	if res.Action != filters.ActionPass {
		t.Fatalf("expected ActionPass for empty matcher, got %s", res.Action)
	}
}

func TestMatcher_AtomicReload(t *testing.T) {
	matcher := filters.NewMatcher()
	matcher.BuildFromRules([]filters.Rule{
		{Domain: "initial.com", Action: filters.ActionBlock, Enabled: true, Source: "initial"},
	})

	var wg sync.WaitGroup
	// Concurrently read while rebuilding
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				_ = matcher.Match("initial.com")
				_ = matcher.Match("reloaded.com")
			}
		}()
	}

	// Rebuild with new rules
	matcher.BuildFromRules([]filters.Rule{
		{Domain: "reloaded.com", Action: filters.ActionBlock, Enabled: true, Source: "reloaded"},
	})

	wg.Wait()

	if matcher.Match("initial.com").Action != filters.ActionPass {
		t.Errorf("expected initial.com to be PASS after reload")
	}
	if matcher.Match("reloaded.com").Action != filters.ActionBlock {
		t.Errorf("expected reloaded.com to be BLOCK after reload")
	}
}

func BenchmarkMatcher_10kRules(b *testing.B) {
	matcher := filters.NewMatcher()
	rules := make([]filters.Rule, 10000)
	for i := 0; i < 10000; i++ {
		rules[i] = filters.Rule{
			Domain:  "tracker-" + string(rune('a'+(i%26))) + ".subdomain.example.org",
			Action:  filters.ActionBlock,
			Enabled: true,
			Source:  "benchmark",
		}
	}
	matcher.BuildFromRules(rules)

	query := "deep.test.tracker-c.subdomain.example.org"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res := matcher.Match(query)
		if res.Action != filters.ActionBlock {
			b.Fatal("unexpected action")
		}
	}
}
