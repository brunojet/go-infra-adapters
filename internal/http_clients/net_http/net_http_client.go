package adapters

import (
	"context"
	"maps"
	"net/http"
	"net/url"
	"time"

	"github.com/brunojet/go-infra-adapters/debugassert"
)

type netHttpClient struct {
	client  *http.Client
	baseURL string
	headers http.Header
}

func NewNetHttpClient(opts ...HttpClientOption) (*netHttpClient, error) {
	cfg := newHttpClientConfig(opts...)

	client := &http.Client{
		Transport: cfg.roundTripper,
		Timeout:   time.Duration(cfg.responseTimeoutMs) * time.Millisecond,
	}

	adapter := &netHttpClient{
		client:  client,
		baseURL: cfg.baseURL,
		headers: cfg.headers,
	}

	return adapter, nil
}

// Do sends the provided request using the context and returns the response.
func (c *netHttpClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// build request (attach context, merge headers, resolve URL)
	r, err := c.buildRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// perform the request (with optional circuit breaker)
	return c.client.Do(r)
}

// mergeConfigHeaders merges the client's configured headers with the request's headers, giving precedence to the request's headers in case of conflicts. This ensures
// that client-level headers are included while allowing per-request overrides.
func (c *netHttpClient) mergeConfigHeaders(req *http.Request) {
	debugassert.Assert(req != nil, "http_client: request should not be nil")
	if c.headers == nil {
		return
	}
	merged := make(http.Header, len(req.Header)+len(c.headers))
	if req.Header != nil {
		maps.Copy(merged, req.Header)
	}
	maps.Copy(merged, c.headers)
	req.Header = merged
}

// buildRequest constructs the final http.Request by attaching the context, merging headers, and resolving the URL against the base URL if necessary. This centralizes request preparation logic and ensures consistent behavior across all requests made by the client.
func (c *netHttpClient) buildRequest(ctx context.Context, req *http.Request) (*http.Request, error) {
	debugassert.Assert(req != nil, "http_client: request should not be nil")
	r := req.WithContext(ctx)
	c.mergeConfigHeaders(r)
	if !r.URL.IsAbs() && c.baseURL != "" {
		if base, err := url.Parse(c.baseURL); err == nil {
			r.URL = base.ResolveReference(r.URL)
		}
	}
	return r, nil
}
