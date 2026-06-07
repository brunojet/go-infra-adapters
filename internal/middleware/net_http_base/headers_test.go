package net_http_base

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeniedHeadersCommon_AllExpected(t *testing.T) {
	// Verify common denied headers contain expected entries
	expectedHeaders := []string{
		headerAuthorization,
		headerCookie,
		headerProxyAuthorization,
		headerConnection,
		headerKeepAlive,
		headerTE,
		headerTrailers,
		headerTransferEncoding,
		headerUpgrade,
		headerProxyConnection,
		headerXForwardedFor,
		headerXForwardedProto,
		headerXForwardedHost,
		headerXRealIP,
		headerContentEncoding,
		HeaderContentLength,
	}

	for _, header := range expectedHeaders {
		assert.True(t, DeniedHeadersCommon[header],
			"expected %s to be in DeniedHeadersCommon", header)
	}

	assert.Equal(t, 16, len(DeniedHeadersCommon),
		"DeniedHeadersCommon should have exactly 16 headers")
}

func TestDeniedHeadersRequestOnly_AllExpected(t *testing.T) {
	// Verify request-specific denied headers
	expectedHeaders := []string{
		headerHost,
		headerUserAgent,
		headerAcceptEncoding,
		headerIfNoneMatch,
		headerIfModifiedSince,
	}

	for _, header := range expectedHeaders {
		assert.True(t, DeniedHeadersRequestOnly[header],
			"expected %s to be in DeniedHeadersRequestOnly", header)
	}

	assert.Equal(t, 5, len(DeniedHeadersRequestOnly),
		"DeniedHeadersRequestOnly should have exactly 5 headers")
}

func TestDeniedHeadersResponseOnly_AllExpected(t *testing.T) {
	// Verify response-specific denied headers
	expectedHeaders := []string{
		headerSetCookie,
	}

	for _, header := range expectedHeaders {
		assert.True(t, DeniedHeadersResponseOnly[header],
			"expected %s to be in DeniedHeadersResponseOnly", header)
	}

	assert.Equal(t, 1, len(DeniedHeadersResponseOnly),
		"DeniedHeadersResponseOnly should have exactly 1 header (Set-Cookie)")
}

func TestDeniedHeadersNoOverlap_CommonVsRequestOnly(t *testing.T) {
	// Verify no overlap between common and request-only denied headers
	for header := range DeniedHeadersRequestOnly {
		assert.False(t, DeniedHeadersCommon[header],
			"header %s should not appear in both Common and RequestOnly", header)
	}
}

func TestDeniedHeadersNoOverlap_CommonVsResponseOnly(t *testing.T) {
	// Verify no overlap between common and response-only denied headers
	for header := range DeniedHeadersResponseOnly {
		assert.False(t, DeniedHeadersCommon[header],
			"header %s should not appear in both Common and ResponseOnly", header)
	}
}

func TestDeniedHeadersNoOverlap_RequestVsResponseOnly(t *testing.T) {
	// Verify no overlap between request-only and response-only denied headers
	for header := range DeniedHeadersRequestOnly {
		assert.False(t, DeniedHeadersResponseOnly[header],
			"header %s should not appear in both RequestOnly and ResponseOnly", header)
	}
}

func TestBuildDeniedHeaders_RequestDirection(t *testing.T) {
	// Build request denied headers (common + request-specific)
	result := BuildDeniedHeaders(DeniedHeadersRequestOnly)

	// Should contain all common headers
	for header := range DeniedHeadersCommon {
		assert.True(t, result[header],
			"request denied headers should contain common header %s", header)
	}

	// Should contain all request-specific headers
	for header := range DeniedHeadersRequestOnly {
		assert.True(t, result[header],
			"request denied headers should contain request-specific header %s", header)
	}

	// Should NOT contain response-only headers
	for header := range DeniedHeadersResponseOnly {
		assert.False(t, result[header],
			"request denied headers should not contain response-specific header %s", header)
	}

	expectedCount := len(DeniedHeadersCommon) + len(DeniedHeadersRequestOnly)
	assert.Equal(t, expectedCount, len(result),
		"request denied headers should have %d headers", expectedCount)
}

func TestBuildDeniedHeaders_ResponseDirection(t *testing.T) {
	// Build response denied headers (common + response-specific)
	result := BuildDeniedHeaders(DeniedHeadersResponseOnly)

	// Should contain all common headers
	for header := range DeniedHeadersCommon {
		assert.True(t, result[header],
			"response denied headers should contain common header %s", header)
	}

	// Should contain all response-specific headers
	for header := range DeniedHeadersResponseOnly {
		assert.True(t, result[header],
			"response denied headers should contain response-specific header %s", header)
	}

	// Should NOT contain request-only headers
	for header := range DeniedHeadersRequestOnly {
		assert.False(t, result[header],
			"response denied headers should not contain request-specific header %s", header)
	}

	expectedCount := len(DeniedHeadersCommon) + len(DeniedHeadersResponseOnly)
	assert.Equal(t, expectedCount, len(result),
		"response denied headers should have %d headers", expectedCount)
}

func TestBuildDeniedHeaders_EmptyDirectionSpecific(t *testing.T) {
	// Build denied headers with no direction-specific additions
	result := BuildDeniedHeaders(make(map[string]bool))

	// Should contain only common headers
	assert.Equal(t, len(DeniedHeadersCommon), len(result),
		"result should contain only common headers")

	for header := range DeniedHeadersCommon {
		assert.True(t, result[header],
			"result should contain common header %s", header)
	}
}

func TestBuildDeniedHeaders_IndependentFromInput(t *testing.T) {
	// Verify BuildDeniedHeaders creates independent copy (no mutation of original)
	input := map[string]bool{headerHost: true}
	result := BuildDeniedHeaders(input)

	// Modify result
	result[headerHost] = false
	result["Custom-Header"] = true

	// Original input should be unchanged
	assert.True(t, input[headerHost], "input should not be modified")
	assert.False(t, input["Custom-Header"], "input should not have new headers")
}

func TestBuildDeniedHeaders_TotalCount(t *testing.T) {
	// Verify total header count across all categories
	// Common: 16 (including Content-Encoding, Content-Length)
	// RequestOnly: 5
	// ResponseOnly: 1
	// Total for request: 16 + 5 = 21
	// Total for response: 16 + 1 = 17

	requestResult := BuildDeniedHeaders(DeniedHeadersRequestOnly)
	responseResult := BuildDeniedHeaders(DeniedHeadersResponseOnly)

	assert.Equal(t, 21, len(requestResult), "request denied headers should total 21")
	assert.Equal(t, 17, len(responseResult), "response denied headers should total 17")
}
