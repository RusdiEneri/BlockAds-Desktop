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

func TestHandler_RFC9520_FailureCaching(t *testing.T) {
	var resolveCalls atomic.Int32
	mockClock := cache.NewMockClock(time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC))

	// Resolver initially fails, then recovers
	shouldFail := true
	fakeResolver := dnsengine.NewFakeResolver(func(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
		resolveCalls.Add(1)
		if shouldFail {
			return nil, errors.New("upstream connection reset")
		}
		resp := new(dns.Msg)
		resp.SetReply(req)
		resp.Rcode = dns.RcodeSuccess
		resp.Answer = append(resp.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: req.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
			A:   net.ParseIP("1.1.1.1"),
		})
		return resp, nil
	})

	dnsCache := cache.NewDNSCache(cache.Config{
		MaxEntries: 100,
		Clock:      mockClock,
	})

	failureCache := cache.NewFailureCache(5*time.Second, 100, mockClock)

	var recordedDecisions []dnsengine.QueryDecision
	observer := func(d dnsengine.QueryDecision) {
		recordedDecisions = append(recordedDecisions, d)
	}

	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Matcher:       filters.NewMatcher(),
		Resolver:      fakeResolver,
		QueryObserver: observer,
		Cache:         dnsCache,
		FailureCache:  failureCache,
	})

	// 1. First query fails -> calls upstream resolver
	req1 := new(dns.Msg)
	req1.SetQuestion("broken.example.com.", dns.TypeA)
	w1 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w1, req1)

	if resolveCalls.Load() != 1 {
		t.Fatalf("expected 1 resolver call, got %d", resolveCalls.Load())
	}
	if w1.writtenMsg.Rcode != dns.RcodeServerFailure {
		t.Fatalf("expected SERVFAIL, got %d", w1.writtenMsg.Rcode)
	}

	// 2. Second query within FailureTTL (5s) -> served from FailureCache without upstream call
	req2 := new(dns.Msg)
	req2.SetQuestion("broken.example.com.", dns.TypeA)
	w2 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w2, req2)

	if resolveCalls.Load() != 1 {
		t.Fatalf("expected resolver not to be called again during FailureTTL, calls=%d", resolveCalls.Load())
	}
	if w2.writtenMsg.Rcode != dns.RcodeServerFailure {
		t.Fatalf("expected cached SERVFAIL, got %d", w2.writtenMsg.Rcode)
	}

	if len(recordedDecisions) != 2 {
		t.Fatalf("expected 2 decisions recorded, got %d", len(recordedDecisions))
	}
	if recordedDecisions[1].UpstreamLatencyMs != 0 {
		t.Errorf("expected 0ms latency for cached failure, got %d", recordedDecisions[1].UpstreamLatencyMs)
	}

	// 3. Upstream recovers and clock advances past 5s FailureTTL
	shouldFail = false
	mockClock.Advance(6 * time.Second)

	req3 := new(dns.Msg)
	req3.SetQuestion("broken.example.com.", dns.TypeA)
	w3 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w3, req3)

	if resolveCalls.Load() != 2 {
		t.Fatalf("expected resolver to be called again after FailureTTL expired, calls=%d", resolveCalls.Load())
	}
	if w3.writtenMsg.Rcode != dns.RcodeSuccess {
		t.Fatalf("expected NOERROR after recovery, got %d", w3.writtenMsg.Rcode)
	}
}

func TestHandler_ResponseIntegrity_CasingAndEDNS(t *testing.T) {
	fakeResolver := dnsengine.NewFakeResolver(func(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
		resp := new(dns.Msg)
		resp.SetReply(req)
		resp.Rcode = dns.RcodeSuccess
		resp.Answer = append(resp.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: req.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
			A:   net.ParseIP("9.9.9.9"),
		})
		resp.SetEdns0(4096, false)
		return resp, nil
	})

	dnsCache := cache.NewDNSCache(cache.Config{MaxEntries: 100})
	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Matcher:  filters.NewMatcher(),
		Resolver: fakeResolver,
		Cache:    dnsCache,
	})

	// Prime cache with lowercase question
	req1 := new(dns.Msg)
	req1.SetQuestion("lookup.example.com.", dns.TypeA)
	req1.SetEdns0(4096, false)
	req1.Id = 0xAAAA

	w1 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w1, req1)

	// Second query with 0x20 mixed-case and NO EDNS
	req2 := new(dns.Msg)
	req2.SetQuestion("LoOkUp.ExAmPlE.cOm.", dns.TypeA)
	req2.Id = 0xBBBB
	// No EDNS set

	w2 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w2, req2)

	if w2.writtenMsg == nil {
		t.Fatalf("expected response on cache hit")
	}
	if w2.writtenMsg.Id != 0xBBBB {
		t.Errorf("expected transaction ID 0xBBBB, got 0x%x", w2.writtenMsg.Id)
	}
	if w2.writtenMsg.Question[0].Name != "LoOkUp.ExAmPlE.cOm." {
		t.Errorf("expected preserved question casing 'LoOkUp.ExAmPlE.cOm.', got %q", w2.writtenMsg.Question[0].Name)
	}
	if opt := w2.writtenMsg.IsEdns0(); opt != nil {
		t.Errorf("non-EDNS client received an OPT record in cached response")
	}
}
