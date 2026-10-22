package cache

import (
	"time"

	"github.com/miekg/dns"
)

const (
	DefaultMinTTL         = 5 * time.Second
	DefaultMaxTTL         = 24 * time.Hour
	DefaultMaxNegativeTTL = 300 * time.Second
	DefaultMaxEntries     = 10000
)

// determineTTL inspects the DNS response message and determines the valid TTL duration
// and whether the response is cacheable per RFC 1035, RFC 2181, and RFC 2308.
func determineTTL(msg *dns.Msg, cfg Config) (time.Duration, bool) {
	if msg == nil {
		return 0, false
	}

	// Only NOERROR (Success) and NXDOMAIN (NameError) are cacheable in DNSCache.
	// SERVFAIL, REFUSED, NOTIMP, FORMERR, etc. are never cached as standard answers.
	if msg.Rcode != dns.RcodeSuccess && msg.Rcode != dns.RcodeNameError {
		return 0, false
	}

	// Positive response (NOERROR with Answer records)
	if msg.Rcode == dns.RcodeSuccess && len(msg.Answer) > 0 {
		var minTTL uint32 = 0
		first := true
		for _, rr := range msg.Answer {
			ttl := rr.Header().Ttl
			if first || ttl < minTTL {
				minTTL = ttl
				first = false
			}
		}

		// RFC 1035 Section 3.2.1 / RFC 2181 Section 8:
		// A TTL of zero specifies that the record can only be used for the current transaction
		// and must not be cached. Configured minimum TTL must never make TTL=0 cacheable.
		if minTTL == 0 {
			return 0, false
		}

		dur := time.Duration(minTTL) * time.Second
		if cfg.MinTTL > 0 && dur < cfg.MinTTL {
			dur = cfg.MinTTL
		}
		if cfg.MaxTTL > 0 && dur > cfg.MaxTTL {
			dur = cfg.MaxTTL
		}
		return dur, true
	}

	// Negative response: NXDOMAIN or NODATA (NOERROR with 0 answers)
	// Follow RFC 2308 Section 3 & 5: Extract SOA record from Authority section (Ns).
	var soaTTL time.Duration
	var foundSOA bool

	for _, rr := range msg.Ns {
		if soa, ok := rr.(*dns.SOA); ok {
			// RFC 2308 Section 5: TTL for negative response is min(SOA.TTL, SOA.MIN_TTL)
			sTTL := soa.Hdr.Ttl
			if soa.Minttl < sTTL {
				sTTL = soa.Minttl
			}
			soaTTL = time.Duration(sTTL) * time.Second
			foundSOA = true
			break
		}
	}

	// RFC 2308 Section 3 and 5: A resolver MUST NOT cache a negative response
	// that does not contain a valid SOA record in the Authority section.
	if !foundSOA || soaTTL <= 0 {
		return 0, false
	}

	negTTL := soaTTL
	maxNeg := cfg.MaxNegativeTTL
	if maxNeg <= 0 {
		maxNeg = DefaultMaxNegativeTTL
	}

	if negTTL > maxNeg {
		negTTL = maxNeg
	}

	return negTTL, true
}

// decrementTTL adjusts all record TTLs in the response based on the elapsed duration.
// It skips OPT records whose header TTL carries EDNS0 extended flags.
func decrementTTL(msg *dns.Msg, elapsed time.Duration) {
	if msg == nil {
		return
	}
	elapsedSec := uint32(elapsed.Seconds())
	if elapsedSec == 0 {
		return
	}

	decrementRRs := func(rrs []dns.RR) {
		for _, rr := range rrs {
			if rr == nil || rr.Header().Rrtype == dns.TypeOPT {
				continue
			}
			orig := rr.Header().Ttl
			if orig <= elapsedSec {
				rr.Header().Ttl = 1
			} else {
				rr.Header().Ttl = orig - elapsedSec
			}
		}
	}

	decrementRRs(msg.Answer)
	decrementRRs(msg.Ns)
	decrementRRs(msg.Extra)
}
