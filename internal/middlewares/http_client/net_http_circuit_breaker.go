package middlewares

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
	settings := gobreaker.Settings{
		Name:        "breaker",
		MaxRequests: uint32(cfg.HalfOpenRequests),
		Timeout:     cfg.ResetTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= uint32(cfg.MaxFailures)
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
			return nil, err
		}
		resp = r
		return resp, nil
	})
	return resp, err
}
