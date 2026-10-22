package models

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// NormalizeURL canonicalizes a filter list URL for stable duplicate detection across sources.
// It lowercases scheme and host, strips default HTTPS port :443, removes URL fragments,
// normalizes root path, and strictly rejects embedded user credentials.
func NormalizeURL(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", errors.New("URL cannot be empty")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" {
		return "", errors.New("source URL must use secure HTTPS scheme (RFC 9110)")
	}

	if parsed.User != nil {
		return "", errors.New("source URL must not contain embedded user credentials")
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return "", errors.New("source URL missing hostname")
	}

	port := parsed.Port()
	if port == "443" {
		port = ""
	}

	hostWithPort := host
	if port != "" {
		hostWithPort = fmt.Sprintf("%s:%s", host, port)
	}

	p := parsed.EscapedPath()
	if p == "" {
		p = "/"
	}

	result := fmt.Sprintf("%s://%s%s", scheme, hostWithPort, p)
	if parsed.RawQuery != "" {
		result += "?" + parsed.RawQuery
	}

	return result, nil
}
