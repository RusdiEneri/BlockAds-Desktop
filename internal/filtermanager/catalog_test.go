package filtermanager

import (
	"strings"
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

func TestExtractFilterMetadata(t *testing.T) {
	content := `# Title: StevenBlack Unified Hosts
# Description: Consolidated hosts file blocking adware, malware, and spam.
# Homepage: https://github.com/StevenBlack/hosts
# Version: 3.14.0
127.0.0.1 localhost
0.0.0.0 adservice.google.com
0.0.0.0 tracker.example.com
`
	meta, err := ExtractFilterMetadata(strings.NewReader(content), "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.Title != "StevenBlack Unified Hosts" {
		t.Errorf("expected title 'StevenBlack Unified Hosts', got %q", meta.Title)
	}
	if meta.Description != "Consolidated hosts file blocking adware, malware, and spam." {
		t.Errorf("expected description match, got %q", meta.Description)
	}
	if meta.Homepage != "https://github.com/StevenBlack/hosts" {
		t.Errorf("expected homepage match, got %q", meta.Homepage)
	}
	if meta.Format != "hosts" {
		t.Errorf("expected format 'hosts', got %q", meta.Format)
	}
	if meta.RuleCount != 2 {
		t.Errorf("expected 2 rules, got %d", meta.RuleCount)
	}
}

func TestExtractFilterMetadata_FallbackToFilename(t *testing.T) {
	content := `
# Simple unannotated list
0.0.0.0 tracking-test.com
0.0.0.0 metrics-test.com
`
	meta, err := ExtractFilterMetadata(strings.NewReader(content), "https://example.com/filters/malware_blocklist.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Falls back to filename without extension
	if meta.Title != "malware_blocklist" {
		t.Errorf("expected title fallback to filename 'malware_blocklist', got %q", meta.Title)
	}
}

func TestExtractFilterMetadata_RejectHTML(t *testing.T) {
	htmlContent := `<!DOCTYPE html>
<html>
<head><title>404 Not Found</title></head>
<body><h1>Page Not Found</h1></body>
</html>`

	_, err := ExtractFilterMetadata(strings.NewReader(htmlContent), "https://example.com/404.html")
	if err == nil {
		t.Fatal("expected error extracting metadata from HTML, got nil")
	}
	if !strings.Contains(err.Error(), "HTML") {
		t.Errorf("expected HTML rejection message, got: %v", err)
	}
}

func TestBuiltinCatalog_Integrity(t *testing.T) {
	catalog := BuiltinCatalog()
	if len(catalog) < 4 {
		t.Fatalf("expected at least 4 catalog items, got %d", len(catalog))
	}

	seenIDs := make(map[string]bool)
	seenURLs := make(map[string]bool)

	for _, item := range catalog {
		if item.ID == "" {
			t.Errorf("item missing ID: %+v", item)
		}
		if seenIDs[item.ID] {
			t.Errorf("duplicate catalog ID: %s", item.ID)
		}
		seenIDs[item.ID] = true

		if item.Name == "" || item.SourceURL == "" || item.Category == "" {
			t.Errorf("item missing required fields: %+v", item)
		}

		normURL, err := NormalizeURL(item.SourceURL)
		if err != nil {
			t.Errorf("invalid catalog URL for %s: %v", item.ID, err)
		}
		if seenURLs[normURL] {
			t.Errorf("duplicate catalog normalized URL: %s", normURL)
		}
		seenURLs[normURL] = true
	}
}
