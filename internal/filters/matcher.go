package filters

import (
	"sync/atomic"
)

const (
	ActionPass Action = "PASS"
)

// Rule represents a domain filtering directive from any source.
type Rule struct {
	Domain  string
	Action  Action
	Enabled bool
	Source  string
}

// MatchResult is the verdict and provenance of a matched domain query.
type MatchResult struct {
	Action        Action
	MatchedDomain string
	RuleSource    string
}

// Matcher performs thread-safe domain matching against loaded rules using an atomic trie.
type Matcher struct {
	index atomic.Pointer[trieIndex]
}

// NewMatcher creates an empty Matcher.
func NewMatcher() *Matcher {
	m := &Matcher{}
	empty := newTrieIndex()
	m.index.Store(empty)
	return m
}

// BuildFromRules builds a brand new trie index and atomically installs it.
// This guarantees that partially loaded or corrupt indexes are never exposed.
func (m *Matcher) BuildFromRules(rules []Rule) {
	newIdx := newTrieIndex()
	for _, r := range rules {
		newIdx.insert(r)
	}
	m.index.Store(newIdx)
}

// Match evaluates a domain against the active trie index.
func (m *Matcher) Match(domain string) MatchResult {
	idx := m.index.Load()
	if idx == nil {
		return MatchResult{Action: ActionPass}
	}
	return idx.match(domain)
}
