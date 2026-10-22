package cache_test

import (
	"testing"
	"time"

	"github.com/miekg/dns"

	"blockads-desktop/internal/cache"
)

func TestNewKey_NormalizationAndSeparation(t *testing.T) {
	// 1. Basic A record with trailing dot and mixed case
	reqA := new(dns.Msg)
	reqA.SetQuestion("ExAmPLe.CoM.", dns.TypeA)

	keyA, ok := cache.NewKey(reqA)
	if !ok {
		t.Fatalf("expected valid key")
	}
	if keyA.Name != "example.com" {
		t.Errorf("expected normalized name 'example.com', got %q", keyA.Name)
	}
	if keyA.Qtype != dns.TypeA {
		t.Errorf("expected Qtype A (%d), got %d", dns.TypeA, keyA.Qtype)
	}
	if keyA.Qclass != dns.ClassINET {
		t.Errorf("expected Qclass INET, got %d", keyA.Qclass)
	}
	if keyA.DO {
		t.Errorf("expected DO=false without EDNS0")
	}

	// 2. AAAA query for same domain must produce a distinct key
	reqAAAA := new(dns.Msg)
	reqAAAA.SetQuestion("example.com.", dns.TypeAAAA)
	keyAAAA, _ := cache.NewKey(reqAAAA)

	if keyA == keyAAAA {
		t.Errorf("TypeA and TypeAAAA keys must not be equal")
	}

	// 3. Different Class (CHAOS)
	reqChaos := new(dns.Msg)
	reqChaos.SetQuestion("example.com.", dns.TypeA)
	reqChaos.Question[0].Qclass = dns.ClassCHAOS
	keyChaos, _ := cache.NewKey(reqChaos)

	if keyA == keyChaos {
		t.Errorf("INET and CHAOS class keys must not be equal")
	}

	// 4. EDNS0 with DO flag
	reqDO := new(dns.Msg)
	reqDO.SetQuestion("example.com.", dns.TypeA)
	reqDO.SetEdns0(4096, true) // do = true
	keyDO, _ := cache.NewKey(reqDO)

	if !keyDO.DO {
		t.Errorf("expected DO=true for EDNS0 with DO bit")
	}
	if keyA == keyDO {
		t.Errorf("DO=false and DO=true keys must not be equal")
	}

	// 5. Zero questions or nil must return false
	if _, ok := cache.NewKey(nil); ok {
		t.Errorf("expected false for nil req")
	}
	emptyReq := new(dns.Msg)
	if _, ok := cache.NewKey(emptyReq); ok {
		t.Errorf("expected false for 0 questions")
	}
}

func TestMockClock(t *testing.T) {
	initial := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	clock := cache.NewMockClock(initial)

	if !clock.Now().Equal(initial) {
		t.Errorf("expected %v, got %v", initial, clock.Now())
	}

	clock.Advance(30 * time.Second)
	expected := initial.Add(30 * time.Second)
	if !clock.Now().Equal(expected) {
		t.Errorf("expected %v, got %v", expected, clock.Now())
	}
}
