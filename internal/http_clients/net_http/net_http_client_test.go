package net_http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	middlewares "github.com/brunojet/go-infra-adapters/internal/middlewares/http_client"
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

func TestNewNetHttpClient_SettingsApplied(t *testing.T) {
	myRT := &dummyRT{}
	aitf, err := NewNetHttpClient(WithBaseURL("https://api.test"), WithHeader("X", "Y"), WithTimeout(10, 20), WithRoundTripper(myRT))
	require.NoError(t, err)
	a := aitf.(*netHttpClient)

	assert.Equal(t, "https://api.test", a.baseURL)
	assert.Equal(t, "Y", a.headers.Get("X"))
	assert.Equal(t, 20*time.Millisecond, a.client.Timeout)
	assert.Equal(t, myRT, a.client.Transport)
}

func TestMergeConfigHeaders_ClientOverridesRequest(t *testing.T) {
	a := &netHttpClient{headers: make(http.Header)}
	a.headers.Set("X", "client-val")

	req := httptest.NewRequest("GET", "http://example.local/test", nil)
	req.Header.Set("X", "req-val")
	req.Header.Set("Y", "req-only")

	a.mergeConfigHeaders(req)

	assert.Equal(t, "client-val", req.Header.Get("X"))
	assert.Equal(t, "req-only", req.Header.Get("Y"))
}

func TestBuildRequest_ResolvesRelativeURLAndMergesHeaders(t *testing.T) {
	a := &netHttpClient{headers: make(http.Header), baseURL: "https://api.host"}
	a.headers.Set("H", "client")

	req := httptest.NewRequest("GET", "/v1/items", nil)
	req.Header.Set("H", "request")

	r, err := a.buildRequest(context.Background(), req)
	require.NoError(t, err)

	assert.True(t, r.URL.IsAbs())
	assert.Equal(t, "https://api.host/v1/items", r.URL.String())
	// client header overrides request header per current implementation
	assert.Equal(t, "client", r.Header.Get("H"))
}

func TestNewNetHttpClient_NoRoundTripper_DoesNotPanic(t *testing.T) {
	aitf, err := NewNetHttpClient(WithRoundTripper(middlewares.NewBreakerMiddleware(nil)))
	require.NoError(t, err)
	a := aitf.(*netHttpClient)
	assert.NotNil(t, a.client)
}
