package dnsengine_test

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"

	"github.com/miekg/dns"

	"blockads-desktop/internal/dnsengine"
	"blockads-desktop/internal/filters"
)

// mockResponseWriter implements dns.ResponseWriter for testing in-memory handlers.
type mockResponseWriter struct {
	remoteAddr net.Addr
	writtenMsg *dns.Msg
	mu         sync.Mutex
}

func newMockResponseWriter(network, addr string) *mockResponseWriter {
	var netAddr net.Addr
	if network == "tcp" {
		netAddr, _ = net.ResolveTCPAddr("tcp", addr)
	} else {
		netAddr, _ = net.ResolveUDPAddr("udp", addr)
	}
	return &mockResponseWriter{remoteAddr: netAddr}
}

func (m *mockResponseWriter) LocalAddr() net.Addr  { return m.remoteAddr }
func (m *mockResponseWriter) RemoteAddr() net.Addr { return m.remoteAddr }
func (m *mockResponseWriter) WriteMsg(msg *dns.Msg) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writtenMsg = msg
	return nil
}
func (m *mockResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (m *mockResponseWriter) Close() error                { return nil }
func (m *mockResponseWriter) TsigStatus() error           { return nil }
func (m *mockResponseWriter) TsigTimersOnly(bool)         {}
func (m *mockResponseWriter) Hijack()                     {}

func setupTestEngine(t *testing.T, resolver dnsengine.Resolver) (*dnsengine.Handler, *[]dnsengine.QueryDecision) {
	matcher := filters.NewMatcher()
	matcher.BuildFromRules([]filters.Rule{
		{Domain: "ads.example.com", Action: filters.ActionBlock, Enabled: true, Source: "blocklist"},
		{Domain: "tracker.org", Action: filters.ActionBlock, Enabled: true, Source: "blocklist"},
		{Domain: "safe.ads.example.com", Action: filters.ActionAllow, Enabled: true, Source: "allowlist"},
	})

	var decisions []dnsengine.QueryDecision
	var mu sync.Mutex
	observer := func(d dnsengine.QueryDecision) {
		mu.Lock()
		defer mu.Unlock()
		decisions = append(decisions, d)
	}

	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Matcher:       matcher,
		Resolver:      resolver,
		QueryObserver: observer,
		SinkholeTTL:   60,
	})

	return handler, &decisions
}

func TestHandler_Blocked_TypeA(t *testing.T) {
	handler, decisions := setupTestEngine(t, nil)

	req := new(dns.Msg)
	req.SetQuestion("ads.example.com.", dns.TypeA)
	req.Id = 0x1234

	w := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w, req)

	resp := w.writtenMsg
	if resp == nil {
		t.Fatalf("expected written response message")
	}

	if resp.Id != 0x1234 {
		t.Errorf("expected ID 0x1234, got %x", resp.Id)
	}
	if resp.Rcode != dns.RcodeSuccess {
		t.Errorf("expected NOERROR (0), got %d", resp.Rcode)
	}
	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 answer record, got %d", len(resp.Answer))
	}
	aRecord, ok := resp.Answer[0].(*dns.A)
	if !ok {
		t.Fatalf("expected *dns.A record, got %T", resp.Answer[0])
	}
	if aRecord.A.String() != "0.0.0.0" {
		t.Errorf("expected 0.0.0.0 sinkhole IP, got %s", aRecord.A.String())
	}
	if aRecord.Hdr.Ttl != 60 {
		t.Errorf("expected TTL 60, got %d", aRecord.Hdr.Ttl)
	}
	if resp.AuthenticatedData {
		t.Errorf("authenticated data (AD) flag must not be set on sinkhole response")
	}

	if len(*decisions) != 1 {
		t.Fatalf("expected 1 recorded decision, got %d", len(*decisions))
	}
	d := (*decisions)[0]
	if d.Decision != dnsengine.DecisionBlocked {
		t.Errorf("expected DecisionBlocked, got %s", d.Decision)
	}
	if d.Domain != "ads.example.com" {
		t.Errorf("expected domain ads.example.com, got %s", d.Domain)
	}
}

func TestHandler_Blocked_TypeAAAA(t *testing.T) {
	handler, _ := setupTestEngine(t, nil)

	req := new(dns.Msg)
	req.SetQuestion("ads.example.com.", dns.TypeAAAA)
	req.Id = 0x5678

	w := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w, req)

	resp := w.writtenMsg
	if resp == nil {
		t.Fatalf("expected response")
	}
	if resp.Rcode != dns.RcodeSuccess {
		t.Errorf("expected NOERROR, got %d", resp.Rcode)
	}
	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 answer record, got %d", len(resp.Answer))
	}
	aaaaRecord, ok := resp.Answer[0].(*dns.AAAA)
	if !ok {
		t.Fatalf("expected *dns.AAAA record, got %T", resp.Answer[0])
	}
	if aaaaRecord.AAAA.String() != "::" {
		t.Errorf("expected :: sinkhole IP, got %s", aaaaRecord.AAAA.String())
	}
	if aaaaRecord.Hdr.Ttl != 60 {
		t.Errorf("expected TTL 60, got %d", aaaaRecord.Hdr.Ttl)
	}
}

func TestHandler_Blocked_OtherTypes(t *testing.T) {
	handler, _ := setupTestEngine(t, nil)

	// Test HTTPS, TXT, MX queries on blocked domains return NOERROR with no answer
	queryTypes := []uint16{dns.TypeHTTPS, dns.TypeTXT, dns.TypeMX, dns.TypeCNAME}
	for _, qtype := range queryTypes {
		req := new(dns.Msg)
		req.SetQuestion("tracker.org.", qtype)

		w := newMockResponseWriter("udp", "127.0.0.1:54321")
		handler.ServeDNS(w, req)

		resp := w.writtenMsg
		if resp == nil {
			t.Fatalf("expected response for type %d", qtype)
		}
		if resp.Rcode != dns.RcodeSuccess {
			t.Errorf("type %d: expected NOERROR, got %d", qtype, resp.Rcode)
		}
		if len(resp.Answer) != 0 {
			t.Errorf("type %d: expected 0 fabricated answer records, got %d", qtype, len(resp.Answer))
		}
	}
}

func TestHandler_Allowed_WithResolver(t *testing.T) {
	fakeResolver := dnsengine.NewFakeResolver(func(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
		resp := new(dns.Msg)
		resp.SetReply(req)
		resp.Rcode = dns.RcodeSuccess
		resp.Answer = append(resp.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: req.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
			A:   net.ParseIP("93.184.216.34"),
		})
		return resp, nil
	})

	handler, decisions := setupTestEngine(t, fakeResolver)

	req := new(dns.Msg)
	req.SetQuestion("safe.ads.example.com.", dns.TypeA)
	req.Id = 0x9999

	w := newMockResponseWriter("tcp", "127.0.0.1:54321")
	handler.ServeDNS(w, req)

	resp := w.writtenMsg
	if resp == nil {
		t.Fatalf("expected response")
	}
	if resp.Rcode != dns.RcodeSuccess {
		t.Errorf("expected NOERROR, got %d", resp.Rcode)
	}
	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 answer record, got %d", len(resp.Answer))
	}
	aRec := resp.Answer[0].(*dns.A)
	if aRec.A.String() != "93.184.216.34" {
		t.Errorf("expected 93.184.216.34, got %s", aRec.A.String())
	}

	if len(*decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(*decisions))
	}
	d := (*decisions)[0]
	if d.Decision != dnsengine.DecisionAllowed {
		t.Errorf("expected DecisionAllowed, got %s", d.Decision)
	}
	if d.Protocol != "TCP" {
		t.Errorf("expected protocol TCP, got %s", d.Protocol)
	}
}

func TestHandler_Allowed_NilResolverReturnsSERVFAIL(t *testing.T) {
	// Zero plaintext fallback! If resolver is nil, must return SERVFAIL.
	handler, decisions := setupTestEngine(t, nil)

	req := new(dns.Msg)
	req.SetQuestion("allowed.example.com.", dns.TypeA)

	w := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w, req)

	resp := w.writtenMsg
	if resp == nil {
		t.Fatalf("expected response")
	}
	if resp.Rcode != dns.RcodeServerFailure {
		t.Errorf("expected SERVFAIL (%d), got %d", dns.RcodeServerFailure, resp.Rcode)
	}

	if len(*decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(*decisions))
	}
	d := (*decisions)[0]
	if d.Decision != dnsengine.DecisionFailed {
		t.Errorf("expected DecisionFailed, got %s", d.Decision)
	}
}

func TestHandler_Allowed_ResolverErrorReturnsSERVFAIL(t *testing.T) {
	fakeResolver := dnsengine.NewFakeResolver(func(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
		return nil, errors.New("upstream timeout")
	})

	handler, _ := setupTestEngine(t, fakeResolver)

	req := new(dns.Msg)
	req.SetQuestion("allowed.example.com.", dns.TypeA)

	w := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w, req)

	resp := w.writtenMsg
	if resp == nil {
		t.Fatalf("expected response")
	}
	if resp.Rcode != dns.RcodeServerFailure {
		t.Errorf("expected SERVFAIL (%d), got %d", dns.RcodeServerFailure, resp.Rcode)
	}
}

func TestHandler_Malformed_ZeroQuestions(t *testing.T) {
	handler, _ := setupTestEngine(t, nil)

	req := new(dns.Msg)
	req.Id = 0x1111
	// 0 questions

	w := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w, req)

	resp := w.writtenMsg
	if resp == nil {
		t.Fatalf("expected response")
	}
	if resp.Rcode != dns.RcodeFormatError {
		t.Errorf("expected FormatError (%d), got %d", dns.RcodeFormatError, resp.Rcode)
	}
}

func TestHandler_Malformed_MultipleQuestions(t *testing.T) {
	handler, _ := setupTestEngine(t, nil)

	req := new(dns.Msg)
	req.Id = 0x2222
	req.Question = []dns.Question{
		{Name: "a.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
		{Name: "b.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	}

	w := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w, req)

	resp := w.writtenMsg
	if resp == nil {
		t.Fatalf("expected response")
	}
	if resp.Rcode != dns.RcodeFormatError {
		t.Errorf("expected FormatError (%d), got %d", dns.RcodeFormatError, resp.Rcode)
	}
}

func TestHandler_Unsupported_OpcodeAndClass(t *testing.T) {
	handler, _ := setupTestEngine(t, nil)

	// Unsupported Opcode (e.g. OpcodeStatus)
	reqOp := new(dns.Msg)
	reqOp.Opcode = dns.OpcodeStatus
	reqOp.SetQuestion("example.com.", dns.TypeA)

	wOp := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(wOp, reqOp)
	if wOp.writtenMsg.Rcode != dns.RcodeNotImplemented {
		t.Errorf("expected NotImplemented for non-Query opcode, got %d", wOp.writtenMsg.Rcode)
	}

	// Unsupported Class (e.g. ClassCHAOS)
	reqClass := new(dns.Msg)
	reqClass.SetQuestion("example.com.", dns.TypeA)
	reqClass.Question[0].Qclass = dns.ClassCHAOS

	wClass := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(wClass, reqClass)
	if wClass.writtenMsg.Rcode != dns.RcodeNotImplemented {
		t.Errorf("expected NotImplemented for Chaos class, got %d", wClass.writtenMsg.Rcode)
	}
}

func TestHandler_AtomicSetMatcherHotReload(t *testing.T) {
	fakeResolver := dnsengine.NewFakeResolver(func(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
		resp := new(dns.Msg)
		resp.SetReply(req)
		resp.Rcode = dns.RcodeSuccess
		resp.Answer = append(resp.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: req.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
			A:   net.ParseIP("93.184.216.34"),
		})
		return resp, nil
	})

	initialMatcher := filters.NewMatcher()
	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Matcher:  initialMatcher,
		Resolver: fakeResolver,
	})

	// 1. Initial query: passes through to resolver
	req := new(dns.Msg)
	req.SetQuestion("blockme.com.", dns.TypeA)
	w1 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w1, req)

	if w1.writtenMsg == nil || len(w1.writtenMsg.Answer) != 1 {
		t.Fatalf("expected resolved answer")
	}
	if w1.writtenMsg.Answer[0].(*dns.A).A.String() != "93.184.216.34" {
		t.Errorf("expected 93.184.216.34, got %s", w1.writtenMsg.Answer[0].(*dns.A).A.String())
	}

	// 2. Hot-reload new matcher with block rule
	newMatcher := filters.NewMatcher()
	newMatcher.BuildFromRules([]filters.Rule{
		{Domain: "blockme.com", Action: filters.ActionBlock, Enabled: true, Source: "hot-reload-feed"},
	})
	handler.SetMatcher(newMatcher)

	// 3. Query again: immediately blocked and sinkholed to 0.0.0.0
	w2 := newMockResponseWriter("udp", "127.0.0.1:54321")
	handler.ServeDNS(w2, req)

	if w2.writtenMsg == nil || len(w2.writtenMsg.Answer) != 1 {
		t.Fatalf("expected blocked sinkhole answer")
	}
	if w2.writtenMsg.Answer[0].(*dns.A).A.String() != "0.0.0.0" {
		t.Errorf("expected sinkhole 0.0.0.0, got %s", w2.writtenMsg.Answer[0].(*dns.A).A.String())
	}
}
