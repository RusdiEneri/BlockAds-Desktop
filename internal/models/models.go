package models

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// AppInfo provides application runtime metadata.
type AppInfo struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	Platform        string `json:"platform"`
	Milestone       string `json:"milestone"`
	GoVersion       string `json:"goVersion"`
	WebView2Version string `json:"webView2Version"`
	DataDir         string `json:"dataDir"`
	IsPortable      bool   `json:"isPortable"`
}

// ProtectionStatus represents the state of DNS filtering.
type ProtectionStatus struct {
	IsActive    bool   `json:"isActive"`
	State       string `json:"state"` // e.g. "INACTIVE_MILESTONE_1"
	UpstreamURL string `json:"upstreamUrl"`
	ListenAddr  string `json:"listenAddr"`
	Message     string `json:"message"`
}

// QueryLogEntry models a single DNS resolution record.
type QueryLogEntry struct {
	ID                string `json:"id"`
	Timestamp         string `json:"timestamp"`
	Domain            string `json:"domain"`
	ClientIP          string `json:"clientIp"`
	QueryType         string `json:"queryType"` // "A", "AAAA", "HTTPS", "CNAME"
	Status            string `json:"status"`    // "ALLOWED", "BLOCKED", "FAILED"
	UpstreamLatencyMs int    `json:"upstreamLatencyMs"`
	FilterRule        string `json:"filterRule"`
	Reason            string `json:"reason"`
}

// FilterList describes an upstream filter list source.
type FilterList struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	SourceURL       string `json:"sourceUrl"`
	Enabled         bool   `json:"enabled"`
	Format          string `json:"format"`
	RuleCount       int    `json:"ruleCount"`
	LastUpdated     string `json:"lastUpdated"`
	LastAttemptAt   string `json:"lastAttemptAt"`
	LastSuccessAt   string `json:"lastSuccessAt"`
	LastStatus      string `json:"lastStatus"`
	LastError       string `json:"lastError"`
	ContentChecksum string `json:"contentChecksum"`
	ETag            string `json:"etag"`
	Description     string `json:"description"`
	Category        string `json:"category"`
}

// CatalogFilterItem represents an item from the curated filter catalog decorated with subscription status.
type CatalogFilterItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Category     string `json:"category"`
	SourceURL    string `json:"sourceUrl"`
	Format       string `json:"format"`
	Homepage     string `json:"homepage"`
	Maintainer   string `json:"maintainer"`
	License      string `json:"license"`
	IsSubscribed bool   `json:"isSubscribed"`
	SubscribedID string `json:"subscribedId,omitempty"`
}

// FilterPreviewResult holds metadata extracted when testing a filter URL before adding.
type FilterPreviewResult struct {
	SourceURL            string `json:"sourceUrl"`
	NormalizedURL        string `json:"normalizedUrl"`
	DetectedName         string `json:"detectedName"`
	Description          string `json:"description"`
	Homepage             string `json:"homepage"`
	Format               string `json:"format"`
	RuleCount            int    `json:"ruleCount"`
	IsAlreadySubscribed  bool   `json:"isAlreadySubscribed"`
	ExistingSubscription string `json:"existingSubscription,omitempty"`
	ExistingID           string `json:"existingId,omitempty"`
}

// StartupStatus provides diagnostics and state for 'Start with Windows'.
type StartupStatus struct {
	Supported      bool   `json:"supported"`
	Enabled        bool   `json:"enabled"`
	ExecutablePath string `json:"executablePath"`
	IsPathValid    bool   `json:"isPathValid"`
	Message        string `json:"message"`
}

// CustomRule represents an explicit user allowlist or blocklist entry.
type CustomRule struct {
	ID        string `json:"id"`
	Domain    string `json:"domain"`
	Action    string `json:"action"` // "BLOCK" or "ALLOW"
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"createdAt"`
	Notes     string `json:"notes"`
}

// AppSettings contains user configuration options.
type AppSettings struct {
	UpstreamDoHURL      string `json:"upstreamDohUrl"`
	BootstrapDNS        string `json:"bootstrapDns"`
	CacheEnabled        bool   `json:"cacheEnabled"`
	CacheSize           int    `json:"cacheSize"`
	AutoStart           bool   `json:"autoStart"`
	SystemDNSIntegrated bool   `json:"systemDnsIntegrated"`
}

// SafeSearchProviderSetting models a SafeSearch provider policy and its runtime state for UI inspection.
type SafeSearchProviderSetting struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Domains   []string `json:"domains"`
	Target    string   `json:"target"`
	DocURL    string   `json:"docUrl"`
	Enabled   bool     `json:"enabled"`
	Status    string   `json:"status"` // "ACTIVE", "DISABLED", "UNSUPPORTED", "PENDING"
	Warning   string   `json:"warning,omitempty"`
	UpdatedAt string   `json:"updatedAt,omitempty"`
}

// DiagnosticCheck represents a single diagnostic evaluation.
type DiagnosticCheck struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Status   string `json:"status"` // "PASS", "WARN", "FAIL", "INFO"
	Details  string `json:"details"`
}

var domainRegex = regexp.MustCompile(`^(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$`)

// ValidateDomain checks whether a string is a valid syntactical domain name.
func ValidateDomain(domain string) error {
	trimmed := strings.TrimSpace(domain)
	if trimmed == "" {
		return errors.New("domain cannot be empty")
	}

	// Reject schemes, slashes, or ports
	if strings.Contains(trimmed, "://") || strings.Contains(trimmed, "/") || strings.Contains(trimmed, ":") {
		return errors.New("domain must not contain protocol schemes, paths, or port numbers")
	}

	// Reject wildcards or whitespace
	if strings.Contains(trimmed, " ") || strings.Contains(trimmed, "*") {
		return errors.New("domain contains illegal characters (spaces or wildcards are not allowed)")
	}

	if len(trimmed) > 253 {
		return errors.New("domain exceeds maximum length of 253 characters")
	}

	// Localhost is valid for testing
	if strings.EqualFold(trimmed, "localhost") {
		return nil
	}

	if !domainRegex.MatchString(trimmed) {
		return errors.New("domain must be a valid fully-qualified domain name (e.g., example.com)")
	}

	return nil
}

// ValidateUpstreamDoHURL checks if the string is a valid HTTPS DoH endpoint.
func ValidateUpstreamDoHURL(rawURL string) error {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return errors.New("DoH upstream URL cannot be empty")
	}

	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil {
		return errors.New("invalid URL format: " + err.Error())
	}

	if !strings.EqualFold(parsed.Scheme, "https") {
		return errors.New("DoH URL must use the 'https://' scheme")
	}

	if parsed.Host == "" {
		return errors.New("DoH URL must contain a valid host name")
	}

	if parsed.Path == "" || parsed.Path == "/" {
		return errors.New("DoH URL should specify a DNS query path (typically /dns-query)")
	}

	return nil
}
