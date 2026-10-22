package dnsengine_test

import (
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"

	"blockads-desktop/internal/dnsengine"
	"blockads-desktop/internal/filters"
	"blockads-desktop/internal/upstream"
)

func TestHandler_Integration_WithRealDoHResolver(t *testing.T) {
	var dohHits uint64

	// 1. Create mock DoH TLS server
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&dohHits, 1)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}

		reqMsg := new(dns.Msg)
		if err := reqMsg.Unpack(body); err != nil {
			http.Error(w, "bad dns wire", http.StatusBadRequest)
			return
		}

		respMsg := new(dns.Msg)
		respMsg.SetReply(reqMsg)
		respMsg.Answer = append(respMsg.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: reqMsg.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 180},
			A:   net.ParseIP("198.51.100.1"),
		})

		wire, _ := respMsg.Pack()
		w.Header().Set("Content-Type", "application/dns-message")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(wire)
	}))
	defer srv.Close()

	certPool := x509.NewCertPool()
	certPool.AddCert(srv.Certificate())

	// 2. Instantiate production DoHResolver
	dohResolver, err := upstream.NewDoHResolver(upstream.Config{
		EndpointURL: srv.URL + "/dns-query",
		RootCAs:     certPool,
		Timeout:     2 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create DoH resolver: %v", err)
	}

	// 3. Configure filter matcher with block rules
	matcher := filters.NewMatcher()
	matcher.BuildFromRules([]filters.Rule{
		{Domain: "ads.tracker.net", Action: filters.ActionBlock, Enabled: true, Source: "blocklist"},
	})

	var decisions []dnsengine.QueryDecision
	observer := func(d dnsengine.QueryDecision) {
		decisions = append(decisions, d)
	}

	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Matcher:       matcher,
		Resolver:      dohResolver,
		QueryObserver: observer,
		SinkholeTTL:   60,
	})

	// Scenario A: Allowed domain routes through DoH resolver
	reqAllowed := new(dns.Msg)
	reqAllowed.SetQuestion("news.example.org.", dns.TypeA)
	reqAllowed.Id = 0x55AA

	wAllowed := newMockResponseWriter("udp", "127.0.0.1:51111")
	handler.ServeDNS(wAllowed, reqAllowed)

	respAllowed := wAllowed.writtenMsg
	if respAllowed == nil {
		t.Fatalf("expected written response for allowed query")
	}
	if respAllowed.Id != 0x55AA {
		t.Errorf("expected client transaction ID 0x55AA, got 0x%X", respAllowed.Id)
	}
	if respAllowed.Rcode != dns.RcodeSuccess {
		t.Errorf("expected NOERROR, got %d", respAllowed.Rcode)
	}
	if len(respAllowed.Answer) != 1 {
		t.Fatalf("expected 1 answer from DoH resolver, got %d", len(respAllowed.Answer))
	}
	if atomic.LoadUint64(&dohHits) != 1 {
		t.Errorf("expected exactly 1 DoH hit, got %d", atomic.LoadUint64(&dohHits))
	}

	// Scenario B: Blocked domain must sinkhole immediately without hitting DoH
	reqBlocked := new(dns.Msg)
	reqBlocked.SetQuestion("sub.ads.tracker.net.", dns.TypeA)
	reqBlocked.Id = 0x7788

	wBlocked := newMockResponseWriter("udp", "127.0.0.1:52222")
	handler.ServeDNS(wBlocked, reqBlocked)

	respBlocked := wBlocked.writtenMsg
	if respBlocked == nil {
		t.Fatalf("expected written response for blocked query")
	}
	if respBlocked.Id != 0x7788 {
		t.Errorf("expected client transaction ID 0x7788, got 0x%X", respBlocked.Id)
	}
	aRecord, ok := respBlocked.Answer[0].(*dns.A)
	if !ok || aRecord.A.String() != "0.0.0.0" {
		t.Errorf("expected 0.0.0.0 sinkhole, got %v", respBlocked.Answer[0])
	}
	// Verify DoH server was NOT called for blocked domain
	if atomic.LoadUint64(&dohHits) != 1 {
		t.Errorf("DoH server must not be contacted for blocked query; got hits: %d", atomic.LoadUint64(&dohHits))
	}

	// Scenario C: DoH failure returns SERVFAIL without plaintext fallback
	srv.Close() // Shut down upstream server so subsequent DoH calls fail

	reqFail := new(dns.Msg)
	reqFail.SetQuestion("other.example.com.", dns.TypeA)
	reqFail.Id = 0x9999

	wFail := newMockResponseWriter("udp", "127.0.0.1:53333")
	handler.ServeDNS(wFail, reqFail)

	respFail := wFail.writtenMsg
	if respFail == nil {
		t.Fatalf("expected written response on upstream failure")
	}
	if respFail.Rcode != dns.RcodeServerFailure {
		t.Errorf("expected SERVFAIL (%d), got %d", dns.RcodeServerFailure, respFail.Rcode)
	}
}
