package cache_test

import (
	"net"
	"sync"
	"testing"
	"time"

	"blockads-desktop/internal/cache"

	"github.com/miekg/dns"
)

func makeTestMessage(qname string, qtype uint16, rcode int, answers []dns.RR, ns []dns.RR) *dns.Msg {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(qname), qtype)
	m.Rcode = rcode
	m.Response = true
	m.RecursionAvailable = true
	m.Answer = answers
	m.Ns = ns
	return m
}

func makeARecord(name string, ipStr string, ttl uint32) dns.RR {
	return &dns.A{
		Hdr: dns.RR_Header{
			Name:   dns.Fqdn(name),
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    ttl,
		},
		A: net.ParseIP(ipStr).To4(),
	}
}

func makeAAAARecord(name string, ipStr string, ttl uint32) dns.RR {
	return &dns.AAAA{
		Hdr: dns.RR_Header{
			Name:   dns.Fqdn(name),
			Rrtype: dns.TypeAAAA,
			Class:  dns.ClassINET,
			Ttl:    ttl,
		},
		AAAA: net.ParseIP(ipStr),
	}
}

func makeCNAMERecord(name string, target string, ttl uint32) dns.RR {
	return &dns.CNAME{
		Hdr: dns.RR_Header{
			Name:   dns.Fqdn(name),
			Rrtype: dns.TypeCNAME,
			Class:  dns.ClassINET,
			Ttl:    ttl,
		},
		Target: dns.Fqdn(target),
	}
}

func makeSOARecord(name string, ttl uint32, minttl uint32) dns.RR {
	return &dns.SOA{
		Hdr: dns.RR_Header{
			Name:   dns.Fqdn(name),
			Rrtype: dns.TypeSOA,
			Class:  dns.ClassINET,
			Ttl:    ttl,
		},
		Ns:      "ns1." + dns.Fqdn(name),
		Mbox:    "hostmaster." + dns.Fqdn(name),
		Serial:  2026102201,
		Refresh: 7200,
		Retry:   3600,
		Expire:  1209600,
		Minttl:  minttl,
	}
}

func TestDNSCache_HitMissAndTTLDecrement(t *testing.T) {
	mockClock := cache.NewMockClock(time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC))
	c := cache.NewDNSCache(cache.Config{
		MaxEntries: 100,
		Clock:      mockClock,
	})

	q := new(dns.Msg)
	q.SetQuestion("example.com.", dns.TypeA)

	// 1. Initial lookup -> Miss
	hit := c.Get(q)
	if hit != nil {
		t.Fatalf("expected initial cache miss, got %v", hit)
	}

	stats := c.Stats()
	if stats.Misses != 1 || stats.Hits != 0 {
		t.Fatalf("unexpected stats after miss: %+v", stats)
	}

	// 2. Store positive A response with 300s TTL
	resp := makeTestMessage("example.com.", dns.TypeA, dns.RcodeSuccess, []dns.RR{
		makeARecord("example.com.", "93.184.216.34", 300),
	}, nil)

	stored := c.Set(q, resp)
	if !stored {
		t.Fatalf("expected Set to return true for valid positive response")
	}

	// 3. Immediate lookup -> Hit, TTL unchanged (300)
	hit = c.Get(q)
	if hit == nil {
		t.Fatalf("expected cache hit immediately after Set")
	}
	if len(hit.Answer) != 1 {
		t.Fatalf("expected 1 answer record, got %d", len(hit.Answer))
	}
	if hit.Answer[0].Header().Ttl != 300 {
		t.Fatalf("expected TTL 300, got %d", hit.Answer[0].Header().Ttl)
	}

	// 4. Advance clock by 100 seconds -> Hit, TTL decremented to 200
	mockClock.Advance(100 * time.Second)
	hit = c.Get(q)
	if hit == nil {
		t.Fatalf("expected cache hit after 100s")
	}
	if hit.Answer[0].Header().Ttl != 200 {
		t.Fatalf("expected TTL 200, got %d", hit.Answer[0].Header().Ttl)
	}

	stats = c.Stats()
	if stats.Hits != 2 || stats.Misses != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	// 5. Advance clock by another 201 seconds (total 301s) -> Expired, Miss
	mockClock.Advance(201 * time.Second)
	hit = c.Get(q)
	if hit != nil {
		t.Fatalf("expected cache miss due to expiration, got %v", hit)
	}

	stats = c.Stats()
	if stats.Expirations != 1 || stats.Misses != 2 {
		t.Fatalf("expected Expirations=1, Misses=2, got %+v", stats)
	}
}

func TestDNSCache_PositiveAAAAAndCNAME(t *testing.T) {
	mockClock := cache.NewMockClock(time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC))
	c := cache.NewDNSCache(cache.Config{
		MaxEntries: 100,
		Clock:      mockClock,
	})

	q := new(dns.Msg)
	q.SetQuestion("alias.example.com.", dns.TypeA)

	// Response with CNAME and A record
	resp := makeTestMessage("alias.example.com.", dns.TypeA, dns.RcodeSuccess, []dns.RR{
		makeCNAMERecord("alias.example.com.", "target.example.com.", 600),
		makeARecord("target.example.com.", "93.184.216.34", 300),
	}, nil)

	c.Set(q, resp)

	// Advance clock by 50 seconds
	mockClock.Advance(50 * time.Second)

	hit := c.Get(q)
	if hit == nil {
		t.Fatalf("expected cache hit")
	}
	if len(hit.Answer) != 2 {
		t.Fatalf("expected 2 answers, got %d", len(hit.Answer))
	}
	if hit.Answer[0].Header().Ttl != 550 {
		t.Errorf("expected CNAME TTL 550, got %d", hit.Answer[0].Header().Ttl)
	}
	if hit.Answer[1].Header().Ttl != 250 {
		t.Errorf("expected A TTL 250, got %d", hit.Answer[1].Header().Ttl)
	}

	// AAAA lookup test
	qAAAA := new(dns.Msg)
	qAAAA.SetQuestion("ipv6.example.com.", dns.TypeAAAA)
	respAAAA := makeTestMessage("ipv6.example.com.", dns.TypeAAAA, dns.RcodeSuccess, []dns.RR{
		makeAAAARecord("ipv6.example.com.", "2606:2800:220:1:248:1893:25c8:1946", 400),
	}, nil)

	c.Set(qAAAA, respAAAA)
	mockClock.Advance(100 * time.Second)

	hitAAAA := c.Get(qAAAA)
	if hitAAAA == nil {
		t.Fatalf("expected AAAA cache hit")
	}
	if hitAAAA.Answer[0].Header().Ttl != 300 {
		t.Errorf("expected AAAA TTL 300, got %d", hitAAAA.Answer[0].Header().Ttl)
	}
}

func TestDNSCache_NegativeCaching(t *testing.T) {
	mockClock := cache.NewMockClock(time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC))
	c := cache.NewDNSCache(cache.Config{
		MaxEntries:         100,
		Clock:              mockClock,
		DefaultNegativeTTL: 30 * time.Second,
		MaxNegativeTTL:     300 * time.Second,
	})

	// 1. NXDOMAIN with SOA record: RFC 2308 min(SOA.Ttl, SOA.Minttl)
	qNX := new(dns.Msg)
	qNX.SetQuestion("nonexistent.example.com.", dns.TypeA)

	respNX := makeTestMessage("nonexistent.example.com.", dns.TypeA, dns.RcodeNameError, nil, []dns.RR{
		makeSOARecord("example.com.", 600, 120), // min(600, 120) = 120s
	})

	stored := c.Set(qNX, respNX)
	if !stored {
		t.Fatalf("expected NXDOMAIN with SOA to be cached")
	}

	hitNX := c.Get(qNX)
	if hitNX == nil {
		t.Fatalf("expected NXDOMAIN cache hit")
	}
	if hitNX.Rcode != dns.RcodeNameError {
		t.Fatalf("expected RcodeNameError, got %d", hitNX.Rcode)
	}
	if len(hitNX.Ns) != 1 || hitNX.Ns[0].Header().Ttl != 120 {
		t.Fatalf("expected SOA TTL 120, got %d", hitNX.Ns[0].Header().Ttl)
	}

	// Advance 60s
	mockClock.Advance(60 * time.Second)
	hitNX = c.Get(qNX)
	if hitNX == nil {
		t.Fatalf("expected NXDOMAIN hit after 60s")
	}
	if hitNX.Ns[0].Header().Ttl != 60 {
		t.Fatalf("expected SOA TTL 60 after 60s, got %d", hitNX.Ns[0].Header().Ttl)
	}

	// Advance another 61s -> Expired
	mockClock.Advance(61 * time.Second)
	hitNX = c.Get(qNX)
	if hitNX != nil {
		t.Fatalf("expected NXDOMAIN to be expired after 121s")
	}

	// 2. NODATA (NOERROR, 0 answers) with SOA record
	qNODATA := new(dns.Msg)
	qNODATA.SetQuestion("nodata.example.com.", dns.TypeAAAA)

	respNODATA := makeTestMessage("nodata.example.com.", dns.TypeAAAA, dns.RcodeSuccess, nil, []dns.RR{
		makeSOARecord("example.com.", 80, 200), // min(80, 200) = 80s
	})

	c.Set(qNODATA, respNODATA)
	hitNODATA := c.Get(qNODATA)
	if hitNODATA == nil {
		t.Fatalf("expected NODATA cache hit")
	}
	if hitNODATA.Rcode != dns.RcodeSuccess || len(hitNODATA.Answer) != 0 {
		t.Fatalf("expected NODATA (RcodeSuccess with 0 answers)")
	}
	if hitNODATA.Ns[0].Header().Ttl != 80 {
		t.Fatalf("expected SOA TTL 80, got %d", hitNODATA.Ns[0].Header().Ttl)
	}

	// 3. Negative response without SOA must NOT be cached (RFC 2308 Section 3 & 5)
	qNXNoSOA := new(dns.Msg)
	qNXNoSOA.SetQuestion("nosoa.example.com.", dns.TypeA)
	respNXNoSOA := makeTestMessage("nosoa.example.com.", dns.TypeA, dns.RcodeNameError, nil, nil)

	if c.Set(qNXNoSOA, respNXNoSOA) {
		t.Fatalf("expected NXDOMAIN without SOA to be rejected by cache per RFC 2308")
	}
	if c.Get(qNXNoSOA) != nil {
		t.Fatalf("expected NXDOMAIN without SOA not to be returned from cache")
	}
}

func TestDNSCache_DoNotCacheSERVFAILOrErrors(t *testing.T) {
	c := cache.NewDNSCache(cache.Config{
		MaxEntries: 100,
	})

	q := new(dns.Msg)
	q.SetQuestion("fail.example.com.", dns.TypeA)

	// SERVFAIL response
	respSERVFAIL := makeTestMessage("fail.example.com.", dns.TypeA, dns.RcodeServerFailure, nil, nil)
	if c.Set(q, respSERVFAIL) {
		t.Fatalf("SERVFAIL response must not be cached")
	}
	if c.Get(q) != nil {
		t.Fatalf("cache should not return SERVFAIL")
	}

	// REFUSED response
	respREFUSED := makeTestMessage("fail.example.com.", dns.TypeA, dns.RcodeRefused, nil, nil)
	if c.Set(q, respREFUSED) {
		t.Fatalf("REFUSED response must not be cached")
	}

	// Nil response
	if c.Set(q, nil) {
		t.Fatalf("nil response must not be cached")
	}
}

func TestDNSCache_LRUEviction(t *testing.T) {
	c := cache.NewDNSCache(cache.Config{
		MaxEntries: 3,
	})

	mkMsg := func(domain string) (*dns.Msg, *dns.Msg) {
		q := new(dns.Msg)
		q.SetQuestion(domain+".", dns.TypeA)
		r := makeTestMessage(domain+".", dns.TypeA, dns.RcodeSuccess, []dns.RR{
			makeARecord(domain+".", "1.1.1.1", 300),
		}, nil)
		return q, r
	}

	q1, r1 := mkMsg("domain1")
	q2, r2 := mkMsg("domain2")
	q3, r3 := mkMsg("domain3")
	q4, r4 := mkMsg("domain4")

	c.Set(q1, r1)
	c.Set(q2, r2)
	c.Set(q3, r3)

	if c.Stats().Size != 3 {
		t.Fatalf("expected size 3, got %d", c.Stats().Size)
	}

	// Access domain1 (now domain2 is oldest/LRU)
	if c.Get(q1) == nil {
		t.Fatalf("expected hit on domain1")
	}

	// Add domain4 -> domain2 should be evicted
	c.Set(q4, r4)

	stats := c.Stats()
	if stats.Evictions != 1 {
		t.Fatalf("expected 1 eviction, got %d", stats.Evictions)
	}
	if stats.Size != 3 {
		t.Fatalf("expected size to remain at capacity 3, got %d", stats.Size)
	}

	// domain2 should be gone
	if c.Get(q2) != nil {
		t.Fatalf("domain2 should have been evicted as LRU")
	}

	// domain1, domain3, domain4 should be present
	if c.Get(q1) == nil || c.Get(q3) == nil || c.Get(q4) == nil {
		t.Fatalf("expected domain1, domain3, and domain4 to be in cache")
	}
}

func TestDNSCache_DeepCopySafety(t *testing.T) {
	c := cache.NewDNSCache(cache.Config{
		MaxEntries: 10,
	})

	q := new(dns.Msg)
	q.SetQuestion("example.com.", dns.TypeA)

	resp := makeTestMessage("example.com.", dns.TypeA, dns.RcodeSuccess, []dns.RR{
		makeARecord("example.com.", "1.2.3.4", 300),
	}, nil)

	c.Set(q, resp)

	// Mutate the original response AFTER storing
	resp.Answer[0].(*dns.A).A = net.ParseIP("9.9.9.9").To4()

	hit1 := c.Get(q)
	if hit1 == nil {
		t.Fatalf("expected cache hit")
	}
	if hit1.Answer[0].(*dns.A).A.String() != "1.2.3.4" {
		t.Fatalf("mutating original response affected cached copy! got %s", hit1.Answer[0].(*dns.A).A.String())
	}

	// Mutate the returned message
	hit1.Answer[0].(*dns.A).A = net.ParseIP("8.8.8.8").To4()

	hit2 := c.Get(q)
	if hit2 == nil {
		t.Fatalf("expected cache hit")
	}
	if hit2.Answer[0].(*dns.A).A.String() != "1.2.3.4" {
		t.Fatalf("mutating returned message affected cached copy! got %s", hit2.Answer[0].(*dns.A).A.String())
	}
}

func TestDNSCache_Flush(t *testing.T) {
	c := cache.NewDNSCache(cache.Config{
		MaxEntries: 10,
	})

	q := new(dns.Msg)
	q.SetQuestion("flush.example.com.", dns.TypeA)
	resp := makeTestMessage("flush.example.com.", dns.TypeA, dns.RcodeSuccess, []dns.RR{
		makeARecord("flush.example.com.", "1.1.1.1", 300),
	}, nil)

	c.Set(q, resp)
	if c.Stats().Size != 1 {
		t.Fatalf("expected size 1")
	}

	c.Flush()

	if c.Stats().Size != 0 {
		t.Fatalf("expected size 0 after flush, got %d", c.Stats().Size)
	}
	if c.Get(q) != nil {
		t.Fatalf("expected miss after flush")
	}
}

func TestDNSCache_ConcurrentAccess(t *testing.T) {
	c := cache.NewDNSCache(cache.Config{
		MaxEntries: 50,
	})

	var wg sync.WaitGroup
	workers := 20
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				q := new(dns.Msg)
				q.SetQuestion("concurrent.example.com.", dns.TypeA)

				if j%2 == 0 {
					c.Get(q)
				} else {
					resp := makeTestMessage("concurrent.example.com.", dns.TypeA, dns.RcodeSuccess, []dns.RR{
						makeARecord("concurrent.example.com.", "1.1.1.1", 300),
					}, nil)
					c.Set(q, resp)
				}
			}
		}(i)
	}

	wg.Wait()

	stats := c.Stats()
	if stats.Hits+stats.Misses == 0 {
		t.Fatalf("expected non-zero operations in concurrency test")
	}
}
