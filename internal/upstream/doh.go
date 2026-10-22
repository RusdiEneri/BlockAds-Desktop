package upstream

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/miekg/dns"
)

const (
	// MediaTypeDNSMessage is the standard media type defined in RFC 8484.
	MediaTypeDNSMessage = "application/dns-message"

	// MaxDNSMessageSize is the maximum permitted size for DNS wire format messages (64KB).
	MaxDNSMessageSize = 65536
)

// Config configures the DoH resolver.
type Config struct {
	EndpointURL  string
	OverrideIP   string
	BootstrapDNS string
	Timeout      time.Duration
	RootCAs      *x509.CertPool
	Transport    http.RoundTripper
}

// DoHResolver implements dnsengine.Resolver using RFC 8484 DNS-over-HTTPS.
type DoHResolver struct {
	endpointURL string
	hostname    string
	client      *http.Client
	dialer      *EndpointDialer
	diagnostics *diagnosticsTracker
}

// NewDoHResolver creates a production DoH resolver.
func NewDoHResolver(cfg Config) (*DoHResolver, error) {
	rawURL := strings.TrimSpace(cfg.EndpointURL)
	if rawURL == "" {
		rawURL = "https://dns.caliph.dev/dns-query"
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid DoH endpoint URL %q: %w", rawURL, err)
	}

	if !strings.EqualFold(parsed.Scheme, "https") {
		return nil, fmt.Errorf("DoH endpoint must use the 'https://' scheme, got %q", parsed.Scheme)
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return nil, errors.New("DoH endpoint URL missing hostname")
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	dialer := NewEndpointDialer(hostname, cfg.OverrideIP, cfg.BootstrapDNS, timeout)
	tlsConfig := NewTLSConfig(hostname, cfg.RootCAs)

	var transport http.RoundTripper = cfg.Transport
	if transport == nil {
		transport = &http.Transport{
			DialContext:           dialer.DialContext,
			TLSClientConfig:       tlsConfig,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   20,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   timeout,
			ResponseHeaderTimeout: timeout,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("redirects are not permitted for DoH queries")
		},
	}

	diag := newDiagnosticsTracker(rawURL, hostname, dialer)

	return &DoHResolver{
		endpointURL: rawURL,
		hostname:    hostname,
		client:      client,
		dialer:      dialer,
		diagnostics: diag,
	}, nil
}

// Resolve resolves a DNS query over RFC 8484 DoH.
func (r *DoHResolver) Resolve(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
	if req == nil {
		return nil, errors.New("nil DNS request message")
	}
	if len(req.Question) == 0 {
		return nil, errors.New("DNS request contains no question")
	}

	start := time.Now()

	// Capture client's original transaction ID
	clientOriginalID := req.Id

	// RFC 8484 Section 4.1: Outbound ID SHOULD be set to 0 to maximize HTTP caching
	reqCopy := req.Copy()
	reqCopy.Id = 0

	wireData, err := reqCopy.Pack()
	if err != nil {
		r.diagnostics.recordFailure(err, 0)
		return nil, fmt.Errorf("failed packing DNS wire format: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpointURL, bytes.NewReader(wireData))
	if err != nil {
		r.diagnostics.recordFailure(err, 0)
		return nil, fmt.Errorf("failed creating HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", MediaTypeDNSMessage)
	httpReq.Header.Set("Accept", MediaTypeDNSMessage)

	resp, err := r.client.Do(httpReq)
	latency := int(time.Since(start).Milliseconds())

	if err != nil {
		r.diagnostics.recordFailure(err, latency)
		return nil, fmt.Errorf("upstream DoH request failed: %w", err)
	}
	defer func() {
		// Ensure response body is safely drained and closed
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
		_ = resp.Body.Close()
	}()

	// Validate HTTP status
	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		statusErr := fmt.Errorf("upstream DoH server returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
		r.diagnostics.recordFailure(statusErr, latency)
		return nil, statusErr
	}

	// Validate Content-Type
	ct := resp.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil || !strings.EqualFold(mediaType, MediaTypeDNSMessage) {
		ctErr := fmt.Errorf("unexpected upstream Content-Type %q (expected %s)", ct, MediaTypeDNSMessage)
		r.diagnostics.recordFailure(ctErr, latency)
		return nil, ctErr
	}

	// Read wire format response with bounded size limit (MaxDNSMessageSize + 1 to detect overflow)
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, MaxDNSMessageSize+1))
	if err != nil {
		r.diagnostics.recordFailure(err, latency)
		return nil, fmt.Errorf("error reading DoH response body: %w", err)
	}

	if len(bodyBytes) > MaxDNSMessageSize {
		sizeErr := fmt.Errorf("upstream DoH response exceeded maximum DNS message size (%d bytes)", MaxDNSMessageSize)
		r.diagnostics.recordFailure(sizeErr, latency)
		return nil, sizeErr
	}

	if len(bodyBytes) == 0 {
		emptyErr := errors.New("upstream DoH returned empty response body")
		r.diagnostics.recordFailure(emptyErr, latency)
		return nil, emptyErr
	}

	respMsg := new(dns.Msg)
	if err := respMsg.Unpack(bodyBytes); err != nil {
		unpackErr := fmt.Errorf("failed unpacking DNS response wire format: %w", err)
		r.diagnostics.recordFailure(unpackErr, latency)
		return nil, unpackErr
	}

	// Restore original client transaction ID
	respMsg.Id = clientOriginalID

	r.diagnostics.recordSuccess(latency)
	return respMsg, nil
}

// Diagnostics returns a snapshot of current upstream diagnostics.
func (r *DoHResolver) Diagnostics() Diagnostics {
	return r.diagnostics.snapshot()
}
