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
	next              http.RoundTripper
	cb                *gobreaker.CircuitBreaker
	failureClassifier FailureClassifier
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
	//nolint:gosec // G115: safe conversion to uint32 after bounds checks above
	halfOpen := uint32(min(max(cfg.HalfOpenRequests, 0), maxUint32Int))
	//nolint:gosec // G115: safe conversion to uint32 after bounds checks above
	maxFailures := uint32(min(max(cfg.MaxFailures, 0), maxUint32Int))

	//nolint:gosec // G115: safe conversion to uint32 after bounds checks above
	settings := gobreaker.Settings{
		Name:        "breaker",
		MaxRequests: halfOpen,
		Timeout:     cfg.ResetTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Don't trip if below consecutive failure threshold
			if counts.ConsecutiveFailures < maxFailures || counts.Requests == 0 {
				return false
			}

			// Trip if failure rate is high (gradual degradation)
			failureRate := float64(counts.TotalFailures) / float64(counts.Requests)
			return failureRate >= 0.5
		},
	}

	return &breakerRoundTripper{
		next:              base,
		cb:                gobreaker.NewCircuitBreaker(settings),
		failureClassifier: cfg.FailureClassifier,
	}
}

func closeResponseBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}

func (b *breakerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if b.next == nil {
		return nil, errors.New("breaker middleware: next RoundTripper is nil; ensure client provides base transport")
	}
	if b.cb == nil {
		return b.next.RoundTrip(req)
	}

	respVal, err := b.cb.Execute(func() (any, error) {
		r, reqErr := b.next.RoundTrip(req)

		if reqErr != nil {
			closeResponseBody(r)
			return nil, reqErr
		}

		// Classify response for circuit breaker tracking but never mask it
		cbErr := b.failureClassifier.ClassifyError(r)
		// Return response as any, use cbErr only to signal CB increment
		return r, cbErr
	})

	// Response comes as any, error is only for CB tracking
	if resp, ok := respVal.(*http.Response); ok {
		return resp, nil
	}

	// No response, only error from request
	return nil, err
}
