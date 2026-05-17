package net_http_client

import (
	"errors"
	"net/http"

	"github.com/sony/gobreaker"
)

// breakerRoundTripper wraps a downstream RoundTripper and executes requests
// under a circuit breaker. When the breaker is nil it delegates directly to
// the next RoundTripper.
type breakerRoundTripper struct {
	next http.RoundTripper
	cb   *gobreaker.CircuitBreaker
}

// NewBreakerMiddleware returns a middleware builder: a function that accepts
// the next RoundTripper and returns a RoundTripper that applies circuit
// breaking according to cfg. This keeps middleware composition the client's
// responsibility (the client composes builders into a final transport).
// NewBreakerMiddleware returns a ready-to-use http.RoundTripper that wraps
// the provided base RoundTripper with a circuit breaker. If base is nil,
// http.DefaultTransport will be used (same pattern as otelhttp.NewTransport).
func NewBreakerMiddleware(base http.RoundTripper, opts ...BreakerOption) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	cfg := newCircuitBreakerConfig(opts...)
	// Guard casts from int -> uint32 to avoid potential overflow warnings
	maxUint32Int := int(^uint32(0))
	halfOpen := min(max(cfg.HalfOpenRequests, 0), maxUint32Int)
	maxFailures := min(max(cfg.MaxFailures, 0), maxUint32Int)

	//nolint:gosec // G115: safe conversion to uint32 after bounds checks above
	settings := gobreaker.Settings{
		Name:        "breaker",
		MaxRequests: uint32(halfOpen),
		Timeout:     cfg.ResetTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= uint32(maxFailures)
		},
	}
	return &breakerRoundTripper{next: base, cb: gobreaker.NewCircuitBreaker(settings)}
}

func (b *breakerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if b.next == nil {
		return nil, errors.New("breaker middleware: next RoundTripper is nil; ensure client provides base transport")
	}
	if b.cb == nil {
		return b.next.RoundTrip(req)
	}

	var resp *http.Response
	_, err := b.cb.Execute(func() (any, error) {
		r, err := b.next.RoundTrip(req)
		if err != nil {
			// ensure we don't leak a body when a RoundTrip returns both
			// a response and an error (defensive)
			if r != nil && r.Body != nil {
				_ = r.Body.Close()
			}
			return nil, err
		}
		resp = r
		return resp, nil
	})
	return resp, err
}
