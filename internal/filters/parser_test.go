package filters_test

import (
	"strings"
	"testing"

	"blockads-desktop/internal/filters"
)

func TestParseFilterList_HostsFormat(t *testing.T) {
	input := `# Standard hosts file
127.0.0.1 localhost
::1 localhost ip6-localhost ip6-loopback
0.0.0.0 ads.example.com tracker.example.net
127.0.0.1 analytics.example.org
::1 unwanted.example.test
`
	res, err := filters.ParseFilterList(strings.NewReader(input), filters.FormatHosts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Stats.AcceptedCount != 4 {
		t.Errorf("expected 4 accepted rules, got %d", res.Stats.AcceptedCount)
	}
	if len(res.Rules) != 4 {
		t.Fatalf("expected 4 rules in slice, got %d", len(res.Rules))
	}

	expected := map[string]bool{
		"ads.example.com":       true,
		"tracker.example.net":   true,
		"analytics.example.org": true,
		"unwanted.example.test": true,
	}

	for _, r := range res.Rules {
		if !expected[r.Domain] {
			t.Errorf("unexpected domain: %s", r.Domain)
		}
		if r.Action != filters.ActionBlock {
			t.Errorf("expected ActionBlock for %s, got %s", r.Domain, r.Action)
		}
	}
}

func TestParseFilterList_PlainDomainFormat(t *testing.T) {
	input := `# Tracking domains list
! Another comment style
ads.example.com
tracker.example.net

# Empty line above
analytics.example.org.
`
	res, err := filters.ParseFilterList(strings.NewReader(input), filters.FormatPlain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Stats.AcceptedCount != 3 {
		t.Errorf("expected 3 accepted rules, got %d", res.Stats.AcceptedCount)
	}

	expectedDomains := []string{"ads.example.com", "tracker.example.net", "analytics.example.org"}
	for i, exp := range expectedDomains {
		if res.Rules[i].Domain != exp {
			t.Errorf("rule %d: expected %s, got %s", i, exp, res.Rules[i].Domain)
		}
	}
}

func TestParseFilterList_AutoDetectFormat(t *testing.T) {
	hostsInput := "0.0.0.0 ad1.com\n127.0.0.1 ad2.com"
	resHosts, err := filters.ParseFilterList(strings.NewReader(hostsInput), filters.FormatAuto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resHosts.Stats.AcceptedCount != 2 {
		t.Errorf("expected 2 accepted rules for hosts, got %d", resHosts.Stats.AcceptedCount)
	}

	plainInput := "ad1.com\nad2.com"
	resPlain, err := filters.ParseFilterList(strings.NewReader(plainInput), filters.FormatAuto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resPlain.Stats.AcceptedCount != 2 {
		t.Errorf("expected 2 accepted rules for plain, got %d", resPlain.Stats.AcceptedCount)
	}
}

func TestParseFilterList_CRLFAndWhitespaceAndCase(t *testing.T) {
	input := "0.0.0.0   ADS.EXAMPLE.COM.  \r\n\r\n  127.0.0.1   Sub.Tracker.NET.\t\r\n"
	res, err := filters.ParseFilterList(strings.NewReader(input), filters.FormatHosts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Stats.AcceptedCount != 2 {
		t.Fatalf("expected 2 accepted rules, got %d", res.Stats.AcceptedCount)
	}

	if res.Rules[0].Domain != "ads.example.com" {
		t.Errorf("expected ads.example.com, got %s", res.Rules[0].Domain)
	}
	if res.Rules[1].Domain != "sub.tracker.net" {
		t.Errorf("expected sub.tracker.net, got %s", res.Rules[1].Domain)
	}
}

func TestParseFilterList_Deduplication(t *testing.T) {
	input := `0.0.0.0 ads.example.com
0.0.0.0 ads.example.com
127.0.0.1 ADS.EXAMPLE.COM.
0.0.0.0 unique.example.com
`
	res, err := filters.ParseFilterList(strings.NewReader(input), filters.FormatHosts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Stats.AcceptedCount != 2 {
		t.Errorf("expected 2 accepted rules, got %d", res.Stats.AcceptedCount)
	}
	if res.Stats.DuplicateCount != 2 {
		t.Errorf("expected 2 duplicates, got %d", res.Stats.DuplicateCount)
	}
}

func TestParseFilterList_LocalhostAndSpecialSkipped(t *testing.T) {
	input := `127.0.0.1 localhost
127.0.0.1 localhost.localdomain
127.0.0.1 local
255.255.255.255 broadcasthost
::1 localhost
::1 ip6-localhost
::1 ip6-loopback
fe80::1%lo0 localhost
0.0.0.0 0.0.0.0
`
	res, err := filters.ParseFilterList(strings.NewReader(input), filters.FormatHosts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Stats.AcceptedCount != 0 {
		t.Errorf("expected 0 accepted rules, got %d", res.Stats.AcceptedCount)
	}
	if res.Stats.SkippedCount == 0 {
		t.Errorf("expected skipped count > 0")
	}
}

func TestParseFilterList_WarningsForMalformedAndUnsupported(t *testing.T) {
	input := `valid.example.com
https://example.com/ad.js
||adservice.google.com^
##.ad-banner
*.wildcard.example.com
bad domain with spaces.com
`
	res, err := filters.ParseFilterList(strings.NewReader(input), filters.FormatPlain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Stats.AcceptedCount != 1 {
		t.Errorf("expected 1 accepted rule, got %d", res.Stats.AcceptedCount)
	}
	if res.Rules[0].Domain != "valid.example.com" {
		t.Errorf("expected valid.example.com, got %s", res.Rules[0].Domain)
	}

	if len(res.Warnings) != 5 {
		t.Fatalf("expected 5 warnings, got %d", len(res.Warnings))
	}

	// Verify warnings contain line numbers and descriptive reasons
	warningLines := map[int]string{}
	for _, w := range res.Warnings {
		warningLines[w.Line] = w.Reason
	}

	if _, ok := warningLines[2]; !ok {
		t.Errorf("expected warning on line 2 (URL)")
	}
	if _, ok := warningLines[3]; !ok {
		t.Errorf("expected warning on line 3 (Adblock syntax ||...^)")
	}
	if _, ok := warningLines[4]; !ok {
		t.Errorf("expected warning on line 4 (Cosmetic filter ##...)")
	}
	if _, ok := warningLines[5]; !ok {
		t.Errorf("expected warning on line 5 (Wildcard)")
	}
	if _, ok := warningLines[6]; !ok {
		t.Errorf("expected warning on line 6 (Spaces)")
	}
}

func TestParseFilterList_EmptyInput(t *testing.T) {
	res, err := filters.ParseFilterList(strings.NewReader(""), filters.FormatAuto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Stats.AcceptedCount != 0 || len(res.Rules) != 0 {
		t.Errorf("expected empty result")
	}
}

func TestParseFilterList_AdblockFormat(t *testing.T) {
	input := `[Adblock Plus 2.0]
! Title: Test Adblock List
! Version: 1.0.0
||ads.example.com^
||analytics.example.org^$third-party
@@||safe.example.com^
||ignored.example.com^$badfilter
##.ad-banner
example.com##.sponsor
site.org#@#.allowed-banner
||example.com/banner.gif^
||localhost^
||127.0.0.1^
||ADS.EXAMPLE.COM.^
||sub.tracker.net^
`
	res, err := filters.ParseFilterList(strings.NewReader(input), filters.FormatAdblock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected accepted:
	// 1. ads.example.com (BLOCK)
	// 2. analytics.example.org (BLOCK)
	// 3. safe.example.com (ALLOW)
	// 4. sub.tracker.net (BLOCK)
	// ads.example.com uppercase is duplicate
	if res.Stats.AcceptedCount != 4 {
		t.Fatalf("expected 4 accepted rules, got %d (rules: %+v)", res.Stats.AcceptedCount, res.Rules)
	}
	if res.Stats.DuplicateCount != 1 {
		t.Errorf("expected 1 duplicate rule, got %d", res.Stats.DuplicateCount)
	}
	if res.Stats.WarningCount != 0 {
		t.Errorf("expected 0 warnings, got %d (%+v)", res.Stats.WarningCount, res.Warnings)
	}

	ruleMap := make(map[string]filters.ParsedRule)
	for _, r := range res.Rules {
		ruleMap[r.Domain] = r
	}

	if r, ok := ruleMap["ads.example.com"]; !ok || r.Action != filters.ActionBlock {
		t.Errorf("expected ads.example.com BLOCK, got %+v", r)
	}
	if r, ok := ruleMap["analytics.example.org"]; !ok || r.Action != filters.ActionBlock {
		t.Errorf("expected analytics.example.org BLOCK, got %+v", r)
	}
	if r, ok := ruleMap["safe.example.com"]; !ok || r.Action != filters.ActionAllow {
		t.Errorf("expected safe.example.com ALLOW, got %+v", r)
	}
	if r, ok := ruleMap["sub.tracker.net"]; !ok || r.Action != filters.ActionBlock {
		t.Errorf("expected sub.tracker.net BLOCK, got %+v", r)
	}

	// Verify rules integrate with Matcher correctly
	matcherRules := make([]filters.Rule, len(res.Rules))
	for i, pr := range res.Rules {
		matcherRules[i] = filters.Rule{
			Domain:  pr.Domain,
			Action:  pr.Action,
			Enabled: pr.Enabled,
			Source:  pr.Source,
		}
	}
	m := filters.NewMatcher()
	m.BuildFromRules(matcherRules)

	if m.Match("ads.example.com").Action != filters.ActionBlock {
		t.Errorf("expected ads.example.com to match ActionBlock")
	}
	if m.Match("sub.ads.example.com").Action != filters.ActionBlock {
		t.Errorf("expected sub.ads.example.com to match ActionBlock")
	}
	if m.Match("safe.example.com").Action != filters.ActionAllow {
		t.Errorf("expected safe.example.com to match ActionAllow")
	}
	if m.Match("clean.example.com").Action != filters.ActionPass {
		t.Errorf("expected clean.example.com to match ActionPass")
	}
}

func TestParseFilterList_AutoDetectAdblock(t *testing.T) {
	input := `[Adblock Plus 2.0]
! HaGeZi style sample
||evil-tracker.com^
@@||good-service.com^
`
	res, err := filters.ParseFilterList(strings.NewReader(input), filters.FormatAuto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Stats.AcceptedCount != 2 {
		t.Fatalf("expected 2 accepted rules, got %d", res.Stats.AcceptedCount)
	}
	if res.Rules[0].Domain != "evil-tracker.com" || res.Rules[0].Action != filters.ActionBlock {
		t.Errorf("rule 0 mismatch: %+v", res.Rules[0])
	}
	if res.Rules[1].Domain != "good-service.com" || res.Rules[1].Action != filters.ActionAllow {
		t.Errorf("rule 1 mismatch: %+v", res.Rules[1])
	}
}

func BenchmarkParseFilterList_Large(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 5000; i++ {
		sb.WriteString("0.0.0.0 ad-tracker-")
		sb.WriteString(string(rune('a' + (i % 26))))
		sb.WriteString(".example.org\n")
	}
	input := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := filters.ParseFilterList(strings.NewReader(input), filters.FormatHosts)
		if err != nil {
			b.Fatal(err)
		}
	}
}
