package dnsengine_test

import (
	"context"
	"net"
	"testing"

	"github.com/miekg/dns"

	"blockads-desktop/internal/cache"
	"blockads-desktop/internal/dnsengine"
	"blockads-desktop/internal/filters"
	"blockads-desktop/internal/rewrites"
)

func TestHandler_DNS_Rewrites_BasicAndPrecedenceOverCache(t *testing.T) {
	fakeResolver := dnsengine.NewFakeResolver(func(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
		resp := new(dns.Msg)
		resp.SetReply(req)
		resp.Rcode = dns.RcodeSuccess
		resp.Answer = append(resp.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: req.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
			A:   net.ParseIP("1.2.3.4"), // upstream IP
		})
		return resp, nil
	})

	dnsCache := cache.NewDNSCache(cache.Config{MaxEntries: 100})
	rewriter := rewrites.NewRewriter()
	matcher := filters.NewMatcher()

	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Matcher:  matcher,
		Resolver: fakeResolver,
		Cache:    dnsCache,
		Rewriter: rewriter,
	})

	// 1. Initial query without rewrite: resolved upstream and cached (1.2.3.4)
	req := new(dns.Msg)
	req.SetQuestion("nas.home.arpa.", dns.TypeA)
	w1 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w1, req)

	if w1.writtenMsg == nil || len(w1.writtenMsg.Answer) != 1 {
		t.Fatalf("expected initial upstream response")
	}
	if w1.writtenMsg.Answer[0].(*dns.A).A.String() != "1.2.3.4" {
		t.Fatalf("expected 1.2.3.4, got %s", w1.writtenMsg.Answer[0].(*dns.A).A.String())
	}

	// 2. Add rewrite rule: nas.home.arpa -> 192.168.18.11
	err := rewriter.AddRules([]rewrites.Rule{
		{Domain: "nas.home.arpa", Type: "A", Target: "192.168.18.11", Enabled: true, TTL: 60},
	})
	if err != nil {
		t.Fatalf("failed to add rewrite rule: %v", err)
	}

	// 3. Query again: must return 192.168.18.11, NOT the cached upstream answer 1.2.3.4!
	w2 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w2, req)

	if w2.writtenMsg == nil || len(w2.writtenMsg.Answer) != 1 {
		t.Fatalf("expected rewritten response")
	}
	actualIP := w2.writtenMsg.Answer[0].(*dns.A).A.String()
	if actualIP != "192.168.18.11" {
		t.Fatalf("rewrite rule was bypassed by cache! expected 192.168.18.11, got %s", actualIP)
	}
}

func TestHandler_DNS_Rewrites_PrecedenceOverBlocklist(t *testing.T) {
	matcher := filters.NewMatcher()
	matcher.BuildFromRules([]filters.Rule{
		{Domain: "local-service.com", Action: filters.ActionBlock, Enabled: true, Source: "adblock"},
	})

	rewriter := rewrites.NewRewriter()
	rewriter.AddRules([]rewrites.Rule{
		{Domain: "local-service.com", Type: "A", Target: "10.0.0.50", Enabled: true},
	})

	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Matcher:  matcher,
		Rewriter: rewriter,
	})

	req := new(dns.Msg)
	req.SetQuestion("local-service.com.", dns.TypeA)
	w := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w, req)

	if w.writtenMsg == nil || len(w.writtenMsg.Answer) != 1 {
		t.Fatalf("expected response")
	}
	actualIP := w.writtenMsg.Answer[0].(*dns.A).A.String()
	if actualIP != "10.0.0.50" {
		t.Fatalf("expected local rewrite IP 10.0.0.50 to override blocklist, got %s", actualIP)
	}
}

func TestHandler_DNS_Rewrites_WildcardIntegration(t *testing.T) {
	rewriter := rewrites.NewRewriter()
	rewriter.AddRules([]rewrites.Rule{
		{Domain: "*.lab.home.arpa", Type: "A", Target: "10.10.10.10", Enabled: true},
	})

	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Rewriter: rewriter,
	})

	req := new(dns.Msg)
	req.SetQuestion("grafana.internal.lab.home.arpa.", dns.TypeA)
	w := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w, req)

	if w.writtenMsg == nil || len(w.writtenMsg.Answer) != 1 {
		t.Fatalf("expected response for wildcard query")
	}
	actualIP := w.writtenMsg.Answer[0].(*dns.A).A.String()
	if actualIP != "10.10.10.10" {
		t.Fatalf("expected 10.10.10.10, got %s", actualIP)
	}
}
