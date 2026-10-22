package upstream_test

import (
	"context"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/miekg/dns"

	"blockads-desktop/internal/upstream"
)

// makeMockDoHServer creates an httptest.Server configured to respond like an RFC 8484 server.
func makeMockDoHServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *x509.CertPool) {
	srv := httptest.NewTLSServer(handler)

	certPool := x509.NewCertPool()
	certPool.AddCert(srv.Certificate())

	return srv, certPool
}

func TestDoHResolver_ValidA_And_IDNormalization(t *testing.T) {
	var receivedWireID uint16
	var receivedContentType string
	var receivedMethod string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedContentType = r.Header.Get("Content-Type")

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
		receivedWireID = reqMsg.Id

		respMsg := new(dns.Msg)
		respMsg.SetReply(reqMsg)
		respMsg.Answer = append(respMsg.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: reqMsg.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
			A:   net.ParseIP("93.184.216.34"),
		})

		wire, err := respMsg.Pack()
		if err != nil {
			http.Error(w, "pack error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/dns-message")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(wire)
	})

	srv, certPool := makeMockDoHServer(t, handler)
	defer srv.Close()

	resolver, err := upstream.NewDoHResolver(upstream.Config{
		EndpointURL: srv.URL + "/dns-query",
		RootCAs:     certPool,
		Timeout:     2 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	req := new(dns.Msg)
	req.SetQuestion("example.com.", dns.TypeA)
	req.Id = 0xABCD

	ctx := context.Background()
	resp, err := resolver.Resolve(ctx, req)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if receivedMethod != http.MethodPost {
		t.Errorf("expected POST method, got %s", receivedMethod)
	}
	if !strings.HasPrefix(receivedContentType, "application/dns-message") {
		t.Errorf("expected Content-Type application/dns-message, got %s", receivedContentType)
	}
	// Verify wire format normalized ID to 0 per RFC 8484 Section 4.1
	if receivedWireID != 0 {
		t.Errorf("expected wire format request ID normalized to 0, got 0x%X", receivedWireID)
	}
	// Verify original client transaction ID restored in response
	if resp.Id != 0xABCD {
		t.Errorf("expected original client ID 0xABCD restored, got 0x%X", resp.Id)
	}
	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 answer record, got %d", len(resp.Answer))
	}
	aRec, ok := resp.Answer[0].(*dns.A)
	if !ok || aRec.A.String() != "93.184.216.34" {
		t.Errorf("unexpected answer: %v", resp.Answer[0])
	}
}

func TestDoHResolver_ValidAAAA_And_HTTPS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		reqMsg := new(dns.Msg)
		_ = reqMsg.Unpack(body)

		respMsg := new(dns.Msg)
		respMsg.SetReply(reqMsg)

		if reqMsg.Question[0].Qtype == dns.TypeAAAA {
			respMsg.Answer = append(respMsg.Answer, &dns.AAAA{
				Hdr:  dns.RR_Header{Name: reqMsg.Question[0].Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 300},
				AAAA: net.ParseIP("2606:2800:220:1:248:1893:25c8:1946"),
			})
		}

		wire, _ := respMsg.Pack()
		w.Header().Set("Content-Type", "application/dns-message")
		_, _ = w.Write(wire)
	})

	srv, certPool := makeMockDoHServer(t, handler)
	defer srv.Close()

	resolver, err := upstream.NewDoHResolver(upstream.Config{
		EndpointURL: srv.URL + "/dns-query",
		RootCAs:     certPool,
	})
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	req := new(dns.Msg)
	req.SetQuestion("example.com.", dns.TypeAAAA)
	resp, err := resolver.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("failed resolving AAAA: %v", err)
	}
	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 AAAA answer record, got %d", len(resp.Answer))
	}
}

func TestDoHResolver_DNSStatusNXDOMAIN(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		reqMsg := new(dns.Msg)
		_ = reqMsg.Unpack(body)

		respMsg := new(dns.Msg)
		respMsg.SetRcode(reqMsg, dns.RcodeNameError)

		wire, _ := respMsg.Pack()
		w.Header().Set("Content-Type", "application/dns-message")
		_, _ = w.Write(wire)
	})

	srv, certPool := makeMockDoHServer(t, handler)
	defer srv.Close()

	resolver, err := upstream.NewDoHResolver(upstream.Config{
		EndpointURL: srv.URL + "/dns-query",
		RootCAs:     certPool,
	})
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	req := new(dns.Msg)
	req.SetQuestion("nonexistent.invalid.", dns.TypeA)
	resp, err := resolver.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("expected successful resolution with NXDOMAIN rcode, got error: %v", err)
	}
	if resp.Rcode != dns.RcodeNameError {
		t.Errorf("expected NXDOMAIN (3), got %d", resp.Rcode)
	}
}

func TestDoHResolver_Non200HTTPError(t *testing.T) {
	statuses := []int{http.StatusBadRequest, http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway}
	for _, status := range statuses {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "upstream rate limited or broken", status)
		})

		srv, certPool := makeMockDoHServer(t, handler)
		resolver, _ := upstream.NewDoHResolver(upstream.Config{
			EndpointURL: srv.URL + "/dns-query",
			RootCAs:     certPool,
		})

		req := new(dns.Msg)
		req.SetQuestion("example.com.", dns.TypeA)
		_, err := resolver.Resolve(context.Background(), req)
		srv.Close()

		if err == nil {
			t.Errorf("status %d: expected error, got nil", status)
		}
	}
}

func TestDoHResolver_UnexpectedContentType(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html>Error</html>"))
	})

	srv, certPool := makeMockDoHServer(t, handler)
	defer srv.Close()

	resolver, _ := upstream.NewDoHResolver(upstream.Config{
		EndpointURL: srv.URL + "/dns-query",
		RootCAs:     certPool,
	})

	req := new(dns.Msg)
	req.SetQuestion("example.com.", dns.TypeA)
	_, err := resolver.Resolve(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error on unexpected content-type")
	}
}

func TestDoHResolver_EmptyAndMalformedBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/dns-message")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte{0xDE, 0xAD, 0xBE}) // Malformed truncated wire
	})

	srv, certPool := makeMockDoHServer(t, handler)
	defer srv.Close()

	resolver, _ := upstream.NewDoHResolver(upstream.Config{
		EndpointURL: srv.URL + "/dns-query",
		RootCAs:     certPool,
	})

	req := new(dns.Msg)
	req.SetQuestion("example.com.", dns.TypeA)
	_, err := resolver.Resolve(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error on malformed wire format body")
	}
}

func TestDoHResolver_OversizedBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/dns-message")
		w.WriteHeader(http.StatusOK)
		// Write 70KB
		oversized := make([]byte, 70000)
		_, _ = w.Write(oversized)
	})

	srv, certPool := makeMockDoHServer(t, handler)
	defer srv.Close()

	resolver, _ := upstream.NewDoHResolver(upstream.Config{
		EndpointURL: srv.URL + "/dns-query",
		RootCAs:     certPool,
	})

	req := new(dns.Msg)
	req.SetQuestion("example.com.", dns.TypeA)
	_, err := resolver.Resolve(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error on oversized response body")
	}
}

func TestDoHResolver_RedirectRejected(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://evil.attacker.com/dns-query", http.StatusMovedPermanently)
	})

	srv, certPool := makeMockDoHServer(t, handler)
	defer srv.Close()

	resolver, _ := upstream.NewDoHResolver(upstream.Config{
		EndpointURL: srv.URL + "/dns-query",
		RootCAs:     certPool,
	})

	req := new(dns.Msg)
	req.SetQuestion("example.com.", dns.TypeA)
	_, err := resolver.Resolve(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error rejecting HTTP redirects")
	}
}

func TestDoHResolver_ContextCancellation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.Header().Set("Content-Type", "application/dns-message")
		w.WriteHeader(http.StatusOK)
	})

	srv, certPool := makeMockDoHServer(t, handler)
	defer srv.Close()

	resolver, _ := upstream.NewDoHResolver(upstream.Config{
		EndpointURL: srv.URL + "/dns-query",
		RootCAs:     certPool,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := new(dns.Msg)
	req.SetQuestion("example.com.", dns.TypeA)
	_, err := resolver.Resolve(ctx, req)
	if err == nil {
		t.Fatalf("expected context deadline exceeded error")
	}
}

func TestDoHResolver_UntrustedCertificateFails(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/dns-message")
		w.WriteHeader(http.StatusOK)
	})

	// Server created, but we deliberately pass an EMPTY cert pool so verification fails
	srv, _ := makeMockDoHServer(t, handler)
	defer srv.Close()

	emptyPool := x509.NewCertPool()
	resolver, _ := upstream.NewDoHResolver(upstream.Config{
		EndpointURL: srv.URL + "/dns-query",
		RootCAs:     emptyPool,
	})

	req := new(dns.Msg)
	req.SetQuestion("example.com.", dns.TypeA)
	_, err := resolver.Resolve(context.Background(), req)
	if err == nil {
		t.Fatalf("expected TLS certificate verification error")
	}
}

func TestDoHResolver_ConcurrentQueries(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		reqMsg := new(dns.Msg)
		_ = reqMsg.Unpack(body)

		respMsg := new(dns.Msg)
		respMsg.SetReply(reqMsg)
		respMsg.Answer = append(respMsg.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: reqMsg.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
			A:   net.ParseIP("1.2.3.4"),
		})

		wire, _ := respMsg.Pack()
		w.Header().Set("Content-Type", "application/dns-message")
		_, _ = w.Write(wire)
	})

	srv, certPool := makeMockDoHServer(t, handler)
	defer srv.Close()

	resolver, _ := upstream.NewDoHResolver(upstream.Config{
		EndpointURL: srv.URL + "/dns-query",
		RootCAs:     certPool,
	})

	var wg sync.WaitGroup
	errCh := make(chan error, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id uint16) {
			defer wg.Done()
			req := new(dns.Msg)
			req.SetQuestion("concurrent.example.", dns.TypeA)
			req.Id = id

			resp, err := resolver.Resolve(context.Background(), req)
			if err != nil {
				errCh <- err
				return
			}
			if resp.Id != id {
				t.Errorf("expected ID 0x%X, got 0x%X", id, resp.Id)
			}
		}(uint16(i + 1))
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent query failed: %v", err)
	}

	diag := resolver.Diagnostics()
	if diag.TotalQueries != 20 {
		t.Errorf("expected 20 recorded queries, got %d", diag.TotalQueries)
	}
	if diag.FailedQueries != 0 {
		t.Errorf("expected 0 failed queries, got %d", diag.FailedQueries)
	}
}

func TestDoHResolver_LiveConnectivity_Optional(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live connectivity test in short mode")
	}

	resolver, err := upstream.NewDoHResolver(upstream.Config{
		EndpointURL: "https://dns.caliph.dev/dns-query",
		Timeout:     4 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create live resolver: %v", err)
	}

	req := new(dns.Msg)
	req.SetQuestion("cloudflare.com.", dns.TypeA)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	resp, err := resolver.Resolve(ctx, req)
	if err != nil {
		t.Skipf("skipping live test due to external network availability: %v", err)
		return
	}

	if resp.Rcode != dns.RcodeSuccess {
		t.Errorf("expected NOERROR from live upstream, got %d", resp.Rcode)
	}
	t.Logf("Live DoH query to dns.caliph.dev succeeded with %d answer records", len(resp.Answer))
}
