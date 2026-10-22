package safesearch

import (
	"strings"
	"sync"

	"blockads-desktop/internal/rewrites"
)

// Registry manages known SafeSearch provider definitions and compiles runtime rewrite rules.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	order     []string
}

// NewRegistry constructs a new Registry from a list of providers.
func NewRegistry(providers []Provider) *Registry {
	r := &Registry{
		providers: make(map[string]Provider, len(providers)),
		order:     make([]string, 0, len(providers)),
	}
	for _, p := range providers {
		r.providers[p.ID] = p
		r.order = append(r.order, p.ID)
	}
	return r
}

// DefaultRegistry returns the production Registry configured with verified provider policies.
func DefaultRegistry() *Registry {
	return NewRegistry([]Provider{
		{
			ID:   "google",
			Name: "Google Search",
			Domains: []string{
				"www.google.com",
				"www.google.co.uk",
				"www.google.ca",
				"www.google.de",
				"www.google.fr",
				"www.google.co.jp",
				"www.google.com.au",
				"www.google.com.br",
				"www.google.co.id",
				"www.google.co.in",
				"www.google.es",
				"www.google.it",
				"www.google.nl",
				"www.google.pl",
			},
			Target: "forcesafesearch.google.com",
			DocURL: "https://support.google.com/websearch/answer/186669",
			Status: StatusSupported,
		},
		{
			ID:   "bing",
			Name: "Microsoft Bing",
			Domains: []string{
				"www.bing.com",
				"edgeservices.bing.com", // Microsoft Edge sidebar scenario
			},
			Target: "strict.bing.com",
			DocURL: "https://support.microsoft.com/en-us/bing/blocking-explicit-content-with-safesearch",
			Status: StatusSupported,
		},
		{
			ID:   "duckduckgo",
			Name: "DuckDuckGo",
			Domains: []string{
				"duckduckgo.com",
				"www.duckduckgo.com",
			},
			Target: "safe.duckduckgo.com",
			DocURL: "https://duckduckgo.com/duckduckgo-help-pages/features/safe-search",
			Status: StatusSupported,
		},
		{
			ID:   "youtube",
			Name: "YouTube",
			Domains: []string{
				"www.youtube.com",
				"m.youtube.com",
				"youtubei.googleapis.com",
				"youtube.googleapis.com",
				"www.youtube-nocookie.com",
			},
			Target: "restrict.youtube.com",
			DocURL: "https://support.google.com/a/answer/6214622",
			Status: StatusSupported,
		},
		{
			ID:   "yandex",
			Name: "Yandex",
			Domains: []string{
				"yandex.ru",
				"www.yandex.ru",
				"yandex.com",
				"www.yandex.com",
			},
			Target:  "",
			DocURL:  "https://yandex.com/support/search/settings/family-search.html",
			Status:  StatusUnsupported,
			Warning: "Yandex Family Search requires routing through dedicated upstream DNS resolvers (77.88.8.7 / 77.88.8.3) rather than standard CNAME host rewrites. CNAME rewriting to family.yandex.ru is not officially supported by Yandex.",
		},
	})
}

// GetProviders returns all registered providers in deterministic order.
func (r *Registry) GetProviders() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Provider, 0, len(r.order))
	for _, id := range r.order {
		if p, ok := r.providers[id]; ok {
			result = append(result, p)
		}
	}
	return result
}

// GetProvider retrieves a single provider by stable ID.
func (r *Registry) GetProvider(id string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[id]
	return p, ok
}

// GenerateRewriteRules produces DNS CNAME rewrite rules for enabled, supported providers.
func (r *Registry) GenerateRewriteRules(enabledProviderIDs []string) []rewrites.Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	enabledMap := make(map[string]bool, len(enabledProviderIDs))
	for _, id := range enabledProviderIDs {
		enabledMap[strings.ToLower(strings.TrimSpace(id))] = true
	}

	var rules []rewrites.Rule
	for _, id := range r.order {
		if !enabledMap[id] {
			continue
		}
		p, exists := r.providers[id]
		if !exists || p.Status != StatusSupported || strings.TrimSpace(p.Target) == "" {
			continue
		}

		for _, domain := range p.Domains {
			cleanDomain := strings.ToLower(strings.TrimSpace(domain))
			cleanDomain = strings.TrimSuffix(cleanDomain, ".")
			cleanTarget := strings.ToLower(strings.TrimSpace(p.Target))
			cleanTarget = strings.TrimSuffix(cleanTarget, ".")

			if cleanDomain == "" || cleanTarget == "" {
				continue
			}

			rules = append(rules, rewrites.Rule{
				Domain:  cleanDomain,
				Type:    "CNAME",
				Target:  cleanTarget,
				TTL:     60,
				Enabled: true,
			})
		}
	}

	return rules
}
