package cache_test

import (
	"fmt"
	"testing"
	"time"

	"blockads-desktop/internal/cache"

	"github.com/miekg/dns"
)

func BenchmarkDNSCache_Hit(b *testing.B) {
	c := cache.NewDNSCache(cache.Config{
		MaxEntries: 10000,
	})

	q := new(dns.Msg)
	q.SetQuestion("bench-hit.example.com.", dns.TypeA)
	resp := makeTestMessage("bench-hit.example.com.", dns.TypeA, dns.RcodeSuccess, []dns.RR{
		makeARecord("bench-hit.example.com.", "1.1.1.1", 300),
	}, nil)

	c.Set(q, resp)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		hit := c.Get(q)
		if hit == nil {
			b.Fatal("expected hit")
		}
	}
}

func BenchmarkDNSCache_Miss(b *testing.B) {
	c := cache.NewDNSCache(cache.Config{
		MaxEntries: 10000,
	})

	q := new(dns.Msg)
	q.SetQuestion("bench-miss.example.com.", dns.TypeA)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		hit := c.Get(q)
		if hit != nil {
			b.Fatal("expected miss")
		}
	}
}

func BenchmarkDNSCache_Expiration(b *testing.B) {
	mockClock := cache.NewMockClock(time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC))
	c := cache.NewDNSCache(cache.Config{
		MaxEntries: 10000,
		Clock:      mockClock,
	})

	q := new(dns.Msg)
	q.SetQuestion("exp.example.com.", dns.TypeA)
	resp := makeTestMessage("exp.example.com.", dns.TypeA, dns.RcodeSuccess, []dns.RR{
		makeARecord("exp.example.com.", "1.1.1.1", 10),
	}, nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Set(q, resp)
		mockClock.Advance(20 * time.Second)
		_ = c.Get(q)
	}
}

func BenchmarkDNSCache_ConcurrentLookups(b *testing.B) {
	c := cache.NewDNSCache(cache.Config{
		MaxEntries: 10000,
	})

	numDomains := 100
	questions := make([]*dns.Msg, numDomains)
	for i := 0; i < numDomains; i++ {
		name := fmt.Sprintf("concurrent-%d.example.com.", i)
		q := new(dns.Msg)
		q.SetQuestion(name, dns.TypeA)
		resp := makeTestMessage(name, dns.TypeA, dns.RcodeSuccess, []dns.RR{
			makeARecord(name, fmt.Sprintf("10.0.0.%d", i%250), 300),
		}, nil)
		c.Set(q, resp)
		questions[i] = q
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		idx := 0
		for pb.Next() {
			q := questions[idx%numDomains]
			_ = c.Get(q)
			idx++
		}
	})
}
