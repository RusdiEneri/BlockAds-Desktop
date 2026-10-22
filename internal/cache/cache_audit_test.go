package cache_test

import (
	"net"
	"testing"
	"time"

	"blockads-desktop/internal/cache"

	"github.com/miekg/dns"
)

// TestAudit_ZeroTTL_NeverMadeCacheableByMinTTL verifies that a response with TTL=0
// is never cached, even when Config.MinTTL is positive (RFC 1035 / RFC 2181).
func TestAudit_ZeroTTL_NeverMadeCacheableByMinTTL(t *testing.T) {
	c := cache.NewDNSCache(cache.Config{
		MaxEntries: 100,
		MinTTL:     60 * time.Second, // configured MinTTL must NOT override TTL=0
	})

	q := new(dns.Msg)
	q.SetQuestion("zero-ttl.example.com.", dns.TypeA)

	resp := makeTestMessage("zero-ttl.example.com.", dns.TypeA, dns.RcodeSuccess, []dns.RR{
		makeARecord("zero-ttl.example.com.", "1.2.3.4", 0), // TTL = 0
	}, nil)

	stored := c.Set(q, resp)
	if stored {
		t.Fatalf("BUG: response with TTL=0 was accepted by cache; RFC 1035/2181 forbids caching TTL=0")
	}

	hit := c.Get(q)
	if hit != nil {
		t.Fatalf("BUG: response with TTL=0 was returned from cache: %v", hit)
	}
}

// TestAudit_NegativeCaching_RequiresValidSOA verifies that NXDOMAIN and NODATA
// responses without an SOA record in the Authority section are NOT cached (RFC 2308).
func TestAudit_NegativeCaching_RequiresValidSOA(t *testing.T) {
	c := cache.NewDNSCache(cache.Config{
		MaxEntries:         100,
		DefaultNegativeTTL: 30 * time.Second,
	})

	// 1. NXDOMAIN without SOA
	qNX := new(dns.Msg)
	qNX.SetQuestion("nosoa-nx.example.com.", dns.TypeA)
	respNX := makeTestMessage("nosoa-nx.example.com.", dns.TypeA, dns.RcodeNameError, nil, nil)

	if c.Set(qNX, respNX) {
		t.Fatalf("BUG: NXDOMAIN without SOA was cached; RFC 2308 Section 5 requires SOA for negative caching")
	}
	if c.Get(qNX) != nil {
		t.Fatalf("BUG: cache returned negative response that lacked SOA metadata")
	}

	// 2. NODATA without SOA
	qNODATA := new(dns.Msg)
	qNODATA.SetQuestion("nosoa-nodata.example.com.", dns.TypeAAAA)
	respNODATA := makeTestMessage("nosoa-nodata.example.com.", dns.TypeAAAA, dns.RcodeSuccess, nil, nil)

	if c.Set(qNODATA, respNODATA) {
		t.Fatalf("BUG: NODATA without SOA was cached; RFC 2308 Section 5 requires SOA for negative caching")
	}
	if c.Get(qNODATA) != nil {
		t.Fatalf("BUG: cache returned NODATA response that lacked SOA metadata")
	}
}

// TestAudit_ResponseIntegrity_QuestionCasingAndID verifies that cache hits
// preserve the exact Question section of the current request (including mixed case / 0x20)
// and set the current request's transaction ID.
func TestAudit_ResponseIntegrity_QuestionCasingAndID(t *testing.T) {
	c := cache.NewDNSCache(cache.Config{MaxEntries: 100})

	// Store with lower case
	q1 := new(dns.Msg)
	q1.SetQuestion("mycorp.example.com.", dns.TypeA)
	q1.Id = 0x1000

	resp := makeTestMessage("mycorp.example.com.", dns.TypeA, dns.RcodeSuccess, []dns.RR{
		makeARecord("mycorp.example.com.", "10.0.0.1", 300),
	}, nil)

	if !c.Set(q1, resp) {
		t.Fatalf("expected Set to succeed")
	}

	// Query with 0x20 mixed-case encoding and different ID
	q2 := new(dns.Msg)
	q2.SetQuestion("MyCoRp.ExAmPlE.cOm.", dns.TypeA)
	q2.Id = 0x2000

	hit := c.Get(q2)
	if hit == nil {
		t.Fatalf("expected cache hit for case-insensitive match")
	}

	if hit.Id != 0x2000 {
		t.Errorf("expected response ID 0x2000, got 0x%x", hit.Id)
	}

	if len(hit.Question) != 1 {
		t.Fatalf("expected 1 question in response, got %d", len(hit.Question))
	}

	// The response Question MUST match the current query's casing!
	if hit.Question[0].Name != "MyCoRp.ExAmPlE.cOm." {
		t.Errorf("BUG: response question casing not preserved; expected 'MyCoRp.ExAmPlE.cOm.', got %q", hit.Question[0].Name)
	}
}

// TestAudit_ResponseIntegrity_EDNSHandling verifies that cached responses do not
// re-play stale EDNS metadata to non-EDNS clients or clients with different buffer sizes.
func TestAudit_ResponseIntegrity_EDNSHandling(t *testing.T) {
	c := cache.NewDNSCache(cache.Config{MaxEntries: 100})

	// Prime cache with an EDNS request (UDP size 4096, DO=0)
	q1 := new(dns.Msg)
	q1.SetQuestion("edns.example.com.", dns.TypeA)
	q1.SetEdns0(4096, false)

	resp1 := makeTestMessage("edns.example.com.", dns.TypeA, dns.RcodeSuccess, []dns.RR{
		makeARecord("edns.example.com.", "1.1.1.1", 300),
	}, nil)
	resp1.SetEdns0(4096, false)

	c.Set(q1, resp1)

	// Query 2: Non-EDNS request (traditional client). Must NOT receive an OPT record!
	q2 := new(dns.Msg)
	q2.SetQuestion("edns.example.com.", dns.TypeA)
	// No SetEdns0

	hit2 := c.Get(q2)
	if hit2 == nil {
		t.Fatalf("expected cache hit for non-EDNS query")
	}
	if opt := hit2.IsEdns0(); opt != nil {
		t.Fatalf("BUG: non-EDNS client received an OPT record in cached response (RFC 6891 violation)")
	}

	// Query 3: EDNS request with buffer size 1232 and DO=0
	q3 := new(dns.Msg)
	q3.SetQuestion("edns.example.com.", dns.TypeA)
	q3.SetEdns0(1232, false)

	hit3 := c.Get(q3)
	if hit3 == nil {
		t.Fatalf("expected cache hit")
	}
	opt3 := hit3.IsEdns0()
	if opt3 == nil {
		t.Fatalf("expected OPT record for EDNS request")
	}
	if opt3.UDPSize() != 1232 {
		t.Errorf("expected UDP buffer size 1232 in response, got %d", opt3.UDPSize())
	}
	if opt3.Do() {
		t.Errorf("expected DO=false in response")
	}

	// Query 4: DO=true query must be a cache miss due to DO isolation
	q4 := new(dns.Msg)
	q4.SetQuestion("edns.example.com.", dns.TypeA)
	q4.SetEdns0(4096, true)
	if c.Get(q4) != nil {
		t.Fatalf("expected DO=true query to miss cache primed with DO=false")
	}
}

// TestAudit_RFC2308_NXDOMAIN_NameLevelSemantics verifies that an NXDOMAIN response
// (without CNAME answers) satisfies subsequent queries for any QTYPE at that name.
func TestAudit_RFC2308_NXDOMAIN_NameLevelSemantics(t *testing.T) {
	c := cache.NewDNSCache(cache.Config{
		MaxEntries: 100,
	})

	qA := new(dns.Msg)
	qA.SetQuestion("nonexistent-domain.org.", dns.TypeA)

	respNX := makeTestMessage("nonexistent-domain.org.", dns.TypeA, dns.RcodeNameError, nil, []dns.RR{
		makeSOARecord("org.", 300, 60),
	})

	if !c.Set(qA, respNX) {
		t.Fatalf("failed to cache valid NXDOMAIN with SOA")
	}

	// Now query for AAAA on the same name. Per RFC 2308 Section 3, NXDOMAIN applies to the NAME!
	qAAAA := new(dns.Msg)
	qAAAA.SetQuestion("nonexistent-domain.org.", dns.TypeAAAA)
	qAAAA.Id = 0x3333

	hitAAAA := c.Get(qAAAA)
	if hitAAAA == nil {
		t.Fatalf("BUG: NXDOMAIN was not applied to other QTYPE on same nonexistent name (RFC 2308 Sec 3)")
	}
	if hitAAAA.Rcode != dns.RcodeNameError {
		t.Fatalf("expected RcodeNameError, got %d", hitAAAA.Rcode)
	}
	if hitAAAA.Question[0].Qtype != dns.TypeAAAA {
		t.Errorf("expected response Question Qtype to be AAAA, got %d", hitAAAA.Question[0].Qtype)
	}
	if hitAAAA.Id != 0x3333 {
		t.Errorf("expected response ID 0x3333, got 0x%x", hitAAAA.Id)
	}

	// Verify that NODATA does NOT have this behavior (NODATA is strictly type-specific!)
	qNODATAA := new(dns.Msg)
	qNODATAA.SetQuestion("has-v4-only.org.", dns.TypeAAAA)
	respNODATA := makeTestMessage("has-v4-only.org.", dns.TypeAAAA, dns.RcodeSuccess, nil, []dns.RR{
		makeSOARecord("org.", 300, 60),
	})
	c.Set(qNODATAA, respNODATA)

	// Query for TXT on has-v4-only.org should be a cache MISS!
	qTXT := new(dns.Msg)
	qTXT.SetQuestion("has-v4-only.org.", dns.TypeTXT)
	if c.Get(qTXT) != nil {
		t.Fatalf("BUG: NODATA response for AAAA was falsely returned for TXT query (NODATA must be type-specific)")
	}
}

// TestAudit_CacheKey_CDAndECS verifies that CD bit is isolated in cache keys
// and queries with EDNS Client Subnet (ECS) bypass caching.
func TestAudit_CacheKey_CDAndECS(t *testing.T) {
	c := cache.NewDNSCache(cache.Config{MaxEntries: 100})

	// 1. CD bit isolation: CD=0 vs CD=1
	qCD0 := new(dns.Msg)
	qCD0.SetQuestion("dnssec-test.org.", dns.TypeA)
	qCD0.CheckingDisabled = false

	resp := makeTestMessage("dnssec-test.org.", dns.TypeA, dns.RcodeSuccess, []dns.RR{
		makeARecord("dnssec-test.org.", "9.9.9.9", 300),
	}, nil)

	c.Set(qCD0, resp)

	qCD1 := new(dns.Msg)
	qCD1.SetQuestion("dnssec-test.org.", dns.TypeA)
	qCD1.CheckingDisabled = true

	hitCD1 := c.Get(qCD1)
	if hitCD1 != nil {
		t.Fatalf("BUG: query with CD=1 matched cached response for CD=0; CD bit must be isolated in cache key")
	}

	// 2. EDNS Client Subnet (ECS) bypass
	qECS := new(dns.Msg)
	qECS.SetQuestion("geo.example.com.", dns.TypeA)
	optECS := new(dns.OPT)
	optECS.Hdr.Name = "."
	optECS.Hdr.Rrtype = dns.TypeOPT
	optECS.Option = append(optECS.Option, &dns.EDNS0_SUBNET{
		Code:          dns.EDNS0SUBNET,
		Family:        1, // IPv4
		SourceNetmask: 24,
		Address:       net.ParseIP("192.0.2.0"),
	})
	qECS.Extra = append(qECS.Extra, optECS)

	if cache.IsCacheableQuery(qECS) {
		t.Fatalf("BUG: queries containing EDNS Client Subnet (ECS) must bypass unpartitioned cache")
	}

	// 3. DNS Cookie (RFC 7873) bypass
	qCookie := new(dns.Msg)
	qCookie.SetQuestion("cookie.example.com.", dns.TypeA)
	optCookie := new(dns.OPT)
	optCookie.Hdr.Name = "."
	optCookie.Hdr.Rrtype = dns.TypeOPT
	optCookie.Option = append(optCookie.Option, &dns.EDNS0_COOKIE{
		Code:   dns.EDNS0COOKIE,
		Cookie: "0123456789abcdef",
	})
	qCookie.Extra = append(qCookie.Extra, optCookie)

	if cache.IsCacheableQuery(qCookie) {
		t.Fatalf("BUG: queries containing DNS Cookies (RFC 7873) must bypass unpartitioned cache")
	}

	// 4. EDNS0 Padding (RFC 7830) is safe and cacheable
	qPadding := new(dns.Msg)
	qPadding.SetQuestion("padding.example.com.", dns.TypeA)
	optPadding := new(dns.OPT)
	optPadding.Hdr.Name = "."
	optPadding.Hdr.Rrtype = dns.TypeOPT
	optPadding.Option = append(optPadding.Option, &dns.EDNS0_PADDING{
		Padding: []byte{0, 0, 0, 0},
	})
	qPadding.Extra = append(qPadding.Extra, optPadding)

	if !cache.IsCacheableQuery(qPadding) {
		t.Fatalf("queries with only EDNS0 padding should remain cacheable")
	}
}
