package dnsengine_test

import (
	"context"
	"net"
	"sync"
	"testing"

	"github.com/miekg/dns"

	"blockads-desktop/internal/cache"
	"blockads-desktop/internal/dnsengine"
	"blockads-desktop/internal/filters"
	"blockads-desktop/internal/rewrites"
	"blockads-desktop/internal/safesearch"
)

func TestHandler_SafeSearch_CNAMEEnforcement(t *testing.T) {
	fakeResolver := dnsengine.NewFakeResolver(func(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
		resp := new(dns.Msg)
		resp.SetReply(req)
		resp.Rcode = dns.RcodeSuccess
		resp.Answer = append(resp.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: req.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
			A:   net.ParseIP("142.250.190.46"),
		})
		return resp, nil
	})

	reg := safesearch.DefaultRegistry()
	ssRules := reg.GenerateRewriteRules([]string{"google", "bing", "duckduckgo", "youtube"})

	rw := rewrites.NewRewriter()
	if err := rw.AddRules(ssRules); err != nil {
		t.Fatalf("failed to add SafeSearch rules: %v", err)
	}

	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Resolver: fakeResolver,
		Rewriter: rw,
	})

	// 1. Google SafeSearch test (A query)
	reqGoogle := new(dns.Msg)
	reqGoogle.SetQuestion("www.google.com.", dns.TypeA)
	w1 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w1, reqGoogle)

	if w1.writtenMsg == nil || len(w1.writtenMsg.Answer) != 1 {
		t.Fatalf("expected 1 answer for www.google.com")
	}
	cnameRR, ok := w1.writtenMsg.Answer[0].(*dns.CNAME)
	if !ok {
		t.Fatalf("expected CNAME record, got %T", w1.writtenMsg.Answer[0])
	}
	if cnameRR.Target != "forcesafesearch.google.com." {
		t.Fatalf("expected target forcesafesearch.google.com., got %s", cnameRR.Target)
	}

	// 2. Google SafeSearch test (AAAA query)
	reqGoogleAAAA := new(dns.Msg)
	reqGoogleAAAA.SetQuestion("www.google.com.", dns.TypeAAAA)
	wAAAA := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(wAAAA, reqGoogleAAAA)

	if wAAAA.writtenMsg == nil || len(wAAAA.writtenMsg.Answer) != 1 {
		t.Fatalf("expected 1 answer for www.google.com AAAA")
	}
	cnameAAAA, ok := wAAAA.writtenMsg.Answer[0].(*dns.CNAME)
	if !ok {
		t.Fatalf("expected CNAME record for AAAA, got %T", wAAAA.writtenMsg.Answer[0])
	}
	if cnameAAAA.Target != "forcesafesearch.google.com." {
		t.Fatalf("expected target forcesafesearch.google.com., got %s", cnameAAAA.Target)
	}

	// 3. Microsoft Bing Edge sidebar scenario
	reqBingEdge := new(dns.Msg)
	reqBingEdge.SetQuestion("edgeservices.bing.com.", dns.TypeA)
	w2 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w2, reqBingEdge)

	if w2.writtenMsg == nil || len(w2.writtenMsg.Answer) != 1 {
		t.Fatalf("expected 1 answer for edgeservices.bing.com")
	}
	bingRR, ok := w2.writtenMsg.Answer[0].(*dns.CNAME)
	if !ok {
		t.Fatalf("expected CNAME record for edgeservices.bing.com, got %T", w2.writtenMsg.Answer[0])
	}
	if bingRR.Target != "strict.bing.com." {
		t.Fatalf("expected strict.bing.com., got %s", bingRR.Target)
	}

	// 4. DuckDuckGo SafeSearch test
	reqDDG := new(dns.Msg)
	reqDDG.SetQuestion("duckduckgo.com.", dns.TypeA)
	w3 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w3, reqDDG)

	if w3.writtenMsg == nil || len(w3.writtenMsg.Answer) != 1 {
		t.Fatalf("expected 1 answer for duckduckgo.com")
	}
	ddgRR, ok := w3.writtenMsg.Answer[0].(*dns.CNAME)
	if !ok {
		t.Fatalf("expected CNAME record for duckduckgo.com, got %T", w3.writtenMsg.Answer[0])
	}
	if ddgRR.Target != "safe.duckduckgo.com." {
		t.Fatalf("expected safe.duckduckgo.com., got %s", ddgRR.Target)
	}

	// 5. YouTube Restricted Mode test
	reqYT := new(dns.Msg)
	reqYT.SetQuestion("www.youtube.com.", dns.TypeA)
	w4 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w4, reqYT)

	if w4.writtenMsg == nil || len(w4.writtenMsg.Answer) != 1 {
		t.Fatalf("expected 1 answer for www.youtube.com")
	}
	ytRR, ok := w4.writtenMsg.Answer[0].(*dns.CNAME)
	if !ok {
		t.Fatalf("expected CNAME record for www.youtube.com, got %T", w4.writtenMsg.Answer[0])
	}
	if ytRR.Target != "restrict.youtube.com." {
		t.Fatalf("expected restrict.youtube.com., got %s", ytRR.Target)
	}
}

func TestHandler_SafeSearch_Precedence_UserRewriteOverridesSafeSearch(t *testing.T) {
	// User explicitly rewrites www.google.com to 192.168.1.100
	userRule := rewrites.Rule{
		Domain:  "www.google.com",
		Type:    "A",
		Target:  "192.168.1.100",
		TTL:     60,
		Enabled: true,
	}

	rw := rewrites.NewRewriter()
	// Add user rule first
	if err := rw.AddRules([]rewrites.Rule{userRule}); err != nil {
		t.Fatalf("failed to add user rule: %v", err)
	}

	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Rewriter: rw,
	})

	req := new(dns.Msg)
	req.SetQuestion("www.google.com.", dns.TypeA)
	w := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w, req)

	if w.writtenMsg == nil || len(w.writtenMsg.Answer) != 1 {
		t.Fatalf("expected 1 answer")
	}
	aRR, ok := w.writtenMsg.Answer[0].(*dns.A)
	if !ok {
		t.Fatalf("expected A record, got %T", w.writtenMsg.Answer[0])
	}
	if aRR.A.String() != "192.168.1.100" {
		t.Fatalf("expected user rewrite IP 192.168.1.100 to win, got %s", aRR.A.String())
	}
}

func TestHandler_SafeSearch_Precedence_SafeSearchOverridesBlocklist(t *testing.T) {
	// Suppose a blocklist rule exists for www.google.com
	matcher := filters.NewMatcher()
	matcher.BuildFromRules([]filters.Rule{
		{Domain: "www.google.com", Action: filters.ActionBlock, Enabled: true, Source: "test-blocklist"},
	})

	reg := safesearch.DefaultRegistry()
	ssRules := reg.GenerateRewriteRules([]string{"google"})

	rw := rewrites.NewRewriter()
	_ = rw.AddRules(ssRules)

	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Matcher:  matcher,
		Rewriter: rw,
	})

	req := new(dns.Msg)
	req.SetQuestion("www.google.com.", dns.TypeA)
	w := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w, req)

	if w.writtenMsg == nil || len(w.writtenMsg.Answer) != 1 {
		t.Fatalf("expected SafeSearch CNAME response, not sinkhole")
	}
	cnameRR, ok := w.writtenMsg.Answer[0].(*dns.CNAME)
	if !ok {
		t.Fatalf("expected CNAME record, got %T (sinkhole was erroneously triggered)", w.writtenMsg.Answer[0])
	}
	if cnameRR.Target != "forcesafesearch.google.com." {
		t.Fatalf("expected forcesafesearch.google.com., got %s", cnameRR.Target)
	}
}

func TestHandler_SafeSearch_CacheFlushInvalidation(t *testing.T) {
	fakeResolver := dnsengine.NewFakeResolver(func(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
		resp := new(dns.Msg)
		resp.SetReply(req)
		resp.Rcode = dns.RcodeSuccess
		resp.Answer = append(resp.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: req.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
			A:   net.ParseIP("142.250.190.46"),
		})
		return resp, nil
	})

	dnsCache := cache.NewDNSCache(cache.Config{MaxEntries: 100})
	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Resolver: fakeResolver,
		Cache:    dnsCache,
	})

	// 1. Resolve www.google.com before SafeSearch is enabled (cached upstream IP)
	req := new(dns.Msg)
	req.SetQuestion("www.google.com.", dns.TypeA)
	w1 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w1, req)

	if w1.writtenMsg == nil || len(w1.writtenMsg.Answer) != 1 {
		t.Fatalf("expected upstream answer")
	}

	// 2. Enable SafeSearch dynamically, update rewriter and flush cache
	reg := safesearch.DefaultRegistry()
	ssRules := reg.GenerateRewriteRules([]string{"google"})
	newRW := rewrites.NewRewriter()
	_ = newRW.AddRules(ssRules)

	handler.SetRewriter(newRW)
	handler.FlushCache()

	// 3. Query again: must return CNAME, NOT the previously cached IP
	w2 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w2, req)

	if w2.writtenMsg == nil || len(w2.writtenMsg.Answer) != 1 {
		t.Fatalf("expected CNAME response after cache flush")
	}
	cnameRR, ok := w2.writtenMsg.Answer[0].(*dns.CNAME)
	if !ok {
		t.Fatalf("expected CNAME record, got %T", w2.writtenMsg.Answer[0])
	}
	if cnameRR.Target != "forcesafesearch.google.com." {
		t.Fatalf("expected forcesafesearch.google.com., got %s", cnameRR.Target)
	}
}

func TestHandler_SafeSearch_ConcurrentAtomicHotSwap(t *testing.T) {
	reg := safesearch.DefaultRegistry()
	ssRules := reg.GenerateRewriteRules([]string{"google"})

	rw1 := rewrites.NewRewriter()
	_ = rw1.AddRules(ssRules)

	rw2 := rewrites.NewRewriter() // empty rewriter (disabled)

	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Rewriter: rw1,
	})

	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	// 10 concurrent query workers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := new(dns.Msg)
			req.SetQuestion("www.google.com.", dns.TypeA)
			for {
				select {
				case <-stopChan:
					return
				default:
					w := newMockResponseWriter("udp", "127.0.0.1:54321")
					handler.ServeDNS(w, req)
				}
			}
		}()
	}

	// Swapping rewriters rapidly
	for i := 0; i < 50; i++ {
		if i%2 == 0 {
			handler.SetRewriter(rw2)
		} else {
			handler.SetRewriter(rw1)
		}
	}

	close(stopChan)
	wg.Wait()
}
