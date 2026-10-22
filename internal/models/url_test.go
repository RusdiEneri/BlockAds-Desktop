package models

import (
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name        string
		rawURL      string
		expected    string
		expectError bool
	}{
		{
			name:     "standard HTTPS url",
			rawURL:   "https://adaway.org/hosts.txt",
			expected: "https://adaway.org/hosts.txt",
		},
		{
			name:     "casing in scheme and host",
			rawURL:   "HTTPS://AdAway.ORG/hosts.txt",
			expected: "https://adaway.org/hosts.txt",
		},
		{
			name:     "default port 443 stripped",
			rawURL:   "https://adaway.org:443/hosts.txt",
			expected: "https://adaway.org/hosts.txt",
		},
		{
			name:     "strip fragment",
			rawURL:   "https://adaway.org/hosts.txt#section-1",
			expected: "https://adaway.org/hosts.txt",
		},
		{
			name:     "root slash normalization",
			rawURL:   "https://basic.oisd.nl",
			expected: "https://basic.oisd.nl/",
		},
		{
			name:     "root slash preserved",
			rawURL:   "https://basic.oisd.nl/",
			expected: "https://basic.oisd.nl/",
		},
		{
			name:     "preserve meaningful query parameters",
			rawURL:   "https://pgl.yoyo.org/adservers/serverlist.php?hostformat=hosts&mimetype=plaintext",
			expected: "https://pgl.yoyo.org/adservers/serverlist.php?hostformat=hosts&mimetype=plaintext",
		},
		{
			name:        "reject embedded credentials",
			rawURL:      "https://user:pass@example.com/hosts.txt",
			expectError: true,
		},
		{
			name:        "reject non-HTTPS scheme",
			rawURL:      "http://example.com/hosts.txt",
			expectError: true,
		},
		{
			name:        "reject empty url",
			rawURL:      "   ",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, err := NormalizeURL(tt.rawURL)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tt.rawURL)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if normalized != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, normalized)
			}
		})
	}
}
