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
	br := &breakerRoundTripper{next: nil, cb: nil}
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
	br := &breakerRoundTripper{next: d, cb: nil}
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
	br := &breakerRoundTripper{next: d, cb: cb}
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
	br := &breakerRoundTripper{next: inner, cb: cb}
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
	br := &breakerRoundTripper{next: inner, cb: cb}
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
