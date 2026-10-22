package models_test

import (
	"testing"

	"blockads-desktop/internal/models"
)

func TestValidateDomain(t *testing.T) {
	validCases := []string{
		"example.com",
		"sub.example.com",
		"tracker.ads.doubleclick.net",
		"dns.caliph.dev",
		"my-domain123.co.id",
		"localhost",
	}

	for _, domain := range validCases {
		if err := models.ValidateDomain(domain); err != nil {
			t.Errorf("expected %q to be valid, got error: %v", domain, err)
		}
	}

	invalidCases := []struct {
		domain string
		reason string
	}{
		{"", "empty string"},
		{"   ", "whitespace"},
		{"http://example.com", "includes protocol scheme"},
		{"https://example.com", "includes https scheme"},
		{"example.com/path", "includes path"},
		{"example.com:8080", "includes port"},
		{"*.example.com", "wildcards not allowed"},
		{"example", "single label without TLD"},
		{"-badlabel.com", "starts with hyphen"},
	}

	for _, tc := range invalidCases {
		if err := models.ValidateDomain(tc.domain); err == nil {
			t.Errorf("expected %q (%s) to fail validation, but it passed", tc.domain, tc.reason)
		}
	}
}

func TestValidateUpstreamDoHURL(t *testing.T) {
	validURLs := []string{
		"https://dns.caliph.dev/dns-query",
		"https://cloudflare-dns.com/dns-query",
		"https://dns.google/dns-query",
		"https://adguard-dns.com/resolve",
	}

	for _, u := range validURLs {
		if err := models.ValidateUpstreamDoHURL(u); err != nil {
			t.Errorf("expected %q to be valid DoH URL, got error: %v", u, err)
		}
	}

	invalidURLs := []struct {
		url    string
		reason string
	}{
		{"", "empty"},
		{"http://dns.caliph.dev/dns-query", "insecure http"},
		{"https://dns.caliph.dev", "missing query path"},
		{"https://dns.caliph.dev/", "root path only"},
		{"not-a-url", "malformed"},
	}

	for _, tc := range invalidURLs {
		if err := models.ValidateUpstreamDoHURL(tc.url); err == nil {
			t.Errorf("expected %q (%s) to fail validation, but it passed", tc.url, tc.reason)
		}
	}
}
