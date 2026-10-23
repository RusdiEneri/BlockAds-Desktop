package filtermanager

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"unicode"

	"blockads-desktop/internal/models"
)

// CatalogFilter represents a curated upstream filter list available in the application catalog.
type CatalogFilter struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"` // "Advertising", "Privacy & Tracking", "Security & Malware", "General Protection"
	SourceURL   string `json:"sourceUrl"`
	Format      string `json:"format"` // "hosts", "domains"
	Homepage    string `json:"homepage"`
	Maintainer  string `json:"maintainer"`
	License     string `json:"license"`
}

// ExtractedMetadata holds discovered metadata parsed from comment headers.
type ExtractedMetadata struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Homepage    string `json:"homepage"`
	Format      string `json:"format"`
	RuleCount   int    `json:"ruleCount"`
}

// BuiltinCatalog returns the curated, verified DNS filter lists.
func BuiltinCatalog() []CatalogFilter {
	return []CatalogFilter{
		{
			ID:          "adaway",
			Name:        "AdAway Default Hosts",
			Description: "Blocks mobile and desktop advertisement networks and telemetry servers.",
			Category:    "Advertising",
			SourceURL:   "https://adaway.org/hosts.txt",
			Format:      "hosts",
			Homepage:    "https://adaway.org/",
			Maintainer:  "AdAway Team",
			License:     "CC BY 3.0",
		},
		{
			ID:          "stevenblack",
			Name:        "StevenBlack Unified Hosts",
			Description: "Consolidated hosts file blocking adware, spam, and telemetry from multiple curated sources.",
			Category:    "General Protection",
			SourceURL:   "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
			Format:      "hosts",
			Homepage:    "https://github.com/StevenBlack/hosts",
			Maintainer:  "Steven Black",
			License:     "MIT",
		},
		{
			ID:          "oisd-basic",
			Name:        "OISD Basic",
			Description: "Lightweight, high-reliability list designed to prevent false positives across smart TVs and desktops.",
			Category:    "Privacy & Tracking",
			SourceURL:   "https://basic.oisd.nl/",
			Format:      "domains",
			Homepage:    "https://oisd.nl/",
			Maintainer:  "Stephan van Velzen (OISD)",
			License:     "Free for personal use",
		},
		{
			ID:          "urlhaus",
			Name:        "URLhaus Malware Hosts",
			Description: "Real-time threat intelligence list blocking verified malware distribution sites and botnet C2s.",
			Category:    "Security & Malware",
			SourceURL:   "https://urlhaus.abuse.ch/downloads/hostfile/",
			Format:      "hosts",
			Homepage:    "https://urlhaus.abuse.ch/",
			Maintainer:  "abuse.ch",
			License:     "CC0",
		},
		{
			ID:          "dan-pollock",
			Name:        "Dan Pollock's Hosts",
			Description: "Maintained since 1999, blocks intrusive advertising, web tracking, and shock sites.",
			Category:    "Advertising",
			SourceURL:   "https://someonewhocares.org/hosts/hosts",
			Format:      "hosts",
			Homepage:    "https://someonewhocares.org/hosts/",
			Maintainer:  "Dan Pollock",
			License:     "Free",
		},
		{
			ID:          "peter-lowe",
			Name:        "Peter Lowe's Ad & Tracking List",
			Description: "Strictly curated list of adservers and tracking domains with near-zero false positive rate.",
			Category:    "Privacy & Tracking",
			SourceURL:   "https://pgl.yoyo.org/adservers/serverlist.php?hostformat=hosts&mimetype=plaintext",
			Format:      "hosts",
			Homepage:    "https://pgl.yoyo.org/adservers/",
			Maintainer:  "Peter Lowe",
			License:     "Free for personal use",
		},
	}
}

// NormalizeURL canonicalizes a filter list URL for stable duplicate detection.
func NormalizeURL(rawURL string) (string, error) {
	return models.NormalizeURL(rawURL)
}

// sanitizeString removes control characters and bounds length.
func sanitizeString(s string, maxLen int) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsPrint(r) {
			b.WriteRune(r)
		}
	}
	clean := strings.TrimSpace(b.String())
	if len(clean) > maxLen {
		return clean[:maxLen]
	}
	return clean
}

// ExtractFilterMetadata parses comment headers and sample rules from a stream.
// It verifies the stream is not HTML, identifies title/description/homepage comments,
// detects the format, and counts candidate rules.
func ExtractFilterMetadata(r io.Reader, sourceURL string) (*ExtractedMetadata, error) {
	scanner := bufio.NewScanner(r)

	var title string
	var description string
	var homepage string
	var detectedFormat = "hosts"
	var ruleCount int
	var lineIdx int
	var isHTML bool

	for scanner.Scan() {
		lineIdx++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// HTML check within first 10 non-empty lines
		if lineIdx <= 10 {
			lower := strings.ToLower(line)
			if strings.HasPrefix(lower, "<!doctype html") ||
				strings.HasPrefix(lower, "<html") ||
				strings.Contains(lower, "<head>") ||
				strings.Contains(lower, "<body>") {
				isHTML = true
				break
			}
		}

		// Comment inspection for metadata
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			content := strings.TrimSpace(line[1:])
			lower := strings.ToLower(content)

			if title == "" {
				if strings.HasPrefix(lower, "title:") {
					title = sanitizeString(strings.TrimSpace(content[6:]), 100)
				} else if strings.HasPrefix(lower, "name:") {
					title = sanitizeString(strings.TrimSpace(content[5:]), 100)
				}
			}

			if description == "" {
				if strings.HasPrefix(lower, "description:") {
					description = sanitizeString(strings.TrimSpace(content[12:]), 250)
				}
			}

			if homepage == "" {
				if strings.HasPrefix(lower, "homepage:") {
					homepage = sanitizeString(strings.TrimSpace(content[9:]), 200)
				}
			}

			continue
		}

		// Active rule detection
		if strings.HasPrefix(line, "[") {
			detectedFormat = "adblock"
			continue
		}

		if strings.HasPrefix(line, "0.0.0.0") || strings.HasPrefix(line, "127.0.0.1") {
			detectedFormat = "hosts"
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				target := strings.ToLower(fields[1])
				if target != "localhost" && target != "localhost.localdomain" && target != "local" && target != "broadcasthost" && target != "0.0.0.0" && target != "127.0.0.1" {
					ruleCount++
				}
			}
		} else if strings.HasPrefix(line, "||") || strings.HasPrefix(line, "@@||") {
			detectedFormat = "adblock"
			ruleCount++
		} else if models.ValidateDomain(line) == nil {
			detectedFormat = "domains"
			ruleCount++
		}
	}

	if isHTML {
		return nil, errors.New("remote source returned an HTML web page instead of a plain-text DNS filter list")
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("error reading filter content: %w", err)
	}

	// Fallback title logic:
	// 1. Title from comments
	// 2. URL filename without extension
	// 3. Hostname
	if title == "" && sourceURL != "" {
		if parsed, err := url.Parse(sourceURL); err == nil {
			filename := path.Base(parsed.Path)
			if filename != "" && filename != "/" && filename != "." {
				ext := path.Ext(filename)
				title = strings.TrimSuffix(filename, ext)
			}
			if title == "" || title == "hosts" || title == "list" {
				title = parsed.Hostname()
			}
		}
	}

	if title == "" {
		title = "Custom DNS Filter"
	}

	return &ExtractedMetadata{
		Title:       title,
		Description: description,
		Homepage:    homepage,
		Format:      detectedFormat,
		RuleCount:   ruleCount,
	}, nil
}
