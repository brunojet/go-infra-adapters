package middlewares

import (
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
	assert.True(t, d.called)
}

func TestBreakerRoundTripper_WithCB_ExecutesAndReturnsResponse(t *testing.T) {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{Name: "test", Timeout: time.Second, MaxRequests: 1})
	d := &dummyRT{}
	br := &breakerRoundTripper{next: d, cb: cb}
	resp, err := br.RoundTrip(httptest.NewRequest("GET", "/cb", nil))
	require.NoError(t, err)
	assert.NotNil(t, resp)
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
