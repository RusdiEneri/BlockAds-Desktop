package upstream_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"

	"blockads-desktop/internal/upstream"
)

func TestNewTLSConfig(t *testing.T) {
	cfg := upstream.NewTLSConfig("dns.caliph.dev", nil)
	if cfg == nil {
		t.Fatalf("expected non-nil tls.Config")
	}
	if cfg.ServerName != "dns.caliph.dev" {
		t.Errorf("expected ServerName dns.caliph.dev, got %s", cfg.ServerName)
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected MinVersion TLS 1.2 (%x), got %x", tls.VersionTLS12, cfg.MinVersion)
	}
	if cfg.InsecureSkipVerify {
		t.Errorf("InsecureSkipVerify must never be true")
	}
}

func TestEndpointDialer_StaticOverride(t *testing.T) {
	// Create a local TCP listener to act as target
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer ln.Close()

	tcpAddr := ln.Addr().(*net.TCPAddr)
	overrideIP := tcpAddr.IP.String()
	port := tcpAddr.Port

	// Dialer configured with hostname "dns.caliph.dev" and static override to our local listener
	dialer := upstream.NewEndpointDialer("dns.caliph.dev", overrideIP, "", 2*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Dialing "dns.caliph.dev:<port>" must route to overrideIP:port
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort("dns.caliph.dev", fmt.Sprintf("%d", port)))
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	if dialer.CurrentIP() != overrideIP {
		t.Errorf("expected current IP %s, got %s", overrideIP, dialer.CurrentIP())
	}
}

func TestEndpointDialer_RejectsNonTCP(t *testing.T) {
	dialer := upstream.NewEndpointDialer("dns.caliph.dev", "127.0.0.1", "", 2*time.Second)
	ctx := context.Background()

	_, err := dialer.DialContext(ctx, "udp", "127.0.0.1:53")
	if err == nil {
		t.Fatalf("expected error dialing non-TCP network")
	}
}

func TestEndpointDialer_ResolveBootstrapWithMockServer(t *testing.T) {
	// Spin up a local mock DNS server on UDP port 0 to serve bootstrap query
	udpConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind mock udp listener: %v", err)
	}
	defer udpConn.Close()

	mockServer := &dns.Server{
		PacketConn: udpConn,
		Handler: dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
			resp := new(dns.Msg)
			resp.SetReply(r)
			if r.Question[0].Name == "dns.caliph.dev." {
				resp.Answer = append(resp.Answer, &dns.A{
					Hdr: dns.RR_Header{Name: "dns.caliph.dev.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 120},
					A:   net.ParseIP("104.21.55.66"),
				})
			}
			_ = w.WriteMsg(resp)
		}),
	}
	go func() {
		_ = mockServer.ActivateAndServe()
	}()
	defer func() {
		_ = mockServer.Shutdown()
	}()

	bootstrapAddr := udpConn.LocalAddr().String()

	dialer := upstream.NewEndpointDialer("dns.caliph.dev", "", bootstrapAddr, 2*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ip, err := dialer.ResolveBootstrap(ctx)
	if err != nil {
		t.Fatalf("ResolveBootstrap failed: %v", err)
	}
	if ip != "104.21.55.66" {
		t.Errorf("expected 104.21.55.66, got %s", ip)
	}

	// Verify caching
	if dialer.CurrentIP() != "104.21.55.66" {
		t.Errorf("expected cached IP 104.21.55.66, got %s", dialer.CurrentIP())
	}
}
