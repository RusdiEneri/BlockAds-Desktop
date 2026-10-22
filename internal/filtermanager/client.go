package filtermanager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"blockads-desktop/internal/storage"
)

// Standard downloader error definitions.
var (
	ErrInvalidURL        = errors.New("invalid filter list URL")
	ErrInsecureScheme    = errors.New("source URL must use HTTPS scheme")
	ErrSSRFBlocked       = errors.New("target IP address is restricted (private/loopback/link-local/multicast)")
	ErrRedirectDowngrade = errors.New("redirect attempted to downgrade from HTTPS to HTTP")
	ErrTooManyRedirects  = errors.New("too many redirects (exceeded limit of 5)")
	ErrResponseTooLarge  = errors.New("response body exceeds maximum allowed size")
)

const (
	DefaultTimeout      = 30 * time.Second
	DefaultMaxRedirects = 5
)

// DownloaderConfig configures the security-hardened HTTP client.
type DownloaderConfig struct {
	Timeout              time.Duration
	MaxBodyBytes         int64
	AllowPrivateNetworks bool
	MaxRedirects         int
	Transport            http.RoundTripper
}

// DownloadRequest parameters for an HTTP fetch.
type DownloadRequest struct {
	URL          string
	ETag         string
	LastModified string
}

// DownloadResponse contains the server response metadata and body.
type DownloadResponse struct {
	StatusCode    int
	Body          []byte
	ETag          string
	LastModified  string
	ContentLength int64
	Duration      time.Duration
	NotModified   bool
}

// Downloader manages secure HTTP fetching of filter list contents.
type Downloader struct {
	client       *http.Client
	maxBodyBytes int64
	allowPrivate bool
}

// isRestrictedIP checks if an IP belongs to loopback, private RFC1918, link-local, multicast, or unspecified.
func isRestrictedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	// Normalize IPv4-mapped IPv6
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}

	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

// ValidateSourceURL checks syntactic validity, enforces HTTPS, and prevents SSRF destinations.
func ValidateSourceURL(rawURL string, allowPrivate bool) error {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return ErrInvalidURL
	}

	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if !allowPrivate && scheme != "https" {
		return ErrInsecureScheme
	}
	if allowPrivate && scheme != "https" && scheme != "http" {
		return ErrInsecureScheme
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("%w: missing host", ErrInvalidURL)
	}

	// If host is an IP literal
	if ip := net.ParseIP(host); ip != nil {
		if !allowPrivate && isRestrictedIP(ip) {
			return ErrSSRFBlocked
		}
		return nil
	}

	// Resolve hostname to check for loopback/private destinations
	if !allowPrivate {
		ips, err := net.LookupIP(host)
		if err == nil {
			for _, ip := range ips {
				if isRestrictedIP(ip) {
					return ErrSSRFBlocked
				}
			}
		}
	}

	return nil
}

// NewDownloader creates a new Downloader with strict redirect and SSRF enforcement.
func NewDownloader(cfg DownloaderConfig) *Downloader {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	maxBytes := cfg.MaxBodyBytes
	if maxBytes <= 0 {
		maxBytes = storage.MaxSnapshotBytes
	}

	maxRedirects := cfg.MaxRedirects
	if maxRedirects <= 0 {
		maxRedirects = DefaultMaxRedirects
	}

	transport := cfg.Transport
	if transport == nil {
		dialer := &net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}

		// DialControl prevents DNS rebinding to private IPs if allowPrivate is false
		if !cfg.AllowPrivateNetworks {
			dialer.Control = func(network, address string, c syscall.RawConn) error {
				host, _, err := net.SplitHostPort(address)
				if err == nil {
					if ip := net.ParseIP(host); ip != nil && isRestrictedIP(ip) {
						return ErrSSRFBlocked
					}
				}
				return nil
			}
		}

		transport = &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialer.DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          50,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return ErrTooManyRedirects
			}

			// Reject HTTPS-to-HTTP downgrades
			if strings.ToLower(req.URL.Scheme) == "http" {
				for _, v := range via {
					if strings.ToLower(v.URL.Scheme) == "https" {
						return ErrRedirectDowngrade
					}
				}
			}

			if !cfg.AllowPrivateNetworks && strings.ToLower(req.URL.Scheme) != "https" {
				return ErrRedirectDowngrade
			}

			// Verify redirect destination
			if err := ValidateSourceURL(req.URL.String(), cfg.AllowPrivateNetworks); err != nil {
				return err
			}

			return nil
		},
	}

	return &Downloader{
		client:       client,
		maxBodyBytes: maxBytes,
		allowPrivate: cfg.AllowPrivateNetworks,
	}
}

// Download performs a conditional HTTP GET request.
func (d *Downloader) Download(ctx context.Context, req DownloadRequest) (*DownloadResponse, error) {
	if err := ValidateSourceURL(req.URL, d.allowPrivate); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", req.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Accept", "text/plain, text/x-hosts, */*")
	httpReq.Header.Set("User-Agent", "BlockAds-Desktop/0.1.0")

	// RFC 9110 conditional request headers
	if strings.TrimSpace(req.ETag) != "" {
		httpReq.Header.Set("If-None-Match", strings.TrimSpace(req.ETag))
	}
	if strings.TrimSpace(req.LastModified) != "" {
		httpReq.Header.Set("If-Modified-Since", strings.TrimSpace(req.LastModified))
	}

	start := time.Now()
	resp, err := d.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	respETag := resp.Header.Get("ETag")
	respLastModified := resp.Header.Get("Last-Modified")

	if resp.StatusCode == http.StatusNotModified {
		return &DownloadResponse{
			StatusCode:   http.StatusNotModified,
			ETag:         respETag,
			LastModified: respLastModified,
			Duration:     duration,
			NotModified:  true,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned HTTP status %d", resp.StatusCode)
	}

	// Bounded reading up to maxBodyBytes + 1
	limitReader := io.LimitReader(resp.Body, d.maxBodyBytes+1)
	body, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if int64(len(body)) > d.maxBodyBytes {
		return nil, ErrResponseTooLarge
	}

	return &DownloadResponse{
		StatusCode:    http.StatusOK,
		Body:          body,
		ETag:          respETag,
		LastModified:  respLastModified,
		ContentLength: int64(len(body)),
		Duration:      duration,
		NotModified:   false,
	}, nil
}
