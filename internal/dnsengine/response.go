package dnsengine

import (
	"net"

	"github.com/miekg/dns"
)

// MakeBlockResponse constructs a DNS sinkhole response for a blocked query.
// - Type A -> 0.0.0.0
// - Type AAAA -> ::
// - Other query types (HTTPS, SVCB, CNAME, TXT, MX, etc.) -> NOERROR with empty Answer (no fabricated records).
// AuthenticatedData (AD) is explicitly false.
func MakeBlockResponse(req *dns.Msg, sinkholeTTL uint32) *dns.Msg {
	resp := new(dns.Msg)
	resp.SetReply(req)
	resp.Authoritative = true
	resp.RecursionAvailable = true
	resp.AuthenticatedData = false
	resp.Rcode = dns.RcodeSuccess

	if len(req.Question) == 0 {
		return resp
	}

	q := req.Question[0]
	if sinkholeTTL == 0 {
		sinkholeTTL = 60
	}

	switch q.Qtype {
	case dns.TypeA:
		resp.Answer = []dns.RR{
			&dns.A{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    sinkholeTTL,
				},
				A: net.IPv4zero,
			},
		}
	case dns.TypeAAAA:
		resp.Answer = []dns.RR{
			&dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    sinkholeTTL,
				},
				AAAA: net.IPv6zero,
			},
		}
	default:
		// For HTTPS, SVCB, CNAME, TXT, MX, etc., return NOERROR with 0 answer records (NODATA).
		// We deliberately never fabricate CNAME, MX, TXT, SVCB, or HTTPS records.
		resp.Answer = nil
	}

	return resp
}

// MakeErrorResponse constructs a reply message with the given Rcode.
func MakeErrorResponse(req *dns.Msg, rcode int) *dns.Msg {
	resp := new(dns.Msg)
	resp.SetRcode(req, rcode)
	resp.RecursionAvailable = true
	resp.AuthenticatedData = false
	return resp
}
