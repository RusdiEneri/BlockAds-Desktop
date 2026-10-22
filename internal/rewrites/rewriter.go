package rewrites

import (
	"net"
	"sort"
	"strings"
	"sync"

	"github.com/miekg/dns"
)

const (
	DefaultRewriteTTL = 60
	MaxCNAMEChain     = 8
)

type wildcardEntry struct {
	suffix string // lowercase without leading "*." and without trailing "."
	rule   Rule
}

// Rewriter manages and executes DNS rewrites with exact matching, wildcard support,
// and CNAME loop detection.
type Rewriter struct {
	mu            sync.RWMutex
	exactRules    map[string]map[uint16]Rule // domain -> qtype -> rule
	wildcardRules []wildcardEntry            // sorted by label depth descending
}

// NewRewriter initializes an empty Rewriter instance.
func NewRewriter() *Rewriter {
	return &Rewriter{
		exactRules:    make(map[string]map[uint16]Rule),
		wildcardRules: make([]wildcardEntry, 0),
	}
}

// AddRules validates and loads rewrite rules.
func (rw *Rewriter) AddRules(rules []Rule) error {
	for i := range rules {
		if err := rules[i].Validate(); err != nil {
			return err
		}
	}

	rw.mu.Lock()
	defer rw.mu.Unlock()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		domain := rule.CanonicalDomain()
		if rule.TTL == 0 {
			rule.TTL = DefaultRewriteTTL
		}

		if strings.HasPrefix(domain, "*.") {
			suffix := strings.TrimPrefix(domain, "*.")
			rw.wildcardRules = append(rw.wildcardRules, wildcardEntry{
				suffix: suffix,
				rule:   rule,
			})
		} else {
			qtype := parseQtype(rule.Type)
			if _, exists := rw.exactRules[domain]; !exists {
				rw.exactRules[domain] = make(map[uint16]Rule)
			}
			rw.exactRules[domain][qtype] = rule
		}
	}

	// Sort wildcards by suffix length (label depth) descending so more specific wildcards win
	sort.SliceStable(rw.wildcardRules, func(i, j int) bool {
		return len(rw.wildcardRules[i].suffix) > len(rw.wildcardRules[j].suffix)
	})

	return nil
}

// Clear removes all rewrite rules.
func (rw *Rewriter) Clear() {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	rw.exactRules = make(map[string]map[uint16]Rule)
	rw.wildcardRules = make([]wildcardEntry, 0)
}

// Rewrite checks if an incoming DNS query matches any active rewrite rule.
// Returns a synthesized *dns.Msg if rewritten, or nil if no rewrite applies.
func (rw *Rewriter) Rewrite(req *dns.Msg) *dns.Msg {
	if req == nil || len(req.Question) != 1 {
		return nil
	}

	q := req.Question[0]
	name := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(q.Name), "."))

	rw.mu.RLock()
	defer rw.mu.RUnlock()

	// 1. Check exact match
	if typeMap, exists := rw.exactRules[name]; exists {
		// A. Check for matching Qtype
		if rule, match := typeMap[q.Qtype]; match {
			return rw.buildResponse(req, q, rule)
		}
		// B. Check if domain has a CNAME rule (CNAME applies to all query types)
		if rule, match := typeMap[dns.TypeCNAME]; match {
			return rw.buildCNAMEResponse(req, q, rule)
		}
		// C. Domain exists in exact rewrites, but requested Qtype is not configured:
		// Return NODATA (NOERROR with 0 answers) per RFC 1034 to prevent leaking local domains upstream
		return rw.buildNODATAResponse(req)
	}

	// 2. Check wildcard matches
	for _, we := range rw.wildcardRules {
		if name == we.suffix || strings.HasSuffix(name, "."+we.suffix) {
			ruleQtype := parseQtype(we.rule.Type)
			if ruleQtype == q.Qtype {
				return rw.buildResponse(req, q, we.rule)
			}
			if ruleQtype == dns.TypeCNAME {
				return rw.buildCNAMEResponse(req, q, we.rule)
			}
			// Wildcard matched domain, but query type does not match rule: NODATA
			return rw.buildNODATAResponse(req)
		}
	}

	return nil
}

func (rw *Rewriter) buildResponse(req *dns.Msg, q dns.Question, rule Rule) *dns.Msg {
	resp := new(dns.Msg)
	resp.SetReply(req)
	resp.Rcode = dns.RcodeSuccess
	resp.RecursionAvailable = true

	ttl := rule.TTL
	if ttl == 0 {
		ttl = DefaultRewriteTTL
	}

	switch strings.ToUpper(rule.Type) {
	case "A":
		resp.Answer = append(resp.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			A: net.ParseIP(rule.Target).To4(),
		})

	case "AAAA":
		resp.Answer = append(resp.Answer, &dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			AAAA: net.ParseIP(rule.Target),
		})
	}

	return resp
}

func (rw *Rewriter) buildNODATAResponse(req *dns.Msg) *dns.Msg {
	resp := new(dns.Msg)
	resp.SetReply(req)
	resp.Rcode = dns.RcodeSuccess
	resp.RecursionAvailable = true
	return resp
}

func (rw *Rewriter) buildCNAMEResponse(req *dns.Msg, q dns.Question, initialRule Rule) *dns.Msg {
	resp := new(dns.Msg)
	resp.SetReply(req)
	resp.RecursionAvailable = true

	visited := make(map[string]bool)
	currentDomain := q.Name
	currentRule := initialRule
	depth := 0

	for {
		if depth >= MaxCNAMEChain {
			// CNAME loop or excessive depth detected
			failResp := new(dns.Msg)
			failResp.SetReply(req)
			failResp.Rcode = dns.RcodeServerFailure
			return failResp
		}

		target := currentRule.CanonicalTarget()
		cleanCurrent := strings.ToLower(strings.TrimSuffix(currentDomain, "."))

		if visited[target] || cleanCurrent == target {
			// Circular loop detected!
			failResp := new(dns.Msg)
			failResp.SetReply(req)
			failResp.Rcode = dns.RcodeServerFailure
			return failResp
		}
		visited[cleanCurrent] = true

		ttl := currentRule.TTL
		if ttl == 0 {
			ttl = DefaultRewriteTTL
		}

		// Add CNAME record to Answer section
		resp.Answer = append(resp.Answer, &dns.CNAME{
			Hdr: dns.RR_Header{
				Name:   dns.Fqdn(currentDomain),
				Rrtype: dns.TypeCNAME,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			Target: dns.Fqdn(target),
		})

		// Now check if target resolves to an A/AAAA record or another CNAME in local rewrites
		if targetMap, exists := rw.exactRules[target]; exists {
			// Does target have A/AAAA matching original q.Qtype?
			if targetRule, ok := targetMap[q.Qtype]; ok {
				tTTL := targetRule.TTL
				if tTTL == 0 {
					tTTL = DefaultRewriteTTL
				}
				if q.Qtype == dns.TypeA {
					resp.Answer = append(resp.Answer, &dns.A{
						Hdr: dns.RR_Header{
							Name:   dns.Fqdn(target),
							Rrtype: dns.TypeA,
							Class:  dns.ClassINET,
							Ttl:    tTTL,
						},
						A: net.ParseIP(targetRule.Target).To4(),
					})
				} else if q.Qtype == dns.TypeAAAA {
					resp.Answer = append(resp.Answer, &dns.AAAA{
						Hdr: dns.RR_Header{
							Name:   dns.Fqdn(target),
							Rrtype: dns.TypeAAAA,
							Class:  dns.ClassINET,
							Ttl:    tTTL,
						},
						AAAA: net.ParseIP(targetRule.Target),
					})
				}
				resp.Rcode = dns.RcodeSuccess
				return resp
			}

			// Does target have another CNAME?
			if nextCNAME, ok := targetMap[dns.TypeCNAME]; ok {
				currentDomain = target
				currentRule = nextCNAME
				depth++
				continue
			}
		}

		// Target has no further local rewrite records; return CNAME record
		resp.Rcode = dns.RcodeSuccess
		return resp
	}
}

func parseQtype(typeStr string) uint16 {
	switch strings.ToUpper(strings.TrimSpace(typeStr)) {
	case "A":
		return dns.TypeA
	case "AAAA":
		return dns.TypeAAAA
	case "CNAME":
		return dns.TypeCNAME
	default:
		return dns.TypeNone
	}
}
