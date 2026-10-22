package dnsengine

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"

	"blockads-desktop/internal/cache"
	"blockads-desktop/internal/filters"
	"blockads-desktop/internal/rewrites"
)

// Decision represents the verdict for a query.
type Decision string

const (
	DecisionBlocked Decision = "BLOCKED"
	DecisionAllowed Decision = "ALLOWED"
	DecisionFailed  Decision = "FAILED"
)

// QueryDecision provides structured query-decision information for observability.
type QueryDecision struct {
	Timestamp         time.Time `json:"timestamp"`
	Domain            string    `json:"domain"`
	QueryType         string    `json:"queryType"`
	Decision          Decision  `json:"decision"`
	MatchedRule       string    `json:"filterRule,omitempty"`
	Reason            string    `json:"reason,omitempty"`
	UpstreamLatencyMs int       `json:"upstreamLatencyMs,omitempty"`
	Rcode             int       `json:"rcode"`
	ClientIP          string    `json:"clientIp"`
	Protocol          string    `json:"protocol"`
}

// QueryObserver callback function receiving structured query decisions.
type QueryObserver func(decision QueryDecision)

// HandlerConfig holds configuration parameters for the DNS engine Handler.
type HandlerConfig struct {
	Matcher        *filters.Matcher
	Resolver       Resolver
	QueryObserver  QueryObserver
	SinkholeTTL    uint32
	Cache          *cache.DNSCache
	FailureCache   *cache.FailureCache
	CircuitBreaker *CircuitBreaker
	Rewriter       *rewrites.Rewriter
}

// Handler handles incoming DNS requests across UDP and TCP.
type Handler struct {
	matcher        atomic.Pointer[filters.Matcher]
	resolver       Resolver
	queryObserver  QueryObserver
	sinkholeTTL    uint32
	cache          *cache.DNSCache
	failureCache   *cache.FailureCache
	circuitBreaker *CircuitBreaker
	rewriter       atomic.Pointer[rewrites.Rewriter]
}

// NewHandler creates a new Handler with the given config.
func NewHandler(cfg HandlerConfig) *Handler {
	ttl := cfg.SinkholeTTL
	if ttl == 0 {
		ttl = 60
	}
	cb := cfg.CircuitBreaker
	if cb == nil {
		cb = NewCircuitBreaker(CircuitBreakerConfig{})
	}
	h := &Handler{
		resolver:       cfg.Resolver,
		queryObserver:  cfg.QueryObserver,
		sinkholeTTL:    ttl,
		cache:          cfg.Cache,
		failureCache:   cfg.FailureCache,
		circuitBreaker: cb,
	}
	if cfg.Matcher != nil {
		h.matcher.Store(cfg.Matcher)
	}
	if cfg.Rewriter != nil {
		h.rewriter.Store(cfg.Rewriter)
	}
	return h
}

// Matcher returns the currently active domain filtering matcher.
func (h *Handler) Matcher() *filters.Matcher {
	return h.matcher.Load()
}

// SetMatcher atomically installs a new active domain filtering matcher.
func (h *Handler) SetMatcher(m *filters.Matcher) {
	h.matcher.Store(m)
}

// Cache returns the configured DNSCache instance, if any.
func (h *Handler) Cache() *cache.DNSCache {
	return h.cache
}

// FailureCache returns the configured FailureCache instance, if any.
func (h *Handler) FailureCache() *cache.FailureCache {
	return h.failureCache
}

// CircuitBreaker returns the configured CircuitBreaker instance, if any.
func (h *Handler) CircuitBreaker() *CircuitBreaker {
	return h.circuitBreaker
}

// Rewriter returns the configured Rewriter instance, if any.
func (h *Handler) Rewriter() *rewrites.Rewriter {
	return h.rewriter.Load()
}

// SetRewriter atomically installs a new active DNS rewriter.
func (h *Handler) SetRewriter(rw *rewrites.Rewriter) {
	h.rewriter.Store(rw)
}

// FlushCache clears all cached DNS responses and resolution failures.
func (h *Handler) FlushCache() {
	if h.cache != nil {
		h.cache.Flush()
	}
	if h.failureCache != nil {
		h.failureCache.Flush()
	}
	if h.circuitBreaker != nil {
		h.circuitBreaker.Reset()
	}
}

// ServeDNS processes incoming DNS queries.
func (h *Handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	// Defend against nil parameters to avoid panics
	if w == nil || r == nil {
		return
	}

	protocol := "UDP"
	clientIP := ""
	if remoteAddr := w.RemoteAddr(); remoteAddr != nil {
		if strings.Contains(strings.ToLower(remoteAddr.Network()), "tcp") {
			protocol = "TCP"
		}
		if host, _, err := net.SplitHostPort(remoteAddr.String()); err == nil {
			clientIP = host
		} else {
			clientIP = remoteAddr.String()
		}
	}

	// Validate Question section: must contain exactly 1 question
	if len(r.Question) != 1 {
		resp := MakeErrorResponse(r, dns.RcodeFormatError)
		_ = w.WriteMsg(resp)
		h.notify(QueryDecision{
			Timestamp: time.Now().UTC(),
			Decision:  DecisionFailed,
			Reason:    "malformed question section (count != 1)",
			Rcode:     dns.RcodeFormatError,
			ClientIP:  clientIP,
			Protocol:  protocol,
		})
		return
	}

	q := r.Question[0]
	qTypeStr := dns.TypeToString[q.Qtype]
	if qTypeStr == "" {
		qTypeStr = "UNKNOWN"
	}

	// Validate Opcode: only standard OpcodeQuery is supported
	if r.Opcode != dns.OpcodeQuery {
		resp := MakeErrorResponse(r, dns.RcodeNotImplemented)
		_ = w.WriteMsg(resp)
		h.notify(QueryDecision{
			Timestamp: time.Now().UTC(),
			Domain:    strings.TrimSuffix(q.Name, "."),
			QueryType: qTypeStr,
			Decision:  DecisionFailed,
			Reason:    "unsupported opcode",
			Rcode:     dns.RcodeNotImplemented,
			ClientIP:  clientIP,
			Protocol:  protocol,
		})
		return
	}

	// Validate Class: only ClassINET is supported
	if q.Qclass != dns.ClassINET {
		resp := MakeErrorResponse(r, dns.RcodeNotImplemented)
		_ = w.WriteMsg(resp)
		h.notify(QueryDecision{
			Timestamp: time.Now().UTC(),
			Domain:    strings.TrimSuffix(q.Name, "."),
			QueryType: qTypeStr,
			Decision:  DecisionFailed,
			Reason:    "unsupported query class (non-INET)",
			Rcode:     dns.RcodeNotImplemented,
			ClientIP:  clientIP,
			Protocol:  protocol,
		})
		return
	}

	canonicalDomain := strings.ToLower(strings.TrimSuffix(q.Name, "."))

	// Check local DNS Rewrites (evaluated before blocklists and cache)
	if rw := h.rewriter.Load(); rw != nil {
		if rewriteResp := rw.Rewrite(r); rewriteResp != nil {
			_ = w.WriteMsg(rewriteResp)

			h.notify(QueryDecision{
				Timestamp:         time.Now().UTC(),
				Domain:            canonicalDomain,
				QueryType:         qTypeStr,
				Decision:          DecisionAllowed,
				Reason:            "matched dns rewrite",
				UpstreamLatencyMs: 0,
				Rcode:             rewriteResp.Rcode,
				ClientIP:          clientIP,
				Protocol:          protocol,
			})
			return
		}
	}

	// Match domain against active filter matcher
	matchRes := filters.MatchResult{Action: filters.ActionPass}
	if m := h.matcher.Load(); m != nil {
		matchRes = m.Match(canonicalDomain)
	}

	if matchRes.Action == filters.ActionBlock {
		// Sinkhole response
		resp := MakeBlockResponse(r, h.sinkholeTTL)
		_ = w.WriteMsg(resp)

		h.notify(QueryDecision{
			Timestamp:   time.Now().UTC(),
			Domain:      canonicalDomain,
			QueryType:   qTypeStr,
			Decision:    DecisionBlocked,
			MatchedRule: matchRes.MatchedDomain,
			Reason:      "matched block rule: " + matchRes.RuleSource,
			Rcode:       resp.Rcode,
			ClientIP:    clientIP,
			Protocol:    protocol,
		})
		return
	}

	// Allowed query (either ActionAllow or ActionPass)
	// Check cache first for valid, non-expired response
	if h.cache != nil {
		if cachedResp := h.cache.Get(r); cachedResp != nil {
			_ = w.WriteMsg(cachedResp)

			reason := "cache hit"
			if matchRes.Action == filters.ActionAllow {
				reason = "cache hit (matched allow rule: " + matchRes.MatchedDomain + ")"
			}

			h.notify(QueryDecision{
				Timestamp:         time.Now().UTC(),
				Domain:            canonicalDomain,
				QueryType:         qTypeStr,
				Decision:          DecisionAllowed,
				MatchedRule:       matchRes.MatchedDomain,
				Reason:            reason,
				UpstreamLatencyMs: 0,
				Rcode:             cachedResp.Rcode,
				ClientIP:          clientIP,
				Protocol:          protocol,
			})
			return
		}
	}

	// Check RFC 9520 resolution failure cache before reaching out to upstream
	if h.failureCache != nil {
		if failEntry, ok := h.failureCache.Get(r); ok {
			resp := MakeErrorResponse(r, dns.RcodeServerFailure)
			_ = w.WriteMsg(resp)

			h.notify(QueryDecision{
				Timestamp:         time.Now().UTC(),
				Domain:            canonicalDomain,
				QueryType:         qTypeStr,
				Decision:          DecisionFailed,
				Reason:            "cached resolution failure (RFC 9520): " + failEntry.Reason,
				UpstreamLatencyMs: 0,
				Rcode:             dns.RcodeServerFailure,
				ClientIP:          clientIP,
				Protocol:          protocol,
			})
			return
		}
	}

	// Upstream resolver check: Zero silent fallback!
	if h.resolver == nil {
		resp := MakeErrorResponse(r, dns.RcodeServerFailure)
		_ = w.WriteMsg(resp)

		h.notify(QueryDecision{
			Timestamp: time.Now().UTC(),
			Domain:    canonicalDomain,
			QueryType: qTypeStr,
			Decision:  DecisionFailed,
			Reason:    "no upstream resolver configured (SERVFAIL)",
			Rcode:     dns.RcodeServerFailure,
			ClientIP:  clientIP,
			Protocol:  protocol,
		})
		return
	}

	// Check upstream-level circuit breaker (RFC 9520)
	if h.circuitBreaker != nil && !h.circuitBreaker.Allow() {
		resp := MakeErrorResponse(r, dns.RcodeServerFailure)
		_ = w.WriteMsg(resp)

		h.notify(QueryDecision{
			Timestamp:         time.Now().UTC(),
			Domain:            canonicalDomain,
			QueryType:         qTypeStr,
			Decision:          DecisionFailed,
			Reason:            "upstream circuit breaker open (RFC 9520 backoff)",
			UpstreamLatencyMs: 0,
			Rcode:             dns.RcodeServerFailure,
			ClientIP:          clientIP,
			Protocol:          protocol,
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	resp, err := h.resolver.Resolve(ctx, r)
	latency := int(time.Since(start).Milliseconds())

	if err != nil || resp == nil {
		if h.circuitBreaker != nil {
			h.circuitBreaker.RecordFailure()
		}

		failResp := MakeErrorResponse(r, dns.RcodeServerFailure)
		_ = w.WriteMsg(failResp)

		errMsg := "upstream resolution failed"
		if err != nil {
			errMsg = err.Error()
		}

		if h.failureCache != nil {
			cat := cache.FailureCategoryTransport
			if errors.Is(ctx.Err(), context.DeadlineExceeded) || strings.Contains(strings.ToLower(errMsg), "timeout") {
				cat = cache.FailureCategoryTimeout
			}
			h.failureCache.Set(r, cat, errMsg)
		}

		h.notify(QueryDecision{
			Timestamp:         time.Now().UTC(),
			Domain:            canonicalDomain,
			QueryType:         qTypeStr,
			Decision:          DecisionFailed,
			Reason:            errMsg,
			UpstreamLatencyMs: latency,
			Rcode:             dns.RcodeServerFailure,
			ClientIP:          clientIP,
			Protocol:          protocol,
		})
		return
	}

	// Upstream SERVFAIL caching per RFC 9520
	if resp.Rcode == dns.RcodeServerFailure {
		if h.failureCache != nil {
			h.failureCache.Set(r, cache.FailureCategorySERVFAIL, "upstream returned SERVFAIL")
		}
		_ = w.WriteMsg(resp)
		h.notify(QueryDecision{
			Timestamp:         time.Now().UTC(),
			Domain:            canonicalDomain,
			QueryType:         qTypeStr,
			Decision:          DecisionFailed,
			Reason:            "upstream returned SERVFAIL",
			UpstreamLatencyMs: latency,
			Rcode:             dns.RcodeServerFailure,
			ClientIP:          clientIP,
			Protocol:          protocol,
		})
		return
	}

	if h.circuitBreaker != nil {
		h.circuitBreaker.RecordSuccess()
	}

	// Preserve request ID and question in upstream reply
	resp.Id = r.Id
	resp.Question = r.Question
	resp.RecursionAvailable = true
	// Ensure we do not falsely claim DNSSEC validation unless verified
	resp.AuthenticatedData = false

	if h.cache != nil {
		h.cache.Set(r, resp)
	}

	_ = w.WriteMsg(resp)

	reason := "allowed"
	if matchRes.Action == filters.ActionAllow {
		reason = "matched allow rule: " + matchRes.MatchedDomain
	}

	h.notify(QueryDecision{
		Timestamp:         time.Now().UTC(),
		Domain:            canonicalDomain,
		QueryType:         qTypeStr,
		Decision:          DecisionAllowed,
		MatchedRule:       matchRes.MatchedDomain,
		Reason:            reason,
		UpstreamLatencyMs: latency,
		Rcode:             resp.Rcode,
		ClientIP:          clientIP,
		Protocol:          protocol,
	})
}

func (h *Handler) notify(d QueryDecision) {
	if h.queryObserver != nil {
		h.queryObserver(d)
	}
}
