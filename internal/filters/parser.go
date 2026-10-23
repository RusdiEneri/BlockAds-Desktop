package filters

import (
	"bufio"
	"io"
	"net"
	"regexp"
	"strings"
)

// Action defines the filtering verdict for a matched domain.
type Action string

const (
	ActionBlock Action = "BLOCK"
	ActionAllow Action = "ALLOW"
)

// Format specifies the input syntax expected by the filter parser.
type Format string

const (
	FormatAuto    Format = "auto"
	FormatHosts   Format = "hosts"
	FormatPlain   Format = "plain"
	FormatAdblock Format = "adblock"
)

// ParsedRule represents a single parsed filtering directive.
type ParsedRule struct {
	Domain  string
	Action  Action
	Source  string
	Line    int
	Enabled bool
}

// ParseWarning describes a malformed or skipped line with line-number context.
type ParseWarning struct {
	Line    int
	Content string
	Reason  string
}

// ParseStats records counting metrics for the parse operation.
type ParseStats struct {
	TotalLines     int
	AcceptedCount  int
	DuplicateCount int
	SkippedCount   int
	WarningCount   int
}

// ParseResult is the complete output from parsing a filter list.
type ParseResult struct {
	Rules    []ParsedRule
	Warnings []ParseWarning
	Stats    ParseStats
}

var (
	validLabelRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
)

// isIgnoredHostname returns true for localhost, loopbacks, broadcast, and IP literals.
func isIgnoredHostname(domain string) bool {
	switch domain {
	case "localhost", "localhost.localdomain", "local", "broadcasthost",
		"ip6-localhost", "ip6-loopback", "0.0.0.0", "127.0.0.1", "::1",
		"fe80::1%lo0", "fe80::1":
		return true
	}
	// Skip raw IP addresses appearing as hostnames
	if net.ParseIP(domain) != nil {
		return true
	}
	return false
}

// validateDomainSyntax validates domain syntax and returns an explicit warning reason if invalid.
func validateDomainSyntax(raw string) (string, string) {
	d := strings.TrimSpace(raw)
	if d == "" {
		return "", "empty domain"
	}

	if strings.Contains(d, "://") || strings.Contains(d, "/") {
		return "", "contains URL scheme or path"
	}
	if strings.Contains(d, "##") || strings.Contains(d, "#?#") || strings.Contains(d, "#@#") || strings.Contains(d, "$$") {
		return "", "unsupported cosmetic filter syntax"
	}
	if strings.HasPrefix(d, "||") || strings.Contains(d, "^") || strings.HasPrefix(d, "@@") || strings.Contains(d, "$") {
		return "", "unsupported adblock rule syntax"
	}
	if strings.Contains(d, "*") {
		return "", "wildcard domains not supported"
	}
	if strings.Contains(d, " ") || strings.Contains(d, "\t") {
		return "", "contains spaces or illegal whitespace"
	}
	if strings.Contains(d, "@") || strings.Contains(d, ":") || strings.Contains(d, ";") {
		return "", "contains illegal characters"
	}

	// Normalize trailing dot and case
	d = strings.ToLower(strings.TrimSuffix(d, "."))
	if d == "" {
		return "", "root dot only"
	}

	if len(d) > 253 {
		return "", "domain exceeds 253 characters"
	}

	labels := strings.Split(d, ".")
	if len(labels) == 1 && labels[0] != "localhost" {
		// Single-label domain without dot (e.g. invalid host or TLD-only)
		return "", "single-label domain without TLD"
	}

	for _, label := range labels {
		if len(label) == 0 {
			return "", "empty label in domain"
		}
		if len(label) > 63 {
			return "", "domain label exceeds 63 characters"
		}
		if !validLabelRegex.MatchString(label) {
			return "", "invalid characters in domain label"
		}
	}

	return d, ""
}

// parseAdblockRule parses an Adblock-style rule (e.g. ||domain.com^ or @@||domain.com^).
// It returns the domain, action, whether it should be skipped (e.g. cosmetic, URL path, comments),
// and any warning reason.
func parseAdblockRule(raw string) (domain string, action Action, skipped bool, reason string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", ActionPass, true, ""
	}

	// Non-DNS adblock rules (element hiding / cosmetic filters)
	if strings.HasPrefix(s, "##") || strings.HasPrefix(s, "#?#") || strings.HasPrefix(s, "#@#") ||
		strings.Contains(s, "##") || strings.Contains(s, "#?#") || strings.Contains(s, "#@#") ||
		strings.Contains(s, "$$") {
		return "", ActionPass, true, ""
	}

	// Determine action
	action = ActionBlock
	if strings.HasPrefix(s, "@@||") {
		action = ActionAllow
		s = s[4:]
	} else if strings.HasPrefix(s, "@@") {
		action = ActionAllow
		s = s[2:]
	} else if strings.HasPrefix(s, "||") {
		action = ActionBlock
		s = s[2:]
	}

	// Handle modifiers after $
	if idx := strings.Index(s, "$"); idx != -1 {
		modifier := s[idx+1:]
		if strings.Contains(modifier, "badfilter") {
			return "", ActionPass, true, ""
		}
		s = s[:idx]
	}

	// Handle separator ^
	if idx := strings.Index(s, "^"); idx != -1 {
		remainder := s[idx+1:]
		if len(remainder) > 0 {
			// URL path rule, e.g. ||example.com^/path
			return "", ActionPass, true, ""
		}
		s = s[:idx]
	}

	// If contains URL path or wildcard, DNS cannot filter path or wildcards
	if strings.Contains(s, "/") || strings.Contains(s, "*") || strings.Contains(s, "://") {
		return "", ActionPass, true, ""
	}

	cleaned := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(s), "."))
	if cleaned == "" || isIgnoredHostname(cleaned) {
		return "", ActionPass, true, ""
	}

	validDomain, warningReason := validateDomainSyntax(cleaned)
	if warningReason != "" {
		return "", ActionPass, false, warningReason
	}

	return validDomain, action, false, ""
}

// ParseFilterList parses an io.Reader according to the requested or auto-detected format.
func ParseFilterList(r io.Reader, format Format) (*ParseResult, error) {
	scanner := bufio.NewScanner(r)
	result := &ParseResult{
		Rules:    make([]ParsedRule, 0),
		Warnings: make([]ParseWarning, 0),
	}

	seen := make(map[string]struct{})
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		result.Stats.TotalLines++

		rawLine := scanner.Text()
		trimmed := strings.TrimSpace(rawLine)

		// Check for adblock cosmetic rules starting with ##, #?#, #@# before standard comment check
		if strings.HasPrefix(trimmed, "##") || strings.HasPrefix(trimmed, "#?#") || strings.HasPrefix(trimmed, "#@#") {
			if format == FormatPlain {
				result.Warnings = append(result.Warnings, ParseWarning{
					Line:    lineNum,
					Content: rawLine,
					Reason:  "unsupported cosmetic filter syntax",
				})
				result.Stats.WarningCount++
			} else {
				result.Stats.SkippedCount++
			}
			continue
		}

		// Ignore blank lines, comment lines, and adblock section headers
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "!") || strings.HasPrefix(trimmed, "[") {
			result.Stats.SkippedCount++
			continue
		}

		// Strip inline comments if prefixed by space and # or !
		if idx := strings.IndexAny(trimmed, "#!"); idx != -1 {
			// Ensure it's preceded by whitespace or at start
			if idx > 0 && (trimmed[idx-1] == ' ' || trimmed[idx-1] == '\t') {
				trimmed = strings.TrimSpace(trimmed[:idx])
			}
		}

		if trimmed == "" {
			result.Stats.SkippedCount++
			continue
		}

		// Determine line format
		activeFormat := format
		fields := strings.Fields(trimmed)
		if len(fields) == 0 {
			result.Stats.SkippedCount++
			continue
		}

		if activeFormat == FormatAuto {
			// If first token is an IP address, treat as hosts line
			first := fields[0]
			if net.ParseIP(first) != nil || first == "0.0.0.0" || first == "127.0.0.1" || first == "::1" {
				activeFormat = FormatHosts
			} else if strings.HasPrefix(trimmed, "||") || strings.HasPrefix(trimmed, "@@||") || strings.HasPrefix(trimmed, "@@") || strings.Contains(trimmed, "^") {
				activeFormat = FormatAdblock
			} else {
				activeFormat = FormatPlain
			}
		}

		if activeFormat == FormatHosts {
			// Format: <IP> <domain1> [domain2] [domain3] ...
			if len(fields) < 2 {
				// Line with only IP or malformed
				ip := fields[0]
				if net.ParseIP(ip) != nil {
					result.Stats.SkippedCount++
					continue
				}
				result.Warnings = append(result.Warnings, ParseWarning{
					Line:    lineNum,
					Content: rawLine,
					Reason:  "hosts line missing domain target",
				})
				result.Stats.WarningCount++
				continue
			}

			// Validate IP in first column
			ipCandidate := fields[0]
			if net.ParseIP(ipCandidate) == nil && ipCandidate != "0.0.0.0" && ipCandidate != "127.0.0.1" && ipCandidate != "::1" {
				result.Warnings = append(result.Warnings, ParseWarning{
					Line:    lineNum,
					Content: rawLine,
					Reason:  "invalid IP address in hosts line",
				})
				result.Stats.WarningCount++
				continue
			}

			// Process domains in subsequent fields
			for _, field := range fields[1:] {
				cleaned := strings.ToLower(strings.TrimSuffix(field, "."))
				if isIgnoredHostname(cleaned) {
					result.Stats.SkippedCount++
					continue
				}

				validDomain, reason := validateDomainSyntax(field)
				if reason != "" {
					result.Warnings = append(result.Warnings, ParseWarning{
						Line:    lineNum,
						Content: field,
						Reason:  reason,
					})
					result.Stats.WarningCount++
					continue
				}

				ruleKey := validDomain + ":" + string(ActionBlock)
				if _, exists := seen[ruleKey]; exists {
					result.Stats.DuplicateCount++
					continue
				}

				seen[ruleKey] = struct{}{}
				result.Rules = append(result.Rules, ParsedRule{
					Domain:  validDomain,
					Action:  ActionBlock,
					Source:  "hosts",
					Line:    lineNum,
					Enabled: true,
				})
				result.Stats.AcceptedCount++
			}
		} else if activeFormat == FormatAdblock {
			validDomain, ruleAction, skipped, reason := parseAdblockRule(trimmed)
			if skipped {
				result.Stats.SkippedCount++
				continue
			}
			if reason != "" {
				result.Warnings = append(result.Warnings, ParseWarning{
					Line:    lineNum,
					Content: rawLine,
					Reason:  reason,
				})
				result.Stats.WarningCount++
				continue
			}

			ruleKey := validDomain + ":" + string(ruleAction)
			if _, exists := seen[ruleKey]; exists {
				result.Stats.DuplicateCount++
				continue
			}

			seen[ruleKey] = struct{}{}
			result.Rules = append(result.Rules, ParsedRule{
				Domain:  validDomain,
				Action:  ruleAction,
				Source:  "adblock",
				Line:    lineNum,
				Enabled: true,
			})
			result.Stats.AcceptedCount++
		} else {
			// FormatPlain: one domain per line
			candidate := fields[0]
			if len(fields) > 1 {
				result.Warnings = append(result.Warnings, ParseWarning{
					Line:    lineNum,
					Content: rawLine,
					Reason:  "multiple tokens on plain domain line",
				})
				result.Stats.WarningCount++
				continue
			}

			cleaned := strings.ToLower(strings.TrimSuffix(candidate, "."))
			if isIgnoredHostname(cleaned) {
				result.Stats.SkippedCount++
				continue
			}

			validDomain, reason := validateDomainSyntax(candidate)
			if reason != "" {
				result.Warnings = append(result.Warnings, ParseWarning{
					Line:    lineNum,
					Content: rawLine,
					Reason:  reason,
				})
				result.Stats.WarningCount++
				continue
			}

			ruleKey := validDomain + ":" + string(ActionBlock)
			if _, exists := seen[ruleKey]; exists {
				result.Stats.DuplicateCount++
				continue
			}

			seen[ruleKey] = struct{}{}
			result.Rules = append(result.Rules, ParsedRule{
				Domain:  validDomain,
				Action:  ActionBlock,
				Source:  "plain",
				Line:    lineNum,
				Enabled: true,
			})
			result.Stats.AcceptedCount++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
