package filters

import "strings"

// trieNode represents a single label in a reversed domain path.
type trieNode struct {
	children  map[string]*trieNode
	hasRule   bool
	action    Action
	domain    string
	source    string
	hasAllow  bool
	allowRule Rule
	hasBlock  bool
	blockRule Rule
}

func newTrieNode() *trieNode {
	return &trieNode{
		children: make(map[string]*trieNode),
	}
}

// splitLabelsReversed splits a domain into reversed labels:
// e.g. "ads.example.com" -> ["com", "example", "ads"]
func splitLabelsReversed(domain string) []string {
	clean := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(domain), "."))
	if clean == "" {
		return nil
	}
	parts := strings.Split(clean, ".")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return parts
}

type trieIndex struct {
	root *trieNode
}

func newTrieIndex() *trieIndex {
	return &trieIndex{
		root: newTrieNode(),
	}
}

func (idx *trieIndex) insert(rule Rule) {
	if !rule.Enabled {
		return
	}
	labels := splitLabelsReversed(rule.Domain)
	if len(labels) == 0 {
		return
	}

	curr := idx.root
	for _, label := range labels {
		next, exists := curr.children[label]
		if !exists {
			next = newTrieNode()
			curr.children[label] = next
		}
		curr = next
	}

	curr.hasRule = true
	if rule.Action == ActionAllow {
		curr.hasAllow = true
		curr.allowRule = rule
	} else if rule.Action == ActionBlock {
		curr.hasBlock = true
		curr.blockRule = rule
	}

	// Precedence tie-breaker: if allow and block rules have same specificity, ALLOW wins
	if curr.hasAllow {
		curr.action = ActionAllow
		curr.domain = curr.allowRule.Domain
		curr.source = curr.allowRule.Source
	} else if curr.hasBlock {
		curr.action = ActionBlock
		curr.domain = curr.blockRule.Domain
		curr.source = curr.blockRule.Source
	}
}

func (idx *trieIndex) match(domain string) MatchResult {
	labels := splitLabelsReversed(domain)
	if len(labels) == 0 {
		return MatchResult{Action: ActionPass}
	}

	curr := idx.root
	var lastMatch *trieNode

	for _, label := range labels {
		next, exists := curr.children[label]
		if !exists {
			break
		}
		curr = next
		if curr.hasRule {
			// Deeper node encountered: this represents a strictly more specific rule match
			lastMatch = curr
		}
	}

	if lastMatch == nil {
		return MatchResult{Action: ActionPass}
	}

	return MatchResult{
		Action:        lastMatch.action,
		MatchedDomain: lastMatch.domain,
		RuleSource:    lastMatch.source,
	}
}
