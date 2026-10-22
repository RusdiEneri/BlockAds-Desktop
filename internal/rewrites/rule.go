package rewrites

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/miekg/dns"
)

// Rule represents a local DNS rewrite rule.
type Rule struct {
	ID      string `json:"id"`
	Domain  string `json:"domain"`
	Type    string `json:"type"`   // "A", "AAAA", "CNAME"
	Target  string `json:"target"` // IPv4, IPv6, or target domain name
	Enabled bool   `json:"enabled"`
	TTL     uint32 `json:"ttl"` // TTL in seconds (default 60)
}

// CanonicalDomain returns the normalized lowercase domain without trailing dot.
func (r *Rule) CanonicalDomain() string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(r.Domain), "."))
}

// CanonicalTarget returns the normalized lowercase target.
func (r *Rule) CanonicalTarget() string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(r.Target), "."))
}

// Validate checks whether the rewrite rule has valid domain, type, and target syntax.
func (r *Rule) Validate() error {
	domain := r.CanonicalDomain()
	if domain == "" {
		return errors.New("domain cannot be empty")
	}

	cleanDomain := domain
	if strings.HasPrefix(cleanDomain, "*.") {
		cleanDomain = strings.TrimPrefix(cleanDomain, "*.")
		if cleanDomain == "" {
			return errors.New("wildcard rule must specify a base domain (e.g. *.example.com)")
		}
	}

	if _, ok := dns.IsDomainName(cleanDomain); !ok {
		return fmt.Errorf("invalid domain syntax: %q", domain)
	}

	qtypeStr := strings.ToUpper(strings.TrimSpace(r.Type))
	switch qtypeStr {
	case "A":
		ip := net.ParseIP(strings.TrimSpace(r.Target))
		if ip == nil || ip.To4() == nil {
			return fmt.Errorf("target %q is not a valid IPv4 address for type A", r.Target)
		}

	case "AAAA":
		ip := net.ParseIP(strings.TrimSpace(r.Target))
		if ip == nil || ip.To4() != nil {
			return fmt.Errorf("target %q is not a valid IPv6 address for type AAAA", r.Target)
		}

	case "CNAME":
		target := r.CanonicalTarget()
		if target == "" {
			return errors.New("CNAME target cannot be empty")
		}
		if net.ParseIP(target) != nil {
			return fmt.Errorf("CNAME target %q cannot be an IP address", r.Target)
		}
		if target == domain {
			return fmt.Errorf("CNAME target cannot point to the same domain %q (self loop)", domain)
		}
		if _, ok := dns.IsDomainName(target); !ok {
			return fmt.Errorf("invalid CNAME target domain syntax: %q", r.Target)
		}

	default:
		return fmt.Errorf("unsupported rewrite type %q; must be A, AAAA, or CNAME", r.Type)
	}

	return nil
}
