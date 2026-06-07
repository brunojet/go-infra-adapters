package net_http_server

import (
	"testing"
)

func TestHeaderControlMiddleware_SanitizeInboundHeaders_RemoveDenied(t *testing.T) {
	runServerHeaderTest(t, &serverHeaderTestCase{
		method: "GET",
		path:   "/",
		headers: map[string]string{
			"Authorization": "Bearer token123",
			"Cookie":        "session=abc",
			"Content-Type":  "application/json",
		},
		allowedHeaders: []string{"Content-Type"},
		expectedAbsent: map[string]bool{
			"Authorization": true,
			"Cookie":        true,
		},
		expectedPresent: map[string]bool{
			"Content-Type": true,
		},
	})
}

func TestHeaderControlMiddleware_SanitizeInboundHeaders_CaseInsensitive(t *testing.T) {
	runServerHeaderTest(t, &serverHeaderTestCase{
		method: "GET",
		path:   "/",
		headers: map[string]string{
			"AUTHORIZATION":   "Bearer token",
			"cookie":          "session=abc",
			"X-FORWARDED-FOR": "127.0.0.1",
		},
		allowedHeaders: []string{},
		expectedAbsent: map[string]bool{
			"Authorization":   true,
			"Cookie":          true,
			"X-Forwarded-For": true,
		},
	})
}

func TestHeaderControlMiddleware_SanitizeInboundHeaders_WithWhitelist(t *testing.T) {
	runServerHeaderTest(t, &serverHeaderTestCase{
		method: "GET",
		path:   "/",
		headers: map[string]string{
			"Content-Type": "application/json",
			"X-Request-ID": "req-123",
			"User-Agent":   "test-client",
			"Accept":       "application/json",
		},
		allowedHeaders: []string{"Content-Type", "X-Request-ID"},
		expectedPresent: map[string]bool{
			"Content-Type": true,
			"X-Request-Id": true,
		},
		expectedAbsent: map[string]bool{
			"User-Agent": true,
			"Accept":     true,
		},
	})
}

func TestHeaderControlMiddleware_RemoveHopByHopHeaders(t *testing.T) {
	runServerHeaderTest(t, &serverHeaderTestCase{
		method: "GET",
		path:   "/",
		headers: map[string]string{
			"Connection":        "keep-alive",
			"Keep-Alive":        "timeout=5",
			"Transfer-Encoding": "chunked",
			"Upgrade":           "websocket",
			"TE":                "trailers",
			"Content-Type":      "application/json",
		},
		allowedHeaders: []string{"Content-Type"},
		expectedAbsent: map[string]bool{
			"Connection":        true,
			"Keep-Alive":        true,
			"Transfer-Encoding": true,
			"Upgrade":           true,
			"TE":                true,
		},
		expectedPresent: map[string]bool{
			"Content-Type": true,
		},
	})
}

func TestHeaderControlMiddleware_RemoveEncodingHeaders(t *testing.T) {
	runServerHeaderTest(t, &serverHeaderTestCase{
		method: "GET",
		path:   "/",
		headers: map[string]string{
			"Accept-Encoding":  "gzip, deflate",
			"Content-Encoding": "gzip",
			"Content-Type":     "application/json",
		},
		allowedHeaders: []string{"Content-Type"},
		expectedAbsent: map[string]bool{
			"Accept-Encoding":  true,
			"Content-Encoding": true,
		},
		expectedPresent: map[string]bool{
			"Content-Type": true,
		},
	})
}

func TestHeaderControlMiddleware_RemoveContextSpecificHeaders(t *testing.T) {
	runServerHeaderTest(t, &serverHeaderTestCase{
		method: "GET",
		path:   "/",
		headers: map[string]string{
			"Host":              "client.example.com",
			"If-None-Match":     "abc123",
			"If-Modified-Since": "Wed, 21 Oct 2025 07:28:00 GMT",
			"Content-Type":      "application/json",
		},
		allowedHeaders: []string{"Content-Type"},
		expectedAbsent: map[string]bool{
			"Host":              true,
			"If-None-Match":     true,
			"If-Modified-Since": true,
		},
		expectedPresent: map[string]bool{
			"Content-Type": true,
		},
	})
}
