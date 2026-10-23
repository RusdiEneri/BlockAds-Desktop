package dnsengine_test

import (
	"context"
	"testing"
	"time"

	"github.com/miekg/dns"

	"blockads-desktop/internal/dnsengine"
	"blockads-desktop/internal/filters"
)

func TestServer_LifecycleAndRealQueries(t *testing.T) {
	matcher := filters.NewMatcher()
	matcher.BuildFromRules([]filters.Rule{
		{Domain: "blocked.local", Action: filters.ActionBlock, Enabled: true, Source: "test"},
	})

	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Matcher:     matcher,
		SinkholeTTL: 60,
	})

	// Bind to loopback with dynamic port (port 0)
	srv := dnsengine.NewServer(dnsengine.ServerConfig{
		ListenAddr: "127.0.0.1",
		Port:       0,
		Handler:    handler,
	})

	// Test Status before start
	if srv.IsRunning() {
		t.Fatalf("expected server not running initially")
	}

	err := srv.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		_ = srv.Shutdown(context.Background())
	}()

	if !srv.IsRunning() {
		t.Fatalf("expected server running after Start()")
	}

	addr := srv.Addr()
	if addr == "" {
		t.Fatalf("expected non-empty server address")
	}

	// Test 1: Real UDP query against the server
	udpClient := &dns.Client{Net: "udp", Timeout: 2 * time.Second}
	reqUDP := new(dns.Msg)
	reqUDP.SetQuestion("blocked.local.", dns.TypeA)

	respUDP, _, err := udpClient.Exchange(reqUDP, addr)
	if err != nil {
		t.Fatalf("UDP query failed: %v", err)
	}
	if respUDP.Rcode != dns.RcodeSuccess {
		t.Errorf("expected UDP NOERROR, got %d", respUDP.Rcode)
	}
	if len(respUDP.Answer) != 1 {
		t.Fatalf("expected 1 answer in UDP response, got %d", len(respUDP.Answer))
	}
	aRecordUDP, ok := respUDP.Answer[0].(*dns.A)
	if !ok || aRecordUDP.A.String() != "0.0.0.0" {
		t.Errorf("expected 0.0.0.0 sinkhole in UDP response, got %v", respUDP.Answer[0])
	}

	// Test 2: Real TCP query against the exact same server address & port
	tcpClient := &dns.Client{Net: "tcp", Timeout: 2 * time.Second}
	reqTCP := new(dns.Msg)
	reqTCP.SetQuestion("blocked.local.", dns.TypeA)

	respTCP, _, err := tcpClient.Exchange(reqTCP, addr)
	if err != nil {
		t.Fatalf("TCP query failed: %v", err)
	}
	if respTCP.Rcode != dns.RcodeSuccess {
		t.Errorf("expected TCP NOERROR, got %d", respTCP.Rcode)
	}
	if len(respTCP.Answer) != 1 {
		t.Fatalf("expected 1 answer in TCP response, got %d", len(respTCP.Answer))
	}

	// Test 3: Duplicate start prevention
	err = srv.Start()
	if err == nil {
		t.Errorf("expected error when starting an already running server")
	}

	// Test 4: Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err = srv.Shutdown(ctx)
	if err != nil {
		t.Fatalf("failed to shutdown server: %v", err)
	}

	if srv.IsRunning() {
		t.Fatalf("expected server not running after Shutdown()")
	}

	// Verify query fails after shutdown
	shortClient := &dns.Client{Net: "udp", Timeout: 200 * time.Millisecond}
	_, _, err = shortClient.Exchange(reqUDP, addr)
	if err == nil {
		t.Errorf("expected query to fail after server shutdown")
	}
}

func TestServer_DefaultConfiguration(t *testing.T) {
	srv := dnsengine.NewServer(dnsengine.ServerConfig{})
	if srv.ListenAddr() != "127.0.0.1" {
		t.Errorf("expected default listen addr 127.0.0.1, got %s", srv.ListenAddr())
	}
	if srv.Port() != 5353 {
		t.Errorf("expected default port 5353, got %d", srv.Port())
	}
}

func TestServer_RejectNonLoopbackBinding(t *testing.T) {
	// Must reject non-loopback bindings by default (e.g. 0.0.0.0 or LAN IP)
	srv := dnsengine.NewServer(dnsengine.ServerConfig{
		ListenAddr: "0.0.0.0",
		Port:       5353,
	})

	err := srv.Start()
	if err == nil {
		_ = srv.Shutdown(context.Background())
		t.Fatalf("expected error binding to 0.0.0.0")
	}
}

func TestServer_IPv6LoopbackBinding(t *testing.T) {
	matcher := filters.NewMatcher()
	matcher.BuildFromRules([]filters.Rule{
		{Domain: "blocked.local", Action: filters.ActionBlock, Enabled: true, Source: "test"},
	})

	handler := dnsengine.NewHandler(dnsengine.HandlerConfig{
		Matcher:     matcher,
		SinkholeTTL: 60,
	})

	srv := dnsengine.NewServer(dnsengine.ServerConfig{
		ListenAddr: "::1",
		Port:       0,
		Handler:    handler,
	})

	err := srv.Start()
	if err != nil {
		t.Skipf("skipping IPv6 loopback test: %v", err)
	}
	defer func() {
		_ = srv.Shutdown(context.Background())
	}()

	addr := srv.Addr()
	if addr == "" {
		t.Fatalf("expected non-empty IPv6 server address")
	}

	udpClient := &dns.Client{Net: "udp", Timeout: 2 * time.Second}
	reqUDP := new(dns.Msg)
	reqUDP.SetQuestion("blocked.local.", dns.TypeAAAA)

	respUDP, _, err := udpClient.Exchange(reqUDP, addr)
	if err != nil {
		t.Fatalf("failed UDP exchange over IPv6 loopback: %v", err)
	}
	if len(respUDP.Answer) == 0 {
		t.Fatalf("expected sinkholed answer over IPv6")
	}
}
