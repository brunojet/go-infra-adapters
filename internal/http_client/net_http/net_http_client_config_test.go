package net_http

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/brunojet/go-infra-adapters/v4/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestHttpClientOptions_Validation(t *testing.T) {
	cfg := defaultHttpClientConfig()

	// WithBaseURL valid
	WithBaseURL("http://example.com")(&cfg)
	if cfg.baseURL != "http://example.com" {
		t.Fatalf("expected baseURL set")
	}
	// invalid base should panic
	require.Panics(t, func() { WithBaseURL("http://%zz")(&cfg) })

	// WithTimeout validations
	WithTimeout(100 * time.Millisecond)(&cfg)
	if cfg.timeout != 100*time.Millisecond {
		t.Fatalf("expected timeout set")
	}
	require.Panics(t, func() { WithTimeout(0)(&cfg) })

	// WithHeader validations
	WithHeader("k", "v")(&cfg)
	if cfg.headers.Get("k") != "v" {
		t.Fatalf("expected header set")
	}
	require.Panics(t, func() { WithHeader("", "v")(&cfg) })
	require.Panics(t, func() { WithHeader("k", "")(&cfg) })

	// WithRoundTripper nil
	rt := rtFunc(func(r *http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200}, nil })
	WithRoundTripper(rt)(&cfg)
	require.NotNil(t, cfg.roundTripper)
	require.Panics(t, func() { WithRoundTripper(nil)(&cfg) })
}

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNewHttpClientConfig_PanicOnInvalidOptions(t *testing.T) {
	require.Panics(t, func() { newHttpClientConfig(WithBaseURL("http://%zz")) })
	require.Panics(t, func() { newHttpClientConfig(WithTimeout(0)) })
	require.Panics(t, func() { newHttpClientConfig(WithHeader("", "v")) })
	require.Panics(t, func() { newHttpClientConfig(WithHeader("k", "")) })
	require.Panics(t, func() { newHttpClientConfig(WithRoundTripper(nil)) })
}

func TestNewHttpClientConfig_NilOptionSkipped(t *testing.T) {
	cfg := newHttpClientConfig(nil, WithBaseURL("http://example.com"), nil)
	if cfg.baseURL != "http://example.com" {
		t.Fatalf("expected baseURL set despite nil options, got %q", cfg.baseURL)
	}
}

func TestNewHttpClientConfig_ValidOptions(t *testing.T) {
	rt := rtFunc(func(r *http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Request: r}, nil })
	cfg := newHttpClientConfig(WithBaseURL("http://example.com"), WithTimeout(100*time.Millisecond), WithHeader("X-K", "v"), WithRoundTripper(rt))
	require.Equal(t, "http://example.com", cfg.baseURL)
	require.Equal(t, 100*time.Millisecond, cfg.timeout)
	require.Equal(t, "v", cfg.headers.Get("X-K"))
	require.NotNil(t, cfg.roundTripper)
	_, ok := cfg.roundTripper.(rtFunc)
	require.True(t, ok, "expected roundTripper to be of rtFunc type")
}

func TestWithMaxIdleConns_SetsTransportParam(t *testing.T) {
	cfg := defaultHttpClientConfig()
	transport, ok := cfg.roundTripper.(*http.Transport)
	require.True(t, ok, "default should be http.Transport")

	WithMaxIdleConns(50)(&cfg)
	require.Equal(t, 50, transport.MaxIdleConns)
	require.Panics(t, func() { WithMaxIdleConns(0)(&cfg) })
	require.Panics(t, func() { WithMaxIdleConns(-1)(&cfg) })
}

func TestWithMaxIdleConnsPerHost_SetsTransportParam(t *testing.T) {
	cfg := defaultHttpClientConfig()
	transport, ok := cfg.roundTripper.(*http.Transport)
	require.True(t, ok, "default should be http.Transport")

	WithMaxIdleConnsPerHost(10)(&cfg)
	require.Equal(t, 10, transport.MaxIdleConnsPerHost)
	require.Panics(t, func() { WithMaxIdleConnsPerHost(0)(&cfg) })
	require.Panics(t, func() { WithMaxIdleConnsPerHost(-1)(&cfg) })
}

func TestWithIdleConnTimeout_SetsTransportParam(t *testing.T) {
	cfg := defaultHttpClientConfig()
	transport, ok := cfg.roundTripper.(*http.Transport)
	require.True(t, ok, "default should be http.Transport")

	WithIdleConnTimeout(2 * time.Minute)(&cfg)
	require.Equal(t, 2*time.Minute, transport.IdleConnTimeout)
	require.Panics(t, func() { WithIdleConnTimeout(0)(&cfg) })
	require.Panics(t, func() { WithIdleConnTimeout(-1 * time.Millisecond)(&cfg) })
}

func TestWithDialContext_SetsDialContextOnTransport(t *testing.T) {
	cfg := defaultHttpClientConfig()
	transport, ok := cfg.roundTripper.(*http.Transport)
	require.True(t, ok, "default should be http.Transport")

	timeout := 5 * time.Second
	keepAlive := 30 * time.Second
	WithDialContext(timeout, keepAlive)(&cfg)
	require.NotNil(t, transport.DialContext)
	require.Panics(t, func() { WithDialContext(0, keepAlive)(&cfg) })
	require.Panics(t, func() { WithDialContext(timeout, 0)(&cfg) })
}

func TestWithLogger_SetsLogger(t *testing.T) {
	cfg := defaultHttpClientConfig()
	mockLogger := &mockLogger{}
	WithLogger(mockLogger)(&cfg)
	require.Equal(t, mockLogger, cfg.logger)
	require.Panics(t, func() { WithLogger(nil)(&cfg) })
}

func TestWithMiddleware_AppendsToMiddlewareChain(t *testing.T) {
	cfg := defaultHttpClientConfig()
	require.Equal(t, 0, len(cfg.customMiddlewares))

	mw1 := func(rt http.RoundTripper) http.RoundTripper { return rt }
	mw2 := func(rt http.RoundTripper) http.RoundTripper { return rt }
	WithMiddleware(mw1)(&cfg)
	require.Equal(t, 1, len(cfg.customMiddlewares))
	WithMiddleware(mw2)(&cfg)
	require.Equal(t, 2, len(cfg.customMiddlewares))
	require.Panics(t, func() { WithMiddleware(nil)(&cfg) })
}

func TestBuildFinalRoundTripper_AppliesMiddlewareInOrder(t *testing.T) {
	// Track which middlewares were applied and in what order
	var executionOrder []string

	mw1 := func(rt http.RoundTripper) http.RoundTripper {
		executionOrder = append(executionOrder, "mw1_wrap")
		return rtFunc(func(r *http.Request) (*http.Response, error) {
			executionOrder = append(executionOrder, "mw1_execute")
			return rt.RoundTrip(r)
		})
	}

	mw2 := func(rt http.RoundTripper) http.RoundTripper {
		executionOrder = append(executionOrder, "mw2_wrap")
		return rtFunc(func(r *http.Request) (*http.Response, error) {
			executionOrder = append(executionOrder, "mw2_execute")
			return rt.RoundTrip(r)
		})
	}

	cfg := newHttpClientConfig(
		WithMiddleware(mw1),
		WithMiddleware(mw2),
		WithRoundTripper(rtFunc(func(r *http.Request) (*http.Response, error) {
			executionOrder = append(executionOrder, "base_execute")
			return &http.Response{
				StatusCode: 200,
				Header:     make(http.Header),
				Body:       http.NoBody,
			}, nil
		})),
	)

	finalRT := cfg.buildFinalRoundTripper()

	// Verify middlewares were wrapped in order (mw1 first, mw2 second)
	require.Equal(t, []string{"mw1_wrap", "mw2_wrap"}, executionOrder)

	// Reset and execute the composed RoundTripper
	executionOrder = nil
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := finalRT.RoundTrip(req)
	require.NoError(t, err)
	if resp != nil {
		_ = resp.Body.Close()
	}

	// Verify execution order: mw2 executes first (outermost), then mw1, then base
	// This confirms middlewares wrap correctly
	require.Equal(t, []string{"mw2_execute", "mw1_execute", "base_execute"}, executionOrder)
}

// mockLogger is a test double for logger.Logger
type mockLogger struct{}

func (m *mockLogger) Debug(ctx context.Context, msg string, fields ...logger.Field)            {}
func (m *mockLogger) Info(ctx context.Context, msg string, fields ...logger.Field)             {}
func (m *mockLogger) Warn(ctx context.Context, msg string, fields ...logger.Field)             {}
func (m *mockLogger) Error(ctx context.Context, msg string, err error, fields ...logger.Field) {}
