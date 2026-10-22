package upstream

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// EndpointDialer routes TCP connections for a specific DoH endpoint hostname
// to an explicit override IP or a bootstrap-resolved IP, while preserving the
// original logical hostname for TLS SNI and HTTP Host verification.
type EndpointDialer struct {
	Hostname     string
	OverrideIP   string
	BootstrapDNS string
	Timeout      time.Duration

	mu          sync.RWMutex
	cachedIP    string
	cachedUntil time.Time
	dialFn      func(ctx context.Context, network, addr string) (net.Conn, error)
}

// NewEndpointDialer creates a new EndpointDialer.
func NewEndpointDialer(hostname, overrideIP, bootstrapDNS string, timeout time.Duration) *EndpointDialer {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	cleanHost := strings.ToLower(strings.TrimSpace(hostname))
	cleanOverride := strings.TrimSpace(overrideIP)

	// If hostname is already an IP literal, direct connection target is the IP itself
	if cleanOverride == "" && net.ParseIP(cleanHost) != nil {
		cleanOverride = cleanHost
	}

	bootstrap := strings.TrimSpace(bootstrapDNS)
	if bootstrap != "" && !strings.Contains(bootstrap, ":") {
		bootstrap = net.JoinHostPort(bootstrap, "53")
	}

	return &EndpointDialer{
		Hostname:     cleanHost,
		OverrideIP:   cleanOverride,
		BootstrapDNS: bootstrap,
		Timeout:      timeout,
	}
}

// CurrentIP returns the currently active target IP (override or cached bootstrap).
func (d *EndpointDialer) CurrentIP() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.OverrideIP != "" {
		return d.OverrideIP
	}
	return d.cachedIP
}

// InvalidateCache clears the cached bootstrap IP.
func (d *EndpointDialer) InvalidateCache() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cachedIP = ""
	d.cachedUntil = time.Time{}
}

// ResolveBootstrap queries an external bootstrap DNS server to look up the DoH hostname's IP.
// It explicitly never queries through the local loopback DNS engine.
func (d *EndpointDialer) ResolveBootstrap(ctx context.Context) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Return cached IP if still fresh
	if d.cachedIP != "" && time.Now().Before(d.cachedUntil) {
		return d.cachedIP, nil
	}

	bootstrapTarget := d.BootstrapDNS
	if bootstrapTarget == "" {
		bootstrapTarget = "1.1.1.1:53"
	}

	client := &dns.Client{
		Net:     "udp",
		Timeout: d.Timeout,
	}

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(d.Hostname), dns.TypeA)
	msg.RecursionDesired = true

	resp, _, err := client.ExchangeContext(ctx, msg, bootstrapTarget)
	if err != nil {
		return "", fmt.Errorf("bootstrap DNS lookup failed for %s via %s: %w", d.Hostname, bootstrapTarget, err)
	}

	if resp == nil || resp.Rcode != dns.RcodeSuccess {
		rcodeStr := "NIL"
		if resp != nil {
			rcodeStr = dns.RcodeToString[resp.Rcode]
		}
		return "", fmt.Errorf("bootstrap DNS lookup returned rcode %s for %s", rcodeStr, d.Hostname)
	}

	for _, ans := range resp.Answer {
		if aRecord, ok := ans.(*dns.A); ok && aRecord.A != nil {
			ipStr := aRecord.A.String()
			ttl := time.Duration(aRecord.Hdr.Ttl) * time.Second
			if ttl < 60*time.Second {
				ttl = 60 * time.Second
			}
			if ttl > 10*time.Minute {
				ttl = 10 * time.Minute
			}

			d.cachedIP = ipStr
			d.cachedUntil = time.Now().Add(ttl)
			return ipStr, nil
		}
	}

	return "", fmt.Errorf("no A records returned for bootstrap hostname %s", d.Hostname)
}

// DialContext intercepts connection requests matching d.Hostname and connects to
// the OverrideIP or cached/resolved bootstrap IP instead of performing system DNS resolution.
func (d *EndpointDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if !strings.HasPrefix(network, "tcp") {
		return nil, fmt.Errorf("unsupported network %q: DoH dialer only supports TCP", network)
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid dial address %q: %w", addr, err)
	}

	targetAddr := addr
	if strings.EqualFold(host, d.Hostname) {
		targetIP := d.OverrideIP
		if targetIP == "" {
			var resolveErr error
			targetIP, resolveErr = d.ResolveBootstrap(ctx)
			if resolveErr != nil {
				return nil, resolveErr
			}
		}
		targetAddr = net.JoinHostPort(targetIP, port)
	}

	if d.dialFn != nil {
		return d.dialFn(ctx, network, targetAddr)
	}

	dialer := &net.Dialer{
		Timeout: d.Timeout,
	}

	conn, err := dialer.DialContext(ctx, network, targetAddr)
	if err != nil && d.OverrideIP == "" && d.cachedIP != "" {
		// If dial failed with a cached IP, invalidate and try one fresh bootstrap resolution
		d.InvalidateCache()
		freshIP, freshErr := d.ResolveBootstrap(ctx)
		if freshErr == nil && freshIP != "" {
			return dialer.DialContext(ctx, network, net.JoinHostPort(freshIP, port))
		}
	}

	return conn, err
}
