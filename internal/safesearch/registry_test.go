package safesearch_test

import (
	"testing"

	"blockads-desktop/internal/safesearch"
)

func TestDefaultRegistry_Providers(t *testing.T) {
	reg := safesearch.DefaultRegistry()
	providers := reg.GetProviders()

	if len(providers) < 5 {
		t.Fatalf("expected at least 5 default providers, got %d", len(providers))
	}

	expected := map[string]struct {
		target   string
		status   string
		hasEdges bool
	}{
		"google":     {target: "forcesafesearch.google.com", status: safesearch.StatusSupported},
		"bing":       {target: "strict.bing.com", status: safesearch.StatusSupported, hasEdges: true},
		"duckduckgo": {target: "safe.duckduckgo.com", status: safesearch.StatusSupported},
		"youtube":    {target: "restrict.youtube.com", status: safesearch.StatusSupported},
		"yandex":     {target: "", status: safesearch.StatusUnsupported},
	}

	for id, exp := range expected {
		p, ok := reg.GetProvider(id)
		if !ok {
			t.Errorf("expected provider %q to exist in registry", id)
			continue
		}
		if p.Target != exp.target {
			t.Errorf("provider %q target mismatch: expected %q, got %q", id, exp.target, p.Target)
		}
		if p.Status != exp.status {
			t.Errorf("provider %q status mismatch: expected %q, got %q", id, exp.status, p.Status)
		}
		if len(p.Domains) == 0 {
			t.Errorf("provider %q has empty domain list", id)
		}
		if p.DocURL == "" {
			t.Errorf("provider %q missing DocURL", id)
		}
		if exp.hasEdges {
			foundEdge := false
			for _, d := range p.Domains {
				if d == "edgeservices.bing.com" {
					foundEdge = true
					break
				}
			}
			if !foundEdge {
				t.Errorf("provider %q missing edgeservices.bing.com in domain coverage", id)
			}
		}
		if exp.status == safesearch.StatusUnsupported && p.Warning == "" {
			t.Errorf("unsupported provider %q must have an explanatory diagnostic warning", id)
		}
	}
}

func TestRegistry_GenerateRewriteRules(t *testing.T) {
	reg := safesearch.DefaultRegistry()

	// 1. Enabling Google and Bing
	rules := reg.GenerateRewriteRules([]string{"google", "bing"})
	if len(rules) == 0 {
		t.Fatalf("expected generated rewrite rules for google and bing")
	}

	googleDomainFound := false
	bingEdgeFound := false
	for _, r := range rules {
		if r.Type != "CNAME" {
			t.Errorf("expected CNAME rule, got %s", r.Type)
		}
		if r.TTL != 60 {
			t.Errorf("expected TTL 60, got %d", r.TTL)
		}
		if !r.Enabled {
			t.Errorf("expected rule to be enabled")
		}
		if r.Domain == "www.google.com" && r.Target == "forcesafesearch.google.com" {
			googleDomainFound = true
		}
		if r.Domain == "edgeservices.bing.com" && r.Target == "strict.bing.com" {
			bingEdgeFound = true
		}
	}

	if !googleDomainFound {
		t.Errorf("expected rewrite rule for www.google.com -> forcesafesearch.google.com")
	}
	if !bingEdgeFound {
		t.Errorf("expected rewrite rule for edgeservices.bing.com -> strict.bing.com")
	}

	// 2. Requesting an unsupported provider (yandex) must NOT generate rules
	yandexRules := reg.GenerateRewriteRules([]string{"yandex"})
	if len(yandexRules) != 0 {
		t.Errorf("unsupported provider yandex must not generate rewrite rules, got %d rules", len(yandexRules))
	}

	// 3. Requesting an unknown provider must be ignored safely
	unknownRules := reg.GenerateRewriteRules([]string{"nonexistent-engine"})
	if len(unknownRules) != 0 {
		t.Errorf("unknown provider must produce 0 rules, got %d", len(unknownRules))
	}
}
