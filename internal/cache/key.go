package cache

import (
	"strings"

	"github.com/miekg/dns"
)

// Key uniquely identifies a cached DNS question.
// It separates by domain name, query type, class, DNSSEC OK (DO) flag, and Checking Disabled (CD) bit.
type Key struct {
	Name   string
	Qtype  uint16
	Qclass uint16
	DO     bool
	CD     bool
}

// IsCacheableQuery checks if the query contains unsupported or client-transaction-bound EDNS options
// such as EDNS Client Subnet (ECS, RFC 7871), DNS Cookies (RFC 7873), or TCP Keepalive (RFC 7828).
// Queries with such options bypass unpartitioned caching to prevent leaking transaction-bound state.
func IsCacheableQuery(req *dns.Msg) bool {
	if req == nil || len(req.Question) != 1 {
		return false
	}
	if opt := req.IsEdns0(); opt != nil {
		for _, o := range opt.Option {
			switch o.Option() {
			case dns.EDNS0PADDING:
				// Safe: message length obfuscation only, does not affect DNS answer semantics
				continue
			default:
				// ECS (RFC 7871), DNS Cookies (RFC 7873), TCP Keepalive (RFC 7828),
				// or unknown/unsupported EDNS options: bypass cache
				return false
			}
		}
	}
	return true
}

// NewKey extracts and normalizes a Key from a DNS request message.
// Returns false if the request is nil or does not contain exactly one question.
func NewKey(req *dns.Msg) (Key, bool) {
	if req == nil || len(req.Question) != 1 {
		return Key{}, false
	}

	q := req.Question[0]
	name := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(q.Name), "."))
	if name == "" {
		name = "."
	}

	do := false
	if opt := req.IsEdns0(); opt != nil {
		do = opt.Do()
	}

	return Key{
		Name:   name,
		Qtype:  q.Qtype,
		Qclass: q.Qclass,
		DO:     do,
		CD:     req.CheckingDisabled,
	}, true
}

// NewNameKey creates a key for name-level caching (RFC 2308 NXDOMAIN).
func NewNameKey(name string, qclass uint16, do bool, cd bool) Key {
	cleanName := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(name), "."))
	if cleanName == "" {
		cleanName = "."
	}
	return Key{
		Name:   cleanName,
		Qtype:  dns.TypeNone, // 0 = wildcard for all record types under this name
		Qclass: qclass,
		DO:     do,
		CD:     cd,
	}
}
