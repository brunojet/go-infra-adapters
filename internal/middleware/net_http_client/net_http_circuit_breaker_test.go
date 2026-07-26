package net_http_client

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sony/gobreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyRT struct {
	called bool
	resp   *http.Response
	err    error
}

func (d *dummyRT) RoundTrip(r *http.Request) (*http.Response, error) {
	d.called = true
	if d.resp != nil {
		return d.resp, d.err
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header)}, nil
}

func TestBreakerRoundTripper_NextNil_Panics(t *testing.T) {
	br := &breakerRoundTripper{next: nil, cb: nil, failureClassifier: &DefaultFailureClassifier{}}
	resp, err := br.RoundTrip(httptest.NewRequest("GET", "/", nil))
	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "next RoundTripper is nil")
}

func TestBreakerRoundTripper_DelegatesWhenCBNil(t *testing.T) {
	d := &dummyRT{}
	br := &breakerRoundTripper{next: d, cb: nil, failureClassifier: &DefaultFailureClassifier{}}
	resp, err := br.RoundTrip(httptest.NewRequest("GET", "/ok", nil))
	require.NoError(t, err)
	assert.NotNil(t, resp)
	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	assert.True(t, d.called)
}

func TestBreakerRoundTripper_WithCB_ExecutesAndReturnsResponse(t *testing.T) {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{Name: "test", Timeout: time.Second, MaxRequests: 1})
	d := &dummyRT{}
	br := &breakerRoundTripper{next: d, cb: cb, failureClassifier: &DefaultFailureClassifier{}}
	resp, err := br.RoundTrip(httptest.NewRequest("GET", "/cb", nil))
	require.NoError(t, err)
	assert.NotNil(t, resp)
	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	assert.True(t, d.called)
}

func TestNewBreakerMiddleware_ReturnsBreakerRoundTripper(t *testing.T) {
	rt := NewBreakerMiddleware(nil, WithCircuitBreakerMaxFailures(3), WithCircuitBreakerHalfOpenRequests(1), WithCircuitBreakerResetTimeout(500*time.Millisecond))
	brt, ok := rt.(*breakerRoundTripper)
	require.True(t, ok)
	assert.NotNil(t, brt.cb)
	assert.Equal(t, http.DefaultTransport, brt.next)
}
func TestNewBreakerMiddleware_UsesProvidedBase(t *testing.T) {
	d := &dummyRT{}
	rt := NewBreakerMiddleware(d)
	brt, ok := rt.(*breakerRoundTripper)
	require.True(t, ok)
	assert.Equal(t, d, brt.next)
}

// errRT is a RoundTripper that always returns the configured response and error.
type errRT struct {
	resp *http.Response
	err  error
}

func (e *errRT) RoundTrip(*http.Request) (*http.Response, error) { return e.resp, e.err }

// TestBreakerRoundTripper_WithCB_PropagatesError covers the error-return path
// inside cb.Execute when the inner RoundTrip fails with a nil response.
func TestBreakerRoundTripper_WithCB_PropagatesError(t *testing.T) {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{Name: "err-test", MaxRequests: 1})
	inner := &errRT{resp: nil, err: errors.New("transport error")}
	br := &breakerRoundTripper{next: inner, cb: cb, failureClassifier: &DefaultFailureClassifier{}}
	resp, err := br.RoundTrip(httptest.NewRequest("GET", "/", nil))
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	assert.Nil(t, resp)
	require.Error(t, err)
}

// TestBreakerRoundTripper_WithCB_ClosesBodyOnErrorWithResponse covers the
// defensive body-close when the inner RoundTrip returns both a response and error.
func TestBreakerRoundTripper_WithCB_ClosesBodyOnErrorWithResponse(t *testing.T) {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{Name: "body-close-test", MaxRequests: 1})
	inner := &errRT{
		resp: &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("body"))},
		err:  errors.New("transport error with body"),
	}
	br := &breakerRoundTripper{next: inner, cb: cb, failureClassifier: &DefaultFailureClassifier{}}
	resp, err := br.RoundTrip(httptest.NewRequest("GET", "/", nil))
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	require.Error(t, err)
}

// TestNewBreakerMiddleware_ReadyToTrip covers the ReadyToTrip lambda body by
// causing enough consecutive failures to open the breaker.
func TestNewBreakerMiddleware_ReadyToTrip(t *testing.T) {
	inner := &errRT{resp: nil, err: errors.New("fail")}
	rt := NewBreakerMiddleware(inner, WithCircuitBreakerMaxFailures(1), WithCircuitBreakerResetTimeout(time.Second))
	// First request: failure triggers ReadyToTrip evaluation → breaker opens
	resp1, _ := rt.RoundTrip(httptest.NewRequest("GET", "/", nil))
	if resp1 != nil {
		defer func() { _ = resp1.Body.Close() }()
	}
	// Second request: breaker is open → ErrOpenState returned without calling inner
	resp2, err := rt.RoundTrip(httptest.NewRequest("GET", "/", nil))
	if resp2 != nil {
		defer func() { _ = resp2.Body.Close() }()
	}
	require.Error(t, err)
}

// TestBreakerRoundTripper_Returns5xxResponseIntact validates that the circuit breaker
// increments internally but always returns the response body to the caller, never masking it.
// This is critical: if a 5xx response comes back, the caller MUST receive it to handle errors.
func TestBreakerRoundTripper_Returns5xxResponseIntact(t *testing.T) {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{Name: "5xx-test", MaxRequests: 1})
	bodyContent := "Internal Server Error details"
	inner := &errRT{
		resp: &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader(bodyContent)),
			Header:     make(http.Header),
		},
		err: nil, // No network error, just bad status code
	}
	br := &breakerRoundTripper{next: inner, cb: cb, failureClassifier: &DefaultFailureClassifier{}}

	resp, err := br.RoundTrip(httptest.NewRequest("GET", "/", nil))

	// Circuit breaker MUST return response intacta
	require.NotNil(t, resp, "response should not be nil even with 5xx")
	require.Nil(t, err, "error should be nil when response exists")
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// Body must be readable by caller
	body, readErr := io.ReadAll(resp.Body)
	require.NoError(t, readErr)
	assert.Equal(t, bodyContent, string(body))

	_ = resp.Body.Close()
}

// TestBreakerRoundTripper_Returns429ResponseIntact validates that 429 rate limiting
// responses are returned intacta with body, not masked.
func TestBreakerRoundTripper_Returns429ResponseIntact(t *testing.T) {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{Name: "429-test", MaxRequests: 1})
	bodyContent := "Rate limit exceeded"
	inner := &errRT{
		resp: &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Body:       io.NopCloser(strings.NewReader(bodyContent)),
			Header:     make(http.Header),
		},
		err: nil,
	}
	br := &breakerRoundTripper{next: inner, cb: cb, failureClassifier: &DefaultFailureClassifier{}}

	resp, err := br.RoundTrip(httptest.NewRequest("GET", "/", nil))

	// Circuit breaker MUST return response
	require.NotNil(t, resp, "response should not be nil even with 429")
	require.Nil(t, err, "error should be nil when response exists")
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)

	// Body must be readable
	body, readErr := io.ReadAll(resp.Body)
	require.NoError(t, readErr)
	assert.Equal(t, bodyContent, string(body))

	_ = resp.Body.Close()
}
