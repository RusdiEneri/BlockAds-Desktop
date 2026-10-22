package dnsengine

import (
	"context"

	"github.com/miekg/dns"
)

// Resolver resolves allowed DNS queries against an upstream resolver service.
type Resolver interface {
	Resolve(ctx context.Context, req *dns.Msg) (*dns.Msg, error)
}

// FakeResolver is a mock resolver implementation exclusively for tests.
type FakeResolver struct {
	fn func(ctx context.Context, req *dns.Msg) (*dns.Msg, error)
}

// NewFakeResolver creates a new FakeResolver.
func NewFakeResolver(fn func(ctx context.Context, req *dns.Msg) (*dns.Msg, error)) *FakeResolver {
	return &FakeResolver{fn: fn}
}

// Resolve executes the mock resolve function.
func (f *FakeResolver) Resolve(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
	if f.fn == nil {
		return nil, nil
	}
	return f.fn(ctx, req)
}
