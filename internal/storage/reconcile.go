package storage

import (
	"context"
	"fmt"
	"strings"

	"blockads-desktop/internal/filters"
	"blockads-desktop/internal/rewrites"
	"blockads-desktop/internal/safesearch"
)

// ActiveState encapsulates the in-memory engines restored from persistent storage.
type ActiveState struct {
	Rewriter *rewrites.Rewriter
	Matcher  *filters.Matcher
}

// ReconcileActiveState restores the in-memory DNS rewrites and filter matcher from SQLite.
// It loads enabled rewrites, SafeSearch policies, custom rules, and accepted filter snapshots in deterministic order.
// If building a candidate matcher fails, the error is returned so the caller can preserve
// the currently active working matcher.
func ReconcileActiveState(ctx context.Context, store *Storage) (*ActiveState, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: storage is nil", ErrInvalidInput)
	}

	// 1. Rebuild DNS Rewrites & SafeSearch policies
	rewriter := rewrites.NewRewriter()
	persistedRewrites, err := store.GetRewrites(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load rewrites: %w", err)
	}

	var enabledRewrites []rewrites.Rule
	userRewrittenDomains := make(map[string]bool)
	for _, r := range persistedRewrites {
		if r.Enabled {
			enabledRewrites = append(enabledRewrites, r)
			userRewrittenDomains[r.CanonicalDomain()] = true
		}
	}

	// Load persistent SafeSearch policies and inject canonical CNAME rules
	ssPolicies, err := store.GetSafeSearchPolicies(ctx)
	if err == nil && len(ssPolicies) > 0 {
		var enabledProviders []string
		for pid, enabled := range ssPolicies {
			if enabled {
				enabledProviders = append(enabledProviders, pid)
			}
		}
		if len(enabledProviders) > 0 {
			reg := safesearch.DefaultRegistry()
			ssRules := reg.GenerateRewriteRules(enabledProviders)
			// User-configured explicit rewrites take precedence over SafeSearch defaults
			for _, ssr := range ssRules {
				if !userRewrittenDomains[ssr.CanonicalDomain()] {
					enabledRewrites = append(enabledRewrites, ssr)
				}
			}
		}
	}

	if len(enabledRewrites) > 0 {
		if err := rewriter.AddRules(enabledRewrites); err != nil {
			return nil, fmt.Errorf("failed to compile active rewrites: %w", err)
		}
	}

	// 2. Load Custom Rules
	var allRules []filters.Rule

	customRules, err := store.GetCustomRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load custom rules: %w", err)
	}

	for _, cr := range customRules {
		if !cr.Enabled {
			continue
		}
		action := filters.ActionBlock
		if strings.EqualFold(cr.Action, "ALLOW") {
			action = filters.ActionAllow
		}
		allRules = append(allRules, filters.Rule{
			Domain:  cr.Domain,
			Action:  action,
			Enabled: true,
			Source:  "custom",
		})
	}

	// 3. Load accepted filter snapshots in deterministic order (by subscription ID)
	subs, err := store.GetFilterSubscriptions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load filter subscriptions: %w", err)
	}

	for _, sub := range subs {
		if !sub.Enabled || strings.TrimSpace(sub.ContentChecksum) == "" {
			continue
		}

		content, _, err := store.GetFilterSnapshot(ctx, sub.ID)
		if err != nil {
			// Failed integrity or missing snapshot: do not crash, report warning/error
			continue
		}

		// Parse snapshot content
		parseRes, err := filters.ParseFilterList(strings.NewReader(content), filters.FormatAuto)
		if err != nil {
			continue
		}

		for _, pr := range parseRes.Rules {
			allRules = append(allRules, filters.Rule{
				Domain:  pr.Domain,
				Action:  pr.Action,
				Enabled: true,
				Source:  sub.Name,
			})
		}
	}

	// 4. Build candidate matcher atomically
	candidateMatcher := filters.NewMatcher()
	candidateMatcher.BuildFromRules(allRules)

	return &ActiveState{
		Rewriter: rewriter,
		Matcher:  candidateMatcher,
	}, nil
}
