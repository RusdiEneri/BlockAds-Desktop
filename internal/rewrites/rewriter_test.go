package rewrites_test

import (
	"testing"

	"blockads-desktop/internal/rewrites"

	"github.com/miekg/dns"
)

func TestValidation(t *testing.T) {
	// 1. Valid A rule
	ruleA := rewrites.Rule{
		Domain: "nas.home.arpa",
		Type:   "A",
		Target: "192.168.18.11",
	}
	if err := ruleA.Validate(); err != nil {
		t.Fatalf("expected valid A rule, got error: %v", err)
	}

	// Invalid A rule (not an IPv4)
	ruleAInvalid := rewrites.Rule{
		Domain: "nas.home.arpa",
		Type:   "A",
		Target: "2001:db8::1",
	}
	if err := ruleAInvalid.Validate(); err == nil {
		t.Fatalf("expected error for IPv6 target in A rule")
	}

	// 2. Valid AAAA rule
	ruleAAAA := rewrites.Rule{
		Domain: "nas.home.arpa",
		Type:   "AAAA",
		Target: "fd00::1",
	}
	if err := ruleAAAA.Validate(); err != nil {
		t.Fatalf("expected valid AAAA rule, got error: %v", err)
	}

	// Invalid AAAA rule (IPv4 target)
	ruleAAAAInvalid := rewrites.Rule{
		Domain: "nas.home.arpa",
		Type:   "AAAA",
		Target: "192.168.1.1",
	}
	if err := ruleAAAAInvalid.Validate(); err == nil {
		t.Fatalf("expected error for IPv4 target in AAAA rule")
	}

	// 3. Valid CNAME rule
	ruleCNAME := rewrites.Rule{
		Domain: "memos.home.arpa",
		Type:   "CNAME",
		Target: "nas.home.arpa",
	}
	if err := ruleCNAME.Validate(); err != nil {
		t.Fatalf("expected valid CNAME rule, got error: %v", err)
	}

	// Invalid CNAME rule: target is an IP
	ruleCNAMEIP := rewrites.Rule{
		Domain: "memos.home.arpa",
		Type:   "CNAME",
		Target: "192.168.1.1",
	}
	if err := ruleCNAMEIP.Validate(); err == nil {
		t.Fatalf("expected error for IP target in CNAME rule")
	}

	// Invalid CNAME rule: target is self (immediate loop)
	ruleCNAMESelf := rewrites.Rule{
		Domain: "memos.home.arpa",
		Type:   "CNAME",
		Target: "memos.home.arpa",
	}
	if err := ruleCNAMESelf.Validate(); err == nil {
		t.Fatalf("expected error for self-referential CNAME target")
	}
}

func TestRewriter_ExactMatch_A_and_AAAA(t *testing.T) {
	rw := rewrites.NewRewriter()
	err := rw.AddRules([]rewrites.Rule{
		{Domain: "nas.home.arpa", Type: "A", Target: "192.168.18.11", Enabled: true, TTL: 60},
		{Domain: "nas.home.arpa", Type: "AAAA", Target: "fd00::1", Enabled: true, TTL: 60},
		{Domain: "disabled.home.arpa", Type: "A", Target: "10.0.0.1", Enabled: false, TTL: 60},
	})
	if err != nil {
		t.Fatalf("AddRules failed: %v", err)
	}

	// Query A for nas.home.arpa
	qA := new(dns.Msg)
	qA.SetQuestion("nas.home.arpa.", dns.TypeA)
	qA.Id = 0x1234

	respA := rw.Rewrite(qA)
	if respA == nil {
		t.Fatalf("expected rewrite response for nas.home.arpa A")
	}
	if respA.Id != 0x1234 {
		t.Errorf("expected response ID 0x1234, got 0x%x", respA.Id)
	}
	if len(respA.Answer) != 1 {
		t.Fatalf("expected 1 answer record, got %d", len(respA.Answer))
	}
	aRec, ok := respA.Answer[0].(*dns.A)
	if !ok || aRec.A.String() != "192.168.18.11" {
		t.Errorf("expected 192.168.18.11, got %v", respA.Answer[0])
	}
	if aRec.Hdr.Ttl != 60 {
		t.Errorf("expected TTL 60, got %d", aRec.Hdr.Ttl)
	}

	// Query AAAA for nas.home.arpa
	qAAAA := new(dns.Msg)
	qAAAA.SetQuestion("NAS.HOME.ARPA.", dns.TypeAAAA) // test case-insensitivity
	respAAAA := rw.Rewrite(qAAAA)
	if respAAAA == nil {
		t.Fatalf("expected rewrite response for nas.home.arpa AAAA")
	}
	if len(respAAAA.Answer) != 1 {
		t.Fatalf("expected 1 answer record, got %d", len(respAAAA.Answer))
	}
	aaaaRec, ok := respAAAA.Answer[0].(*dns.AAAA)
	if !ok || aaaaRec.AAAA.String() != "fd00::1" {
		t.Errorf("expected fd00::1, got %v", respAAAA.Answer[0])
	}

	// Query for disabled rule should return nil (no rewrite)
	qDisabled := new(dns.Msg)
	qDisabled.SetQuestion("disabled.home.arpa.", dns.TypeA)
	if rw.Rewrite(qDisabled) != nil {
		t.Fatalf("expected nil for disabled rewrite rule")
	}
}

func TestRewriter_NODATA_WhenTypeNotConfigured(t *testing.T) {
	rw := rewrites.NewRewriter()
	rw.AddRules([]rewrites.Rule{
		{Domain: "ipv4only.home.arpa", Type: "A", Target: "192.168.1.50", Enabled: true},
	})

	// Query AAAA on domain that only has an A rewrite rule
	qAAAA := new(dns.Msg)
	qAAAA.SetQuestion("ipv4only.home.arpa.", dns.TypeAAAA)

	resp := rw.Rewrite(qAAAA)
	if resp == nil {
		t.Fatalf("expected NODATA rewrite response to prevent forwarding local domain to public upstream")
	}
	if resp.Rcode != dns.RcodeSuccess {
		t.Errorf("expected NOERROR, got %d", resp.Rcode)
	}
	if len(resp.Answer) != 0 {
		t.Errorf("expected 0 answer records for NODATA, got %d", len(resp.Answer))
	}
}

func TestRewriter_Wildcard(t *testing.T) {
	rw := rewrites.NewRewriter()
	rw.AddRules([]rewrites.Rule{
		{Domain: "*.lab.home.arpa", Type: "A", Target: "10.0.0.99", Enabled: true},
		{Domain: "special.lab.home.arpa", Type: "A", Target: "10.0.0.1", Enabled: true}, // exact override
	})

	// 1. Exact match override wins over wildcard
	qExact := new(dns.Msg)
	qExact.SetQuestion("special.lab.home.arpa.", dns.TypeA)
	respExact := rw.Rewrite(qExact)
	if respExact == nil || respExact.Answer[0].(*dns.A).A.String() != "10.0.0.1" {
		t.Fatalf("exact match should take precedence over wildcard")
	}

	// 2. Subdomain matches wildcard
	qSub := new(dns.Msg)
	qSub.SetQuestion("k8s.lab.home.arpa.", dns.TypeA)
	respSub := rw.Rewrite(qSub)
	if respSub == nil || respSub.Answer[0].(*dns.A).A.String() != "10.0.0.99" {
		t.Fatalf("expected wildcard match for k8s.lab.home.arpa")
	}

	// 3. Deep nested subdomain also matches wildcard
	qDeep := new(dns.Msg)
	qDeep.SetQuestion("api.v1.k8s.lab.home.arpa.", dns.TypeA)
	respDeep := rw.Rewrite(qDeep)
	if respDeep == nil || respDeep.Answer[0].(*dns.A).A.String() != "10.0.0.99" {
		t.Fatalf("expected wildcard match for deep nested subdomain")
	}

	// 4. Base domain matches wildcard
	qBase := new(dns.Msg)
	qBase.SetQuestion("lab.home.arpa.", dns.TypeA)
	respBase := rw.Rewrite(qBase)
	if respBase == nil || respBase.Answer[0].(*dns.A).A.String() != "10.0.0.99" {
		t.Fatalf("expected wildcard match for base lab.home.arpa")
	}

	// 5. Unrelated domain does not match
	qOther := new(dns.Msg)
	qOther.SetQuestion("other.home.arpa.", dns.TypeA)
	if rw.Rewrite(qOther) != nil {
		t.Fatalf("unrelated domain must not match wildcard")
	}
}

func TestRewriter_CNAMEChain_And_LoopDetection(t *testing.T) {
	rw := rewrites.NewRewriter()
	rw.AddRules([]rewrites.Rule{
		{Domain: "memos.home.arpa", Type: "CNAME", Target: "nas.home.arpa", Enabled: true, TTL: 60},
		{Domain: "nas.home.arpa", Type: "A", Target: "192.168.18.11", Enabled: true, TTL: 60},

		// Circular CNAME loop: loop1 -> loop2 -> loop1
		{Domain: "loop1.local", Type: "CNAME", Target: "loop2.local", Enabled: true},
		{Domain: "loop2.local", Type: "CNAME", Target: "loop1.local", Enabled: true},
	})

	// 1. Chained CNAME + A resolution
	q := new(dns.Msg)
	q.SetQuestion("memos.home.arpa.", dns.TypeA)

	resp := rw.Rewrite(q)
	if resp == nil {
		t.Fatalf("expected response for CNAME chain")
	}
	if len(resp.Answer) != 2 {
		t.Fatalf("expected 2 answers (CNAME + A), got %d", len(resp.Answer))
	}

	cnameRec, ok := resp.Answer[0].(*dns.CNAME)
	if !ok || cnameRec.Target != "nas.home.arpa." {
		t.Errorf("expected CNAME to nas.home.arpa., got %v", resp.Answer[0])
	}

	aRec, ok := resp.Answer[1].(*dns.A)
	if !ok || aRec.A.String() != "192.168.18.11" {
		t.Errorf("expected target A record 192.168.18.11, got %v", resp.Answer[1])
	}

	// 2. Loop detection test: must not hang or infinite loop!
	qLoop := new(dns.Msg)
	qLoop.SetQuestion("loop1.local.", dns.TypeA)

	respLoop := rw.Rewrite(qLoop)
	if respLoop == nil {
		t.Fatalf("expected loop to be handled gracefully")
	}
	if respLoop.Rcode != dns.RcodeServerFailure {
		t.Errorf("expected SERVFAIL on CNAME loop detection, got RCODE %d", respLoop.Rcode)
	}
}
