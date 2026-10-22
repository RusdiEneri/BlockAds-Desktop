package dnsengine_test

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"

	"blockads-desktop/internal/cache"
	"blockads-desktop/internal/dnsengine"
	"blockads-desktop/internal/filters"
)

func TestHandler_CacheHitAndLatency(t *testing.T) {
	var resolveCalls atomic.Int32

	fakeResolver := dnsengine.NewFakeResolver(func(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
		resolveCalls.Add(1)
		time.Sleep(10 * time.Millisecond) // simulate upstream latency
		resp := new(dns.Msg)
		resp.SetReply(req)
		resp.Rcode = dns.RcodeSuccess
		resp.Answer = append(resp.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: req.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
			A:   net.ParseIP("93.184.216.34"),
		})
		return resp, nil
	})

	dnsCache := cache.NewDNSCache(cache.Config{
		MaxEntries: 100,
	})

	matcher := filters.NewMatcher()

	var recordedDecisions []dnsengine.QueryDecision
	observer := func(d dnsengine.QueryDecision) {
		recordedDecisions = append(recordedDecisions, d)
	}

	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Matcher:       matcher,
		Resolver:      fakeResolver,
		QueryObserver: observer,
		Cache:         dnsCache,
	})

	// 1. First query: cache miss, upstream resolved
	req1 := new(dns.Msg)
	req1.SetQuestion("example.com.", dns.TypeA)
	req1.Id = 0x1111

	w1 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w1, req1)

	if resolveCalls.Load() != 1 {
		t.Fatalf("expected 1 resolver call on first query, got %d", resolveCalls.Load())
	}
	if w1.writtenMsg == nil || len(w1.writtenMsg.Answer) != 1 {
		t.Fatalf("expected valid response on first query")
	}

	// 2. Second query: cache hit, upstream NOT called
	req2 := new(dns.Msg)
	req2.SetQuestion("example.com.", dns.TypeA)
	req2.Id = 0x2222

	w2 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w2, req2)

	if resolveCalls.Load() != 1 {
		t.Fatalf("expected still 1 resolver call after cache hit, got %d", resolveCalls.Load())
	}
	if w2.writtenMsg == nil || len(w2.writtenMsg.Answer) != 1 {
		t.Fatalf("expected valid response on cache hit")
	}
	if w2.writtenMsg.Id != 0x2222 {
		t.Errorf("expected response ID to match req2 (0x2222), got %x", w2.writtenMsg.Id)
	}

	stats := dnsCache.Stats()
	if stats.Hits != 1 || stats.Misses != 1 {
		t.Errorf("expected Hits=1, Misses=1, got %+v", stats)
	}

	if len(recordedDecisions) != 2 {
		t.Fatalf("expected 2 recorded decisions, got %d", len(recordedDecisions))
	}
	secondDecision := recordedDecisions[1]
	if secondDecision.UpstreamLatencyMs != 0 {
		t.Errorf("expected 0ms latency for cache hit, got %d", secondDecision.UpstreamLatencyMs)
	}
	if secondDecision.Reason != "cache hit" {
		t.Errorf("expected 'cache hit' reason, got %q", secondDecision.Reason)
	}
}

func TestHandler_BlockRuleOverridesCache(t *testing.T) {
	fakeResolver := dnsengine.NewFakeResolver(func(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
		resp := new(dns.Msg)
		resp.SetReply(req)
		resp.Rcode = dns.RcodeSuccess
		resp.Answer = append(resp.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: req.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
			A:   net.ParseIP("1.2.3.4"),
		})
		return resp, nil
	})

	dnsCache := cache.NewDNSCache(cache.Config{MaxEntries: 100})
	matcher := filters.NewMatcher()

	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Matcher:     matcher,
		Resolver:    fakeResolver,
		Cache:       dnsCache,
		SinkholeTTL: 60,
	})

	// 1. Initial query: domain is allowed and cached
	req := new(dns.Msg)
	req.SetQuestion("adservice.com.", dns.TypeA)
	w1 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w1, req)

	if w1.writtenMsg.Answer[0].(*dns.A).A.String() != "1.2.3.4" {
		t.Fatalf("expected 1.2.3.4, got %s", w1.writtenMsg.Answer[0].(*dns.A).A.String())
	}

	// 2. Block rule activated dynamically
	matcher.BuildFromRules([]filters.Rule{
		{Domain: "adservice.com", Action: filters.ActionBlock, Enabled: true, Source: "new_blocklist"},
	})

	// 3. Subsequent query: must be blocked, NOT served from cache!
	w2 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w2, req)

	if w2.writtenMsg.Answer[0].(*dns.A).A.String() != "0.0.0.0" {
		t.Fatalf("new block rule was bypassed by cache! expected 0.0.0.0, got %s", w2.writtenMsg.Answer[0].(*dns.A).A.String())
	}
}

func TestHandler_DoNotCacheSERVFAIL(t *testing.T) {
	var calls atomic.Int32
	fakeResolver := dnsengine.NewFakeResolver(func(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
		calls.Add(1)
		return nil, errors.New("upstream timeout")
	})

	dnsCache := cache.NewDNSCache(cache.Config{MaxEntries: 100})
	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Resolver: fakeResolver,
		Cache:    dnsCache,
	})

	req := new(dns.Msg)
	req.SetQuestion("fail.example.com.", dns.TypeA)

	w1 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w1, req)

	if w1.writtenMsg.Rcode != dns.RcodeServerFailure {
		t.Fatalf("expected SERVFAIL, got %d", w1.writtenMsg.Rcode)
	}

	if dnsCache.Stats().Size != 0 {
		t.Fatalf("expected cache size 0 after SERVFAIL, got %d", dnsCache.Stats().Size)
	}

	// Second query should invoke resolver again
	w2 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w2, req)

	if calls.Load() != 2 {
		t.Fatalf("expected 2 resolver calls (SERVFAIL not cached), got %d", calls.Load())
	}
}

func TestHandler_CacheSeparatesQtypes(t *testing.T) {
	fakeResolver := dnsengine.NewFakeResolver(func(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
		resp := new(dns.Msg)
		resp.SetReply(req)
		resp.Rcode = dns.RcodeSuccess
		if req.Question[0].Qtype == dns.TypeA {
			resp.Answer = append(resp.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: req.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
				A:   net.ParseIP("1.1.1.1"),
			})
		} else if req.Question[0].Qtype == dns.TypeAAAA {
			resp.Answer = append(resp.Answer, &dns.AAAA{
				Hdr:  dns.RR_Header{Name: req.Question[0].Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 300},
				AAAA: net.ParseIP("2606:4700:4700::1111"),
			})
		}
		return resp, nil
	})

	dnsCache := cache.NewDNSCache(cache.Config{MaxEntries: 100})
	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Resolver: fakeResolver,
		Cache:    dnsCache,
	})

	reqA := new(dns.Msg)
	reqA.SetQuestion("example.com.", dns.TypeA)
	wA := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(wA, reqA)

	reqAAAA := new(dns.Msg)
	reqAAAA.SetQuestion("example.com.", dns.TypeAAAA)
	wAAAA := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(wAAAA, reqAAAA)

	if dnsCache.Stats().Size != 2 {
		t.Fatalf("expected 2 distinct cache entries for A and AAAA, got %d", dnsCache.Stats().Size)
	}
}
